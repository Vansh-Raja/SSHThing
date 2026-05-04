package app

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/Vansh-Raja/SSHThing/internal/config"
	"github.com/Vansh-Raja/SSHThing/internal/db"
	"github.com/Vansh-Raja/SSHThing/internal/ssh"
	syncpkg "github.com/Vansh-Raja/SSHThing/internal/sync"
	"github.com/Vansh-Raja/SSHThing/internal/teams"
	"github.com/Vansh-Raja/SSHThing/internal/ui"
	"github.com/Vansh-Raja/SSHThing/internal/unlock"
	tea "github.com/charmbracelet/bubbletea"
)

// ── Overlay key dispatch ──────────────────────────────────────────────

func (m Model) handleOverlayKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.overlay {
	case OverlayLogin:
		return m.handleLoginKeys(msg)
	case OverlaySetup:
		return m.handleSetupKeys(msg)
	case OverlayHelp:
		return m.handleHelpKeys(msg)
	case OverlaySearch:
		return m.handleSearchKeys(msg)
	case OverlayAddHost:
		return m.handleAddHostKeys(msg)
	case OverlayKeyEditor:
		return m.handlePrivateKeyEditorKeys(msg)
	case OverlayDeleteHost:
		return m.handleDeleteHostKeys(msg)
	case OverlayCreateGroup:
		return m.handleCreateGroupKeys(msg)
	case OverlayRenameGroup:
		return m.handleRenameGroupKeys(msg)
	case OverlayDeleteGroup:
		return m.handleDeleteGroupKeys(msg)
	case OverlayQuit:
		return m.handleQuitKeys(msg)
	case OverlayImportHost:
		return m.handleImportConflictKeys(msg)
	}
	return m, nil
}

// ── Page key dispatch ─────────────────────────────────────────────────

func (m Model) handlePageKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.commandModeActive() {
		return m.handleCommandLineKeys(msg)
	}
	if msg.String() == ":" && m.commandLineAllowed() {
		return m.openCommandLine()
	}
	switch m.page {
	case PageHome:
		return m.handleHomeKeys(msg)
	case PageProfile:
		return m.handleProfileKeys(msg)
	case PageSettings:
		return m.handleSettingsKeys(msg)
	case PageTokens:
		return m.handleTokensKeys(msg)
	case PageTeams:
		return m.handleTeamsKeys(msg)
	}
	return m, nil
}

func (m Model) commandLineAllowed() bool {
	if m.page == PageSettings {
		return !m.settingsSearching && !m.settingsEditing
	}
	if m.page == PageTokens {
		return m.tokenMode == tokenModeList && !m.tokenRevealOpen
	}
	return true
}

// ── Login overlay ─────────────────────────────────────────────────────

func (m Model) handleLoginKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		password := m.loginField.Value
		store, err := db.Init(password)
		if err != nil {
			msg := err.Error()
			if strings.Contains(strings.ToLower(msg), "invalid password for database") {
				m.loginError = "incorrect password"
			} else {
				m.loginError = msg
			}
			m.loginField.SetValue("")
			return m, nil
		}

		m.store = store
		m.masterPassword = password
		m.loadHosts()
		m.restoreMountsFromDB()

		if m.store != nil {
			syncMgr, err := syncpkg.NewManagerWithOptions(&m.cfg, m.store, password, m.syncManagerOptions())
			if err == nil {
				m.syncManager = syncMgr
			}
		}

		m.overlay = OverlayNone
		m.page = PageHome
		m.teamsHealthAutoRefreshed = map[string]bool{}
		_ = unlock.Save(password, time.Duration(m.cfg.Automation.SessionTTLSeconds)*time.Second)
		return m, m.beginPersonalHealthRefreshWithOptions(healthRefreshOptions{SilentIfEmpty: true, Source: "login"})

	case tea.KeyEsc:
		return m, tea.Quit
	}

	if msg.Type == tea.KeyBackspace {
		m.loginField.DeleteBack()
	} else if msg.Type == tea.KeyLeft {
		m.loginField.MoveLeft()
	} else if msg.Type == tea.KeyRight {
		m.loginField.MoveRight()
	} else {
		for _, r := range msg.Runes {
			m.loginField.InsertRune(r)
		}
	}
	m.loginError = ""
	return m, nil
}

// ── Setup overlay ─────────────────────────────────────────────────────

func (m Model) handleSetupKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyTab, tea.KeyDown:
		m.loginError = ""
		m.setupFocus = (m.setupFocus + 1) % 3
		return m, nil

	case tea.KeyShiftTab, tea.KeyUp:
		m.loginError = ""
		m.setupFocus = (m.setupFocus + 2) % 3
		return m, nil

	case tea.KeyEnter:
		if m.setupFocus == 2 {
			// Submit
			password := m.setupFields[0].Value
			confirm := m.setupFields[1].Value

			if len(password) < 8 {
				m.loginError = "password must be at least 8 characters"
				return m, nil
			}

			if password != confirm {
				m.loginError = "passwords do not match"
				m.setupFields[1].SetValue("")
				return m, nil
			}

			store, err := db.Init(password)
			if err != nil {
				m.loginError = err.Error()
				return m, nil
			}

			m.store = store
			m.masterPassword = password
			m.loadHosts()
			m.restoreMountsFromDB()

			if m.store != nil {
				syncMgr, err := syncpkg.NewManagerWithOptions(&m.cfg, m.store, password, m.syncManagerOptions())
				if err == nil {
					m.syncManager = syncMgr
				}
			}

			m.overlay = OverlayNone
			m.page = PageHome
			m.teamsHealthAutoRefreshed = map[string]bool{}
			return m, m.beginPersonalHealthRefreshWithOptions(healthRefreshOptions{SilentIfEmpty: true, Source: "setup"})
		}

	case tea.KeyEsc:
		return m, tea.Quit
	}

	// Forward to focused field
	if m.setupFocus < 2 {
		f := &m.setupFields[m.setupFocus]
		if msg.Type == tea.KeyBackspace {
			f.DeleteBack()
		} else if msg.Type == tea.KeyLeft {
			f.MoveLeft()
		} else if msg.Type == tea.KeyRight {
			f.MoveRight()
		} else {
			for _, r := range msg.Runes {
				f.InsertRune(r)
			}
		}
		m.loginError = ""
	}
	return m, nil
}

// ── Help overlay ──────────────────────────────────────────────────────

func (m Model) handleHelpKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.overlay = OverlayNone
	return m, nil
}

// ── Search overlay ────────────────────────────────────────────────────

