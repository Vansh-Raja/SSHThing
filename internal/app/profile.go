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
}

func (m *Model) startProfileSignIn(ctx context.Context) tea.Cmd {
	m.refreshTeamsClient()

	started, err := m.teamsClient.StartCLIAuth(ctx, "SSHThing TUI")
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
		DisplayName:      m.profileDisplayName,
		Email:            m.profileEmail,
		ShowOpenTeamsCTA: m.profileShowOpenTeamsCTA,
		AppModeLabel:     m.modeLabel(),
		Err:              m.err,
		Page:             m.page,
	}
}

func (m Model) handleProfileKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	ctx := context.Background()

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
			return m, m.startProfileSignIn(ctx)
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
