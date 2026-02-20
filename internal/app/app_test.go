package app

import (
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
)

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

func TestGetFilteredHosts(t *testing.T) {
	m := NewModel()
	m.hosts = []Host{
		{Hostname: "web-prod-1.example.com", Username: "ec2-user", Label: "web-prod-1"},
		{Hostname: "db-server.internal", Username: "ubuntu", Label: "db-server"},
		{Hostname: "staging.dev.local", Username: "deploy", Label: "staging"},
		{Hostname: "backup-nas.home", Username: "admin", Label: "backup-nas"},
	}
	m.searchInput = textinput.New()

	// Test case 1: No filter
	m.searchInput.SetValue("")
	filtered := m.getFilteredHosts()
	if len(filtered) != 4 {
		t.Errorf("Expected 4 hosts, got %d", len(filtered))
	}

	// Test case 2: Filter by hostname
	m.searchInput.SetValue("web")
	filtered = m.getFilteredHosts()
	if len(filtered) != 1 {
		t.Errorf("Expected 1 host, got %d", len(filtered))
	}
	if filtered[0].Hostname != "web-prod-1.example.com" {
		t.Errorf("Expected web-prod-1.example.com, got %s", filtered[0].Hostname)
	}

	// Test case 3: Filter by username
	m.searchInput.SetValue("ubuntu")
	filtered = m.getFilteredHosts()
	if len(filtered) != 1 {
		t.Errorf("Expected 1 host, got %d", len(filtered))
	}
	if filtered[0].Username != "ubuntu" {
		t.Errorf("Expected ubuntu, got %s", filtered[0].Username)
	}
}

func TestValidateForm(t *testing.T) {
	m := NewModel()

	// Helper to set up a basic valid form
	setupForm := func() {
		m.modalForm = m.newModalForm("myhost", "", "example.com", "user", "22", "ed25519", "")
	}

	// Test case 1: Valid form
	setupForm()
	err := m.validateForm()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Test case 2: Empty hostname
	setupForm()
	m.modalForm.hostnameInput.SetValue("")
	err = m.validateForm()
	if err == nil {
		t.Error("Expected error for empty hostname, got nil")
	}

	// Test case 3: Empty username
	setupForm()
	m.modalForm.usernameInput.SetValue("")
	err = m.validateForm()
	if err == nil {
		t.Error("Expected error for empty username, got nil")
	}

	// Test case 4: Invalid port
	setupForm()
	m.modalForm.portInput.SetValue("invalid")
	err = m.validateForm()
	if err == nil {
		t.Error("Expected error for invalid port, got nil")
	}

	// Test case 5: Empty port
	setupForm()
	m.modalForm.portInput.SetValue("")
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

func assertErr(msg string) error { return &testErr{msg: msg} }

type testErr struct{ msg string }

func (e *testErr) Error() string { return e.msg }
