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

func TestRenderTeamsViewKeepsFooterWithVerboseHealth(t *testing.T) {
	renderer := Renderer{
		Theme: Themes[0],
		Icons: UnicodeIcons,
		W:     100,
		H:     16,
	}

	out := renderer.RenderTeamsView(TeamsHomeViewParams{
		Page:              4,
		SessionValid:      true,
		HasTeams:          true,
		HealthDisplayMode: "graph_values",
		CurrentTeam:       TeamsHomeTeamSummary{Name: "Acme", Slug: "acme", Role: "owner", TeamCount: 1, HostCount: 1},
		Items: []TeamsHomeListItem{
			{IsGroup: true, GroupName: "Work", HostCount: 1},
			{
				Label:              "GPU Server",
				Hostname:           "gpu.internal",
				Username:           "root",
				Port:               22,
				Group:              "Work",
				CredentialType:     "private_key",
				CredentialMode:     "per_member",
				LastConnectedLabel: "7 minutes ago",
				Notes:              "Vaani Deployment Server with a long note that should not push the footer away.",
				Selected:           true,
				Health: &HostHealthView{
					Status:       "online",
					StatusLabel:  "online",
					CheckedLabel: "7m ago",
					LatencyLabel: "1897ms",
					UptimeLabel:  "12d 19h",
					CPULabel:     "13%",
					RAMUsedPct:   23,
					DiskUsedPct:  58,
					GPULabel:     "NVIDIA L40S",
					SummaryLine:  "online · 7m ago · 1897ms",
					ResourceLine: "cpu 13% · ram 23% · disk 58% used",
					SystemLine:   "up 12d 19h · gpu NVIDIA L40S",
				},
			},
		},
	})

	if !strings.Contains(out, "q quit") {
		t.Fatalf("expected footer to stay visible")
	}
	if !strings.Contains(out, "Health") {
		t.Fatalf("expected detailed health card")
	}
}