func (m Model) handleSearchKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		if m.teamsImportMode {
			m.overlay = OverlayAddHost
		} else {
			m.overlay = OverlayNone
		}
		m.searchQuery = ""
		m.spotlightItems = nil
		m.armedSFTP = false
		m.armedMount = false
		m.armedUnmount = false
		m.teamsImportMode = false
		return m, nil

	case "S":
		if m.appMode == appModeTeams {
			return m, nil
		}
		item, ok := m.selectedSpotlightItem()
		if !ok || item.Kind != SpotlightItemHost {
			m.err = fmt.Errorf("select a host first")
			return m, nil
		}
		m.armedSFTP = !m.armedSFTP
		m.armedMount = false
		m.armedUnmount = false
		if m.armedSFTP {
			m.err = fmt.Errorf("SFTP armed \u2014 press Enter")
		} else {
			m.err = nil
		}
		return m, nil

	case "M":
		if m.appMode == appModeTeams {
			return m, nil
		}
		if !m.cfg.Mount.Enabled {
			m.err = fmt.Errorf("\u26A0 mounts are disabled in settings")
			return m, nil
		}
		if runtime.GOOS == "windows" {
			m.err = fmt.Errorf("\u26A0 mount is not available on Windows")
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
			m.err = fmt.Errorf("Unmount armed \u2014 press Enter")
		} else {
			m.armedMount = true
			m.err = fmt.Errorf("Mount (beta) armed \u2014 press Enter")
		}
		return m, nil

	case "enter":
		item, ok := m.selectedSpotlightItem()
		if !ok {
			return m, nil
		}
		if m.teamsImportMode {
			if item.Kind != SpotlightItemHost {
				return m, nil
			}
			return m.importPersonalHostToCurrentTeam(item.Host)
		}
		if item.Kind == SpotlightItemCommand {
			m.overlay = OverlayNone
			m.searchQuery = ""
			m.spotlightItems = nil
			switch item.Command {
			case "create_team":
				m.openTeamsCreateFlow()
			case "open_settings":
				m.enterPage(PageSettings)
			case "open_profile":
				m.enterPage(PageProfile)
			case "switch_team":
				if err := m.switchTeamByID(context.Background(), item.Team.ID); err != nil {
					m.err = err
				} else {
					m.err = fmt.Errorf("✓ %s", item.Team.Name)
				}
			}
			return m, nil
		}
		if item.Kind == SpotlightItemGroup {
			m.overlay = OverlayNone
			m.searchQuery = ""
			m.spotlightItems = nil
			m.selectGroupInList(item.GroupName)
			return m, nil
		}
		if m.appMode == appModeTeams {
			m.overlay = OverlayNone
			m.searchQuery = ""
			m.spotlightItems = nil
			for idx, host := range m.teamsItems {
				if host.ID == item.TeamHost.ID {
					m.teamsCursor = idx
					break
				}
			}
			return m, nil
		}
		host := item.Host
		m.overlay = OverlayNone
		m.searchQuery = ""
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
			break
		}
		if m.selectedIdx > 0 {
			m.selectedIdx--
		}
		return m, nil

	case "down", "j":
		if msg.String() == "j" && !m.cfg.UI.VimMode {
			break
		}
		if m.selectedIdx < len(m.spotlightItems)-1 {
			m.selectedIdx++
		}
		return m, nil
	}

	// Forward key to search input
	if msg.Type == tea.KeyBackspace {
		m.searchQuery = removeLastRune(m.searchQuery)
	} else {
		for _, r := range msg.Runes {
			m.searchQuery += string(r)
		}
	}
	m.spotlightItems = m.buildSpotlightItems(m.searchQuery)
	m.selectedIdx = 0
	return m, nil
}

func removeLastRune(s string) string {
	if s == "" {
		return ""
	}
	runes := []rune(s)
	return string(runes[:len(runes)-1])
}

// ── Quit overlay ──────────────────────────────────────────────────────

func (m Model) handleQuitKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.overlay = OverlayNone
		m.quitCursor = 0
		return m, nil

	case "left", "h", "shift+tab":
		if m.quitCursor > 0 {
			m.quitCursor--
		}
		return m, nil

	case "right", "l", "tab":
		hasMounts := m.mountManager != nil && len(m.mountManager.ListActive()) > 0
		maxBtn := 1 // no-mounts: yes(0), cancel(1)
		if hasMounts {
			maxBtn = 2 // mounts: unmount(0), leave(1), cancel(2)
		}
		if m.quitCursor < maxBtn {
			m.quitCursor++
		}
		return m, nil

	case "enter":
		hasMounts := m.mountManager != nil && len(m.mountManager.ListActive()) > 0
		switch m.quitCursor {
		case 0: // "unmount & quit" or "yes"
			if hasMounts {
				m.err = fmt.Errorf("Unmounting mounts\u2026")
				return m, func() tea.Msg {
					m.mountManager.UnmountAll()
					if m.store != nil {
						_ = m.store.DeleteAllMountStates()
					}
					return quitFinishedMsg{}
				}
			}
			return m, tea.Quit
		case 1: // "leave mounted" or "cancel"
			if hasMounts {
				if m.store != nil && m.mountManager != nil {
					for _, mt := range m.mountManager.ListActive() {
						_ = m.store.UpsertMountState(mt.HostID, mt.LocalPath, mt.RemotePath)
					}
				}
				return m, tea.Quit
			}
			m.overlay = OverlayNone
			m.quitCursor = 0
			return m, nil
		default: // "cancel" (only reachable with mounts)
			m.overlay = OverlayNone
			m.quitCursor = 0
			return m, nil
		}
	}
	return m, nil
}

// ── Add/Edit host overlay ─────────────────────────────────────────────

