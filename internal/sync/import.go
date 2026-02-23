package sync

import (
	"fmt"
	"strings"
	"time"

	"github.com/Vansh-Raja/SSHThing/internal/db"
)

// ImportResult contains the result of an import operation
type ImportResult struct {
	Added     int
	Updated   int
	Unchanged int
	Conflicts []SyncConflict
}

// Import merges remote sync data into the local database.
// Uses "last write wins" strategy based on UpdatedAt timestamps.
// The store must already be opened with the correct master password.
// If the remote salt differs from local, keys are re-encrypted with the local salt.
func Import(store *db.Store, remote *SyncData, password string) (*ImportResult, error) {
	if remote == nil {
		return &ImportResult{}, nil
	}

	// Merge/apply groups first (so tombstoned groups ungroup hosts promptly).
	if len(remote.Groups) > 0 {
		localGroups, err := store.GetGroupsForSync(GroupTombstoneRetention)
		if err != nil {
			return nil, fmt.Errorf("failed to get local groups: %w", err)
		}
		localByName := make(map[string]db.GroupModel, len(localGroups))
		for _, g := range localGroups {
			localByName[strings.ToLower(g.Name)] = g
		}

		for _, rg := range remote.Groups {
			name := strings.TrimSpace(rg.Name)
			if name == "" {
				continue
			}
			lg, exists := localByName[strings.ToLower(name)]
			if exists {
				// If local is newer, keep it.
				if lg.UpdatedAt.After(rg.UpdatedAt) {
					continue
				}
			}
			if err := store.UpsertGroupFromSync(name, rg.CreatedAt, rg.UpdatedAt, rg.DeletedAt); err != nil {
				return nil, fmt.Errorf("failed to apply group %q: %w", name, err)
			}
		}
	}

	// Check if we need to re-encrypt keys (different salt)
	localSalt, err := store.GetSalt()
	if err != nil {
		return nil, fmt.Errorf("failed to get local salt: %w", err)
	}
	needsReencrypt := remote.Salt != "" && remote.Salt != localSalt

	// Get current local hosts
	localHosts, err := store.GetHosts()
	if err != nil {
		return nil, fmt.Errorf("failed to get local hosts: %w", err)
	}

	// Build a map of local hosts by ID for quick lookup
	localByID := make(map[int]db.HostModel)
	for _, h := range localHosts {
		localByID[h.ID] = h
	}

	result := &ImportResult{}

	// Helper to get key data, re-encrypting if needed
	getKeyData := func(keyData string) (string, error) {
		if !needsReencrypt || keyData == "" {
			return keyData, nil
		}
		return store.ReencryptKeyData(keyData, remote.Salt, password)
	}

	for _, remoteHost := range remote.Hosts {
		localHost, exists := localByID[remoteHost.ID]

		if !exists {
			// New host from remote - add it
			keyData, err := getKeyData(remoteHost.KeyData)
			if err != nil {
				return nil, fmt.Errorf("failed to re-encrypt key for host %d: %w", remoteHost.ID, err)
			}
			if err := addHostFromSync(store, remoteHost, keyData); err != nil {
				return nil, fmt.Errorf("failed to add host %d: %w", remoteHost.ID, err)
			}
			result.Added++
			continue
		}

		// Host exists locally - check timestamps for conflict resolution
		if remoteHost.UpdatedAt.After(localHost.UpdatedAt) {
			// Remote is newer - update local
			keyData, err := getKeyData(remoteHost.KeyData)
			if err != nil {
				return nil, fmt.Errorf("failed to re-encrypt key for host %d: %w", remoteHost.ID, err)
			}
			if err := updateHostFromSync(store, remoteHost, keyData); err != nil {
				return nil, fmt.Errorf("failed to update host %d: %w", remoteHost.ID, err)
			}
			result.Updated++
			result.Conflicts = append(result.Conflicts, SyncConflict{
				HostID:     remoteHost.ID,
				Hostname:   remoteHost.Hostname,
				LocalTime:  localHost.UpdatedAt,
				RemoteTime: remoteHost.UpdatedAt,
				Resolution: "remote",
			})
		} else if localHost.UpdatedAt.After(remoteHost.UpdatedAt) {
			// Local is newer - keep local (will be pushed on next sync)
			result.Conflicts = append(result.Conflicts, SyncConflict{
				HostID:     remoteHost.ID,
				Hostname:   remoteHost.Hostname,
				LocalTime:  localHost.UpdatedAt,
				RemoteTime: remoteHost.UpdatedAt,
				Resolution: "local",
			})
			result.Unchanged++
		} else {
			// Same timestamp - no change needed
			result.Unchanged++
		}

		// Mark as processed
		delete(localByID, remoteHost.ID)
	}

	// Remaining hosts in localByID exist only locally - they will be pushed on next sync
	// We don't delete them as they might be new local additions

	// Best-effort garbage collection of old group tombstones.
	_ = store.PurgeDeletedGroups(GroupTombstoneRetention)

	return result, nil
}

