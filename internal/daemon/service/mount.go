package service

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"strconv"
	"strings"

	"github.com/Vansh-Raja/SSHThing/internal/config"
	mountpkg "github.com/Vansh-Raja/SSHThing/internal/mount"
	"github.com/Vansh-Raja/SSHThing/internal/ssh"
)

// MountService provides SSHFS mount management via the mount.Manager.
type MountService struct {
	Vault    *Vault
	CfgStore *CfgStore
	Manager  *mountpkg.Manager
}

// MountSummary is the wire-safe representation of an active mount.
type MountSummary struct {
	HostID     string `json:"hostId"`
	Hostname   string `json:"hostname"`
	LocalPath  string `json:"localPath"`
	RemotePath string `json:"remotePath"`
}

// MountStart mounts a host filesystem. It mirrors the TUI's handleMountEnter flow.
// The mount runs synchronously (PrepareMount → Start → Finalize).
func (ms *MountService) Start(ctx context.Context, hostID, remotePath string) (*MountSummary, error) {
	store := ms.Vault.Store()
	if store == nil {
		return nil, ErrVaultLocked
	}
	intID, err := strconv.Atoi(hostID)
	if err != nil {
		return nil, fmt.Errorf("invalid host id %q: %w", hostID, err)
	}
	model, err := store.GetHostByID(intID)
	if err != nil {
		return nil, fmt.Errorf("get host: %w", err)
	}
	secret, err := store.GetHostSecret(intID)
	if err != nil {
		return nil, fmt.Errorf("get secret: %w", err)
	}

	hostKeyPolicy := string(config.HostKeyAcceptNew)
	keepAlive := 60
	localMountBase := ""
	if ms.CfgStore != nil {
		cfg := ms.CfgStore.Get()
		hostKeyPolicy = string(cfg.SSH.HostKeyPolicy)
		keepAlive = cfg.SSH.KeepAliveSeconds
		localMountBase = cfg.Mount.LocalMountPath
	}
	if remotePath == "" && ms.CfgStore != nil {
		remotePath = ms.CfgStore.Get().Mount.DefaultRemotePath
	}

	conn := ssh.Connection{
		Hostname:         model.Hostname,
		Username:         model.Username,
		Port:             model.Port,
		HostKeyPolicy:    hostKeyPolicy,
		KeepAliveSeconds: keepAlive,
	}
	if model.KeyType != "password" && secret != "" {
		conn.PrivateKey = secret
	}

	displayName := model.Label
	if displayName == "" {
		displayName = model.Hostname
	}

	prep, err := ms.Manager.PrepareMount(intID, conn, remotePath, displayName, localMountBase)
	if err != nil {
		return nil, err
	}

	if err := prep.Cmd().Start(); err != nil {
		ms.Manager.AbortMount(prep)
		return nil, fmt.Errorf("start sshfs: %w", err)
	}

	if err := ms.Manager.FinalizeMount(prep); err != nil {
		return nil, err
	}

	// Persist mount state — best effort.
	_ = store.UpsertMountState(intID, prep.LocalPath, prep.RemotePath())

	return &MountSummary{
		HostID:     hostID,
		Hostname:   model.Hostname,
		LocalPath:  prep.LocalPath,
		RemotePath: prep.RemotePath(),
	}, nil
}

// Stop unmounts a host's filesystem.
func (ms *MountService) Stop(ctx context.Context, hostID string) error {
	store := ms.Vault.Store()
	if store == nil {
		return ErrVaultLocked
	}
	intID, err := strconv.Atoi(hostID)
	if err != nil {
		return fmt.Errorf("invalid host id %q: %w", hostID, err)
	}

	cmd, localPath, err := ms.Manager.PrepareUnmount(intID)
	if err != nil {
		return err
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("unmount: %w", err)
	}
	if err := ms.Manager.FinalizeUnmount(intID, nil); err != nil {
		return err
	}

	// Remove persisted mount state — best effort.
	_ = store.DeleteMountState(intID)
	_ = localPath // used for logging if needed
	return nil
}

