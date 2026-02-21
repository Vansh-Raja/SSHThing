package sync

import (
	"testing"
	"time"
)

func TestStatusStringIncludesStageWhenSyncing(t *testing.T) {
	m := &Manager{status: SyncStatusSyncing, stage: "pulling"}
	if got := m.StatusString(); got != "Syncing: pulling" {
		t.Fatalf("expected syncing status with stage, got %q", got)
	}
}

func TestStageStringReturnsCurrentStage(t *testing.T) {
	m := &Manager{stage: "exporting"}
	if got := m.StageString(); got != "exporting" {
		t.Fatalf("expected exporting stage, got %q", got)
	}
}

func TestComputeHostsPushed(t *testing.T) {
	now := time.Now()
	older := now.Add(-time.Hour)

	local := &SyncData{Hosts: []SyncHost{{ID: 1, UpdatedAt: now}, {ID: 2, UpdatedAt: now}}}
	remote := &SyncData{Hosts: []SyncHost{{ID: 1, UpdatedAt: older}}}

	if got := computeHostsPushed(local, remote); got != 2 {
		t.Fatalf("expected 2 pushed hosts, got %d", got)
	}

	if got := computeHostsPushed(local, nil); got != 2 {
		t.Fatalf("expected all local hosts pushed when remote missing, got %d", got)
	}
}
