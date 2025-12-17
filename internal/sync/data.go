package sync

import (
	"time"
)

// SyncData represents the portable format for syncing hosts across devices.
// The KeyData field in each host remains encrypted - we never export decrypted keys.
// The Salt field is required to re-encrypt keys when importing to a different database.
type SyncData struct {
	Version   int        `json:"version"`
	Salt      string     `json:"salt"` // Hex-encoded encryption salt from source database
	UpdatedAt time.Time  `json:"updated_at"`
	Hosts     []SyncHost `json:"hosts"`
}

// SyncHost represents a host entry in the sync file.
// This mirrors db.HostModel but is designed for JSON serialization.
type SyncHost struct {
	ID            int        `json:"id"`
	Label         string     `json:"label"`
	Hostname      string     `json:"hostname"`
	Username      string     `json:"username"`
	Port          int        `json:"port"`
	KeyData       string     `json:"key_data"` // Encrypted blob (stays encrypted)
	KeyType       string     `json:"key_type"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	LastConnected *time.Time `json:"last_connected,omitempty"`
}

// SyncStatus represents the current state of sync operations
type SyncStatus int

const (
	SyncStatusDisabled SyncStatus = iota
	SyncStatusIdle
	SyncStatusSyncing
	SyncStatusError
	SyncStatusSuccess
)

// SyncResult represents the outcome of a sync operation
type SyncResult struct {
	Success      bool
	Message      string
	HostsAdded   int
	HostsUpdated int
	HostsRemoved int
	Conflicts    []SyncConflict
	Error        error
	Timestamp    time.Time
}

// SyncConflict represents a conflict between local and remote host data
type SyncConflict struct {
	HostID     int
	Hostname   string
	LocalTime  time.Time
	RemoteTime time.Time
	Resolution string // "local", "remote", or "skipped"
}

// CurrentSyncVersion is the version of the sync data format
const CurrentSyncVersion = 1

// SyncFileName is the name of the sync data file in the repository
const SyncFileName = "sshthing-hosts.json"
