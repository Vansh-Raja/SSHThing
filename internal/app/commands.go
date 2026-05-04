package app

import (
	"context"
	"fmt"
	"runtime"
	"slices"
	"strings"

	"github.com/Vansh-Raja/SSHThing/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) openCommandLine() (tea.Model, tea.Cmd) {
	m.commandQuery = ""
	m.commandCursor = 0
	m.commandItems = m.buildCommandItems("")
	return m, nil
}

func (m Model) handleCommandLineKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.closeCommandLine()
		return m, nil
	case ":":
		m.closeCommandLine()
		return m, nil
	case "up", "ctrl+p", "k":
		if msg.String() == "k" && !m.cfg.UI.VimMode {
			break
		}
		if m.commandCursor > 0 {
			m.commandCursor--
		}
		return m, nil
	case "down", "ctrl+n", "j":
		if msg.String() == "j" && !m.cfg.UI.VimMode {
			break
		}
		if m.commandCursor < len(m.commandItems)-1 {
			m.commandCursor++
		}
		return m, nil
	case "tab":
		if item, ok := m.selectedCommandItem(); ok {
			m.commandQuery = item.Command.Name
			m.commandItems = m.buildCommandItems(m.commandQuery)
			m.commandCursor = 0
		}
		return m, nil
	case "enter":
		item, ok := m.exactCommandItem(m.commandQuery)
		if !ok {
			item, ok = m.selectedCommandItem()
		}
		if !ok {
			return m, nil
		}
		if item.Disabled {
			m.err = fmt.Errorf("\u26A0 %s", item.DisabledReason)
			return m, nil
		}
		m.closeCommandLine()
		return item.Command.Run(m)
	}

	if msg.Type == tea.KeyBackspace {
		m.commandQuery = removeLastRune(m.commandQuery)
	} else {
		for _, r := range msg.Runes {
			if r == ':' && strings.TrimSpace(m.commandQuery) == "" {
				continue
			}
			m.commandQuery += string(r)
		}
	}
	m.commandItems = m.buildCommandItems(m.commandQuery)
	m.commandCursor = 0
	return m, nil
}

func (m *Model) closeCommandLine() {
	m.commandQuery = ""
	m.commandCursor = 0
	m.commandItems = nil
}

func (m Model) commandModeActive() bool {
	return m.commandItems != nil
}

func (m Model) selectedCommandItem() (commandItem, bool) {
	if m.commandCursor < 0 || m.commandCursor >= len(m.commandItems) {
		return commandItem{}, false
	}
	return m.commandItems[m.commandCursor], true
}

func (m Model) exactCommandItem(query string) (commandItem, bool) {
	needle := normalizeCommandQuery(query)
	if needle == "" {
		return commandItem{}, false
	}
	for _, item := range m.commandItems {
		cmd := item.Command
		if strings.EqualFold(needle, cmd.Name) {
			return item, true
		}
		for _, alias := range cmd.Aliases {
			if strings.EqualFold(needle, alias) {
				return item, true
			}
		}
	}
	return commandItem{}, false
}

func (m Model) buildCommandLineView() *ui.CommandLineView {
	if !m.commandModeActive() {
		return nil
	}
	items := make([]ui.CommandLineItem, 0, len(m.commandItems))
	for _, item := range m.commandItems {
		items = append(items, ui.CommandLineItem{
			Name:           item.Command.Name,
			Description:    item.Command.Description,
			Disabled:       item.Disabled,
			DisabledReason: item.DisabledReason,
			Danger:         item.Command.Danger,
		})
	}
	return &ui.CommandLineView{
		Query:  m.commandQuery,
		Cursor: m.commandCursor,
		Items:  items,
	}
}

func (m Model) buildCommandItems(query string) []commandItem {
	query = normalizeCommandQuery(query)
	ctx := m.currentCommandContext()
	commands := m.commandRegistry()
	items := make([]commandItem, 0, len(commands))
	for _, cmd := range commands {
		if !commandVisibleInContext(cmd, ctx) {
			continue
		}
		score, ok := commandScore(cmd, query)
		if !ok {
			continue
		}
		enabled := true
		reason := ""
		if cmd.Enabled != nil {
			enabled, reason = cmd.Enabled(m)
		}
		items = append(items, commandItem{Command: cmd, Score: score, Disabled: !enabled, DisabledReason: reason})
	}
	slices.SortFunc(items, func(a, b commandItem) int {
		if a.Disabled != b.Disabled {
			if a.Disabled {
				return 1
			}
			return -1
		}
		return b.Score - a.Score
	})
	if len(items) > 10 {
		items = items[:10]
	}
	return items
}

func normalizeCommandQuery(query string) string {
	return strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(query), ":"))
}

