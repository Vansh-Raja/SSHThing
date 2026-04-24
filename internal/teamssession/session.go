package teamssession

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/Vansh-Raja/SSHThing/internal/config"
)

const fileName = "teams_session.json"

type Session struct {
	AccessToken   string `json:"accessToken"`
	RefreshToken  string `json:"refreshToken"`
	ExpiresAt     int64  `json:"expiresAt"`
	CurrentTeamID string `json:"currentTeamId,omitempty"`
	UserID        string `json:"userId"`
	UserName      string `json:"userName"`
	UserEmail     string `json:"userEmail"`
}

func Path() (string, error) {
	dir, err := config.DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, fileName), nil
}

func Load() (Session, error) {
	path, err := Path()
	if err != nil {
		return Session{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Session{}, nil
		}
		return Session{}, err
	}
	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return Session{}, err
	}
	return session, nil
}

func Save(session Session) error {
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(session, "", "  ")
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

func (s Session) Valid() bool {
	return s.AccessToken != "" && s.RefreshToken != "" && s.ExpiresAt > 0
}

func (s Session) Refreshable() bool {
	return s.RefreshToken != ""
}

func (s Session) Expired(now time.Time) bool {
	if s.ExpiresAt == 0 {
		return true
	}
	return now.UnixMilli() >= s.ExpiresAt
}
