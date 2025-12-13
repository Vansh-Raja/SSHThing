package app

import (
	"fmt"
	"strings"

	"github.com/Vansh-Raja/SSHThing/internal/db"
	"github.com/Vansh-Raja/SSHThing/internal/ssh"
	"github.com/Vansh-Raja/SSHThing/internal/ui"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// Model represents the application state
type Model struct {
	store        *db.Store
	hosts        []Host
	selectedIdx  int
	viewMode     ViewMode
	width        int
	height       int

	// Login/Setup state
	loginInput   textinput.Model
	confirmInput textinput.Model // For password confirmation in setup mode
	isFirstRun   bool            // True if no database exists yet
	setupFocus   int             // 0=password, 1=confirm, 2=submit

	// Search state (Spotlight)
	searchInput  textinput.Model
	isSearching  bool

	// Delete state
	deleteConfirmFocus bool // true = Delete button focused, false = Cancel button focused

	styles       *ui.Styles
	err          error

	// Modal state
	modalForm    *ModalForm
}

// ModalForm holds form state for add/edit modals
type ModalForm struct {
	labelInput    textinput.Model
	hostnameInput textinput.Model
	usernameInput textinput.Model
	portInput     textinput.Model
	
	authMethod    int // 0=Pass, 1=Paste, 2=Gen
	passwordInput textinput.Model
	
	keyOption    string // Legacy
	keyType      string // "ed25519", "rsa", "ecdsa"
	
	pastedKeyInput textarea.Model
	
	focusedField int
}


// NewModel creates a new application model
func NewModel() Model {
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

	return Model{
		hosts:        []Host{},
		selectedIdx:  0,
		viewMode:     viewMode,
		styles:       ui.NewStyles(),
		searchInput:  searchInput,
		loginInput:   loginInput,
		confirmInput: confirmInput,
		isFirstRun:   isFirstRun,
		isSearching:  false,
	}
}

// Init initializes the application
func (m Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, tea.HideCursor)
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case sshFinishedMsg:
		// SSH session ended, reload hosts to update last_connected
		m.loadHosts()
		m.viewMode = ViewModeList
		if msg.err != nil {
			m.err = fmt.Errorf("SSH session ended: %v", msg.err)
		} else {
			m.err = fmt.Errorf("Disconnected from %s", msg.hostname)
		}
		return m, tea.HideCursor
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
	} else if m.isSearching {
		m.searchInput, cmd = m.searchInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

// handleKeyPress processes keyboard input
func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global quit keybinding
	if msg.String() == "ctrl+c" {
		return m, tea.Quit
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
	case ViewModeSpotlight:
		return m.handleSpotlightKeys(msg)
	case ViewModeHelp:
		return m.handleHelpKeys(msg)
	}

	return m, nil
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
		m.loadHosts()
		m.err = nil
		m.viewMode = ViewModeList
		return m, nil

	case tea.KeyEsc:
		return m, tea.Quit
	}
	if msg.String() == "ctrl+r" {
		// Destructive reset: delete DB so user can re-run setup.
		if err := db.Delete(); err != nil {
			m.err = fmt.Errorf("failed to delete database: %v", err)
			return m, nil
		}
		m.err = fmt.Errorf("database deleted — run setup")
		m.isFirstRun = true
		m.viewMode = ViewModeSetup
		m.setupFocus = 0
		m.loginInput.SetValue("")
		m.confirmInput.SetValue("")
		m.loginInput.Focus()
		m.confirmInput.Blur()
		m.store = nil
		m.hosts = nil
		m.selectedIdx = 0
		return m, nil
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
			m.loadHosts()
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
			Hostname:      h.Hostname,
			Username:      h.Username,
			Port:          h.Port,
			HasKey:        hasKey,
			KeyType:       h.KeyType,
			CreatedAt:     h.CreatedAt,
			LastConnected: h.LastConnected,
		}
	}
}

