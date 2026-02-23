package app

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Vansh-Raja/SSHThing/internal/config"
	"github.com/Vansh-Raja/SSHThing/internal/db"
	"github.com/Vansh-Raja/SSHThing/internal/mount"
	"github.com/Vansh-Raja/SSHThing/internal/ssh"
	"github.com/Vansh-Raja/SSHThing/internal/sync"
	"github.com/Vansh-Raja/SSHThing/internal/ui"
	"github.com/Vansh-Raja/SSHThing/internal/update"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// Model represents the application state
type Model struct {
	store        *db.Store
	hosts        []Host
	groups       []string
	listItems    []ListItem
	selectedIdx  int
	viewMode     ViewMode
	width        int
	height       int
	armedSFTP    bool
	armedMount   bool
	armedUnmount bool
	collapsed    map[string]bool // group name -> collapsed
	quitPrevView ViewMode
	quitFocus    int // 0=Unmount&Quit, 1=Leave&Quit, 2=Cancel

	cfg         config.Config
	cfgOriginal config.Config

	// Login/Setup state
	loginInput   textinput.Model
	confirmInput textinput.Model // For password confirmation in setup mode
	isFirstRun   bool            // True if no database exists yet
	setupFocus   int             // 0=password, 1=confirm, 2=submit

	// Search state (Spotlight)
	searchInput textinput.Model
	isSearching bool

	// Delete state
	deleteConfirmFocus bool // true = Delete button focused, false = Cancel button focused

	styles *ui.Styles
	err    error
	errSeq int

	// Modal state
	modalForm *ModalForm

	// Group modal state (create/rename)
	groupInput       textinput.Model
	groupOldName     string
	groupFocus       int  // 0=input, 1=submit, 2=cancel
	groupDeleteFocus bool // true=Delete focused

	// Spotlight results
	spotlightItems []SpotlightItem

	mountManager *mount.Manager
	pendingMount *mount.PreparedMount

	// Settings state
	settingsPrevView ViewMode
	settingsIdx      int
	settingsEditing  bool
	settingsInput    textinput.Model

	// Sync state
	syncManager    *sync.Manager
	masterPassword string // Stored for sync re-encryption (cleared on quit)
	syncing        bool
	syncRunID      int
	syncAnimFrame  int
	syncProgress   float64

	// Update state
	currentVersion string
	updateChecking bool
	updateApplying bool
	updateRunID    int
	updateLast     *update.CheckResult
}

// ModalForm holds form state for add/edit modals
type ModalForm struct {
	labelInput    textinput.Model
	groupInput    textinput.Model
	tagsInput     textinput.Model
	groupOptions  []string
	groupSelected int
	hostnameInput textinput.Model
	usernameInput textinput.Model
	portInput     textinput.Model

	authMethod    int // 0=Pass, 1=Paste, 2=Gen
	passwordInput textinput.Model

	keyOption string // Legacy
	keyType   string // "ed25519", "rsa", "ecdsa"

	pastedKeyInput textarea.Model

	focusedField int
}

const (
	groupFocusInput = iota
	groupFocusSubmit
	groupFocusCancel
)

// NewModel creates a new application model
func NewModel() Model {
	return NewModelWithVersion("dev")
}

// NewModelWithVersion creates a new application model with explicit binary version.
func NewModelWithVersion(version string) Model {
	cfg, _ := config.Load()

	// Initialize search input
	searchInput := textinput.New()
	searchInput.Placeholder = "Search hosts..."
	searchInput.CharLimit = 50
	searchInput.Width = 40
	searchInput.Prompt = ""

	// Initialize login input
	loginInput := textinput.New()
	loginInput.Placeholder = "Master Password"
	loginInput.EchoMode = textinput.EchoPassword
	loginInput.EchoCharacter = '•'
	loginInput.Prompt = ""
	loginInput.Focus()

	// Initialize confirm input (for setup mode)
	confirmInput := textinput.New()
	confirmInput.Placeholder = "Confirm Password"
	confirmInput.EchoMode = textinput.EchoPassword
	confirmInput.EchoCharacter = '•'
	confirmInput.Prompt = ""

	// Check if database exists (first-run detection)
	isFirstRun := false
	viewMode := ViewModeLogin
	exists, _ := db.Exists()
	if !exists {
		isFirstRun = true
		viewMode = ViewModeSetup
	}

	settingsInput := textinput.New()
	settingsInput.Prompt = ""
	settingsInput.Placeholder = ""

	groupInput := textinput.New()
	groupInput.Prompt = ""
	groupInput.Placeholder = "Group name"

	return Model{
		cfg:              cfg,
		cfgOriginal:      cfg,
		hosts:            []Host{},
		groups:           []string{},
		listItems:        []ListItem{},
		selectedIdx:      0,
		viewMode:         viewMode,
		styles:           ui.NewStyles(),
		searchInput:      searchInput,
		loginInput:       loginInput,
		confirmInput:     confirmInput,
		isFirstRun:       isFirstRun,
		isSearching:      false,
		armedSFTP:        false,
		armedMount:       false,
		armedUnmount:     false,
		collapsed:        map[string]bool{},
		quitPrevView:     ViewModeList,
		quitFocus:        0,
		mountManager:     mount.NewManager(),
		settingsPrevView: ViewModeList,
		settingsIdx:      0,
		settingsEditing:  false,
		settingsInput:    settingsInput,
		groupInput:       groupInput,
		spotlightItems:   []SpotlightItem{},
		currentVersion:   strings.TrimSpace(version),
	}
}

// Init initializes the application
func (m Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, tea.HideCursor)
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	prevErr := ""
	if m.err != nil {
		prevErr = m.err.Error()
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		nextModel, nextCmd := m.handleKeyPress(msg)
		nm, ok := nextModel.(Model)
		if !ok {
			return nextModel, nextCmd
		}
		return nm, tea.Batch(nextCmd, nm.errorAutoClearCmd(prevErr))

	case syncAnimTickMsg:
		if !m.syncing || msg.runID != m.syncRunID {
			return m, nil
		}
		m.syncAnimFrame++
		target := 0.92
		m.syncProgress += (target - m.syncProgress) * 0.12
		if m.syncProgress < 0 {
			m.syncProgress = 0
		}
		if m.syncProgress > target {
			m.syncProgress = target
		}
		return m, syncAnimTickCmd(msg.runID)

	case syncFinishedMsg:
		if msg.runID != m.syncRunID {
			return m, nil
		}
		m.syncing = false
		m.syncProgress = 1
		if msg.result == nil {
			m.err = fmt.Errorf("⚠ sync failed: empty result")
			return m, m.errorAutoClearCmd(prevErr)
		}
		if msg.result.Success {
			m.loadHosts()
			m.loadGroups()
			m.rebuildListItems()
			m.err = fmt.Errorf("✓ Sync: ↓%d ↑%d", msg.result.HostsPulled, msg.result.HostsPushed)
		} else {
			m.err = fmt.Errorf("⚠ %s", msg.result.Message)
		}
		return m, m.errorAutoClearCmd(prevErr)

	case updateCheckedMsg:
		if msg.runID != m.updateRunID {
			return m, nil
		}
		m.updateChecking = false
		if msg.err != nil {
			m.err = fmt.Errorf("⚠ update check failed: %v", msg.err)
			return m, m.errorAutoClearCmd(prevErr)
		}
		if msg.result == nil {
			m.err = fmt.Errorf("⚠ update check failed: empty result")
			return m, m.errorAutoClearCmd(prevErr)
		}
		m.updateLast = msg.result
		m.cfg.Updates.LastCheckedAt = msg.result.CheckedAt.Format(time.RFC3339)
		m.cfg.Updates.LastSeenVersion = msg.result.LatestVersion
		m.cfg.Updates.LastSeenTag = msg.result.LatestTag
		m.cfg.Updates.ETagLatest = msg.result.ETag
		if msg.result.UpdateAvailable {
			m.err = fmt.Errorf("✓ Update available: %s", msg.result.LatestTag)
		} else {
			m.err = fmt.Errorf("✓ Already on latest stable release")
		}
		return m, m.errorAutoClearCmd(prevErr)

	case updateAppliedMsg:
		if msg.runID != m.updateRunID {
			return m, nil
		}
		m.updateApplying = false
		if msg.err != nil {
			m.err = fmt.Errorf("⚠ update failed: %v", msg.err)
			return m, m.errorAutoClearCmd(prevErr)
		}
		if msg.result == nil || !msg.result.Success {
			m.err = fmt.Errorf("⚠ update failed")
			return m, m.errorAutoClearCmd(prevErr)
		}
		if msg.handoffStarted {
			return m, tea.Quit
		}
		if msg.result.NeedsRelaunch && msg.result.RelaunchPath != "" {
			cmd := exec.Command(msg.result.RelaunchPath, msg.result.RelaunchArgs...)
			_ = cmd.Start()
			return m, tea.Quit
		}
		m.err = fmt.Errorf("✓ Update applied")
		return m, m.errorAutoClearCmd(prevErr)

	case updatePathFixedMsg:
		if msg.runID != m.updateRunID {
			return m, nil
		}
		m.updateApplying = false
		if msg.err != nil {
			m.err = fmt.Errorf("⚠ path fix failed: %v", msg.err)
			return m, m.errorAutoClearCmd(prevErr)
		}
		if m.updateLast != nil {
			m.updateLast.PathHealth = msg.pathHealth
		}
		m.err = fmt.Errorf("✓ PATH updated. Open a new terminal for changes.")
		return m, m.errorAutoClearCmd(prevErr)

	case clearErrMsg:
		if msg.seq == m.errSeq {
			m.err = nil
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case sshFinishedMsg:
		// Session ended, reload hosts to update last_connected
		m.loadHosts()
		m.viewMode = ViewModeList
		if msg.err != nil {
			m.err = fmt.Errorf("%s session ended: %v", msg.proto, msg.err)
		} else {
			m.err = fmt.Errorf("Disconnected from %s", msg.hostname)
		}
		return m, tea.Batch(tea.HideCursor, m.errorAutoClearCmd(prevErr))

	case mountFinishedMsg:
		m.viewMode = ViewModeList
		switch msg.action {
		case "mount":
			if m.pendingMount == nil {
				m.err = fmt.Errorf("mount failed: missing pending state")
				return m, tea.Batch(tea.HideCursor, m.errorAutoClearCmd(prevErr))
			}
			if msg.err != nil {
				m.mountManager.AbortMount(m.pendingMount)
				m.pendingMount = nil
				m.err = fmt.Errorf("mount failed: %v", msg.err)
				return m, tea.Batch(tea.HideCursor, m.errorAutoClearCmd(prevErr))
			}
			if err := m.mountManager.FinalizeMount(m.pendingMount); err != nil {
				m.pendingMount = nil
				m.err = err
				return m, tea.Batch(tea.HideCursor, m.errorAutoClearCmd(prevErr))
			}
			local := m.pendingMount.LocalPath
			m.pendingMount = nil
			m.err = fmt.Errorf("✓ Mounted at %s", local)
			if m.store != nil {
				// Persist that this host is mounted (best-effort). Next launch will reconcile with system mounts.
				_ = m.store.UpsertMountState(msg.hostID, local, "")
			}
			return m, tea.Batch(tea.HideCursor, m.errorAutoClearCmd(prevErr))
		case "unmount":
			if err := m.mountManager.FinalizeUnmount(msg.hostID, msg.err); err != nil {
				m.err = err
				return m, tea.Batch(tea.HideCursor, m.errorAutoClearCmd(prevErr))
			}
			if m.store != nil {
				_ = m.store.DeleteMountState(msg.hostID)
			}
			m.err = fmt.Errorf("✓ Unmounted")
			return m, tea.Batch(tea.HideCursor, m.errorAutoClearCmd(prevErr))
		default:
			m.err = fmt.Errorf("mount error: unknown action")
			return m, tea.Batch(tea.HideCursor, m.errorAutoClearCmd(prevErr))
		}

	case quitFinishedMsg:
		return m, tea.Quit
	}

	// Handle blink for inputs
	var cmd tea.Cmd
	if m.viewMode == ViewModeSetup {
		if m.setupFocus == 0 {
			m.loginInput, cmd = m.loginInput.Update(msg)
		} else if m.setupFocus == 1 {
			m.confirmInput, cmd = m.confirmInput.Update(msg)
		}
		return m, cmd
	} else if m.viewMode == ViewModeLogin {
		m.loginInput, cmd = m.loginInput.Update(msg)
		return m, cmd
	} else if m.viewMode == ViewModeAddHost || m.viewMode == ViewModeEditHost {
		cmd = m.handleInputUpdate(msg)
		return m, cmd
	} else if m.viewMode == ViewModeCreateGroup || m.viewMode == ViewModeRenameGroup {
		if m.groupFocus == groupFocusInput {
			m.groupInput, cmd = m.groupInput.Update(msg)
			return m, cmd
		}
		return m, nil
	} else if m.isSearching {
		m.searchInput, cmd = m.searchInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *Model) errorAutoClearCmd(prevErr string) tea.Cmd {
	currErr := ""
	if m.err != nil {
		currErr = m.err.Error()
	}
	if currErr == "" || currErr == prevErr {
		return nil
	}

	m.errSeq++
	seq := m.errSeq
	d := autoClearDuration(currErr)
	return tea.Tick(d, func(time.Time) tea.Msg {
		return clearErrMsg{seq: seq}
	})
}

func autoClearDuration(msg string) time.Duration {
	msg = strings.TrimSpace(msg)
	if strings.HasPrefix(msg, "✓") {
		return 5 * time.Second
	}
	return 10 * time.Second
}

// handleKeyPress processes keyboard input
func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global quit keybinding
	if msg.String() == "ctrl+c" {
		return m.requestQuit()
	}

	switch m.viewMode {
	case ViewModeSetup:
		return m.handleSetupKeys(msg)
	case ViewModeLogin:
		return m.handleLoginKeys(msg)
	case ViewModeList:
		return m.handleListKeys(msg)
	case ViewModeAddHost, ViewModeEditHost:
		return m.handleModalKeys(msg)
	case ViewModeDeleteHost:
		return m.handleDeleteKeys(msg)
	case ViewModeCreateGroup, ViewModeRenameGroup:
		return m.handleGroupInputKeys(msg)
	case ViewModeDeleteGroup:
		return m.handleDeleteGroupKeys(msg)
	case ViewModeSpotlight:
		return m.handleSpotlightKeys(msg)
	case ViewModeHelp:
		return m.handleHelpKeys(msg)
	case ViewModeQuitConfirm:
		return m.handleQuitConfirmKeys(msg)
	case ViewModeSettings:
		return m.handleSettingsKeys(msg)
	}

	return m, nil
}