func (m Model) handleAddHostKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	isTextField := func(f int) bool {
		switch f {
		case ui.FFLabel, ui.FFTags, ui.FFHostname, ui.FFPort, ui.FFUsername, ui.FFAuthDet:
			return true
		}
		return false
	}

	submitAndClose := func() (tea.Model, tea.Cmd) {
		if m.formAuthIdx == 1 && m.formSecretRevealed {
			m.syncFormKeyFieldFromEditor()
		}
		validationErr := m.validateForm()
		if validationErr != nil {
			if m.formAuthIdx == 1 {
				m.formSecretRevealed = true
				m.formEditing = true
			}
			m.err = validationErr
			return m, nil
		}

		var portInt int
		fmt.Sscanf(m.formFields[ui.FFPort].Value, "%d", &portInt)

		var keyType, plainKey string
		switch m.formAuthIdx {
		case 0: // password
			keyType = "password"
			plainKey = m.formFields[ui.FFAuthDet].Value
		case 1: // paste key
			keyType = "pasted"
			plainKey = normalizePrivateKey(m.formFields[ui.FFAuthDet].Value)
		case 2: // generate
			keyType = m.formKeyTypes[m.formKeyIdx]
			comment := fmt.Sprintf("%s@%s", m.formFields[ui.FFUsername].Value, m.formFields[ui.FFHostname].Value)
			privateKey, _, err := ssh.GenerateKey(ssh.KeyType(keyType), comment)
			if err != nil {
				m.err = fmt.Errorf("failed to generate key: %v", err)
				return m, nil
			}
			plainKey = normalizePrivateKey(privateKey)
		}

		groupName := m.modalSelectedGroupName()
		tags := db.ParseTagInput(m.formFields[ui.FFTags].Value)
		if m.appMode != appModeTeams && groupName != "" {
			if err := m.store.UpsertGroup(groupName); err != nil {
				m.err = err
				return m, nil
			}
		}

		if m.appMode == appModeTeams {
			team, ok := m.teamsCurrentTeam()
			if !ok {
				m.err = fmt.Errorf("no team selected")
				return m, nil
			}
			accessToken, err := m.teamsAccessToken(context.Background())
			if err != nil {
				m.err = err
				return m, nil
			}

			credentialType := "none"
			sharedCredential := ""
			switch m.formAuthIdx {
			case 0:
				credentialType = "password"
				sharedCredential = plainKey
			case 1, 2:
				credentialType = "private_key"
				sharedCredential = plainKey
			}

			if m.formTeamHostID != "" {
				req := teams.UpdateTeamHostRequest{
					Label:            strings.TrimSpace(m.formFields[ui.FFLabel].Value),
					Hostname:         m.formFields[ui.FFHostname].Value,
					Username:         m.formFields[ui.FFUsername].Value,
					Port:             portInt,
					Group:            groupName,
					Tags:             tags,
					CredentialMode:   m.formTeamCredentialMode,
					CredentialType:   m.formTeamCredentialType,
					SecretVisibility: "revealed_to_access_holders",
				}
				if m.formTeamCredentialMode != "per_member" {
					req.CredentialType = credentialType
					req.CredentialMode = "shared"
					req.SharedCredential = sharedCredential
				}
				err = m.teamsClient.UpdateTeamHost(context.Background(), accessToken, m.formTeamHostID, req)
			} else {
				_, err = m.teamsClient.CreateTeamHost(context.Background(), accessToken, team.ID, teams.CreateTeamHostRequest{
					Label:            strings.TrimSpace(m.formFields[ui.FFLabel].Value),
					Hostname:         m.formFields[ui.FFHostname].Value,
					Username:         m.formFields[ui.FFUsername].Value,
					Port:             portInt,
					Group:            groupName,
					Tags:             tags,
					CredentialMode:   "shared",
					CredentialType:   credentialType,
					SecretVisibility: "revealed_to_access_holders",
					SharedCredential: sharedCredential,
				})
			}
			if err != nil {
				m.err = err
				return m, nil
			}
			label := strings.TrimSpace(m.formFields[ui.FFLabel].Value)
			if label == "" {
				label = m.formFields[ui.FFHostname].Value
			}
			if err := m.loadCurrentTeamHosts(context.Background()); err != nil {
				m.err = err
				return m, nil
			}
			if m.formTeamHostID != "" {
				m.err = fmt.Errorf("\u2713 Team host '%s' updated", label)
			} else {
				m.err = fmt.Errorf("\u2713 Team host '%s' added", label)
			}
			m.closeAddHostOverlay()
			return m, nil
		}

		if m.formEditIdx < 0 {
			// Add new
			host := &db.HostModel{
				Label:     strings.TrimSpace(m.formFields[ui.FFLabel].Value),
				GroupName: groupName,
				Tags:      tags,
				Hostname:  m.formFields[ui.FFHostname].Value,
				Username:  m.formFields[ui.FFUsername].Value,
				Port:      portInt,
				KeyType:   keyType,
			}
			if err := m.store.CreateHost(host, plainKey); err != nil {
				m.err = err
				return m, nil
			}
			m.err = fmt.Errorf("\u2713 Host '%s' added", host.Hostname)
		} else {
			// Edit
			selectedHost, ok := m.selectedHost()
			if ok {
				keepExistingSecret := m.formAuthIdx == 0 &&
					selectedHost.KeyType == "password" &&
					plainKey == ""
				host := &db.HostModel{
					ID:        selectedHost.ID,
					Label:     strings.TrimSpace(m.formFields[ui.FFLabel].Value),
					GroupName: groupName,
					Tags:      tags,
					Hostname:  m.formFields[ui.FFHostname].Value,
					Username:  m.formFields[ui.FFUsername].Value,
					Port:      portInt,
					KeyType:   keyType,
				}
				if keepExistingSecret {
					if err := m.store.UpdateHost(host); err != nil {
						m.err = err
						return m, nil
					}
				} else if m.formAuthIdx == 0 || plainKey != "" {
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
				m.err = fmt.Errorf("\u2713 Host '%s' updated", host.Hostname)
			}
		}

		m.loadHosts()
		m.loadGroups()
		m.rebuildListItems()
		m.closeAddHostOverlay()
		return m, nil
	}

	// Shift+Enter / Ctrl+S always submits
	if s := msg.String(); s == "shift+enter" || s == "shift+return" || s == "ctrl+s" {
		return submitAndClose()
	}

	if privateKeyFocused := m.formAuthIdx == 1 && m.formFocus == ui.FFAuthDet; privateKeyFocused && !m.formEditing {
		switch {
		case msg.Type == tea.KeyEnter || msg.String() == "v":
			cmd := m.openPrivateKeyEditor(false)
			m.ensureFormFocusVisible()
			return m, cmd
		case msg.String() == "c":
			secret := strings.TrimSpace(m.formFields[ui.FFAuthDet].Value)
			if secret == "" {
				m.err = fmt.Errorf("no private key to copy")
				return m, nil
			}
			if err := copyTokenToClipboard(secret); err != nil {
				m.err = fmt.Errorf("failed to copy private key: %v", err)
			} else {
				m.err = fmt.Errorf("\u2713 private key copied to clipboard")
			}
			return m, nil
		case msg.Type == tea.KeyBackspace:
			return m, nil
		}
	}

	cycleGroup := func(dir int) {
		if len(m.formGroups) == 0 {
			return
		}
		n := len(m.formGroups)
		m.formGroupIdx = (m.formGroupIdx + dir + n) % n
	}

	cycleAuth := func(dir int) {
		if m.appMode == appModeTeams && m.formTeamCredentialMode == "per_member" {
			return
		}
		m.formAuthIdx = (m.formAuthIdx + dir + len(m.formAuthOpts)) % len(m.formAuthOpts)
		m.formSecretRevealed = false
	}

	formOrder := []int{ui.FFLabel, ui.FFGroup, ui.FFTags, ui.FFHostname, ui.FFPort, ui.FFUsername, ui.FFAuthMeth, ui.FFAuthDet, ui.FFSave}
	leftColumn := []int{ui.FFLabel, ui.FFGroup, ui.FFTags, ui.FFHostname, ui.FFPort}
	rightColumn := []int{ui.FFUsername, ui.FFAuthMeth, ui.FFAuthDet, ui.FFSave}

	findIdx := func(f int) int {
		for i, v := range formOrder {
			if v == f {
				return i
			}
		}
		return 0
	}

	inList := func(fields []int, focus int) bool {
		for _, field := range fields {
			if field == focus {
				return true
			}
		}
		return false
	}

	moveHorizontal := func(dir int) {
		if dir > 0 && inList(leftColumn, m.formFocus) {
			switch m.formFocus {
			case ui.FFLabel:
				m.formFocus = ui.FFUsername
			case ui.FFGroup:
				m.formFocus = ui.FFAuthMeth
			case ui.FFTags, ui.FFHostname:
				m.formFocus = ui.FFAuthDet
			default:
				m.formFocus = ui.FFSave
			}
			return
		}
		if dir < 0 && inList(rightColumn, m.formFocus) {
			switch m.formFocus {
			case ui.FFUsername:
				m.formFocus = ui.FFLabel
			case ui.FFAuthMeth:
				m.formFocus = ui.FFGroup
			case ui.FFAuthDet:
				m.formFocus = ui.FFTags
			default:
				m.formFocus = ui.FFPort
			}
		}
	}

	switch msg.Type {
	case tea.KeyEsc:
		if m.formEditing && isTextField(m.formFocus) {
			m.formEditing = false
			return m, nil
		}
		if m.formEditing && (m.formFocus == ui.FFGroup || m.formFocus == ui.FFAuthMeth || (m.formFocus == ui.FFAuthDet && m.formAuthIdx == 2)) {
			m.formEditing = false
			return m, nil
		}
		m.closeAddHostOverlay()
		return m, nil

	case tea.KeyTab, tea.KeyDown:
		m.formEditing = false
		idx := findIdx(m.formFocus)
		idx = (idx + 1) % len(formOrder)
		m.formFocus = formOrder[idx]
		m.ensureFormFocusVisible()
		return m, nil

	case tea.KeyShiftTab, tea.KeyUp:
		m.formEditing = false
		idx := findIdx(m.formFocus)
		idx = (idx - 1 + len(formOrder)) % len(formOrder)
		m.formFocus = formOrder[idx]
		m.ensureFormFocusVisible()
		return m, nil

	case tea.KeyLeft:
		if m.formEditing && m.formFocus == ui.FFGroup {
			cycleGroup(-1)
		} else if m.formEditing && m.formFocus == ui.FFAuthMeth {
			cycleAuth(-1)
		} else if m.formEditing && isTextField(m.formFocus) {
			m.formFields[m.formFocus].MoveLeft()
		} else if m.formEditing && m.formFocus == ui.FFAuthDet && m.formAuthIdx == 2 {
			m.formKeyIdx = (m.formKeyIdx - 1 + len(m.formKeyTypes)) % len(m.formKeyTypes)
		} else if !m.formEditing {
			moveHorizontal(-1)
			m.ensureFormFocusVisible()
		}
		return m, nil

	case tea.KeyRight:
		if m.formEditing && m.formFocus == ui.FFGroup {
			cycleGroup(1)
		} else if m.formEditing && m.formFocus == ui.FFAuthMeth {
			cycleAuth(1)
		} else if m.formEditing && isTextField(m.formFocus) {
			m.formFields[m.formFocus].MoveRight()
		} else if m.formEditing && m.formFocus == ui.FFAuthDet && m.formAuthIdx == 2 {
			m.formKeyIdx = (m.formKeyIdx + 1) % len(m.formKeyTypes)
		} else if !m.formEditing {
			moveHorizontal(1)
			m.ensureFormFocusVisible()
		}
		return m, nil

	case tea.KeyEnter:
		if m.formFocus == ui.FFSave {
			return submitAndClose()
		}
		if m.formFocus == ui.FFGroup || m.formFocus == ui.FFAuthMeth || (m.formFocus == ui.FFAuthDet && m.formAuthIdx == 2) {
			m.formEditing = !m.formEditing
			m.ensureFormFocusVisible()
			return m, nil
		}
		if isTextField(m.formFocus) {
			if m.appMode == appModeTeams && m.formTeamCredentialMode == "per_member" && m.formFocus == ui.FFAuthDet {
				m.err = fmt.Errorf("ℹ per-member credentials are not editable from Teams TUI yet")
				return m, nil
			}
			if m.formEditing {
				m.formEditing = false
			} else {
				m.formEditing = true
			}
			m.ensureFormFocusVisible()
			return m, nil
		}
		// For group/auth selectors, enter could submit
		return submitAndClose()

	case tea.KeyBackspace:
		if m.formEditing && isTextField(m.formFocus) {
			m.formFields[m.formFocus].DeleteBack()
		}
		m.ensureFormFocusVisible()
		return m, nil
	}

	// Rune input
	if m.formEditing && isTextField(m.formFocus) && len(msg.Runes) > 0 {
		for _, r := range msg.Runes {
			m.formFields[m.formFocus].InsertRune(r)
		}
		m.ensureFormFocusVisible()
		return m, nil
	}

	// Spacebar for key gen type cycling
	str := msg.String()
	if str == "v" && m.formAuthIdx == 1 && !m.formEditing {
		cmd := m.openPrivateKeyEditor(false)
		m.ensureFormFocusVisible()
		return m, cmd
	}
	if str == "c" && m.formAuthIdx == 1 && !m.formEditing {
		secret := strings.TrimSpace(m.formFields[ui.FFAuthDet].Value)
		if secret == "" {
			m.err = fmt.Errorf("no private key to copy")
			return m, nil
		}
		if err := copyTokenToClipboard(secret); err != nil {
			m.err = fmt.Errorf("failed to copy private key: %v", err)
		} else {
			m.err = fmt.Errorf("\u2713 private key copied to clipboard")
		}
		return m, nil
	}
	if str == " " && m.formEditing && m.formFocus == ui.FFAuthDet && m.formAuthIdx == 2 {
		m.formKeyIdx = (m.formKeyIdx + 1) % len(m.formKeyTypes)
		return m, nil
	}

	// Vim nav for selectors
	if m.cfg.UI.VimMode {
		if str == "h" && m.formEditing && m.formFocus == ui.FFGroup {
			cycleGroup(-1)
			return m, nil
		} else if str == "l" && m.formEditing && m.formFocus == ui.FFGroup {
			cycleGroup(1)
			return m, nil
		} else if str == "h" && m.formEditing && m.formFocus == ui.FFAuthMeth {
			cycleAuth(-1)
			return m, nil
		} else if str == "l" && m.formEditing && m.formFocus == ui.FFAuthMeth {
			cycleAuth(1)
			return m, nil
		}
	}

	if m.appMode == appModeTeams && !m.formEditing && m.formTeamHostID == "" && str == "I" {
		m.teamsImportMode = true
		m.overlay = OverlaySearch
		m.searchQuery = ""
		m.spotlightItems = m.buildSpotlightItems("")
		m.selectedIdx = 0
		m.err = fmt.Errorf("select a personal host to import into this team")
		return m, nil
	}

	return m, nil
}

