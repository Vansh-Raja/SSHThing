package db_test

import (
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Vansh-Raja/SSHThing/internal/crypto"
	"github.com/Vansh-Raja/SSHThing/internal/db"
	_ "github.com/mutecomm/go-sqlcipher/v4"
)

func TestInitMigratesLegacyHostsTable(t *testing.T) {
	tempDir := t.TempDir()

	originalDataDir := os.Getenv("SSHTHING_DATA_DIR")
	if err := os.Setenv("SSHTHING_DATA_DIR", tempDir); err != nil {
		t.Fatalf("Setenv failed: %v", err)
	}
	defer os.Setenv("SSHTHING_DATA_DIR", originalDataDir)

	const password = "testpassword123"
	dbPath := filepath.Join(tempDir, "hosts.db")
	if err := createLegacyEncryptedDB(dbPath, password); err != nil {
		t.Fatalf("createLegacyEncryptedDB failed: %v", err)
	}

	store, err := db.Init(password)
	if err != nil {
		t.Fatalf("Init should migrate legacy schema: %v", err)
	}
	defer store.Close()

	hosts, err := store.GetHosts()
	if err != nil {
		t.Fatalf("GetHosts after migration failed: %v", err)
	}
	if len(hosts) != 1 {
		t.Fatalf("expected 1 host after migration, got %d", len(hosts))
	}

	host := hosts[0]
	if host.Hostname != "legacy.example.com" {
		t.Fatalf("hostname mismatch after migration: %q", host.Hostname)
	}
	if host.UpdatedAt.IsZero() {
		t.Fatal("updated_at should be populated during migration")
	}
	if !host.UpdatedAt.Equal(host.CreatedAt) {
		t.Fatalf("expected updated_at to fall back to created_at, got created_at=%s updated_at=%s", host.CreatedAt, host.UpdatedAt)
	}
	if host.GroupName != "" {
		t.Fatalf("expected empty group_name for legacy host, got %q", host.GroupName)
	}
	if len(host.Tags) != 0 {
		t.Fatalf("expected no tags for legacy host, got %v", host.Tags)
	}

	hasUpdatedAt, err := encryptedColumnExists(dbPath, password, "hosts", "updated_at")
	if err != nil {
		t.Fatalf("schema verification failed: %v", err)
	}
	if !hasUpdatedAt {
		t.Fatal("expected migration to add updated_at column")
	}
}

func createLegacyEncryptedDB(dbPath, password string) error {
	dbConn, err := openEncryptedDB(dbPath, password)
	if err != nil {
		return err
	}
	defer dbConn.Close()

	if _, err := dbConn.Exec(`
		CREATE TABLE hosts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			label TEXT,
			hostname TEXT NOT NULL,
			username TEXT NOT NULL,
			port INTEGER DEFAULT 22,
			key_data TEXT,
			key_type TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_connected TIMESTAMP
		);
	`); err != nil {
		return err
	}

	if _, err := dbConn.Exec(`
		CREATE TABLE config (
			key TEXT PRIMARY KEY,
			value TEXT
		);
	`); err != nil {
		return err
	}

	salt, err := crypto.GenerateRandomBytes(16)
	if err != nil {
		return err
	}
	if _, err := dbConn.Exec(`INSERT INTO config (key, value) VALUES ('salt', ?)`, fmt.Sprintf("%x", salt)); err != nil {
		return err
	}

	createdAt := time.Date(2025, time.December, 16, 14, 58, 7, 0, time.UTC).Format("2006-01-02 15:04:05")
	_, err = dbConn.Exec(`
		INSERT INTO hosts (label, hostname, username, port, key_data, key_type, created_at, last_connected)
		VALUES (?, ?, ?, ?, ?, ?, ?, NULL)
	`, "legacy", "legacy.example.com", "ubuntu", 22, "", "password", createdAt)
	return err
}

func encryptedColumnExists(dbPath, password, table, column string) (bool, error) {
	dbConn, err := openEncryptedDB(dbPath, password)
	if err != nil {
		return false, err
	}
	defer dbConn.Close()

	rows, err := dbConn.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull int
		var dflt sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}

	return false, rows.Err()
}

func openEncryptedDB(dbPath, password string) (*sql.DB, error) {
	sqlcipherSalt := []byte("ssh-manager-sqlcipher-salt-v1")
	dbKey, _, err := crypto.DeriveKey(password, sqlcipherSalt)
	if err != nil {
		return nil, err
	}

	dsn := fmt.Sprintf(
		"file:%s?mode=rwc&_pragma_key=x'%s'&_pragma_cipher_page_size=4096",
		dbPath,
		hex.EncodeToString(dbKey),
	)

	return sql.Open("sqlite3", dsn)
}
