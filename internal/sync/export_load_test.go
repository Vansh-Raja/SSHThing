package sync

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Vansh-Raja/SSHThing/internal/authtoken"
	"github.com/Vansh-Raja/SSHThing/internal/crypto"
)

func TestLoadFromFile_EncryptedPayload(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, SyncFileName)

	now := time.Now().UTC().Truncate(time.Second)
	payload := SyncData{
		Version:   CurrentSyncVersion,
		Salt:      "abc123",
		UpdatedAt: now,
		Hosts: []SyncHost{
			{ID: 1, Hostname: "prod.example.com", Username: "ubuntu", Port: 22, KeyData: "ciphertext", KeyType: "password", Tags: []string{"gpu", "ec2"}, CreatedAt: now, UpdatedAt: now},
		},
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	salt := []byte("1234567890abcdef")
	key, _, err := crypto.DeriveKey("test-password", salt)
	if err != nil {
		t.Fatalf("derive key: %v", err)
	}
	encrypted, err := crypto.Encrypt(payloadJSON, key)
	if err != nil {
		t.Fatalf("encrypt payload: %v", err)
	}

	fileData := SyncFile{
		Version:   CurrentSyncVersion,
		UpdatedAt: now,
		EncSalt:   hex.EncodeToString(salt),
		Data:      encrypted,
	}
	fileJSON, err := json.Marshal(fileData)
	if err != nil {
		t.Fatalf("marshal file: %v", err)
	}
	if err := os.WriteFile(path, fileJSON, 0600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	loaded, err := LoadFromFile(path, "test-password")
	if err != nil {
		t.Fatalf("load encrypted file: %v", err)
	}
	if loaded == nil || len(loaded.Hosts) != 1 {
		t.Fatalf("expected one host, got %+v", loaded)
	}
	if loaded.Hosts[0].Hostname != "prod.example.com" {
		t.Fatalf("unexpected hostname: %q", loaded.Hosts[0].Hostname)
	}
	if len(loaded.Hosts[0].Tags) != 2 || loaded.Hosts[0].Tags[0] != "gpu" || loaded.Hosts[0].Tags[1] != "ec2" {
		t.Fatalf("unexpected tags after decrypt: %+v", loaded.Hosts[0].Tags)
	}

	if _, err := LoadFromFile(path, "wrong-password"); err == nil {
		t.Fatalf("expected decrypt error with wrong password")
	}
}

func TestLoadFromFile_LegacyPlaintext(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, SyncFileName)

	now := time.Now().UTC().Truncate(time.Second)
	legacy := SyncData{
		Version:   2,
		Salt:      "legacy-salt",
		UpdatedAt: now,
		Hosts: []SyncHost{
			{ID: 7, Hostname: "legacy.example.com", Username: "root", Port: 2222, KeyType: "password", CreatedAt: now, UpdatedAt: now},
		},
	}
	b, err := json.Marshal(legacy)
	if err != nil {
		t.Fatalf("marshal legacy: %v", err)
	}
	if err := os.WriteFile(path, b, 0600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	loaded, err := LoadFromFile(path, "unused")
	if err != nil {
		t.Fatalf("load legacy: %v", err)
	}
	if loaded == nil || len(loaded.Hosts) != 1 {
		t.Fatalf("expected one legacy host, got %+v", loaded)
	}
	if loaded.Version != 2 {
		t.Fatalf("expected legacy version 2, got %d", loaded.Version)
	}
}

func TestExportDataToFilePreservesPortableProviderPayload(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, SyncFileName)

	now := time.Now().UTC().Truncate(time.Second)
	data := &SyncData{
		Version:   CurrentSyncVersion,
		Salt:      "abc123",
		UpdatedAt: now,
		Groups: []SyncGroup{{
			SyncID:    "group-1",
			Name:      "Work",
			CreatedAt: now,
			UpdatedAt: now,
		}},
		Hosts: []SyncHost{{
			ID:        1,
			SyncID:    "host-1",
			Label:     "GPU",
			GroupName: "Work",
			Hostname:  "gpu.example.com",
			Username:  "root",
			Port:      22,
			KeyData:   "ciphertext",
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
			Hosts:       []authtoken.SyncTokenHost{{HostID: 1, DisplayLabel: "GPU"}},
		}},
	}

	if err := ExportDataToFile(data, path, "pw"); err != nil {
		t.Fatalf("ExportDataToFile failed: %v", err)
	}
	loaded, err := LoadFromFile(path, "pw")
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}
	if loaded.Version != CurrentSyncVersion || loaded.Salt != data.Salt {
		t.Fatalf("unexpected loaded metadata: %+v", loaded)
	}
	if len(loaded.Groups) != 1 || loaded.Groups[0].SyncID != "group-1" {
		t.Fatalf("expected group payload preserved, got %+v", loaded.Groups)
	}
	if len(loaded.Hosts) != 1 || loaded.Hosts[0].SyncID != "host-1" || loaded.Hosts[0].KeyData != "ciphertext" {
		t.Fatalf("expected host payload preserved, got %+v", loaded.Hosts)
	}
	if len(loaded.TokenDefs) != 1 || loaded.TokenDefs[0].TokenID != "token-1" {
		t.Fatalf("expected token definitions preserved, got %+v", loaded.TokenDefs)
	}
}
