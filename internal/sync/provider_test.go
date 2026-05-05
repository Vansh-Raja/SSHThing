package sync

import (
	"errors"
	"testing"
	"time"

	"github.com/Vansh-Raja/SSHThing/internal/authtoken"
	"github.com/Vansh-Raja/SSHThing/internal/personalsync"
)

func TestConvexVaultItemEncryptionPreservesPortableProviderPayload(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	data := &SyncData{
		Version:   CurrentSyncVersion,
		Salt:      "source-db-salt",
		UpdatedAt: now,
		Groups: []SyncGroup{{
			SyncID:    "group-1",
			Name:      "Work",
			CreatedAt: now,
			UpdatedAt: now,
		}},
		Hosts: []SyncHost{{
			ID:        10,
			SyncID:    "host-1",
			Label:     "GPU",
			GroupName: "Work",
			Hostname:  "gpu.example.com",
			Username:  "root",
			Port:      22,
			KeyData:   "encrypted-secret",
			KeyType:   "private_key",
			CreatedAt: now,
			UpdatedAt: now,
		}},
		TokenDefs: []authtoken.SyncTokenDef{{
			TokenID:     "token-1",
			Name:        "agent",
			CreatedAt:   now,
			UpdatedAt:   now,
			SyncEnabled: true,
			Hosts:       []authtoken.SyncTokenHost{{HostID: 10, DisplayLabel: "GPU"}},
		}},
	}

	items, err := encryptVaultItems(data, "pw", "00112233445566778899aabbccddeeff")
	if err != nil {
		t.Fatalf("encryptVaultItems failed: %v", err)
	}
	if len(items) != 4 {
		t.Fatalf("expected meta + group + host + token_def, got %d", len(items))
	}

	loaded, err := decryptVaultItems(items, "pw", "00112233445566778899aabbccddeeff")
	if err != nil {
		t.Fatalf("decryptVaultItems failed: %v", err)
	}
	if loaded.Salt != data.Salt {
		t.Fatalf("expected source salt preserved, got %q", loaded.Salt)
	}
	if len(loaded.Groups) != 1 || loaded.Groups[0].SyncID != "group-1" {
		t.Fatalf("expected group payload preserved, got %+v", loaded.Groups)
	}
	if len(loaded.Hosts) != 1 || loaded.Hosts[0].SyncID != "host-1" || loaded.Hosts[0].KeyData != "encrypted-secret" {
		t.Fatalf("expected host payload preserved, got %+v", loaded.Hosts)
	}
	if len(loaded.TokenDefs) != 1 || loaded.TokenDefs[0].TokenID != "token-1" {
		t.Fatalf("expected token definition preserved, got %+v", loaded.TokenDefs)
	}

	if _, err := decryptVaultItems(items, "wrong", "00112233445566778899aabbccddeeff"); err == nil {
		t.Fatalf("expected decrypt failure with wrong password")
	}
}

func TestUnsupportedBaseRevisionCompatibilityHelpers(t *testing.T) {
	err := errors.New("Server Error ArgumentValidationError: Object contains extra field `baseRevision` that is not in the validator. Path: .items[0]")
	if !isUnsupportedBaseRevisionError(err) {
		t.Fatalf("expected baseRevision validator error to be detected")
	}
	if isUnsupportedBaseRevisionError(errors.New("permission denied")) {
		t.Fatalf("unexpected detection for unrelated error")
	}

	items := []personalsync.VaultItem{{SyncID: "a", BaseRevision: "12"}, {SyncID: "b", BaseRevision: "13"}}
	compat := clearItemBaseRevisions(items)
	for _, item := range compat {
		if item.BaseRevision != "" {
			t.Fatalf("expected baseRevision to be cleared, got %+v", compat)
		}
	}
	if items[0].BaseRevision == "" || items[1].BaseRevision == "" {
		t.Fatalf("clearItemBaseRevisions should not mutate original slice")
	}
}
