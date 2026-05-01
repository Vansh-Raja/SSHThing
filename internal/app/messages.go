package app

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Vansh-Raja/SSHThing/internal/config"
	"github.com/Vansh-Raja/SSHThing/internal/health"
	syncpkg "github.com/Vansh-Raja/SSHThing/internal/sync"
	"github.com/Vansh-Raja/SSHThing/internal/teams"
	"github.com/Vansh-Raja/SSHThing/internal/teamsclient"
	"github.com/Vansh-Raja/SSHThing/internal/update"
	tea "github.com/charmbracelet/bubbletea"
)

// ── Message types ─────────────────────────────────────────────────────

type sshFinishedMsg struct {
	err      error
	hostname string
	proto    string
	keyType  string
}

type mountFinishedMsg struct {
	action string // "mount" | "unmount"
	hostID int
	local  string
	err    error
	stderr string
}

type syncFinishedMsg struct {
	runID  int
	result *syncpkg.SyncResult
}

type syncAnimTickMsg struct {
	runID int
}

type updateCheckedMsg struct {
	runID  int
	result *update.CheckResult
	err    error
}

type updateAppliedMsg struct {
	runID          int
	result         *update.ApplyResult
	handoffStarted bool
	err            error
}

type updatePathFixedMsg struct {
	runID      int
	pathHealth update.PathHealth
	err        error
}

type quitFinishedMsg struct{}

type clearErrMsg struct {
	seq int
}

type tickMsg struct{}

type profileAuthPolledMsg struct {
	runID  int
	result teams.CliAuthPollResponse
	err    error
}

type healthRefreshStartedMsg struct {
	runID int
	total int
}

type hostHealthResultMsg struct {
	runID     int
	targetKey string
	hostID    int
	result    health.Result
}

type healthRefreshFinishedMsg struct {
	runID int
}

// ── Command constructors ──────────────────────────────────────────────

func tickCmd() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(time.Time) tea.Msg {
		return tickMsg{}
	})
}

func pollProfileAuthCmd(runID int, client *teamsclient.Client, sessionID, pollSecret string, interval time.Duration) tea.Cmd {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	return tea.Tick(interval, func(time.Time) tea.Msg {
		if client == nil {
			return profileAuthPolledMsg{runID: runID, err: fmt.Errorf("teams client is not configured")}
		}
		result, err := client.PollCLIAuth(context.Background(), sessionID, pollSecret)
		return profileAuthPolledMsg{runID: runID, result: result, err: err}
	})
}

func runSyncCmd(runID int, mgr *syncpkg.Manager) tea.Cmd {
	return func() tea.Msg {
		if mgr == nil {
			return syncFinishedMsg{runID: runID, result: &syncpkg.SyncResult{Success: false, Message: "sync manager is nil", Timestamp: time.Now()}}
		}
		return syncFinishedMsg{runID: runID, result: mgr.Sync()}
	}
}

func syncAnimTickCmd(runID int) tea.Cmd {
	return tea.Tick(120*time.Millisecond, func(time.Time) tea.Msg {
		return syncAnimTickMsg{runID: runID}
	})
}

func runUpdateCheckCmd(runID int, currentVersion string, cfg config.Config) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		result, err := update.Check(ctx, currentVersion, &cfg)
		if err != nil {
			return updateCheckedMsg{runID: runID, err: err}
		}
		return updateCheckedMsg{runID: runID, result: &result}
	}
}

func runUpdateApplyCmd(runID int, check update.CheckResult) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		exe, err := os.Executable()
		if err != nil {
			return updateAppliedMsg{runID: runID, err: err}
		}
		result, err := update.Apply(ctx, check, exe)
		if err != nil {
			return updateAppliedMsg{runID: runID, err: err}
		}
		handoffStarted := false
		if result.Handoff != nil {
			if err := update.LaunchHandoff(result.Handoff); err != nil {
				return updateAppliedMsg{runID: runID, err: err}
			}
			handoffStarted = true
		}
		return updateAppliedMsg{runID: runID, result: &result, handoffStarted: handoffStarted}
	}
}

func runUpdatePathFixCmd(runID int, desiredExe string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		ph, err := update.FixPathConflicts(ctx, desiredExe)
		if err != nil {
			return updatePathFixedMsg{runID: runID, err: err}
		}
		return updatePathFixedMsg{runID: runID, pathHealth: ph}
	}
}
