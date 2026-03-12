// sshmock6 — "Zen"
// Ultra-minimal, whitespace-driven design with built-in theming,
// sidebar page navigation, and explicit background painting.
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// ── theme system ──────────────────────────────────────────────────────

type theme struct {
	name                                  string
	base, mantle, crust                   lipgloss.Color // backgrounds
	surface0, surface1, surface2          lipgloss.Color // elevated/borders
	text, subtext, overlay                lipgloss.Color // text levels
	accent, green, yellow, red, sky, pink lipgloss.Color // semantic
}

var themes = []theme{
	{
		name:     "Catppuccin Latte",
		base:     "#eff1f5",
		mantle:   "#e6e9ef",
		crust:    "#dce0e8",
		surface0: "#ccd0da",
		surface1: "#bcc0cc",
		surface2: "#acb0be",
		text:     "#4c4f69",
		subtext:  "#6c6f85",
		overlay:  "#9ca0b0",
		accent:   "#1e66f5",
		green:    "#40a02b",
		yellow:   "#df8e1d",
		red:      "#d20f39",
		sky:      "#04a5e5",
		pink:     "#ea76cb",
	},
	{
		name:     "Catppuccin Mocha",
		base:     "#1e1e2e",
		mantle:   "#181825",
		crust:    "#11111b",
		surface0: "#363a4f",
		surface1: "#45475a",
		surface2: "#585b70",
		text:     "#cdd6f4",
		subtext:  "#bac2de",
		overlay:  "#6c7086",
		accent:   "#89b4fa",
		green:    "#a6e3a1",
		yellow:   "#f9e2af",
		red:      "#f38ba8",
		sky:      "#89dceb",
		pink:     "#f5c2e7",
	},
	{
		name:     "Tokyo Night",
		base:     "#1a1b26",
		mantle:   "#16161e",
		crust:    "#0c0e14",
		surface0: "#292e42",
		surface1: "#545c7e",
		surface2: "#3b4261",
		text:     "#c0caf5",
		subtext:  "#a9b1d6",
		overlay:  "#565f89",
		accent:   "#7aa2f7",
		green:    "#9ece6a",
		yellow:   "#e0af68",
		red:      "#f7768e",
		sky:      "#7dcfff",
		pink:     "#bb9af7",
	},
	{
		name:     "Dracula",
		base:     "#282a36",
		mantle:   "#21222c",
		crust:    "#1a1b23",
		surface0: "#44475a",
		surface1: "#6272a4",
		surface2: "#6272a4",
		text:     "#f8f8f2",
		subtext:  "#6272a4",
		overlay:  "#44475a",
		accent:   "#8be9fd",
		green:    "#50fa7b",
		yellow:   "#f1fa8c",
		red:      "#ff5555",
		sky:      "#8be9fd",
		pink:     "#ff79c6",
	},
	{
		name:     "Nord",
		base:     "#2e3440",
		mantle:   "#3b4252",
		crust:    "#434c5e",
		surface0: "#4c566a",
		surface1: "#4c566a",
		surface2: "#616e88",
		text:     "#eceff4",
		subtext:  "#d8dee9",
		overlay:  "#4c566a",
		accent:   "#5e81ac",
		green:    "#a3be8c",
		yellow:   "#ebcb8b",
		red:      "#bf616a",
		sky:      "#88c0d0",
		pink:     "#b48ead",
	},
	{
		name:     "Gruvbox",
		base:     "#282828",
		mantle:   "#32302f",
		crust:    "#1d2021",
		surface0: "#3c3836",
		surface1: "#504945",
		surface2: "#665c54",
		text:     "#ebdbb2",
		subtext:  "#d5c4a1",
		overlay:  "#928374",
		accent:   "#83a598",
		green:    "#b8bb26",
		yellow:   "#fabd2f",
		red:      "#fb4934",
		sky:      "#8ec07c",
		pink:     "#d3869b",
	},
	{
		name:     "Rose Pine",
		base:     "#191724",
		mantle:   "#1f1d2e",
		crust:    "#26233a",
		surface0: "#403d52",
		surface1: "#524f67",
		surface2: "#6e6a86",
		text:     "#e0def4",
		subtext:  "#908caa",
		overlay:  "#6e6a86",
		accent:   "#31748f",
		green:    "#9ccfd8",
		yellow:   "#f6c177",
		red:      "#eb6f92",
		sky:      "#9ccfd8",
		pink:     "#c4a7e7",
	},
}

// ── icon system ──────────────────────────────────────────────────────

type iconSet struct {
	name string
	// Sidebar
	home, settings, tokens string
	// Status
	connected, idle, offline string
	// Markers
	activeMarker, inactiveMarker string
	// Groups
	expanded, collapsed string
	// Selection
	selected, focused string
	// Nav
	leftArrow, rightArrow string
	// Input
	bar, cursor string
	// Misc
	truncation, rule string
	// New
	lock, warning, errorIcon, success string
	folder, edit, deleteIcon, add     string
	save, cancel, shield              string
}

var unicodeIcons = iconSet{
	name:            "Unicode",
	home:            "⌂",
	settings:        "○",
	tokens:          "◇",
	connected:       "●",
	idle:            "○",
	offline:         "·",
	activeMarker:    "•",
	inactiveMarker:  "·",
	expanded:        "▿",
	collapsed:       "▹",
	selected:        "▸",
	focused:         "→",
	leftArrow:       "◄",
	rightArrow:      "►",
	bar:             "▏",
	cursor:          "█",
	truncation:      "…",
	rule:            "─",
	lock:            "◆",
	warning:         "△",
	errorIcon:       "✗",
	success:         "✓",
	folder:          "▪",
	edit:            "~",
	deleteIcon:      "×",
	add:             "+",
	save:            "▸",
	cancel:          "○",
	shield:          "◇",
}

var nerdFontIcons = iconSet{
	name:            "Nerd Font",
	home:            "\uf015",
	settings:        "\uf013",
	tokens:          "\uf084",
	connected:       "\uf058",
	idle:            "\uf192",
	offline:         "\uf10c",
	activeMarker:    "\uf111",
	inactiveMarker:  "\uf10c",
	expanded:        "\uf078",
	collapsed:       "\uf054",
	selected:        "\uf0da",
	focused:         "\uf061",
	leftArrow:       "\uf053",
	rightArrow:      "\uf054",
	bar:             "▏",
	cursor:          "█",
	truncation:      "\uf141",
	rule:            "─",
	lock:            "\uf023",
	warning:         "\uf071",
	errorIcon:       "\uf00d",
	success:         "\uf00c",
	folder:          "\uf07b",
	edit:            "\uf044",
	deleteIcon:      "\uf1f8",
	add:             "\uf067",
	save:            "\uf0c7",
	cancel:          "\uf05e",
	shield:          "\uf132",
}

var iconPresets = []iconSet{unicodeIcons, nerdFontIcons}

// ── pages & overlays ──────────────────────────────────────────────────

const (
	pageHome     = 0
	pageSettings = 1
	pageTokens   = 2
	numPages     = 3
)

const (
	overlayNone        = 0
	overlayHelp        = 1
	overlaySearch      = 2
	overlayAddHost     = 3
	overlayQuit        = 4
	overlayLogin       = 5
	overlaySetup       = 6
	overlayDeleteHost  = 7
	overlayCreateGroup = 8
	overlayRenameGroup = 9
	overlayDeleteGroup = 10
)

// ── sidebar ───────────────────────────────────────────────────────────

const sidebarW = 4

type pageIndicator struct {
	icon  string
	index int
}

func (m model) pageIcons() []pageIndicator {
	return []pageIndicator{
		{m.icons.home, pageHome},
		{m.icons.settings, pageSettings},
		{m.icons.tokens, pageTokens},
	}
}

// ── data types ────────────────────────────────────────────────────────

type host struct {
	label, group, hostname, user, keyType string
	port                                  int
	tags                                  []string
	status                                int
	lastSSH                               string
}

type grp struct {
	name      string
	collapsed bool
}

type listItem struct {
	isGroup    bool
	isNewGroup bool
	grp        grp
	host       host
	hostCount  int
}

type formField struct {
	label  string
	value  string
	cursor int
	masked bool
}

func newFormField(label string) formField {
	return formField{label: label}
}

func newMaskedField(label string) formField {
	return formField{label: label, masked: true}
}

type settingsItem struct {
	category string
	label    string
	value    string
	kind     int // 0=toggle, 1=enum, 2=readonly
	options  []string
	optIdx   int
}

type token struct {
	name    string
	scope   string
	created string
	lastUse string
}

// ── mock data ─────────────────────────────────────────────────────────

func mockGroups() []grp {
	return []grp{
		{name: "Production"},
		{name: "Staging"},
		{name: "Personal"},
	}
}

func mockHosts() []host {
	return []host{
		{label: "api-gateway", group: "Production", hostname: "api.prod.example.com", user: "deploy", port: 22, keyType: "ed25519", tags: []string{"api", "nginx"}, status: 2, lastSSH: "2 minutes ago"},
		{label: "database", group: "Production", hostname: "10.0.1.50", user: "root", port: 5432, keyType: "rsa", tags: []string{"postgres"}, status: 2, lastSSH: "active session"},
		{label: "redis", group: "Production", hostname: "10.0.1.75", user: "admin", port: 6379, keyType: "ecdsa", tags: []string{"cache"}, status: 1, lastSSH: "6 hours ago"},
		{label: "workers", group: "Production", hostname: "10.0.1.100", user: "deploy", port: 22, keyType: "ed25519", tags: []string{"sidekiq"}, status: 0, lastSSH: "3 days ago"},
		{label: "app", group: "Staging", hostname: "staging.example.com", user: "dev", port: 2222, keyType: "password", tags: []string{"web"}, status: 1, lastSSH: "1 hour ago"},
		{label: "db", group: "Staging", hostname: "stg-db.local", user: "postgres", port: 5432, keyType: "ecdsa", tags: []string{"database"}, status: 0, lastSSH: "yesterday"},
		{label: "dev-machine", group: "Personal", hostname: "192.168.1.100", user: "vansh", port: 22, keyType: "ed25519", tags: []string{"local"}, status: 0, lastSSH: "just now"},
		{label: "raspberry-pi", group: "Personal", hostname: "192.168.1.200", user: "pi", port: 22, keyType: "rsa", tags: []string{"arm", "k3s"}, status: 0, lastSSH: "2 weeks ago"},
	}
}

func mockSettings() []settingsItem {
	return []settingsItem{
		{category: "interface", label: "Vim navigation", value: "on", kind: 0},
		{category: "interface", label: "Show icons", value: "off", kind: 0},
		{category: "interface", label: "Theme", value: "Catppuccin Mocha", kind: 1,
			options: []string{"Catppuccin Latte", "Catppuccin Mocha", "Tokyo Night", "Dracula", "Nord", "Gruvbox", "Rose Pine"},
			optIdx:  1},
		{category: "interface", label: "Icon set", value: "Unicode", kind: 1,
			options: []string{"Unicode", "Nerd Font"},
			optIdx:  0},
		{category: "ssh", label: "Host key policy", value: "accept-new", kind: 1, options: []string{"accept-new", "strict", "off"}, optIdx: 0},
		{category: "ssh", label: "Keepalive interval", value: "30s", kind: 1, options: []string{"0s", "15s", "30s", "60s"}, optIdx: 2},
		{category: "ssh", label: "TERM mode", value: "auto", kind: 1, options: []string{"auto", "xterm", "xterm-256color"}, optIdx: 0},
		{category: "ssh", label: "Password auto-login", value: "off", kind: 0},
		{category: "mounts", label: "Enabled", value: "off", kind: 0},
		{category: "mounts", label: "Default remote path", value: "~", kind: 2},
		{category: "mounts", label: "Quit behavior", value: "prompt", kind: 1, options: []string{"prompt", "always unmount", "leave mounted"}, optIdx: 0},
		{category: "sync", label: "Enabled", value: "off", kind: 0},
		{category: "sync", label: "Repository URL", value: "—", kind: 2},
		{category: "sync", label: "Branch", value: "main", kind: 2},
	}
}

func mockTokens() []token {
	return []token{
		{name: "deploy-ci", scope: "read, connect", created: "3 days ago", lastUse: "2 hours ago"},
		{name: "backup-cron", scope: "connect", created: "2 weeks ago", lastUse: "12 hours ago"},
		{name: "monitoring", scope: "read", created: "1 month ago", lastUse: "never"},
	}
}