func (m *Model) closeAddHostOverlay() {
	m.overlay = OverlayNone
	m.formFields = nil
	m.formEditIdx = -1
	m.formScrollOffset = 0
	m.formSecretRevealed = false
	m.formKeyEditorOriginal = ""
	m.formTeamHostID = ""
	m.formTeamCredentialMode = ""
	m.formTeamCredentialType = ""
	m.teamsImportMode = false
	m.teamsImportConflict = nil
}

func (m *Model) ensureFormFocusVisible() {
	available := m.height - 8
	if available < 8 {
		available = 8
	}

	lineForFocus := 0
	switch m.formFocus {
	case ui.FFLabel:
		lineForFocus = 0
	case ui.FFGroup:
		lineForFocus = 2
	case ui.FFTags:
		lineForFocus = 4
	case ui.FFHostname, ui.FFPort:
		lineForFocus = 7
	case ui.FFUsername:
		lineForFocus = 10
	case ui.FFAuthMeth:
		lineForFocus = 12
	case ui.FFAuthDet:
		lineForFocus = 14
		if m.formAuthIdx == 1 && m.formSecretRevealed {
			lineForFocus += 3
		}
	case ui.FFSave:
		lineForFocus = 18
	}

	if m.formScrollOffset > lineForFocus {
		m.formScrollOffset = lineForFocus
	}
	if lineForFocus >= m.formScrollOffset+available {
		m.formScrollOffset = lineForFocus - available + 1
	}
	if m.formScrollOffset < 0 {
		m.formScrollOffset = 0
	}
}

// ── Delete host overlay ───────────────────────────────────────────────

