package teamcache

import (
	"testing"
	"time"

	"github.com/Vansh-Raja/SSHThing/internal/teams"
)

func TestCacheSaveLoadClear(t *testing.T) {
	t.Setenv("SSHTHING_DATA_DIR", t.TempDir())

	cache := Cache{
		CurrentTeamID: "team_123",
		Teams:         []teams.TeamSummary{{ID: "team_123", Name: "Acme", Slug: "acme", DisplayOrder: 0}},
		Hosts:         []teams.TeamHost{{ID: "host_1", TeamID: "team_123", Label: "api", Hostname: "api.internal", Username: "root", Port: 22}},
		LastFetchedAt: time.Now().UnixMilli(),
	}
	if err := Save(cache); err != nil {
		t.Fatalf("save cache: %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("load cache: %v", err)
	}
	if got.CurrentTeamID != "team_123" {
		t.Fatalf("expected current team to round-trip")
	}
	if len(got.Teams) != 1 || len(got.Hosts) != 1 {
		t.Fatalf("expected cache slices to round-trip")
	}

	if err := Clear(); err != nil {
		t.Fatalf("clear cache: %v", err)
	}
	empty, err := Load()
	if err != nil {
		t.Fatalf("load cleared cache: %v", err)
	}
	if empty.CurrentTeamID != "" || len(empty.Teams) != 0 || len(empty.Hosts) != 0 {
		t.Fatalf("expected cache to be empty after clear")
	}
}
