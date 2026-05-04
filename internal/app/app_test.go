package app

import (
	"testing"
	"time"

	"github.com/Vansh-Raja/SSHThing/internal/config"
	"github.com/Vansh-Raja/SSHThing/internal/db"
	ssync "github.com/Vansh-Raja/SSHThing/internal/sync"
	"github.com/Vansh-Raja/SSHThing/internal/teams"
	"github.com/Vansh-Raja/SSHThing/internal/teamssession"
	"github.com/Vansh-Raja/SSHThing/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
)

func TestNewModelKeepsExpiredRefreshableTeamsSession(t *testing.T) {
	t.Setenv("SSHTHING_DATA_DIR", t.TempDir())
	session := teamSessionForTests(time.Now().Add(-time.Hour))
	if err := teamssession.Save(session); err != nil {
		t.Fatalf("save teams session: %v", err)
	}

	m := NewModel()
	if m.teamsSession.RefreshToken != session.RefreshToken {
		t.Fatalf("expected expired refreshable teams session to be kept for refresh")
	}
}

func TestBuildSettingsItemsIncludesUpdateNote(t *testing.T) {
	m := NewModel()
	items := m.buildSettingsItems()

	found := false
	for _, item := range items {
		if item.Category == "updates" && item.Label == updateSettingsNoteLabel() {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("expected updates note row %q", updateSettingsNoteLabel())
	}
}

func TestBuildSettingsItemsOmitsTeamsRows(t *testing.T) {
	m := NewModel()
	items := m.buildSettingsItems()

	unexpected := map[string]bool{
		"enable teams":           true,
		"teams api base url":     true,
		"teams browser base url": true,
		"clear teams session":    true,
		"clear teams cache":      true,
	}
	for _, item := range items {
		if unexpected[item.Label] {
			t.Fatalf("did not expect teams settings row %q", item.Label)
		}
	}
}

func TestBuildSettingsItemsTeamsModeOmitsTeamManagementRows(t *testing.T) {
	m := NewModel()
	m.teamsSession = teamSessionForTests(time.Now().Add(time.Hour))
	m.appMode = appModeTeams
	items := m.buildSettingsItems()

	unexpected := map[string]bool{
		"current team":      true,
		"create team":       true,
		"rename team":       true,
		"delete team":       true,
		"move team earlier": true,
		"move team later":   true,
	}
	for _, item := range items {
		if unexpected[item.Label] {
			t.Fatalf("did not expect teams management settings row %q", item.Label)
		}
	}
	foundWrap := false
	for _, item := range items {
		if item.Label == "wrap labels" {
			foundWrap = true
			break
		}
	}
	if !foundWrap {
		t.Fatalf("expected teams settings to include wrap labels")
	}
	foundHealth := false
	for _, item := range items {
		if item.Label == "health display" {
			foundHealth = true
			break
		}
	}
	if !foundHealth {
		t.Fatalf("expected teams settings to include health display")
	}
}

func TestVisiblePagesPersonalModeOmitsTeams(t *testing.T) {
	m := NewModel()
	for _, page := range m.visiblePages() {
		if page == PageTeams {
			t.Fatalf("did not expect teams page in personal mode")
		}
	}
}

func TestVisiblePagesTeamsModeShowsTeamsShell(t *testing.T) {
	m := NewModel()
	m.teamsSession = teamSessionForTests(time.Now().Add(time.Hour))
	m.appMode = appModeTeams

	found := false
	for _, page := range m.visiblePages() {
		if page == PageTeams {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected teams page in teams mode")
	}
}

func TestToggleAppModeRequiresSignIn(t *testing.T) {
	t.Setenv("SSHTHING_DATA_DIR", t.TempDir())
	m := NewModel()
	m.toggleAppMode()
	if m.appMode != appModePersonal {
		t.Fatalf("expected to stay in personal mode")
	}
	if m.page != PageProfile {
		t.Fatalf("expected to be redirected to profile, got %d", m.page)
	}
}

func TestToggleAppModeSwitchesBetweenModes(t *testing.T) {
	m := NewModel()
	m.teamsSession = teamSessionForTests(time.Now().Add(time.Hour))
	m.syncProfileFromSession()

	m.toggleAppMode()
	if m.appMode != appModeTeams {
		t.Fatalf("expected teams mode")
	}
	if m.page != PageTeams {
		t.Fatalf("expected teams page, got %d", m.page)
	}

	m.toggleAppMode()
	if m.appMode != appModePersonal {
		t.Fatalf("expected personal mode")
	}
	if m.page != PageHome {
		t.Fatalf("expected home page, got %d", m.page)
	}
}

func TestCloudServiceBaseURLSupportsEnvOverride(t *testing.T) {
	t.Setenv("SSHTHING_CLOUD_BASE_URL", "https://cloud.example.com/")
	if got := cloudServiceBaseURL(); got != "https://cloud.example.com" {
		t.Fatalf("expected env override, got %q", got)
	}
}

func TestCloudServiceBaseURLUsesEmbeddedDefaultWhenEnvMissing(t *testing.T) {
	original := defaultCloudBaseURL
	defaultCloudBaseURL = "https://testsshthing.vanshraja.me/"
	t.Cleanup(func() {
		defaultCloudBaseURL = original
	})

	if got := cloudServiceBaseURL(); got != "https://testsshthing.vanshraja.me" {
		t.Fatalf("expected embedded default, got %q", got)
	}
}

func teamSessionForTests(expiresAt time.Time) teamssession.Session {
	return teamssession.Session{
		AccessToken:  "access",
		RefreshToken: "refresh",
		ExpiresAt:    expiresAt.UnixMilli(),
		UserName:     "Test User",
		UserEmail:    "test@example.com",
	}
}

func TestRebuildListItemsGrouped(t *testing.T) {
	m := NewModel()
	m.hosts = []Host{
		{ID: 1, Label: "db", Hostname: "db.example.com", Username: "ubuntu", GroupName: "Work"},
		{ID: 2, Label: "api", Hostname: "api.example.com", Username: "ubuntu", GroupName: "Work"},
		{ID: 3, Label: "nas", Hostname: "nas.local", Username: "admin", GroupName: ""},
	}
	m.groups = []string{"Work", "VMs"}
	m.collapsed = map[string]bool{"VMs": false}
	m.rebuildListItems()

	if len(m.listItems) == 0 {
		t.Fatalf("expected list items")
	}

	seenWork := false
	seenVMs := false
	seenUngrouped := false
	for _, it := range m.listItems {
		if it.Kind != ListItemGroup {
			continue
		}
		switch it.GroupName {
		case "Work":
			seenWork = true
		case "VMs":
			seenVMs = true
		case "Ungrouped":
			seenUngrouped = true
		}
	}

	if !seenWork || !seenVMs || !seenUngrouped {
		t.Fatalf("expected Work, VMs, and Ungrouped headers")
	}
}

func TestBuildSpotlightItemsGroupMatchIncludesHosts(t *testing.T) {
	m := NewModel()
	m.hosts = []Host{
		{ID: 1, Label: "db", Hostname: "db.example.com", Username: "ubuntu", GroupName: "Work"},
		{ID: 2, Label: "api", Hostname: "api.example.com", Username: "ubuntu", GroupName: "Work"},
		{ID: 3, Label: "lab", Hostname: "lab.example.com", Username: "me", GroupName: "Home"},
	}
	m.groups = []string{"Work", "Home"}
	m.rebuildListItems()

	items := m.buildSpotlightItems("wor")
	if len(items) == 0 {
		t.Fatalf("expected spotlight results")
	}

	if items[0].Kind != SpotlightItemGroup || items[0].GroupName != "Work" {
		t.Fatalf("expected first item to be Work group, got %+v", items[0])
	}

	foundWorkHost := false
	for _, it := range items {
		if it.Kind == SpotlightItemHost && it.Host.GroupName == "Work" {
			foundWorkHost = true
			break
		}
	}
	if !foundWorkHost {
		t.Fatalf("expected at least one Work host under group match")
	}
}

func TestBuildSpotlightItems_TagSearchMatchesHost(t *testing.T) {
	m := NewModel()
	m.hosts = []Host{
		{ID: 1, Label: "gpu-box", Hostname: "gpu.local", Username: "admin", GroupName: "Lab", Tags: []string{"gpu", "cuda"}},
		{ID: 2, Label: "web", Hostname: "web.local", Username: "ubuntu", GroupName: "Prod", Tags: []string{"nginx"}},
	}
	m.groups = []string{"Lab", "Prod"}
	m.rebuildListItems()

	items := m.buildSpotlightItems("#gpu")
	if len(items) == 0 {
		t.Fatalf("expected spotlight results for tag query")
	}

	found := false
	for _, it := range items {
		if it.Kind == SpotlightItemHost && it.Host.ID == 1 {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected gpu host in spotlight results")
	}
}

func TestBuildSpotlightItems_VirtualGroupTagMatchesHost(t *testing.T) {
	m := NewModel()
	m.hosts = []Host{
		{ID: 1, Label: "api", Hostname: "api.example.com", Username: "ubuntu", GroupName: "Work"},
	}
	m.groups = []string{"Work"}
	m.rebuildListItems()

	items := m.buildSpotlightItems("#work")
	if len(items) == 0 {
		t.Fatalf("expected spotlight results for virtual group tag query")
	}

	foundHost := false
	for _, it := range items {
		if it.Kind == SpotlightItemHost && it.Host.ID == 1 {
			foundHost = true
			break
		}
	}
	if !foundHost {
		t.Fatalf("expected grouped host for virtual group tag query")
	}
}

func TestHomeShiftRStartsAllHostHealthRefresh(t *testing.T) {
	t.Setenv("SSHTHING_DATA_DIR", t.TempDir())
	store, err := db.Init("testpassword123")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer store.Close()

	for _, host := range []*db.HostModel{
		{Label: "one", Hostname: "one.example.com", Username: "ubuntu", Port: 22, KeyType: ""},
		{Label: "two", Hostname: "two.example.com", Username: "ubuntu", Port: 22, KeyType: ""},
	} {
		if err := store.CreateHost(host, ""); err != nil {
			t.Fatalf("CreateHost failed: %v", err)
		}
	}

	m := NewModel()
	m.store = store
	m.overlay = OverlayNone
	m.page = PageHome
	m.loadHosts()

	updated, cmd := m.handleHomeKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("R")})
	got := updated.(Model)
	if cmd == nil {
		t.Fatalf("expected health probe command")
	}
	if !got.healthChecking {
		t.Fatalf("expected health refresh to be active")
	}
	if got.healthTotal != 2 {
		t.Fatalf("expected 2 health targets, got %d", got.healthTotal)
	}
	if got.healthInFlight != 2 || len(got.healthQueue) != 0 {
		t.Fatalf("expected two in-flight probes and empty queue, inFlight=%d queue=%d", got.healthInFlight, len(got.healthQueue))
	}
	for _, host := range got.hosts {
		result, ok := got.healthResults[localHealthKey(host.ID)]
		if !ok {
			t.Fatalf("missing checking result for host %d", host.ID)
		}
		if result.Status != "checking" {
			t.Fatalf("expected checking status, got %q", result.Status)
		}
	}
}

func TestHealthDisplaySettingCyclesInPersonalAndTeamsMode(t *testing.T) {
	m := NewModel()
	m.cfg.UI.HealthDisplayMode = config.HealthDisplayGraphValues
	items := m.buildSettingsItems()
	idx := -1
	for i, item := range items {
		if item.Label == "health display" {
			idx = i
			break
		}
	}
	if idx < 0 {
		t.Fatalf("expected personal settings to include health display")
	}
	m.applySettingChange(idx, "toggle")
	if m.cfg.UI.HealthDisplayMode != config.HealthDisplayMinimal {
		t.Fatalf("expected health display to cycle to minimal, got %q", m.cfg.UI.HealthDisplayMode)
	}

	m.appMode = appModeTeams
	items = m.buildSettingsItems()
	idx = -1
	for i, item := range items {
		if item.Label == "health display" {
			idx = i
			break
		}
	}
	if idx < 0 {
		t.Fatalf("expected teams settings to include health display")
	}
	m.applySettingChange(idx, "toggle")
	if m.cfg.UI.HealthDisplayMode != config.HealthDisplayValues {
		t.Fatalf("expected teams setting to cycle shared health display to values, got %q", m.cfg.UI.HealthDisplayMode)
	}
}

func TestColonOpensCommandLineFromHome(t *testing.T) {
	m := NewModel()
	m.overlay = OverlayNone
	m.page = PageHome

	updated, cmd := m.handlePageKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(":")})
	got := updated.(Model)
	if cmd != nil {
		t.Fatalf("did not expect command")
	}
	if len(got.commandItems) == 0 {
		t.Fatalf("expected command suggestions")
	}
	if !got.commandModeActive() {
		t.Fatalf("expected command mode")
	}
}

