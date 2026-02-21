package sync

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Vansh-Raja/SSHThing/internal/crypto"
	"github.com/Vansh-Raja/SSHThing/internal/db"
)

// Export reads all hosts from the database and returns them as SyncData.
// The key data remains encrypted - we do not decrypt keys during export.
// The salt is included so importing databases can re-encrypt with their own salt.
func Export(store *db.Store) (*SyncData, error) {
	// Get the encryption salt for this database
	salt, err := store.GetSalt()
	if err != nil {
		return nil, fmt.Errorf("failed to get salt: %w", err)
	}

	hosts, err := store.GetHosts()
	if err != nil {
		return nil, fmt.Errorf("failed to get hosts: %w", err)
	}

	groups, err := store.GetGroupsForSync(GroupTombstoneRetention)
	if err != nil {
		return nil, fmt.Errorf("failed to get groups: %w", err)
	}

	syncGroups := make([]SyncGroup, 0, len(groups))
	seenGroups := make(map[string]bool, len(groups))
	for _, g := range groups {
		seenGroups[strings.ToLower(strings.TrimSpace(g.Name))] = true
		syncGroups = append(syncGroups, SyncGroup{
			Name:      g.Name,
			CreatedAt: g.CreatedAt,
			UpdatedAt: g.UpdatedAt,
			DeletedAt: g.DeletedAt,
		})
	}

	syncHosts := make([]SyncHost, len(hosts))
	for i, h := range hosts {
		if gn := strings.TrimSpace(h.GroupName); gn != "" {
			k := strings.ToLower(gn)
			if !seenGroups[k] {
				now := time.Now()
				syncGroups = append(syncGroups, SyncGroup{Name: gn, CreatedAt: now, UpdatedAt: now})
				seenGroups[k] = true
			}
		}
		syncHosts[i] = SyncHost{
			ID:            h.ID,
			Label:         h.Label,
			GroupName:     h.GroupName,
			Hostname:      h.Hostname,
			Username:      h.Username,
			Port:          h.Port,
			KeyData:       h.KeyData, // Already encrypted
			KeyType:       h.KeyType,
			CreatedAt:     h.CreatedAt,
			UpdatedAt:     h.UpdatedAt,
			LastConnected: h.LastConnected,
		}
	}

	return &SyncData{
		Version:   CurrentSyncVersion,
		Salt:      salt,
		UpdatedAt: time.Now(),
		Groups:    syncGroups,
		Hosts:     syncHosts,
	}, nil
}

// ExportToFile exports sync data to an encrypted JSON file at the specified path.
func ExportToFile(store *db.Store, filePath string, password string) error {
	data, err := Export(store)
	if err != nil {
		return err
	}
	return ExportDataToFile(data, filePath, password)
}

// ExportDataToFile writes sync data to an encrypted JSON file at the specified path.
func ExportDataToFile(data *SyncData, filePath string, password string) error {
	if data == nil {
		return fmt.Errorf("missing sync data")
	}
	if strings.TrimSpace(password) == "" {
		return fmt.Errorf("missing master password for sync encryption")
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal sync payload: %w", err)
	}

	encSalt, err := crypto.GenerateRandomBytes(16)
	if err != nil {
		return fmt.Errorf("failed to generate sync encryption salt: %w", err)
	}
	key, _, err := crypto.DeriveKey(password, encSalt)
	if err != nil {
		return fmt.Errorf("failed to derive sync encryption key: %w", err)
	}
	encryptedPayload, err := crypto.Encrypt(payload, key)
	if err != nil {
		return fmt.Errorf("failed to encrypt sync payload: %w", err)
	}

	fileData := SyncFile{
		Version:   CurrentSyncVersion,
		UpdatedAt: time.Now(),
		EncSalt:   fmt.Sprintf("%x", encSalt),
		Data:      encryptedPayload,
	}

	jsonBytes, err := json.MarshalIndent(fileData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal encrypted sync data: %w", err)
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(filePath), 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write to temp file first, then rename (atomic write)
	tmpPath := filePath + ".tmp"
	if err := os.WriteFile(tmpPath, jsonBytes, 0600); err != nil {
		return fmt.Errorf("failed to write sync file: %w", err)
	}

	if err := os.Rename(tmpPath, filePath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to rename sync file: %w", err)
	}

	return nil
}

// LoadFromFile reads sync data from a JSON file, supporting encrypted and legacy plaintext formats.
func LoadFromFile(filePath string, password string) (*SyncData, error) {
	jsonBytes, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No sync file exists yet
		}
		return nil, fmt.Errorf("failed to read sync file: %w", err)
	}

	var fileData SyncFile
	if err := json.Unmarshal(jsonBytes, &fileData); err != nil {
		return nil, fmt.Errorf("failed to parse sync data: %w", err)
	}

	if fileData.Data != "" && fileData.EncSalt != "" {
		if strings.TrimSpace(password) == "" {
			return nil, fmt.Errorf("missing master password for encrypted sync import")
		}
		salt, err := hex.DecodeString(fileData.EncSalt)
		if err != nil {
			return nil, fmt.Errorf("invalid sync encryption salt: %w", err)
		}
		key, _, err := crypto.DeriveKey(password, salt)
		if err != nil {
			return nil, fmt.Errorf("failed to derive sync decryption key: %w", err)
		}
		plain, err := crypto.Decrypt(fileData.Data, key)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt sync payload: %w", err)
		}

		var data SyncData
		if err := json.Unmarshal(plain, &data); err != nil {
			return nil, fmt.Errorf("failed to parse decrypted sync payload: %w", err)
		}
		if data.Version == 0 {
			data.Version = fileData.Version
		}
		if data.Version == 0 {
			data.Version = CurrentSyncVersion
		}
		if data.Hosts == nil {
			data.Hosts = []SyncHost{}
		}
		return &data, nil
	}

	if fileData.Version != 0 || fileData.Salt != "" || len(fileData.Groups) > 0 || len(fileData.Hosts) > 0 {
		legacy := &SyncData{
			Version:   fileData.Version,
			Salt:      fileData.Salt,
			UpdatedAt: fileData.UpdatedAt,
			Groups:    fileData.Groups,
			Hosts:     fileData.Hosts,
		}
		if legacy.Version == 0 {
			legacy.Version = 2
		}
		if legacy.Hosts == nil {
			legacy.Hosts = []SyncHost{}
		}
		return legacy, nil
	}

	var fallback SyncData
	if err := json.Unmarshal(jsonBytes, &fallback); err != nil {
		return nil, fmt.Errorf("failed to parse legacy sync payload: %w", err)
	}
	if fallback.Hosts == nil {
		fallback.Hosts = []SyncHost{}
	}
	if fallback.Version == 0 {
		fallback.Version = 2
	}
	return &fallback, nil
}
