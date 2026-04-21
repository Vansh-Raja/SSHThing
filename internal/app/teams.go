package app

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/Vansh-Raja/SSHThing/internal/ssh"
	"github.com/Vansh-Raja/SSHThing/internal/teamcache"
	"github.com/Vansh-Raja/SSHThing/internal/teams"
	"github.com/Vansh-Raja/SSHThing/internal/teamsclient"
	"github.com/Vansh-Raja/SSHThing/internal/teamssession"
	tea "github.com/charmbracelet/bubbletea"
)

func normalizeTeamText(value string) string {
	return strings.TrimSpace(strings.ToLower(value))
}

func normalizedTagSet(tags []string) []string {
	out := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = normalizeTeamText(tag)
		if tag == "" {
			continue
		}
		out = append(out, tag)
	}
	sort.Strings(out)
	return out
}

func teamHostScore(candidate teams.TeamHost, req teams.CreateTeamHostRequest) int {
	score := 0
	if normalizeTeamText(candidate.Username) == normalizeTeamText(req.Username) {
		score += 4
	}
	if candidate.Port == req.Port {
		score += 3
	}
	if normalizeTeamText(candidate.Label) == normalizeTeamText(req.Label) {
		score += 2
	}
	if normalizeTeamText(candidate.Group) == normalizeTeamText(req.Group) {
		score++
	}
	return score
}

func teamHostMatchesImport(detail teams.TeamHostDetail, req teams.CreateTeamHostRequest) bool {
	if normalizeTeamText(detail.Hostname) != normalizeTeamText(req.Hostname) {
		return false
	}
	if normalizeTeamText(detail.Username) != normalizeTeamText(req.Username) {
		return false
	}
	if detail.Port != req.Port {
		return false
	}
	if normalizeTeamText(detail.Group) != normalizeTeamText(req.Group) {
		return false
	}
	if normalizeTeamText(detail.Label) != normalizeTeamText(req.Label) {
		return false
	}
	if detail.CredentialMode != req.CredentialMode || detail.CredentialType != req.CredentialType {
		return false
	}
	if strings.TrimSpace(detail.SharedCredential) != strings.TrimSpace(req.SharedCredential) {
		return false
	}

	leftTags := normalizedTagSet(detail.Tags)
	rightTags := normalizedTagSet(req.Tags)
	if len(leftTags) != len(rightTags) {
		return false
	}
	for i := range leftTags {
		if leftTags[i] != rightTags[i] {
			return false
		}
	}
	return true
}

func (m Model) buildTeamHostRequestFromPersonalHost(host Host) (teams.CreateTeamHostRequest, error) {
	req := teams.CreateTeamHostRequest{
		Label:            strings.TrimSpace(host.Label),
		Hostname:         host.Hostname,
		Username:         host.Username,
		Port:             host.Port,
		Group:            strings.TrimSpace(host.GroupName),
		Tags:             append([]string(nil), host.Tags...),
		CredentialMode:   "shared",
		CredentialType:   "none",
		SecretVisibility: "revealed_to_access_holders",
	}
	if host.HasKey {
		secret, err := m.store.GetHostSecret(host.ID)
		if err != nil {
			return req, fmt.Errorf("failed to read local host secret: %v", err)
		}
		switch host.KeyType {
		case "password":
			req.CredentialType = "password"
			req.SharedCredential = secret
		default:
			if err := ssh.ValidatePrivateKey(secret); err != nil {
				return req, fmt.Errorf("local private key is invalid: %v", err)
			}
			req.CredentialType = "private_key"
			req.SharedCredential = normalizePrivateKey(secret)
		}
	}
	return req, nil
}

func (m Model) updateTeamHostFromRequest(ctx context.Context, accessToken, hostID string, req teams.CreateTeamHostRequest) error {
	return m.teamsClient.UpdateTeamHost(ctx, accessToken, hostID, teams.UpdateTeamHostRequest{
		Label:            req.Label,
		Hostname:         req.Hostname,
		Username:         req.Username,
		Port:             req.Port,
		Group:            req.Group,
		Tags:             req.Tags,
		CredentialMode:   req.CredentialMode,
		CredentialType:   req.CredentialType,
		SecretVisibility: req.SecretVisibility,
		SharedCredential: req.SharedCredential,
	})
}