func TestColonTogglesCommandLineClosed(t *testing.T) {
	m := NewModel()
	m.overlay = OverlayNone
	m.page = PageHome

	updated, _ := m.handlePageKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(":")})
	open := updated.(Model)
	if !open.commandModeActive() {
		t.Fatalf("expected command line to open")
	}

	updated, _ = open.handlePageKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(":")})
	closed := updated.(Model)
	if closed.commandModeActive() {
		t.Fatalf("expected second colon to close command line")
	}
}

func TestCommandLineExactSettingsCommand(t *testing.T) {
	m := NewModel()
	m.page = PageHome
	m.commandQuery = "settings"
	m.commandItems = m.buildCommandItems(m.commandQuery)

	updated, _ := m.handleCommandLineKeys(tea.KeyMsg{Type: tea.KeyEnter})
	got := updated.(Model)
	if got.commandModeActive() {
		t.Fatalf("expected command mode to close")
	}
	if got.page != PageSettings {
		t.Fatalf("expected settings page, got %d", got.page)
	}
}

func TestColonDoesNotOpenCommandLineInsideSettingsFilter(t *testing.T) {
	m := NewModel()
	m.overlay = OverlayNone
	m.page = PageSettings
	m.settingsSearching = true

	updated, _ := m.handlePageKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(":")})
	got := updated.(Model)
	if got.commandModeActive() {
		t.Fatalf("did not expect command mode while typing in settings filter")
	}
	if got.settingsFilter != ":" {
		t.Fatalf("expected colon to be typed into filter, got %q", got.settingsFilter)
	}
}

