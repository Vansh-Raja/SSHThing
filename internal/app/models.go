package app

import "time"

// Host represents an SSH host configuration
type Host struct {
	ID            int        `json:"id"`
	Label         string     `json:"label,omitempty"`
	GroupName     string     `json:"group_name,omitempty"`
	Tags          []string   `json:"tags,omitempty"`
	Hostname      string     `json:"hostname"`
	Username      string     `json:"username"`
	Port          int        `json:"port"`
	HasKey        bool       `json:"has_key"`
	KeyType       string     `json:"key_type"` // "ed25519", "rsa", "ecdsa", or "pasted"
	CreatedAt     time.Time  `json:"created_at"`
	LastConnected *time.Time `json:"last_connected,omitempty"`
}

// ViewMode represents the current view state
type ViewMode int

const (
	ViewModeList ViewMode = iota
	ViewModeAddHost
	ViewModeEditHost
	ViewModeDeleteHost
	ViewModeCreateGroup
	ViewModeRenameGroup
	ViewModeDeleteGroup
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
	case ViewModeCreateGroup:
		return "create_group"
	case ViewModeRenameGroup:
		return "rename_group"
	case ViewModeDeleteGroup:
		return "delete_group"
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

type ListItemKind int

const (
	ListItemGroup ListItemKind = iota
	ListItemHost
	ListItemNewGroup
)

// ListItem represents a selectable item in the grouped list view.
type ListItem struct {
	Kind      ListItemKind
	GroupName string // for groups and host membership (empty means ungrouped)
	Host      Host   // valid for Kind==ListItemHost
	Count     int    // host count for group header
}

type SpotlightItemKind int

const (
	SpotlightItemGroup SpotlightItemKind = iota
	SpotlightItemHost
)

// SpotlightItem represents one row in spotlight results.
type SpotlightItem struct {
	Kind      SpotlightItemKind
	GroupName string
	Host      Host
	Score     int
	Indent    int
}
