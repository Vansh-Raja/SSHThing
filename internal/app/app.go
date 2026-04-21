package app

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/Vansh-Raja/SSHThing/internal/authtoken"
	"github.com/Vansh-Raja/SSHThing/internal/config"
	"github.com/Vansh-Raja/SSHThing/internal/db"
	"github.com/Vansh-Raja/SSHThing/internal/mount"
	syncpkg "github.com/Vansh-Raja/SSHThing/internal/sync"
	"github.com/Vansh-Raja/SSHThing/internal/teamcache"
	"github.com/Vansh-Raja/SSHThing/internal/teams"
	"github.com/Vansh-Raja/SSHThing/internal/teamsclient"
	"github.com/Vansh-Raja/SSHThing/internal/teamssession"
	"github.com/Vansh-Raja/SSHThing/internal/ui"
	"github.com/Vansh-Raja/SSHThing/internal/update"
	tea "github.com/charmbracelet/bubbletea"
)

// Model represents the application state
type Model struct {
	store  *db.Store
	hosts  []Host
	groups []string

	listItems   []ListItem
	selectedIdx int
	collapsed   map[string]bool

	// Navigation
	appMode      int
	page         int // PageHome, PageProfile, PageSettings, PageTokens, PageTeams
	overlay      int // OverlayNone, OverlayLogin, etc.
	personalPage int
	teamsPage    int

	width  int
	height int

	// Theme + Icons
	theme    ui.Theme
	themeIdx int
	icons    ui.IconSet
	iconIdx  int
	tick     int

	// Config
	cfg         config.Config
	cfgOriginal config.Config

	// Login / Setup
	loginField  ui.FormField
	setupFields [2]ui.FormField // [password, confirm]
	setupFocus  int             // 0=password, 1=confirm, 2=submit
	loginError  string

	// Search
	searchQuery    string
	spotlightItems []SpotlightItem

	// Add/Edit host form
	formFields             []ui.FormField // [label, tags, hostname, port, username, authDetail]
	formGroups             []string
	formGroupIdx           int
	formAuthOpts           []string
	formAuthIdx            int
	formKeyTypes           []string
	formKeyIdx             int
	formFocus              int
	formEditing            bool
	formEditIdx            int // -1 for add, >=0 for edit index
	formScrollOffset       int
	formSecretRevealed     bool
	formTeamHostID         string
	formTeamCredentialMode string
	formTeamCredentialType string

	// Delete host
	deleteCursor int // 0=delete, 1=cancel

	// Group overlays
	groupInputValue   string
	groupInputCursor  int
	groupOldName      string
	groupFocus        int // 0=input, 1=action, 2=cancel
	groupDeleteCursor int // 0=delete, 1=cancel

	// Quit overlay
	quitCursor int // 0,1,2

	// Armed modes
	armedSFTP    bool
	armedMount   bool
	armedUnmount bool

	// Mount
	mountManager *mount.Manager
	pendingMount *mount.PreparedMount

	// Settings
	settingsItems     []ui.SettingsItem
	settingsCursor    int
	settingsFilter    string
	settingsSearching bool
	settingsEditing   bool
	settingsEditVal   string

	// Tokens
	tokenSummaries    []authtoken.TokenSummary
	tokenIdx          int
	tokenHostIdx      int
	tokenHostPick     map[int]bool
	tokenMode         int
	tokenNameValue    string
	tokenRevealOpen   bool
	tokenRevealValue  string
	tokenRevealCopied bool

	// Teams
	teamsClient         *teamsclient.Client
	teamsSession        teamssession.Session
	teamsCache          teamcache.Cache
	teamsState          int
	teamsItems          []teams.TeamHost
	teamsCursor         int
	teamsList           []teams.TeamSummary
	teamsCurrentTeamID  string
	teamsImportMode     bool
	teamsImportConflict *teamsImportConflictState

	// Profile
	profileState            int
	profileShowOpenTeamsCTA bool
	profileLastAuthURL      string
	profilePendingAuth      *teams.CliAuthStartResponse
	profileDisplayName      string
	profileEmail            string
	profileAuthRunID        int

	// Sync
	syncManager    *syncpkg.Manager
	masterPassword string
	syncing        bool
	syncRunID      int
	syncAnimFrame  int
	syncProgress   float64

	// Update
	currentVersion string
	updateChecking bool
	updateApplying bool
	updateRunID    int
	updateLast     *update.CheckResult

	// Error
	err    error
	errSeq int
}