func (m Model) requestQuit() (tea.Model, tea.Cmd) {
	if m.viewMode == ViewModeQuitConfirm {
		return m, nil
	}
	if m.mountManager != nil {
		if mounts := m.mountManager.ListActive(); len(mounts) > 0 {
			switch m.cfg.Mount.QuitBehavior {
			case config.MountQuitAlwaysUnmount:
				return m.quitAndUnmountAll()
			case config.MountQuitLeaveMounted:
				m.err = nil
				return m, tea.Quit
			default:
				// prompt
			}
			m.quitPrevView = m.viewMode
			m.quitFocus = 0
			m.viewMode = ViewModeQuitConfirm
			m.err = nil
			return m, nil
		}
	}
	return m, tea.Quit
}

func (m Model) quitAndUnmountAll() (tea.Model, tea.Cmd) {
	if m.mountManager == nil {
		return m, tea.Quit
	}
	// Run unmount synchronously on quit. This can take a moment, but avoids leaving mounts behind.
	unmountCmd := func() tea.Msg {
		m.mountManager.UnmountAll()
		return quitFinishedMsg{}
	}
	return m, tea.Sequence(tea.ShowCursor, unmountCmd, tea.Quit)
}

func (m Model) handleLoginKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		// Try to unlock
		password := m.loginInput.Value()
		store, err := db.Init(password)
		if err != nil {
			m.err = err
			m.loginInput.SetValue("")
			return m, nil
		}

		m.store = store
		m.masterPassword = password // Store for sync re-encryption
		m.loadHosts()
		m.restoreMountsFromDB()

		// Initialize sync manager
		if m.store != nil {
			syncMgr, err := sync.NewManager(&m.cfg, m.store, password)
			if err != nil {
				m.err = fmt.Errorf("sync init failed: %v", err)
			} else {
				m.syncManager = syncMgr
			}
		}

		m.err = nil
		m.viewMode = ViewModeList
		return m, nil

	case tea.KeyEsc:
		return m, tea.Quit
	}

	var cmd tea.Cmd
	// Clear old error once user starts typing again.
	if msg.Type != tea.KeyEnter {
		m.err = nil
	}
	m.loginInput, cmd = m.loginInput.Update(msg)
	return m, cmd
}

func (m Model) handleSetupKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyTab, tea.KeyDown:
		m.err = nil
		m.loginInput.Blur()
		m.confirmInput.Blur()
		m.setupFocus = (m.setupFocus + 1) % 3
		if m.setupFocus == 0 {
			m.loginInput.Focus()
		} else if m.setupFocus == 1 {
			m.confirmInput.Focus()
		}
		return m, nil

	case tea.KeyShiftTab, tea.KeyUp:
		m.err = nil
		m.loginInput.Blur()
		m.confirmInput.Blur()
		m.setupFocus = (m.setupFocus + 2) % 3 // -1 mod 3
		if m.setupFocus == 0 {
			m.loginInput.Focus()
		} else if m.setupFocus == 1 {
			m.confirmInput.Focus()
		}
		return m, nil

	case tea.KeyEnter:
		if m.setupFocus == 2 || msg.String() == "enter" {
			// Submit - validate and create database
			password := m.loginInput.Value()
			confirm := m.confirmInput.Value()

			// Validate password length
			if len(password) < 8 {
				m.err = fmt.Errorf("password must be at least 8 characters")
				return m, nil
			}

			// Validate passwords match
			if password != confirm {
				m.err = fmt.Errorf("passwords do not match")
				m.confirmInput.SetValue("")
				return m, nil
			}

			// Create the encrypted database
			store, err := db.Init(password)
			if err != nil {
				m.err = err
				return m, nil
			}

			m.store = store
			m.masterPassword = password // Store for sync re-encryption
			m.loadHosts()
			m.restoreMountsFromDB()

			// Initialize sync manager
			if m.store != nil {
				syncMgr, err := sync.NewManager(&m.cfg, m.store, password)
				if err != nil {
					m.err = fmt.Errorf("sync init failed: %v", err)
				} else {
					m.syncManager = syncMgr
				}
			}

			m.viewMode = ViewModeList
			m.err = nil
			return m, nil
		}

	case tea.KeyEsc:
		return m, tea.Quit
	}

	// Forward key to focused input
	var cmd tea.Cmd
	if msg.Type != tea.KeyEnter {
		m.err = nil
	}
	if m.setupFocus == 0 {
		m.loginInput, cmd = m.loginInput.Update(msg)
	} else if m.setupFocus == 1 {
		m.confirmInput, cmd = m.confirmInput.Update(msg)
	}
	return m, cmd
}

func (m Model) handleQuitConfirmKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.viewMode = m.quitPrevView
		m.quitFocus = 0
		return m, nil

	case "left", "h", "shift+tab":
		if m.quitFocus > 0 {
			m.quitFocus--
		}
		return m, nil

	case "right", "l", "tab":
		if m.quitFocus < 2 {
			m.quitFocus++
		}
		return m, nil

	case "enter":
		switch m.quitFocus {
		case 0: // unmount & quit
			m.err = fmt.Errorf("Unmounting mounts…")
			return m, func() tea.Msg {
				if m.mountManager != nil {
					m.mountManager.UnmountAll()
				}
				if m.store != nil {
					_ = m.store.DeleteAllMountStates()
				}
				return quitFinishedMsg{}
			}
		case 1: // leave mounted & quit
			if m.store != nil && m.mountManager != nil {
				for _, mt := range m.mountManager.ListActive() {
					_ = m.store.UpsertMountState(mt.HostID, mt.LocalPath, mt.RemotePath)
				}
			}
			return m, tea.Quit
		default: // cancel
			m.viewMode = m.quitPrevView
			m.quitFocus = 0
			return m, nil
		}
	}
	return m, nil
}

func (m *Model) loadHosts() {
	if m.store == nil {
		return
	}

	dbHosts, err := m.store.GetHosts()
	if err != nil {
		m.err = err
		return
	}

	m.hosts = make([]Host, len(dbHosts))
	for i, h := range dbHosts {
		hasKey := h.KeyData != ""
		if h.KeyType == "password" {
			// Password-auth hosts are still connectable even though no key material is stored.
			hasKey = true
		}
		label := strings.TrimSpace(h.Label)
		m.hosts[i] = Host{
			ID:            h.ID,
			Label:         label,
			GroupName:     strings.TrimSpace(h.GroupName),
			Tags:          append([]string(nil), h.Tags...),
			Hostname:      h.Hostname,
			Username:      h.Username,
			Port:          h.Port,
			HasKey:        hasKey,
			KeyType:       h.KeyType,
			CreatedAt:     h.CreatedAt,
			LastConnected: h.LastConnected,
		}
	}

	m.loadGroups()
	m.rebuildListItems()
}

func (m *Model) loadGroups() {
	if m.store == nil {
		return
	}
	groups, err := m.store.GetGroups()
	if err != nil {
		m.err = err
		return
	}
	m.groups = groups
}

func hostDisplayName(h Host) string {
	d := strings.TrimSpace(h.Label)
	if d == "" {
		d = strings.TrimSpace(h.Hostname)
	}
	return d
}

func hostVirtualGroupTag(h Host) string {
	if strings.TrimSpace(h.GroupName) == "" {
		return ""
	}
	return db.NormalizeTagToken(h.GroupName)
}

func hostSearchTags(h Host) []string {
	tags := db.NormalizeTags(h.Tags)
	if gt := hostVirtualGroupTag(h); gt != "" {
		tags = db.NormalizeTags(append(tags, gt))
	}
	return tags
}

func hostSearchCorpus(h Host) string {
	parts := []string{
		hostDisplayName(h),
		h.Hostname,
		h.Username,
		h.GroupName,
	}

	for _, tag := range hostSearchTags(h) {
		parts = append(parts, tag, "#"+tag)
	}

	return strings.Join(parts, " ")
}

func (m *Model) rebuildListItems() {
	counts := make(map[string]int)
	hostsByGroup := make(map[string][]Host)
	for _, h := range m.hosts {
		g := strings.TrimSpace(h.GroupName)
		hostsByGroup[g] = append(hostsByGroup[g], h)
		counts[g]++
	}

	groupSet := make(map[string]struct{})
	for _, g := range m.groups {
		name := strings.TrimSpace(g)
		if name == "" {
			continue
		}
		groupSet[name] = struct{}{}
	}
	for g := range hostsByGroup {
		if g == "" {
			continue
		}
		groupSet[g] = struct{}{}
	}

	groups := make([]string, 0, len(groupSet))
	for g := range groupSet {
		groups = append(groups, g)
	}
	sort.Slice(groups, func(i, j int) bool {
		return strings.ToLower(groups[i]) < strings.ToLower(groups[j])
	})

	for g := range hostsByGroup {
		sort.Slice(hostsByGroup[g], func(i, j int) bool {
			a := strings.ToLower(hostDisplayName(hostsByGroup[g][i]))
			b := strings.ToLower(hostDisplayName(hostsByGroup[g][j]))
			if a == b {
				return strings.ToLower(hostsByGroup[g][i].Hostname) < strings.ToLower(hostsByGroup[g][j].Hostname)
			}
			return a < b
		})
	}

	items := make([]ListItem, 0, len(m.hosts)+len(groups)+2)
	for _, g := range groups {
		items = append(items, ListItem{Kind: ListItemGroup, GroupName: g, Count: counts[g]})
		if !m.collapsed[g] {
			for _, h := range hostsByGroup[g] {
				items = append(items, ListItem{Kind: ListItemHost, GroupName: g, Host: h})
			}
		}
	}

	if len(hostsByGroup[""]) > 0 {
		items = append(items, ListItem{Kind: ListItemGroup, GroupName: "Ungrouped", Count: counts[""]})
		if !m.collapsed["Ungrouped"] {
			for _, h := range hostsByGroup[""] {
				items = append(items, ListItem{Kind: ListItemHost, GroupName: "", Host: h})
			}
		}
	}

	items = append(items, ListItem{Kind: ListItemNewGroup})
	m.listItems = items
	if len(m.listItems) == 0 {
		m.selectedIdx = 0
	} else if m.selectedIdx >= len(m.listItems) {
		m.selectedIdx = len(m.listItems) - 1
	}
}