func (m Model) handleDeleteHostKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	doDelete := func() Model {
		if m.appMode == appModeTeams {
			host, ok := m.teamsCurrentHost()
			if !ok {
				m.overlay = OverlayNone
				return m
			}
			accessToken, err := m.teamsAccessToken(context.Background())
			if err != nil {
				m.err = err
				return m
			}
			if err := m.teamsClient.DeleteTeamHost(context.Background(), accessToken, host.ID); err != nil {
				m.err = err
				return m
			}
			m.err = fmt.Errorf("\u2713 Host '%s' deleted", host.Hostname)
			if err := m.loadCurrentTeamHosts(context.Background()); err != nil {
				m.err = err
				return m
			}
			if m.teamsCursor >= len(m.teamsItems) && len(m.teamsItems) > 0 {
				m.teamsCursor = len(m.teamsItems) - 1
			}
			m.overlay = OverlayNone
			return m
		}
		if host, ok := m.selectedHost(); ok {
			if err := m.store.DeleteHost(host.ID); err != nil {
				m.err = err
			} else {
				m.err = fmt.Errorf("\u2713 Host '%s' deleted", host.Hostname)
				m.loadHosts()
				m.loadGroups()
				m.rebuildListItems()
				if m.selectedIdx >= len(m.listItems) && len(m.listItems) > 0 {
					m.selectedIdx = len(m.listItems) - 1
				}
			}
		}
		m.overlay = OverlayNone
		return m
	}

	switch msg.String() {
	case "y", "Y":
		m = doDelete()
		return m, nil
	case "n", "N", "esc":
		m.overlay = OverlayNone
		return m, nil
	case "left", "h", "right", "l", "tab", "shift+tab":
		m.deleteCursor = 1 - m.deleteCursor
		return m, nil
	case "enter":
		if m.deleteCursor == 0 {
			m = doDelete()
		} else {
			m.overlay = OverlayNone
		}
		return m, nil
	}
	return m, nil
}

// ── Create group overlay ──────────────────────────────────────────────

func (m Model) handleCreateGroupKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	return m.handleGroupInputKeys(msg, false)
}

// ── Rename group overlay ──────────────────────────────────────────────

func (m Model) handleRenameGroupKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	return m.handleGroupInputKeys(msg, true)
}

func (m Model) handleGroupInputKeys(msg tea.KeyMsg, isRename bool) (tea.Model, tea.Cmd) {
	submit := func() (tea.Model, tea.Cmd) {
		name := strings.TrimSpace(m.groupInputValue)
		if name == "" {
			m.err = fmt.Errorf("group name cannot be empty")
			return m, nil
		}
		if strings.EqualFold(name, "Ungrouped") {
			m.err = fmt.Errorf("'Ungrouped' is reserved")
			return m, nil
		}
		if isRename {
			if err := m.store.RenameGroup(m.groupOldName, name); err != nil {
				m.err = err
				return m, nil
			}
			m.err = fmt.Errorf("\u2713 Group '%s' renamed", name)
		} else {
			if err := m.store.UpsertGroup(name); err != nil {
				m.err = err
				return m, nil
			}
			m.err = fmt.Errorf("\u2713 Group '%s' created", name)
		}
		m.loadHosts()
		m.loadGroups()
		m.rebuildListItems()
		m.selectGroupInList(name)
		m.overlay = OverlayNone
		m.groupInputValue = ""
		m.groupInputCursor = 0
		return m, nil
	}

	cancel := func() (tea.Model, tea.Cmd) {
		m.overlay = OverlayNone
		m.groupInputValue = ""
		m.groupInputCursor = 0
		m.groupFocus = 0
		m.err = nil
		return m, nil
	}

	switch msg.String() {
	case "esc":
		return cancel()
	case "tab":
		m.groupFocus = (m.groupFocus + 1) % 3
		return m, nil
	case "shift+tab":
		m.groupFocus = (m.groupFocus + 2) % 3
		return m, nil
	case "left", "h":
		if m.groupFocus == 1 {
			m.groupFocus = 2
		} else if m.groupFocus == 2 {
			m.groupFocus = 1
		} else if m.groupFocus == 0 {
			m.groupInputCursor = max(0, m.groupInputCursor-1)
		}
		return m, nil
	case "right", "l":
		if m.groupFocus == 1 {
			m.groupFocus = 2
		} else if m.groupFocus == 2 {
			m.groupFocus = 1
		} else if m.groupFocus == 0 {
			runes := []rune(m.groupInputValue)
			if m.groupInputCursor < len(runes) {
				m.groupInputCursor++
			}
		}
		return m, nil
	case "enter":
		if m.groupFocus == 2 {
			return cancel()
		}
		return submit()
	}

	// Text input
	if m.groupFocus == 0 {
		if msg.Type == tea.KeyBackspace {
			runes := []rune(m.groupInputValue)
			if m.groupInputCursor > 0 && m.groupInputCursor <= len(runes) {
				m.groupInputValue = string(runes[:m.groupInputCursor-1]) + string(runes[m.groupInputCursor:])
				m.groupInputCursor--
			}
		} else {
			for _, r := range msg.Runes {
				runes := []rune(m.groupInputValue)
				if m.groupInputCursor > len(runes) {
					m.groupInputCursor = len(runes)
				}
				m.groupInputValue = string(runes[:m.groupInputCursor]) + string(r) + string(runes[m.groupInputCursor:])
				m.groupInputCursor++
			}
		}
	}
	return m, nil
}

// ── Delete group overlay ──────────────────────────────────────────────

func (m Model) handleDeleteGroupKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "n", "N":
		m.overlay = OverlayNone
		m.groupDeleteCursor = 1
		return m, nil
	case "left", "right", "h", "l", "tab", "shift+tab":
		m.groupDeleteCursor = 1 - m.groupDeleteCursor
		return m, nil
	case "y", "Y":
		m.groupDeleteCursor = 0
	}

	if msg.String() == "enter" && m.groupDeleteCursor == 0 {
		if err := m.store.DeleteGroup(m.groupOldName); err != nil {
			m.err = err
			return m, nil
		}
		m.err = fmt.Errorf("\u2713 Group '%s' deleted", m.groupOldName)
		m.loadHosts()
		m.loadGroups()
		m.rebuildListItems()
		m.overlay = OverlayNone
		m.groupDeleteCursor = 1
		return m, nil
	}

	if msg.String() == "enter" {
		m.overlay = OverlayNone
		m.groupDeleteCursor = 1
	}

	return m, nil
}

// ── Home page ─────────────────────────────────────────────────────────

