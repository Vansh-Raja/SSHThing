package teamcache

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/Vansh-Raja/SSHThing/internal/config"
	"github.com/Vansh-Raja/SSHThing/internal/teams"
)

const fileName = "teams_cache.json"

type Cache struct {
	CurrentTeamID string              `json:"currentTeamId,omitempty"`
	Teams         []teams.TeamSummary `json:"teams,omitempty"`
	Hosts         []teams.TeamHost    `json:"hosts,omitempty"`
	LastFetchedAt int64               `json:"lastFetchedAt"`
}

func Path() (string, error) {
	dir, err := config.DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, fileName), nil
}

func Load() (Cache, error) {
	path, err := Path()
	if err != nil {
		return Cache{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Cache{}, nil
		}
		return Cache{}, err
	}
	var cache Cache
	if err := json.Unmarshal(data, &cache); err != nil {
		return Cache{}, err
	}
	return cache, nil
}

func Save(cache Cache) error {
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func Clear() error {
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (c Cache) LastFetched() time.Time {
	if c.LastFetchedAt == 0 {
		return time.Time{}
	}
	return time.UnixMilli(c.LastFetchedAt)
}