func (m *Model) selectedListItem() (ListItem, bool) {
	if m.selectedIdx < 0 || m.selectedIdx >= len(m.listItems) {
		return ListItem{}, false
	}
	return m.listItems[m.selectedIdx], true
}

func (m *Model) selectedHost() (Host, bool) {
	item, ok := m.selectedListItem()
	if !ok || item.Kind != ListItemHost {
		return Host{}, false
	}
	return item.Host, true
}

func (m *Model) selectedGroup() (string, bool) {
	item, ok := m.selectedListItem()
	if !ok || item.Kind != ListItemGroup {
		return "", false
	}
	if item.GroupName == "Ungrouped" {
		return "", true
	}
	return item.GroupName, true
}

func (m *Model) hostCountForGroup(groupName string) int {
	count := 0
	norm := strings.TrimSpace(groupName)
	if strings.EqualFold(norm, "Ungrouped") {
		norm = ""
	}
	for _, h := range m.hosts {
		hg := strings.TrimSpace(h.GroupName)
		if norm == "" {
			if hg == "" {
				count++
			}
			continue
		}
		if strings.EqualFold(hg, norm) {
			count++
		}
	}
	return count
}

func (m *Model) restoreMountsFromDB() {
	if m.store == nil || m.mountManager == nil {
		return
	}
	states, err := m.store.GetMountStates()
	if err != nil {
		// Non-fatal; mounts are a beta/optional feature.
		return
	}

	byID := make(map[int]Host, len(m.hosts))
	for _, h := range m.hosts {
		byID[h.ID] = h
	}

	var toRestore []mount.Mount
	for _, st := range states {
		ok, err := mount.IsMounted(st.LocalPath)
		if err != nil {
			continue
		}
		if !ok {
			_ = m.store.DeleteMountState(st.HostID)
			continue
		}
		host, _ := byID[st.HostID]
		hostname := strings.TrimSpace(host.Hostname)
		if hostname == "" {
			hostname = fmt.Sprintf("host_%d", st.HostID)
		}
		toRestore = append(toRestore, mount.Mount{
			HostID:     st.HostID,
			Hostname:   hostname,
			LocalPath:  st.LocalPath,
			RemotePath: st.RemotePath,
		})
	}

	if len(toRestore) > 0 {
		m.mountManager.RestoreMounted(toRestore)
	}
}

// initSyncManager initializes the sync manager after store is ready
func (m *Model) initSyncManager() {
	if m.store == nil {
		return
	}

	syncMgr, err := sync.NewManager(&m.cfg, m.store, m.masterPassword)
	if err != nil {
		// Non-fatal; sync is optional
		m.err = fmt.Errorf("sync init failed: %v", err)
		return
	}

	m.syncManager = syncMgr
}

// handleSpotlightKeys handles input for the spotlight view
func (m Model) handleSpotlightKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.viewMode = ViewModeList
		m.isSearching = false
		m.searchInput.Blur()
		m.searchInput.Reset()
		m.spotlightItems = nil
		m.armedSFTP = false
		m.armedMount = false
		m.armedUnmount = false
		return m, nil

	case "S":
		item, ok := m.selectedSpotlightItem()
		if !ok || item.Kind != SpotlightItemHost {
			m.err = fmt.Errorf("select a host first")
			return m, nil
		}
		m.armedSFTP = !m.armedSFTP
		m.armedMount = false
		m.armedUnmount = false
		if m.armedSFTP {
			m.err = fmt.Errorf("SFTP armed — press Enter")
		} else {
			m.err = nil
		}
		return m, nil

	case "M":
		if !m.cfg.Mount.Enabled {
			m.err = fmt.Errorf("⚠ mounts are disabled in settings")
			return m, nil
		}
		item, ok := m.selectedSpotlightItem()
		if !ok || item.Kind != SpotlightItemHost {
			m.err = fmt.Errorf("select a host first")
			return m, nil
		}
		host := item.Host
		isMounted := false
		if m.mountManager != nil {
			isMounted, _ = m.mountManager.IsMounted(host.ID)
		}

		if m.armedMount || m.armedUnmount {
			m.armedMount = false
			m.armedUnmount = false
			m.err = nil
			return m, nil
		}

		m.armedSFTP = false
		if isMounted {
			m.armedUnmount = true
			m.err = fmt.Errorf("Unmount armed — press Enter")
		} else {
			m.armedMount = true
			m.err = fmt.Errorf("Mount (beta) armed — press Enter")
		}
		return m, nil

	case "enter":
		item, ok := m.selectedSpotlightItem()
		if !ok {
			return m, nil
		}
		if item.Kind == SpotlightItemGroup {
			m.viewMode = ViewModeList
			m.isSearching = false
			m.searchInput.Blur()
			m.searchInput.Reset()
			m.spotlightItems = nil
			m.selectGroupInList(item.GroupName)
			return m, nil
		}
		host := item.Host
		m.isSearching = false
		m.searchInput.Blur()
		m.searchInput.Reset()
		m.spotlightItems = nil
		if m.armedMount || m.armedUnmount {
			return m.handleMountEnter(host)
		}
		if m.armedSFTP {
			m.armedSFTP = false
			return m.connectToHostSFTP(host)
		}
		return m.connectToHost(host)

	case "up", "k":
		if msg.String() == "k" && !m.cfg.UI.VimMode {
			return m, nil
		}
		if m.selectedIdx > 0 {
			m.selectedIdx--
		}

	case "down", "j":
		if msg.String() == "j" && !m.cfg.UI.VimMode {
			return m, nil
		}
		if m.selectedIdx < len(m.spotlightItems)-1 {
			m.selectedIdx++
		}

	default:
		// Forward key to search input
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		m.spotlightItems = m.buildSpotlightItems(m.searchInput.Value())
		// Reset selection when typing
		m.selectedIdx = 0
		return m, cmd
	}

	return m, nil
}

// handleListKeys handles keyboard input in list view
func (m Model) handleListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	m.rebuildListItems()

	// Normal list navigation
	switch key {
	case "q":
		return m.requestQuit()

	case "S":
		m.armedSFTP = !m.armedSFTP
		m.armedMount = false
		m.armedUnmount = false
		if m.armedSFTP {
			m.err = fmt.Errorf("SFTP armed — press Enter")
		} else {
			m.err = nil
		}
		return m, nil

	case "M":
		// Arm mount/unmount (global chord: M then Enter)
		if !m.cfg.Mount.Enabled {
			m.err = fmt.Errorf("⚠ mounts are disabled in settings")
			return m, nil
		}
		host, ok := m.selectedHost()
		if !ok {
			m.err = fmt.Errorf("select a host first")
			return m, nil
		}
		isMounted := false
		if m.mountManager != nil {
			isMounted, _ = m.mountManager.IsMounted(host.ID)
		}

		// Toggle behavior: pressing M again cancels.
		if m.armedMount || m.armedUnmount {
			m.armedMount = false
			m.armedUnmount = false
			m.err = nil
			return m, nil
		}

		m.armedSFTP = false
		if isMounted {
			m.armedUnmount = true
			m.err = fmt.Errorf("Unmount armed — press Enter")
		} else {
			m.armedMount = true
			m.err = fmt.Errorf("Mount (beta) armed — press Enter")
		}
		return m, nil

	case "esc":
		if m.armedSFTP || m.armedMount || m.armedUnmount {
			m.armedSFTP = false
			m.armedMount = false
			m.armedUnmount = false
			m.err = nil
			return m, nil
		}

	case "up", "k":
		if key == "k" && !m.cfg.UI.VimMode {
			return m, nil
		}
		if m.selectedIdx > 0 {
			m.selectedIdx--
		}

	case "down", "j":
		if key == "j" && !m.cfg.UI.VimMode {
			return m, nil
		}
		if len(m.listItems) > 0 && m.selectedIdx < len(m.listItems)-1 {
			m.selectedIdx++
		}

	case "ctrl+u": // Page up
		m.selectedIdx = max(0, m.selectedIdx-10)

	case "ctrl+d": // Page down
		if len(m.listItems) > 0 {
			m.selectedIdx = min(len(m.listItems)-1, m.selectedIdx+10)
		}

	case "home", "g":
		if key == "g" && !m.cfg.UI.VimMode {
			return m, nil
		}
		m.selectedIdx = 0

	case "end", "G":
		if key == "G" && !m.cfg.UI.VimMode {
			return m, nil
		}
		if len(m.listItems) > 0 {
			m.selectedIdx = len(m.listItems) - 1
		}

	case "/", "ctrl+f":
		m.viewMode = ViewModeSpotlight
		m.isSearching = true
		m.searchInput.Focus()
		m.spotlightItems = m.buildSpotlightItems(m.searchInput.Value())
		m.selectedIdx = 0 // Reset selection for search

	case "a", "ctrl+n":
		if item, ok := m.selectedListItem(); ok && item.Kind == ListItemNewGroup {
			m.groupInput.SetValue("")
			m.groupFocus = groupFocusInput
			m.groupInput.Focus()
			m.viewMode = ViewModeCreateGroup
			return m, textinput.Blink
		}
		groupPrefill := ""
		if g, ok := m.selectedGroup(); ok {
			groupPrefill = g
		}
		m.viewMode = ViewModeAddHost
		m.modalForm = m.newModalForm("", groupPrefill, "", "", "", "22", "", "")

	case "e":
		if item, ok := m.selectedListItem(); ok && item.Kind == ListItemGroup {
			if item.GroupName == "Ungrouped" {
				m.err = fmt.Errorf("cannot rename Ungrouped")
				return m, nil
			}
			m.groupOldName = item.GroupName
			m.groupInput.SetValue(item.GroupName)
			m.groupInput.CursorEnd()
			m.groupFocus = groupFocusInput
			m.groupInput.Focus()
			m.viewMode = ViewModeRenameGroup
			return m, textinput.Blink
		}
		host, ok := m.selectedHost()
		if ok {
			m.viewMode = ViewModeEditHost
			var existingKey string
			if m.store != nil && host.HasKey && host.KeyType != "password" {
				key, err := m.store.GetHostSecret(host.ID)
				if err == nil {
					existingKey = key
				}
			}
			tagInput := strings.Join(host.Tags, ", ")
			m.modalForm = m.newModalForm(host.Label, host.GroupName, tagInput, host.Hostname, host.Username, fmt.Sprintf("%d", host.Port), host.KeyType, existingKey)
		}

	case "d", "delete":
		if item, ok := m.selectedListItem(); ok && item.Kind == ListItemGroup {
			if item.GroupName == "Ungrouped" {
				m.err = fmt.Errorf("cannot delete Ungrouped")
				return m, nil
			}
			m.groupOldName = item.GroupName
			m.groupDeleteFocus = false
			m.viewMode = ViewModeDeleteGroup
			return m, nil
		}
		if _, ok := m.selectedHost(); ok {
			m.viewMode = ViewModeDeleteHost
			m.deleteConfirmFocus = false // Default to Cancel
		}

	case "ctrl+g":
		m.groupInput.SetValue("")
		m.groupFocus = groupFocusInput
		m.groupInput.Focus()
		m.viewMode = ViewModeCreateGroup
		return m, textinput.Blink

	case "enter":
		item, ok := m.selectedListItem()
		if !ok {
			return m, nil
		}
		if item.Kind == ListItemGroup {
			m.collapsed[item.GroupName] = !m.collapsed[item.GroupName]
			m.rebuildListItems()
			return m, nil
		}
		if item.Kind == ListItemNewGroup {
			m.groupInput.SetValue("")
			m.groupFocus = groupFocusInput
			m.groupInput.Focus()
			m.viewMode = ViewModeCreateGroup
			return m, textinput.Blink
		}
		host := item.Host
		if m.armedMount || m.armedUnmount {
			return m.handleMountEnter(host)
		}
		if m.armedSFTP {
			m.armedSFTP = false
			return m.connectToHostSFTP(host)
		}
		return m.connectToHost(host)

	case "?":
		m.viewMode = ViewModeHelp

	case ",":
		m.settingsPrevView = m.viewMode
		m.cfgOriginal = m.cfg
		m.settingsIdx = 0
		m.settingsEditing = false
		m.settingsInput.SetValue("")
		m.settingsInput.Blur()
		m.viewMode = ViewModeSettings
		m.err = nil

	case "Y":
		if m.syncing {
			m.err = fmt.Errorf("ℹ sync already in progress")
			return m, nil
		}

		// Manual sync trigger - always show feedback
		if m.syncManager == nil {
			m.err = fmt.Errorf("⚠ sync manager is nil")
			return m, nil
		}
		if !m.syncManager.IsEnabled() {
			m.err = fmt.Errorf("⚠ sync is disabled — enable in settings")
			return m, nil
		}

		m.syncing = true
		m.syncRunID++
		m.syncAnimFrame = 0
		m.syncProgress = 0.02
		runID := m.syncRunID
		m.err = fmt.Errorf("ℹ Syncing...")
		return m, tea.Batch(runSyncCmd(runID, m.syncManager), syncAnimTickCmd(runID))
	}

	return m, nil
}