func TestCommandLineTabAutocompletesSelectedCommand(t *testing.T) {
	m := NewModel()
	m.page = PageHome
	m.commandQuery = "he"
	m.commandItems = m.buildCommandItems(m.commandQuery)

	updated, _ := m.handleCommandLineKeys(tea.KeyMsg{Type: tea.KeyTab})
	got := updated.(Model)
	if got.commandQuery != "help" && got.commandQuery != "health" {
		t.Fatalf("expected autocomplete to fill selected command, got %q", got.commandQuery)
	}
}

func TestCommandLineDeleteOpensConfirmation(t *testing.T) {
	t.Setenv("SSHTHING_DATA_DIR", t.TempDir())
	store, err := db.Init("testpassword123")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer store.Close()
	if err := store.CreateHost(&db.HostModel{Label: "api", Hostname: "api.example.com", Username: "ubuntu", Port: 22, KeyType: ""}, ""); err != nil {
		t.Fatalf("CreateHost failed: %v", err)
	}

	m := NewModel()
	m.store = store
	m.page = PageHome
	m.loadHosts()
	for i, item := range m.listItems {
		if item.Kind == ListItemHost {
			m.selectedIdx = i
			break
		}
	}
	m.commandQuery = "delete"
	m.commandItems = m.buildCommandItems(m.commandQuery)

	updated, _ := m.handleCommandLineKeys(tea.KeyMsg{Type: tea.KeyEnter})
	got := updated.(Model)
	if got.overlay != OverlayDeleteHost {
		t.Fatalf("expected delete confirmation overlay, got %d", got.overlay)
	}
	if len(got.hosts) != 1 {
		t.Fatalf("delete command should not delete immediately")
	}
}

