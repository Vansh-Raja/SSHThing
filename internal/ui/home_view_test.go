package ui

import (
	"strings"
	"testing"
)

func TestRenderHomeViewKeepsFooterWithVerboseHealth(t *testing.T) {
	renderer := Renderer{
		Theme: Themes[0],
		Icons: UnicodeIcons,
		W:     96,
		H:     16,
	}

	out := renderer.RenderHomeView(HomeViewParams{
		Page:              0,
		HostCount:         1,
		HealthDisplayMode: "graph_values",
		Items: []HomeListItem{
			{
				Label:    "GPU Server",
				Hostname: "gpu.example.com",
				Username: "root",
				Port:     22,
				KeyType:  "private_key",
				Tags:     []string{"gpu", "cuda"},
				Health: &HostHealthView{
					Status:       "online",
					StatusLabel:  "online",
					CheckedLabel: "7m ago",
					LatencyLabel: "1897ms",
					UptimeLabel:  "12d 19h",
					CPULabel:     "13%",
					RAMLabel:     "48.3 GiB / 201.4 GiB (23%)",
					RAMUsedPct:   23,
					DiskLabel:    "98.3 GiB free / 232.8 GiB",
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

func TestRenderHomeViewCompactHealthWhenDetailsOff(t *testing.T) {
	renderer := Renderer{Theme: Themes[0], Icons: UnicodeIcons, W: 96, H: 24}
	out := renderer.RenderHomeView(HomeViewParams{
		Page:              0,
		HostCount:         1,
		HealthDisplayMode: "minimal",
		Items: []HomeListItem{{
			Label:    "api",
			Hostname: "api.example.com",
			Username: "ubuntu",
			KeyType:  "private_key",
			Health: &HostHealthView{
				Status:       "online",
				StatusLabel:  "online",
				SummaryLine:  "online · 1m ago · 20ms",
				ResourceLine: "cpu 10% · ram 20% · disk 30% used",
			},
		}},
	})

	if !strings.Contains(strings.ToLower(out), "health") {
		t.Fatalf("expected compact health line")
	}
	if !strings.Contains(out, "cpu") {
		t.Fatalf("expected minimal health graph labels")
	}
}

func TestRenderHomeViewHealthDisplayValuesMode(t *testing.T) {
	renderer := Renderer{Theme: Themes[0], Icons: UnicodeIcons, W: 120, H: 28}
	out := renderer.RenderHomeView(HomeViewParams{
		Page:              0,
		HostCount:         1,
		HealthDisplayMode: "values",
		Items: []HomeListItem{{
			Label:    "api",
			Hostname: "api.example.com",
			Username: "ubuntu",
			KeyType:  "private_key",
			Health: &HostHealthView{
				Status:       "online",
				StatusLabel:  "online",
				SummaryLine:  "online · 1m ago · 20ms",
				ResourceLine: "cpu 10% · ram 4.0 GiB/16.0 GiB · 25% · disk 90.0 GiB/100.0 GiB free",
				SystemLine:   "up 1d · gpu no",
			},
		}},
	})

	if !strings.Contains(out, "4.0 GiB/16.0 GiB") {
		t.Fatalf("expected exact RAM values in values mode")
	}
	if strings.Contains(out, "█") {
		t.Fatalf("did not expect graph bars in values mode")
	}
}

func TestRenderHomeViewHealthDisplayGraphValuesMode(t *testing.T) {
	renderer := Renderer{Theme: Themes[0], Icons: UnicodeIcons, W: 120, H: 28}
	out := renderer.RenderHomeView(HomeViewParams{
		Page:              0,
		HostCount:         1,
		HealthDisplayMode: "graph_values",
		Items: []HomeListItem{{
			Label:    "api",
			Hostname: "api.example.com",
			Username: "ubuntu",
			KeyType:  "private_key",
			Health: &HostHealthView{
				Status:         "online",
				StatusLabel:    "online",
				SummaryLine:    "online · 1m ago · 20ms",
				CPULabel:       "10%",
				CPUPercent:     10,
				RAMUsedPct:     25,
				RAMUsedLabel:   "4.0 GiB",
				RAMTotalLabel:  "16.0 GiB",
				DiskUsedPct:    10,
				DiskFreeLabel:  "90.0 GiB",
				DiskTotalLabel: "100.0 GiB",
			},
		}},
	})

	if !strings.Contains(out, "█") {
		t.Fatalf("expected graph bars in graph+values mode")
	}
	if !strings.Contains(out, "4.0 GiB/16.0 GiB") {
		t.Fatalf("expected exact values in graph+values mode")
	}
}

func TestRenderCommandLineShowsAutocompleteRows(t *testing.T) {
	renderer := Renderer{Theme: Themes[0], Icons: UnicodeIcons, W: 100, H: 28}
	out := renderer.RenderCommandLine(CommandLineView{
		Query:  "he",
		Cursor: 0,
		Items: []CommandLineItem{
			{Name: "health", Description: "refresh host health"},
			{Name: "help", Description: "show commands and shortcuts"},
			{Name: "delete", Description: "delete selected host", Danger: true},
			{Name: "sync", Description: "sync now", Disabled: true, DisabledReason: "sync is disabled"},
		},
	})
	for _, want := range []string{":he", ":health", "refresh host health", "tab complete"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in command line output", want)
		}
	}
}

func TestCommandLineHeightScalesOnRoomyTerminals(t *testing.T) {
	renderer := Renderer{Theme: Themes[0], Icons: UnicodeIcons, W: 120, H: 28}
	out := renderer.RenderCommandLine(CommandLineView{
		Query:  "",
		Cursor: 0,
		Items: []CommandLineItem{
			{Name: "add", Description: "add a host"},
			{Name: "edit", Description: "edit selected item"},
			{Name: "delete", Description: "delete selected item", Danger: true},
			{Name: "health", Description: "refresh host health"},
			{Name: "sync", Description: "sync personal hosts"},
		},
	})

	for _, want := range []string{":add", ":edit", ":delete", ":health"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected roomy terminal to show %q in suggestions:\n%s", want, out)
		}
	}
}

func TestRenderCommandLineUsesExactReservedHeight(t *testing.T) {
	renderer := Renderer{Theme: Themes[0], Icons: UnicodeIcons, W: 100, H: 28}
	view := CommandLineView{
		Query:  "e",
		Cursor: 0,
		Items: []CommandLineItem{
			{Name: "edit", Description: "edit selected item"},
			{Name: "delete", Description: "delete selected item", Danger: true},
			{Name: "help", Description: "show commands and shortcuts"},
		},
	}

	for _, height := range []int{1, 2, 4} {
		out := renderer.RenderCommandLineWithHeight(view, height)
		lines := strings.Split(out, "\n")
		if len(lines) != height {
			t.Fatalf("expected %d lines, got %d:\n%s", height, len(lines), out)
		}
		if !strings.Contains(out, ":e") {
			t.Fatalf("expected prompt in %d-line command output", height)
		}
	}
}

func TestRenderCommandLineScrollsVisibleSuggestions(t *testing.T) {
	renderer := Renderer{Theme: Themes[0], Icons: UnicodeIcons, W: 100, H: 28}
	view := CommandLineView{
		Query:  "",
		Cursor: 3,
		Items: []CommandLineItem{
			{Name: "add", Description: "add a host"},
			{Name: "edit", Description: "edit selected item"},
			{Name: "delete", Description: "delete selected item", Danger: true},
			{Name: "health", Description: "refresh host health"},
			{Name: "sync", Description: "sync personal hosts"},
		},
	}

	out := renderer.RenderCommandLineWithHeight(view, 4)
	if strings.Contains(out, ":add") || strings.Contains(out, ":edit") {
		t.Fatalf("expected old top suggestions to scroll out:\n%s", out)
	}
	if !strings.Contains(out, ":delete") || !strings.Contains(out, ":health") {
		t.Fatalf("expected selected suggestion window around cursor:\n%s", out)
	}

	strip := renderer.RenderCommandLineWithHeight(view, 2)
	if !strings.Contains(strip, ":health") {
		t.Fatalf("expected compact strip to include selected suggestion:\n%s", strip)
	}
}

func TestRenderHomeViewBudgetsCommandLineFooter(t *testing.T) {
	renderer := Renderer{Theme: Themes[0], Icons: UnicodeIcons, W: 100, H: 16}
	out := renderer.RenderHomeView(HomeViewParams{
		Page: 0,
		Items: []HomeListItem{{
			Label:    "api",
			Hostname: "api.example.com",
			Username: "ubuntu",
			KeyType:  "private_key",
			Health: &HostHealthView{
				Status:         "online",
				StatusLabel:    "online",
				SummaryLine:    "online · 1m ago · 20ms",
				CPULabel:       "10%",
				CPUPercent:     10,
				RAMUsedPct:     25,
				RAMUsedLabel:   "4.0 GiB",
				RAMTotalLabel:  "16.0 GiB",
				DiskUsedPct:    10,
				DiskFreeLabel:  "90.0 GiB",
				DiskTotalLabel: "100.0 GiB",
			},
		}},
		CommandLine: &CommandLineView{
			Query:  "ed",
			Cursor: 0,
			Items: []CommandLineItem{
				{Name: "edit", Description: "edit selected item"},
				{Name: "delete", Description: "delete selected item", Danger: true},
			},
		},
	})

	if !strings.Contains(out, ":ed") {
		t.Fatalf("expected command prompt to stay visible")
	}
	if !strings.Contains(out, ":edit") {
		t.Fatalf("expected command suggestion to stay visible")
	}
}
