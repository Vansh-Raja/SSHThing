package ui

import (
	"strings"
	"testing"
)

func TestRenderTeamsViewDoesNotLeakMaskedHostname(t *testing.T) {
	renderer := Renderer{
		Theme: Themes[0],
		Icons: UnicodeIcons,
		W:     120,
		H:     40,
	}

	out := renderer.RenderTeamsView(TeamsHomeViewParams{
		Page:         4,
		SessionValid: true,
		HasTeams:     true,
		CurrentTeam:  TeamsHomeTeamSummary{Name: "Acme", Slug: "acme", TeamCount: 1, HostCount: 1},
		Items: []TeamsHomeListItem{
			{IsGroup: true, GroupName: "Ungrouped", HostCount: 1},
			{
				Label:    "app",
				Hostname: "",
				Username: "root",
				Port:     22,
				Selected: true,
			},
		},
	})

	if strings.Contains(out, "secret.internal") {
		t.Fatalf("render unexpectedly leaked masked hostname")
	}
	if !strings.Contains(out, "app") {
		t.Fatalf("expected host label in output")
	}
}

func TestRenderTeamsViewShowsGroupedHostsAndFooter(t *testing.T) {
	renderer := Renderer{
		Theme: Themes[0],
		Icons: UnicodeIcons,
		W:     120,
		H:     24,
	}

	out := renderer.RenderTeamsView(TeamsHomeViewParams{
		Page:         4,
		SessionValid: true,
		HasTeams:     true,
		CurrentTeam:  TeamsHomeTeamSummary{Name: "Acme", Slug: "acme", Role: "owner", TeamCount: 2, HostCount: 2},
		Items: []TeamsHomeListItem{
			{IsGroup: true, GroupName: "Work", HostCount: 1},
			{Label: "api", Hostname: "api.internal", Username: "root", Port: 22, Group: "Work", Selected: true},
			{IsGroup: true, GroupName: "Ungrouped", HostCount: 1},
			{Label: "db", Hostname: "db.internal", Username: "ubuntu", Port: 2222, Group: "Ungrouped"},
		},
		StatusLines: []TeamsHomeStatusLine{{Kind: "info", Message: "refresh complete"}},
	})

	for _, want := range []string{"Work", "Ungrouped", "api", "enter connect", "q quit", "refresh complete"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected output to contain %q", want)
		}
	}
}
