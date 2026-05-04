package app

import (
	"time"

	"github.com/Vansh-Raja/SSHThing/internal/teams"
	tea "github.com/charmbracelet/bubbletea"
)

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
	PageProfile  = 1
	PageSettings = 2
	PageTokens   = 3
	PageTeams    = 4
)

const (
	appModePersonal = iota
	appModeTeams
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
	OverlayImportHost  = 11
	OverlayKeyEditor   = 12
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
	SpotlightItemCommand
)

// SpotlightItem represents one row in spotlight results.
type SpotlightItem struct {
	Kind      SpotlightItemKind
	GroupName string
	Host      Host
	TeamHost  teams.TeamHost
	Command   string
	Detail    string
	Team      teams.TeamSummary
	Score     int
	Indent    int
}

type commandContext string

const (
	commandContextGlobal   commandContext = "global"
	commandContextHome     commandContext = "home"
	commandContextTeams    commandContext = "teams"
	commandContextProfile  commandContext = "profile"
	commandContextSettings commandContext = "settings"
	commandContextTokens   commandContext = "tokens"
)

type appCommand struct {
	ID          string
	Name        string
	Aliases     []string
	Title       string
	Description string
	Contexts    []commandContext
	Danger      bool
	Run         func(Model) (tea.Model, tea.Cmd)
	Enabled     func(Model) (bool, string)
}

type commandItem struct {
	Command        appCommand
	Score          int
	Disabled       bool
	DisabledReason string
}

type teamsImportConflictState struct {
	PersonalHost Host
	ExistingHost teams.TeamHostDetail
	Cursor       int
}

// ── Token mode constants ──────────────────────────────────────────────

const (
	tokenModeList = iota
	tokenModeCreateName
	tokenModeCreateScope
)

const (
	profileStateSignedOut = iota
	profileStateSigningIn
	profileStateSignedIn
)

const (
	teamsStateZero = iota
	teamsStateHosts
)
