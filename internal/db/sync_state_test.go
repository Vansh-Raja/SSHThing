package db

import (
	"testing"
	"time"
)

func TestSyncMetadataTablesAndCRUD(t *testing.T) {
	t.Setenv("SSHTHING_DATA_DIR", t.TempDir())
	store, err := Init("pw")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer store.Close()

	now := time.Now().UTC().Truncate(time.Second)
	if err := store.UpsertSyncItemState(SyncItemState{
		ItemType:       "host",
		SyncID:         "sync-1",
		LocalRevision:  "l1",
		RemoteRevision: "r1",
		LocalUpdatedAt: &now,
		Dirty:          true,
	}); err != nil {
		t.Fatalf("UpsertSyncItemState failed: %v", err)
	}

	got, ok, err := store.GetSyncItemState("host", "sync-1")
	if err != nil {
		t.Fatalf("GetSyncItemState failed: %v", err)
	}
	if !ok || got.ItemType != "host" || got.SyncID != "sync-1" || !got.Dirty || got.RemoteRevision != "r1" {
		t.Fatalf("unexpected sync item state: ok=%v got=%+v", ok, got)
	}

	dirty, err := store.ListDirtySyncItems()
	if err != nil {
		t.Fatalf("ListDirtySyncItems failed: %v", err)
	}
	if len(dirty) != 1 || dirty[0].SyncID != "sync-1" {
		t.Fatalf("unexpected dirty states: %+v", dirty)
	}

	if err := store.ClearSyncItemDirty("host", "sync-1", "r2", now); err != nil {
		t.Fatalf("ClearSyncItemDirty failed: %v", err)
	}
	got, ok, err = store.GetSyncItemState("host", "sync-1")
	if err != nil {
		t.Fatalf("GetSyncItemState after clear failed: %v", err)
	}
	if !ok || got.Dirty || got.RemoteRevision != "r2" || got.LastPushedAt == nil {
		t.Fatalf("expected clean pushed state, got ok=%v state=%+v", ok, got)
	}

	if err := store.UpsertSyncProviderState(SyncProviderState{
		Provider:           "convex",
		VaultID:            "vault-1",
		LastPulledRevision: "10",
		LastPushedRevision: "9",
		LastSyncAt:         &now,
	}); err != nil {
		t.Fatalf("UpsertSyncProviderState failed: %v", err)
	}
	provider, ok, err := store.GetSyncProviderState("convex")
	if err != nil {
		t.Fatalf("GetSyncProviderState failed: %v", err)
	}
	if !ok || provider.VaultID != "vault-1" || provider.LastPulledRevision != "10" {
		t.Fatalf("unexpected provider state: ok=%v state=%+v", ok, provider)
	}

	id, err := store.EnqueueSyncOutbox(SyncOutboxItem{
		ItemType:         "host",
		SyncID:           "sync-1",
		Operation:        "upsert",
		BaseRevision:     "r1",
		EncryptedPayload: "ciphertext",
		CreatedAt:        now,
	})
	if err != nil {
		t.Fatalf("EnqueueSyncOutbox failed: %v", err)
	}
	if id == 0 {
		t.Fatalf("expected outbox id")
	}
	outbox, err := store.ListSyncOutbox(10)
	if err != nil {
		t.Fatalf("ListSyncOutbox failed: %v", err)
	}
	if len(outbox) != 1 || outbox[0].ID != id || outbox[0].Operation != "upsert" {
		t.Fatalf("unexpected outbox: %+v", outbox)
	}
	if err := store.MarkSyncOutboxError(id, "network"); err != nil {
		t.Fatalf("MarkSyncOutboxError failed: %v", err)
	}
	outbox, err = store.ListSyncOutbox(10)
	if err != nil {
		t.Fatalf("ListSyncOutbox after error failed: %v", err)
	}
	if outbox[0].Attempts != 1 || outbox[0].LastError != "network" {
		t.Fatalf("expected outbox error state, got %+v", outbox[0])
	}
	if err := store.DeleteSyncOutboxItem(id); err != nil {
		t.Fatalf("DeleteSyncOutboxItem failed: %v", err)
	}
	outbox, err = store.ListSyncOutbox(10)
	if err != nil {
		t.Fatalf("ListSyncOutbox after delete failed: %v", err)
	}
	if len(outbox) != 0 {
		t.Fatalf("expected empty outbox, got %+v", outbox)
	}
}