// Helper to create new modal form with initialized text inputs
func (m Model) newModalForm(label, groupName, tags, hostname, username, port, keyType, existingKey string) *ModalForm {
	authMethod := ui.AuthPassword
	switch keyType {
	case "password":
		authMethod = ui.AuthPassword
	case "pasted":
		authMethod = ui.AuthKeyPaste
	case "ed25519", "rsa", "ecdsa":
		authMethod = ui.AuthKeyGen
	default:
		authMethod = ui.AuthPassword
	}
	if strings.TrimSpace(existingKey) != "" {
		// Editing an existing host: show the current key for copy/edit.
		authMethod = ui.AuthKeyPaste
	}

	initialKeyType := "ed25519"
	switch keyType {
	case "ed25519", "rsa", "ecdsa":
		initialKeyType = keyType
	}

	groupOptions := m.modalGroupOptions(groupName)
	groupSelected := 0
	for i, opt := range groupOptions {
		if strings.EqualFold(strings.TrimSpace(opt), strings.TrimSpace(groupName)) {
			groupSelected = i
			break
		}
	}

	f := &ModalForm{
		labelInput:     textinput.New(),
		groupInput:     textinput.New(),
		tagsInput:      textinput.New(),
		groupOptions:   groupOptions,
		groupSelected:  groupSelected,
		hostnameInput:  textinput.New(),
		usernameInput:  textinput.New(),
		portInput:      textinput.New(),
		passwordInput:  textinput.New(), // Fixed: Initialize passwordInput
		pastedKeyInput: textarea.New(),
		authMethod:     authMethod,
		keyType:        initialKeyType,
		focusedField:   ui.FieldLabel,
	}

	f.labelInput.Placeholder = "Prod DB, Staging, Home NAS..."
	f.labelInput.SetValue(label)
	f.labelInput.Focus()
	f.labelInput.Prompt = ""

	f.groupInput.Placeholder = "Work, VMs, Personal..."
	f.groupInput.SetValue(groupOptions[groupSelected])
	f.groupInput.Prompt = ""

	f.tagsInput.Placeholder = "cpu, gpu, ec2"
	f.tagsInput.SetValue(tags)
	f.tagsInput.Prompt = ""

	f.hostnameInput.Placeholder = "example.com"
	f.hostnameInput.SetValue(hostname)
	f.hostnameInput.Prompt = ""

	f.usernameInput.Placeholder = "user"
	f.usernameInput.SetValue(username)
	f.usernameInput.Prompt = ""

	f.portInput.Placeholder = "22"
	f.portInput.SetValue(port)
	f.portInput.Prompt = ""

	f.passwordInput.Placeholder = "Password"
	f.passwordInput.EchoMode = textinput.EchoPassword
	f.passwordInput.EchoCharacter = '•'
	f.passwordInput.Prompt = ""

	f.pastedKeyInput.Placeholder = "Paste private key here..."
	f.pastedKeyInput.ShowLineNumbers = false
	f.pastedKeyInput.SetHeight(6)
	f.pastedKeyInput.SetValue(existingKey)

	return f
}

func (m Model) modalGroupOptions(current string) []string {
	options := []string{"Ungrouped"}
	seen := map[string]bool{"ungrouped": true}

	for _, g := range m.groups {
		name := strings.TrimSpace(g)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if seen[key] {
			continue
		}
		seen[key] = true
		options = append(options, name)
	}

	current = strings.TrimSpace(current)
	if current != "" {
		key := strings.ToLower(current)
		if !seen[key] {
			options = append(options, current)
		}
	}

	return options
}

func (m Model) modalSelectedGroupName() string {
	if m.modalForm == nil || len(m.modalForm.groupOptions) == 0 {
		return ""
	}
	idx := m.modalForm.groupSelected
	if idx < 0 || idx >= len(m.modalForm.groupOptions) {
		idx = 0
	}
	name := strings.TrimSpace(m.modalForm.groupOptions[idx])
	if strings.EqualFold(name, "Ungrouped") {
		return ""
	}
	return name
}

// handleModalKeys handles keyboard input in modal views
func (m Model) handleModalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.modalForm == nil {
		return m, nil
	}

	submitAndClose := func() (tea.Model, tea.Cmd) {
		validationErr := m.validateForm()
		if validationErr != nil {
			m.err = validationErr
			return m, nil
		}

		// Prepare data
		var portInt int
		fmt.Sscanf(m.modalForm.portInput.Value(), "%d", &portInt)

		var keyType, plainKey string
		switch m.modalForm.authMethod {
		case ui.AuthPassword:
			keyType = "password"
			plainKey = m.modalForm.passwordInput.Value()
		case ui.AuthKeyPaste:
			keyType = "pasted"
			plainKey = normalizePrivateKey(m.modalForm.pastedKeyInput.Value())
		case ui.AuthKeyGen:
			keyType = m.modalForm.keyType
			// Generate a new SSH key
			comment := fmt.Sprintf("%s@%s", m.modalForm.usernameInput.Value(), m.modalForm.hostnameInput.Value())
			privateKey, _, err := ssh.GenerateKey(ssh.KeyType(keyType), comment)
			if err != nil {
				m.err = fmt.Errorf("failed to generate key: %v", err)
				return m, nil
			}
			plainKey = normalizePrivateKey(privateKey)
		}

		// Save to DB
		groupName := m.modalSelectedGroupName()
		tags := db.ParseTagInput(m.modalForm.tagsInput.Value())
		if groupName != "" {
			if err := m.store.UpsertGroup(groupName); err != nil {
				m.err = err
				return m, nil
			}
		}
		if m.viewMode == ViewModeAddHost {
			host := &db.HostModel{
				Label:     strings.TrimSpace(m.modalForm.labelInput.Value()),
				GroupName: groupName,
				Tags:      tags,
				Hostname:  m.modalForm.hostnameInput.Value(),
				Username:  m.modalForm.usernameInput.Value(),
				Port:      portInt,
				KeyType:   keyType,
			}
			if err := m.store.CreateHost(host, plainKey); err != nil {
				m.err = err
				return m, nil
			}
			m.err = fmt.Errorf("✓ Host '%s' added", host.Hostname)
		} else {
			// Edit - update host with key data if provided, otherwise just metadata
			selectedHost, ok := m.selectedHost()
			if ok {
				originalID := selectedHost.ID
				keepExistingSecret := m.modalForm.authMethod == ui.AuthPassword &&
					selectedHost.KeyType == "password" &&
					plainKey == ""
				host := &db.HostModel{
					ID:        originalID,
					Label:     strings.TrimSpace(m.modalForm.labelInput.Value()),
					GroupName: groupName,
					Tags:      tags,
					Hostname:  m.modalForm.hostnameInput.Value(),
					Username:  m.modalForm.usernameInput.Value(),
					Port:      portInt,
					KeyType:   keyType,
				}
				// Update with key data when auth mode changed or secret provided.
				if keepExistingSecret {
					if err := m.store.UpdateHost(host); err != nil {
						m.err = err
						return m, nil
					}
				} else if m.modalForm.authMethod == ui.AuthPassword || plainKey != "" {
					if err := m.store.UpdateHostWithKey(host, plainKey); err != nil {
						m.err = err
						return m, nil
					}
				} else {
					if err := m.store.UpdateHost(host); err != nil {
						m.err = err
						return m, nil
					}
				}
				m.err = fmt.Errorf("✓ Host '%s' updated", host.Hostname)
			}
		}

		m.loadHosts() // Refresh list
		m.loadGroups()
		m.rebuildListItems()
		m.viewMode = ViewModeList
		m.modalForm = nil
		return m, nil
	}

	// Helper to blur all inputs
	blurAll := func() {
		m.modalForm.labelInput.Blur()
		m.modalForm.groupInput.Blur()
		m.modalForm.tagsInput.Blur()
		m.modalForm.hostnameInput.Blur()
		m.modalForm.usernameInput.Blur()
		m.modalForm.portInput.Blur()
		m.modalForm.passwordInput.Blur()
		m.modalForm.pastedKeyInput.Blur()
	}

	// Helper to focus based on index
	focusField := func(idx int) {
		blurAll()
		switch idx {
		case ui.FieldLabel:
			m.modalForm.labelInput.Focus()
		case ui.FieldGroup:
			m.modalForm.groupInput.Focus()
		case ui.FieldTags:
			m.modalForm.tagsInput.Focus()
		case ui.FieldHostname:
			m.modalForm.hostnameInput.Focus()
		case ui.FieldUsername:
			m.modalForm.usernameInput.Focus()
		case ui.FieldPort:
			m.modalForm.portInput.Focus()
		case ui.FieldAuthDetails:
			if m.modalForm.authMethod == ui.AuthKeyPaste {
				m.modalForm.pastedKeyInput.Focus()
			} else if m.modalForm.authMethod == ui.AuthPassword {
				m.modalForm.passwordInput.Focus()
			}
		}
	}

	totalFields := 10
	var cmd tea.Cmd

	// Save from anywhere in the modal.
	if s := msg.String(); s == "shift+enter" || s == "shift+return" || s == "ctrl+s" {
		return submitAndClose()
	}

	// Navigation Helpers
	cycleAuth := func(dir int) {
		m.modalForm.authMethod = (m.modalForm.authMethod + dir + 3) % 3
	}

	cycleGroup := func(dir int) {
		if len(m.modalForm.groupOptions) == 0 {
			return
		}
		n := len(m.modalForm.groupOptions)
		m.modalForm.groupSelected = (m.modalForm.groupSelected + dir + n) % n
		m.modalForm.groupInput.SetValue(m.modalForm.groupOptions[m.modalForm.groupSelected])
	}

	cycleButtons := func() {
		if m.modalForm.focusedField == ui.FieldSubmit {
			m.modalForm.focusedField = ui.FieldCancel
		} else {
			m.modalForm.focusedField = ui.FieldSubmit
		}
	}

	switch msg.Type {
	case tea.KeyEsc:
		m.viewMode = ViewModeList
		m.modalForm = nil
		return m, nil

	case tea.KeyUp, tea.KeyShiftTab:
		m.modalForm.focusedField = (m.modalForm.focusedField - 1 + totalFields) % totalFields
		focusField(m.modalForm.focusedField)

	case tea.KeyDown, tea.KeyTab:
		m.modalForm.focusedField = (m.modalForm.focusedField + 1) % totalFields
		focusField(m.modalForm.focusedField)

	case tea.KeyLeft:
		if m.modalForm.focusedField == ui.FieldGroup {
			cycleGroup(-1)
		} else if m.modalForm.focusedField == ui.FieldAuthMethod {
			cycleAuth(-1)
		} else if m.modalForm.focusedField == ui.FieldSubmit || m.modalForm.focusedField == ui.FieldCancel {
			cycleButtons()
		} else {
			cmd = m.handleInputUpdate(msg)
		}

	case tea.KeyRight:
		if m.modalForm.focusedField == ui.FieldGroup {
			cycleGroup(1)
		} else if m.modalForm.focusedField == ui.FieldAuthMethod {
			cycleAuth(1)
		} else if m.modalForm.focusedField == ui.FieldSubmit || m.modalForm.focusedField == ui.FieldCancel {
			cycleButtons()
		} else {
			cmd = m.handleInputUpdate(msg)
		}

	case tea.KeyEnter:
		if m.modalForm.focusedField == ui.FieldCancel {
			// Cancel - close the modal
			m.viewMode = ViewModeList
			m.modalForm = nil
			return m, nil
		}
		// Allow entering newlines while pasting/editing keys.
		if m.modalForm.focusedField == ui.FieldAuthDetails && m.modalForm.authMethod == ui.AuthKeyPaste {
			cmd = m.handleInputUpdate(msg)
			return m, cmd
		}
		// Save from any non-textarea field for terminal compatibility.
		// (Many terminals cannot reliably distinguish Shift+Enter.)
		return submitAndClose()

	default:
		// Handle runes and string-based keys
		str := msg.String()

		// Spacebar for Key Gen Type (keep specific inner toggle)
		if str == " " && m.modalForm.focusedField == ui.FieldAuthDetails && m.modalForm.authMethod == ui.AuthKeyGen {
			keyTypes := []string{"ed25519", "rsa", "ecdsa"}
			for i, kt := range keyTypes {
				if m.modalForm.keyType == kt {
					m.modalForm.keyType = keyTypes[(i+1)%len(keyTypes)]
					break
				}
			}
			return m, cmd
		}

		// Vim navigation h/l
		if str == "h" && m.modalForm.focusedField == ui.FieldGroup {
			cycleGroup(-1)
			return m, cmd
		} else if str == "l" && m.modalForm.focusedField == ui.FieldGroup {
			cycleGroup(1)
			return m, cmd
		} else if str == "h" && m.modalForm.focusedField == ui.FieldAuthMethod {
			cycleAuth(-1)
			return m, cmd
		} else if str == "l" && m.modalForm.focusedField == ui.FieldAuthMethod {
			cycleAuth(1)
			return m, cmd
		} else if str == "h" && (m.modalForm.focusedField == ui.FieldSubmit || m.modalForm.focusedField == ui.FieldCancel) {
			cycleButtons()
			return m, cmd
		} else if str == "l" && (m.modalForm.focusedField == ui.FieldSubmit || m.modalForm.focusedField == ui.FieldCancel) {
			cycleButtons()
			return m, cmd
		}

		// Shift+Tab fallback
		if str == "shift+tab" {
			m.modalForm.focusedField = (m.modalForm.focusedField - 1 + totalFields) % totalFields
			focusField(m.modalForm.focusedField)
			return m, nil
		}

		// Default: Pass to inputs
		cmd = m.handleInputUpdate(msg)
	}
	return m, cmd
}

