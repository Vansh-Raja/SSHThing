package app

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/Vansh-Raja/SSHThing/internal/teamcache"
	"github.com/Vansh-Raja/SSHThing/internal/teams"
	"github.com/Vansh-Raja/SSHThing/internal/teamsclient"
	"github.com/Vansh-Raja/SSHThing/internal/teamssession"
	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) prepareTeamsPage() {
	m.refreshTeamsClient()
	if !m.profileSignedIn() {
		m.teamsState = teamsStateZero
		return
	}
	if err := m.loadTeamsData(context.Background()); err != nil {
		m.err = err
	}
}

func (m *Model) refreshTeamsClient() {
	baseURL := cloudServiceBaseURL()
	if m.teamsClient == nil || m.teamsClient.BaseURL() != baseURL {
		m.teamsClient = teamsclient.New(baseURL)
	}
}

func (m *Model) saveTeamsSession() {
	if !m.cfg.Teams.SessionCacheEnabled {
		return
	}
	m.teamsSession.CurrentTeamID = strings.TrimSpace(m.teamsCurrentTeamID)
	_ = teamssession.Save(m.teamsSession)
}

func (m *Model) saveTeamsCache() {
	if !m.cfg.Teams.SessionCacheEnabled {
		return
	}
	m.teamsCache = teamcache.Cache{
		CurrentTeamID: m.teamsCurrentTeamID,
		Teams:         append([]teams.TeamSummary(nil), m.teamsList...),
		Hosts:         append([]teams.TeamHost(nil), m.teamsItems...),
		LastFetchedAt: time.Now().UnixMilli(),
	}
	_ = teamcache.Save(m.teamsCache)
}

func (m *Model) persistCurrentTeam() {
	m.cfg.Teams.LastTeamID = strings.TrimSpace(m.teamsCurrentTeamID)
	m.teamsSession.CurrentTeamID = strings.TrimSpace(m.teamsCurrentTeamID)
	m.saveTeamsSession()
	m.saveTeamsCache()
}

func (m *Model) clearTeamsSessionState() error {
	m.teamsSession = teamssession.Session{}
	m.teamsState = teamsStateZero
	m.teamsItems = nil
	m.teamsList = nil
	m.teamsCurrentTeamID = ""
	m.cfg.Teams.LastTeamID = ""
	return teamssession.Clear()
}

func (m *Model) clearTeamsCacheState() error {
	m.teamsCache = teamcache.Cache{}
	m.teamsState = teamsStateZero
	m.teamsItems = nil
	m.teamsList = nil
	m.teamsCurrentTeamID = ""
	return teamcache.Clear()
}

func (m *Model) teamsAccessToken(ctx context.Context) (string, error) {
	m.refreshTeamsClient()
	if !m.teamsClient.Enabled() || !m.teamsSession.Valid() {
		return "", fmt.Errorf("cloud session unavailable. Return to Profile to sign in again.")
	}
	if m.teamsSession.Expired(time.Now()) {
		refreshed, err := m.teamsClient.Refresh(ctx, m.teamsSession.RefreshToken)
		if err != nil {
			_ = m.clearTeamsSessionState()
			m.syncProfileFromSession()
			return "", err
		}
		m.teamsSession.AccessToken = refreshed.AccessToken
		m.teamsSession.ExpiresAt = refreshed.ExpiresAt
		m.saveTeamsSession()
	}
	return m.teamsSession.AccessToken, nil
}

func (m *Model) teamsCurrentTeam() (teams.TeamSummary, bool) {
	for _, team := range m.teamsList {
		if team.ID == m.teamsCurrentTeamID {
			return team, true
		}
	}
	return teams.TeamSummary{}, false
}

func (m *Model) teamsCurrentHost() (teams.TeamHost, bool) {
	if m.teamsCursor < 0 || m.teamsCursor >= len(m.teamsItems) {
		return teams.TeamHost{}, false
	}
	return m.teamsItems[m.teamsCursor], true
}

func (m *Model) loadTeamsData(ctx context.Context) error {
	accessToken, err := m.teamsAccessToken(ctx)
	if err != nil {
		return err
	}
	teamList, err := m.teamsClient.ListTeams(ctx, accessToken)
	if err != nil {
		return err
	}
	m.teamsList = teamList
	if len(teamList) == 0 {
		m.teamsCurrentTeamID = ""
		m.teamsItems = nil
		m.teamsCursor = 0
		m.teamsState = teamsStateZero
		m.persistCurrentTeam()
		return nil
	}

	currentID := strings.TrimSpace(m.teamsCurrentTeamID)
	if currentID == "" {
		currentID = strings.TrimSpace(m.teamsSession.CurrentTeamID)
	}
	if currentID == "" {
		currentID = strings.TrimSpace(m.cfg.Teams.LastTeamID)
	}
	if currentID == "" || !m.hasTeam(currentID) {
		currentID = teamList[0].ID
	}
	m.teamsCurrentTeamID = currentID
	m.persistCurrentTeam()
	return m.loadCurrentTeamHosts(ctx)
}