func buildList(groups []grp, hosts []host) []listItem {
	var items []listItem
	for _, g := range groups {
		c := 0
		for _, h := range hosts {
			if h.group == g.name {
				c++
			}
		}
		items = append(items, listItem{isGroup: true, grp: g, hostCount: c})
		if !g.collapsed {
			for _, h := range hosts {
				if h.group == g.name {
					items = append(items, listItem{host: h})
				}
			}
		}
	}
	items = append(items, listItem{isNewGroup: true})
	return items
}

// ── model ─────────────────────────────────────────────────────────────

// hexToOSC11 converts "#RRGGBB" to XParseColor format "rgb:RR/GG/BB"
// which is accepted by Windows Terminal, iTerm2, Kitty, and xterm.
func hexToOSC11(hex string) string {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return "rgb:00/00/00"
	}
	return fmt.Sprintf("rgb:%s/%s/%s", hex[0:2], hex[2:4], hex[4:6])
}

// ── model ─────────────────────────────────────────────────────────────

type model struct {
	// theme
	theme    theme
	themeIdx int

	// icons
	icons   iconSet
	iconIdx int

	// navigation
	page    int
	overlay int

	// core data
	items  []listItem
	groups []grp
	hosts  []host
	cursor int
	w, h   int
	tick   int

	// search state
	searchInput  string
	searchCursor int
	armedSFTP    bool

	// add-host form state
	formFields   []formField
	formFocus    int
	formEditing  bool
	formGroups   []string
	formGroupIdx int
	formAuth     int // 0=password, 1=paste key, 2=generate
	formAuthOpts []string
	formKeyTypes []string
	formKeyIdx   int
	formEditIdx  int // -1=add mode, >=0=editing m.hosts[idx]

	// settings state
	settings       []settingsItem
	settingsCur    int
	settingsFilter string
	settingsSearch bool

	// tokens state
	tokens    []token
	tokensCur int

	// login/setup state
	isFirstRun  bool
	isLoggedIn  bool
	loginFields []formField
	setupFields []formField
	loginFocus  int
	loginError  string

	// delete host
	deleteHostIdx int
	deleteCursor  int // 0=Delete, 1=Cancel

	// group CRUD
	groupInput       string
	groupInputCursor int
	groupFocus       int // 0=input, 1=submit, 2=cancel
	groupEditIdx     int // index into m.groups (-1 for create)
	groupDeleteIdx   int
	groupDeleteCursor int // 0=Delete, 1=Cancel

	// quit confirmation
	quitCursor int // 0..2
	mockMounts []string
}

func initialModel() model {
	g := mockGroups()
	h := mockHosts()

	groupNames := make([]string, len(g))
	for i, gr := range g {
		groupNames[i] = gr.name
	}

	return model{
		theme:        themes[1], // Catppuccin Mocha default
		themeIdx:     1,
		icons:        unicodeIcons,
		iconIdx:      0,
		groups:       g,
		hosts:        h,
		items:        buildList(g, h),
		formFields:   initFormFields(),
		formGroups:   groupNames,
		formAuthOpts: []string{"Password", "Paste Key", "Generate Key"},
		formKeyTypes: []string{"ed25519", "rsa", "ecdsa"},
		formEditIdx:  -1,
		settings:     mockSettings(),
		tokens:       mockTokens(),
		isFirstRun:   true,
		overlay:      overlaySetup,
		loginFields:  []formField{newMaskedField("password")},
		setupFields:  []formField{newMaskedField("master password"), newMaskedField("confirm password")},
		mockMounts:   []string{"/mnt/sshfs/api-gateway", "/mnt/sshfs/database"},
	}
}

func initFormFields() []formField {
	return []formField{
		newFormField("label"),
		newFormField("tags"),
		newFormField("hostname"),
		newFormField("port"),
		newFormField("username"),
		newMaskedField("password"),
	}
}

// field indices
const (
	ffLabel    = 0
	ffTags     = 1
	ffHostname = 2
	ffPort     = 3
	ffUsername  = 4
	ffAuthDet  = 5
	// virtual focus targets (not text fields):
	ffGroup    = 6
	ffAuthMeth = 7
	ffSave     = 8
)

type tickMsg time.Time

func (m model) Init() tea.Cmd {
	return tea.Batch(
		// Set terminal default bg AFTER alt screen is active (Init runs post-alt-screen)
		setTermBgCmd(string(m.theme.base)),
		tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) }),
	)
}

// ── update ────────────────────────────────────────────────────────────

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
	case tickMsg:
		m.tick++
		return m, tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
	case tea.KeyMsg:
		// Overlays handle their own keys first
		if m.overlay != overlayNone {
			switch m.overlay {
			case overlayHelp:
				m.overlay = overlayNone
				return m, nil
			case overlaySearch:
				return m.updateSearch(msg)
			case overlayAddHost:
				return m.updateAddHost(msg)
			case overlayQuit:
				return m.updateQuit(msg)
			case overlayLogin:
				return m.updateLogin(msg)
			case overlaySetup:
				return m.updateSetup(msg)
			case overlayDeleteHost:
				return m.updateDeleteHost(msg)
			case overlayCreateGroup:
				return m.updateCreateGroup(msg)
			case overlayRenameGroup:
				return m.updateRenameGroup(msg)
			case overlayDeleteGroup:
				return m.updateDeleteGroup(msg)
			}
		}

		// Global keys (no overlay active)
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			if m.settingsSearch {
				break // let settings handler deal with it
			}
			if m.page != pageHome {
				m.page = pageHome
				return m, nil
			}
			m.overlay = overlayQuit
			m.quitCursor = 0 // default to Yes
			return m, nil
		case "shift+tab":
			m.page = (m.page + 1) % numPages
			return m, nil
		case "?":
			m.overlay = overlayHelp
			return m, nil
		case "/":
			if m.page == pageSettings {
				m.settingsSearch = true
				m.settingsFilter = ""
				return m, nil
			}
			m.overlay = overlaySearch
			m.searchInput = ""
			m.searchCursor = 0
			m.armedSFTP = false
			return m, nil
		case "esc":
			if m.page == pageSettings && m.settingsFilter != "" {
				m.settingsFilter = ""
				m.settingsSearch = false
				return m, nil
			}
			if m.page != pageHome {
				m.page = pageHome
				return m, nil
			}
		}

		// Page-specific keys
		switch m.page {
		case pageHome:
			return m.updateHome(msg)
		case pageSettings:
			return m.updateSettings(msg)
		case pageTokens:
			return m.updateTokens(msg)
		}
	}
	return m, nil
}

func (m model) updateHome(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if m.cursor < len(m.items)-1 {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "enter", " ":
		if m.cursor < len(m.items) {
			item := m.items[m.cursor]
			if item.isGroup {
				n := item.grp.name
				for i := range m.groups {
					if m.groups[i].name == n {
						m.groups[i].collapsed = !m.groups[i].collapsed
					}
				}
				m.items = buildList(m.groups, m.hosts)
				if m.cursor >= len(m.items) {
					m.cursor = len(m.items) - 1
				}
			} else if item.isNewGroup {
				m.overlay = overlayCreateGroup
				m.groupInput = ""
				m.groupInputCursor = 0
				m.groupFocus = 0
			}
		}
	case "a":
		if m.cursor < len(m.items) && m.items[m.cursor].isNewGroup {
			m.overlay = overlayCreateGroup
			m.groupInput = ""
			m.groupInputCursor = 0
			m.groupFocus = 0
		} else {
			m.overlay = overlayAddHost
			m.formFields = initFormFields()
			m.formFields[ffPort].value = "22"
			m.formFocus = 0
			m.formEditing = false
			m.formGroupIdx = 0
			m.formAuth = 0
			m.formKeyIdx = 0
			m.formEditIdx = -1
		}
	case "e":
		if m.cursor < len(m.items) {
			item := m.items[m.cursor]
			if item.isGroup {
				for i, g := range m.groups {
					if g.name == item.grp.name {
						m.groupEditIdx = i
						break
					}
				}
				m.overlay = overlayRenameGroup
				m.groupInput = item.grp.name
				m.groupInputCursor = utf8.RuneCountInString(m.groupInput)
				m.groupFocus = 0
			} else if !item.isNewGroup {
				for i, h := range m.hosts {
					if h.label == item.host.label && h.group == item.host.group {
						m.formEditIdx = i
						break
					}
				}
				m.overlay = overlayAddHost
				m.formFields = initFormFields()
				m.formFields[ffLabel].value = item.host.label
				m.formFields[ffTags].value = strings.Join(item.host.tags, ", ")
				m.formFields[ffHostname].value = item.host.hostname
				m.formFields[ffPort].value = strconv.Itoa(item.host.port)
				m.formFields[ffUsername].value = item.host.user
				for i, g := range m.formGroups {
					if g == item.host.group {
						m.formGroupIdx = i
						break
					}
				}
				m.formFocus = 0
				m.formEditing = false
				m.formAuth = 0
				m.formKeyIdx = 0
			}
		}
	case "d":
		if m.cursor < len(m.items) {
			item := m.items[m.cursor]
			if item.isGroup {
				if len(m.groups) > 1 {
					for i, g := range m.groups {
						if g.name == item.grp.name {
							m.groupDeleteIdx = i
							break
						}
					}
					m.overlay = overlayDeleteGroup
					m.groupDeleteCursor = 1
				}
			} else if !item.isNewGroup {
				for i, h := range m.hosts {
					if h.label == item.host.label && h.group == item.host.group {
						m.deleteHostIdx = i
						break
					}
				}
				m.overlay = overlayDeleteHost
				m.deleteCursor = 1
			}
		}
	case ",":
		m.page = pageSettings
	case "ctrl+g":
		m.overlay = overlayCreateGroup
		m.groupInput = ""
		m.groupInputCursor = 0
		m.groupFocus = 0
	}
	return m, nil
}

func (m model) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch key {
	case "esc":
		m.overlay = overlayNone
	case "enter":
		m.overlay = overlayNone
	case "backspace":
		if len(m.searchInput) > 0 {
			m.searchInput = m.searchInput[:len(m.searchInput)-1]
			m.searchCursor = 0
		}
	case "up", "ctrl+k":
		if m.searchCursor > 0 {
			m.searchCursor--
		}
	case "down", "ctrl+j":
		results := m.filterHosts()
		if m.searchCursor < len(results)-1 {
			m.searchCursor++
		}
	case "ctrl+s":
		m.armedSFTP = !m.armedSFTP
	default:
		if len(key) == 1 && key[0] >= 32 && key[0] < 127 {
			m.searchInput += key
			m.searchCursor = 0
		}
	}
	return m, nil
}

type navEntry struct {
	field, row, col int
}

// formNavMap returns the visual layout of form fields based on terminal width.
func (m model) formNavMap() []navEntry {
	if m.w >= 60 {
		return []navEntry{
			{ffLabel, 0, 0},
			{ffGroup, 1, 0},
			{ffTags, 2, 0},
			{ffHostname, 3, 0},
			{ffPort, 3, 1},
			{ffUsername, 4, 0},
			{ffAuthMeth, 5, 0},
			{ffAuthDet, 6, 0},
			{ffSave, 7, 0},
		}
	}
	return []navEntry{
		{ffLabel, 0, 0},
		{ffGroup, 1, 0},
		{ffTags, 2, 0},
		{ffHostname, 3, 0},
		{ffPort, 4, 0},
		{ffUsername, 5, 0},
		{ffAuthMeth, 6, 0},
		{ffAuthDet, 7, 0},
		{ffSave, 8, 0},
	}
}

// formIsTextField returns true if the field index is a text input (not selector/button).
func formIsTextField(f int) bool {
	return f >= 0 && f <= ffAuthDet
}

func (m model) formNavFind() (row, col int) {
	for _, e := range m.formNavMap() {
		if e.field == m.formFocus {
			return e.row, e.col
		}
	}
	return 0, 0
}

func (m model) formNavByRowCol(targetRow, targetCol int) int {
	nav := m.formNavMap()
	// Try exact match first
	for _, e := range nav {
		if e.row == targetRow && e.col == targetCol {
			return e.field
		}
	}
	// Fall back to col 0 on that row
	for _, e := range nav {
		if e.row == targetRow && e.col == 0 {
			return e.field
		}
	}
	return -1
}

