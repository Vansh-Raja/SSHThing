package sync

import (
	"fmt"
	"sync"
	"time"

	"github.com/Vansh-Raja/SSHThing/internal/config"
	"github.com/Vansh-Raja/SSHThing/internal/db"
)

// Manager orchestrates sync operations
type Manager struct {
	mu sync.RWMutex

	cfg        *config.Config
	store      *db.Store
	git        *GitManager
	password   string // Master password for re-encrypting keys during import
	lastSync   time.Time
	lastResult *SyncResult
	status     SyncStatus
	stage      string
}

// NewManager creates a new sync manager
func NewManager(cfg *config.Config, store *db.Store, password string) (*Manager, error) {
	if !cfg.Sync.Enabled {
		return &Manager{
			cfg:      cfg,
			store:    store,
			password: password,
			status:   SyncStatusDisabled,
			stage:    "",
		}, nil
	}

	syncPath, err := cfg.SyncPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get sync path: %w", err)
	}

	git := NewGitManager(
		syncPath,
		cfg.Sync.RepoURL,
		cfg.Sync.Branch,
		cfg.Sync.SSHKeyPath,
	)

	return &Manager{
		cfg:      cfg,
		store:    store,
		git:      git,
		password: password,
		status:   SyncStatusIdle,
		stage:    "",
	}, nil
}

// Init initializes the sync manager and Git repository
func (m *Manager) Init() error {
	if m.GetStatus() == SyncStatusDisabled {
		return nil
	}

	if m.git == nil {
		return fmt.Errorf("git manager not initialized")
	}

	return m.git.Init()
}

// Sync performs a full sync operation: pull -> import -> export -> commit -> push
func (m *Manager) Sync() *SyncResult {
	if m.GetStatus() == SyncStatusDisabled {
		return &SyncResult{
			Success:   false,
			Message:   "Sync is disabled",
			Timestamp: time.Now(),
		}
	}

	m.setSyncState(SyncStatusSyncing, "initializing", nil, false)
	result := &SyncResult{Timestamp: time.Now()}

	// Step 1: Initialize repository if needed
	m.setStage("initializing")
	if err := m.Init(); err != nil {
		result.Error = err
		result.Message = fmt.Sprintf("Init failed: %v", err)
		m.setSyncState(SyncStatusError, "", result, false)
		return result
	}

	// Step 2: Pull remote changes
	if m.git.HasRemote() {
		m.setStage("pulling")
		if err := m.git.Pull(); err != nil {
			result.Error = err
			result.Message = fmt.Sprintf("Pull failed: %v", err)
			m.setSyncState(SyncStatusError, "", result, false)
			return result
		}
	}

	// Step 3: Load remote data and import
	m.setStage("loading")
	remoteData, err := LoadFromFile(m.git.GetSyncFilePath(), m.password)
	if err != nil {
		result.Error = err
		result.Message = fmt.Sprintf("Load failed: %v", err)
		m.setSyncState(SyncStatusError, "", result, false)
		return result
	}

	if remoteData != nil {
		m.setStage("importing")
		importResult, err := Import(m.store, remoteData, m.password)
		if err != nil {
			result.Error = err
			result.Message = fmt.Sprintf("Import failed: %v", err)
			m.setSyncState(SyncStatusError, "", result, false)
			return result
		}
		result.HostsAdded = importResult.Added
		result.HostsUpdated = importResult.Updated
		result.HostsPulled = importResult.Added + importResult.Updated
		result.Conflicts = importResult.Conflicts
	}

	// Step 4: Export local data
	m.setStage("exporting")
	// Best-effort: garbage collect old group tombstones before exporting.
	if m.store != nil {
		_ = m.store.PurgeDeletedGroups(GroupTombstoneRetention)
	}
	localData, err := Export(m.store)
	if err != nil {
		result.Error = err
		result.Message = fmt.Sprintf("Export failed: %v", err)
		m.setSyncState(SyncStatusError, "", result, false)
		return result
	}
	result.HostsPushed = computeHostsPushed(localData, remoteData)
	if err := ExportDataToFile(localData, m.git.GetSyncFilePath(), m.password); err != nil {
		result.Error = err
		result.Message = fmt.Sprintf("Export failed: %v", err)
		m.setSyncState(SyncStatusError, "", result, false)
		return result
	}

	// Step 5: Commit changes
	m.setStage("committing")
	commitMsg := fmt.Sprintf("Sync: %s", time.Now().Format(time.RFC3339))
	if err := m.git.CommitChanges(commitMsg); err != nil {
		result.Error = err
		result.Message = fmt.Sprintf("Commit failed: %v", err)
		m.setSyncState(SyncStatusError, "", result, false)
		return result
	}

	// Step 6: Push to remote
	if m.git.HasRemote() {
		m.setStage("pushing")
		if err := m.git.Push(); err != nil {
			result.Error = err
			result.Message = fmt.Sprintf("Push failed: %v", err)
			m.setSyncState(SyncStatusError, "", result, false)
			return result
		}
	}

	// Success
	m.setStage("finishing")
	now := time.Now()
	result.Success = true
	result.Message = "Sync completed successfully"
	result.Timestamp = now
	m.mu.Lock()
	m.status = SyncStatusSuccess
	m.stage = ""
	m.lastSync = now
	m.lastResult = result
	m.mu.Unlock()

	return result
}