func TestLoginHealthRefreshSilentWhenNoHosts(t *testing.T) {
	t.Setenv("SSHTHING_DATA_DIR", t.TempDir())
	m := NewModel()
	store, err := db.Init("testpassword123")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer store.Close()
	m.store = store
	m.loadHosts()

	cmd := m.beginPersonalHealthRefreshWithOptions(healthRefreshOptions{SilentIfEmpty: true, Source: "login"})
	if cmd != nil {
		t.Fatalf("did not expect health command with no hosts")
	}
	if m.err != nil {
		t.Fatalf("did not expect no-host error during silent login refresh, got %v", m.err)
	}

	cmd = m.beginPersonalHealthRefresh()
	if cmd != nil {
		t.Fatalf("did not expect manual health command with no hosts")
	}
	if m.err == nil || m.err.Error() != "no hosts to refresh" {
		t.Fatalf("expected manual no-host error, got %v", m.err)
	}
}

func TestTeamsAutoHealthRefreshRunsOnlyOnce(t *testing.T) {
	m := NewModel()
	m.teamsSession = teamSessionForTests(time.Now().Add(time.Hour))
	m.profileDisplayName = "Test User"
	m.profileEmail = "test@example.com"
	m.teamsCurrentTeamID = "team_1"
	m.teamsItems = []teams.TeamHost{
		{ID: "host_1", Label: "GPU", Hostname: "gpu.example.com", Username: "root", Port: 22},
		{ID: "host_2", Label: "CPU", Hostname: "cpu.example.com", Username: "root", Port: 22},
	}
	m.teamsCursor = 0
	m.appMode = appModeTeams
	m.page = PageTeams
	m.teamsClient = nil

	cmd := m.maybeAutoRefreshTeamsHealthOnEnter()
	if cmd == nil {
		t.Fatalf("expected first teams entry to schedule health refresh")
	}
	if !m.teamsHealthAutoRefreshed["team_1"] {
		t.Fatalf("expected teams auto refresh to be marked done")
	}
	if !m.healthChecking || m.healthTotal != 2 {
		t.Fatalf("expected all team hosts health refresh, checking=%v total=%d", m.healthChecking, m.healthTotal)
	}

	cmd = m.maybeAutoRefreshTeamsHealthOnEnter()
	if cmd != nil {
		t.Fatalf("did not expect second teams entry to schedule health refresh")
	}
}