func (m Model) findImportConflict(ctx context.Context, accessToken string, req teams.CreateTeamHostRequest) (*teams.TeamHostDetail, bool, error) {
	var candidates []teams.TeamHost
	for _, teamHost := range m.teamsItems {
		if normalizeTeamText(teamHost.Hostname) == normalizeTeamText(req.Hostname) {
			candidates = append(candidates, teamHost)
		}
	}
	if len(candidates) == 0 {
		return nil, false, nil
	}

	sort.Slice(candidates, func(i, j int) bool {
		return teamHostScore(candidates[i], req) > teamHostScore(candidates[j], req)
	})

	var best *teams.TeamHostDetail
	for _, candidate := range candidates {
		detail, err := m.teamsClient.GetTeamHost(ctx, accessToken, candidate.ID)
		if err != nil {
			return nil, false, err
		}
		if teamHostMatchesImport(detail, req) {
			return &detail, true, nil
		}
		if best == nil {
			copy := detail
			best = &copy
		}
	}
	return best, false, nil
}

func (m *Model) clearTeamsImportFlow() {
	m.teamsImportMode = false
	m.teamsImportConflict = nil
	m.searchQuery = ""
	m.spotlightItems = nil
}

func (m Model) importPersonalHostToCurrentTeam(host Host) (tea.Model, tea.Cmd) {
	if m.store == nil {
		m.err = fmt.Errorf("local host database is unavailable")
		return m, nil
	}

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

	req, err := m.buildTeamHostRequestFromPersonalHost(host)
	if err != nil {
		m.err = err
		return m, nil
	}

	conflict, identical, err := m.findImportConflict(context.Background(), accessToken, req)
	if err != nil {
		m.err = err
		return m, nil
	}
	if conflict != nil {
		m.clearTeamsImportFlow()
		if identical {
			m.overlay = OverlayAddHost
			m.err = fmt.Errorf("This host entry has already been imported.")
			return m, nil
		}
		m.teamsImportConflict = &teamsImportConflictState{
			PersonalHost: host,
			ExistingHost: *conflict,
			Cursor:       0,
		}
		m.overlay = OverlayImportHost
		m.err = nil
		return m, nil
	}

	_, err = m.teamsClient.CreateTeamHost(context.Background(), accessToken, team.ID, req)
	if err != nil {
		m.err = err
		return m, nil
	}

	if err := m.loadCurrentTeamHosts(context.Background()); err != nil {
		m.err = err
		return m, nil
	}

	m.clearTeamsImportFlow()
	m.closeAddHostOverlay()

	label := strings.TrimSpace(host.Label)
	if label == "" {
		label = host.Hostname
	}
	m.err = fmt.Errorf("✓ Imported '%s' into %s", label, team.Name)
	return m, nil
}

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
	sort.Slice(hosts, func(i, j int) bool {
		leftGroup := strings.TrimSpace(hosts[i].Group)
		rightGroup := strings.TrimSpace(hosts[j].Group)
		if leftGroup == "" {
			leftGroup = "Ungrouped"
		}
		if rightGroup == "" {
			rightGroup = "Ungrouped"
		}
		if leftGroup != rightGroup {
			if leftGroup == "Ungrouped" {
				return false
			}
			if rightGroup == "Ungrouped" {
				return true
			}
			return strings.ToLower(leftGroup) < strings.ToLower(rightGroup)
		}
		leftLabel := strings.TrimSpace(hosts[i].Label)
		if leftLabel == "" {
			leftLabel = strings.TrimSpace(hosts[i].Hostname)
		}
		rightLabel := strings.TrimSpace(hosts[j].Label)
		if rightLabel == "" {
			rightLabel = strings.TrimSpace(hosts[j].Hostname)
		}
		if strings.EqualFold(leftLabel, rightLabel) {
			return strings.ToLower(hosts[i].Hostname) < strings.ToLower(hosts[j].Hostname)
		}
		return strings.ToLower(leftLabel) < strings.ToLower(rightLabel)
	})
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
	m.teamsImportMode = false
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

