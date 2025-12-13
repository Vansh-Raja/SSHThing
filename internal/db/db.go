package db

import (
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Vansh-Raja/SSHThing/internal/crypto"
	_ "github.com/mutecomm/go-sqlcipher/v4" // SQLCipher driver
)

// Store handles database operations
type Store struct {
	db        *sql.DB
	masterKey []byte
}

// HostModel mirrors the Host struct but for DB interactions
type HostModel struct {
	ID            int
	Label         string
	Hostname      string
	Username      string
	Port          int
	KeyData       string // Encrypted blob
	KeyType       string
	CreatedAt     time.Time
	LastConnected *time.Time
}

// DBPath returns the path to the database file
func DBPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	appDir := filepath.Join(home, ".ssh-manager")
	if err := os.MkdirAll(appDir, 0700); err != nil {
		return "", err
	}
	return filepath.Join(appDir, "hosts.db"), nil
}

// Delete removes the database file (destructive).
func Delete() error {
	dbPath, err := DBPath()
	if err != nil {
		return err
	}
	err = os.Remove(dbPath)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// Exists checks if the database file exists
func Exists() (bool, error) {
	dbPath, err := DBPath()
	if err != nil {
		return false, err
	}
	_, err = os.Stat(dbPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}

func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// Init opens the database and initializes the schema.
// It derives the master key from the password and uses it for SQLCipher encryption.
func Init(password string) (*Store, error) {
	dbPath, err := DBPath()
	if err != nil {
		return nil, err
	}

	exists, err := fileExists(dbPath)
	if err != nil {
		return nil, err
	}

	// Derive a 256-bit key from the password using PBKDF2
	// We use a fixed salt for SQLCipher key derivation (different from per-key salt)
	// This is acceptable because SQLCipher has its own internal KDF
	sqlcipherSalt := []byte("ssh-manager-sqlcipher-salt-v1")
	dbKey, _, err := crypto.DeriveKey(password, sqlcipherSalt)
	if err != nil {
		return nil, err
	}

	// Convert key to hex string for SQLCipher
	keyHex := hex.EncodeToString(dbKey)

	// If the DB file already exists, verify the password in read-only mode *before*
	// attempting any schema changes. This prevents accidentally "creating" a new empty
	// database view with a wrong key and overwriting/corrupting the real DB.
	if exists {
		roDSN := fmt.Sprintf("file:%s?mode=ro&_pragma_key=x'%s'&_pragma_cipher_page_size=4096", dbPath, keyHex)
		ro, err := sql.Open("sqlite3", roDSN)
		if err != nil {
			return nil, err
		}
		if err := verifyUnlocked(ro); err != nil {
			ro.Close()
			return nil, fmt.Errorf("invalid password")
		}
		_ = ro.Close()
	}

	// Open DB read-write with SQLCipher encryption key.
	// The _pragma_key is the raw hex key, _pragma_cipher_page_size sets page size.
	rwDSN := fmt.Sprintf("file:%s?mode=rwc&_pragma_key=x'%s'&_pragma_cipher_page_size=4096", dbPath, keyHex)
	db, err := sql.Open("sqlite3", rwDSN)
	if err != nil {
		return nil, err
	}

	// Ensure schema exists (first-run / migrations).
	if err := createSchema(db); err != nil {
		db.Close()
		return nil, err
	}

	// Get or create salt for per-key AES-GCM encryption (second layer)
	salt, err := getSalt(db)
	if err != nil {
		db.Close()
		return nil, err
	}

	// Derive per-key encryption key from password + salt
	perKeyKey, _, err := crypto.DeriveKey(password, salt)
	if err != nil {
		db.Close()
		return nil, err
	}

	return &Store{
		db:        db,
		masterKey: perKeyKey,
	}, nil
}

func verifyUnlocked(db *sql.DB) error {
	// If config table exists, reading from it should succeed with the correct key.
	// With an incorrect key, SQLCipher should fail to read any pages and error.
	var dummy string
	err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' LIMIT 1").Scan(&dummy)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	// Prefer a stronger check when the expected schema exists.
	hasConfig, err := hasTable(db, "config")
	if err != nil {
		return err
	}
	if hasConfig {
		var saltHex string
		err := db.QueryRow("SELECT value FROM config WHERE key = 'salt'").Scan(&saltHex)
		if err != nil {
			return err
		}
		if saltHex == "" {
			return fmt.Errorf("missing salt")
		}
		return nil
	}

	// If the DB existed on disk but we cannot see the config table at all, treat it
	// as locked with a wrong password (or corrupted) to avoid overwriting it.
	hasHosts, err := hasTable(db, "hosts")
	if err != nil {
		return err
	}
	if hasHosts {
		// Older DB version; allow migration to create config later.
		return nil
	}

	return fmt.Errorf("locked or uninitialized")
}

func hasTable(db *sql.DB, name string) (bool, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", name).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func createSchema(db *sql.DB) error {
	// Hosts table
	_, err := db.Exec(`
	CREATE TABLE IF NOT EXISTS hosts (
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
	`)
	if err != nil {
		return err
	}

	// Migrations (older DBs won't have new columns even though CREATE TABLE changed)
	if err := ensureColumn(db, "hosts", "label", "TEXT"); err != nil {
		return err
	}

	// If the legacy `notes` column exists, migrate it away entirely by rebuilding
	// the table without the column and copying values into `label` (when label empty).
	hasNotes, err := hasColumn(db, "hosts", "notes")
	if err != nil {
		return err
	}
	if hasNotes {
		if err := migrateHostsDropNotes(db); err != nil {
			return err
		}
	}

	// Metadata table (for Salt)
	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS config (
		key TEXT PRIMARY KEY,
		value TEXT
	);
	`)
	return err
}

func hasColumn(db *sql.DB, table, column string) (bool, error) {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
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
	if err := rows.Err(); err != nil {
		return false, err
	}
	return false, nil
}

func migrateHostsDropNotes(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.Exec(`
		CREATE TABLE IF NOT EXISTS hosts_new (
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
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		INSERT INTO hosts_new (id, label, hostname, username, port, key_data, key_type, created_at, last_connected)
		SELECT
			id,
			CASE WHEN COALESCE(label, '') != '' THEN label ELSE COALESCE(notes, '') END,
			hostname, username, port, key_data, key_type, created_at, last_connected
		FROM hosts;
	`)
	if err != nil {
		return err
	}

	if _, err := tx.Exec(`DROP TABLE hosts;`); err != nil {
		return err
	}
	if _, err := tx.Exec(`ALTER TABLE hosts_new RENAME TO hosts;`); err != nil {
		return err
	}

	return tx.Commit()
}

func ensureColumn(db *sql.DB, table, column, typ string) error {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull int
		var dflt sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return err
		}
		if name == column {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	_, err = db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, typ))
	return err
}

func getSalt(db *sql.DB) ([]byte, error) {
	var saltHex string
	err := db.QueryRow("SELECT value FROM config WHERE key = 'salt'").Scan(&saltHex)
	if err == sql.ErrNoRows {
		// Generate new salt
		newSalt, err := crypto.GenerateRandomBytes(16)
		if err != nil {
			return nil, err
		}
		// Store as hex or base64. Let's use base64 for consistency.
		// Salt is public (metadata).
		saltStr := fmt.Sprintf("%x", newSalt)
		
		_, err = db.Exec("INSERT INTO config (key, value) VALUES ('salt', ?)", saltStr)
		if err != nil {
			return nil, err
		}
		return newSalt, nil
	} else if err != nil {
		return nil, err
	}

	salt, err := hex.DecodeString(saltHex)
	if err != nil {
		return nil, err
	}
	return salt, nil
}

// Close closes the database connection
func (s *Store) Close() error {
	return s.db.Close()
}

// CreateHost adds a new host
func (s *Store) CreateHost(h *HostModel, plainKey string) error {
	// Encrypt the key
	var encryptedKey string
	var err error
	if plainKey != "" {
		encryptedKey, err = crypto.Encrypt([]byte(plainKey), s.masterKey)
		if err != nil {
			return err
		}
	}

	_, err = s.db.Exec(`
		INSERT INTO hosts (label, hostname, username, port, key_data, key_type, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, h.Label, h.Hostname, h.Username, h.Port, encryptedKey, h.KeyType, time.Now())
	
	return err
}

// GetHosts returns all hosts
func (s *Store) GetHosts() ([]HostModel, error) {
	rows, err := s.db.Query(`
		SELECT id, COALESCE(label, ''), hostname, username, port,
		       COALESCE(key_type, ''), COALESCE(key_data, ''),
		       created_at, last_connected
		FROM hosts
		ORDER BY CASE WHEN COALESCE(label, '') != '' THEN label ELSE hostname END
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hosts []HostModel
	for rows.Next() {
		var h HostModel
		var lastConn sql.NullTime
		if err := rows.Scan(&h.ID, &h.Label, &h.Hostname, &h.Username, &h.Port, &h.KeyType, &h.KeyData, &h.CreatedAt, &lastConn); err != nil {
			return nil, err
		}
		if lastConn.Valid {
			h.LastConnected = &lastConn.Time
		}
		hosts = append(hosts, h)
	}
	return hosts, nil
}

// GetHostKey retrieves the decrypted private key for a host
func (s *Store) GetHostKey(id int) (string, error) {
	var encryptedKey string
	err := s.db.QueryRow("SELECT key_data FROM hosts WHERE id = ?", id).Scan(&encryptedKey)
	if err != nil {
		return "", err
	}
	
	if encryptedKey == "" {
		return "", nil
	}

	decrypted, err := crypto.Decrypt(encryptedKey, s.masterKey)
	if err != nil {
		return "", err
	}
	return string(decrypted), nil
}

// UpdateHost updates a host's metadata (without changing the key)
func (s *Store) UpdateHost(h *HostModel) error {
	_, err := s.db.Exec(`
		UPDATE hosts SET label=?, hostname=?, username=?, port=?, key_type=?
		WHERE id=?
	`, h.Label, h.Hostname, h.Username, h.Port, h.KeyType, h.ID)
	return err
}

// UpdateHostWithKey updates a host including the encrypted key data
func (s *Store) UpdateHostWithKey(h *HostModel, plainKey string) error {
	var encryptedKey string
	var err error
	if plainKey != "" {
		encryptedKey, err = crypto.Encrypt([]byte(plainKey), s.masterKey)
		if err != nil {
			return err
		}
	}

	_, err = s.db.Exec(`
		UPDATE hosts SET label=?, hostname=?, username=?, port=?, key_type=?, key_data=?
		WHERE id=?
	`, h.Label, h.Hostname, h.Username, h.Port, h.KeyType, encryptedKey, h.ID)
	return err
}

// UpdateLastConnected updates the last_connected timestamp for a host
func (s *Store) UpdateLastConnected(id int) error {
	_, err := s.db.Exec(`UPDATE hosts SET last_connected=? WHERE id=?`, time.Now(), id)
	return err
}

// DeleteHost deletes a host
func (s *Store) DeleteHost(id int) error {
	_, err := s.db.Exec("DELETE FROM hosts WHERE id=?", id)
	return err
}
