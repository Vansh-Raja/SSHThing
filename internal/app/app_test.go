package app

import (
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
)

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
		m.modalForm = m.newModalForm("myhost", "example.com", "user", "22", "ed25519", "")
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