func (m model) formNavMaxRow() int {
	nav := m.formNavMap()
	return nav[len(nav)-1].row
}

func (m *model) formExitEditing() {
	if m.formEditing {
		m.formEditing = false
	}
}

func (m *model) formEnterEditing() {
	if formIsTextField(m.formFocus) {
		m.formEditing = true
		// Place cursor at end of field value
		m.formFields[m.formFocus].cursor = utf8.RuneCountInString(m.formFields[m.formFocus].value)
	}
}

func (m model) updateAddHost(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// 1. Escape
	if key == "esc" {
		if m.formEditing {
			m.formExitEditing()
			return m, nil
		}
		m.overlay = overlayNone
		return m, nil
	}

	// 2. Tab / shift+tab — always navigate, exit editing
	if key == "tab" {
		m.formExitEditing()
		m.formFocus = (m.formFocus + 1) % 9
		return m, nil
	}
	if key == "shift+tab" {
		m.formExitEditing()
		m.formFocus = (m.formFocus - 1 + 9) % 9
		return m, nil
	}

	// 3. Up/down (and j/k when not editing) — always navigate via nav map, exit editing
	isUp := key == "up" || (key == "k" && !m.formEditing && !formIsTextField(m.formFocus))
	isDown := key == "down" || (key == "j" && !m.formEditing && !formIsTextField(m.formFocus))
	if isUp || isDown {
		if isUp {
			key = "up"
		} else {
			key = "down"
		}
		m.formExitEditing()
		curRow, curCol := m.formNavFind()
		maxRow := m.formNavMaxRow()
		var newRow int
		if key == "up" {
			newRow = curRow - 1
			if newRow < 0 {
				newRow = 0
			}
		} else {
			newRow = curRow + 1
			if newRow > maxRow {
				newRow = maxRow
			}
		}
		if f := m.formNavByRowCol(newRow, curCol); f >= 0 {
			m.formFocus = f
		}
		return m, nil
	}

	// 4. Left/right (and h/l for selectors when not editing text)
	isLeft := key == "left" || (key == "h" && !m.formEditing && !formIsTextField(m.formFocus))
	isRight := key == "right" || (key == "l" && !m.formEditing && !formIsTextField(m.formFocus))
	if isLeft || isRight {
		// Normalize to left/right
		if isLeft {
			key = "left"
		} else {
			key = "right"
		}
		// Editing a text field: move cursor
		if m.formEditing && formIsTextField(m.formFocus) {
			f := &m.formFields[m.formFocus]
			runes := []rune(f.value)
			if key == "left" {
				if f.cursor > 0 {
					f.cursor--
				}
			} else {
				if f.cursor < len(runes) {
					f.cursor++
				}
			}
			return m, nil
		}
		// Selector fields: cycle option
		if m.formFocus == ffGroup {
			if key == "left" {
				m.formGroupIdx = (m.formGroupIdx - 1 + len(m.formGroups)) % len(m.formGroups)
			} else {
				m.formGroupIdx = (m.formGroupIdx + 1) % len(m.formGroups)
			}
			return m, nil
		}
		if m.formFocus == ffAuthMeth {
			if key == "left" {
				m.formAuth = (m.formAuth - 1 + len(m.formAuthOpts)) % len(m.formAuthOpts)
			} else {
				m.formAuth = (m.formAuth + 1) % len(m.formAuthOpts)
			}
			return m, nil
		}
		// Not editing text: navigate left/right in nav map
		curRow, curCol := m.formNavFind()
		var newCol int
		if key == "left" {
			newCol = curCol - 1
			if newCol < 0 {
				newCol = 0
			}
		} else {
			newCol = curCol + 1
		}
		if f := m.formNavByRowCol(curRow, newCol); f >= 0 && f != m.formFocus {
			m.formFocus = f
		}
		return m, nil
	}

	// 5. Enter
	if key == "enter" {
		if m.formFocus == ffSave {
			if m.formFields[ffLabel].value != "" {
				p, _ := strconv.Atoi(m.formFields[ffPort].value)
				if p == 0 {
					p = 22
				}
				newHost := host{
					label:    m.formFields[ffLabel].value,
					group:    m.formGroups[m.formGroupIdx],
					hostname: m.formFields[ffHostname].value,
					user:     m.formFields[ffUsername].value,
					port:     p,
					keyType:  "ed25519",
					tags:     splitTags(m.formFields[ffTags].value),
					status:   0,
					lastSSH:  "never",
				}
				if m.formEditIdx >= 0 {
					newHost.status = m.hosts[m.formEditIdx].status
					newHost.lastSSH = m.hosts[m.formEditIdx].lastSSH
					m.hosts[m.formEditIdx] = newHost
				} else {
					m.hosts = append(m.hosts, newHost)
				}
				m.items = buildList(m.groups, m.hosts)
			}
			m.overlay = overlayNone
			return m, nil
		}
		if formIsTextField(m.formFocus) && !m.formEditing {
			m.formEnterEditing()
			return m, nil
		}
		// Editing: exit and move down
		m.formExitEditing()
		m.formFocus = (m.formFocus + 1) % 9
		return m, nil
	}

	// 6. Backspace
	if key == "backspace" {
		if formIsTextField(m.formFocus) {
			if !m.formEditing {
				m.formEnterEditing()
			}
			f := &m.formFields[m.formFocus]
			runes := []rune(f.value)
			if f.cursor > 0 {
				f.value = string(runes[:f.cursor-1]) + string(runes[f.cursor:])
				f.cursor--
			}
		}
		return m, nil
	}

	// 7. Printable characters
	if len(key) == 1 && key[0] >= 32 && key[0] < 127 {
		if formIsTextField(m.formFocus) {
			if !m.formEditing {
				m.formEnterEditing()
			}
			f := &m.formFields[m.formFocus]
			runes := []rune(f.value)
			f.value = string(runes[:f.cursor]) + key + string(runes[f.cursor:])
			f.cursor++
		}
		return m, nil
	}

	return m, nil
}

func (m model) updateLogin(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch key {
	case "esc":
		return m, tea.Quit
	case "enter":
		if m.loginFields[0].value != "" {
			m.isLoggedIn = true
			m.overlay = overlayNone
		} else {
			m.loginError = "password required"
		}
	case "backspace":
		f := &m.loginFields[0]
		runes := []rune(f.value)
		if f.cursor > 0 {
			f.value = string(runes[:f.cursor-1]) + string(runes[f.cursor:])
			f.cursor--
		}
	case "left":
		f := &m.loginFields[0]
		if f.cursor > 0 {
			f.cursor--
		}
	case "right":
		f := &m.loginFields[0]
		if f.cursor < utf8.RuneCountInString(f.value) {
			f.cursor++
		}
	default:
		if len(key) == 1 && key[0] >= 32 && key[0] < 127 {
			f := &m.loginFields[0]
			runes := []rune(f.value)
			f.value = string(runes[:f.cursor]) + key + string(runes[f.cursor:])
			f.cursor++
			m.loginError = ""
		}
	}
	return m, nil
}

func (m model) updateSetup(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch key {
	case "esc":
		return m, tea.Quit
	case "tab":
		m.loginFocus = (m.loginFocus + 1) % 3
	case "shift+tab":
		m.loginFocus = (m.loginFocus - 1 + 3) % 3
	case "up":
		if m.loginFocus > 0 {
			m.loginFocus--
		}
	case "down":
		if m.loginFocus < 2 {
			m.loginFocus++
		}
	case "enter":
		if m.loginFocus == 2 {
			pw := m.setupFields[0].value
			confirm := m.setupFields[1].value
			if len(pw) < 8 {
				m.loginError = "password must be at least 8 characters"
			} else if pw != confirm {
				m.loginError = "passwords don't match"
			} else {
				m.isFirstRun = false
				m.isLoggedIn = true
				m.overlay = overlayNone
			}
		} else {
			m.loginFocus = (m.loginFocus + 1) % 3
		}
	case "backspace":
		if m.loginFocus < 2 {
			f := &m.setupFields[m.loginFocus]
			runes := []rune(f.value)
			if f.cursor > 0 {
				f.value = string(runes[:f.cursor-1]) + string(runes[f.cursor:])
				f.cursor--
			}
		}
	case "left":
		if m.loginFocus < 2 {
			f := &m.setupFields[m.loginFocus]
			if f.cursor > 0 {
				f.cursor--
			}
		}
	case "right":
		if m.loginFocus < 2 {
			f := &m.setupFields[m.loginFocus]
			if f.cursor < utf8.RuneCountInString(f.value) {
				f.cursor++
			}
		}
	default:
		if m.loginFocus < 2 && len(key) == 1 && key[0] >= 32 && key[0] < 127 {
			f := &m.setupFields[m.loginFocus]
			runes := []rune(f.value)
			f.value = string(runes[:f.cursor]) + key + string(runes[f.cursor:])
			f.cursor++
			m.loginError = ""
		}
	}
	return m, nil
}

func (m model) updateDeleteHost(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.overlay = overlayNone
	case "left", "right", "up", "down", "tab", "shift+tab", "h", "l", "j", "k":
		m.deleteCursor = 1 - m.deleteCursor
	case "enter":
		if m.deleteCursor == 0 {
			m.hosts = append(m.hosts[:m.deleteHostIdx], m.hosts[m.deleteHostIdx+1:]...)
			m.items = buildList(m.groups, m.hosts)
			if m.cursor >= len(m.items) {
				m.cursor = len(m.items) - 1
			}
		}
		m.overlay = overlayNone
	}
	return m, nil
}

func (m model) updateCreateGroup(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// When on the text input, handle arrow keys + text editing directly
	if m.groupFocus == 0 {
		switch key {
		case "esc":
			m.overlay = overlayNone
		case "tab":
			m.groupFocus = 1
		case "shift+tab":
			m.groupFocus = 2
		case "up":
			// already at top, do nothing
		case "down":
			m.groupFocus = 1
		case "enter":
			m.groupFocus = 1
		case "left":
			if m.groupInputCursor > 0 {
				m.groupInputCursor--
			}
		case "right":
			if m.groupInputCursor < utf8.RuneCountInString(m.groupInput) {
				m.groupInputCursor++
			}
		case "backspace":
			runes := []rune(m.groupInput)
			if m.groupInputCursor > 0 {
				m.groupInput = string(runes[:m.groupInputCursor-1]) + string(runes[m.groupInputCursor:])
				m.groupInputCursor--
			}
		default:
			if len(key) == 1 && key[0] >= 32 && key[0] < 127 {
				runes := []rune(m.groupInput)
				m.groupInput = string(runes[:m.groupInputCursor]) + key + string(runes[m.groupInputCursor:])
				m.groupInputCursor++
			}
		}
		return m, nil
	}

	// On buttons (groupFocus 1 or 2)
	switch key {
	case "esc":
		m.overlay = overlayNone
	case "tab":
		m.groupFocus = (m.groupFocus + 1) % 3
	case "shift+tab":
		m.groupFocus = (m.groupFocus - 1 + 3) % 3
	case "up", "k":
		m.groupFocus = 0
	case "down", "j":
		// already at bottom
	case "left", "h":
		if m.groupFocus == 2 {
			m.groupFocus = 1
		}
	case "right", "l":
		if m.groupFocus == 1 {
			m.groupFocus = 2
		}
	case "enter":
		if m.groupFocus == 1 {
			if m.groupInput != "" {
				m.groups = append(m.groups, grp{name: m.groupInput})
				m.formGroups = append(m.formGroups, m.groupInput)
				m.items = buildList(m.groups, m.hosts)
			}
			m.overlay = overlayNone
		} else if m.groupFocus == 2 {
			m.overlay = overlayNone
		}
	}
	return m, nil
}