func (m *Model) hasTeam(teamID string) bool {
	for _, team := range m.teamsList {
		if team.ID == teamID {
			return true
		}
	}
	return false
}

func (m *Model) loadCurrentTeamHosts(ctx context.Context) error {
	teamID := strings.TrimSpace(m.teamsCurrentTeamID)
	if teamID == "" {
		m.teamsItems = nil
		m.teamsCursor = 0
		m.teamsState = teamsStateZero
		return nil
	}
	accessToken, err := m.teamsAccessToken(ctx)
	if err != nil {
		return err
	}
	hosts, err := m.teamsClient.ListTeamHosts(ctx, accessToken, teamID)
	if err != nil {
		return err
	}
	m.teamsItems = hosts
	if m.teamsCursor >= len(m.teamsItems) {
		m.teamsCursor = max(0, len(m.teamsItems)-1)
	}
	if m.teamsCursor < 0 {
		m.teamsCursor = 0
	}
	m.teamsState = teamsStateHosts
	m.saveTeamsCache()
	return nil
}

func (m *Model) selectTeamByIndex(ctx context.Context, idx int) error {
	if idx < 0 || idx >= len(m.teamsList) {
		return nil
	}
	m.teamsCurrentTeamID = m.teamsList[idx].ID
	m.teamsCursor = 0
	m.persistCurrentTeam()
	return m.loadCurrentTeamHosts(ctx)
}

func (m *Model) createTeam(ctx context.Context, name string) error {
	accessToken, err := m.teamsAccessToken(ctx)
	if err != nil {
		return err
	}
	created, err := m.teamsClient.CreateTeam(ctx, accessToken, name)
	if err != nil {
		return err
	}
	m.teamsCurrentTeamID = created.ID
	return m.loadTeamsData(ctx)
}

func (m *Model) renameCurrentTeam(ctx context.Context, name string) error {
	team, ok := m.teamsCurrentTeam()
	if !ok {
		return fmt.Errorf("no team selected")
	}
	accessToken, err := m.teamsAccessToken(ctx)
	if err != nil {
		return err
	}
	_, err = m.teamsClient.RenameTeam(ctx, accessToken, team.ID, name)
	if err != nil {
		return err
	}
	return m.loadTeamsData(ctx)
}

func (m *Model) deleteCurrentTeam(ctx context.Context) error {
	team, ok := m.teamsCurrentTeam()
	if !ok {
		return fmt.Errorf("no team selected")
	}
	accessToken, err := m.teamsAccessToken(ctx)
	if err != nil {
		return err
	}
	if err := m.teamsClient.DeleteTeam(ctx, accessToken, team.ID); err != nil {
		return err
	}
	m.teamsCurrentTeamID = ""
	m.teamsCursor = 0
	return m.loadTeamsData(ctx)
}

func (m *Model) reorderCurrentTeam(ctx context.Context, dir int) error {
	team, ok := m.teamsCurrentTeam()
	if !ok {
		return fmt.Errorf("no team selected")
	}
	idx := -1
	for i := range m.teamsList {
		if m.teamsList[i].ID == team.ID {
			idx = i
			break
		}
	}
	if idx < 0 {
		return fmt.Errorf("team not found")
	}
	next := idx + dir
	if next < 0 || next >= len(m.teamsList) {
		return nil
	}
	ordered := append([]teams.TeamSummary(nil), m.teamsList...)
	ordered[idx], ordered[next] = ordered[next], ordered[idx]
	ids := make([]string, 0, len(ordered))
	for _, item := range ordered {
		ids = append(ids, item.ID)
	}
	accessToken, err := m.teamsAccessToken(ctx)
	if err != nil {
		return err
	}
	if err := m.teamsClient.ReorderTeams(ctx, accessToken, ids); err != nil {
		return err
	}
	m.teamsList = ordered
	m.persistCurrentTeam()
	m.saveTeamsCache()
	return nil
}

func parseTeamShortcut(key string) (int, bool) {
	for _, prefix := range []string{"ctrl+", "alt+"} {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		digit := strings.TrimPrefix(key, prefix)
		if len(digit) == 1 && digit[0] >= '1' && digit[0] <= '9' {
			return int(digit[0] - '1'), true
		}
	}
	return 0, false
}

