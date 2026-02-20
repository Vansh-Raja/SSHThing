package db

import (
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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
	GroupName     string
	Hostname      string
	Username      string
	Port          int
	KeyData       string // Encrypted blob
	KeyType       string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	LastConnected *time.Time
}

// GroupModel represents a named group used to organize hosts.
// Groups are identified by Name (case-insensitive uniqueness in DB).
// Deleted groups are tombstoned via DeletedAt and may be garbage collected later.
type GroupModel struct {
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

type MountState struct {
	HostID     int
	LocalPath  string
	RemotePath string
	MountedAt  time.Time
}

// DBPath returns the path to the database file
func DBPath() (string, error) {
	// Allow overriding the DB path for testing or custom setups.
	// - SSHTHING_DB_PATH: absolute/relative path to the DB file
	// - SSHTHING_DATA_DIR: directory where hosts.db will be stored
	if p := os.Getenv("SSHTHING_DB_PATH"); p != "" {
		if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			return "", err
		}
		return p, nil
	}
	if dir := os.Getenv("SSHTHING_DATA_DIR"); dir != "" {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return "", err
		}
		return filepath.Join(dir, "hosts.db"), nil
	}

	// Platform-specific default paths
	if runtime.GOOS == "windows" {
		// On Windows, use %APPDATA%\sshthing\hosts.db
		configDir, err := os.UserConfigDir()
		if err != nil {
			return "", err
		}
		appDir := filepath.Join(configDir, "sshthing")
		if err := os.MkdirAll(appDir, 0700); err != nil {
			return "", err
		}
		return filepath.Join(appDir, "hosts.db"), nil
	}

	// On Unix (macOS/Linux), keep ~/.ssh-manager for backward compatibility
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
			return nil, classifyUnlockError(err, dbPath)
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

func classifyUnlockError(err error, dbPath string) error {
	if err == nil {
		return nil
	}
	msg := strings.ToLower(err.Error())

	switch {
	case strings.Contains(msg, "requires cgo"),
		strings.Contains(msg, "compiled with 'cgo_enabled=0'"):
		return fmt.Errorf("this binary was built without CGO support; rebuild with CGO_ENABLED=1")
	case strings.Contains(msg, "database is locked"),
		strings.Contains(msg, "database table is locked"),
		strings.Contains(msg, "database is busy"):
		return fmt.Errorf("database is in use by another process: %s", dbPath)
	case strings.Contains(msg, "access is denied"),
		strings.Contains(msg, "permission denied"):
		return fmt.Errorf("cannot access database file: %s", dbPath)
	case strings.Contains(msg, "file is encrypted"),
		strings.Contains(msg, "file is not a database"),
		strings.Contains(msg, "locked or uninitialized"),
		strings.Contains(msg, "missing salt"):
		return fmt.Errorf("invalid password for database: %s", dbPath)
	default:
		return fmt.Errorf("failed to unlock database: %w", err)
	}
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
		group_name TEXT,
		hostname TEXT NOT NULL,
		username TEXT NOT NULL,
		port INTEGER DEFAULT 22,
		key_data TEXT,
		key_type TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
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

	// Add updated_at column for sync conflict resolution
	// Note: SQLite ALTER TABLE doesn't support DEFAULT CURRENT_TIMESTAMP, so we use NULL default
	if err := ensureColumn(db, "hosts", "updated_at", "TIMESTAMP"); err != nil {
		return err
	}

	// Add group_name column for host grouping
	if err := ensureColumn(db, "hosts", "group_name", "TEXT"); err != nil {
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

	// Groups table (for organizing hosts)
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS groups (
			name TEXT PRIMARY KEY COLLATE NOCASE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			deleted_at TIMESTAMP
		);
	`)
	if err != nil {
		return err
	}

	// Metadata table (for Salt)
	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS config (
		key TEXT PRIMARY KEY,
		value TEXT
	);
	`)
	if err != nil {
		return err
	}

	// Mount state table (best-effort persistence for Finder mounts)
	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS mounts (
		host_id INTEGER PRIMARY KEY,
		local_path TEXT NOT NULL,
		remote_path TEXT,
		mounted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
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
			group_name TEXT,
			hostname TEXT NOT NULL,
			username TEXT NOT NULL,
			port INTEGER DEFAULT 22,
			key_data TEXT,
			key_type TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_connected TIMESTAMP
		);
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		INSERT INTO hosts_new (id, label, group_name, hostname, username, port, key_data, key_type, created_at, updated_at, last_connected)
		SELECT
			id,
			CASE WHEN COALESCE(label, '') != '' THEN label ELSE COALESCE(notes, '') END,
			'',
			hostname, username, port, key_data, key_type, created_at, created_at, last_connected
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

// GetSalt returns the encryption salt for this database (hex encoded)
func (s *Store) GetSalt() (string, error) {
	var saltHex string
	err := s.db.QueryRow("SELECT value FROM config WHERE key = 'salt'").Scan(&saltHex)
	if err != nil {
		return "", err
	}
	return saltHex, nil
}

// ReencryptKeyData decrypts key data using a source salt and re-encrypts with local salt.
// This is used during sync import when the source database had a different salt.
func (s *Store) ReencryptKeyData(encryptedData string, sourceSaltHex string, password string) (string, error) {
	if encryptedData == "" {
		return "", nil
	}

	// Decode source salt
	sourceSalt, err := hex.DecodeString(sourceSaltHex)
	if err != nil {
		return "", fmt.Errorf("invalid source salt: %w", err)
	}

	// Derive source key
	sourceKey, _, err := crypto.DeriveKey(password, sourceSalt)
	if err != nil {
		return "", fmt.Errorf("failed to derive source key: %w", err)
	}

	// Decrypt with source key
	plaintext, err := crypto.Decrypt(encryptedData, sourceKey)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt key: %w", err)
	}

	// Re-encrypt with local key (s.masterKey)
	reencrypted, err := crypto.Encrypt(plaintext, s.masterKey)
	if err != nil {
		return "", fmt.Errorf("failed to re-encrypt key: %w", err)
	}

	return reencrypted, nil
}