// handleInputUpdate forwards messages to the focused textinput
func (m *Model) handleInputUpdate(msg tea.Msg) tea.Cmd {
	if m.modalForm == nil {
		return nil
	}

	var cmd tea.Cmd
	switch m.modalForm.focusedField {
	case ui.FieldLabel:
		m.modalForm.labelInput, cmd = m.modalForm.labelInput.Update(msg)
	case ui.FieldGroup:
		// Group selection is a spinner; use left/right (or h/l) to cycle.
		return nil
	case ui.FieldTags:
		m.modalForm.tagsInput, cmd = m.modalForm.tagsInput.Update(msg)
	case ui.FieldHostname:
		m.modalForm.hostnameInput, cmd = m.modalForm.hostnameInput.Update(msg)
	case ui.FieldUsername:
		m.modalForm.usernameInput, cmd = m.modalForm.usernameInput.Update(msg)
	case ui.FieldPort:
		m.modalForm.portInput, cmd = m.modalForm.portInput.Update(msg)
	case ui.FieldAuthDetails:
		if m.modalForm.authMethod == ui.AuthKeyPaste {
			m.modalForm.pastedKeyInput, cmd = m.modalForm.pastedKeyInput.Update(msg)
		} else if m.modalForm.authMethod == ui.AuthPassword {
			m.modalForm.passwordInput, cmd = m.modalForm.passwordInput.Update(msg)
		}
	}
	return cmd
}

// handleDeleteKeys handles keyboard input in delete confirmation view
func (m Model) handleDeleteKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	doDelete := func() Model {
		if host, ok := m.selectedHost(); ok {
			if err := m.store.DeleteHost(host.ID); err != nil {
				m.err = err
			} else {
				m.err = fmt.Errorf("✓ Host '%s' deleted", host.Hostname)
				m.loadHosts()
				m.loadGroups()
				m.rebuildListItems()
				if m.selectedIdx >= len(m.listItems) && len(m.listItems) > 0 {
					m.selectedIdx = len(m.listItems) - 1
				}
			}
		}
		m.viewMode = ViewModeList
		return m
	}
	switch key {
	case "y", "Y":
		// Shortcut to Delete
		m = doDelete()
		return m, nil

	case "n", "N", "esc":
		// Shortcut to Cancel
		m.viewMode = ViewModeList
		return m, nil

	case "left", "h", "right", "l", "tab", "shift+tab":
		m.deleteConfirmFocus = !m.deleteConfirmFocus
		return m, nil

	case "enter":
		if m.deleteConfirmFocus {
			// Delete confirmed
			m = doDelete()
		} else {
			// Cancel
			m.viewMode = ViewModeList
		}
		return m, nil
	}

	return m, nil
}

func (m Model) handleGroupInputKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	setFocus := func(focus int) {
		m.groupFocus = focus
		if focus == groupFocusInput {
			m.groupInput.Focus()
		} else {
			m.groupInput.Blur()
		}
	}

	submit := func() (tea.Model, tea.Cmd) {
		name := strings.TrimSpace(m.groupInput.Value())
		if name == "" {
			m.err = fmt.Errorf("group name cannot be empty")
			return m, nil
		}
		if strings.EqualFold(name, "Ungrouped") {
			m.err = fmt.Errorf("'Ungrouped' is reserved")
			return m, nil
		}
		if m.viewMode == ViewModeCreateGroup {
			if err := m.store.UpsertGroup(name); err != nil {
				m.err = err
				return m, nil
			}
			m.err = fmt.Errorf("✓ Group '%s' created", name)
		} else {
			if err := m.store.RenameGroup(m.groupOldName, name); err != nil {
				m.err = err
				return m, nil
			}
			m.err = fmt.Errorf("✓ Group '%s' renamed", name)
		}
		m.loadHosts()
		m.loadGroups()
		m.rebuildListItems()
		m.selectGroupInList(name)
		m.groupInput.SetValue("")
		m.groupInput.Blur()
		m.groupFocus = groupFocusInput
		m.viewMode = ViewModeList
		return m, nil
	}

	cancel := func() (tea.Model, tea.Cmd) {
		m.groupInput.SetValue("")
		m.groupInput.Blur()
		m.groupFocus = groupFocusInput
		m.viewMode = ViewModeList
		m.err = nil
		return m, nil
	}

	switch msg.String() {
	case "esc":
		return cancel()
	case "tab":
		setFocus((m.groupFocus + 1) % 3)
		return m, nil
	case "shift+tab":
		setFocus((m.groupFocus + 2) % 3)
		return m, nil
	case "left", "h":
		if m.groupFocus == groupFocusSubmit {
			setFocus(groupFocusCancel)
		} else if m.groupFocus == groupFocusCancel {
			setFocus(groupFocusSubmit)
		}
		return m, nil
	case "right", "l":
		if m.groupFocus == groupFocusSubmit {
			setFocus(groupFocusCancel)
		} else if m.groupFocus == groupFocusCancel {
			setFocus(groupFocusSubmit)
		}
		return m, nil
	case "enter":
		if m.groupFocus == groupFocusCancel {
			return cancel()
		}
		if m.groupFocus == groupFocusSubmit {
			return submit()
		}
		// Input focus: Enter submits as convenience.
		return submit()
	}

	if m.groupFocus == groupFocusInput {
		var cmd tea.Cmd
		m.groupInput, cmd = m.groupInput.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) handleDeleteGroupKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "n", "N":
		m.viewMode = ViewModeList
		m.groupDeleteFocus = false
		return m, nil
	case "left", "right", "h", "l", "tab", "shift+tab":
		m.groupDeleteFocus = !m.groupDeleteFocus
		return m, nil
	case "y", "Y":
		m.groupDeleteFocus = true
	}

	if msg.String() == "enter" && m.groupDeleteFocus {
		if err := m.store.DeleteGroup(m.groupOldName); err != nil {
			m.err = err
			return m, nil
		}
		m.err = fmt.Errorf("✓ Group '%s' deleted", m.groupOldName)
		m.loadHosts()
		m.loadGroups()
		m.rebuildListItems()
		m.viewMode = ViewModeList
		m.groupDeleteFocus = false
		return m, nil
	}

	if msg.String() == "enter" {
		m.viewMode = ViewModeList
		m.groupDeleteFocus = false
	}

	return m, nil
}

func (m Model) selectedSpotlightItem() (SpotlightItem, bool) {
	if m.selectedIdx < 0 || m.selectedIdx >= len(m.spotlightItems) {
		return SpotlightItem{}, false
	}
	return m.spotlightItems[m.selectedIdx], true
}

func (m *Model) selectGroupInList(groupName string) {
	m.rebuildListItems()
	for i, it := range m.listItems {
		if it.Kind != ListItemGroup {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(it.GroupName), strings.TrimSpace(groupName)) {
			m.selectedIdx = i
			return
		}
	}
}

func fuzzyScore(query, candidate string) (int, bool) {
	q := strings.ToLower(strings.TrimSpace(query))
	c := strings.ToLower(candidate)
	if q == "" {
		return 0, true
	}
	if strings.Contains(c, q) {
		return 100 + len(q)*4, true
	}
	qi := 0
	score := 0
	streak := 0
	lastMatch := -2
	for i := 0; i < len(c) && qi < len(q); i++ {
		if c[i] != q[qi] {
			continue
		}
		if i == 0 || c[i-1] == ' ' || c[i-1] == '-' || c[i-1] == '_' || c[i-1] == '.' || c[i-1] == '/' {
			score += 8
		}
		if i == lastMatch+1 {
			streak++
			score += 4 + streak
		} else {
			streak = 0
			score += 2
		}
		lastMatch = i
		qi++
	}
	if qi != len(q) {
		return 0, false
	}
	score += max(0, 20-(len(c)-len(q)))
	return score, true
}

func (m Model) buildSpotlightItems(query string) []SpotlightItem {
	query = strings.TrimSpace(query)
	if query == "" {
		out := make([]SpotlightItem, 0, min(8, len(m.hosts)))
		for _, h := range m.hosts {
			out = append(out, SpotlightItem{Kind: SpotlightItemHost, Host: h, GroupName: h.GroupName})
			if len(out) >= 8 {
				break
			}
		}
		return out
	}

	type scoredGroup struct {
		name  string
		score int
	}
	var groupScores []scoredGroup
	for _, it := range m.listItems {
		if it.Kind != ListItemGroup {
			continue
		}
		name := it.GroupName
		if name == "Ungrouped" {
			name = "Ungrouped"
		}
		if score, ok := fuzzyScore(query, name); ok {
			groupScores = append(groupScores, scoredGroup{name: it.GroupName, score: score})
		}
	}
	sort.Slice(groupScores, func(i, j int) bool { return groupScores[i].score > groupScores[j].score })

	seenHost := map[int]bool{}
	out := make([]SpotlightItem, 0, 12)
	for i, g := range groupScores {
		if i >= 3 {
			break
		}
		out = append(out, SpotlightItem{Kind: SpotlightItemGroup, GroupName: g.name, Score: g.score})
		groupHosts := make([]SpotlightItem, 0, 4)
		for _, h := range m.hosts {
			hg := strings.TrimSpace(h.GroupName)
			if g.name == "Ungrouped" {
				if hg != "" {
					continue
				}
			} else if !strings.EqualFold(hg, g.name) {
				continue
			}
			score, ok := fuzzyScore(query, hostSearchCorpus(h))
			if !ok {
				score = 1
			}
			groupHosts = append(groupHosts, SpotlightItem{Kind: SpotlightItemHost, Host: h, GroupName: h.GroupName, Score: score, Indent: 1})
		}
		sort.Slice(groupHosts, func(i, j int) bool { return groupHosts[i].Score > groupHosts[j].Score })
		for i := 0; i < len(groupHosts) && i < 3; i++ {
			if seenHost[groupHosts[i].Host.ID] {
				continue
			}
			seenHost[groupHosts[i].Host.ID] = true
			out = append(out, groupHosts[i])
		}
	}

	var direct []SpotlightItem
	for _, h := range m.hosts {
		if seenHost[h.ID] {
			continue
		}
		score, ok := fuzzyScore(query, hostSearchCorpus(h))
		if !ok {
			continue
		}
		direct = append(direct, SpotlightItem{Kind: SpotlightItemHost, Host: h, GroupName: h.GroupName, Score: score})
	}
	sort.Slice(direct, func(i, j int) bool { return direct[i].Score > direct[j].Score })
	for i := 0; i < len(direct) && len(out) < 16; i++ {
		out = append(out, direct[i])
	}
	return out
}

