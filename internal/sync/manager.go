package sync

import (
	"fmt"
	"time"

	"github.com/Vansh-Raja/SSHThing/internal/config"
	"github.com/Vansh-Raja/SSHThing/internal/db"
)

// Manager orchestrates sync operations
type Manager struct {
	cfg        *config.Config
	store      *db.Store
	git        *GitManager
	password   string // Master password for re-encrypting keys during import
	lastSync   time.Time
	lastResult *SyncResult
	status     SyncStatus
}

// NewManager creates a new sync manager
func NewManager(cfg *config.Config, store *db.Store, password string) (*Manager, error) {
	if !cfg.Sync.Enabled {
		return &Manager{
			cfg:      cfg,
			store:    store,
			password: password,
			status:   SyncStatusDisabled,
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
	}, nil
}

// Init initializes the sync manager and Git repository
func (m *Manager) Init() error {
	if m.status == SyncStatusDisabled {
		return nil
	}

	if m.git == nil {
		return fmt.Errorf("git manager not initialized")
	}

	return m.git.Init()
}

// Sync performs a full sync operation: pull -> import -> export -> commit -> push
func (m *Manager) Sync() *SyncResult {
	if m.status == SyncStatusDisabled {
		return &SyncResult{
			Success:   false,
			Message:   "Sync is disabled",
			Timestamp: time.Now(),
		}
	}

	m.status = SyncStatusSyncing
	result := &SyncResult{Timestamp: time.Now()}

	// Step 1: Initialize repository if needed
	if err := m.Init(); err != nil {
		m.status = SyncStatusError
		result.Error = err
		result.Message = fmt.Sprintf("Init failed: %v", err)
		m.lastResult = result
		return result
	}

	// Step 2: Pull remote changes
	if m.git.HasRemote() {
		if err := m.git.Pull(); err != nil {
			m.status = SyncStatusError
			result.Error = err
			result.Message = fmt.Sprintf("Pull failed: %v", err)
			m.lastResult = result
			return result
		}
	}

	// Step 3: Load remote data and import
	remoteData, err := LoadFromFile(m.git.GetSyncFilePath())
	if err != nil {
		m.status = SyncStatusError
		result.Error = err
		result.Message = fmt.Sprintf("Load failed: %v", err)
		m.lastResult = result
		return result
	}

	if remoteData != nil {
		importResult, err := Import(m.store, remoteData, m.password)
		if err != nil {
			m.status = SyncStatusError
			result.Error = err
			result.Message = fmt.Sprintf("Import failed: %v", err)
			m.lastResult = result
			return result
		}
		result.HostsAdded = importResult.Added
		result.HostsUpdated = importResult.Updated
		result.Conflicts = importResult.Conflicts
	}

	// Step 4: Export local data
	if err := ExportToFile(m.store, m.git.GetSyncFilePath()); err != nil {
		m.status = SyncStatusError
		result.Error = err
		result.Message = fmt.Sprintf("Export failed: %v", err)
		m.lastResult = result
		return result
	}

	// Step 5: Commit changes
	commitMsg := fmt.Sprintf("Sync: %s", time.Now().Format(time.RFC3339))
	if err := m.git.CommitChanges(commitMsg); err != nil {
		m.status = SyncStatusError
		result.Error = err
		result.Message = fmt.Sprintf("Commit failed: %v", err)
		m.lastResult = result
		return result
	}

	// Step 6: Push to remote
	if m.git.HasRemote() {
		if err := m.git.Push(); err != nil {
			m.status = SyncStatusError
			result.Error = err
			result.Message = fmt.Sprintf("Push failed: %v", err)
			m.lastResult = result
			return result
		}
	}

	// Success
	m.status = SyncStatusSuccess
	m.lastSync = time.Now()
	result.Success = true
	result.Message = "Sync completed successfully"
	m.lastResult = result

	return result
}

// GetStatus returns the current sync status
func (m *Manager) GetStatus() SyncStatus {
	return m.status
}

// GetLastSync returns the time of the last successful sync
func (m *Manager) GetLastSync() time.Time {
	return m.lastSync
}

// GetLastResult returns the result of the last sync operation
func (m *Manager) GetLastResult() *SyncResult {
	return m.lastResult
}

// IsEnabled returns true if sync is enabled
func (m *Manager) IsEnabled() bool {
	return m.status != SyncStatusDisabled
}

// StatusString returns a human-readable status string
func (m *Manager) StatusString() string {
	switch m.status {
	case SyncStatusDisabled:
		return "Disabled"
	case SyncStatusIdle:
		if m.lastSync.IsZero() {
			return "Not synced"
		}
		return fmt.Sprintf("Last: %s", m.timeSince(m.lastSync))
	case SyncStatusSyncing:
		return "Syncing..."
	case SyncStatusError:
		if m.lastResult != nil && m.lastResult.Error != nil {
			return fmt.Sprintf("Error: %v", m.lastResult.Error)
		}
		return "Error"
	case SyncStatusSuccess:
		return fmt.Sprintf("Synced %s", m.timeSince(m.lastSync))
	default:
		return "Unknown"
	}
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
