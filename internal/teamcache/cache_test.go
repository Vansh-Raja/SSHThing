package teamcache

import (
	"testing"
	"time"

	"github.com/Vansh-Raja/SSHThing/internal/teams"
)

func TestCacheStructHoldsTeamData(t *testing.T) {
	now := time.Now()
	c := Cache{
		TeamSummary:   &teams.TeamSummary{ID: "team_1", Name: "Acme"},
		Hosts:         []teams.Host{{ID: "host_1", Label: "prod-api", ShareMode: teams.ShareModeHostOnly}},
		Members:       []teams.Member{{ID: "member_1", DisplayName: "Maya", Role: teams.RoleOwner}},
		LastFetchedAt: now,
	}
	if c.TeamSummary == nil || c.TeamSummary.Name != "Acme" {
		t.Fatalf("expected team summary to be preserved")
	}
	if len(c.Hosts) != 1 || len(c.Members) != 1 {
		t.Fatalf("expected cache collections to be preserved")
	}
}