// handleHelpKeys handles keyboard input in help view
func (m Model) handleHelpKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "?", "esc", "q":
		m.viewMode = ViewModeList
	}
	return m, nil
}

func (m Model) handleSettingsKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	const maxIdx = 21

	key := msg.String()

	applyEditValue := func(val string) bool {
		val = strings.TrimSpace(val)
		switch m.settingsIdx {
		case 3:
			n, err := strconv.Atoi(val)
			if err != nil {
				m.err = fmt.Errorf("keepalive must be a number")
				return false
			}
			if n < 10 {
				n = 10
			}
			if n > 600 {
				n = 600
			}
			m.cfg.SSH.KeepAliveSeconds = n
			return true
		case 5:
			m.cfg.SSH.TermCustom = val
			return true
		case 9:
			m.cfg.Mount.DefaultRemotePath = val
			return true
		case 12:
			m.cfg.Sync.RepoURL = val
			return true
		case 13:
			m.cfg.Sync.SSHKeyPath = val
			return true
		case 14:
			if val == "" {
				val = "main"
			}
			m.cfg.Sync.Branch = val
			return true
		case 15:
			m.cfg.Sync.LocalPath = val
			return true
		}
		return true
	}

	startEdit := func(initial string, placeholder string) (tea.Model, tea.Cmd) {
		m.settingsEditing = true
		m.settingsInput.SetValue(initial)
		m.settingsInput.Placeholder = placeholder
		m.settingsInput.CursorEnd()
		m.settingsInput.Focus()
		m.err = nil
		return m, textinput.Blink
	}

	if m.settingsEditing {
		switch key {
		case "esc":
			m.settingsEditing = false
			m.settingsInput.Blur()
			m.settingsInput.SetValue("")
			m.err = nil
			return m, nil
		case "enter":
			if !applyEditValue(m.settingsInput.Value()) {
				return m, nil
			}
			m.settingsEditing = false
			m.settingsInput.Blur()
			m.settingsInput.SetValue("")
			m.err = nil
			return m, nil
		}
		var cmd tea.Cmd
		m.settingsInput, cmd = m.settingsInput.Update(msg)
		return m, cmd
	}

	switch key {
	case "esc", "q":
		// Auto-save changes when leaving settings
		if m.cfg != m.cfgOriginal {
			if err := config.Save(m.cfg); err != nil {
				m.err = fmt.Errorf("failed to save settings: %v", err)
				return m, nil
			}
			// Reinitialize sync manager if sync settings changed
			if m.cfg.Sync != m.cfgOriginal.Sync && m.store != nil {
				syncMgr, err := sync.NewManager(&m.cfg, m.store, m.masterPassword)
				if err == nil {
					m.syncManager = syncMgr
				}
			}
			m.err = fmt.Errorf("✓ Settings saved")
		}
		m.viewMode = m.settingsPrevView
		return m, nil

	case "up", "k":
		if key == "k" && !m.cfg.UI.VimMode {
			return m, nil
		}
		if m.settingsIdx > 0 {
			m.settingsIdx--
		}
		m.err = nil
		return m, nil

	case "down", "j":
		if key == "j" && !m.cfg.UI.VimMode {
			return m, nil
		}
		if m.settingsIdx < maxIdx {
			m.settingsIdx++
		}
		m.err = nil
		return m, nil

	case "left", "h":
		if key == "h" && !m.cfg.UI.VimMode {
			return m, nil
		}
		switch m.settingsIdx {
		case 2:
			// Host key policy (cycle backwards)
			switch m.cfg.SSH.HostKeyPolicy {
			case config.HostKeyStrict:
				m.cfg.SSH.HostKeyPolicy = config.HostKeyAcceptNew
			case config.HostKeyOff:
				m.cfg.SSH.HostKeyPolicy = config.HostKeyStrict
			default:
				m.cfg.SSH.HostKeyPolicy = config.HostKeyOff
			}
		case 3:
			// Keepalive seconds
			m.cfg.SSH.KeepAliveSeconds = max(10, m.cfg.SSH.KeepAliveSeconds-5)
		case 4:
			// TERM mode
			switch m.cfg.SSH.TermMode {
			case config.TermXterm:
				m.cfg.SSH.TermMode = config.TermAuto
			case config.TermCustom:
				m.cfg.SSH.TermMode = config.TermXterm
			default:
				m.cfg.SSH.TermMode = config.TermCustom
			}
		case 7:
			// Unix password backend order
			if runtime.GOOS != "windows" && m.cfg.SSH.PasswordAutoLogin {
				switch m.cfg.SSH.PasswordBackendUnix {
				case config.PasswordBackendAskpassFirst:
					m.cfg.SSH.PasswordBackendUnix = config.PasswordBackendSSHPassFirst
				default:
					m.cfg.SSH.PasswordBackendUnix = config.PasswordBackendAskpassFirst
				}
			}
		case 10:
			// Mount quit behavior
			switch m.cfg.Mount.QuitBehavior {
			case config.MountQuitAlwaysUnmount:
				m.cfg.Mount.QuitBehavior = config.MountQuitPrompt
			case config.MountQuitLeaveMounted:
				m.cfg.Mount.QuitBehavior = config.MountQuitAlwaysUnmount
			default:
				m.cfg.Mount.QuitBehavior = config.MountQuitLeaveMounted
			}
		}
		return m, nil

	case "right", "l":
		if key == "l" && !m.cfg.UI.VimMode {
			return m, nil
		}
		switch m.settingsIdx {
		case 2:
			// Host key policy (cycle forwards)
			switch m.cfg.SSH.HostKeyPolicy {
			case config.HostKeyAcceptNew:
				m.cfg.SSH.HostKeyPolicy = config.HostKeyStrict
			case config.HostKeyStrict:
				m.cfg.SSH.HostKeyPolicy = config.HostKeyOff
			default:
				m.cfg.SSH.HostKeyPolicy = config.HostKeyAcceptNew
			}
		case 3:
			m.cfg.SSH.KeepAliveSeconds = min(300, m.cfg.SSH.KeepAliveSeconds+5)
		case 4:
			switch m.cfg.SSH.TermMode {
			case config.TermAuto:
				m.cfg.SSH.TermMode = config.TermXterm
			case config.TermXterm:
				m.cfg.SSH.TermMode = config.TermCustom
			default:
				m.cfg.SSH.TermMode = config.TermAuto
			}
		case 7:
			if runtime.GOOS != "windows" && m.cfg.SSH.PasswordAutoLogin {
				switch m.cfg.SSH.PasswordBackendUnix {
				case config.PasswordBackendSSHPassFirst:
					m.cfg.SSH.PasswordBackendUnix = config.PasswordBackendAskpassFirst
				default:
					m.cfg.SSH.PasswordBackendUnix = config.PasswordBackendSSHPassFirst
				}
			}
		case 10:
			switch m.cfg.Mount.QuitBehavior {
			case config.MountQuitPrompt:
				m.cfg.Mount.QuitBehavior = config.MountQuitAlwaysUnmount
			case config.MountQuitAlwaysUnmount:
				m.cfg.Mount.QuitBehavior = config.MountQuitLeaveMounted
			default:
				m.cfg.Mount.QuitBehavior = config.MountQuitPrompt
			}
		}
		return m, nil

	default:
		// Quick save from anywhere
		if key == "shift+enter" || key == "shift+return" {
			if err := config.Save(m.cfg); err != nil {
				m.err = fmt.Errorf("failed to save settings: %v", err)
				return m, nil
			}
			// Reinitialize sync manager if sync settings changed
			if m.cfg.Sync != m.cfgOriginal.Sync && m.store != nil {
				syncMgr, err := sync.NewManager(&m.cfg, m.store, m.masterPassword)
				if err == nil {
					m.syncManager = syncMgr
				}
			}
			m.viewMode = m.settingsPrevView
			m.err = fmt.Errorf("✓ Settings saved")
			return m, nil
		}

		if key == " " || key == "enter" {
			switch m.settingsIdx {
			case 0:
				m.cfg.UI.VimMode = !m.cfg.UI.VimMode
				return m, nil
			case 1:
				m.cfg.UI.ShowIcons = !m.cfg.UI.ShowIcons
				return m, nil
			case 2:
				// Cycle host key policy forward
				switch m.cfg.SSH.HostKeyPolicy {
				case config.HostKeyAcceptNew:
					m.cfg.SSH.HostKeyPolicy = config.HostKeyStrict
				case config.HostKeyStrict:
					m.cfg.SSH.HostKeyPolicy = config.HostKeyOff
				default:
					m.cfg.SSH.HostKeyPolicy = config.HostKeyAcceptNew
				}
				return m, nil
			case 3:
				return startEdit(fmt.Sprintf("%d", m.cfg.SSH.KeepAliveSeconds), "10-600 (default 60)")
			case 4:
				// Cycle TERM mode forward
				switch m.cfg.SSH.TermMode {
				case config.TermAuto:
					m.cfg.SSH.TermMode = config.TermXterm
				case config.TermXterm:
					m.cfg.SSH.TermMode = config.TermCustom
				default:
					m.cfg.SSH.TermMode = config.TermAuto
				}
				return m, nil
			case 5:
				if m.cfg.SSH.TermMode != config.TermCustom {
					m.err = fmt.Errorf("TERM mode must be 'custom' to edit")
					return m, nil
				}
				return startEdit(m.cfg.SSH.TermCustom, "e.g. xterm-256color")
			case 6:
				m.cfg.SSH.PasswordAutoLogin = !m.cfg.SSH.PasswordAutoLogin
				if m.cfg.SSH.PasswordAutoLogin && (runtime.GOOS == "linux" || runtime.GOOS == "darwin") {
					if err := ssh.CheckSSHPass(); err != nil {
						m.err = fmt.Errorf("Tip: install sshpass for best password auto-login on %s", runtime.GOOS)
					}
				}
				return m, nil
			case 7:
				if runtime.GOOS == "windows" {
					m.err = fmt.Errorf("unix backend is only used on Linux/macOS")
					return m, nil
				}
				if !m.cfg.SSH.PasswordAutoLogin {
					m.err = fmt.Errorf("enable password auto-login first")
					return m, nil
				}
				switch m.cfg.SSH.PasswordBackendUnix {
				case config.PasswordBackendSSHPassFirst:
					m.cfg.SSH.PasswordBackendUnix = config.PasswordBackendAskpassFirst
				default:
					m.cfg.SSH.PasswordBackendUnix = config.PasswordBackendSSHPassFirst
				}
				return m, nil
			case 8:
				m.cfg.Mount.Enabled = !m.cfg.Mount.Enabled
				return m, nil
			case 9:
				return startEdit(m.cfg.Mount.DefaultRemotePath, "empty = remote home (recommended)")
			case 10:
				// Cycle quit behavior forward
				switch m.cfg.Mount.QuitBehavior {
				case config.MountQuitPrompt:
					m.cfg.Mount.QuitBehavior = config.MountQuitAlwaysUnmount
				case config.MountQuitAlwaysUnmount:
					m.cfg.Mount.QuitBehavior = config.MountQuitLeaveMounted
				default:
					m.cfg.Mount.QuitBehavior = config.MountQuitPrompt
				}
				return m, nil
			case 11:
				// Toggle sync enabled
				m.cfg.Sync.Enabled = !m.cfg.Sync.Enabled
				// Reinitialize sync manager when enabled/disabled
				if m.store != nil {
					syncMgr, err := sync.NewManager(&m.cfg, m.store, m.masterPassword)
					if err == nil {
						m.syncManager = syncMgr
					}
				}
				return m, nil
			case 12:
				// Sync repo URL
				if !m.cfg.Sync.Enabled {
					m.err = fmt.Errorf("enable sync first")
					return m, nil
				}
				return startEdit(m.cfg.Sync.RepoURL, "git@github.com:user/hosts.git")
			case 13:
				// Sync SSH key path
				if !m.cfg.Sync.Enabled {
					m.err = fmt.Errorf("enable sync first")
					return m, nil
				}
				return startEdit(m.cfg.Sync.SSHKeyPath, "~/.ssh/id_ed25519 (empty = auto)")
			case 14:
				// Sync branch
				if !m.cfg.Sync.Enabled {
					m.err = fmt.Errorf("enable sync first")
					return m, nil
				}
				return startEdit(m.cfg.Sync.Branch, "main")
			case 15:
				// Sync local path
				if !m.cfg.Sync.Enabled {
					m.err = fmt.Errorf("enable sync first")
					return m, nil
				}
				return startEdit(m.cfg.Sync.LocalPath, "empty = default path")
			case 16:
				m.err = fmt.Errorf("ℹ %s", m.updateSettingsState().ChannelLabel)
				return m, nil
			case 17:
				m.err = fmt.Errorf("ℹ %s", m.updateSettingsState().VersionLabel)
				return m, nil
			case 18:
				if m.updateChecking || m.updateApplying {
					m.err = fmt.Errorf("ℹ update operation already in progress")
					return m, nil
				}
				m.updateChecking = true
				m.updateRunID++
				runID := m.updateRunID
				m.err = fmt.Errorf("ℹ checking for updates...")
				return m, runUpdateCheckCmd(runID, m.currentVersion, m.cfg)
			case 19:
				if m.updateChecking || m.updateApplying {
					m.err = fmt.Errorf("ℹ update operation already in progress")
					return m, nil
				}
				if m.updateLast == nil {
					m.err = fmt.Errorf("ℹ run 'check now' first")
					return m, nil
				}
				if !m.updateLast.UpdateAvailable {
					m.err = fmt.Errorf("ℹ already on latest stable release")
					return m, nil
				}
				if m.updateLast.ApplyMode == update.ApplyModeGuidance || m.updateLast.ApplyMode == update.ApplyModeNone {
					m.err = fmt.Errorf("ℹ auto-apply not available on this platform yet")
					return m, nil
				}
				m.updateApplying = true
				m.updateRunID++
				runID := m.updateRunID
				m.err = fmt.Errorf("ℹ applying update...")
				return m, runUpdateApplyCmd(runID, *m.updateLast)
			case 20:
				m.err = fmt.Errorf("ℹ %s", m.updateSettingsState().PathHealth)
				return m, nil
			case 21:
				if m.updateChecking || m.updateApplying {
					m.err = fmt.Errorf("ℹ update operation already in progress")
					return m, nil
				}
				if m.updateLast == nil || m.updateLast.PathHealth.Healthy || strings.TrimSpace(m.updateLast.PathHealth.DesiredPath) == "" {
					m.err = fmt.Errorf("ℹ no PATH conflict detected")
					return m, nil
				}
				m.updateApplying = true
				m.updateRunID++
				runID := m.updateRunID
				m.err = fmt.Errorf("ℹ fixing PATH...")
				return m, runUpdatePathFixCmd(runID, m.updateLast.PathHealth.DesiredPath)
			}
		}
	}

	return m, nil
}

