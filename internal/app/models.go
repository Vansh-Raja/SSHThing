package app

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

// ViewMode represents the current view state
type ViewMode int

const (
	ViewModeList ViewMode = iota
	ViewModeAddHost
	ViewModeEditHost
	ViewModeDeleteHost
	ViewModeHelp
	ViewModeSpotlight
	ViewModeSettings
	ViewModeLogin
	ViewModeSetup // First-run password setup
	ViewModeQuitConfirm
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
	case ViewModeSettings:
		return "settings"
	case ViewModeLogin:
		return "login"
	case ViewModeSetup:
		return "setup"
	case ViewModeQuitConfirm:
		return "quit_confirm"
	default:
		return "unknown"
	}
}