func commandVisibleInContext(cmd appCommand, ctx commandContext) bool {
	for _, c := range cmd.Contexts {
		if c == commandContextGlobal || c == ctx {
			return true
		}
	}
	return false
}

func commandScore(cmd appCommand, query string) (int, bool) {
	if query == "" {
		return 100, true
	}
	q := strings.ToLower(query)
	names := append([]string{cmd.Name, cmd.Title, cmd.Description}, cmd.Aliases...)
	best := -1
	for _, name := range names {
		n := strings.ToLower(name)
		score, ok := fuzzyScore(q, n)
		if strings.EqualFold(q, cmd.Name) {
			score, ok = 2000, true
		} else if strings.HasPrefix(n, q) {
			score, ok = 1500-len(n), true
		}
		if ok && score > best {
			best = score
		}
	}
	return best, best >= 0
}

func (m Model) currentCommandContext() commandContext {
	switch m.page {
	case PageTeams:
		return commandContextTeams
	case PageProfile:
		return commandContextProfile
	case PageSettings:
		return commandContextSettings
	case PageTokens:
		return commandContextTokens
	default:
		return commandContextHome
	}
}

func (m Model) commandRegistry() []appCommand {
	return []appCommand{
		{ID: "help", Name: "help", Aliases: []string{"h", "?"}, Title: "help", Description: "show commands and shortcuts", Contexts: []commandContext{commandContextGlobal}, Run: runHelpCommand},
		{ID: "search", Name: "search", Aliases: []string{"find", "/"}, Title: "search", Description: "search hosts or page content", Contexts: []commandContext{commandContextGlobal}, Run: runSearchCommand},
		{ID: "settings", Name: "settings", Aliases: []string{"config", "preferences"}, Title: "settings", Description: "open settings", Contexts: []commandContext{commandContextGlobal}, Run: runSettingsCommand},
		{ID: "profile", Name: "profile", Aliases: []string{"account"}, Title: "profile", Description: "open cloud profile", Contexts: []commandContext{commandContextGlobal}, Run: runProfileCommand},
		{ID: "home", Name: "home", Aliases: []string{"personal"}, Title: "home", Description: "go to personal hosts", Contexts: []commandContext{commandContextGlobal}, Run: runHomeCommand},
		{ID: "teams", Name: "teams", Aliases: []string{"team"}, Title: "teams", Description: "switch to teams mode", Contexts: []commandContext{commandContextGlobal}, Run: runTeamsCommand},
		{ID: "quit", Name: "quit", Aliases: []string{"q", "exit"}, Title: "quit", Description: "quit SSHThing", Contexts: []commandContext{commandContextGlobal}, Run: runQuitCommand},

		{ID: "add", Name: "add", Aliases: []string{"new"}, Title: "add", Description: "add a host", Contexts: []commandContext{commandContextHome, commandContextTeams}, Run: runAddCommand},
		{ID: "edit", Name: "edit", Aliases: []string{"rename"}, Title: "edit", Description: "edit selected item", Contexts: []commandContext{commandContextHome, commandContextTeams}, Run: runEditCommand},
		{ID: "delete", Name: "delete", Aliases: []string{"del", "remove"}, Title: "delete", Description: "delete selected item", Contexts: []commandContext{commandContextHome, commandContextTeams, commandContextTokens}, Danger: true, Run: runDeleteCommand},
		{ID: "group", Name: "group", Aliases: []string{"newgroup", "addgroup"}, Title: "group", Description: "create a group", Contexts: []commandContext{commandContextHome}, Run: runGroupCommand},
		{ID: "health", Name: "health", Aliases: []string{"check", "status"}, Title: "health", Description: "refresh host health", Contexts: []commandContext{commandContextHome, commandContextTeams}, Run: runHealthCommand},
		{ID: "sync", Name: "sync", Aliases: []string{"syncnow"}, Title: "sync", Description: "sync personal hosts", Contexts: []commandContext{commandContextHome}, Run: runSyncCommand, Enabled: syncCommandEnabled},
		{ID: "sftp", Name: "sftp", Aliases: []string{"ftp"}, Title: "sftp", Description: "connect via SFTP", Contexts: []commandContext{commandContextHome}, Run: runSFTPCommand, Enabled: selectedHostEnabled},
		{ID: "mount", Name: "mount", Aliases: []string{"mnt", "unmount"}, Title: "mount", Description: "mount or unmount selected host", Contexts: []commandContext{commandContextHome}, Run: runMountCommand, Enabled: mountCommandEnabled},
		{ID: "tokens", Name: "tokens", Aliases: []string{"automation"}, Title: "tokens", Description: "manage automation tokens", Contexts: []commandContext{commandContextHome, commandContextTeams, commandContextSettings}, Run: runTokensCommand},

		{ID: "refresh", Name: "refresh", Aliases: []string{"reload"}, Title: "refresh", Description: "refresh team data", Contexts: []commandContext{commandContextTeams}, Run: runTeamsRefreshCommand},
		{ID: "create", Name: "create", Aliases: []string{"create-team", "new-team"}, Title: "create", Description: "create a team", Contexts: []commandContext{commandContextTeams}, Run: runCreateTeamCommand},
		{ID: "import", Name: "import", Aliases: []string{"import-host"}, Title: "import", Description: "import a personal host", Contexts: []commandContext{commandContextTeams}, Run: runImportCommand},

		{ID: "signin", Name: "signin", Aliases: []string{"login"}, Title: "signin", Description: "sign in to cloud", Contexts: []commandContext{commandContextProfile}, Run: runSigninCommand},
		{ID: "signout", Name: "signout", Aliases: []string{"logout"}, Title: "signout", Description: "sign out of cloud", Contexts: []commandContext{commandContextProfile}, Run: runSignoutCommand},

		{ID: "save", Name: "save", Aliases: []string{"write"}, Title: "save", Description: "save settings", Contexts: []commandContext{commandContextSettings}, Run: runSettingsSaveCommand},
		{ID: "create-token", Name: "create", Aliases: []string{"new", "add"}, Title: "create", Description: "create automation token", Contexts: []commandContext{commandContextTokens}, Run: runTokenCreateCommand},
		{ID: "revoke-token", Name: "revoke", Aliases: []string{"disable"}, Title: "revoke", Description: "revoke selected token", Contexts: []commandContext{commandContextTokens}, Danger: true, Run: runTokenRevokeCommand},
	}
}