func (s *Store) UpsertMountState(hostID int, localPath, remotePath string) error {
	_, err := s.db.Exec(`
		INSERT INTO mounts (host_id, local_path, remote_path, mounted_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(host_id) DO UPDATE SET
			local_path=excluded.local_path,
			remote_path=excluded.remote_path,
			mounted_at=excluded.mounted_at
	`, hostID, localPath, remotePath, time.Now())
	return err
}

func (s *Store) DeleteMountState(hostID int) error {
	_, err := s.db.Exec(`DELETE FROM mounts WHERE host_id = ?`, hostID)
	return err
}

func (s *Store) DeleteAllMountStates() error {
	_, err := s.db.Exec(`DELETE FROM mounts`)
	return err
}

func (s *Store) GetMountStates() ([]MountState, error) {
	rows, err := s.db.Query(`
		SELECT host_id, local_path, COALESCE(remote_path, ''), mounted_at
		FROM mounts
		ORDER BY mounted_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []MountState
	for rows.Next() {
		var ms MountState
		if err := rows.Scan(&ms.HostID, &ms.LocalPath, &ms.RemotePath, &ms.MountedAt); err != nil {
			return nil, err
		}
		out = append(out, ms)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
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

	now := time.Now()
	_, err = s.db.Exec(`
		INSERT INTO hosts (label, group_name, hostname, username, port, key_data, key_type, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, h.Label, normalizeGroupName(h.GroupName), h.Hostname, h.Username, h.Port, encryptedKey, h.KeyType, now, now)

	return err
}

// GetHosts returns all hosts
func (s *Store) GetHosts() ([]HostModel, error) {
	// Check if updated_at column exists (for backward compatibility)
	hasUpdatedAt, err := hasColumn(s.db, "hosts", "updated_at")
	if err != nil {
		return nil, err
	}

	var query string
	if hasUpdatedAt {
		query = `
			SELECT id, COALESCE(label, ''), COALESCE(group_name, ''), hostname, username, port,
			       COALESCE(key_type, ''), COALESCE(key_data, ''),
			       created_at, COALESCE(updated_at, created_at), last_connected
			FROM hosts
			ORDER BY CASE WHEN COALESCE(label, '') != '' THEN label ELSE hostname END
		`
	} else {
		query = `
			SELECT id, COALESCE(label, ''), COALESCE(group_name, ''), hostname, username, port,
			       COALESCE(key_type, ''), COALESCE(key_data, ''),
			       created_at, created_at, last_connected
			FROM hosts
			ORDER BY CASE WHEN COALESCE(label, '') != '' THEN label ELSE hostname END
		`
	}

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hosts []HostModel
	for rows.Next() {
		var h HostModel
		var createdAtStr, updatedAtStr string
		var lastConnStr sql.NullString
		if err := rows.Scan(&h.ID, &h.Label, &h.GroupName, &h.Hostname, &h.Username, &h.Port, &h.KeyType, &h.KeyData, &createdAtStr, &updatedAtStr, &lastConnStr); err != nil {
			return nil, err
		}
		h.GroupName = normalizeGroupName(h.GroupName)
		h.CreatedAt = parseTimestamp(createdAtStr)
		h.UpdatedAt = parseTimestamp(updatedAtStr)
		if h.UpdatedAt.IsZero() {
			h.UpdatedAt = h.CreatedAt
		}
		if lastConnStr.Valid && lastConnStr.String != "" {
			t := parseTimestamp(lastConnStr.String)
			if !t.IsZero() {
				h.LastConnected = &t
			}
		}
		hosts = append(hosts, h)
	}
	return hosts, nil
}

// parseTimestamp attempts to parse a SQLite timestamp string
func parseTimestamp(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	// Try common SQLite timestamp formats
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02T15:04:05.999999999",
		"2006-01-02 15:04:05-07:00",
		"2006-01-02T15:04:05-07:00",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t
		}
	}
	return time.Time{}
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
		UPDATE hosts SET label=?, group_name=?, hostname=?, username=?, port=?, key_type=?, updated_at=?
		WHERE id=?
	`, h.Label, normalizeGroupName(h.GroupName), h.Hostname, h.Username, h.Port, h.KeyType, time.Now(), h.ID)
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
		UPDATE hosts SET label=?, group_name=?, hostname=?, username=?, port=?, key_type=?, key_data=?, updated_at=?
		WHERE id=?
	`, h.Label, normalizeGroupName(h.GroupName), h.Hostname, h.Username, h.Port, h.KeyType, encryptedKey, time.Now(), h.ID)
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

