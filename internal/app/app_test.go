package app

import (
	"strings"
	"testing"
	"time"

	"github.com/Vansh-Raja/SSHThing/internal/db"
	ssync "github.com/Vansh-Raja/SSHThing/internal/sync"
	"github.com/Vansh-Raja/SSHThing/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
)

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

func TestBuildTeamsMockViewParams_DefaultsToLoginScreen(t *testing.T) {
	m := NewModel()
	m.page = PageTeams

	p := m.buildTeamsActionViewParams()

	if p.Message != "Login / Sign up" {
		t.Fatalf("expected login message, got %q", p.Message)
	}
	if p.Options[0].Label != "Login" {
		t.Fatalf("expected first login option Login, got %+v", p.Options)
	}
	if p.Description != "Shared hosts for your team" {
		t.Fatalf("unexpected entry description: %q", p.Description)
	}
}

func TestTeamsEmptyStateShowsCreateOrJoin(t *testing.T) {
	m := NewModel()
	m.page = PageTeams
	m.teamsAuthed = true
	m.teamsHasTeam = false

	p := m.buildTeamsActionViewParams()
	if p.Message != "You're not part of a team yet" {
		t.Fatalf("unexpected empty-state message: %q", p.Message)
	}
	if len(p.Options) < 2 || p.Options[0].Label != "Create team" || p.Options[1].Label != "Join team" {
		t.Fatalf("unexpected empty-state options: %+v", p.Options)
	}
}

func TestTeamsHostsParamsDoNotReferenceWorkspaceOrVault(t *testing.T) {
	m := NewModel()
	m.page = PageTeams
	m.teamsAuthed = true
	m.teamsHasTeam = true
	m.resolveTeamsState()
	p := m.buildTeamsHostsViewParams()

	joined := strings.Join([]string{p.TeamName, p.Selected.ShareMode}, "\n")
	if strings.Contains(strings.ToLower(joined), "workspace") || strings.Contains(strings.ToLower(joined), "vault") {
		t.Fatalf("unexpected terminology leaked into teams hosts params: %s", joined)
	}
}

func TestTeamsHostsIncludeGroupedAndUngroupedItems(t *testing.T) {
	m := NewModel()
	m.teamsAuthed = true
	m.teamsHasTeam = true

	items := m.buildTeamsHostItems()
	if len(items) == 0 {
		t.Fatalf("expected team host items")
	}

	seenProduction := false
	seenStaging := false
	seenUngrouped := false
	for _, item := range items {
		if !item.IsGroup {
			continue
		}
		switch item.GroupName {
		case "Production":
			seenProduction = true
		case "Staging":
			seenStaging = true
		case "Ungrouped":
			seenUngrouped = true
		}
	}

	if !seenProduction || !seenStaging || !seenUngrouped {
		t.Fatalf("expected Production, Staging, and Ungrouped in team hosts")
	}
}

func TestTeamsBackFromMembersReturnsToHosts(t *testing.T) {
	m := NewModel()
	m.page = PageTeams
	m.overlay = OverlayNone
	m.teamsAuthed = true
	m.teamsHasTeam = true
	m.teamsState = teamsStateMembers

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m1 := updated.(Model)

	if m1.page != PageTeams {
		t.Fatalf("expected to stay on teams page, got %v", m1.page)
	}
	if m1.teamsState != teamsStateHosts {
		t.Fatalf("expected q from members to return to hosts, got %v", m1.teamsState)
	}
}

func TestTeamsInviteMemberAddsPendingMember(t *testing.T) {
	m := NewModel()
	m.page = PageTeams
	m.overlay = OverlayNone
	m.teamsAuthed = true
	m.teamsHasTeam = true
	m.teamsState = teamsStateInviteMember
	m.teamsInviteEmail = "new.person@acme.dev"
	m.teamsInviteRole = 2
	m.teamsManageFocus = 2

	before := len(m.teamsTeam.Members)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m1 := updated.(Model)

	if len(m1.teamsTeam.Members) != before+1 {
		t.Fatalf("expected invited member to be added")
	}
	if m1.teamsState != teamsStateMembers {
		t.Fatalf("expected invite flow to return to members, got %v", m1.teamsState)
	}
	last := m1.teamsTeam.Members[len(m1.teamsTeam.Members)-1]
	if last.Email != "new.person@acme.dev" || last.Status != "invited" {
		t.Fatalf("unexpected invited member: %+v", last)
	}
}

func TestTeamsManagementBackReturnsToMembers(t *testing.T) {
	m := NewModel()
	m.page = PageTeams
	m.overlay = OverlayNone
	m.teamsAuthed = true
	m.teamsHasTeam = true
	m.teamsState = teamsStateInviteMember

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m1 := updated.(Model)

	if m1.page != PageTeams {
		t.Fatalf("expected to stay on teams page, got %v", m1.page)
	}
	if m1.teamsState != teamsStateMembers {
		t.Fatalf("expected q from management screen to return to members, got %v", m1.teamsState)
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