// validateForm validates the modal form data
func (m Model) validateForm() error {
	if m.modalForm == nil {
		return fmt.Errorf("No form data")
	}

	// Label is optional.

	// Validate hostname
	if strings.TrimSpace(m.modalForm.hostnameInput.Value()) == "" {
		return fmt.Errorf("⚠ Host cannot be empty")
	}

	// Validate username
	if strings.TrimSpace(m.modalForm.usernameInput.Value()) == "" {
		return fmt.Errorf("⚠ Username cannot be empty")
	}

	// Validate port
	if m.modalForm.portInput.Value() == "" {
		return fmt.Errorf("⚠ Port cannot be empty")
	}

	// Validate port is a number
	port := 0
	_, err := fmt.Sscanf(m.modalForm.portInput.Value(), "%d", &port)
	if err != nil {
		return fmt.Errorf("⚠ Port must be a valid number")
	}

	// Validate port range
	if port < 1 || port > 65535 {
		return fmt.Errorf("⚠ Port must be between 1 and 65535")
	}

	// Validate auth details
	switch m.modalForm.authMethod {
	case ui.AuthPassword:
		// Password is optional. Blank keeps existing password on edit or falls back
		// to connect-time prompt when no stored secret exists.
	case ui.AuthKeyPaste:
		pastedKey := strings.TrimSpace(m.modalForm.pastedKeyInput.Value())
		if pastedKey == "" {
			return fmt.Errorf("⚠ Please paste your SSH private key or switch auth method")
		}
		if err := ssh.ValidatePrivateKey(pastedKey); err != nil {
			return fmt.Errorf("⚠ Invalid private key: %v", err)
		}
	case ui.AuthKeyGen:
		switch m.modalForm.keyType {
		case "ed25519", "rsa", "ecdsa":
		default:
			return fmt.Errorf("⚠ Invalid key type")
		}
	default:
		return fmt.Errorf("⚠ Invalid auth method")
	}

	return nil
}

// getFilteredHosts returns hosts filtered by search query
func (m Model) getFilteredHosts() []Host {
	query := m.searchInput.Value()
	if query == "" {
		return m.hosts
	}

	var filtered []Host
	query = strings.ToLower(query)
	for _, host := range m.hosts {
		if strings.Contains(strings.ToLower(host.Label), query) ||
			strings.Contains(strings.ToLower(host.Hostname), query) ||
			strings.Contains(strings.ToLower(host.Username), query) {
			filtered = append(filtered, host)
		}
	}
	return filtered
}

// View renders the UI
func (m Model) View() string {
	const hideCursorAndMoveAway = "\x1b[?25l\x1b[999;999H"

	switch m.viewMode {
	case ViewModeSetup:
		return m.styles.RenderSetupView(m.width, m.height, m.loginInput, m.confirmInput, m.setupFocus, m.err) + hideCursorAndMoveAway
	case ViewModeLogin:
		return m.styles.RenderLoginView(m.width, m.height, m.loginInput, m.err) + hideCursorAndMoveAway
	case ViewModeList:
		m.rebuildListItems()
		hostsInterface := m.listItemsToRenderData()
		syncStatus := ""
		syncStage := ""
		if m.syncManager != nil && m.syncManager.IsEnabled() {
			syncStatus = m.syncManager.StatusString()
			syncStage = m.syncManager.StageString()
		}
		syncActivity := &ui.SyncActivity{Active: m.syncing, Frame: m.syncAnimFrame, Progress: m.syncProgress, Stage: syncStage}
		return m.styles.RenderListViewWithSync(m.width, m.height, hostsInterface, m.selectedIdx, m.searchInput.Value(), m.isSearching, m.err, syncStatus, syncActivity) + hideCursorAndMoveAway
	case ViewModeHelp:
		return m.styles.RenderHelpView(m.width, m.height) + hideCursorAndMoveAway
	case ViewModeAddHost, ViewModeEditHost:
		return m.renderModalView() + hideCursorAndMoveAway
	case ViewModeDeleteHost:
		return m.renderDeleteView() + hideCursorAndMoveAway
	case ViewModeCreateGroup, ViewModeRenameGroup:
		return m.renderGroupInputView() + hideCursorAndMoveAway
	case ViewModeDeleteGroup:
		return m.renderDeleteGroupView() + hideCursorAndMoveAway
	case ViewModeSpotlight:
		return m.renderSpotlightView() + hideCursorAndMoveAway
	case ViewModeQuitConfirm:
		return m.renderQuitConfirmView() + hideCursorAndMoveAway
	case ViewModeSettings:
		return m.renderSettingsView() + hideCursorAndMoveAway
	default:
		return "Unknown view mode" + hideCursorAndMoveAway
	}
}

func (m Model) renderSettingsView() string {
	return m.styles.RenderSettingsView(m.width, m.height, m.cfg, m.updateSettingsState(), m.settingsIdx, m.settingsEditing, m.settingsInput, m.err)
}

func (m Model) updateSettingsState() ui.UpdateSettingsState {
	state := ui.UpdateSettingsState{
		Checking: m.updateChecking,
		Applying: m.updateApplying,
	}
	if m.updateLast != nil {
		state.ChannelLabel = update.ChannelLabel(m.updateLast.Channel, m.updateLast.ChannelDetail)
		current := strings.TrimSpace(m.updateLast.CurrentVersion)
		if current == "" {
			current = "(unknown)"
		}
		latest := strings.TrimSpace(m.updateLast.LatestVersion)
		if latest == "" {
			latest = "(unknown)"
		}
		state.VersionLabel = current + " -> " + latest
		state.PathHealth = update.PathHealthLabel(m.updateLast.PathHealth)
		state.CanApply = m.updateLast.UpdateAvailable && m.updateLast.ApplyMode != update.ApplyModeGuidance && m.updateLast.ApplyMode != update.ApplyModeNone
		state.CanFixPath = !m.updateLast.PathHealth.Healthy && strings.TrimSpace(m.updateLast.PathHealth.DesiredPath) != ""
	}
	if state.VersionLabel == "" {
		state.VersionLabel = normalizeVersionStringForUI(m.currentVersion) + " -> (not checked)"
	}
	if state.ChannelLabel == "" {
		state.ChannelLabel = "(not checked)"
	}
	if state.PathHealth == "" {
		state.PathHealth = "(not checked)"
	}
	return state
}

func normalizeVersionStringForUI(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "(unknown)"
	}
	return v
}

func (m Model) listItemsToRenderData() []interface{} {
	hostsInterface := make([]interface{}, 0, len(m.listItems))
	for _, item := range m.listItems {
		switch item.Kind {
		case ListItemGroup:
			hostsInterface = append(hostsInterface, map[string]interface{}{
				"Kind":      "group",
				"GroupName": item.GroupName,
				"Count":     item.Count,
				"Collapsed": m.collapsed[item.GroupName],
			})
		case ListItemNewGroup:
			hostsInterface = append(hostsInterface, map[string]interface{}{
				"Kind": "new_group",
			})
		case ListItemHost:
			host := item.Host
			mounted := false
			mountPath := ""
			if m.mountManager != nil {
				if ok, mt := m.mountManager.IsMounted(host.ID); ok && mt != nil {
					mounted = true
					mountPath = mt.LocalPath
				}
			}
			hostsInterface = append(hostsInterface, map[string]interface{}{
				"Kind":          "host",
				"ID":            host.ID,
				"Label":         host.Label,
				"GroupName":     host.GroupName,
				"Tags":          hostSearchTags(host),
				"Hostname":      host.Hostname,
				"Username":      host.Username,
				"Port":          host.Port,
				"HasKey":        host.HasKey,
				"KeyType":       host.KeyType,
				"Mounted":       mounted,
				"MountPath":     mountPath,
				"ShowIcons":     m.cfg.UI.ShowIcons,
				"CreatedAt":     host.CreatedAt,
				"LastConnected": host.LastConnected,
				"Indent":        1,
			})
		}
	}
	return hostsInterface
}

func (m Model) renderGroupInputView() string {
	title := "Create Group"
	submit := "Create"
	if m.viewMode == ViewModeRenameGroup {
		title = "Rename Group"
		submit = "Rename"
	}
	return m.styles.RenderGroupInputModal(m.width, m.height, title, m.groupInput, submit, m.groupFocus)
}

func (m Model) renderDeleteGroupView() string {
	count := m.hostCountForGroup(m.groupOldName)
	return m.styles.RenderDeleteGroupModal(m.width, m.height, m.groupOldName, count, m.groupDeleteFocus)
}

func (m Model) renderQuitConfirmView() string {
	var lines []string
	if m.mountManager != nil {
		for _, mt := range m.mountManager.ListActive() {
			host := strings.TrimSpace(mt.Hostname)
			if host == "" {
				host = fmt.Sprintf("host_%d", mt.HostID)
			}
			lines = append(lines, fmt.Sprintf("%s → %s", host, mt.LocalPath))
		}
	}
	return m.styles.RenderQuitModal(m.width, m.height, lines, m.quitFocus)
}

