package config

import "testing"

func TestDefaultTeamsSettings(t *testing.T) {
	cfg := Default()
	if cfg.Version != 4 {
		t.Fatalf("expected version 4, got %d", cfg.Version)
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

func TestWithDefaultsMigratesTeamsVersion(t *testing.T) {
	cfg := Config{}
	cfg.Version = 2

	got := withDefaults(cfg)
	if got.Version != 4 {
		t.Fatalf("expected migration to version 4, got %d", got.Version)
	}
	if !got.Teams.SessionCacheEnabled {
		t.Fatalf("expected session cache enabled after migration")
	}
	if got.TeamsUI.Theme == "" || got.TeamsUI.IconSet == "" {
		t.Fatalf("expected teams ui defaults after migration")
	}
}
