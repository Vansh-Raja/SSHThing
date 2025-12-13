package main

import "time"

// Host represents an SSH host configuration
type Host struct {
	ID            int       `json:"id"`
	Label         string    `json:"label,omitempty"`
	Hostname      string    `json:"hostname"`
	Username      string    `json:"username"`
	Port          int       `json:"port"`
	HasKey        bool      `json:"has_key"`
	KeyType       string    `json:"key_type"` // "ed25519", "rsa", "ecdsa", or "pasted"
	CreatedAt     time.Time `json:"created_at"`
	LastConnected *time.Time `json:"last_connected,omitempty"`
}

// GetHardcodedHosts returns sample data for the MVP
func GetHardcodedHosts() []Host {
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)

	return []Host{
		{
			ID:            1,
			Label:         "web-prod-1",
			Hostname:      "web-prod-1.example.com",
			Username:      "ec2-user",
			Port:          22,
			HasKey:        true,
			KeyType:       "ed25519",
			CreatedAt:     now.Add(-30 * 24 * time.Hour),
			LastConnected: &yesterday,
		},
		{
			ID:            2,
			Label:         "db-server",
			Hostname:      "db-server.internal",
			Username:      "ubuntu",
			Port:          22,
			HasKey:        true,
			KeyType:       "rsa",
			CreatedAt:     now.Add(-15 * 24 * time.Hour),
			LastConnected: nil,
		},
		{
			ID:            3,
			Label:         "staging",
			Hostname:      "staging.dev.local",
			Username:      "deploy",
			Port:          2222,
			HasKey:        true,
			KeyType:       "ed25519",
			CreatedAt:     now.Add(-7 * 24 * time.Hour),
			LastConnected: &now,
		},
		{
			ID:            4,
			Label:         "backup-nas",
			Hostname:      "backup-nas.home",
			Username:      "admin",
			Port:          22,
			HasKey:        false,
			KeyType:       "",
			CreatedAt:     now.Add(-60 * 24 * time.Hour),
			LastConnected: nil,
		},
	}
}

// ViewMode represents the current view state
type ViewMode int

const (
	ViewModeList ViewMode = iota
	ViewModeAddHost
	ViewModeEditHost
	ViewModeDeleteHost
	ViewModeHelp
	ViewModeSpotlight
	ViewModeLogin
	ViewModeSetup // First-run password setup
)

// String returns the string representation of ViewMode
func (v ViewMode) String() string {
	switch v {
	case ViewModeList:
		return "list"
	case ViewModeAddHost:
		return "add"
	case ViewModeEditHost:
		return "edit"
	case ViewModeDeleteHost:
		return "delete"
	case ViewModeHelp:
		return "help"
	case ViewModeSpotlight:
		return "spotlight"
	case ViewModeLogin:
		return "login"
	case ViewModeSetup:
		return "setup"
	default:
		return "unknown"
	}
}
