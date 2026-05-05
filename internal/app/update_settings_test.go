package app

import (
	"testing"

	"github.com/Vansh-Raja/SSHThing/internal/update"
)

func TestBuildSettingsItemsIncludesBetaUpdateRows(t *testing.T) {
	m := NewModel()
	items := m.buildSettingsItems()

	found := map[string]bool{
		"beta releases":      false,
		"auto apply updates": false,
		"feed":               false,
	}
	for _, item := range items {
		if _, ok := found[item.Label]; ok && item.Category == "updates" {
			found[item.Label] = true
		}
	}
	for label, ok := range found {
		if !ok {
			t.Fatalf("expected updates row %q", label)
		}
	}
}

func TestApplySettingChangeTogglesBetaUpdateSettings(t *testing.T) {
	m := NewModel()
	m.cfg.Updates.ReleaseChannel = "stable"
	m.cfg.Updates.AutoApplyUpdates = false

	items := m.buildSettingsItems()
	betaIdx := -1
	autoIdx := -1
	for i, item := range items {
		switch item.Label {
		case "beta releases":
			betaIdx = i
		case "auto apply updates":
			autoIdx = i
		}
	}
	if betaIdx < 0 || autoIdx < 0 {
		t.Fatalf("expected update setting rows, got %+v", items)
	}

	m.applySettingChange(betaIdx, "toggle")
	if m.cfg.Updates.ReleaseChannel != "beta" {
		t.Fatalf("expected beta release channel after toggle, got %q", m.cfg.Updates.ReleaseChannel)
	}

	m.applySettingChange(autoIdx, "toggle")
	if !m.cfg.Updates.AutoApplyUpdates {
		t.Fatalf("expected auto apply updates enabled after toggle")
	}
}

func TestUpdateCheckedMsgAutoAppliesBetaInstallerUpdate(t *testing.T) {
	m := NewModel()
	m.updateRunID = 7
	m.cfg.Updates.ReleaseChannel = "beta"
	m.cfg.Updates.AutoApplyUpdates = true

	msg := updateCheckedMsg{
		runID: 7,
		result: &update.CheckResult{
			LatestTag:       "v0.10.0-beta.1",
			LatestVersion:   "0.10.0-beta.1",
			UpdateAvailable: true,
			ReleaseChannel:  update.ReleaseChannelBeta,
			ApplyMode:       update.ApplyModeInstaller,
		},
	}

	updated, cmd := m.Update(msg)
	got := updated.(Model)
	if !got.updateApplying {
		t.Fatalf("expected auto-apply to start")
	}
	if cmd == nil {
		t.Fatalf("expected auto-apply command")
	}
}