func (m model) updateRenameGroup(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// When on the text input, handle arrow keys + text editing directly
	if m.groupFocus == 0 {
		switch key {
		case "esc":
			m.overlay = overlayNone
		case "tab":
			m.groupFocus = 1
		case "shift+tab":
			m.groupFocus = 2
		case "up":
			// already at top
		case "down":
			m.groupFocus = 1
		case "enter":
			m.groupFocus = 1
		case "left":
			if m.groupInputCursor > 0 {
				m.groupInputCursor--
			}
		case "right":
			if m.groupInputCursor < utf8.RuneCountInString(m.groupInput) {
				m.groupInputCursor++
			}
		case "backspace":
			runes := []rune(m.groupInput)
			if m.groupInputCursor > 0 {
				m.groupInput = string(runes[:m.groupInputCursor-1]) + string(runes[m.groupInputCursor:])
				m.groupInputCursor--
			}
		default:
			if len(key) == 1 && key[0] >= 32 && key[0] < 127 {
				runes := []rune(m.groupInput)
				m.groupInput = string(runes[:m.groupInputCursor]) + key + string(runes[m.groupInputCursor:])
				m.groupInputCursor++
			}
		}
		return m, nil
	}

	// On buttons (groupFocus 1 or 2)
	switch key {
	case "esc":
		m.overlay = overlayNone
	case "tab":
		m.groupFocus = (m.groupFocus + 1) % 3
	case "shift+tab":
		m.groupFocus = (m.groupFocus - 1 + 3) % 3
	case "up", "k":
		m.groupFocus = 0
	case "down", "j":
		// already at bottom
	case "left", "h":
		if m.groupFocus == 2 {
			m.groupFocus = 1
		}
	case "right", "l":
		if m.groupFocus == 1 {
			m.groupFocus = 2
		}
	case "enter":
		if m.groupFocus == 1 {
			if m.groupInput != "" {
				oldName := m.groups[m.groupEditIdx].name
				m.groups[m.groupEditIdx].name = m.groupInput
				for i := range m.hosts {
					if m.hosts[i].group == oldName {
						m.hosts[i].group = m.groupInput
					}
				}
				m.formGroups = make([]string, len(m.groups))
				for i, g := range m.groups {
					m.formGroups[i] = g.name
				}
				m.items = buildList(m.groups, m.hosts)
			}
			m.overlay = overlayNone
		} else if m.groupFocus == 2 {
			m.overlay = overlayNone
		}
	}
	return m, nil
}

func (m model) updateDeleteGroup(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.overlay = overlayNone
	case "left", "right", "up", "down", "tab", "shift+tab", "h", "l", "j", "k":
		m.groupDeleteCursor = 1 - m.groupDeleteCursor
	case "enter":
		if m.groupDeleteCursor == 0 {
			if len(m.groups) > 1 {
				delName := m.groups[m.groupDeleteIdx].name
				targetName := ""
				for i, g := range m.groups {
					if i != m.groupDeleteIdx {
						targetName = g.name
						break
					}
				}
				for i := range m.hosts {
					if m.hosts[i].group == delName {
						m.hosts[i].group = targetName
					}
				}
				m.groups = append(m.groups[:m.groupDeleteIdx], m.groups[m.groupDeleteIdx+1:]...)
				m.formGroups = make([]string, len(m.groups))
				for i, g := range m.groups {
					m.formGroups[i] = g.name
				}
				m.items = buildList(m.groups, m.hosts)
				if m.cursor >= len(m.items) {
					m.cursor = len(m.items) - 1
				}
			}
		}
		m.overlay = overlayNone
	}
	return m, nil
}

func (m model) updateQuit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "n":
		m.overlay = overlayNone
	case "enter":
		switch m.quitCursor {
		case 0, 1: // unmount & quit, or leave mounted
			return m, tea.Quit
		case 2: // cancel
			m.overlay = overlayNone
		}
	case "y":
		return m, tea.Quit
	case "left", "h", "up", "k":
		if m.quitCursor > 0 {
			m.quitCursor--
		}
	case "right", "l", "down", "j":
		if m.quitCursor < 2 {
			m.quitCursor++
		}
	case "tab":
		m.quitCursor = (m.quitCursor + 1) % 3
	case "shift+tab":
		m.quitCursor = (m.quitCursor - 1 + 3) % 3
	}
	return m, nil
}

// setTermBgCmd returns a tea.Cmd that updates the terminal's default bg via OSC 11.
func setTermBgCmd(hex string) tea.Cmd {
	return func() tea.Msg {
		fmt.Fprintf(os.Stdout, "\x1b]11;%s\x1b\\", hexToOSC11(hex))
		return nil
	}
}

// applyThemeSetting checks if the changed setting is Theme and updates accordingly.
// Returns true if the theme was changed.
func (m *model) applyThemeSetting(s *settingsItem) bool {
	if s.label != "Theme" {
		return false
	}
	for i, t := range themes {
		if t.name == s.value {
			m.theme = themes[i]
			m.themeIdx = i
			return true
		}
	}
	return false
}

func (m *model) applyIconSetting(s *settingsItem) bool {
	if s.label != "Icon set" {
		return false
	}
	for i, p := range iconPresets {
		if p.name == s.value {
			m.icons = iconPresets[i]
			m.iconIdx = i
			return true
		}
	}
	return false
}

func (m model) filteredSettings() []int {
	if m.settingsFilter == "" {
		indices := make([]int, len(m.settings))
		for i := range m.settings {
			indices[i] = i
		}
		return indices
	}
	query := strings.ToLower(m.settingsFilter)
	var indices []int
	for i, s := range m.settings {
		if strings.Contains(strings.ToLower(s.label), query) ||
			strings.Contains(strings.ToLower(s.category), query) ||
			strings.Contains(strings.ToLower(s.value), query) {
			indices = append(indices, i)
		}
	}
	return indices
}

func (m model) updateSettings(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Handle search mode input
	if m.settingsSearch {
		switch key {
		case "esc":
			m.settingsSearch = false
			m.settingsFilter = ""
			return m, nil
		case "enter":
			m.settingsSearch = false
			// keep filter active, snap cursor to first visible
			filtered := m.filteredSettings()
			if len(filtered) > 0 {
				m.settingsCur = filtered[0]
			}
			return m, nil
		case "backspace":
			if len(m.settingsFilter) > 0 {
				m.settingsFilter = removeLastRune(m.settingsFilter)
				m.settingsCur = 0
			}
			return m, nil
		default:
			if len(key) == 1 && key[0] >= 32 && key[0] < 127 {
				m.settingsFilter += key
				m.settingsCur = 0
			}
			return m, nil
		}
	}

	filtered := m.filteredSettings()

	// find current position in filtered list
	curFiltered := 0
	for i, idx := range filtered {
		if idx == m.settingsCur {
			curFiltered = i
			break
		}
	}

	var themeChanged bool
	switch key {
	case "j", "down":
		if curFiltered < len(filtered)-1 {
			m.settingsCur = filtered[curFiltered+1]
		}
	case "k", "up":
		if curFiltered > 0 {
			m.settingsCur = filtered[curFiltered-1]
		}
	case "space", "enter":
		s := &m.settings[m.settingsCur]
		if s.kind == 0 { // toggle
			if s.value == "on" {
				s.value = "off"
			} else {
				s.value = "on"
			}
		} else if s.kind == 1 { // enum: cycle forward
			s.optIdx = (s.optIdx + 1) % len(s.options)
			s.value = s.options[s.optIdx]
			themeChanged = m.applyThemeSetting(s)
			m.applyIconSetting(s)
		}
	case "left", "h":
		s := &m.settings[m.settingsCur]
		if s.kind == 1 {
			s.optIdx = (s.optIdx - 1 + len(s.options)) % len(s.options)
			s.value = s.options[s.optIdx]
			themeChanged = m.applyThemeSetting(s)
			m.applyIconSetting(s)
		}
	case "right", "l":
		s := &m.settings[m.settingsCur]
		if s.kind == 1 {
			s.optIdx = (s.optIdx + 1) % len(s.options)
			s.value = s.options[s.optIdx]
			themeChanged = m.applyThemeSetting(s)
			m.applyIconSetting(s)
		}
	}
	if themeChanged {
		return m, setTermBgCmd(string(m.theme.base))
	}
	return m, nil
}

func (m model) updateTokens(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if m.tokensCur < len(m.tokens)-1 {
			m.tokensCur++
		}
	case "k", "up":
		if m.tokensCur > 0 {
			m.tokensCur--
		}
	case "a":
		// Mock: create a new token
		m.tokens = append(m.tokens, token{
			name:    fmt.Sprintf("token-%d", len(m.tokens)+1),
			scope:   "read",
			created: "just now",
			lastUse: "never",
		})
	case "d":
		// Mock: revoke selected token
		if len(m.tokens) > 0 && m.tokensCur < len(m.tokens) {
			m.tokens = append(m.tokens[:m.tokensCur], m.tokens[m.tokensCur+1:]...)
			if m.tokensCur >= len(m.tokens) && m.tokensCur > 0 {
				m.tokensCur--
			}
		}
	}
	return m, nil
}

// ── view ──────────────────────────────────────────────────────────────

func (m model) View() string {
	if m.w == 0 {
		return ""
	}

	var content string
	if m.overlay != overlayNone {
		switch m.overlay {
		case overlayHelp:
			content = m.helpView()
		case overlaySearch:
			content = m.searchView()
		case overlayAddHost:
			content = m.addHostView()
		case overlayQuit:
			content = m.quitView()
		case overlayLogin:
			content = m.loginView()
		case overlaySetup:
			content = m.setupView()
		case overlayDeleteHost:
			content = m.deleteHostView()
		case overlayCreateGroup:
			content = m.createGroupView()
		case overlayRenameGroup:
			content = m.renameGroupView()
		case overlayDeleteGroup:
			content = m.deleteGroupView()
		}
	} else {
		switch m.page {
		case pageSettings:
			content = m.settingsView()
		case pageTokens:
			content = m.tokensView()
		default:
			content = m.listView()
		}
	}

	// Wrap content in a full-screen box with theme background.
	// lipgloss.Place fills all whitespace (right padding + empty rows)
	// with explicit bg-colored characters, preventing the terminal's
	// native background from showing through.
	filled := lipgloss.Place(m.w, m.h, lipgloss.Left, lipgloss.Top, content,
		lipgloss.WithWhitespaceBackground(m.theme.base))

	// Re-inject base bg after every ANSI reset so the terminal's native
	// background never bleeds through between styled text spans.
	return m.applyBaseBg(filled)
}

// applyBaseBg replaces every \e[0m (full SGR reset) with \e[0m\e[48;2;R;G;Bm
// so our theme background stays active after each styled span. Combined with
// OSC 11 (which handles bubbletea's post-View erase codes), this ensures
// complete bg coverage.
func (m model) applyBaseBg(s string) string {
	r, g, b := hexToRGB(string(m.theme.base))
	bgCode := fmt.Sprintf("\x1b[48;2;%d;%d;%dm", r, g, b)
	return strings.ReplaceAll(s, "\x1b[0m", "\x1b[0m"+bgCode)
}

// hexToRGB parses "#RRGGBB" into integer components.
func hexToRGB(hex string) (int, int, int) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return 0, 0, 0
	}
	r, _ := strconv.ParseInt(hex[0:2], 16, 64)
	g, _ := strconv.ParseInt(hex[2:4], 16, 64)
	b, _ := strconv.ParseInt(hex[4:6], 16, 64)
	return int(r), int(g), int(b)
}

// ── layout helpers ────────────────────────────────────────────────────

// contentWidth returns the total usable inner width (padding each side)
func (m model) contentWidth() int {
	cw := m.w - 8
	if cw > 160 {
		cw = 160
	}
	if cw < 40 {
		cw = 40
	}
	return cw
}

// showSidebar returns true when the terminal is wide enough for the sidebar
func (m model) showSidebar() bool {
	return m.w >= 60
}

// pageContentWidth returns the content width for page views (minus sidebar + gap)
func (m model) pageContentWidth() int {
	cw := m.contentWidth()
	if m.showSidebar() {
		cw -= sidebarW + 2
	}
	if cw < 36 {
		cw = 36
	}
	return cw
}

// leftPad returns the left padding to center the total content block
func (m model) leftPad() int {
	cw := m.contentWidth()
	total := m.w - cw
	if total < 0 {
		return 0
	}
	return total / 2
}

// wrapFull fills every cell with the theme background, guaranteeing no terminal bg leaks
func (m model) wrapFull(content string) string {
	bg := lipgloss.NewStyle().Background(m.theme.base)
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		visualW := lipgloss.Width(line)
		if visualW < m.w {
			lines[i] = line + bg.Render(strings.Repeat(" ", m.w-visualW))
		}
	}

	emptyLine := bg.Render(strings.Repeat(" ", m.w))
	for len(lines) < m.h {
		lines = append(lines, emptyLine)
	}
	if len(lines) > m.h {
		lines = lines[:m.h]
	}

	return strings.Join(lines, "\n")
}