func TestValidateForm(t *testing.T) {
	m := NewModel()

	setupForm := func() {
		m.initAddHostForm("myhost", "", "", "example.com", "user", "22", "ed25519", "")
	}

	// Test case 1: Valid form
	setupForm()
	err := m.validateForm()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Test case 2: Empty hostname
	setupForm()
	m.formFields[ui.FFHostname].Value = ""
	err = m.validateForm()
	if err == nil {
		t.Error("Expected error for empty hostname, got nil")
	}

	// Test case 3: Empty username
	setupForm()
	m.formFields[ui.FFUsername].Value = ""
	err = m.validateForm()
	if err == nil {
		t.Error("Expected error for empty username, got nil")
	}

	// Test case 4: Invalid port
	setupForm()
	m.formFields[ui.FFPort].Value = "invalid"
	err = m.validateForm()
	if err == nil {
		t.Error("Expected error for invalid port, got nil")
	}

	// Test case 5: Empty port
	setupForm()
	m.formFields[ui.FFPort].Value = ""
	err = m.validateForm()
	if err == nil {
		t.Error("Expected error for empty port, got nil")
	}
}

func TestInitAddHostFormStartsInNavigationMode(t *testing.T) {
	m := NewModel()
	m.initAddHostForm("", "", "", "", "", "22", "", "")

	if m.formFocus != ui.FFLabel {
		t.Fatalf("expected label focus, got %d", m.formFocus)
	}
	if m.formEditing {
		t.Fatalf("expected add-host form to start in navigation mode")
	}
}

