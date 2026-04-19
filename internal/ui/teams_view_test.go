package ui

import (
	"strings"
	"testing"

	"github.com/Vansh-Raja/SSHThing/internal/teams"
)

func TestRenderTeamsViewDoesNotLeakMaskedHostname(t *testing.T) {
	renderer := Renderer{
		Theme: Themes[0],
		Icons: UnicodeIcons,
		W:     120,
		H:     40,
	}

	out := renderer.RenderTeamsView(TeamsViewParams{
		Page:         4,
		State:        1,
		SessionValid: true,
		CurrentTeam:  teams.TeamSummary{ID: "team_1", Name: "Acme", Slug: "acme", DisplayOrder: 0},
		Hosts: []teams.TeamHost{{
			ID:       "host_1",
			TeamID:   "team_1",
			Label:    "app",
			Hostname: "",
			Username: "root",
			Port:     22,
		}},
	})

	if strings.Contains(out, "secret.internal") {
		t.Fatalf("render unexpectedly leaked masked hostname")
	}
	if !strings.Contains(out, "app") {
		t.Fatalf("expected resource label in output")
	}
}
