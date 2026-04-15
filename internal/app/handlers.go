package app

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/Vansh-Raja/SSHThing/internal/config"
	"github.com/Vansh-Raja/SSHThing/internal/db"
	"github.com/Vansh-Raja/SSHThing/internal/ssh"
	syncpkg "github.com/Vansh-Raja/SSHThing/internal/sync"
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
	}
	return m, nil
}

// ── Page key dispatch ─────────────────────────────────────────────────

func (m Model) handlePageKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.page {
	case PageHome:
		return m.handleHomeKeys(msg)
	case PageSettings:
		return m.handleSettingsKeys(msg)
	case PageTokens:
		return m.handleTokensKeys(msg)
	case PageTeams:
		return m.handleTeamsKeys(msg)
	}
	return m, nil
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
			syncMgr, err := syncpkg.NewManager(&m.cfg, m.store, password)
			if err == nil {
				m.syncManager = syncMgr
			}
		}

		m.overlay = OverlayNone
		m.page = PageHome
		_ = unlock.Save(password, time.Duration(m.cfg.Automation.SessionTTLSeconds)*time.Second)
		return m, nil

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
				syncMgr, err := syncpkg.NewManager(&m.cfg, m.store, password)
				if err == nil {
					m.syncManager = syncMgr
				}
			}

			m.overlay = OverlayNone
			m.page = PageHome
			return m, nil
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
		m.overlay = OverlayNone
		m.searchQuery = ""
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
		if item.Kind == SpotlightItemGroup {
			m.overlay = OverlayNone
			m.searchQuery = ""
			m.spotlightItems = nil
			m.selectGroupInList(item.GroupName)
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
		validationErr := m.validateForm()
		if validationErr != nil {
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
		if groupName != "" {
			if err := m.store.UpsertGroup(groupName); err != nil {
				m.err = err
				return m, nil
			}
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
		m.overlay = OverlayNone
		m.formFields = nil
		return m, nil
	}

	// Shift+Enter / Ctrl+S always submits
	if s := msg.String(); s == "shift+enter" || s == "shift+return" || s == "ctrl+s" {
		return submitAndClose()
	}

	cycleGroup := func(dir int) {
		if len(m.formGroups) == 0 {
			return
		}
		n := len(m.formGroups)
		m.formGroupIdx = (m.formGroupIdx + dir + n) % n
	}

	cycleAuth := func(dir int) {
		m.formAuthIdx = (m.formAuthIdx + dir + len(m.formAuthOpts)) % len(m.formAuthOpts)
	}

	formOrder := []int{ui.FFLabel, ui.FFGroup, ui.FFTags, ui.FFHostname, ui.FFPort, ui.FFUsername, ui.FFAuthMeth, ui.FFAuthDet, ui.FFSave}

	findIdx := func(f int) int {
		for i, v := range formOrder {
			if v == f {
				return i
			}
		}
		return 0
	}

	switch msg.Type {
	case tea.KeyEsc:
		if m.formEditing && isTextField(m.formFocus) {
			m.formEditing = false
			return m, nil
		}
		m.overlay = OverlayNone
		m.formFields = nil
		return m, nil

	case tea.KeyTab, tea.KeyDown:
		m.formEditing = false
		idx := findIdx(m.formFocus)
		idx = (idx + 1) % len(formOrder)
		m.formFocus = formOrder[idx]
		return m, nil

	case tea.KeyShiftTab, tea.KeyUp:
		m.formEditing = false
		idx := findIdx(m.formFocus)
		idx = (idx - 1 + len(formOrder)) % len(formOrder)
		m.formFocus = formOrder[idx]
		return m, nil

	case tea.KeyLeft:
		if m.formFocus == ui.FFGroup {
			cycleGroup(-1)
		} else if m.formFocus == ui.FFAuthMeth {
			cycleAuth(-1)
		} else if m.formEditing && isTextField(m.formFocus) {
			m.formFields[m.formFocus].MoveLeft()
		}
		return m, nil

	case tea.KeyRight:
		if m.formFocus == ui.FFGroup {
			cycleGroup(1)
		} else if m.formFocus == ui.FFAuthMeth {
			cycleAuth(1)
		} else if m.formEditing && isTextField(m.formFocus) {
			m.formFields[m.formFocus].MoveRight()
		}
		return m, nil

	case tea.KeyEnter:
		if m.formFocus == ui.FFSave {
			return submitAndClose()
		}
		if isTextField(m.formFocus) {
			if m.formEditing {
				m.formEditing = false
			} else {
				m.formEditing = true
			}
			return m, nil
		}
		// For group/auth selectors, enter could submit
		return submitAndClose()

	case tea.KeyBackspace:
		if m.formEditing && isTextField(m.formFocus) {
			m.formFields[m.formFocus].DeleteBack()
		}
		return m, nil
	}

	// Rune input
	if m.formEditing && isTextField(m.formFocus) {
		for _, r := range msg.Runes {
			m.formFields[m.formFocus].InsertRune(r)
		}
		return m, nil
	}

	// Spacebar for key gen type cycling
	str := msg.String()
	if str == " " && m.formFocus == ui.FFAuthDet && m.formAuthIdx == 2 {
		m.formKeyIdx = (m.formKeyIdx + 1) % len(m.formKeyTypes)
		return m, nil
	}

	// Vim nav for selectors
	if m.cfg.UI.VimMode {
		if str == "h" && m.formFocus == ui.FFGroup {
			cycleGroup(-1)
			return m, nil
		} else if str == "l" && m.formFocus == ui.FFGroup {
			cycleGroup(1)
			return m, nil
		} else if str == "h" && m.formFocus == ui.FFAuthMeth {
			cycleAuth(-1)
			return m, nil
		} else if str == "l" && m.formFocus == ui.FFAuthMeth {
			cycleAuth(1)
			return m, nil
		}
	}

	return m, nil
}

// ── Delete host overlay ───────────────────────────────────────────────

func (m Model) handleDeleteHostKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	doDelete := func() Model {
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

	case "T":
		m.page = PageTeams
		m.openTeamsPage()
		return m, nil

	case "shift+tab":
		m.page = (m.page + 1) % NumPages
		if m.page == PageTokens {
			m.loadTokenSummaries()
		} else if m.page == PageSettings {
			m.cfgOriginal = m.cfg
			m.settingsItems = m.buildSettingsItems()
		} else if m.page == PageTeams {
			m.openTeamsPage()
		}
		return m, nil

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
		if item, ok := m.selectedListItem(); ok && item.Kind == ListItemNewGroup {
			m.groupInputValue = ""
			m.groupInputCursor = 0
			m.groupFocus = 0
			m.overlay = OverlayCreateGroup
			return m, nil
		}
		groupPrefill := ""
		if g, ok := m.selectedGroup(); ok {
			groupPrefill = g
		}
		m.initAddHostForm("", groupPrefill, "", "", "", "22", "", "")
		m.overlay = OverlayAddHost
		m.formEditIdx = -1

	case "e":
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
		if ok {
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
		}

	case "d", "delete":
		if item, ok := m.selectedListItem(); ok && item.Kind == ListItemGroup {
			if item.GroupName == "Ungrouped" {
				m.err = fmt.Errorf("cannot delete Ungrouped")
				return m, nil
			}
			m.groupOldName = item.GroupName
			m.groupDeleteCursor = 1 // default to cancel
			m.overlay = OverlayDeleteGroup
			return m, nil
		}
		if _, ok := m.selectedHost(); ok {
			m.deleteCursor = 1 // default to cancel
			m.overlay = OverlayDeleteHost
		}

	case "ctrl+g":
		m.groupInputValue = ""
		m.groupInputCursor = 0
		m.groupFocus = 0
		m.overlay = OverlayCreateGroup
		return m, nil

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
		m.cfgOriginal = m.cfg
		m.settingsItems = m.buildSettingsItems()
		m.settingsCursor = 0
		m.settingsFilter = ""
		m.settingsSearching = false
		m.page = PageSettings
		m.err = nil

	case "Y":
		if m.syncing {
			m.err = fmt.Errorf("\u2139 sync already in progress")
			return m, nil
		}
		if m.syncManager == nil {
			m.err = fmt.Errorf("\u26A0 sync manager is nil")
			return m, nil
		}
		if !m.syncManager.IsEnabled() {
			m.err = fmt.Errorf("\u26A0 sync is disabled \u2014 enable in settings")
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

	return m, nil
}

// ── Teams page ───────────────────────────────────────────────────────

func (m Model) handleTeamsKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	m.clampTeamsMockState()

	switch key {
	case "esc", "q", "Q":
		if m.teamsState == teamsStateInviteMember ||
			m.teamsState == teamsStateEditMember ||
			m.teamsState == teamsStateRemoveMember {
			m.leaveTeamsMemberManagement()
			return m, nil
		}
		if m.teamsState == teamsStateMembers {
			m.teamsState = teamsStateHosts
			m.err = nil
			return m, nil
		}
		m.page = PageHome
		m.err = nil
		return m, nil

	case "shift+tab":
		m.page = (m.page + 1) % NumPages
		if m.page == PageTokens {
			m.loadTokenSummaries()
		} else if m.page == PageSettings {
			m.cfgOriginal = m.cfg
			m.settingsItems = m.buildSettingsItems()
		} else if m.page == PageTeams {
			m.openTeamsPage()
		}
		return m, nil

	case "h":
		if m.teamsAuthed && m.teamsHasTeam &&
			m.teamsState != teamsStateInviteMember &&
			m.teamsState != teamsStateEditMember &&
			m.teamsState != teamsStateRemoveMember {
			m.teamsState = teamsStateHosts
		}
		return m, nil

	case "m":
		if m.teamsAuthed && m.teamsHasTeam &&
			m.teamsState != teamsStateInviteMember &&
			m.teamsState != teamsStateEditMember &&
			m.teamsState != teamsStateRemoveMember {
			m.teamsState = teamsStateMembers
		}
		return m, nil
	}

	switch m.teamsState {
	case teamsStateInviteMember:
		return m.handleTeamsInviteMemberKeys(msg)
	case teamsStateEditMember:
		return m.handleTeamsEditMemberKeys(msg)
	case teamsStateRemoveMember:
		return m.handleTeamsRemoveMemberKeys(msg)
	}

	switch key {
	case "up", "k":
		if key == "k" && !m.cfg.UI.VimMode {
			return m, nil
		}
		if m.teamsState == teamsStateMembers {
			m.moveTeamsMemberSelection(-1)
		} else if m.teamsState == teamsStateHosts {
			m.moveTeamsHostSelection(-1)
		} else {
			if m.teamsActionIdx > 0 {
				m.teamsActionIdx--
			}
		}
		return m, nil

	case "down", "j":
		if key == "j" && !m.cfg.UI.VimMode {
			return m, nil
		}
		if m.teamsState == teamsStateMembers {
			m.moveTeamsMemberSelection(1)
		} else if m.teamsState == teamsStateHosts {
			m.moveTeamsHostSelection(1)
		} else {
			maxIdx := len(m.teamsActionOptions()) - 1
			if m.teamsActionIdx < maxIdx {
				m.teamsActionIdx++
			}
		}
		return m, nil

	case "a":
		switch m.teamsState {
		case teamsStateHosts:
			m.err = fmt.Errorf("ℹ Mock: add shared host later")
		case teamsStateMembers:
			m.openTeamsInviteMember()
		}
		return m, nil

	case "e":
		switch m.teamsState {
		case teamsStateHosts:
			m.err = fmt.Errorf("ℹ Mock: edit shared host later")
		case teamsStateMembers:
			m.openTeamsEditMember()
		}
		return m, nil

	case "d", "delete":
		switch m.teamsState {
		case teamsStateHosts:
			m.err = fmt.Errorf("ℹ Mock: delete shared host later")
		case teamsStateMembers:
			m.openTeamsRemoveMember()
		}
		return m, nil

	case "/":
		if m.teamsState != teamsStateHosts {
			return m, nil
		}
		m.err = fmt.Errorf("ℹ Mock: team search later")
		return m, nil
	}

	if key != "enter" {
		return m, nil
	}

	switch m.teamsState {
	case teamsStateLogin:
		switch m.teamsActionIdx {
		case 0:
			m.teamsAuthed = true
			m.teamsHasTeam = true
			m.resolveTeamsState()
			m.err = fmt.Errorf("✓ Logged in to Teams")
		case 1:
			m.teamsAuthed = true
			m.teamsHasTeam = false
			m.resolveTeamsState()
			m.err = fmt.Errorf("✓ Teams account created")
		default:
			m.page = PageHome
			m.err = nil
		}
		return m, nil
	case teamsStateEmpty:
		switch m.teamsActionIdx {
		case 0:
			m.teamsHasTeam = true
			m.resolveTeamsState()
			m.err = fmt.Errorf("✓ Team created")
		case 1:
			m.teamsHasTeam = true
			m.resolveTeamsState()
			m.err = fmt.Errorf("✓ Joined team")
		default:
			m.page = PageHome
			m.err = nil
		}
		return m, nil
	case teamsStateHosts:
		item, ok := m.selectedTeamsHostItem()
		if !ok {
			return m, nil
		}
		if item.IsGroup {
			group := item.GroupName
			if group == "" {
				group = "Ungrouped"
			}
			m.collapsed[group] = !m.collapsed[group]
			return m, nil
		}
		m.err = fmt.Errorf("✓ Mock connect: %s", item.Label)
		return m, nil
	}

	return m, nil
}

func (m Model) handleTeamsInviteMemberKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "up", "k":
		if key == "k" && !m.cfg.UI.VimMode {
			return m, nil
		}
		if m.teamsManageFocus > 0 {
			m.teamsManageFocus--
		}
		return m, nil
	case "down", "j", "tab":
		if key == "j" && !m.cfg.UI.VimMode {
			return m, nil
		}
		if m.teamsManageFocus < 3 {
			m.teamsManageFocus++
		}
		return m, nil
	case "shift+tab":
		if m.teamsManageFocus > 0 {
			m.teamsManageFocus--
		}
		return m, nil
	case "left", "h":
		if key == "h" && !m.cfg.UI.VimMode {
			return m, nil
		}
		switch m.teamsManageFocus {
		case 1:
			m.teamsInviteRole = (m.teamsInviteRole - 1 + len(teamsMemberRoles)) % len(teamsMemberRoles)
		case 3:
			m.teamsManageFocus = 2
		}
		return m, nil
	case "right", "l":
		if key == "l" && !m.cfg.UI.VimMode {
			return m, nil
		}
		switch m.teamsManageFocus {
		case 1:
			m.teamsInviteRole = (m.teamsInviteRole + 1) % len(teamsMemberRoles)
		case 2:
			m.teamsManageFocus = 3
		}
		return m, nil
	case "backspace":
		if m.teamsManageFocus == 0 && m.teamsInviteEmail != "" {
			m.teamsInviteEmail = removeLastRune(m.teamsInviteEmail)
		}
		return m, nil
	case "enter":
		switch m.teamsManageFocus {
		case 0:
			m.teamsManageFocus = 1
			return m, nil
		case 1:
			m.teamsManageFocus = 2
			return m, nil
		case 2:
			email := strings.TrimSpace(m.teamsInviteEmail)
			if email == "" {
				m.err = fmt.Errorf("enter an email first")
				return m, nil
			}
			name := strings.TrimSpace(strings.Split(email, "@")[0])
			if name == "" {
				name = "new member"
			}
			m.teamsTeam.Members = append(m.teamsTeam.Members, teamsMockMember{
				ID:       fmt.Sprintf("invite-%d", len(m.teamsTeam.Members)+1),
				Name:     teamsMockDisplayName(name),
				Email:    email,
				Role:     teamsMemberRoles[m.teamsInviteRole],
				Status:   "invited",
				LastSeen: "invite pending",
			})
			m.teamsMemberIdx = len(m.teamsTeam.Members) - 1
			m.teamsInviteEmail = ""
			m.leaveTeamsMemberManagement()
			m.err = fmt.Errorf("✓ Invite sent to %s", email)
			return m, nil
		default:
			m.leaveTeamsMemberManagement()
			return m, nil
		}
	}

	if m.teamsManageFocus == 0 {
		for _, r := range msg.Runes {
			m.teamsInviteEmail += string(r)
		}
	}

	return m, nil
}

func (m Model) handleTeamsEditMemberKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "up", "k":
		if key == "k" && !m.cfg.UI.VimMode {
			return m, nil
		}
		if m.teamsManageFocus > 0 {
			m.teamsManageFocus--
		}
		return m, nil
	case "down", "j", "tab":
		if key == "j" && !m.cfg.UI.VimMode {
			return m, nil
		}
		if m.teamsManageFocus < 2 {
			m.teamsManageFocus++
		}
		return m, nil
	case "shift+tab":
		if m.teamsManageFocus > 0 {
			m.teamsManageFocus--
		}
		return m, nil
	case "left", "h":
		if key == "h" && !m.cfg.UI.VimMode {
			return m, nil
		}
		switch m.teamsManageFocus {
		case 0:
			m.teamsEditRole = (m.teamsEditRole - 1 + len(teamsMemberRoles)) % len(teamsMemberRoles)
		case 2:
			m.teamsManageFocus = 1
		}
		return m, nil
	case "right", "l":
		if key == "l" && !m.cfg.UI.VimMode {
			return m, nil
		}
		switch m.teamsManageFocus {
		case 0:
			m.teamsEditRole = (m.teamsEditRole + 1) % len(teamsMemberRoles)
		case 1:
			m.teamsManageFocus = 2
		}
		return m, nil
	case "enter":
		switch m.teamsManageFocus {
		case 0:
			m.teamsManageFocus = 1
			return m, nil
		case 1:
			if member, ok := m.selectedTeamsMember(); ok {
				for i := range m.teamsTeam.Members {
					if m.teamsTeam.Members[i].ID == member.ID {
						m.teamsTeam.Members[i].Role = teamsMemberRoles[m.teamsEditRole]
						break
					}
				}
				m.leaveTeamsMemberManagement()
				m.err = fmt.Errorf("✓ Updated %s", member.Name)
			}
			return m, nil
		default:
			m.leaveTeamsMemberManagement()
			return m, nil
		}
	}

	return m, nil
}

func (m Model) handleTeamsRemoveMemberKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "left", "h", "up", "k", "right", "l", "down", "j", "tab", "shift+tab":
		if (key == "h" || key == "k" || key == "l" || key == "j") && !m.cfg.UI.VimMode {
			return m, nil
		}
		if m.teamsManageFocus == 0 {
			m.teamsManageFocus = 1
		} else {
			m.teamsManageFocus = 0
		}
		return m, nil
	case "enter":
		if m.teamsManageFocus == 1 {
			m.leaveTeamsMemberManagement()
			return m, nil
		}
		member, ok := m.selectedTeamsMember()
		if !ok {
			m.leaveTeamsMemberManagement()
			return m, nil
		}
		nextMembers := make([]teamsMockMember, 0, len(m.teamsTeam.Members)-1)
		for _, existing := range m.teamsTeam.Members {
			if existing.ID != member.ID {
				nextMembers = append(nextMembers, existing)
			}
		}
		m.teamsTeam.Members = nextMembers
		if m.teamsMemberIdx >= len(m.teamsTeam.Members) && len(m.teamsTeam.Members) > 0 {
			m.teamsMemberIdx = len(m.teamsTeam.Members) - 1
		}
		if len(m.teamsTeam.Members) == 0 {
			m.teamsMemberIdx = 0
		}
		m.leaveTeamsMemberManagement()
		m.err = fmt.Errorf("✓ Removed %s", member.Name)
		return m, nil
	}

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
				syncMgr, err := syncpkg.NewManager(&m.cfg, m.store, m.masterPassword)
				if err == nil {
					m.syncManager = syncMgr
				}
			}
			m.err = fmt.Errorf("\u2713 Settings saved")
		}
		m.page = PageHome
		return m, nil

	case "shift+tab":
		// Save settings before navigating away
		if m.cfg != m.cfgOriginal {
			if err := config.Save(m.cfg); err != nil {
				m.err = fmt.Errorf("failed to save settings: %v", err)
				return m, nil
			}
			if m.cfg.Sync != m.cfgOriginal.Sync && m.store != nil {
				syncMgr, err := syncpkg.NewManager(&m.cfg, m.store, m.masterPassword)
				if err == nil {
					m.syncManager = syncMgr
				}
			}
			m.err = fmt.Errorf("\u2713 Settings saved")
		}
		m.page = (m.page + 1) % NumPages
		if m.page == PageTokens {
			m.loadTokenSummaries()
		} else if m.page == PageSettings {
			m.cfgOriginal = m.cfg
			m.settingsItems = m.buildSettingsItems()
		}
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
		case "channel", "version", "PATH health", updateSettingsNoteLabel():
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
			m.page = PageTokens
			m.loadTokenSummaries()
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
			if m.tokenHostIdx < len(m.hosts)-1 {
				m.tokenHostIdx++
			}
			return m, nil
		case " ":
			if len(m.hosts) > 0 {
				h := m.hosts[m.tokenHostIdx]
				if m.tokenHostPick[h.ID] {
					delete(m.tokenHostPick, h.ID)
				} else {
					m.tokenHostPick[h.ID] = true
				}
			}
			return m, nil
		case "enter":
			if len(m.tokenHostPick) == 0 {
				m.err = fmt.Errorf("select at least one host")
				return m, nil
			}
			name := strings.TrimSpace(m.tokenNameValue)
			raw, err := m.createToken(name)
			if err != nil {
				m.err = err
				return m, nil
			}
			m.tokenMode = tokenModeList
			m.tokenHostPick = map[int]bool{}
			m.tokenNameValue = ""
			m.loadTokenSummaries()
			m.tokenRevealOpen = true
			m.tokenRevealValue = raw
			m.tokenRevealCopied = false
			m.err = fmt.Errorf("\u2713 Token created")
			return m, nil
		}
		return m, nil
	}

	// Token list mode
	switch key {
	case "esc", "q", "Q":
		m.page = PageHome
		m.err = nil
		return m, nil

	case "shift+tab":
		m.page = (m.page + 1) % NumPages
		if m.page == PageTokens {
			m.loadTokenSummaries()
		} else if m.page == PageSettings {
			m.cfgOriginal = m.cfg
			m.settingsItems = m.buildSettingsItems()
		} else if m.page == PageTeams {
			m.openTeamsPage()
		}
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
		if m.tokenIdx < len(m.tokenSummaries)-1 {
			m.tokenIdx++
		}
		return m, nil

	case "a":
		m.tokenMode = tokenModeCreateName
		m.tokenHostPick = map[int]bool{}
		m.tokenHostIdx = 0
		m.tokenNameValue = ""
		m.err = fmt.Errorf("enter token name and press Enter")
		return m, nil

	case "r":
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

	case "d":
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
		m.formFields[ui.FFAuthDet] = ui.NewFormField("key")
		m.formFields[ui.FFAuthDet].SetValue(existingKey)
	}

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
}