func (m Model) handleHomeKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	m.rebuildListItems()

	switch key {
	case "q", "Q":
		return m.requestQuit()

	case "shift+tab":
		m.enterPage(m.nextVisiblePage(m.page))
		return m, nil

	case "R":
		return m, m.beginPersonalHealthRefresh()

	case "S":
		m.armedSFTP = !m.armedSFTP
		m.armedMount = false
		m.armedUnmount = false
		if m.armedSFTP {
			m.err = fmt.Errorf("SFTP armed \u2014 press Enter")
		} else {
			m.err = nil
		}
		return m, nil

	case "M":
		if !m.cfg.Mount.Enabled {
			m.err = fmt.Errorf("\u26A0 mounts are disabled in settings")
			return m, nil
		}
		if runtime.GOOS == "windows" {
			m.err = fmt.Errorf("\u26A0 mount is not available on Windows")
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
		if m.armedMount || m.armedUnmount {
			m.armedMount = false
			m.armedUnmount = false
			m.err = nil
			return m, nil
		}
		m.armedSFTP = false
		if isMounted {
			m.armedUnmount = true
			m.err = fmt.Errorf("Unmount armed \u2014 press Enter")
		} else {
			m.armedMount = true
			m.err = fmt.Errorf("Mount (beta) armed \u2014 press Enter")
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

	case "ctrl+u":
		m.selectedIdx = max(0, m.selectedIdx-10)

	case "ctrl+d":
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
		m.overlay = OverlaySearch
		m.searchQuery = ""
		m.spotlightItems = m.buildSpotlightItems("")
		m.selectedIdx = 0

	case "a", "ctrl+n":
		return runAddCommand(m)

	case "e":
		return m.openPersonalEditFlow()

	case "d", "delete":
		return m.openPersonalDeleteFlow()

	case "ctrl+g":
		return runGroupCommand(m)

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
			m.groupInputValue = ""
			m.groupInputCursor = 0
			m.groupFocus = 0
			m.overlay = OverlayCreateGroup
			return m, nil
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
		m.overlay = OverlayHelp

	case ",":
		m.enterPage(PageSettings)
		m.err = nil

	case "Y":
		return runSyncCommand(m)
	}

	return m, nil
}

func (m Model) openPersonalEditFlow() (tea.Model, tea.Cmd) {
	if item, ok := m.selectedListItem(); ok && item.Kind == ListItemGroup {
		if item.GroupName == "Ungrouped" {
			m.err = fmt.Errorf("cannot rename Ungrouped")
			return m, nil
		}
		m.groupOldName = item.GroupName
		m.groupInputValue = item.GroupName
		m.groupInputCursor = len([]rune(item.GroupName))
		m.groupFocus = 0
		m.overlay = OverlayRenameGroup
		return m, nil
	}
	host, ok := m.selectedHost()
	if !ok {
		m.err = fmt.Errorf("select a host or group first")
		return m, nil
	}
	var existingKey string
	if m.store != nil && host.HasKey && host.KeyType != "password" {
		key, err := m.store.GetHostSecret(host.ID)
		if err == nil {
			existingKey = key
		}
	}
	tagInput := strings.Join(host.Tags, ", ")
	m.initAddHostForm(host.Label, host.GroupName, tagInput, host.Hostname, host.Username, fmt.Sprintf("%d", host.Port), host.KeyType, existingKey)
	m.formEditIdx = m.selectedIdx
	m.overlay = OverlayAddHost
	return m, nil
}

func (m Model) openPersonalDeleteFlow() (tea.Model, tea.Cmd) {
	if item, ok := m.selectedListItem(); ok && item.Kind == ListItemGroup {
		if item.GroupName == "Ungrouped" {
			m.err = fmt.Errorf("cannot delete Ungrouped")
			return m, nil
		}
		m.groupOldName = item.GroupName
		m.groupDeleteCursor = 1
		m.overlay = OverlayDeleteGroup
		return m, nil
	}
	if _, ok := m.selectedHost(); ok {
		m.deleteCursor = 1
		m.overlay = OverlayDeleteHost
		return m, nil
	}
	m.err = fmt.Errorf("select a host or group first")
	return m, nil
}

// ── Settings page ─────────────────────────────────────────────────────

func (m Model) handleSettingsKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Settings filter mode
	if m.settingsSearching {
		switch key {
		case "esc":
			m.settingsSearching = false
			m.settingsFilter = ""
			m.settingsCursor = 0
			return m, nil
		case "enter":
			m.settingsSearching = false
			// Snap cursor to first filtered item
			f := m.filteredSettingsIdxs()
			if len(f) > 0 {
				m.settingsCursor = f[0]
			}
			return m, nil
		}
		if msg.Type == tea.KeyBackspace {
			m.settingsFilter = removeLastRune(m.settingsFilter)
		} else {
			for _, r := range msg.Runes {
				m.settingsFilter += string(r)
			}
		}
		// Snap cursor to first filtered item as filter changes
		f := m.filteredSettingsIdxs()
		if len(f) > 0 {
			m.settingsCursor = f[0]
		}
		return m, nil
	}

	// Settings edit mode (for text-editable fields)
	if m.settingsEditing {
		switch key {
		case "esc":
			m.settingsEditing = false
			m.settingsEditVal = ""
			m.err = nil
			return m, nil
		case "enter":
			if !m.applySettingsEditValue(m.settingsCursor, m.settingsEditVal) {
				return m, nil
			}
			m.settingsEditing = false
			m.settingsEditVal = ""
			m.settingsItems = m.buildSettingsItems()
			return m, nil
		}
		if msg.Type == tea.KeyBackspace {
			m.settingsEditVal = removeLastRune(m.settingsEditVal)
		} else {
			for _, r := range msg.Runes {
				m.settingsEditVal += string(r)
			}
		}
		return m, nil
	}

	filtered := m.filteredSettingsIdxs()

	switch key {
	case "esc", "q", "Q":
		// Auto-save when leaving
		if m.cfg != m.cfgOriginal {
			if err := config.Save(m.cfg); err != nil {
				m.err = fmt.Errorf("failed to save settings: %v", err)
				return m, nil
			}
			if m.cfg.Sync != m.cfgOriginal.Sync && m.store != nil {
				syncMgr, err := syncpkg.NewManagerWithOptions(&m.cfg, m.store, m.masterPassword, m.syncManagerOptions())
				if err == nil {
					m.syncManager = syncMgr
				}
			}
			m.err = fmt.Errorf("\u2713 Settings saved")
		}
		m.page = m.modeHomePage()
		return m, nil

	case "shift+tab":
		// Save settings before navigating away
		if m.cfg != m.cfgOriginal {
			if err := config.Save(m.cfg); err != nil {
				m.err = fmt.Errorf("failed to save settings: %v", err)
				return m, nil
			}
			if m.cfg.Sync != m.cfgOriginal.Sync && m.store != nil {
				syncMgr, err := syncpkg.NewManagerWithOptions(&m.cfg, m.store, m.masterPassword, m.syncManagerOptions())
				if err == nil {
					m.syncManager = syncMgr
				}
			}
			m.err = fmt.Errorf("\u2713 Settings saved")
		}
		m.enterPage(m.nextVisiblePage(m.page))
		return m, nil

	case "/":
		m.settingsSearching = true
		m.settingsFilter = ""
		return m, nil

	case "up", "k":
		if key == "k" && !m.cfg.UI.VimMode {
			return m, nil
		}
		// Move to the previous visible (filtered) item
		for i, idx := range filtered {
			if idx == m.settingsCursor {
				if i > 0 {
					m.settingsCursor = filtered[i-1]
				}
				break
			}
		}
		m.err = nil
		return m, nil

	case "down", "j":
		if key == "j" && !m.cfg.UI.VimMode {
			return m, nil
		}
		// Move to the next visible (filtered) item
		for i, idx := range filtered {
			if idx == m.settingsCursor {
				if i < len(filtered)-1 {
					m.settingsCursor = filtered[i+1]
				}
				break
			}
		}
		m.err = nil
		return m, nil

	case "left", "h":
		if key == "h" && !m.cfg.UI.VimMode {
			return m, nil
		}
		m.applySettingChange(m.settingsCursor, "left")
		m.settingsItems = m.buildSettingsItems()
		return m, nil

	case "right", "l":
		if key == "l" && !m.cfg.UI.VimMode {
			return m, nil
		}
		m.applySettingChange(m.settingsCursor, "right")
		m.settingsItems = m.buildSettingsItems()
		return m, nil

	case " ", "enter":
		idx := m.settingsCursor
		if idx >= len(m.settingsItems) {
			return m, nil
		}
		item := m.settingsItems[idx]
		// Action items that need commands or navigation
		switch item.Label {
		case "current team":
			return m, nil
		case "create team":
			m.settingsEditing = true
			m.settingsEditVal = ""
			return m, nil
		case "rename team":
			if item.Disabled {
				return m, nil
			}
			m.settingsEditing = true
			m.settingsEditVal = item.Value
			return m, nil
		case "delete team":
			if item.Disabled {
				return m, nil
			}
			if err := m.deleteCurrentTeam(context.Background()); err != nil {
				m.err = err
			} else {
				m.err = fmt.Errorf("✓ Team deleted")
			}
			m.settingsItems = m.buildSettingsItems()
			return m, nil
		case "move team earlier":
			if item.Disabled {
				return m, nil
			}
			if err := m.reorderCurrentTeam(context.Background(), -1); err != nil {
				m.err = err
			} else {
				m.err = fmt.Errorf("✓ Team order updated")
			}
			m.settingsItems = m.buildSettingsItems()
			return m, nil
		case "move team later":
			if item.Disabled {
				return m, nil
			}
			if err := m.reorderCurrentTeam(context.Background(), 1); err != nil {
				m.err = err
			} else {
				m.err = fmt.Errorf("✓ Team order updated")
			}
			m.settingsItems = m.buildSettingsItems()
			return m, nil
		case "cloud status", "feed", "channel", "version", "PATH health", updateSettingsNoteLabel():
			return m, nil
		case "check now":
			if !m.updateChecking {
				m.updateRunID++
				m.updateChecking = true
				m.err = fmt.Errorf("\u2139 Checking for updates...")
				m.settingsItems = m.buildSettingsItems()
				return m, runUpdateCheckCmd(m.updateRunID, m.currentVersion, m.cfg)
			}
			return m, nil
		case "apply update":
			if m.updateLast != nil && m.updateLast.UpdateAvailable && !m.updateApplying {
				m.updateRunID++
				m.updateApplying = true
				m.err = fmt.Errorf("\u2139 Applying update...")
				m.settingsItems = m.buildSettingsItems()
				return m, runUpdateApplyCmd(m.updateRunID, *m.updateLast)
			}
			return m, nil
		case "fix PATH":
			if m.updateLast != nil && !m.updateApplying {
				exe, _ := os.Executable()
				m.updateRunID++
				m.updateApplying = true
				m.settingsItems = m.buildSettingsItems()
				return m, runUpdatePathFixCmd(m.updateRunID, exe)
			}
			return m, nil
		case "manage tokens":
			m.enterPage(PageTokens)
			return m, nil
		}
		// Kind=2 editable text fields
		if item.Kind == 2 && !item.Disabled {
			m.settingsEditing = true
			m.settingsEditVal = item.Value
			return m, nil
		}
		// Toggle/enum
		m.applySettingChange(m.settingsCursor, "toggle")
		m.settingsItems = m.buildSettingsItems()
		return m, nil
	}

	return m, nil
}