// padContent adds bg-colored left padding and a top blank line
func (m model) padContent(inner string, leftPadN int) string {
	bg := lipgloss.NewStyle().Background(m.theme.base)
	padStr := bg.Render(strings.Repeat(" ", leftPadN))
	lines := strings.Split(inner, "\n")
	for i, l := range lines {
		lines[i] = padStr + l
	}
	return "\n" + strings.Join(lines, "\n")
}

// renderHeader returns the standard header with optional subtitle
func (m model) renderHeader(subtitle string) string {
	title := lipgloss.NewStyle().Foreground(m.theme.text).Bold(true).Render("sshthing")
	if subtitle == "" {
		live := 0
		for _, h := range m.hosts {
			if h.status == 2 {
				live++
			}
		}
		subtitle = fmt.Sprintf("%d hosts  %d connected", len(m.hosts), live)
	}
	meta := lipgloss.NewStyle().Foreground(m.theme.subtext).Render(subtitle)
	return title + "    " + meta
}

// renderFooter returns a dimmed footer hint line
func (m model) renderFooter(text string) string {
	return lipgloss.NewStyle().Foreground(m.theme.overlay).Render(text)
}

// ── sidebar ───────────────────────────────────────────────────────────

func (m model) renderSidebarItem(icon string, active bool) string {
	dotStyle := lipgloss.NewStyle().Foreground(m.theme.surface0)
	iconStyle := lipgloss.NewStyle().Foreground(m.theme.overlay)
	dot := m.icons.inactiveMarker
	if active {
		dot = m.icons.activeMarker
		dotStyle = lipgloss.NewStyle().Foreground(m.theme.accent)
		iconStyle = lipgloss.NewStyle().Foreground(m.theme.text)
	}

	rendered := dotStyle.Render(dot) + iconStyle.Render(icon)
	visualW := lipgloss.Width(rendered)
	pad := sidebarW - visualW
	if pad < 0 {
		pad = 0
	}
	return strings.Repeat(" ", pad) + rendered
}

func (m model) renderSidebar(bodyH int) string {
	var items []string
	for _, pi := range m.pageIcons() {
		items = append(items, m.renderSidebarItem(pi.icon, m.page == pi.index))
	}

	topPad := bodyH/3 - len(items)/2
	if topPad < 0 {
		topPad = 0
	}

	var lines []string
	for i := 0; i < bodyH; i++ {
		idx := i - topPad
		if idx >= 0 && idx < len(items) {
			lines = append(lines, items[idx])
		} else {
			lines = append(lines, strings.Repeat(" ", sidebarW))
		}
	}
	return lipgloss.NewStyle().Width(sidebarW).Render(strings.Join(lines, "\n"))
}

// ── home page: host list ──────────────────────────────────────────────

func (m model) listView() string {
	cw := m.pageContentWidth()
	pad := m.leftPad()

	listW := cw * 30 / 100
	if listW < 24 {
		listW = 24
	}
	gapW := 4
	detailW := cw - listW - gapW
	if detailW < 20 {
		detailW = 20
	}
	bodyH := m.h - 6
	if bodyH < 4 {
		bodyH = 4
	}

	narrowMode := m.w < 70

	// ── list column ──
	var listLines []string
	for i, item := range m.items {
		sel := i == m.cursor
		if item.isNewGroup {
			nameStyle := lipgloss.NewStyle().Foreground(m.theme.overlay)
			prefix := "    "
			if sel {
				nameStyle = lipgloss.NewStyle().Foreground(m.theme.accent)
				prefix = lipgloss.NewStyle().Foreground(m.theme.accent).Render("  " + m.icons.focused + " ")
			}
			listLines = append(listLines, "")
			listLines = append(listLines, prefix+nameStyle.Render(m.icons.add+" new group"))
		} else if item.isGroup {
			arrow := m.icons.expanded
			if item.grp.collapsed {
				arrow = m.icons.collapsed
			}
			nameStyle := lipgloss.NewStyle().Foreground(m.theme.subtext)
			if sel {
				nameStyle = lipgloss.NewStyle().Foreground(m.theme.accent)
			}
			countStr := lipgloss.NewStyle().Foreground(m.theme.overlay).Render(fmt.Sprintf(" %d", item.hostCount))
			arrowR := lipgloss.NewStyle().Foreground(m.theme.overlay).Render(arrow)
			if sel {
				arrowR = lipgloss.NewStyle().Foreground(m.theme.accent).Render(arrow)
			}
			listLines = append(listLines, "")
			listLines = append(listLines, arrowR+" "+nameStyle.Render(item.grp.name)+countStr)
		} else {
			h := item.host
			prefix := "    "
			nameStyle := lipgloss.NewStyle().Foreground(m.theme.subtext)

			var dot string
			switch h.status {
			case 2:
				dot = lipgloss.NewStyle().Foreground(m.theme.green).Render(m.icons.connected)
			case 1:
				dot = lipgloss.NewStyle().Foreground(m.theme.yellow).Render(m.icons.idle)
			default:
				dot = lipgloss.NewStyle().Foreground(m.theme.surface0).Render(m.icons.offline)
			}

			if sel {
				nameStyle = lipgloss.NewStyle().Foreground(m.theme.accent).Bold(true)
				prefix = lipgloss.NewStyle().Foreground(m.theme.accent).Render("  " + m.icons.focused + " ")
			}

			maxLblW := listW - 10
			if narrowMode {
				maxLblW = m.w - 16
			}
			lbl := m.truncStr(h.label, maxLblW)

			listLines = append(listLines, prefix+dot+" "+nameStyle.Render(lbl))
		}
	}

	for len(listLines) < bodyH {
		listLines = append(listLines, "")
	}
	if len(listLines) > bodyH {
		listLines = listLines[:bodyH]
	}

	listBlock := lipgloss.NewStyle().Width(listW).Render(strings.Join(listLines, "\n"))

	// ── body ──
	var body string
	if narrowMode {
		body = listBlock
	} else {
		detailBlock := lipgloss.NewStyle().Width(detailW).Foreground(m.theme.subtext).
			Render(m.renderDetail(detailW, bodyH))
		gapBlock := lipgloss.NewStyle().Width(gapW).Render(strings.Repeat("\n", bodyH))
		body = lipgloss.JoinHorizontal(lipgloss.Top, listBlock, gapBlock, detailBlock)
	}

	// ── sidebar ──
	if m.showSidebar() {
		sidebar := m.renderSidebar(bodyH)
		sideGap := lipgloss.NewStyle().Width(2).Render(strings.Repeat("\n", bodyH))
		body = lipgloss.JoinHorizontal(lipgloss.Top, body, sideGap, sidebar)
	}

	// ── footer ──
	footerHint := m.renderFooter("↑↓ nav  ⏎ connect  / search  a add  e edit  d del  , settings  ? help  q quit")

	headerLine := m.renderHeader("")
	inner := headerLine + "\n\n" + body + "\n\n" + footerHint
	padded := m.padContent(inner, pad)

	return padded
}

// ── detail panel ──────────────────────────────────────────────────────

func (m model) renderDetail(w, h int) string {
	if m.cursor >= len(m.items) {
		return ""
	}
	item := m.items[m.cursor]

	if item.isNewGroup {
		hint := lipgloss.NewStyle().Foreground(m.theme.overlay).Render("press enter or a to create a new group")
		return lipgloss.NewStyle().Foreground(m.theme.text).Bold(true).Render("new group") + "\n\n" + hint
	}

	if item.isGroup {
		name := lipgloss.NewStyle().Foreground(m.theme.text).Bold(true).Render(item.grp.name)
		sub := lipgloss.NewStyle().Foreground(m.theme.subtext).Render(fmt.Sprintf("%d servers", item.hostCount))
		hint := lipgloss.NewStyle().Foreground(m.theme.overlay).Render("enter toggle  " + m.icons.offline + "  e rename  " + m.icons.offline + "  d delete")
		return name + "\n" + sub + "\n\n" + hint
	}

	ho := item.host
	kStyle := lipgloss.NewStyle().Foreground(m.theme.subtext)
	vStyle := lipgloss.NewStyle().Foreground(m.theme.text)
	dimStyle := lipgloss.NewStyle().Foreground(m.theme.subtext)

	var statusR string
	switch ho.status {
	case 2:
		statusR = lipgloss.NewStyle().Foreground(m.theme.green).Render("connected")
	case 1:
		statusR = lipgloss.NewStyle().Foreground(m.theme.yellow).Render("idle")
	default:
		statusR = lipgloss.NewStyle().Foreground(m.theme.subtext).Render("offline")
	}

	connStr := fmt.Sprintf("%s@%s", ho.user, ho.hostname)
	if ho.port != 22 {
		connStr += fmt.Sprintf(":%d", ho.port)
	}

	tagStr := ""
	for _, t := range ho.tags {
		tagStr += lipgloss.NewStyle().Foreground(m.theme.pink).Render(t) + "  "
	}
	if tagStr == "" {
		tagStr = lipgloss.NewStyle().Foreground(m.theme.overlay).Render("no tags")
	}

	title := lipgloss.NewStyle().Foreground(m.theme.text).Bold(true).Render(ho.label)

	lines := []string{
		title,
		statusR,
		"",
		lipgloss.NewStyle().Foreground(m.theme.accent).Render(connStr),
		"",
		kStyle.Render("auth        ") + vStyle.Render(ho.keyType),
		kStyle.Render("group       ") + dimStyle.Render(ho.group),
		kStyle.Render("last seen   ") + dimStyle.Render(ho.lastSSH),
		"",
		kStyle.Render("tags        ") + tagStr,
		"",
		"",
		lipgloss.NewStyle().Foreground(m.theme.overlay).Render("enter connect  ·  s sftp  ·  e edit  ·  d delete"),
	}

	return strings.Join(lines, "\n")
}

// ── help view (overlay) ───────────────────────────────────────────────

func (m model) quitView() string {
	bg := m.theme.mantle
	title := lipgloss.NewStyle().Foreground(m.theme.text).Background(bg).Bold(true).Render("quit sshthing?")

	btnStyle := func(label string, idx int) string {
		s := lipgloss.NewStyle().Foreground(m.theme.subtext).Background(bg).Padding(0, 2)
		if m.quitCursor == idx {
			s = s.Foreground(m.theme.base).Background(m.theme.accent).Bold(true)
		}
		return s.Render(label)
	}

	var contentParts []string
	contentParts = append(contentParts, title)

	if len(m.mockMounts) > 0 {
		contentParts = append(contentParts, "")
		mountLabel := lipgloss.NewStyle().Foreground(m.theme.yellow).Background(bg).Render(m.icons.warning + " active mounts:")
		contentParts = append(contentParts, mountLabel)
		for _, mt := range m.mockMounts {
			contentParts = append(contentParts, "  "+lipgloss.NewStyle().Foreground(m.theme.subtext).Background(bg).Render(mt))
		}
		contentParts = append(contentParts, "")
		buttons := btnStyle("unmount & quit", 0) + "  " + btnStyle("leave mounted", 1) + "  " + btnStyle("cancel", 2)
		contentParts = append(contentParts, buttons)
	} else {
		hint := lipgloss.NewStyle().Foreground(m.theme.overlay).Background(bg).Render("are you sure you want to exit?")
		contentParts = append(contentParts, "")
		contentParts = append(contentParts, hint)
		contentParts = append(contentParts, "")
		buttons := btnStyle("yes", 0) + "  " + btnStyle("no", 1) + "  " + btnStyle("cancel", 2)
		contentParts = append(contentParts, buttons)
	}

	footer := lipgloss.NewStyle().Foreground(m.theme.overlay).Background(bg).Render("←→ select · enter confirm · esc cancel")
	contentParts = append(contentParts, "")
	contentParts = append(contentParts, footer)

	content := strings.Join(contentParts, "\n")

	boxW := 50
	if len(m.mockMounts) == 0 {
		boxW = 40
	}

	box := lipgloss.NewStyle().
		Width(boxW).
		Background(bg).
		Padding(1, 2).
		Align(lipgloss.Center).
		Render(content)

	centered := lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center,
		box,
		lipgloss.WithWhitespaceBackground(m.theme.base))

	return centered
}

