package config

import "testing"

func TestDefaultTeamsSettings(t *testing.T) {
	cfg := Default()
	if cfg.Version != 9 {
		t.Fatalf("expected version 9, got %d", cfg.Version)
	}
	if cfg.Teams.Enabled {
		t.Fatalf("expected teams disabled by default")
	}
	if cfg.Teams.APIBaseURL != "" || cfg.Teams.BrowserBaseURL != "" {
		t.Fatalf("expected empty teams URLs by default")
	}
	if !cfg.Teams.SessionCacheEnabled {
		t.Fatalf("expected teams session cache enabled by default")
	}
	if cfg.TeamsUI.Theme == "" || cfg.TeamsUI.IconSet == "" {
		t.Fatalf("expected teams UI defaults")
	}
	if cfg.UI.WrapLabels {
		t.Fatalf("expected personal wrap labels off by default")
	}
	if cfg.TeamsUI.WrapLabels {
		t.Fatalf("expected teams wrap labels off by default")
	}
	if cfg.UI.HealthDisplayMode != HealthDisplayGraphValues {
		t.Fatalf("expected graph+values health display by default, got %q", cfg.UI.HealthDisplayMode)
	}
	if cfg.Updates.ReleaseChannel != "stable" {
		t.Fatalf("expected stable release channel by default, got %q", cfg.Updates.ReleaseChannel)
	}
	if cfg.Updates.AutoApplyUpdates {
		t.Fatalf("expected auto apply updates off by default")
	}
	if cfg.Sync.Provider != SyncProviderOff {
		t.Fatalf("expected sync provider off by default, got %q", cfg.Sync.Provider)
	}
	if !cfg.Sync.Scope.Hosts || !cfg.Sync.Scope.Groups || !cfg.Sync.Scope.Credentials || !cfg.Sync.Scope.TokenDefinitions {
		t.Fatalf("expected portable sync scope enabled by default")
	}
	if cfg.Sync.Scope.Health || cfg.Sync.Scope.MountState {
		t.Fatalf("expected device-specific sync scope disabled by default")
	}
}

func TestLoadSavePersistsTeamsSettings(t *testing.T) {
	t.Setenv("SSHTHING_DATA_DIR", t.TempDir())

	cfg := Default()
	cfg.Teams.Enabled = true
	cfg.Teams.APIBaseURL = "https://api.example.com"
	cfg.Teams.BrowserBaseURL = "https://app.example.com"
	cfg.Teams.SessionCacheEnabled = false

	if err := Save(cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if !got.Teams.Enabled {
		t.Fatalf("expected teams enabled after reload")
	}
	if got.Teams.APIBaseURL != cfg.Teams.APIBaseURL {
		t.Fatalf("expected api base url %q, got %q", cfg.Teams.APIBaseURL, got.Teams.APIBaseURL)
	}
	if got.Teams.BrowserBaseURL != cfg.Teams.BrowserBaseURL {
		t.Fatalf("expected browser base url %q, got %q", cfg.Teams.BrowserBaseURL, got.Teams.BrowserBaseURL)
	}
	if got.Teams.SessionCacheEnabled != cfg.Teams.SessionCacheEnabled {
		t.Fatalf("expected session cache %v, got %v", cfg.Teams.SessionCacheEnabled, got.Teams.SessionCacheEnabled)
	}
}

func TestLoadSavePersistsHealthDisplayMode(t *testing.T) {
	t.Setenv("SSHTHING_DATA_DIR", t.TempDir())

	cfg := Default()
	cfg.UI.HealthDisplayMode = HealthDisplayValues
	if err := Save(cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}
	got, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if got.UI.HealthDisplayMode != HealthDisplayValues {
		t.Fatalf("expected values health display after reload, got %q", got.UI.HealthDisplayMode)
	}
}

func TestWithDefaultsMigratesTeamsVersion(t *testing.T) {
	cfg := Config{}
	cfg.Version = 2

	got := withDefaults(cfg)
	if got.Version != 9 {
		t.Fatalf("expected migration to version 9, got %d", got.Version)
	}
	if !got.Teams.SessionCacheEnabled {
		t.Fatalf("expected session cache enabled after migration")
	}
	if got.TeamsUI.Theme == "" || got.TeamsUI.IconSet == "" {
		t.Fatalf("expected teams ui defaults after migration")
	}
	if got.Updates.ReleaseChannel != "stable" {
		t.Fatalf("expected stable release channel after migration, got %q", got.Updates.ReleaseChannel)
	}
	if got.Updates.AutoApplyUpdates {
		t.Fatalf("expected auto apply updates off after migration")
	}
	if got.UI.HealthDisplayMode != HealthDisplayGraphValues {
		t.Fatalf("expected graph+values health display after migration, got %q", got.UI.HealthDisplayMode)
	}
	if got.Sync.Provider != SyncProviderOff {
		t.Fatalf("expected sync provider off after migration, got %q", got.Sync.Provider)
	}
}

func TestWithDefaultsMigratesLegacyUpdateETag(t *testing.T) {
	cfg := Config{}
	cfg.Version = 5
	cfg.Updates.ETagLatest = "legacy-etag"

	got := withDefaults(cfg)
	if got.Version != 9 {
		t.Fatalf("expected migration to version 9, got %d", got.Version)
	}
	if got.Updates.ETagStable != "legacy-etag" {
		t.Fatalf("expected legacy etag to migrate to stable slot, got %q", got.Updates.ETagStable)
	}
}

func TestWithDefaultsMigratesLegacyEnabledSyncToGitProvider(t *testing.T) {
	cfg := Config{}
	cfg.Version = 8
	cfg.Sync.Enabled = true
	cfg.Automation.SyncTokenDefinitions = true

	got := withDefaults(cfg)
	if got.Sync.Provider != SyncProviderGit {
		t.Fatalf("expected legacy enabled sync to become git provider, got %q", got.Sync.Provider)
	}
	if !got.Sync.Enabled {
		t.Fatalf("expected sync enabled for git provider")
	}
	if !got.Sync.Scope.TokenDefinitions {
		t.Fatalf("expected token definition scope to preserve legacy automation setting")
	}
}

func TestWithDefaultsMigratesV7HealthDetailsOffToMinimal(t *testing.T) {
	cfg := Config{}
	cfg.Version = 7
	cfg.UI.ShowHealthDetails = false

	got := withDefaults(cfg)
	if got.UI.HealthDisplayMode != HealthDisplayMinimal {
		t.Fatalf("expected minimal mode from disabled v7 health details, got %q", got.UI.HealthDisplayMode)
	}
}