// addHostFromSync creates a new host from sync data.
// Note: We insert with the original ID to maintain consistency across devices.
func addHostFromSync(store *db.Store, h SyncHost, keyData string) error {
	if h.GroupName != "" {
		if err := store.UpsertGroup(h.GroupName); err != nil {
			return err
		}
	}

	host := &db.HostModel{
		ID:            h.ID,
		Label:         h.Label,
		GroupName:     h.GroupName,
		Tags:          db.NormalizeTags(h.Tags),
		Hostname:      h.Hostname,
		Username:      h.Username,
		Port:          h.Port,
		KeyType:       h.KeyType,
		CreatedAt:     h.CreatedAt,
		UpdatedAt:     h.UpdatedAt,
		LastConnected: h.LastConnected,
	}

	// Use CreateHostWithID to preserve the remote ID
	return store.CreateHostWithID(host, keyData)
}

// updateHostFromSync updates an existing host with sync data.
func updateHostFromSync(store *db.Store, h SyncHost, keyData string) error {
	if h.GroupName != "" {
		if err := store.UpsertGroup(h.GroupName); err != nil {
			return err
		}
	}

	host := &db.HostModel{
		ID:            h.ID,
		Label:         h.Label,
		GroupName:     h.GroupName,
		Tags:          db.NormalizeTags(h.Tags),
		Hostname:      h.Hostname,
		Username:      h.Username,
		Port:          h.Port,
		KeyType:       h.KeyType,
		CreatedAt:     h.CreatedAt,
		UpdatedAt:     h.UpdatedAt,
		LastConnected: h.LastConnected,
	}

	// Use UpdateHostFromSync to set exact values including encrypted key
	return store.UpdateHostFromSync(host, keyData, h.UpdatedAt)
}

// Merge combines local and remote data, returning the merged result.
// This is useful for preparing data to push after a pull.
func Merge(local, remote *SyncData) *SyncData {
	if local == nil && remote == nil {
		return &SyncData{
			Version:   CurrentSyncVersion,
			UpdatedAt: time.Now(),
			Groups:    []SyncGroup{},
			Hosts:     []SyncHost{},
		}
	}

	if local == nil {
		return remote
	}

	if remote == nil {
		return local
	}

	// Merge groups (prefer newer UpdatedAt by name)
	mergedGroups := make(map[string]SyncGroup)
	for _, g := range remote.Groups {
		k := strings.ToLower(strings.TrimSpace(g.Name))
		if k == "" {
			continue
		}
		mergedGroups[k] = g
	}
	for _, g := range local.Groups {
		k := strings.ToLower(strings.TrimSpace(g.Name))
		if k == "" {
			continue
		}
		if existing, ok := mergedGroups[k]; !ok || g.UpdatedAt.After(existing.UpdatedAt) {
			mergedGroups[k] = g
		}
	}
	groups := make([]SyncGroup, 0, len(mergedGroups))
	for _, g := range mergedGroups {
		groups = append(groups, g)
	}

	// Build map of all hosts, preferring the newer version
	merged := make(map[int]SyncHost)

	for _, h := range remote.Hosts {
		merged[h.ID] = h
	}

	for _, h := range local.Hosts {
		existing, exists := merged[h.ID]
		if !exists || h.UpdatedAt.After(existing.UpdatedAt) {
			merged[h.ID] = h
		}
	}

	// Convert back to slice
	hosts := make([]SyncHost, 0, len(merged))
	for _, h := range merged {
		hosts = append(hosts, h)
	}

	return &SyncData{
		Version:   CurrentSyncVersion,
		UpdatedAt: time.Now(),
		Groups:    groups,
		Hosts:     hosts,
	}
}