func (m model) helpView() string {
	title := lipgloss.NewStyle().Foreground(m.theme.text).Bold(true).Render("shortcuts")
	pairs := [][2]string{
		{"↑ ↓  j k", "navigate"},
		{"enter", "connect or toggle"},
		{"/", "search"},
		{"s", "sftp"},
		{"a", "add host"},
		{"e", "edit"},
		{"d", "delete"},
		{"shift+tab", "switch page"},
		{"?", "help"},
		{"q", "quit"},
	}

	kS := lipgloss.NewStyle().Foreground(m.theme.accent).Width(16).Align(lipgloss.Right)
	vS := lipgloss.NewStyle().Foreground(m.theme.subtext)
	var lines []string
	for _, p := range pairs {
		lines = append(lines, kS.Render(p[0])+"    "+vS.Render(p[1]))
	}

	content := title + "\n\n" + strings.Join(lines, "\n") + "\n\n" +
		lipgloss.NewStyle().Foreground(m.theme.overlay).Render("any key to close")

	centered := lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center,
		lipgloss.NewStyle().Padding(2, 4).Render(content),
		lipgloss.WithWhitespaceBackground(m.theme.base))

	return centered
}

// ── search view (overlay) ─────────────────────────────────────────────

func (m model) filterHosts() []host {
	if m.searchInput == "" {
		return m.hosts
	}
	q := strings.ToLower(m.searchInput)
	var results []host
	for _, h := range m.hosts {
		if strings.Contains(strings.ToLower(h.label), q) ||
			strings.Contains(strings.ToLower(h.hostname), q) ||
			strings.Contains(strings.ToLower(h.user), q) ||
			strings.Contains(strings.ToLower(h.group), q) {
			results = append(results, h)
		}
	}
	return results
}

func (m model) searchView() string {
	searchW := 56
	if searchW > m.w-8 {
		searchW = m.w - 8
	}
	if searchW < 30 {
		searchW = 30
	}

	// search input line
	bg := m.theme.mantle
	inputStyle := lipgloss.NewStyle().Foreground(m.theme.text).Background(bg)
	placeholder := lipgloss.NewStyle().Foreground(m.theme.overlay).Background(bg).Render("search hosts...")
	inputText := m.searchInput
	cursor := ""
	if m.tick%2 == 0 {
		cursor = lipgloss.NewStyle().Foreground(m.theme.accent).Background(bg).Render(m.icons.cursor)
	} else {
		cursor = lipgloss.NewStyle().Foreground(m.theme.overlay).Background(bg).Render(m.icons.cursor)
	}

	var inputLine string
	if inputText == "" {
		inputLine = "  " + placeholder + cursor
	} else {
		inputLine = "  " + inputStyle.Render(inputText) + cursor
	}

	sep := lipgloss.NewStyle().Foreground(m.theme.surface0).Background(bg).Render("  " + strings.Repeat(m.icons.rule, searchW-4))

	// results
	results := m.filterHosts()
	maxResults := 8
	if len(results) > maxResults {
		results = results[:maxResults]
	}

	var resultLines []string
	for i, h := range results {
		sel := i == m.searchCursor
		var dot string
		switch h.status {
		case 2:
			dot = lipgloss.NewStyle().Foreground(m.theme.green).Background(bg).Render(m.icons.connected)
		case 1:
			dot = lipgloss.NewStyle().Foreground(m.theme.yellow).Background(bg).Render(m.icons.idle)
		default:
			dot = lipgloss.NewStyle().Foreground(m.theme.surface0).Background(bg).Render(m.icons.offline)
		}

		nameStyle := lipgloss.NewStyle().Foreground(m.theme.subtext).Background(bg)
		groupHint := lipgloss.NewStyle().Foreground(m.theme.overlay).Background(bg).Render(" " + h.group)
		prefix := "    "
		if sel {
			nameStyle = lipgloss.NewStyle().Foreground(m.theme.accent).Background(bg).Bold(true)
			prefix = lipgloss.NewStyle().Foreground(m.theme.accent).Background(bg).Render("  " + m.icons.selected + " ")
			groupHint = lipgloss.NewStyle().Foreground(m.theme.subtext).Background(bg).Render(" " + h.group)
		}
		resultLines = append(resultLines, prefix+dot+" "+nameStyle.Render(h.label)+groupHint)
	}

	if len(results) == 0 && m.searchInput != "" {
		resultLines = append(resultLines, "  "+lipgloss.NewStyle().Foreground(m.theme.overlay).Background(bg).Render("no matches"))
	}

	// footer
	footerText := "esc close  ·  enter connect"
	if m.armedSFTP {
		footerText = "esc close  ·  enter sftp  ·  ctrl+s disarm"
	} else {
		footerText = "esc close  ·  enter connect  ·  ctrl+s arm sftp"
	}
	footer := lipgloss.NewStyle().Foreground(m.theme.overlay).Background(bg).Render("  " + footerText)

	// assemble overlay content
	var contentParts []string
	contentParts = append(contentParts, inputLine)
	contentParts = append(contentParts, sep)
	if len(resultLines) > 0 {
		contentParts = append(contentParts, "")
		contentParts = append(contentParts, resultLines...)
	}
	contentParts = append(contentParts, "")
	contentParts = append(contentParts, footer)

	overlayContent := strings.Join(contentParts, "\n")

	overlayBox := lipgloss.NewStyle().
		Width(searchW).
		Background(m.theme.mantle).
		Padding(1, 0).
		Render(overlayContent)

	centered := lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center,
		overlayBox,
		lipgloss.WithWhitespaceBackground(m.theme.base))

	return centered
}

// ── add host view (overlay) ───────────────────────────────────────────

func (m model) addHostView() string {
	cw := m.contentWidth()
	pad := m.leftPad()
	compact := m.h < 30

	titleText := "add new host"
	if m.formEditIdx >= 0 {
		titleText = m.icons.edit + " edit host"
	}
	headerLine := m.renderHeader(titleText)

	ruleW := cw
	if ruleW > 40 {
		ruleW = 40
	}
	rule := lipgloss.NewStyle().Foreground(m.theme.surface0).Render(strings.Repeat(m.icons.rule, ruleW))

	formW := cw
	if formW > 70 {
		formW = 70
	}

	blink := m.tick%2 == 0

	spacer := func() string {
		if compact {
			return ""
		}
		return "\n"
	}

	var lines []string
	lines = append(lines, headerLine)
	lines = append(lines, rule)

	// label
	lines = append(lines, spacer()+m.renderFormLabel("label", m.formFocus == ffLabel))
	lines = append(lines, m.renderInput(m.formFields[ffLabel], m.formFocus == ffLabel, formW-4, blink))

	// group selector
	lines = append(lines, spacer()+m.renderFormLabel("group", m.formFocus == ffGroup))
	gName := m.formGroups[m.formGroupIdx]
	gCount := fmt.Sprintf("[%d/%d]", m.formGroupIdx+1, len(m.formGroups))
	if m.formFocus == ffGroup {
		lines = append(lines, "  "+
			lipgloss.NewStyle().Foreground(m.theme.accent).Render(m.icons.leftArrow)+
			" "+lipgloss.NewStyle().Foreground(m.theme.text).Render(gName)+
			" "+lipgloss.NewStyle().Foreground(m.theme.accent).Render(m.icons.rightArrow)+
			"  "+lipgloss.NewStyle().Foreground(m.theme.overlay).Render(gCount))
	} else {
		lines = append(lines, "  "+
			lipgloss.NewStyle().Foreground(m.theme.overlay).Render(m.icons.leftArrow)+
			" "+lipgloss.NewStyle().Foreground(m.theme.subtext).Render(gName)+
			" "+lipgloss.NewStyle().Foreground(m.theme.overlay).Render(m.icons.rightArrow)+
			"  "+lipgloss.NewStyle().Foreground(m.theme.overlay).Render(gCount))
	}

	// tags
	lines = append(lines, spacer()+m.renderFormLabel("tags", m.formFocus == ffTags))
	lines = append(lines, m.renderInput(m.formFields[ffTags], m.formFocus == ffTags, formW-4, blink))
	if !compact {
		lines = append(lines, lipgloss.NewStyle().Foreground(m.theme.overlay).Render("  comma-separated"))
	}

	// hostname + port side by side
	hostW := formW - 20
	if hostW < 20 {
		hostW = 20
	}
	portW := 12

	hostLabel := m.renderFormLabel("hostname", m.formFocus == ffHostname)
	portLabel := m.renderFormLabel("port", m.formFocus == ffPort)

	if m.w >= 60 {
		if !compact {
			lines = append(lines, "")
		}
		labelRow := lipgloss.NewStyle().Width(hostW).Render(hostLabel) +
			lipgloss.NewStyle().Width(portW).Render(portLabel)
		lines = append(lines, labelRow)

		hostInput := m.renderInput(m.formFields[ffHostname], m.formFocus == ffHostname, hostW-4, blink)
		portInput := m.renderInput(m.formFields[ffPort], m.formFocus == ffPort, portW-4, blink)
		inputRow := lipgloss.NewStyle().Width(hostW).Render(hostInput) +
			lipgloss.NewStyle().Width(portW).Render(portInput)
		lines = append(lines, inputRow)
	} else {
		lines = append(lines, spacer()+hostLabel)
		lines = append(lines, m.renderInput(m.formFields[ffHostname], m.formFocus == ffHostname, formW-4, blink))
		lines = append(lines, spacer()+portLabel)
		lines = append(lines, m.renderInput(m.formFields[ffPort], m.formFocus == ffPort, formW-4, blink))
	}

	// username
	lines = append(lines, spacer()+m.renderFormLabel("username", m.formFocus == ffUsername))
	lines = append(lines, m.renderInput(m.formFields[ffUsername], m.formFocus == ffUsername, formW-4, blink))

	// auth method selector
	lines = append(lines, spacer()+m.renderFormLabel("authentication", m.formFocus == ffAuthMeth))
	aName := m.formAuthOpts[m.formAuth]
	if m.formFocus == ffAuthMeth {
		lines = append(lines, "  "+
			lipgloss.NewStyle().Foreground(m.theme.accent).Render(m.icons.leftArrow)+
			" "+lipgloss.NewStyle().Foreground(m.theme.text).Render(aName)+
			" "+lipgloss.NewStyle().Foreground(m.theme.accent).Render(m.icons.rightArrow))
	} else {
		lines = append(lines, "  "+
			lipgloss.NewStyle().Foreground(m.theme.overlay).Render(m.icons.leftArrow)+
			" "+lipgloss.NewStyle().Foreground(m.theme.subtext).Render(aName)+
			" "+lipgloss.NewStyle().Foreground(m.theme.overlay).Render(m.icons.rightArrow))
	}

	// auth detail based on method
	switch m.formAuth {
	case 0: // password
		lines = append(lines, spacer()+m.renderFormLabel("password", m.formFocus == ffAuthDet))
		lines = append(lines, m.renderInput(m.formFields[ffAuthDet], m.formFocus == ffAuthDet, formW-4, blink))
	case 1: // paste key
		lines = append(lines, spacer()+m.renderFormLabel("private key", m.formFocus == ffAuthDet))
		lines = append(lines, m.renderInput(m.formFields[ffAuthDet], m.formFocus == ffAuthDet, formW-4, blink))
		if !compact {
			lines = append(lines, lipgloss.NewStyle().Foreground(m.theme.overlay).Render("  paste your private key"))
		}
	case 2: // generate
		lines = append(lines, spacer()+m.renderFormLabel("key type", false))
		kt := m.formKeyTypes[m.formKeyIdx]
		lines = append(lines, "  "+lipgloss.NewStyle().Foreground(m.theme.text).Render(kt)+
			"  "+lipgloss.NewStyle().Foreground(m.theme.overlay).Render("(space to cycle)"))
	}

	// save button
	saveLabel := "save host"
	if m.formEditIdx >= 0 {
		saveLabel = "update host"
	}
	lines = append(lines, "")
	if m.formFocus == ffSave {
		lines = append(lines, "  "+lipgloss.NewStyle().Foreground(m.theme.accent).Bold(true).Render(m.icons.save+" "+saveLabel))
	} else {
		lines = append(lines, "  "+lipgloss.NewStyle().Foreground(m.theme.overlay).Render("  "+saveLabel))
	}

	// footer
	if !compact {
		lines = append(lines, "")
	}
	var footerText string
	if m.formEditing {
		footerText = "type to edit  ·  ↑↓ leave  ·  ←→ cursor  ·  esc done  ·  tab next"
	} else {
		footerText = "↑↓←→ navigate  ·  enter edit  ·  tab next  ·  esc cancel"
	}
	footer := m.renderFooter(footerText)
	lines = append(lines, footer)

	inner := strings.Join(lines, "\n")
	padded := m.padContent(inner, pad)

	return padded
}