// NewModel creates a new application model
func NewModel() Model {
	return NewModelWithVersion("dev")
}

// NewModelWithVersion creates a new application model with explicit binary version.
func NewModelWithVersion(version string) Model {
	cfg, _ := config.Load()
	teamsSession, _ := teamssession.Load()
	teamsCache, _ := teamcache.Load()
	if teamsSession.Valid() && teamsSession.Expired(time.Now()) {
		teamsSession = teamssession.Session{}
		teamsCache = teamcache.Cache{}
	}

	theme, themeIdx := ui.ThemeByName(cfg.UI.Theme)
	icons, iconIdx := ui.IconSetByName(cfg.UI.IconSet)

	// First-run detection
	overlay := OverlayLogin
	exists, _ := db.Exists()
	if !exists {
		overlay = OverlaySetup
	}

	loginField := ui.NewMaskedField("password")
	var setupFields [2]ui.FormField
	setupFields[0] = ui.NewMaskedField("password")
	setupFields[1] = ui.NewMaskedField("confirm")

	m := Model{
		cfg:            cfg,
		cfgOriginal:    cfg,
		hosts:          []Host{},
		groups:         []string{},
		listItems:      []ListItem{},
		selectedIdx:    0,
		appMode:        appModePersonal,
		page:           PageHome,
		personalPage:   PageHome,
		teamsPage:      PageTeams,
		overlay:        overlay,
		collapsed:      map[string]bool{},
		theme:          theme,
		themeIdx:       themeIdx,
		icons:          icons,
		iconIdx:        iconIdx,
		loginField:     loginField,
		setupFields:    setupFields,
		quitCursor:     0,
		mountManager:   mount.NewManager(),
		tokenSummaries: []authtoken.TokenSummary{},
		tokenHostPick:  map[int]bool{},
		tokenMode:      tokenModeList,
		teamsClient:    teamsclient.New(cloudServiceBaseURL()),
		teamsSession:   teamsSession,
		teamsCache:     teamsCache,
		teamsState:     teamsStateZero,
		teamsItems:     append([]teams.TeamHost(nil), teamsCache.Hosts...),
		teamsList:      append([]teams.TeamSummary(nil), teamsCache.Teams...),
		teamsCurrentTeamID: func() string {
			if teamsSession.CurrentTeamID != "" {
				return teamsSession.CurrentTeamID
			}
			if cfg.Teams.LastTeamID != "" {
				return cfg.Teams.LastTeamID
			}
			return teamsCache.CurrentTeamID
		}(),
		currentVersion: strings.TrimSpace(version),
		formEditIdx:    -1,
	}
	m.syncProfileFromSession()
	return m
}