// renderSpotlightView renders the spotlight overlay
func (m Model) renderSpotlightView() string {
	items := m.spotlightItems
	if items == nil {
		m.rebuildListItems()
		items = m.buildSpotlightItems(m.searchInput.Value())
	}
	out := make([]interface{}, 0, len(items))
	for _, it := range items {
		if it.Kind == SpotlightItemGroup {
			out = append(out, map[string]interface{}{
				"Kind":      "group",
				"GroupName": it.GroupName,
				"Count":     m.hostCountForGroup(it.GroupName),
			})
			continue
		}
		mounted := false
		if m.mountManager != nil {
			mounted, _ = m.mountManager.IsMounted(it.Host.ID)
		}
		out = append(out, map[string]interface{}{
			"Kind":      "host",
			"Label":     it.Host.Label,
			"GroupName": it.Host.GroupName,
			"Hostname":  it.Host.Hostname,
			"Username":  it.Host.Username,
			"Mounted":   mounted,
			"Indent":    it.Indent,
		})
	}

	return m.styles.RenderSpotlight(m.width, m.height, m.searchInput, out, m.selectedIdx, m.armedSFTP, m.armedMount, m.armedUnmount)
}

// renderModalView renders the add/edit modal
func (m Model) renderModalView() string {
	if m.modalForm == nil {
		return "Error: No form data"
	}

	// Convert ModalForm to ui.ModalFormData
	// Note: We are passing copies of the models, which is fine for rendering
	formData := &ui.ModalFormData{
		Label:        m.modalForm.labelInput,
		Group:        m.modalForm.groupInput,
		Tags:         m.modalForm.tagsInput,
		GroupOptions: m.modalForm.groupOptions,
		GroupIndex:   m.modalForm.groupSelected,
		Hostname:     m.modalForm.hostnameInput,
		Username:     m.modalForm.usernameInput,
		Port:         m.modalForm.portInput,
		AuthMethod:   m.modalForm.authMethod,
		Password:     m.modalForm.passwordInput,
		KeyOption:    m.modalForm.keyOption,
		KeyType:      m.modalForm.keyType,
		PastedKey:    m.modalForm.pastedKeyInput,
		FocusedField: m.modalForm.focusedField,
		// TitleSuffix:  fmt.Sprintf(" [F:%d A:%d]", m.modalForm.focusedField, m.modalForm.authMethod), // Debug
	}

	isEdit := m.viewMode == ViewModeEditHost
	return m.styles.RenderAddHostModal(m.width, m.height, formData, isEdit)
}

// renderDeleteView renders the delete confirmation modal
func (m Model) renderDeleteView() string {
	host, ok := m.selectedHost()
	if !ok {
		return m.styles.Error.Render("No host selected")
	}
	return m.styles.RenderDeleteModal(m.width, m.height, host.Hostname, host.Username, m.deleteConfirmFocus)
}

// connectToHost initiates an SSH connection to the given host
func (m Model) connectToHost(host Host) (tea.Model, tea.Cmd) {
	m.armedSFTP = false
	m.armedMount = false
	m.armedUnmount = false

	// Get the decrypted key if available
	var privateKey string
	var password string
	if host.HasKey && host.KeyType != "password" {
		key, err := m.store.GetHostSecret(host.ID)
		if err != nil {
			m.err = fmt.Errorf("failed to decrypt key: %v", err)
			return m, nil
		}
		if err := ssh.ValidatePrivateKey(key); err != nil {
			m.err = fmt.Errorf("stored private key is invalid format: %v", err)
			return m, nil
		}
		privateKey = key
	}
	if host.KeyType == "password" && m.cfg.SSH.PasswordAutoLogin {
		secret, err := m.store.GetHostSecret(host.ID)
		if err != nil {
			m.err = fmt.Errorf("failed to decrypt password: %v", err)
			return m, nil
		}
		password = secret
	}

	// Build the SSH connection
	term := ""
	switch m.cfg.SSH.TermMode {
	case config.TermXterm:
		term = "xterm-256color"
	case config.TermCustom:
		term = strings.TrimSpace(m.cfg.SSH.TermCustom)
	}
	conn := ssh.Connection{
		Hostname:            host.Hostname,
		Username:            host.Username,
		Port:                host.Port,
		PrivateKey:          privateKey,
		Password:            password,
		PasswordBackendUnix: string(m.cfg.SSH.PasswordBackendUnix),
		HostKeyPolicy:       string(m.cfg.SSH.HostKeyPolicy),
		KeepAliveSeconds:    m.cfg.SSH.KeepAliveSeconds,
		Term:                term,
	}

	cmd, tempKey, err := ssh.Connect(conn)
	if err != nil {
		m.err = fmt.Errorf("failed to prepare SSH connection: %v", err)
		return m, nil
	}

	// Update last connected time
	if m.store != nil {
		m.store.UpdateLastConnected(host.ID)
	}

	// Use tea.ExecProcess to suspend TUI and run SSH
	return m, tea.Sequence(
		tea.ShowCursor,
		tea.ExecProcess(cmd, func(err error) tea.Msg {
			// Cleanup temp key file
			if tempKey != nil {
				tempKey.Cleanup()
			}
			return sshFinishedMsg{err: err, hostname: host.Hostname, proto: "SSH"}
		}),
	)
}

func (m Model) connectToHostSFTP(host Host) (tea.Model, tea.Cmd) {
	m.armedSFTP = false
	m.armedMount = false
	m.armedUnmount = false

	// Get the decrypted key if available
	var privateKey string
	var password string
	if host.HasKey && host.KeyType != "password" {
		key, err := m.store.GetHostSecret(host.ID)
		if err != nil {
			m.err = fmt.Errorf("failed to decrypt key: %v", err)
			return m, nil
		}
		if err := ssh.ValidatePrivateKey(key); err != nil {
			m.err = fmt.Errorf("stored private key is invalid format: %v", err)
			return m, nil
		}
		privateKey = key
	}
	if host.KeyType == "password" && m.cfg.SSH.PasswordAutoLogin {
		secret, err := m.store.GetHostSecret(host.ID)
		if err != nil {
			m.err = fmt.Errorf("failed to decrypt password: %v", err)
			return m, nil
		}
		password = secret
	}

	term := ""
	switch m.cfg.SSH.TermMode {
	case config.TermXterm:
		term = "xterm-256color"
	case config.TermCustom:
		term = strings.TrimSpace(m.cfg.SSH.TermCustom)
	}
	conn := ssh.Connection{
		Hostname:            host.Hostname,
		Username:            host.Username,
		Port:                host.Port,
		PrivateKey:          privateKey,
		Password:            password,
		PasswordBackendUnix: string(m.cfg.SSH.PasswordBackendUnix),
		HostKeyPolicy:       string(m.cfg.SSH.HostKeyPolicy),
		KeepAliveSeconds:    m.cfg.SSH.KeepAliveSeconds,
		Term:                term,
	}

	cmd, tempKey, err := ssh.ConnectSFTP(conn)
	if err != nil {
		m.err = fmt.Errorf("failed to prepare SFTP session: %v", err)
		return m, nil
	}

	if m.store != nil {
		m.store.UpdateLastConnected(host.ID)
	}

	return m, tea.Sequence(
		tea.ShowCursor,
		tea.ExecProcess(cmd, func(err error) tea.Msg {
			if tempKey != nil {
				tempKey.Cleanup()
			}
			return sshFinishedMsg{err: err, hostname: host.Hostname, proto: "SFTP"}
		}),
	)
}

func (m Model) handleMountEnter(host Host) (tea.Model, tea.Cmd) {
	m.armedSFTP = false

	if m.mountManager == nil {
		m.armedMount = false
		m.armedUnmount = false
		m.err = fmt.Errorf("⚠ mount manager not initialized")
		return m, nil
	}

	// Unmount flow
	if m.armedUnmount {
		m.armedUnmount = false
		cmd, localPath, err := m.mountManager.PrepareUnmount(host.ID)
		if err != nil {
			m.err = err
			return m, nil
		}
		_ = localPath
		return m, tea.Sequence(
			tea.ShowCursor,
			tea.ExecProcess(cmd, func(err error) tea.Msg {
				return mountFinishedMsg{action: "unmount", hostID: host.ID, local: localPath, err: err}
			}),
		)
	}

	// Mount flow
	m.armedMount = false

	// Get the decrypted key if available (optional; sshfs can also use default agent keys).
	var privateKey string
	if host.HasKey && host.KeyType != "password" {
		key, err := m.store.GetHostSecret(host.ID)
		if err != nil {
			m.err = fmt.Errorf("failed to decrypt key: %v", err)
			return m, nil
		}
		if err := ssh.ValidatePrivateKey(key); err != nil {
			m.err = fmt.Errorf("stored private key is invalid format: %v", err)
			return m, nil
		}
		privateKey = key
	}

	remotePath := m.cfg.Mount.DefaultRemotePath
	display := strings.TrimSpace(host.Label)
	if display == "" {
		display = host.Hostname
	}

	term := ""
	switch m.cfg.SSH.TermMode {
	case config.TermXterm:
		term = "xterm-256color"
	case config.TermCustom:
		term = strings.TrimSpace(m.cfg.SSH.TermCustom)
	}
	prep, err := m.mountManager.PrepareMount(host.ID, ssh.Connection{
		Hostname:         host.Hostname,
		Username:         host.Username,
		Port:             host.Port,
		PrivateKey:       privateKey,
		HostKeyPolicy:    string(m.cfg.SSH.HostKeyPolicy),
		KeepAliveSeconds: m.cfg.SSH.KeepAliveSeconds,
		Term:             term,
	}, remotePath, display)
	if err != nil {
		m.err = err
		return m, nil
	}

	m.pendingMount = prep
	return m, tea.Sequence(
		tea.ShowCursor,
		tea.ExecProcess(prep.Cmd(), func(err error) tea.Msg {
			return mountFinishedMsg{action: "mount", hostID: host.ID, local: prep.LocalPath, err: err}
		}),
	)
}

// sshFinishedMsg is sent when an SSH/SFTP session ends
type sshFinishedMsg struct {
	err      error
	hostname string
	proto    string
}

type mountFinishedMsg struct {
	action string // "mount" | "unmount"
	hostID int
	local  string
	err    error
}

type syncFinishedMsg struct {
	runID  int
	result *sync.SyncResult
}

type syncAnimTickMsg struct {
	runID int
}

type updateCheckedMsg struct {
	runID  int
	result *update.CheckResult
	err    error
}

type updateAppliedMsg struct {
	runID          int
	result         *update.ApplyResult
	handoffStarted bool
	err            error
}

type updatePathFixedMsg struct {
	runID      int
	pathHealth update.PathHealth
	err        error
}

type quitFinishedMsg struct{}

type clearErrMsg struct {
	seq int
}

func runSyncCmd(runID int, mgr *sync.Manager) tea.Cmd {
	return func() tea.Msg {
		if mgr == nil {
			return syncFinishedMsg{runID: runID, result: &sync.SyncResult{Success: false, Message: "sync manager is nil", Timestamp: time.Now()}}
		}
		return syncFinishedMsg{runID: runID, result: mgr.Sync()}
	}
}

func syncAnimTickCmd(runID int) tea.Cmd {
	return tea.Tick(120*time.Millisecond, func(time.Time) tea.Msg {
		return syncAnimTickMsg{runID: runID}
	})
}

func runUpdateCheckCmd(runID int, currentVersion string, cfg config.Config) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		result, err := update.Check(ctx, currentVersion, &cfg)
		if err != nil {
			return updateCheckedMsg{runID: runID, err: err}
		}
		return updateCheckedMsg{runID: runID, result: &result}
	}
}

func runUpdateApplyCmd(runID int, check update.CheckResult) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		exe, err := os.Executable()
		if err != nil {
			return updateAppliedMsg{runID: runID, err: err}
		}
		result, err := update.Apply(ctx, check, exe)
		if err != nil {
			return updateAppliedMsg{runID: runID, err: err}
		}
		handoffStarted := false
		if result.Handoff != nil {
			if err := update.LaunchHandoff(result.Handoff); err != nil {
				return updateAppliedMsg{runID: runID, err: err}
			}
			handoffStarted = true
		}
		return updateAppliedMsg{runID: runID, result: &result, handoffStarted: handoffStarted}
	}
}

func runUpdatePathFixCmd(runID int, desiredExe string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		ph, err := update.FixPathConflicts(ctx, desiredExe)
		if err != nil {
			return updatePathFixedMsg{runID: runID, err: err}
		}
		return updatePathFixedMsg{runID: runID, pathHealth: ph}
	}
}

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func normalizePrivateKey(key string) string {
	// Fix common paste issues: CRLF newlines and missing trailing newline.
	key = strings.ReplaceAll(key, "\r\n", "\n")
	key = strings.ReplaceAll(key, "\r", "\n")
	if key != "" && !strings.HasSuffix(key, "\n") {
		key += "\n"
	}
	return key
}