func selectedHostEnabled(m Model) (bool, string) {
	if _, ok := m.selectedHost(); !ok {
		return false, "select a host first"
	}
	return true, ""
}

func mountCommandEnabled(m Model) (bool, string) {
	if runtime.GOOS == "windows" {
		return false, "mount is not available on Windows"
	}
	if !m.cfg.Mount.Enabled {
		return false, "mounts are disabled in settings"
	}
	return selectedHostEnabled(m)
}

func syncCommandEnabled(m Model) (bool, string) {
	if m.syncing {
		return false, "sync already in progress"
	}
	if m.syncManager == nil {
		return false, "sync manager is nil"
	}
	if !m.syncManager.IsEnabled() {
		return false, "sync is disabled"
	}
	return true, ""
}

func runHelpCommand(m Model) (tea.Model, tea.Cmd) {
	m.overlay = OverlayHelp
	return m, nil
}

func runSearchCommand(m Model) (tea.Model, tea.Cmd) {
	if m.page == PageSettings {
		m.settingsSearching = true
		m.settingsFilter = ""
		return m, nil
	}
	m.teamsImportMode = false
	m.overlay = OverlaySearch
	m.searchQuery = ""
	m.spotlightItems = m.buildSpotlightItems("")
	m.selectedIdx = 0
	return m, nil
}

func runSettingsCommand(m Model) (tea.Model, tea.Cmd) {
	m.enterPage(PageSettings)
	m.err = nil
	return m, nil
}

func runProfileCommand(m Model) (tea.Model, tea.Cmd) {
	m.enterPage(PageProfile)
	m.err = nil
	return m, nil
}

func runHomeCommand(m Model) (tea.Model, tea.Cmd) {
	m.appMode = appModePersonal
	m.syncModeAppearance()
	m.enterPage(PageHome)
	m.err = fmt.Errorf("\u2713 Personal mode")
	return m, nil
}

func runTeamsCommand(m Model) (tea.Model, tea.Cmd) {
	if m.appMode == appModeTeams {
		m.err = fmt.Errorf("\u2139 already in Teams mode")
		return m, nil
	}
	m.toggleAppMode()
	return m, m.toggleAppModeCmd()
}

func runQuitCommand(m Model) (tea.Model, tea.Cmd) {
	return m.requestQuit()
}

func runAddCommand(m Model) (tea.Model, tea.Cmd) {
	if m.appMode == appModeTeams {
		groupPrefill := ""
		if host, ok := m.teamsCurrentHost(); ok {
			groupPrefill = host.Group
		}
		m.initAddHostForm("", groupPrefill, "", "", "", "22", "", "")
		m.overlay = OverlayAddHost
		m.formEditIdx = -1
		return m, nil
	}
	if item, ok := m.selectedListItem(); ok && item.Kind == ListItemNewGroup {
		return runGroupCommand(m)
	}
	groupPrefill := ""
	if g, ok := m.selectedGroup(); ok {
		groupPrefill = g
	}
	m.initAddHostForm("", groupPrefill, "", "", "", "22", "", "")
	m.overlay = OverlayAddHost
	m.formEditIdx = -1
	return m, nil
}