func (m Model) leaveSettings() (tea.Model, tea.Cmd) {
	if m.cfg != m.cfgOriginal {
		if err := config.Save(m.cfg); err != nil {
			m.err = fmt.Errorf("failed to save settings: %v", err)
			return m, nil
		}
		if m.cfg.Sync != m.cfgOriginal.Sync && m.store != nil {
			syncMgr, err := syncpkg.NewManagerWithOptions(&m.cfg, m.store, m.masterPassword, m.syncManagerOptions())
			if err == nil {
				m.syncManager = syncMgr
			}
		}
		m.cfgOriginal = m.cfg
		m.err = fmt.Errorf("\u2713 Settings saved")
	}
	m.page = m.modeHomePage()
	m.settingsSearching = false
	m.settingsEditing = false
	return m, nil
}

// ── Tokens page ───────────────────────────────────────────────────────

func (m Model) handleTokensKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.store == nil {
		m.err = fmt.Errorf("database is locked")
		return m, nil
	}

	key := msg.String()

	// Token reveal mode
	if m.tokenRevealOpen {
		switch key {
		case "c", "C":
			if err := copyTokenToClipboard(m.tokenRevealValue); err != nil {
				m.err = fmt.Errorf("copy failed: %v", err)
				return m, nil
			}
			m.tokenRevealCopied = true
			m.err = fmt.Errorf("\u2713 token copied to clipboard")
			return m, nil
		case "esc":
			m.tokenRevealOpen = false
			m.tokenRevealValue = ""
			m.tokenRevealCopied = false
			m.err = nil
			return m, nil
		}
		return m, nil
	}

	// Token create name mode
	if m.tokenMode == tokenModeCreateName {
		switch key {
		case "esc":
			m.tokenMode = tokenModeList
			m.tokenNameValue = ""
			m.tokenHostPick = map[int]bool{}
			m.teamTokenHostPick = map[string]bool{}
			m.err = nil
			return m, nil
		case "enter":
			name := strings.TrimSpace(m.tokenNameValue)
			if name == "" {
				m.err = fmt.Errorf("token name cannot be empty")
				return m, nil
			}
			m.tokenMode = tokenModeCreateScope
			m.tokenHostPick = map[int]bool{}
			m.teamTokenHostPick = map[string]bool{}
			m.tokenHostIdx = 0
			m.err = fmt.Errorf("select hosts and press Enter to create token")
			return m, nil
		}
		if msg.Type == tea.KeyBackspace {
			m.tokenNameValue = removeLastRune(m.tokenNameValue)
		} else {
			for _, r := range msg.Runes {
				m.tokenNameValue += string(r)
			}
		}
		return m, nil
	}

	// Token create scope (host picker)
	if m.tokenMode == tokenModeCreateScope {
		switch key {
		case "esc":
			m.tokenMode = tokenModeList
			m.tokenHostPick = map[int]bool{}
			m.teamTokenHostPick = map[string]bool{}
			m.tokenHostIdx = 0
			m.tokenNameValue = ""
			m.err = nil
			return m, nil
		case "up", "k":
			if key == "k" && !m.cfg.UI.VimMode {
				return m, nil
			}
			if m.tokenHostIdx > 0 {
				m.tokenHostIdx--
			}
			return m, nil
		case "down", "j":
			if key == "j" && !m.cfg.UI.VimMode {
				return m, nil
			}
			if m.tokenHostIdx < len(m.buildTokenHostItems())-1 {
				m.tokenHostIdx++
			}
			return m, nil
		case " ":
			items := m.buildTokenHostItems()
			if len(items) > 0 {
				item := items[m.tokenHostIdx]
				if m.appMode == appModeTeams {
					if m.teamTokenHostPick[item.ID] {
						delete(m.teamTokenHostPick, item.ID)
					} else {
						m.teamTokenHostPick[item.ID] = true
					}
				} else {
					id, _ := strconv.Atoi(item.ID)
					if m.tokenHostPick[id] {
						delete(m.tokenHostPick, id)
					} else {
						m.tokenHostPick[id] = true
					}
				}
			}
			return m, nil
		case "enter":
			if (m.appMode == appModeTeams && len(m.teamTokenHostPick) == 0) || (m.appMode != appModeTeams && len(m.tokenHostPick) == 0) {
				m.err = fmt.Errorf("select at least one host")
				return m, nil
			}
			name := strings.TrimSpace(m.tokenNameValue)
			var raw string
			var err error
			if m.appMode == appModeTeams {
				raw, err = m.createTeamToken(name)
			} else {
				raw, err = m.createToken(name)
			}
			if err != nil {
				m.err = err
				return m, nil
			}
			m.tokenMode = tokenModeList
			m.tokenHostPick = map[int]bool{}
			m.teamTokenHostPick = map[string]bool{}
			m.tokenNameValue = ""
			if m.appMode == appModeTeams {
				m.loadTeamTokenSummaries()
			} else {
				m.loadTokenSummaries()
			}
			m.tokenRevealOpen = true
			m.tokenRevealValue = raw
			m.tokenRevealCopied = false
			m.err = fmt.Errorf("\u2713 Token created")
			return m, nil
		}
		return m, nil
	}

	// Token list mode
	tokenCount := len(m.tokenSummaries)
	if m.appMode == appModeTeams {
		tokenCount = len(m.teamTokenSummaries)
	}
	switch key {
	case "esc", "q", "Q":
		m.enterPage(m.modeHomePage())
		m.err = nil
		return m, nil

	case "shift+tab":
		m.enterPage(m.nextVisiblePage(m.page))
		return m, nil

	case "up", "k":
		if key == "k" && !m.cfg.UI.VimMode {
			return m, nil
		}
		if m.tokenIdx > 0 {
			m.tokenIdx--
		}
		return m, nil

	case "down", "j":
		if key == "j" && !m.cfg.UI.VimMode {
			return m, nil
		}
		if m.tokenIdx < tokenCount-1 {
			m.tokenIdx++
		}
		return m, nil

	case "a":
		m.tokenMode = tokenModeCreateName
		m.tokenHostPick = map[int]bool{}
		m.teamTokenHostPick = map[string]bool{}
		m.tokenHostIdx = 0
		m.tokenNameValue = ""
		m.err = fmt.Errorf("enter token name and press Enter")
		return m, nil

	case "r":
		return m.revokeSelectedToken()

	case "d":
		return m.deleteSelectedRevokedToken()
	}

	return m, nil
}

