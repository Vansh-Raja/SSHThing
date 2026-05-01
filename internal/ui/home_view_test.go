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

	if !strings.Contains(out, "health") {
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