// CreateHostWithID creates a host with a specific ID (used for sync import).
// The keyData should already be encrypted.
func (s *Store) CreateHostWithID(h *HostModel, encryptedKeyData string) error {
	_, err := s.db.Exec(`
		INSERT INTO hosts (id, label, group_name, hostname, username, port, key_data, key_type, created_at, updated_at, last_connected)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, h.ID, h.Label, normalizeGroupName(h.GroupName), h.Hostname, h.Username, h.Port, encryptedKeyData, h.KeyType, h.CreatedAt, h.UpdatedAt, h.LastConnected)
	return err
}

// UpdateHostFromSync updates a host with sync data, preserving exact timestamps.
// The keyData should already be encrypted.
func (s *Store) UpdateHostFromSync(h *HostModel, encryptedKeyData string, updatedAt time.Time) error {
	_, err := s.db.Exec(`
		UPDATE hosts SET label=?, group_name=?, hostname=?, username=?, port=?, key_type=?, key_data=?, updated_at=?, last_connected=?
		WHERE id=?
	`, h.Label, normalizeGroupName(h.GroupName), h.Hostname, h.Username, h.Port, h.KeyType, encryptedKeyData, updatedAt, h.LastConnected, h.ID)
	return err
}

func normalizeGroupName(name string) string {
	name = strings.TrimSpace(name)
	return name
}

// GetGroups returns all non-deleted groups (names only), ordered alphabetically.
func (s *Store) GetGroups() ([]string, error) {
	rows, err := s.db.Query(`
		SELECT name
		FROM groups
		WHERE deleted_at IS NULL
		ORDER BY LOWER(name)
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		name = normalizeGroupName(name)
		if name == "" {
			continue
		}
		out = append(out, name)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// GetGroupsForSync returns groups including recent tombstones.
// Tombstones older than retention are excluded (they should be garbage-collected separately).
func (s *Store) GetGroupsForSync(retention time.Duration) ([]GroupModel, error) {
	cutoff := time.Now().Add(-retention)
	rows, err := s.db.Query(`
		SELECT name,
		       created_at,
		       COALESCE(updated_at, created_at),
		       deleted_at
		FROM groups
		WHERE deleted_at IS NULL OR deleted_at >= ?
		ORDER BY LOWER(name)
	`, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []GroupModel
	for rows.Next() {
		var gm GroupModel
		var createdStr, updatedStr string
		var deletedStr sql.NullString
		if err := rows.Scan(&gm.Name, &createdStr, &updatedStr, &deletedStr); err != nil {
			return nil, err
		}
		gm.Name = normalizeGroupName(gm.Name)
		gm.CreatedAt = parseTimestamp(createdStr)
		gm.UpdatedAt = parseTimestamp(updatedStr)
		if deletedStr.Valid && strings.TrimSpace(deletedStr.String) != "" {
			t := parseTimestamp(deletedStr.String)
			if !t.IsZero() {
				gm.DeletedAt = &t
			}
		}
		if gm.Name == "" {
			continue
		}
		out = append(out, gm)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// UpsertGroup creates or revives a group, updating timestamps.
func (s *Store) UpsertGroup(name string) error {
	name = normalizeGroupName(name)
	if name == "" {
		return nil
	}
	now := time.Now()
	_, err := s.db.Exec(`
		INSERT INTO groups (name, created_at, updated_at, deleted_at)
		VALUES (?, ?, ?, NULL)
		ON CONFLICT(name) DO UPDATE SET
			updated_at=excluded.updated_at,
			deleted_at=NULL
	`, name, now, now)
	return err
}

// RenameGroup renames a group and moves any hosts assigned to it.
// The old name is tombstoned so the deletion propagates via sync.
func (s *Store) RenameGroup(oldName, newName string) error {
	oldName = normalizeGroupName(oldName)
	newName = normalizeGroupName(newName)
	if oldName == "" || newName == "" {
		return fmt.Errorf("group name cannot be empty")
	}
	if strings.EqualFold(oldName, newName) {
		// Treat as a touch/update.
		return s.UpsertGroup(newName)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	now := time.Now()
	// Ensure new group exists and is active.
	if _, err := tx.Exec(`
		INSERT INTO groups (name, created_at, updated_at, deleted_at)
		VALUES (?, ?, ?, NULL)
		ON CONFLICT(name) DO UPDATE SET
			updated_at=excluded.updated_at,
			deleted_at=NULL
	`, newName, now, now); err != nil {
		return err
	}

	// Move hosts (case-insensitive match).
	if _, err := tx.Exec(`
		UPDATE hosts
		SET group_name=?, updated_at=?
		WHERE LOWER(COALESCE(group_name, '')) = LOWER(?)
	`, newName, now, oldName); err != nil {
		return err
	}

	// Tombstone old group.
	if _, err := tx.Exec(`
		INSERT INTO groups (name, created_at, updated_at, deleted_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(name) DO UPDATE SET
			updated_at=excluded.updated_at,
			deleted_at=excluded.deleted_at
	`, oldName, now, now, now); err != nil {
		return err
	}

	return tx.Commit()
}

// DeleteGroup tombstones a group and ungroups all hosts assigned to it.
func (s *Store) DeleteGroup(name string) error {
	name = normalizeGroupName(name)
	if name == "" {
		return nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	now := time.Now()
	if _, err := tx.Exec(`
		UPDATE hosts
		SET group_name=NULL, updated_at=?
		WHERE LOWER(COALESCE(group_name, '')) = LOWER(?)
	`, now, name); err != nil {
		return err
	}

	if _, err := tx.Exec(`
		INSERT INTO groups (name, created_at, updated_at, deleted_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(name) DO UPDATE SET
			updated_at=excluded.updated_at,
			deleted_at=excluded.deleted_at
	`, name, now, now, now); err != nil {
		return err
	}

	return tx.Commit()
}

// PurgeDeletedGroups permanently removes tombstoned groups older than the given age.
func (s *Store) PurgeDeletedGroups(olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan)
	_, err := s.db.Exec(`
		DELETE FROM groups
		WHERE deleted_at IS NOT NULL AND deleted_at < ?
	`, cutoff)
	return err
}

// UpsertGroupFromSync applies group state from a sync payload, preserving timestamps.
// If deletedAt is non-nil, hosts assigned to the group are ungrouped (hosts updated_at is set to now).
func (s *Store) UpsertGroupFromSync(name string, createdAt, updatedAt time.Time, deletedAt *time.Time) error {
	name = normalizeGroupName(name)
	if name == "" {
		return nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.Exec(`
		INSERT INTO groups (name, created_at, updated_at, deleted_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(name) DO UPDATE SET
			updated_at=excluded.updated_at,
			deleted_at=excluded.deleted_at
	`, name, createdAt, updatedAt, deletedAt)
	if err != nil {
		return err
	}

	if deletedAt != nil {
		// Ungroup any hosts pointing at this group.
		if _, err := tx.Exec(`
			UPDATE hosts
			SET group_name=NULL, updated_at=?
			WHERE LOWER(COALESCE(group_name, '')) = LOWER(?)
		`, time.Now(), name); err != nil {
			return err
		}
	}

	return tx.Commit()
}