func (m model) renderFormLabel(text string, focused bool) string {
	if focused {
		return "  " + lipgloss.NewStyle().Foreground(m.theme.accent).Render(text)
	}
	return "  " + lipgloss.NewStyle().Foreground(m.theme.overlay).Render(text)
}

func (m model) renderInput(f formField, focused bool, width int, blink bool) string {
	if width < 8 {
		width = 8
	}

	barColor := m.theme.surface0
	textColor := m.theme.subtext
	if focused {
		barColor = m.theme.accent
		textColor = m.theme.text
	}

	bar := lipgloss.NewStyle().Foreground(barColor).Render(m.icons.bar)

	val := f.value
	if f.masked && val != "" {
		val = strings.Repeat("•", utf8.RuneCountInString(val))
	}

	editing := focused && m.formEditing

	if editing {
		// Show cursor at position within the text
		runes := []rune(val)
		cur := f.cursor
		if cur > len(runes) {
			cur = len(runes)
		}
		before := string(runes[:cur])
		after := string(runes[cur:])
		beforeStyled := lipgloss.NewStyle().Foreground(textColor).Render(before)
		afterStyled := lipgloss.NewStyle().Foreground(textColor).Render(after)
		cursorChar := " "
		if cur < len(runes) {
			cursorChar = string(runes[cur])
			afterStyled = lipgloss.NewStyle().Foreground(textColor).Render(string(runes[cur+1:]))
		}
		var cursorStyled string
		if blink {
			cursorStyled = lipgloss.NewStyle().Foreground(m.theme.base).Background(m.theme.accent).Render(cursorChar)
		} else {
			cursorStyled = lipgloss.NewStyle().Foreground(m.theme.base).Background(m.theme.overlay).Render(cursorChar)
		}
		return "  " + bar + beforeStyled + cursorStyled + afterStyled
	}

	displayVal := lipgloss.NewStyle().Foreground(textColor).Render(val)
	if focused {
		// Focused but not editing: show underline hint, no blinking cursor
		underline := lipgloss.NewStyle().Foreground(m.theme.accent).Render("_")
		return "  " + bar + displayVal + underline
	}

	return "  " + bar + displayVal
}

// ── modal field helper ────────────────────────────────────────────────

func (m model) renderModalField(value string, cursor int, masked bool, focused bool, blink bool, bg lipgloss.Color) string {
	barColor := m.theme.surface0
	textColor := m.theme.subtext
	if focused {
		barColor = m.theme.accent
		textColor = m.theme.text
	}
	bar := lipgloss.NewStyle().Foreground(barColor).Background(bg).Render(m.icons.bar)

	val := value
	if masked && val != "" {
		val = strings.Repeat("•", utf8.RuneCountInString(val))
	}

	if focused {
		runes := []rune(val)
		cur := cursor
		if cur > len(runes) {
			cur = len(runes)
		}
		before := lipgloss.NewStyle().Foreground(textColor).Background(bg).Render(string(runes[:cur]))
		cursorChar := " "
		afterStart := cur
		if cur < len(runes) {
			cursorChar = string(runes[cur])
			afterStart = cur + 1
		}
		var cursorR string
		if blink {
			cursorR = lipgloss.NewStyle().Foreground(m.theme.base).Background(m.theme.accent).Render(cursorChar)
		} else {
			cursorR = lipgloss.NewStyle().Foreground(m.theme.base).Background(m.theme.overlay).Render(cursorChar)
		}
		after := lipgloss.NewStyle().Foreground(textColor).Background(bg).Render(string(runes[afterStart:]))
		return "  " + bar + before + cursorR + after
	}

	displayVal := lipgloss.NewStyle().Foreground(textColor).Background(bg).Render(val)
	return "  " + bar + displayVal
}

// ── login view (overlay) ─────────────────────────────────────────────

func (m model) loginView() string {
	bg := m.theme.mantle
	blink := m.tick%2 == 0

	title := lipgloss.NewStyle().Foreground(m.theme.accent).Background(bg).Bold(true).
		Render(m.icons.lock + " sshthing")

	label := lipgloss.NewStyle().Foreground(m.theme.overlay).Background(bg).Render("  password")
	input := m.renderModalField(m.loginFields[0].value, m.loginFields[0].cursor, true, true, blink, bg)

	var errLine string
	if m.loginError != "" {
		errLine = lipgloss.NewStyle().Foreground(m.theme.red).Background(bg).
			Render("  " + m.icons.errorIcon + " " + m.loginError)
	}

	footer := lipgloss.NewStyle().Foreground(m.theme.overlay).Background(bg).
		Render("enter unlock  " + m.icons.offline + "  esc quit")

	var contentParts []string
	contentParts = append(contentParts, title, "", label, input)
	if errLine != "" {
		contentParts = append(contentParts, "", errLine)
	}
	contentParts = append(contentParts, "", footer)

	content := strings.Join(contentParts, "\n")

	box := lipgloss.NewStyle().
		Width(40).
		Background(bg).
		Padding(1, 2).
		Align(lipgloss.Center).
		Render(content)

	return lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, box,
		lipgloss.WithWhitespaceBackground(m.theme.base))
}

// ── setup view (overlay) ─────────────────────────────────────────────

func (m model) setupView() string {
	bg := m.theme.mantle
	blink := m.tick%2 == 0

	title := lipgloss.NewStyle().Foreground(m.theme.accent).Background(bg).Bold(true).
		Render(m.icons.shield + " first-time setup")

	pwLabel := lipgloss.NewStyle().Foreground(m.theme.overlay).Background(bg).Render("  master password")
	pwInput := m.renderModalField(m.setupFields[0].value, m.setupFields[0].cursor, true, m.loginFocus == 0, blink, bg)

	cfLabel := lipgloss.NewStyle().Foreground(m.theme.overlay).Background(bg).Render("  confirm password")
	cfInput := m.renderModalField(m.setupFields[1].value, m.setupFields[1].cursor, true, m.loginFocus == 1, blink, bg)

	// submit button
	var submitLine string
	if m.loginFocus == 2 {
		submitLine = "  " + lipgloss.NewStyle().Foreground(m.theme.accent).Background(bg).Bold(true).
			Render(m.icons.save+" create vault")
	} else {
		submitLine = "  " + lipgloss.NewStyle().Foreground(m.theme.overlay).Background(bg).
			Render("  create vault")
	}

	var errLine string
	if m.loginError != "" {
		errLine = lipgloss.NewStyle().Foreground(m.theme.red).Background(bg).
			Render("  " + m.icons.errorIcon + " " + m.loginError)
	}

	footer := lipgloss.NewStyle().Foreground(m.theme.overlay).Background(bg).
		Render("tab next  " + m.icons.offline + "  enter submit  " + m.icons.offline + "  esc quit")

	var contentParts []string
	contentParts = append(contentParts, title, "", pwLabel, pwInput, "", cfLabel, cfInput)
	if errLine != "" {
		contentParts = append(contentParts, "", errLine)
	}
	contentParts = append(contentParts, "", submitLine, "", footer)

	content := strings.Join(contentParts, "\n")

	box := lipgloss.NewStyle().
		Width(50).
		Background(bg).
		Padding(1, 2).
		Align(lipgloss.Center).
		Render(content)

	return lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, box,
		lipgloss.WithWhitespaceBackground(m.theme.base))
}

// ── delete host view (overlay) ────────────────────────────────────────

func (m model) deleteHostView() string {
	bg := m.theme.mantle

	if m.deleteHostIdx >= len(m.hosts) {
		m.overlay = overlayNone
		return ""
	}
	ho := m.hosts[m.deleteHostIdx]

	title := lipgloss.NewStyle().Foreground(m.theme.red).Background(bg).Bold(true).
		Render(m.icons.warning + " delete host")

	hostLabel := lipgloss.NewStyle().Foreground(m.theme.text).Background(bg).Bold(true).Render(ho.label)
	connStr := lipgloss.NewStyle().Foreground(m.theme.subtext).Background(bg).
		Render(ho.user + "@" + ho.hostname)
	warn := lipgloss.NewStyle().Foreground(m.theme.red).Background(bg).
		Render("cannot be undone!")

	delStyle := lipgloss.NewStyle().Foreground(m.theme.subtext).Background(bg).Padding(0, 2)
	cancelStyle := lipgloss.NewStyle().Foreground(m.theme.subtext).Background(bg).Padding(0, 2)
	if m.deleteCursor == 0 {
		delStyle = delStyle.Foreground(m.theme.base).Background(m.theme.red).Bold(true)
	} else {
		cancelStyle = cancelStyle.Foreground(m.theme.base).Background(m.theme.accent).Bold(true)
	}
	buttons := delStyle.Render("delete") + "  " + cancelStyle.Render("cancel")

	footer := lipgloss.NewStyle().Foreground(m.theme.overlay).Background(bg).
		Render("←→ select · enter confirm · esc cancel")

	content := strings.Join([]string{title, "", hostLabel, connStr, "", warn, "", buttons, "", footer}, "\n")

	box := lipgloss.NewStyle().
		Width(42).
		Background(bg).
		Padding(1, 2).
		Align(lipgloss.Center).
		Render(content)

	return lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, box,
		lipgloss.WithWhitespaceBackground(m.theme.base))
}

// ── create group view (overlay) ───────────────────────────────────────

func (m model) createGroupView() string {
	bg := m.theme.mantle
	blink := m.tick%2 == 0

	title := lipgloss.NewStyle().Foreground(m.theme.accent).Background(bg).Bold(true).
		Render(m.icons.add + " new group")

	label := lipgloss.NewStyle().Foreground(m.theme.overlay).Background(bg).Render("  name")
	input := m.renderModalField(m.groupInput, m.groupInputCursor, false, m.groupFocus == 0, blink, bg)

	createStyle := lipgloss.NewStyle().Foreground(m.theme.subtext).Background(bg).Padding(0, 2)
	cancelStyle := lipgloss.NewStyle().Foreground(m.theme.subtext).Background(bg).Padding(0, 2)
	if m.groupFocus == 1 {
		createStyle = createStyle.Foreground(m.theme.base).Background(m.theme.accent).Bold(true)
	} else if m.groupFocus == 2 {
		cancelStyle = cancelStyle.Foreground(m.theme.base).Background(m.theme.accent).Bold(true)
	}
	buttons := createStyle.Render("create") + "  " + cancelStyle.Render("cancel")

	footer := lipgloss.NewStyle().Foreground(m.theme.overlay).Background(bg).
		Render("tab nav · enter submit · esc cancel")

	content := strings.Join([]string{title, "", label, input, "", buttons, "", footer}, "\n")

	box := lipgloss.NewStyle().
		Width(40).
		Background(bg).
		Padding(1, 2).
		Align(lipgloss.Center).
		Render(content)

	return lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, box,
		lipgloss.WithWhitespaceBackground(m.theme.base))
}

// ── rename group view (overlay) ───────────────────────────────────────