func openTeamsURL(rawURL string) error {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return fmt.Errorf("missing auth url")
	}
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", rawURL).Run()
	case "linux":
		xdgOpen, err := exec.LookPath("xdg-open")
		if err != nil {
			return err
		}
		return exec.Command(xdgOpen, rawURL).Run()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL).Run()
	default:
		return fmt.Errorf("unsupported platform")
	}
}

func (m *Model) openTeamsCommandPalette() {
	m.overlay = OverlaySearch
	m.searchQuery = ">"
	m.spotlightItems = m.buildSpotlightItems(m.searchQuery)
	m.selectedIdx = 0
	m.armedSFTP = false
	m.armedMount = false
	m.armedUnmount = false
}

func (m *Model) openTeamsCreateFlow() {
	m.enterPage(PageSettings)
	for idx, item := range m.settingsItems {
		if strings.EqualFold(item.Label, "create team") {
			m.settingsCursor = idx
			m.settingsEditing = true
			m.settingsEditVal = ""
			break
		}
	}
}

func (m Model) handleTeamsKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	ctx := context.Background()

	if idx, ok := parseTeamShortcut(key); ok {
		if err := m.selectTeamByIndex(ctx, idx); err != nil {
			m.err = err
		} else if idx < len(m.teamsList) {
			m.err = fmt.Errorf("✓ %s", m.teamsList[idx].Name)
		}
		return m, nil
	}

	switch key {
	case "q", "Q":
		m.page = m.modeHomePage()
		return m, nil
	case ",":
		m.enterPage(PageSettings)
		m.err = nil
		return m, nil
	case "shift+tab":
		m.enterPage(m.nextVisiblePage(m.page))
		return m, nil
	case "r", "R":
		if err := m.loadTeamsData(ctx); err != nil {
			m.err = err
		} else {
			m.err = fmt.Errorf("✓ Teams refreshed")
		}
		return m, nil
	case "/":
		m.overlay = OverlaySearch
		m.searchQuery = ""
		m.spotlightItems = m.buildSpotlightItems("")
		m.selectedIdx = 0
		return m, nil
	case "?", "shift+/":
		m.openTeamsCommandPalette()
		return m, nil
	}

	if !m.profileSignedIn() {
		m.err = fmt.Errorf("cloud session unavailable. Return to Profile to sign in again.")
		return m, nil
	}

	if m.teamsState == teamsStateZero {
		if key == "enter" {
			m.openTeamsCreateFlow()
			return m, nil
		}
		return m, nil
	}

	switch key {
	case "up", "k":
		if key == "k" && !m.cfg.UI.VimMode {
			return m, nil
		}
		if m.teamsCursor > 0 {
			m.teamsCursor--
		}
		return m, nil
	case "down", "j":
		if key == "j" && !m.cfg.UI.VimMode {
			return m, nil
		}
		if m.teamsCursor < len(m.teamsItems)-1 {
			m.teamsCursor++
		}
		return m, nil
	case "enter":
		if _, ok := m.teamsCurrentHost(); !ok {
			m.err = fmt.Errorf("no team hosts yet")
			return m, nil
		}
		m.err = fmt.Errorf("team host actions are next")
		return m, nil
	}

	return m, nil
}

func (m *Model) switchTeamByID(ctx context.Context, teamID string) error {
	for idx, team := range m.teamsList {
		if team.ID == teamID {
			return m.selectTeamByIndex(ctx, idx)
		}
	}
	return fmt.Errorf("team_not_found")
}

func (m *Model) teamCommandItems(query string) []SpotlightItem {
	trimmed := strings.TrimSpace(strings.TrimPrefix(query, ">"))
	lower := strings.ToLower(trimmed)
	items := []SpotlightItem{
		{Kind: SpotlightItemCommand, Command: "create_team", Detail: "create team", Score: 1000},
		{Kind: SpotlightItemCommand, Command: "open_settings", Detail: "open teams settings", Score: 900},
		{Kind: SpotlightItemCommand, Command: "open_profile", Detail: "open teams profile", Score: 800},
	}
	for _, team := range m.teamsList {
		score, ok := fuzzyScore(lower, team.Name+" "+team.Slug)
		if lower == "" {
			score = 700 - team.DisplayOrder
			ok = true
		}
		if ok {
			items = append(items, SpotlightItem{
				Kind:    SpotlightItemCommand,
				Command: "switch_team",
				Detail:  "switch to " + team.Name,
				Team:    team,
				Score:   score,
			})
		}
	}
	slices.SortFunc(items, func(a, b SpotlightItem) int { return b.Score - a.Score })
	if len(items) > 12 {
		items = items[:12]
	}
	return items
}