// GetStatus returns the current sync status
func (m *Manager) GetStatus() SyncStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status
}

// GetLastSync returns the time of the last successful sync
func (m *Manager) GetLastSync() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastSync
}

// GetLastResult returns the result of the last sync operation
func (m *Manager) GetLastResult() *SyncResult {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastResult
}

// IsEnabled returns true if sync is enabled
func (m *Manager) IsEnabled() bool {
	return m.GetStatus() != SyncStatusDisabled
}

// StageString returns the current sync stage during an in-flight sync.
func (m *Manager) StageString() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.stage
}

// StatusString returns a human-readable status string
func (m *Manager) StatusString() string {
	m.mu.RLock()
	status := m.status
	lastSync := m.lastSync
	lastResult := m.lastResult
	stage := m.stage
	m.mu.RUnlock()

	switch status {
	case SyncStatusDisabled:
		return "Disabled"
	case SyncStatusIdle:
		if lastSync.IsZero() {
			return "Not synced"
		}
		return fmt.Sprintf("Last: %s", m.timeSince(lastSync))
	case SyncStatusSyncing:
		if stage != "" {
			return "Syncing: " + stage
		}
		return "Syncing"
	case SyncStatusError:
		if lastResult != nil && lastResult.Error != nil {
			return fmt.Sprintf("Error: %v", lastResult.Error)
		}
		return "Error"
	case SyncStatusSuccess:
		return fmt.Sprintf("Synced %s", m.timeSince(lastSync))
	default:
		return "Unknown"
	}
}

func (m *Manager) setStage(stage string) {
	m.mu.Lock()
	m.stage = stage
	m.mu.Unlock()
}

func (m *Manager) setSyncState(status SyncStatus, stage string, result *SyncResult, updateLastSync bool) {
	m.mu.Lock()
	m.status = status
	m.stage = stage
	if result != nil {
		m.lastResult = result
	}
	if updateLastSync {
		m.lastSync = time.Now()
	}
	m.mu.Unlock()
}

// timeSince returns a human-readable time since string
func (m *Manager) timeSince(t time.Time) string {
	d := time.Since(t)
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
	return fmt.Sprintf("%dd ago", int(d.Hours()/24))
}

func computeHostsPushed(local, remote *SyncData) int {
	if local == nil {
		return 0
	}
	if remote == nil {
		return len(local.Hosts)
	}

	remoteByID := make(map[int]SyncHost, len(remote.Hosts))
	for _, h := range remote.Hosts {
		remoteByID[h.ID] = h
	}

	pushed := 0
	for _, h := range local.Hosts {
		rh, ok := remoteByID[h.ID]
		if !ok || h.UpdatedAt.After(rh.UpdatedAt) {
			pushed++
		}
	}
	return pushed
}
