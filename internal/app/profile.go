package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Vansh-Raja/SSHThing/internal/teams"
	"github.com/Vansh-Raja/SSHThing/internal/teamssession"
	"github.com/Vansh-Raja/SSHThing/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) prepareProfilePage() {
	m.refreshTeamsClient()
	m.syncProfileFromSession()
}

func (m *Model) syncProfileFromSession() {
	if m.profileSignedIn() {
		m.profileState = profileStateSignedIn
		m.profileDisplayName = strings.TrimSpace(m.teamsSession.UserName)
		if m.profileDisplayName == "" {
			m.profileDisplayName = strings.TrimSpace(m.teamsSession.UserEmail)
		}
		if m.profileDisplayName == "" {
			m.profileDisplayName = strings.TrimSpace(m.teamsSession.UserID)
		}
		m.profileEmail = strings.TrimSpace(m.teamsSession.UserEmail)
		return
	}

	m.profileState = profileStateSignedOut
	m.profileDisplayName = ""
	m.profileEmail = ""
	m.profileShowOpenTeamsCTA = false
	m.profilePendingAuth = nil
	m.profileLastAuthURL = ""
	m.profileHeadlessCode = ""
	m.profileHeadlessClaiming = false
}

// startProfileSignIn runs the browser (poll-based) sign-in: open the URL
// locally and poll until the web flow completes.
func (m *Model) startProfileSignIn(ctx context.Context) tea.Cmd {
	m.refreshTeamsClient()

	started, err := m.teamsClient.StartCLIAuth(ctx, "SSHThing TUI", false)
	if err != nil {
		m.err = err
		return nil
	}

	m.profileAuthRunID++
	m.profilePendingAuth = &started
	m.profileLastAuthURL = started.AuthURL
	m.profileState = profileStateSigningIn
	m.profileShowOpenTeamsCTA = false

	if err := openTeamsURL(started.AuthURL); err != nil {
		m.err = fmt.Errorf("sign-in started, but could not open browser: %v", err)
	} else {
		m.err = fmt.Errorf("browser sign-in started")
	}

	return pollProfileAuthCmd(m.profileAuthRunID, m.teamsClient, started.SessionID, started.PollSecret, time.Duration(started.PollIntervalSeconds)*time.Second)
}

// startHeadlessSignIn runs the headless (paste-back) sign-in: show the
// auth URL, let the user open it on any device, and finish by pasting
// the browser-displayed claim code. No local browser, no polling.
func (m *Model) startHeadlessSignIn(ctx context.Context) tea.Cmd {
	m.refreshTeamsClient()

	started, err := m.teamsClient.StartCLIAuth(ctx, "SSHThing TUI", true)
	if err != nil {
		m.err = err
		return nil
	}

	m.profileAuthRunID++
	m.profilePendingAuth = &started
	m.profileLastAuthURL = started.AuthURL
	m.profileHeadlessCode = ""
	m.profileHeadlessClaiming = false
	m.profileState = profileStateHeadless
	m.profileShowOpenTeamsCTA = false
	m.err = nil
	return nil
}

func (m *Model) cancelHeadlessSignIn() {
	m.profileAuthRunID++
	m.profilePendingAuth = nil
	m.profileLastAuthURL = ""
	m.profileHeadlessCode = ""
	m.profileHeadlessClaiming = false
	m.syncProfileFromSession()
}

// headlessClaimError maps the raw claim error to a friendly message.
func headlessClaimError(err error) string {
	if err == nil {
		return ""
	}
	s := err.Error()
	switch {
	case strings.Contains(s, "invalid_claim_code"):
		return "that code didn't match — copy it again from the browser"
	case strings.Contains(s, "not_completed"):
		return "finish signing in on the browser first, then paste the code"
	case strings.Contains(s, "session_expired"), strings.Contains(s, "expired"):
		return "this sign-in expired — press Esc and start again"
	case strings.Contains(s, "session_not_found"):
		return "sign-in session not found — press Esc and start again"
	default:
		return s
	}
}

func (m *Model) completeProfileSignIn(result teams.CliAuthPollResponse) {
	currentTeamID := strings.TrimSpace(m.teamsSession.CurrentTeamID)
	m.teamsSession = teamssession.Session{
		AccessToken:   result.AccessToken,
		RefreshToken:  result.RefreshToken,
		ExpiresAt:     result.ExpiresAt,
		CurrentTeamID: currentTeamID,
		UserID:        result.User.ID,
		UserName:      result.User.Name,
		UserEmail:     result.User.Email,
	}
	m.saveTeamsSession()

	m.profilePendingAuth = nil
	m.profileLastAuthURL = ""
	m.profileState = profileStateSignedIn
	m.profileDisplayName = strings.TrimSpace(m.teamsSession.UserName)
	if m.profileDisplayName == "" {
		m.profileDisplayName = strings.TrimSpace(m.teamsSession.UserEmail)
	}
	m.profileEmail = strings.TrimSpace(m.teamsSession.UserEmail)
	m.profileShowOpenTeamsCTA = true
}

func (m *Model) cancelProfileSignIn() {
	m.profileAuthRunID++
	m.profilePendingAuth = nil
	m.profileLastAuthURL = ""
	m.profileHeadlessCode = ""
	m.profileHeadlessClaiming = false
	m.syncProfileFromSession()
}