// handleSpotlightKeys handles input for the spotlight view
func (m Model) handleSpotlightKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.viewMode = ViewModeList
		m.isSearching = false
		m.searchInput.Blur()
		m.searchInput.Reset()
		return m, nil
	
	case "enter":
		// Connect to selected host
		filtered := m.getFilteredHosts()
		if len(filtered) > 0 {
			if m.selectedIdx >= len(filtered) {
				m.selectedIdx = 0
			}
			host := filtered[m.selectedIdx]
			m.isSearching = false
			m.searchInput.Blur()
			m.searchInput.Reset()
			return m.connectToHost(host)
		}
		return m, nil

	case "up", "k":
		if m.selectedIdx > 0 {
			m.selectedIdx--
		}

	case "down", "j":
		filtered := m.getFilteredHosts()
		if m.selectedIdx < len(filtered)-1 {
			m.selectedIdx++
		}
		
	default:
		// Forward key to search input
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		// Reset selection when typing
		m.selectedIdx = 0
		return m, cmd
	}
	
	return m, nil
}

// handleListKeys handles keyboard input in list view
func (m Model) handleListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Normal list navigation
	switch msg.String() {
	case "q":
		return m, tea.Quit

	case "up", "k":
		if m.selectedIdx > 0 {
			m.selectedIdx--
		}

	case "down", "j":
		filtered := m.getFilteredHosts()
		if m.selectedIdx < len(filtered)-1 {
			m.selectedIdx++
		}

	case "ctrl+u": // Page up
		m.selectedIdx = max(0, m.selectedIdx-10)

	case "ctrl+d": // Page down
		filtered := m.getFilteredHosts()
		m.selectedIdx = min(len(filtered)-1, m.selectedIdx+10)

	case "home", "g":
		m.selectedIdx = 0

	case "end", "G":
		filtered := m.getFilteredHosts()
		m.selectedIdx = len(filtered) - 1

	case "/", "ctrl+f":
		m.viewMode = ViewModeSpotlight
		m.isSearching = true
		m.searchInput.Focus()
		m.selectedIdx = 0 // Reset selection for search

	case "a", "ctrl+n":
		m.viewMode = ViewModeAddHost
		m.modalForm = m.newModalForm("", "", "", "22", "", "")

	case "e":
		filtered := m.getFilteredHosts()
		if len(filtered) > 0 && m.selectedIdx < len(filtered) {
			host := filtered[m.selectedIdx]
			m.viewMode = ViewModeEditHost
			var existingKey string
			if m.store != nil && host.HasKey && host.KeyType != "password" {
				key, err := m.store.GetHostKey(host.ID)
				if err == nil {
					existingKey = key
				}
			}
			m.modalForm = m.newModalForm(host.Label, host.Hostname, host.Username, fmt.Sprintf("%d", host.Port), host.KeyType, existingKey)
		}

	case "d", "delete":
		m.viewMode = ViewModeDeleteHost
		m.deleteConfirmFocus = false // Default to Cancel

	case "enter":
		// Connect to selected host
		filtered := m.getFilteredHosts()
		if len(filtered) > 0 && m.selectedIdx < len(filtered) {
			host := filtered[m.selectedIdx]
			return m.connectToHost(host)
		}

	case "?":
		m.viewMode = ViewModeHelp
	}

	return m, nil
}

