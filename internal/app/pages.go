package app

import (
	"fmt"
	"os"
	"strings"

	"github.com/Vansh-Raja/SSHThing/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
)

var defaultCloudBaseURL = "http://localhost:3000"

func cloudServiceBaseURL() string {
	if value := strings.TrimRight(strings.TrimSpace(os.Getenv("SSHTHING_CLOUD_BASE_URL")), "/"); value != "" {
		return value
	}
	if value := strings.TrimRight(strings.TrimSpace(defaultCloudBaseURL), "/"); value != "" {
		return value
	}
	return "http://localhost:3000"
}

func (m Model) profileSignedIn() bool {
	return m.teamsSession.Valid()
}

func (m Model) teamsTabVisible() bool {
	return m.profileSignedIn()
}

func (m Model) visiblePages() []int {
	switch m.appMode {
	case appModeTeams:
		if !m.profileSignedIn() {
			return []int{PageProfile, PageSettings}
		}
		return []int{PageTeams, PageProfile, PageSettings}
	default:
		return []int{PageHome, PageProfile, PageSettings, PageTokens}
	}
}

func (m Model) visiblePageIndicators() []ui.PageIndicator {
	var indicators []ui.PageIndicator
	for _, page := range m.visiblePages() {
		switch page {
		case PageHome:
			indicators = append(indicators, ui.PageIndicator{Icon: m.icons.Home, Index: page})
		case PageTeams:
			indicators = append(indicators, ui.PageIndicator{Icon: m.icons.Teams, Index: page})
		case PageProfile:
			indicators = append(indicators, ui.PageIndicator{Icon: m.icons.Profile, Index: page})
		case PageSettings:
			indicators = append(indicators, ui.PageIndicator{Icon: m.icons.Settings, Index: page})
		case PageTokens:
			indicators = append(indicators, ui.PageIndicator{Icon: m.icons.Tokens, Index: page})
		}
	}
	return indicators
}

func (m Model) nextVisiblePage(current int) int {
	pages := m.visiblePages()
	if len(pages) == 0 {
		return PageHome
	}
	for i, page := range pages {
		if page == current {
			return pages[(i+1)%len(pages)]
		}
	}
	return pages[0]
}

func (m Model) modeHomePage() int {
	if m.appMode == appModeTeams {
		if !m.profileSignedIn() {
			return PageProfile
		}
		return PageTeams
	}
	return PageHome
}

func (m Model) modeLabel() string {
	if m.appMode == appModeTeams {
		return "teams mode"
	}
	return "personal mode"
}

func (m *Model) syncModeAppearance() {
	if m.appMode == appModeTeams {
		m.theme, m.themeIdx = ui.ThemeByName(m.cfg.TeamsUI.Theme)
		m.icons, m.iconIdx = ui.IconSetByName(m.cfg.TeamsUI.IconSet)
		return
	}
	m.theme, m.themeIdx = ui.ThemeByName(m.cfg.UI.Theme)
	m.icons, m.iconIdx = ui.IconSetByName(m.cfg.UI.IconSet)
}

func (m *Model) toggleAppMode() {
	if m.appMode == appModeTeams {
		m.teamsPage = m.page
		m.appMode = appModePersonal
		m.profileShowOpenTeamsCTA = false
		m.syncModeAppearance()
		m.enterPage(m.personalPage)
		m.err = fmt.Errorf("✓ Personal mode")
		return
	}

	if !m.profileSignedIn() {
		m.enterPage(PageProfile)
		m.err = fmt.Errorf("sign in first to enter Teams mode")
		return
	}

	m.personalPage = m.page
	m.appMode = appModeTeams
	m.profileShowOpenTeamsCTA = false
	m.syncModeAppearance()
	m.enterPage(m.teamsPage)
	m.err = fmt.Errorf("✓ Teams mode")
}

func (m *Model) toggleAppModeCmd() tea.Cmd {
	if m.appMode == appModeTeams {
		return m.maybeAutoRefreshTeamsHealthOnEnter()
	}
	return nil
}

func (m *Model) enterPage(page int) {
	if m.page == PageProfile && page != PageProfile {
		m.profileShowOpenTeamsCTA = false
	}

	visible := false
	for _, candidate := range m.visiblePages() {
		if candidate == page {
			visible = true
			break
		}
	}
	if !visible {
		page = m.modeHomePage()
	}

	m.page = page
	if m.appMode == appModeTeams {
		m.teamsPage = page
	} else {
		m.personalPage = page
	}

	switch page {
	case PageProfile:
		m.prepareProfilePage()
	case PageSettings:
		m.cfgOriginal = m.cfg
		m.settingsItems = m.buildSettingsItems()
		m.settingsCursor = 0
		m.settingsFilter = ""
		m.settingsSearching = false
	case PageTokens:
		m.loadTokenSummaries()
	case PageTeams:
		m.prepareTeamsPage()
	}
}