func (m *Model) signOutProfile(ctx context.Context) {
	if refreshToken := strings.TrimSpace(m.teamsSession.RefreshToken); refreshToken != "" {
		if err := m.teamsClient.Logout(ctx, refreshToken); err != nil {
			m.err = fmt.Errorf("signed out locally; remote revoke failed: %v", err)
		}
	}

	_ = m.clearTeamsSessionState()
	_ = m.clearTeamsCacheState()
	m.profileAuthRunID++
	m.appMode = appModePersonal
	m.syncProfileFromSession()

	if m.err == nil {
		m.err = fmt.Errorf("✓ Signed out")
	}
}

func (m Model) buildProfileViewParams() ui.ProfileViewParams {
	return ui.ProfileViewParams{
		SignedIn:         m.profileState == profileStateSignedIn,
		SigningIn:        m.profileState == profileStateSigningIn,
		ChoosingMode:     m.profileState == profileStateChooseMode,
		Headless:         m.profileState == profileStateHeadless,
		HeadlessURL:      m.profileLastAuthURL,
		HeadlessCode:     m.profileHeadlessCode,
		HeadlessClaiming: m.profileHeadlessClaiming,
		DisplayName:      m.profileDisplayName,
		Email:            m.profileEmail,
		ShowOpenTeamsCTA: m.profileShowOpenTeamsCTA,
		AppModeLabel:     m.modeLabel(),
		Err:              m.err,
		Page:             m.page,
		CommandLine:      m.buildCommandLineView(),
	}
}

func (m Model) handleProfileKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	ctx := context.Background()

	// Headless code entry captures every keystroke — so a claim code
	// containing 'q', ',', etc. isn't hijacked by the nav shortcuts.
	if m.profileState == profileStateHeadless {
		return m.handleHeadlessKeys(msg)
	}

	key := msg.String()
	switch key {
	case "q", "Q":
		m.page = m.modeHomePage()
		m.profileShowOpenTeamsCTA = false
		return m, nil
	case ",":
		m.err = nil
		m.enterPage(PageSettings)
		return m, nil
	case "shift+tab":
		m.enterPage(m.nextVisiblePage(m.page))
		return m, nil
	}

	switch m.profileState {
	case profileStateSignedOut:
		if key == "enter" {
			m.err = nil
			m.profileState = profileStateChooseMode
			return m, nil
		}
	case profileStateChooseMode:
		switch key {
		case "b", "B", "enter":
			return m, m.startProfileSignIn(ctx)
		case "h", "H":
			return m, m.startHeadlessSignIn(ctx)
		case "esc":
			m.profileState = profileStateSignedOut
			m.err = nil
			return m, nil
		}
	case profileStateSigningIn:
		switch key {
		case "o", "O":
			if err := openTeamsURL(m.profileLastAuthURL); err != nil {
				m.err = err
			}
			return m, nil
		case "c", "C":
			m.cancelProfileSignIn()
			m.err = fmt.Errorf("sign-in cancelled")
			return m, nil
		}
	case profileStateSignedIn:
		switch key {
		case "s", "S":
			m.signOutProfile(ctx)
			return m, nil
		}
	}

	return m, nil
}

// handleHeadlessKeys drives the headless paste-back screen: type/paste
// the claim code, Enter to submit, 'c' (empty field) to copy the link,
// Esc to cancel.
func (m Model) handleHeadlessKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// While a claim request is in flight, only Esc is honoured.
	if m.profileHeadlessClaiming {
		if msg.Type == tea.KeyEsc {
			m.cancelHeadlessSignIn()
			m.err = fmt.Errorf("sign-in cancelled")
		}
		return m, nil
	}

	switch msg.Type {
	case tea.KeyEsc:
		m.cancelHeadlessSignIn()
		m.err = fmt.Errorf("sign-in cancelled")
		return m, nil
	case tea.KeyEnter:
		code := strings.TrimSpace(m.profileHeadlessCode)
		if code == "" {
			m.err = fmt.Errorf("paste the code from the browser first")
			return m, nil
		}
		if m.profilePendingAuth == nil {
			m.err = fmt.Errorf("sign-in session lost — press Esc and try again")
			return m, nil
		}
		m.profileHeadlessClaiming = true
		m.err = fmt.Errorf("verifying code…")
		return m, claimProfileAuthCmd(m.profileAuthRunID, m.teamsClient,
			m.profilePendingAuth.SessionID, m.profilePendingAuth.PollSecret, code)
	case tea.KeyBackspace:
		if r := []rune(m.profileHeadlessCode); len(r) > 0 {
			m.profileHeadlessCode = string(r[:len(r)-1])
		}
		return m, nil
	case tea.KeyRunes, tea.KeySpace:
		runes := msg.Runes
		// A single 'c' on an empty field copies the auth link. A paste,
		// or 'c' once the field has text, is literal input.
		if !msg.Paste && len(runes) == 1 && (runes[0] == 'c' || runes[0] == 'C') && m.profileHeadlessCode == "" {
			return m, copyToTerminalClipboardCmd(m.profileLastAuthURL)
		}
		m.profileHeadlessCode += string(runes)
		return m, nil
	}

	return m, nil
}
