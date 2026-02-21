package sync

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

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
			{ID: 1, Hostname: "prod.example.com", Username: "ubuntu", Port: 22, KeyData: "ciphertext", KeyType: "password", CreatedAt: now, UpdatedAt: now},
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