// PrereqsResult is the structured result of CheckPrereqs.
type PrereqsResult struct {
	OK       bool     `json:"ok"`
	Platform string   `json:"platform"`
	Missing  []string `json:"missing"`
	Hints    []string `json:"hints"`
}

// CheckPrereqs verifies that SSHFS prerequisites are installed and returns
// a structured result suitable for display in the renderer's prereqs modal.
func (ms *MountService) CheckPrereqs() PrereqsResult {
	platform := runtime.GOOS
	err := ms.Manager.CheckPrereqs()
	if err == nil {
		return PrereqsResult{OK: true, Platform: platform, Missing: []string{}, Hints: []string{}}
	}

	// Parse the error string into missing tools and human-readable install hints.
	// Manager.CheckPrereqs returns descriptive strings; we split on newlines.
	errStr := err.Error()
	lines := strings.Split(errStr, "\n")
	var missing []string
	var hints []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Lines starting with "⚠" are high-level missing-tool descriptions.
		// Lines starting with spaces or "brew"/"apt"/"dnf"/"pacman" are hints.
		if strings.HasPrefix(line, "⚠") || strings.HasPrefix(line, "Mount feature") || strings.HasPrefix(line, "Mount requires") {
			missing = append(missing, strings.TrimPrefix(strings.TrimPrefix(line, "⚠ "), "⚠"))
		} else {
			hints = append(hints, line)
		}
	}
	if len(missing) == 0 {
		missing = []string{errStr}
	}
	return PrereqsResult{OK: false, Platform: platform, Missing: missing, Hints: hints}
}

// RestoreFromDB reads persisted mount states from the DB and re-registers any
// that are still actively mounted. Stale states (where the mount point no
// longer exists or the process is gone) are dropped from the DB.
// This mirrors the TUI's restoreMountsFromDB helper.
func (ms *MountService) RestoreFromDB(ctx context.Context) {
	store := ms.Vault.Store()
	if store == nil {
		log.Printf("mount: RestoreFromDB: vault locked, skipping restore")
		return
	}

	states, err := store.GetMountStates()
	if err != nil {
		log.Printf("mount: RestoreFromDB: get states: %v", err)
		return
	}
	if len(states) == 0 {
		return
	}

	log.Printf("mount: RestoreFromDB: checking %d persisted mount(s)", len(states))

	// Build Mount records for RestoreMounted; it will check liveness via isMounted.
	records := make([]mountpkg.Mount, 0, len(states))
	for _, state := range states {
		hostname := ""
		if model, err := store.GetHostByID(state.HostID); err == nil && model != nil {
			hostname = model.Hostname
		}
		records = append(records, mountpkg.Mount{
			HostID:     state.HostID,
			Hostname:   hostname,
			LocalPath:  state.LocalPath,
			RemotePath: state.RemotePath,
		})
	}

	// RestoreMounted checks liveness and registers live mounts in-memory,
	// discarding stale ones. We then reconcile the DB: drop states that
	// were not actually alive.
	ms.Manager.RestoreMounted(records)

	active := ms.Manager.ListActive()
	activeSet := make(map[int]bool, len(active))
	for _, m := range active {
		activeSet[m.HostID] = true
	}

	restored := 0
	for _, state := range states {
		if activeSet[state.HostID] {
			restored++
			log.Printf("mount: RestoreFromDB: restored hostID=%d at %s", state.HostID, state.LocalPath)
		} else {
			_ = store.DeleteMountState(state.HostID)
			log.Printf("mount: RestoreFromDB: dropped stale hostID=%d at %s", state.HostID, state.LocalPath)
		}
	}
	log.Printf("mount: RestoreFromDB: %d restored, %d dropped", restored, len(states)-restored)
}

// List returns summaries of all currently active mounts.
func (ms *MountService) List() []MountSummary {
	mounts := ms.Manager.ListActive()
	out := make([]MountSummary, 0, len(mounts))
	for _, m := range mounts {
		out = append(out, MountSummary{
			HostID:     strconv.Itoa(m.HostID),
			Hostname:   m.Hostname,
			LocalPath:  m.LocalPath,
			RemotePath: m.RemotePath,
		})
	}
	return out
}