func runEditCommand(m Model) (tea.Model, tea.Cmd) {
	if m.appMode == appModeTeams {
		if err := m.openCurrentTeamHostEditor(context.Background()); err != nil {
			m.err = err
		}
		return m, nil
	}
	return m.openPersonalEditFlow()
}

func runDeleteCommand(m Model) (tea.Model, tea.Cmd) {
	if m.page == PageTokens {
		return runTokenDeleteCommand(m)
	}
	if m.appMode == appModeTeams {
		if _, ok := m.teamsCurrentHost(); !ok {
			m.err = fmt.Errorf("no team host selected")
			return m, nil
		}
		m.deleteCursor = 1
		m.overlay = OverlayDeleteHost
		return m, nil
	}
	return m.openPersonalDeleteFlow()
}

func runGroupCommand(m Model) (tea.Model, tea.Cmd) {
	m.groupInputValue = ""
	m.groupInputCursor = 0
	m.groupFocus = 0
	m.overlay = OverlayCreateGroup
	return m, nil
}

func runHealthCommand(m Model) (tea.Model, tea.Cmd) {
	if m.appMode == appModeTeams || m.page == PageTeams {
		return m, m.beginTeamHealthRefreshWithOptions(healthRefreshOptions{Source: "manual_team"})
	}
	return m, m.beginPersonalHealthRefresh()
}

func runSyncCommand(m Model) (tea.Model, tea.Cmd) {
	if ok, reason := syncCommandEnabled(m); !ok {
		m.err = fmt.Errorf("\u26A0 %s", reason)
		return m, nil
	}
	m.syncing = true
	m.syncRunID++
	m.syncAnimFrame = 0
	m.syncProgress = 0.02
	runID := m.syncRunID
	m.err = fmt.Errorf("\u2139 Syncing...")
	return m, tea.Batch(runSyncCmd(runID, m.syncManager), syncAnimTickCmd(runID))
}

func runSFTPCommand(m Model) (tea.Model, tea.Cmd) {
	host, ok := m.selectedHost()
	if !ok {
		m.err = fmt.Errorf("select a host first")
		return m, nil
	}
	return m.connectToHostSFTP(host)
}

func runMountCommand(m Model) (tea.Model, tea.Cmd) {
	m.armedSFTP = false
	m.armedMount = false
	m.armedUnmount = false
	host, ok := m.selectedHost()
	if !ok {
		m.err = fmt.Errorf("select a host first")
		return m, nil
	}
	if m.mountManager != nil {
		if mounted, _ := m.mountManager.IsMounted(host.ID); mounted {
			m.armedUnmount = true
		}
	}
	return m.handleMountEnter(host)
}

func runTokensCommand(m Model) (tea.Model, tea.Cmd) {
	m.enterPage(PageTokens)
	return m, nil
}

func runTeamsRefreshCommand(m Model) (tea.Model, tea.Cmd) {
	if err := m.loadTeamsData(context.Background()); err != nil {
		m.err = err
	} else {
		m.err = fmt.Errorf("\u2713 Teams refreshed")
	}
	return m, nil
}

func runCreateTeamCommand(m Model) (tea.Model, tea.Cmd) {
	m.openTeamsCreateFlow()
	return m, nil
}

func runImportCommand(m Model) (tea.Model, tea.Cmd) {
	m.teamsImportMode = true
	m.overlay = OverlaySearch
	m.searchQuery = ""
	m.spotlightItems = m.buildSpotlightItems("")
	m.selectedIdx = 0
	return m, nil
}

func runSigninCommand(m Model) (tea.Model, tea.Cmd) {
	if m.profileState == profileStateSignedIn {
		m.err = fmt.Errorf("\u2139 already signed in")
		return m, nil
	}
	return m, m.startProfileSignIn(context.Background())
}

func runSignoutCommand(m Model) (tea.Model, tea.Cmd) {
	if m.profileState != profileStateSignedIn {
		m.err = fmt.Errorf("\u2139 not signed in")
		return m, nil
	}
	m.signOutProfile(context.Background())
	return m, nil
}

func runSettingsSaveCommand(m Model) (tea.Model, tea.Cmd) {
	return m.leaveSettings()
}

func runTokenCreateCommand(m Model) (tea.Model, tea.Cmd) {
	m.tokenMode = tokenModeCreateName
	m.tokenHostPick = map[int]bool{}
	m.teamTokenHostPick = map[string]bool{}
	m.tokenHostIdx = 0
	m.tokenNameValue = ""
	m.err = fmt.Errorf("enter token name and press Enter")
	return m, nil
}

func runTokenRevokeCommand(m Model) (tea.Model, tea.Cmd) {
	return m.revokeSelectedToken()
}

func runTokenDeleteCommand(m Model) (tea.Model, tea.Cmd) {
	return m.deleteSelectedRevokedToken()
}
