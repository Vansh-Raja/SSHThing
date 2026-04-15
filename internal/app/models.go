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

// ── Page constants ────────────────────────────────────────────────────

const (
	PageHome     = 0
	PageSettings = 1
	PageTokens   = 2
	PageTeams    = 3
	NumPages     = 4
)

// ── Overlay constants ─────────────────────────────────────────────────

const (
	OverlayNone        = 0
	OverlayLogin       = 1
	OverlaySetup       = 2
	OverlayHelp        = 3
	OverlaySearch      = 4
	OverlayAddHost     = 5
	OverlayDeleteHost  = 6
	OverlayCreateGroup = 7
	OverlayRenameGroup = 8
	OverlayDeleteGroup = 9
	OverlayQuit        = 10
)

// ── List types ────────────────────────────────────────────────────────

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

// ── Token mode constants ──────────────────────────────────────────────

const (
	tokenModeList = iota
	tokenModeCreateName
	tokenModeCreateScope
)
