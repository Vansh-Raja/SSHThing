package sync

import (
	"testing"
	"time"
)

func TestMergeIncludesGroupsAndPrefersNewer(t *testing.T) {
	now := time.Now()
	older := now.Add(-time.Hour)

	local := &SyncData{
		Version: CurrentSyncVersion,
		Groups: []SyncGroup{
			{Name: "Work", UpdatedAt: now},
		},
		Hosts: []SyncHost{
			{ID: 1, Hostname: "a", Username: "u", Port: 22, UpdatedAt: now},
		},
	}
	remote := &SyncData{
		Version: CurrentSyncVersion,
		Groups: []SyncGroup{
			{Name: "Work", UpdatedAt: older},
			{Name: "Home", UpdatedAt: now},
		},
		Hosts: []SyncHost{
			{ID: 1, Hostname: "old", Username: "u", Port: 22, UpdatedAt: older},
			{ID: 2, Hostname: "b", Username: "u", Port: 22, UpdatedAt: now},
		},
	}

	merged := Merge(local, remote)
	if merged == nil {
		t.Fatalf("expected merged data")
	}

	if len(merged.Groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(merged.Groups))
	}

	// Host ID 1 should come from local (newer)
	found := false
	for _, h := range merged.Hosts {
		if h.ID == 1 {
			found = true
			if h.Hostname != "a" {
				t.Fatalf("expected newer local host for ID 1, got %q", h.Hostname)
			}
		}
	}
	if !found {
		t.Fatalf("expected host with ID 1")
	}
}