func TestAddHostTextFieldsRequireEnterBeforeEditing(t *testing.T) {
	m := NewModel()
	m.initAddHostForm("", "", "", "", "", "22", "", "")
	m.formFocus = ui.FFHostname

	updated, _ := m.handleAddHostKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	got := updated.(Model)

	if got.formEditing {
		t.Fatalf("expected typing to be ignored before edit mode")
	}
	if got.formFields[ui.FFHostname].Value != "" {
		t.Fatalf("expected hostname to remain empty before edit mode, got %q", got.formFields[ui.FFHostname].Value)
	}

	updated, _ = got.handleAddHostKeys(tea.KeyMsg{Type: tea.KeyEnter})
	got = updated.(Model)
	updated, _ = got.handleAddHostKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	got = updated.(Model)

	if !got.formEditing {
		t.Fatalf("expected enter to enable edit mode")
	}
	if got.formFields[ui.FFHostname].Value != "a" {
		t.Fatalf("expected hostname field to receive typed rune in edit mode, got %q", got.formFields[ui.FFHostname].Value)
	}
}

func TestAddHostPrivateKeyEnterOpensPopupEditor(t *testing.T) {
	m := NewModel()
	m.initAddHostForm("", "", "", "", "", "22", "pasted", "")
	m.formFocus = ui.FFAuthDet

	updated, _ := m.handleAddHostKeys(tea.KeyMsg{Type: tea.KeyEnter})
	got := updated.(Model)

	if got.overlay != OverlayKeyEditor {
		t.Fatalf("expected key field to open editor overlay, got %d", got.overlay)
	}
}

func TestPrivateKeyPopupPastePreservesMultilineText(t *testing.T) {
	m := NewModel()
	m.initAddHostForm("", "", "", "", "", "22", "pasted", "")
	m.formFocus = ui.FFAuthDet

	updated, _ := m.handleAddHostKeys(tea.KeyMsg{Type: tea.KeyEnter})
	got := updated.(Model)

	pasted := "-----BEGIN OPENSSH PRIVATE KEY-----\nline-one\nline-two\n-----END OPENSSH PRIVATE KEY-----"
	updated, _ = got.handlePrivateKeyEditorKeys(tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune(pasted),
		Paste: true,
	})
	got = updated.(Model)

	if got.formFields[ui.FFAuthDet].Value != pasted {
		t.Fatalf("expected pasted multiline key to be preserved, got %q", got.formFields[ui.FFAuthDet].Value)
	}
}

func TestAddHostPrivateKeyEscDiscardsPopupEdit(t *testing.T) {
	m := NewModel()
	original := "original\nprivate\nkey"
	m.initAddHostForm("", "", "", "", "", "22", "pasted", original)
	m.formFocus = ui.FFAuthDet
	m.formEditing = false

	updated, _ := m.handleAddHostKeys(tea.KeyMsg{Type: tea.KeyEnter})
	got := updated.(Model)
	if got.overlay != OverlayKeyEditor {
		t.Fatalf("expected key editor overlay after enter, got %d", got.overlay)
	}
	got.formKeyEditor.SetValue("changed")
	got.syncFormKeyFieldFromEditor()

	updated, _ = got.handlePrivateKeyEditorKeys(tea.KeyMsg{Type: tea.KeyEsc})
	got = updated.(Model)

	if got.overlay != OverlayAddHost {
		t.Fatalf("expected escape to return to add host overlay, got %d", got.overlay)
	}
	if got.formFields[ui.FFAuthDet].Value != original {
		t.Fatalf("expected escape to discard edit and restore original key, got %q", got.formFields[ui.FFAuthDet].Value)
	}
}

