package sync

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

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

	syncHosts := make([]SyncHost, len(hosts))
	for i, h := range hosts {
		syncHosts[i] = SyncHost{
			ID:            h.ID,
			Label:         h.Label,
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
		Hosts:     syncHosts,
	}, nil
}

// ExportToFile exports sync data to a JSON file at the specified path.
func ExportToFile(store *db.Store, filePath string) error {
	data, err := Export(store)
	if err != nil {
		return err
	}

	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal sync data: %w", err)
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

// LoadFromFile reads sync data from a JSON file.
func LoadFromFile(filePath string) (*SyncData, error) {
	jsonBytes, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No sync file exists yet
		}
		return nil, fmt.Errorf("failed to read sync file: %w", err)
	}

	var data SyncData
	if err := json.Unmarshal(jsonBytes, &data); err != nil {
		return nil, fmt.Errorf("failed to parse sync data: %w", err)
	}

	return &data, nil
}