// Init initializes the application
func (m Model) Init() tea.Cmd {
	return tea.Batch(tickCmd(), tea.HideCursor)
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	prevErr := ""
	if m.err != nil {
		prevErr = m.err.Error()
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Global quit
		if msg.String() == "ctrl+c" {
			return m.requestQuit()
		}
		if m.overlay == OverlayNone && (msg.String() == "T" || msg.String() == "shift+t") {
			m.toggleAppMode()
			return m, m.errorAutoClearCmd(prevErr)
		}
		// Dispatch to overlay or page
		var nextModel tea.Model
		var nextCmd tea.Cmd
		if m.overlay != OverlayNone {
			nextModel, nextCmd = m.handleOverlayKeys(msg)
		} else {
			nextModel, nextCmd = m.handlePageKeys(msg)
		}
		nm, ok := nextModel.(Model)
		if !ok {
			return nextModel, nextCmd
		}
		return nm, tea.Batch(nextCmd, nm.errorAutoClearCmd(prevErr))

	case tickMsg:
		m.tick++
		return m, tickCmd()

	case profileAuthPolledMsg:
		if msg.runID != m.profileAuthRunID || m.profileState != profileStateSigningIn || m.profilePendingAuth == nil {
			return m, nil
		}
		if msg.err != nil {
			m.err = msg.err
			return m, pollProfileAuthCmd(m.profileAuthRunID, m.teamsClient, m.profilePendingAuth.SessionID, m.profilePendingAuth.PollSecret, time.Duration(m.profilePendingAuth.PollIntervalSeconds)*time.Second)
		}
		switch msg.result.Status {
		case "completed":
			if msg.result.User == nil {
				m.err = fmt.Errorf("sign-in completed but returned incomplete session data")
				m.cancelProfileSignIn()
				return m, m.errorAutoClearCmd(prevErr)
			}
			m.completeProfileSignIn(msg.result)
			m.err = fmt.Errorf("✓ Signed in")
			return m, m.errorAutoClearCmd(prevErr)
		case "expired":
			m.cancelProfileSignIn()
			m.err = fmt.Errorf("cloud sign-in expired")
			return m, m.errorAutoClearCmd(prevErr)
		default:
			return m, pollProfileAuthCmd(m.profileAuthRunID, m.teamsClient, m.profilePendingAuth.SessionID, m.profilePendingAuth.PollSecret, time.Duration(m.profilePendingAuth.PollIntervalSeconds)*time.Second)
		}

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
			m.err = fmt.Errorf("\u26A0 sync failed: empty result")
			return m, m.errorAutoClearCmd(prevErr)
		}
		if msg.result.Success {
			m.loadHosts()
			m.loadGroups()
			m.rebuildListItems()
			m.err = fmt.Errorf("\u2713 Sync: \u2193%d \u2191%d", msg.result.HostsPulled, msg.result.HostsPushed)
		} else {
			m.err = fmt.Errorf("\u26A0 %s", msg.result.Message)
		}
		return m, m.errorAutoClearCmd(prevErr)

	case updateCheckedMsg:
		if msg.runID != m.updateRunID {
			return m, nil
		}
		m.updateChecking = false
		if msg.err != nil {
			m.err = fmt.Errorf("\u26A0 update check failed: %v", msg.err)
			return m, m.errorAutoClearCmd(prevErr)
		}
		if msg.result == nil {
			m.err = fmt.Errorf("\u26A0 update check failed: empty result")
			return m, m.errorAutoClearCmd(prevErr)
		}
		m.updateLast = msg.result
		m.cfg.Updates.LastCheckedAt = msg.result.CheckedAt.Format(time.RFC3339)
		m.cfg.Updates.LastSeenVersion = msg.result.LatestVersion
		m.cfg.Updates.LastSeenTag = msg.result.LatestTag
		m.cfg.Updates.ETagLatest = msg.result.ETag
		if msg.result.UpdateAvailable {
			m.err = fmt.Errorf("\u2713 Update available: %s", msg.result.LatestTag)
		} else {
			m.err = fmt.Errorf("\u2713 Already on latest stable release")
		}
		return m, m.errorAutoClearCmd(prevErr)

	case updateAppliedMsg:
		if msg.runID != m.updateRunID {
			return m, nil
		}
		m.updateApplying = false
		if msg.err != nil {
			m.err = fmt.Errorf("\u26A0 update failed: %v", msg.err)
			return m, m.errorAutoClearCmd(prevErr)
		}
		if msg.result == nil || !msg.result.Success {
			m.err = fmt.Errorf("\u26A0 update failed")
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
		m.err = fmt.Errorf("\u2713 Update applied")
		return m, m.errorAutoClearCmd(prevErr)

	case updatePathFixedMsg:
		if msg.runID != m.updateRunID {
			return m, nil
		}
		m.updateApplying = false
		if msg.err != nil {
			m.err = fmt.Errorf("\u26A0 path fix failed: %v", msg.err)
			return m, m.errorAutoClearCmd(prevErr)
		}
		if m.updateLast != nil {
			m.updateLast.PathHealth = msg.pathHealth
		}
		m.err = fmt.Errorf("\u2713 PATH updated. Open a new terminal for changes.")
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
		m.loadHosts()
		m.overlay = OverlayNone
		m.enterPage(m.modeHomePage())
		if msg.keyType == "password" && m.cfg.SSH.PasswordAutoLogin && !m.cfg.SSH.PasswordNoticeShown {
			m.err = fmt.Errorf("\u2139 password auto-login is enabled \u2014 disable in settings (,) for security")
			m.cfg.SSH.PasswordNoticeShown = true
			_ = config.Save(m.cfg)
		} else if msg.err != nil {
			m.err = fmt.Errorf("%s session ended: %v", msg.proto, msg.err)
		} else {
			m.err = fmt.Errorf("Disconnected from %s", msg.hostname)
		}
		return m, tea.Batch(tea.HideCursor, m.errorAutoClearCmd(prevErr))

	case mountFinishedMsg:
		m.overlay = OverlayNone
		m.enterPage(m.modeHomePage())
		switch msg.action {
		case "mount":
			if m.pendingMount == nil {
				m.err = fmt.Errorf("mount failed: missing pending state")
				return m, tea.Batch(tea.HideCursor, m.errorAutoClearCmd(prevErr))
			}
			if msg.err != nil {
				m.mountManager.AbortMount(m.pendingMount)
				m.pendingMount = nil
				if msg.stderr != "" {
					m.err = fmt.Errorf("mount failed: %s", msg.stderr)
				} else {
					m.err = fmt.Errorf("mount failed: %v", msg.err)
				}
				return m, tea.Batch(tea.HideCursor, m.errorAutoClearCmd(prevErr))
			}
			if err := m.mountManager.FinalizeMount(m.pendingMount); err != nil {
				m.pendingMount = nil
				m.err = err
				return m, tea.Batch(tea.HideCursor, m.errorAutoClearCmd(prevErr))
			}
			local := m.pendingMount.LocalPath
			remote := m.pendingMount.RemotePath()
			m.pendingMount = nil
			m.err = fmt.Errorf("\u2713 Mounted at %s", local)
			if m.store != nil {
				_ = m.store.UpsertMountState(msg.hostID, local, remote)
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
			m.err = fmt.Errorf("\u2713 Unmounted")
			return m, tea.Batch(tea.HideCursor, m.errorAutoClearCmd(prevErr))
		default:
			m.err = fmt.Errorf("mount error: unknown action")
			return m, tea.Batch(tea.HideCursor, m.errorAutoClearCmd(prevErr))
		}

	case quitFinishedMsg:
		return m, tea.Quit
	}

	return m, nil
}

// View renders the UI
func (m Model) View() string {
	r := ui.Renderer{
		Theme: m.theme,
		Icons: m.icons,
		WrapLabels: func() bool {
			if m.appMode == appModeTeams {
				return m.cfg.TeamsUI.WrapLabels
			}
			return m.cfg.UI.WrapLabels
		}(),
		W:              m.width,
		H:              m.height,
		Tick:           m.tick,
		PageIndicators: m.visiblePageIndicators(),
	}

	var content string

	// Overlays that replace entire screen
	switch m.overlay {
	case OverlayLogin:
		content = r.RenderLoginOverlay(ui.LoginViewParams{
			Password: m.loginField,
			Err:      m.loginError,
		})
		return r.WrapFull(content)

	case OverlaySetup:
		content = r.RenderSetupOverlay(ui.SetupViewParams{
			Password: m.setupFields[0],
			Confirm:  m.setupFields[1],
			Focus:    m.setupFocus,
			Err:      m.loginError,
		})
		return r.WrapFull(content)

	case OverlayHelp:
		content = r.RenderHelpOverlay()
		return r.WrapFull(content)

	case OverlaySearch:
		content = r.RenderSearchOverlay(ui.SearchViewParams{
			Query:        m.searchQuery,
			Cursor:       m.selectedIdx,
			Results:      m.buildSearchResults(),
			ArmedSFTP:    m.armedSFTP,
			ArmedMount:   m.armedMount,
			ArmedUnmount: m.armedUnmount,
			CommandMode:  !m.teamsImportMode && strings.HasPrefix(strings.TrimSpace(m.searchQuery), ">"),
			ImportMode:   m.teamsImportMode,
		})
		return r.WrapFull(content)

	case OverlayAddHost:
		if m.formFields != nil {
			content = r.RenderAddHostOverlay(ui.AddHostViewParams{
				IsEdit:         m.formEditIdx >= 0 || m.formTeamHostID != "",
				Fields:         m.formFields,
				Focus:          m.formFocus,
				Editing:        m.formEditing,
				Groups:         m.formGroups,
				GroupIdx:       m.formGroupIdx,
				AuthOptions:    m.formAuthOpts,
				AuthIdx:        m.formAuthIdx,
				KeyTypes:       m.formKeyTypes,
				KeyTypeIdx:     m.formKeyIdx,
				AllowImport:    m.appMode == appModeTeams && m.formEditIdx < 0 && m.formTeamHostID == "",
				AuthLocked:     m.appMode == appModeTeams && m.formTeamCredentialMode == "per_member",
				SecretRevealed: m.formSecretRevealed,
				ScrollOffset:   m.formScrollOffset,
				Err:            m.err,
			})
			return r.WrapFull(content)
		}

	case OverlayDeleteHost:
		if m.appMode == appModeTeams {
			host, ok := m.teamsCurrentHost()
			if ok {
				lbl := host.Label
				if lbl == "" {
					lbl = host.Hostname
				}
				content = r.RenderDeleteHostOverlay(ui.DeleteHostViewParams{
					Label:        lbl,
					Hostname:     host.Hostname,
					Username:     host.Username,
					DeleteCursor: m.deleteCursor,
				})
				return r.WrapFull(content)
			}
		}
		host, ok := m.selectedHost()
		if ok {
			lbl := host.Label
			if lbl == "" {
				lbl = host.Hostname
			}
			content = r.RenderDeleteHostOverlay(ui.DeleteHostViewParams{
				Label:        lbl,
				Hostname:     host.Hostname,
				Username:     host.Username,
				DeleteCursor: m.deleteCursor,
			})
			return r.WrapFull(content)
		}

	case OverlayImportHost:
		if m.teamsImportConflict != nil {
			label := strings.TrimSpace(m.teamsImportConflict.ExistingHost.Label)
			if label == "" {
				label = m.teamsImportConflict.ExistingHost.Hostname
			}
			conn := m.teamsImportConflict.ExistingHost.Hostname
			if user := strings.TrimSpace(m.teamsImportConflict.ExistingHost.Username); user != "" {
				conn = user + "@" + conn
			}
			content = r.RenderImportConflictOverlay(ui.ImportConflictViewParams{
				ExistingLabel: label,
				ExistingConn:  conn,
				Cursor:        m.teamsImportConflict.Cursor,
			})
			return r.WrapFull(content)
		}

	case OverlayCreateGroup:
		content = r.RenderCreateGroupOverlay(ui.GroupInputViewParams{
			InputValue:  m.groupInputValue,
			InputCursor: m.groupInputCursor,
			Focus:       m.groupFocus,
			ActionLabel: "create",
		})
		return r.WrapFull(content)

	case OverlayRenameGroup:
		content = r.RenderRenameGroupOverlay(ui.GroupInputViewParams{
			InputValue:  m.groupInputValue,
			InputCursor: m.groupInputCursor,
			Focus:       m.groupFocus,
			ActionLabel: "rename",
		})
		return r.WrapFull(content)

	case OverlayDeleteGroup:
		content = r.RenderDeleteGroupOverlay(ui.DeleteGroupViewParams{
			GroupName:    m.groupOldName,
			HostCount:    m.hostCountForGroup(m.groupOldName),
			TargetGroup:  m.findAlternateGroup(m.groupOldName),
			DeleteCursor: m.groupDeleteCursor,
		})
		return r.WrapFull(content)

	case OverlayQuit:
		var mountLines []string
		if m.mountManager != nil {
			for _, mt := range m.mountManager.ListActive() {
				host := strings.TrimSpace(mt.Hostname)
				if host == "" {
					host = fmt.Sprintf("host_%d", mt.HostID)
				}
				mountLines = append(mountLines, fmt.Sprintf("%s \u2192 %s", host, mt.LocalPath))
			}
		}
		content = r.RenderQuitOverlay(ui.QuitViewParams{
			Mounts:     mountLines,
			QuitCursor: m.quitCursor,
		})
		return r.WrapFull(content)
	}

	// Pages
	switch m.page {
	case PageHome:
		content = r.RenderHomeView(m.buildHomeViewParams())
	case PageProfile:
		content = r.RenderProfileView(m.buildProfileViewParams())
	case PageSettings:
		if m.settingsEditing && m.settingsCursor < len(m.settingsItems) {
			item := m.settingsItems[m.settingsCursor]
			errStr := ""
			if m.err != nil {
				errStr = m.err.Error()
			}
			content = r.RenderSettingsEditOverlay(ui.SettingsEditParams{
				Label: item.Label,
				Value: m.settingsEditVal,
				Err:   errStr,
			})
			return r.WrapFull(content)
		}
		content = r.RenderSettingsView(ui.SettingsViewParams{
			Items:        m.settingsItems,
			Cursor:       m.settingsCursor,
			Filter:       m.settingsFilter,
			Searching:    m.settingsSearching,
			FilteredIdxs: m.filteredSettingsIdxs(),
			Page:         m.page,
			Err:          m.err,
		})
	case PageTokens:
		// Token sub-overlays take priority over the list
		if m.tokenRevealOpen {
			content = r.RenderTokenRevealOverlay(ui.TokenRevealParams{
				TokenValue: m.tokenRevealValue,
				Copied:     m.tokenRevealCopied,
			})
			return r.WrapFull(content)
		}
		if m.tokenMode == tokenModeCreateName {
			errStr := ""
			if m.err != nil {
				errStr = m.err.Error()
			}
			content = r.RenderTokenCreateNameOverlay(ui.TokenCreateNameParams{
				NameValue: m.tokenNameValue,
				Err:       errStr,
			})
			return r.WrapFull(content)
		}
		if m.tokenMode == tokenModeCreateScope {
			errStr := ""
			if m.err != nil {
				errStr = m.err.Error()
			}
			content = r.RenderTokenSelectHostsOverlay(ui.TokenSelectHostsParams{
				Hosts:  m.buildTokenHostItems(),
				Cursor: m.tokenHostIdx,
				Err:    errStr,
			})
			return r.WrapFull(content)
		}
		content = r.RenderTokensView(ui.TokensViewParams{
			Tokens: m.tokenManagerTokenRows(),
			Cursor: m.tokenIdx,
			Page:   m.page,
			Err:    m.err,
		})
	case PageTeams:
		content = r.RenderTeamsView(m.buildTeamsViewParams())
	default:
		content = "Unknown page"
	}

	return r.WrapFull(content)
}

func (m Model) findAlternateGroup(excluding string) string {
	for _, g := range m.groups {
		if !strings.EqualFold(g, excluding) {
			return g
		}
	}
	return "Ungrouped"
}