// Helper to create new modal form with initialized text inputs
func (m Model) newModalForm(label, hostname, username, port, keyType, existingKey string) *ModalForm {
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

	f := &ModalForm{
		labelInput:     textinput.New(),
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
			// Don't store SSH passwords. SSH will prompt at connect-time.
			plainKey = ""
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
		if m.viewMode == ViewModeAddHost {
			host := &db.HostModel{
				Label:    strings.TrimSpace(m.modalForm.labelInput.Value()),
				Hostname: m.modalForm.hostnameInput.Value(),
				Username: m.modalForm.usernameInput.Value(),
				Port:     portInt,
				KeyType:  keyType,
			}
			if err := m.store.CreateHost(host, plainKey); err != nil {
				m.err = err
				return m, nil
			}
			m.err = fmt.Errorf("✓ Host '%s' added", host.Hostname)
		} else {
			// Edit - update host with key data if provided, otherwise just metadata
			filtered := m.getFilteredHosts()
			if m.selectedIdx < len(filtered) {
				originalID := filtered[m.selectedIdx].ID
				host := &db.HostModel{
					ID:       originalID,
					Label:    strings.TrimSpace(m.modalForm.labelInput.Value()),
					Hostname: m.modalForm.hostnameInput.Value(),
					Username: m.modalForm.usernameInput.Value(),
					Port:     portInt,
					KeyType:  keyType,
				}
				// Update with key data if provided, otherwise just metadata
				if m.modalForm.authMethod == ui.AuthPassword || plainKey != "" {
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
		m.viewMode = ViewModeList
		m.modalForm = nil
		return m, nil
	}

	// Helper to blur all inputs
	blurAll := func() {
		m.modalForm.labelInput.Blur()
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
		case ui.FieldLabel:    m.modalForm.labelInput.Focus()
		case ui.FieldHostname: m.modalForm.hostnameInput.Focus()
		case ui.FieldUsername: m.modalForm.usernameInput.Focus()
		case ui.FieldPort:     m.modalForm.portInput.Focus()
		case ui.FieldAuthDetails: 
			if m.modalForm.authMethod == ui.AuthKeyPaste {
				m.modalForm.pastedKeyInput.Focus()
			}
		}
	}

	totalFields := 8
	var cmd tea.Cmd

	// Navigation Helpers
	cycleAuth := func(dir int) {
		m.modalForm.authMethod = (m.modalForm.authMethod + dir + 3) % 3
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
		if m.modalForm.focusedField == ui.FieldAuthMethod {
			cycleAuth(-1)
		} else if m.modalForm.focusedField == ui.FieldSubmit || m.modalForm.focusedField == ui.FieldCancel {
			cycleButtons()
		} else {
			cmd = m.handleInputUpdate(msg)
		}

	case tea.KeyRight:
		if m.modalForm.focusedField == ui.FieldAuthMethod {
			cycleAuth(1)
		} else if m.modalForm.focusedField == ui.FieldSubmit || m.modalForm.focusedField == ui.FieldCancel {
			cycleButtons()
		} else {
			cmd = m.handleInputUpdate(msg)
		}

	case tea.KeyEnter:
		if m.modalForm.focusedField == ui.FieldSubmit {
			return submitAndClose()
		} else if m.modalForm.focusedField == ui.FieldCancel {
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

	default:
			// Handle runes and string-based keys
			str := msg.String()

			// Quick-save from anywhere
			if str == "shift+enter" || str == "shift+return" {
				return submitAndClose()
			}
	
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
			if str == "h" && m.modalForm.focusedField == ui.FieldAuthMethod {
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
	case ui.FieldHostname:
		m.modalForm.hostnameInput, cmd = m.modalForm.hostnameInput.Update(msg)
	case ui.FieldUsername:
		m.modalForm.usernameInput, cmd = m.modalForm.usernameInput.Update(msg)
	case ui.FieldPort:
		m.modalForm.portInput, cmd = m.modalForm.portInput.Update(msg)
	case ui.FieldAuthDetails:
		if m.modalForm.authMethod == ui.AuthKeyPaste {
			m.modalForm.pastedKeyInput, cmd = m.modalForm.pastedKeyInput.Update(msg)
		}
	}
	return cmd
}


// handleDeleteKeys handles keyboard input in delete confirmation view
func (m Model) handleDeleteKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	doDelete := func() Model {
		filtered := m.getFilteredHosts()
		if m.selectedIdx < len(filtered) {
			host := filtered[m.selectedIdx]
			if err := m.store.DeleteHost(host.ID); err != nil {
				m.err = err
			} else {
				m.err = fmt.Errorf("✓ Host '%s' deleted", host.Hostname)
				m.loadHosts()
				if m.selectedIdx >= len(m.hosts) && len(m.hosts) > 0 {
					m.selectedIdx = len(m.hosts) - 1
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

// handleHelpKeys handles keyboard input in help view
func (m Model) handleHelpKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "?", "esc", "q":
		m.viewMode = ViewModeList
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
		// No extra validation: SSH will prompt for password at connect-time.
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

	// Convert hosts to interface{} for rendering
	filtered := m.getFilteredHosts()
	hostsInterface := make([]interface{}, len(filtered))
	for i, host := range filtered {
		hostsInterface[i] = map[string]interface{}{
			"ID":            host.ID,
			"Label":         host.Label,
			"Hostname":      host.Hostname,
			"Username":      host.Username,
			"Port":          host.Port,
			"HasKey":        host.HasKey,
			"KeyType":       host.KeyType,
			"CreatedAt":     host.CreatedAt,
			"LastConnected": host.LastConnected,
		}
	}

	switch m.viewMode {
	case ViewModeSetup:
		return m.styles.RenderSetupView(m.width, m.height, m.loginInput, m.confirmInput, m.setupFocus, m.err) + hideCursorAndMoveAway
	case ViewModeLogin:
		return m.styles.RenderLoginView(m.width, m.height, m.loginInput, m.err) + hideCursorAndMoveAway
	case ViewModeList:
		return m.styles.RenderListView(m.width, m.height, hostsInterface, m.selectedIdx, m.searchInput.Value(), m.isSearching, m.err) + hideCursorAndMoveAway
	case ViewModeHelp:
		return m.styles.RenderHelpView(m.width, m.height) + hideCursorAndMoveAway
	case ViewModeAddHost, ViewModeEditHost:
		return m.renderModalView() + hideCursorAndMoveAway
	case ViewModeDeleteHost:
		return m.renderDeleteView() + hideCursorAndMoveAway
	case ViewModeSpotlight:
		return m.renderSpotlightView() + hideCursorAndMoveAway
	default:
		return "Unknown view mode" + hideCursorAndMoveAway
	}
}

// renderSpotlightView renders the spotlight overlay
func (m Model) renderSpotlightView() string {
	// Convert filtered hosts to interface{} for rendering
	filtered := m.getFilteredHosts()
	hostsInterface := make([]interface{}, len(filtered))
	for i, host := range filtered {
		hostsInterface[i] = map[string]interface{}{
			"Label":    host.Label,
			"Hostname": host.Hostname,
			"Username": host.Username,
		}
	}
	
	return m.styles.RenderSpotlight(m.width, m.height, m.searchInput, hostsInterface, m.selectedIdx)
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
	filtered := m.getFilteredHosts()
	if len(filtered) == 0 || m.selectedIdx >= len(filtered) {
		return m.styles.Error.Render("No host selected")
	}

	host := filtered[m.selectedIdx]
	return m.styles.RenderDeleteModal(m.width, m.height, host.Hostname, host.Username, m.deleteConfirmFocus)
}

// connectToHost initiates an SSH connection to the given host
func (m Model) connectToHost(host Host) (tea.Model, tea.Cmd) {
	// Get the decrypted key if available
	var privateKey string
	if host.HasKey && host.KeyType != "password" {
		key, err := m.store.GetHostKey(host.ID)
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

	// Build the SSH connection
	conn := ssh.Connection{
		Hostname:   host.Hostname,
		Username:   host.Username,
		Port:       host.Port,
		PrivateKey: privateKey,
	}

	// For password auth, we don't pass a password - SSH will prompt
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
			return sshFinishedMsg{err: err, hostname: host.Hostname}
		}),
	)
}

// sshFinishedMsg is sent when an SSH session ends
type sshFinishedMsg struct {
	err      error
	hostname string
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