func (m model) renameGroupView() string {
	bg := m.theme.mantle
	blink := m.tick%2 == 0

	title := lipgloss.NewStyle().Foreground(m.theme.accent).Background(bg).Bold(true).
		Render(m.icons.edit + " rename group")

	label := lipgloss.NewStyle().Foreground(m.theme.overlay).Background(bg).Render("  name")
	input := m.renderModalField(m.groupInput, m.groupInputCursor, false, m.groupFocus == 0, blink, bg)

	renameStyle := lipgloss.NewStyle().Foreground(m.theme.subtext).Background(bg).Padding(0, 2)
	cancelStyle := lipgloss.NewStyle().Foreground(m.theme.subtext).Background(bg).Padding(0, 2)
	if m.groupFocus == 1 {
		renameStyle = renameStyle.Foreground(m.theme.base).Background(m.theme.accent).Bold(true)
	} else if m.groupFocus == 2 {
		cancelStyle = cancelStyle.Foreground(m.theme.base).Background(m.theme.accent).Bold(true)
	}
	buttons := renameStyle.Render("rename") + "  " + cancelStyle.Render("cancel")

	footer := lipgloss.NewStyle().Foreground(m.theme.overlay).Background(bg).
		Render("tab nav · enter submit · esc cancel")

	content := strings.Join([]string{title, "", label, input, "", buttons, "", footer}, "\n")

	box := lipgloss.NewStyle().
		Width(40).
		Background(bg).
		Padding(1, 2).
		Align(lipgloss.Center).
		Render(content)

	return lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, box,
		lipgloss.WithWhitespaceBackground(m.theme.base))
}

// ── delete group view (overlay) ───────────────────────────────────────

func (m model) deleteGroupView() string {
	bg := m.theme.mantle

	if m.groupDeleteIdx >= len(m.groups) {
		m.overlay = overlayNone
		return ""
	}
	g := m.groups[m.groupDeleteIdx]
	hostCount := 0
	for _, h := range m.hosts {
		if h.group == g.name {
			hostCount++
		}
	}

	title := lipgloss.NewStyle().Foreground(m.theme.red).Background(bg).Bold(true).
		Render(m.icons.warning + " delete group")

	info := lipgloss.NewStyle().Foreground(m.theme.text).Background(bg).Bold(true).
		Render(fmt.Sprintf("%q (%d hosts)", g.name, hostCount))

	targetName := ""
	for i, gr := range m.groups {
		if i != m.groupDeleteIdx {
			targetName = gr.name
			break
		}
	}
	hint := lipgloss.NewStyle().Foreground(m.theme.subtext).Background(bg).
		Render(fmt.Sprintf("hosts will move to %q", targetName))

	delStyle := lipgloss.NewStyle().Foreground(m.theme.subtext).Background(bg).Padding(0, 2)
	cancelStyle := lipgloss.NewStyle().Foreground(m.theme.subtext).Background(bg).Padding(0, 2)
	if m.groupDeleteCursor == 0 {
		delStyle = delStyle.Foreground(m.theme.base).Background(m.theme.red).Bold(true)
	} else {
		cancelStyle = cancelStyle.Foreground(m.theme.base).Background(m.theme.accent).Bold(true)
	}
	buttons := delStyle.Render("delete") + "  " + cancelStyle.Render("cancel")

	footer := lipgloss.NewStyle().Foreground(m.theme.overlay).Background(bg).
		Render("←→ select · enter confirm · esc cancel")

	content := strings.Join([]string{title, "", info, hint, "", buttons, "", footer}, "\n")

	box := lipgloss.NewStyle().
		Width(42).
		Background(bg).
		Padding(1, 2).
		Align(lipgloss.Center).
		Render(content)

	return lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, box,
		lipgloss.WithWhitespaceBackground(m.theme.base))
}

// ── settings view (page) ──────────────────────────────────────────────

func (m model) settingsView() string {
	cw := m.pageContentWidth()
	pad := m.leftPad()

	headerLine := m.renderHeader("settings")

	ruleW := cw
	if ruleW > 40 {
		ruleW = 40
	}
	rule := lipgloss.NewStyle().Foreground(m.theme.surface0).Render(strings.Repeat(m.icons.rule, ruleW))

	valueW := cw - 40
	if valueW < 16 {
		valueW = 16
	}
	if valueW > 28 {
		valueW = 28
	}
	labelW := cw - valueW - 8
	if labelW < 20 {
		labelW = 20
	}

	bodyH := m.h - 8

	var lines []string
	lines = append(lines, headerLine)

	// search/filter bar
	if m.settingsSearch || m.settingsFilter != "" {
		filterStyle := lipgloss.NewStyle().Foreground(m.theme.overlay)
		inputStyle := lipgloss.NewStyle().Foreground(m.theme.text)
		cursor := ""
		if m.settingsSearch && m.tick%2 == 0 {
			cursor = lipgloss.NewStyle().Foreground(m.theme.accent).Render(m.icons.cursor)
		} else if m.settingsSearch {
			cursor = lipgloss.NewStyle().Foreground(m.theme.overlay).Render(m.icons.cursor)
		}
		var filterLine string
		if m.settingsFilter == "" {
			filterLine = "  " + filterStyle.Render("filter...") + cursor
		} else {
			filterLine = "  " + inputStyle.Render(m.settingsFilter) + cursor
		}
		filterSep := lipgloss.NewStyle().Foreground(m.theme.surface0).Render("  " + strings.Repeat(m.icons.rule, ruleW-4))
		lines = append(lines, filterLine)
		lines = append(lines, filterSep)
	} else {
		lines = append(lines, rule)
	}

	lastCat := ""
	filtered := m.filteredSettings()

	// calculate scroll offset
	scrollOffset := 0
	if m.settingsCur > bodyH-6 {
		scrollOffset = m.settingsCur - bodyH + 6
	}

	for _, i := range filtered {
		s := m.settings[i]
		if s.category != lastCat {
			lines = append(lines, "")
			catStyle := lipgloss.NewStyle().Foreground(m.theme.subtext)
			lines = append(lines, "  "+catStyle.Render(s.category))
			lastCat = s.category
		}

		sel := i == m.settingsCur

		marker := "    "
		lblStyle := lipgloss.NewStyle().Foreground(m.theme.subtext)
		valStyle := lipgloss.NewStyle().Foreground(m.theme.text)

		if sel {
			marker = lipgloss.NewStyle().Foreground(m.theme.accent).Render("  " + m.icons.selected + " ")
			lblStyle = lipgloss.NewStyle().Foreground(m.theme.accent)
			valStyle = lipgloss.NewStyle().Foreground(m.theme.accent).Bold(true)
		}

		// value display
		valDisplay := s.value
		if s.kind == 0 { // toggle
			if s.value == "on" {
				valDisplay = lipgloss.NewStyle().Foreground(m.theme.green).Render("on")
			} else {
				valDisplay = valStyle.Render("off")
			}
		} else if s.kind == 1 && sel { // enum, selected
			valDisplay = lipgloss.NewStyle().Foreground(m.theme.accent).Render(m.icons.leftArrow) +
				" " + valStyle.Render(s.value) +
				" " + lipgloss.NewStyle().Foreground(m.theme.accent).Render(m.icons.rightArrow)
		} else {
			valDisplay = valStyle.Render(s.value)
		}

		// build the row with right-aligned value
		label := lblStyle.Render(m.truncStr(s.label, labelW))
		gap := strings.Repeat(" ", max(1, labelW-lipgloss.Width(s.label)))
		row := marker + label + gap + valDisplay
		lines = append(lines, row)
	}

	if len(filtered) == 0 && m.settingsFilter != "" {
		lines = append(lines, "")
		lines = append(lines, "  "+lipgloss.NewStyle().Foreground(m.theme.overlay).Render("no matching settings"))
	}

	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Foreground(m.theme.surface0).Render(strings.Repeat(m.icons.rule, ruleW)))
	lines = append(lines, "")
	footerHint := "↑↓ navigate  space toggle  ◄► cycle  / filter  q home"
	if m.settingsSearch {
		footerHint = "type to filter  enter confirm  esc clear"
	} else if m.settingsFilter != "" {
		footerHint = "↑↓ navigate  space toggle  ◄► cycle  / filter  esc clear  q home"
	}
	lines = append(lines, m.renderFooter(footerHint))

	// apply scroll
	inner := strings.Join(lines, "\n")
	allLines := strings.Split(inner, "\n")
	if scrollOffset > 0 && scrollOffset < len(allLines) {
		allLines = allLines[scrollOffset:]
	}
	if len(allLines) > m.h-2 {
		allLines = allLines[:m.h-2]
	}
	inner = strings.Join(allLines, "\n")

	// Attach sidebar
	if m.showSidebar() {
		sBodyH := m.h - 4
		if sBodyH < 4 {
			sBodyH = 4
		}
		sidebar := m.renderSidebar(sBodyH)
		innerBlock := lipgloss.NewStyle().Width(cw).Render(inner)
		sideGap := lipgloss.NewStyle().Width(2).Render(strings.Repeat("\n", sBodyH))
		inner = lipgloss.JoinHorizontal(lipgloss.Top, innerBlock, sideGap, sidebar)
	}

	padded := m.padContent(inner, pad)

	return padded
}

// ── tokens view (page) ────────────────────────────────────────────────

func (m model) tokensView() string {
	cw := m.pageContentWidth()
	pad := m.leftPad()

	headerLine := m.renderHeader("tokens")

	ruleW := cw
	if ruleW > 40 {
		ruleW = 40
	}

	bodyH := m.h - 6
	if bodyH < 4 {
		bodyH = 4
	}

	var lines []string
	lines = append(lines, headerLine)
	lines = append(lines, "")

	if len(m.tokens) == 0 {
		lines = append(lines, "  "+lipgloss.NewStyle().Foreground(m.theme.overlay).Render("no tokens"))
		lines = append(lines, "")
		lines = append(lines, "  "+lipgloss.NewStyle().Foreground(m.theme.overlay).Render("press a to create one"))
	} else {
		for i, tok := range m.tokens {
			sel := i == m.tokensCur

			nameStyle := lipgloss.NewStyle().Foreground(m.theme.text).Bold(true)
			prefix := "  "
			if sel {
				nameStyle = lipgloss.NewStyle().Foreground(m.theme.accent).Bold(true)
				prefix = lipgloss.NewStyle().Foreground(m.theme.accent).Render(m.icons.selected + " ")
			}

			lines = append(lines, prefix+nameStyle.Render(tok.name))

			// scope + created on same line
			scopeStr := lipgloss.NewStyle().Foreground(m.theme.subtext).Render("scope: " + tok.scope)
			createdStr := lipgloss.NewStyle().Foreground(m.theme.overlay).Render("created " + tok.created)
			scopeW := lipgloss.Width("scope: " + tok.scope)
			gapN := ruleW - scopeW - lipgloss.Width("created "+tok.created) - 4
			if gapN < 2 {
				gapN = 2
			}
			lines = append(lines, "  "+scopeStr+strings.Repeat(" ", gapN)+createdStr)

			// last use
			if tok.lastUse == "never" {
				lines = append(lines, "  "+lipgloss.NewStyle().Foreground(m.theme.overlay).Render("never used"))
			} else {
				lines = append(lines, "  "+lipgloss.NewStyle().Foreground(m.theme.overlay).Render("last used "+tok.lastUse))
			}

			// separator
			sepW := ruleW
			if sepW > 20 {
				sepW = 20
			}
			lines = append(lines, "  "+lipgloss.NewStyle().Foreground(m.theme.surface0).Render(strings.Repeat(m.icons.rule, sepW)))
			lines = append(lines, "")
		}
	}

	// footer
	lines = append(lines, m.renderFooter("↑↓ navigate  a create  d revoke  shift+tab pages  esc home  q quit"))

	inner := strings.Join(lines, "\n")

	// Attach sidebar
	if m.showSidebar() {
		sidebar := m.renderSidebar(bodyH)
		innerBlock := lipgloss.NewStyle().Width(cw).Render(inner)
		sideGap := lipgloss.NewStyle().Width(2).Render(strings.Repeat("\n", bodyH))
		inner = lipgloss.JoinHorizontal(lipgloss.Top, innerBlock, sideGap, sidebar)
	}

	padded := m.padContent(inner, pad)

	return padded
}

// ── utility functions ─────────────────────────────────────────────────

func splitTags(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var tags []string
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t != "" {
			tags = append(tags, t)
		}
	}
	return tags
}

func removeLastRune(s string) string {
	if s == "" {
		return ""
	}
	runes := []rune(s)
	return string(runes[:len(runes)-1])
}

func (m model) truncStr(s string, w int) string {
	if w <= 0 || len(s) <= w {
		return s
	}
	if w <= 1 {
		return m.icons.truncation
	}
	return s[:w-1] + m.icons.truncation
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func main() {
	lipgloss.SetColorProfile(termenv.TrueColor)
	m := initialModel()

	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// OSC 111: restore the terminal's original default background on exit.
	fmt.Fprint(os.Stdout, "\x1b]111\x1b\\")
}