func TestClearErrMsgClearsOnlyMatchingSequence(t *testing.T) {
	m := NewModel()
	m.err = assertErr("test error")
	m.errSeq = 2

	updated, _ := m.Update(clearErrMsg{seq: 1})
	m1 := updated.(Model)
	if m1.err == nil {
		t.Fatalf("expected error to remain when sequence does not match")
	}

	updated, _ = m1.Update(clearErrMsg{seq: 2})
	m2 := updated.(Model)
	if m2.err != nil {
		t.Fatalf("expected error cleared when sequence matches")
	}
}

func TestAutoClearDuration(t *testing.T) {
	if got := autoClearDuration("✓ Host updated"); got != 5*time.Second {
		t.Fatalf("expected success duration 5s, got %v", got)
	}
	if got := autoClearDuration("⚠ mount failed"); got != 10*time.Second {
		t.Fatalf("expected error duration 10s, got %v", got)
	}
}

func TestParseTagInput_NormalizesAndDedupes(t *testing.T) {
	got := db.ParseTagInput("#CPU, gpu, Gpu, ec2-prod, !!!, ec2")
	if len(got) != 4 {
		t.Fatalf("expected 4 tags, got %v", got)
	}
	if got[0] != "cpu" || got[1] != "gpu" || got[2] != "ec2prod" || got[3] != "ec2" {
		t.Fatalf("unexpected normalized tags: %v", got)
	}
}

func TestSyncAnimTickAdvancesProgress(t *testing.T) {
	m := NewModel()
	m.syncing = true
	m.syncRunID = 7
	m.syncProgress = 0.10

	updated, cmd := m.Update(syncAnimTickMsg{runID: 7})
	m1 := updated.(Model)

	if !m1.syncing {
		t.Fatalf("expected syncing to remain true")
	}
	if m1.syncAnimFrame == 0 {
		t.Fatalf("expected sync animation frame to advance")
	}
	if m1.syncProgress <= 0.10 {
		t.Fatalf("expected sync progress to increase, got %f", m1.syncProgress)
	}
	if cmd == nil {
		t.Fatalf("expected follow-up tick command")
	}
}

func TestSyncFinishedMsgSuccessSetsNotice(t *testing.T) {
	m := NewModel()
	m.syncing = true
	m.syncRunID = 3

	updated, _ := m.Update(syncFinishedMsg{runID: 3, result: &ssync.SyncResult{Success: true, HostsPulled: 2, HostsPushed: 1}})
	m1 := updated.(Model)

	if m1.syncing {
		t.Fatalf("expected syncing to stop")
	}
	if m1.err == nil || m1.err.Error() != "✓ Sync: ↓2 ↑1" {
		t.Fatalf("expected success sync notice, got %v", m1.err)
	}
}

func TestSyncFinishedMsgStaleIgnored(t *testing.T) {
	m := NewModel()
	m.syncing = true
	m.syncRunID = 4
	m.err = assertErr("existing")

	updated, _ := m.Update(syncFinishedMsg{runID: 3, result: &ssync.SyncResult{Success: true}})
	m1 := updated.(Model)

	if !m1.syncing {
		t.Fatalf("expected stale sync completion to be ignored")
	}
	if m1.err == nil || m1.err.Error() != "existing" {
		t.Fatalf("expected existing error to remain, got %v", m1.err)
	}
}

func assertErr(msg string) error { return &testErr{msg: msg} }

type testErr struct{ msg string }

func (e *testErr) Error() string { return e.msg }