func (m Model) revokeSelectedToken() (tea.Model, tea.Cmd) {
	if m.appMode == appModeTeams {
		return m.revokeSelectedTeamToken()
	}
	if len(m.tokenSummaries) == 0 {
		m.err = fmt.Errorf("no tokens to revoke")
		return m, nil
	}
	t := m.tokenSummaries[m.tokenIdx]
	if t.RevokedAt != nil {
		m.err = fmt.Errorf("token already revoked")
		return m, nil
	}
	if err := revokeToken(t.TokenID); err != nil {
		m.err = err
		return m, nil
	}
	m.loadTokenSummaries()
	m.err = fmt.Errorf("\u2713 Token revoked")
	return m, nil
}

func (m Model) deleteSelectedRevokedToken() (tea.Model, tea.Cmd) {
	if m.appMode == appModeTeams {
		return m.deleteSelectedRevokedTeamToken()
	}
	if len(m.tokenSummaries) == 0 {
		m.err = fmt.Errorf("no tokens to delete")
		return m, nil
	}
	t := m.tokenSummaries[m.tokenIdx]
	deleted, err := deleteRevokedToken(t.TokenID)
	if err != nil {
		m.err = err
		return m, nil
	}
	if !deleted {
		m.err = fmt.Errorf("token not found")
		return m, nil
	}
	m.loadTokenSummaries()
	m.err = fmt.Errorf("\u2713 Revoked token deleted")
	return m, nil
}

// ── Quit request ──────────────────────────────────────────────────────

func (m Model) requestQuit() (tea.Model, tea.Cmd) {
	if m.overlay == OverlayQuit {
		return m, nil
	}
	// Check config-driven auto behavior only when mounts are active
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
		}
	}
	// Always show quit confirmation
	m.quitCursor = 0
	m.overlay = OverlayQuit
	m.err = nil
	return m, nil
}

func (m Model) quitAndUnmountAll() (tea.Model, tea.Cmd) {
	if m.mountManager == nil {
		return m, tea.Quit
	}
	unmountCmd := func() tea.Msg {
		m.mountManager.UnmountAll()
		return quitFinishedMsg{}
	}
	return m, tea.Sequence(tea.ShowCursor, unmountCmd, tea.Quit)
}

// ── Form initialization ───────────────────────────────────────────────

func (m *Model) initAddHostForm(label, groupName, tags, hostname, username, port, keyType, existingKey string) {
	authIdx := 0
	switch keyType {
	case "password":
		authIdx = 0
	case "pasted":
		authIdx = 1
	case "ed25519", "rsa", "ecdsa":
		authIdx = 2
	default:
		authIdx = 0
	}
	if strings.TrimSpace(existingKey) != "" {
		authIdx = 1
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

	m.formFields = make([]ui.FormField, 6)
	m.formFields[ui.FFLabel] = ui.NewFormField("label")
	m.formFields[ui.FFLabel].SetValue(label)
	m.formFields[ui.FFTags] = ui.NewFormField("tags")
	m.formFields[ui.FFTags].SetValue(tags)
	m.formFields[ui.FFHostname] = ui.NewFormField("hostname")
	m.formFields[ui.FFHostname].SetValue(hostname)
	m.formFields[ui.FFPort] = ui.NewFormField("port")
	m.formFields[ui.FFPort].SetValue(port)
	m.formFields[ui.FFUsername] = ui.NewFormField("username")
	m.formFields[ui.FFUsername].SetValue(username)

	if authIdx == 0 {
		m.formFields[ui.FFAuthDet] = ui.NewMaskedField("password")
	} else {
		m.formFields[ui.FFAuthDet] = ui.NewMaskedField("key")
		m.formFields[ui.FFAuthDet].SetValue(existingKey)
	}
	m.initFormKeyEditor(existingKey)

	m.formGroups = groupOptions
	m.formGroupIdx = groupSelected
	m.formAuthOpts = []string{"password", "paste key", "generate key"}
	m.formAuthIdx = authIdx
	m.formKeyTypes = []string{"ed25519", "rsa", "ecdsa"}
	keyIdx := 0
	for i, kt := range m.formKeyTypes {
		if kt == initialKeyType {
			keyIdx = i
			break
		}
	}
	m.formKeyIdx = keyIdx
	m.formFocus = ui.FFLabel
	m.formEditing = false
	m.formScrollOffset = 0
	m.formSecretRevealed = false
	m.formTeamHostID = ""
	m.formTeamCredentialMode = ""
	m.formTeamCredentialType = ""
}