func (m *Model) openCurrentTeamHostEditor(ctx context.Context) error {
	host, ok := m.teamsCurrentHost()
	if !ok {
		return fmt.Errorf("no team host selected")
	}
	accessToken, err := m.teamsAccessToken(ctx)
	if err != nil {
		return err
	}
	detail, err := m.teamsClient.GetTeamHost(ctx, accessToken, host.ID)
	if err != nil {
		return err
	}

	keyType := detail.CredentialType
	if keyType == "private_key" {
		keyType = "pasted"
	}
	tagInput := strings.Join(detail.Tags, ", ")
	existingKey := ""
	if detail.CredentialMode == "shared" {
		existingKey = detail.SharedCredential
	}

	m.initAddHostForm(detail.Label, detail.Group, tagInput, detail.Hostname, detail.Username, fmt.Sprintf("%d", detail.Port), keyType, existingKey)
	m.formTeamHostID = detail.ID
	m.formTeamCredentialMode = detail.CredentialMode
	m.formTeamCredentialType = detail.CredentialType
	m.overlay = OverlayAddHost
	if detail.CredentialMode == "per_member" {
		m.err = fmt.Errorf("ℹ per-member credentials are not editable from Teams TUI yet")
	}
	return nil
}

func (m Model) handleImportConflictKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.teamsImportConflict == nil {
		m.overlay = OverlayNone
		return m, nil
	}

	switch msg.String() {
	case "esc":
		m.overlay = OverlayAddHost
		m.teamsImportConflict = nil
		return m, nil
	case "left", "h":
		if m.teamsImportConflict.Cursor > 0 {
			m.teamsImportConflict.Cursor--
		}
		return m, nil
	case "right", "l", "tab", "shift+tab":
		if m.teamsImportConflict.Cursor < 2 {
			m.teamsImportConflict.Cursor++
		} else {
			m.teamsImportConflict.Cursor = 0
		}
		return m, nil
	case "enter":
		accessToken, err := m.teamsAccessToken(context.Background())
		if err != nil {
			m.err = err
			return m, nil
		}
		req, err := m.buildTeamHostRequestFromPersonalHost(m.teamsImportConflict.PersonalHost)
		if err != nil {
			m.err = err
			return m, nil
		}
		switch m.teamsImportConflict.Cursor {
		case 0:
			if err := m.updateTeamHostFromRequest(context.Background(), accessToken, m.teamsImportConflict.ExistingHost.ID, req); err != nil {
				m.err = err
				return m, nil
			}
			if err := m.loadCurrentTeamHosts(context.Background()); err != nil {
				m.err = err
				return m, nil
			}
			label := strings.TrimSpace(req.Label)
			if label == "" {
				label = req.Hostname
			}
			m.err = fmt.Errorf("✓ Updated '%s' from personal host", label)
			m.teamsImportConflict = nil
			m.closeAddHostOverlay()
			return m, nil
		case 1:
			team, ok := m.teamsCurrentTeam()
			if !ok {
				m.err = fmt.Errorf("no team selected")
				return m, nil
			}
			if _, err := m.teamsClient.CreateTeamHost(context.Background(), accessToken, team.ID, req); err != nil {
				m.err = err
				return m, nil
			}
			if err := m.loadCurrentTeamHosts(context.Background()); err != nil {
				m.err = err
				return m, nil
			}
			label := strings.TrimSpace(req.Label)
			if label == "" {
				label = req.Hostname
			}
			m.err = fmt.Errorf("✓ Imported duplicate '%s'", label)
			m.teamsImportConflict = nil
			m.closeAddHostOverlay()
			return m, nil
		default:
			m.overlay = OverlayAddHost
			m.teamsImportConflict = nil
			return m, nil
		}
	}
	return m, nil
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
		return m.requestQuit()
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
		m.teamsImportMode = false
		m.overlay = OverlaySearch
		m.searchQuery = ""
		m.spotlightItems = m.buildSpotlightItems("")
		m.selectedIdx = 0
		return m, nil
	case "?", "shift+/":
		m.openTeamsCommandPalette()
		return m, nil
	case "a", "ctrl+n":
		groupPrefill := ""
		if host, ok := m.teamsCurrentHost(); ok {
			groupPrefill = host.Group
		}
		m.initAddHostForm("", groupPrefill, "", "", "", "22", "", "")
		m.overlay = OverlayAddHost
		m.formEditIdx = -1
		return m, nil
	case "e":
		if err := m.openCurrentTeamHostEditor(ctx); err != nil {
			m.err = err
		}
		return m, nil
	case "d", "delete":
		if _, ok := m.teamsCurrentHost(); !ok {
			m.err = fmt.Errorf("no team host selected")
			return m, nil
		}
		m.deleteCursor = 1
		m.overlay = OverlayDeleteHost
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
		host, ok := m.teamsCurrentHost()
		if !ok {
			m.err = fmt.Errorf("no team hosts yet")
			return m, nil
		}
		return m.connectToTeamHost(host)
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
