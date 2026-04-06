package app

import (
	"testing"
	"time"

	"github.com/Vansh-Raja/SSHThing/internal/db"
	ssync "github.com/Vansh-Raja/SSHThing/internal/sync"
	"github.com/Vansh-Raja/SSHThing/internal/ui"
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
