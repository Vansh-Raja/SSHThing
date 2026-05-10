package service

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/Vansh-Raja/SSHThing/internal/config"
	"github.com/Vansh-Raja/SSHThing/internal/update"
)

// nudgeInterval is how long we wait between two release-feed polls. Set
// to a week so we never beat up the GitHub API and never spam the user.
// The first poll on daemon start is also throttled by this value via the
// LastCheckedAt cfg field — if the user just dismissed a banner, restarting
// the daemon doesn't re-fetch.
const nudgeInterval = 7 * 24 * time.Hour

// nudgeTickEvery is the loop cadence — every 6h we *consider* polling.
// The actual poll only fires when the throttle says so. 6h means a daemon
// that runs continuously will catch a new release within ~6h of when its
// LastCheckedAt expires, no matter what time of day the user reboots.
const nudgeTickEvery = 6 * time.Hour

// UpdateNudge polls GitHub for the latest release once per `nudgeInterval`,
// emits an `update.available` notification when a newer release is out
// (and not already dismissed by the user), and persists the observation
// state in CfgStore so reboots don't lose throttle position.
//
// The actual update apply path lives in `sshthing update` (CLI) — this
// service only surfaces awareness; nothing here ever modifies the
// installed binary or .app.
type UpdateNudge struct {
	CfgStore       *CfgStore
	CurrentVersion string                          // version of the running daemon binary
	Notify         func(method string, params any) // forwarded as a daemon notification

	mu      sync.Mutex
	stopped bool
	cancel  context.CancelFunc
}

// Start kicks off the background loop. Idempotent — calling Start twice
// without an intervening Stop is a no-op on the second call.
func (n *UpdateNudge) Start() {
	if n == nil || n.CfgStore == nil {
		return
	}
	n.mu.Lock()
	if n.cancel != nil {
		n.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	n.cancel = cancel
	n.mu.Unlock()

	go n.loop(ctx)
}

// Stop terminates the background loop. Safe to call multiple times.
func (n *UpdateNudge) Stop() {
	if n == nil {
		return
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	n.stopped = true
	if n.cancel != nil {
		n.cancel()
		n.cancel = nil
	}
}

func (n *UpdateNudge) loop(ctx context.Context) {
	// First poll: a small delay so we don't fight the daemon's own
	// startup IO. Then run on a steady tick.
	first := time.NewTimer(30 * time.Second)
	defer first.Stop()

	select {
	case <-ctx.Done():
		return
	case <-first.C:
	}

	n.maybePoll(ctx)

	tick := time.NewTicker(nudgeTickEvery)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			n.maybePoll(ctx)
		}
	}
}

// maybePoll runs a poll only when LastCheckedAt is older than nudgeInterval.
// Persists the new timestamp + observed version into CfgStore on success;
// silently logs and bails on transient network failures so we try again
// on the next tick.
func (n *UpdateNudge) maybePoll(ctx context.Context) {
	cfg := n.CfgStore.Get()
	if !shouldPoll(cfg.Updates.LastCheckedAt, time.Now()) {
		return
	}

	pollCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	result, err := update.Check(pollCtx, n.CurrentVersion, update.ReleaseChannelStable)
	if err != nil {
		log.Printf("update-nudge: check failed: %v", err)
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	_, _ = n.CfgStore.Mutate(func(c *config.Config) error {
		c.Updates.LastCheckedAt = now
		c.Updates.LastSeenVersion = result.LatestVersion
		c.Updates.LastSeenTag = result.LatestTag
		return nil
	})

	if !result.UpdateAvailable {
		return
	}
	// Don't re-banner a version the user has already dismissed. Empty
	// dismissed-version means "never dismissed" — banner shows.
	dismissed := strings.TrimSpace(cfg.Updates.DismissedVersion)
	if dismissed != "" && dismissed == result.LatestVersion {
		return
	}

	if n.Notify != nil {
		n.Notify("update.available", map[string]any{
			"currentVersion": result.CurrentVersion,
			"latestVersion":  result.LatestVersion,
			"latestTag":      result.LatestTag,
			"releaseUrl":     result.ReleaseURL,
		})
	}
}

func shouldPoll(lastCheckedRFC3339 string, now time.Time) bool {
	last := strings.TrimSpace(lastCheckedRFC3339)
	if last == "" {
		return true
	}
	t, err := time.Parse(time.RFC3339, last)
	if err != nil {
		// Corrupt timestamp → treat as never-checked rather than blocking
		// forever.
		return true
	}
	return now.Sub(t) >= nudgeInterval
}
