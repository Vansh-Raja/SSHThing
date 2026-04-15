package teamssession

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/Vansh-Raja/SSHThing/internal/config"
)

type Session struct {
	AccessToken  string    `json:"accessToken"`
	RefreshToken string    `json:"refreshToken"`
	ExpiresAt    time.Time `json:"expiresAt"`
	ActiveTeamID string    `json:"activeTeamId,omitempty"`
	UserID       string    `json:"userId,omitempty"`
	UserEmail    string    `json:"userEmail,omitempty"`
}

func Path() (string, error) {
	dir, err := config.DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "teams_session.json"), nil
}

func Load() (Session, bool, error) {
	path, err := Path()
	if err != nil {
		return Session{}, false, err
	}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Session{}, false, nil
	}
	if err != nil {
		return Session{}, false, err
	}
	var s Session
	if err := json.Unmarshal(b, &s); err != nil {
		return Session{}, false, err
	}
	return s, true, nil
}

func Save(s Session) error {
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func Clear() error {
	path, err := Path()
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func (s Session) Valid(now time.Time) bool {
	return s.AccessToken != "" && s.RefreshToken != "" && now.Before(s.ExpiresAt)
}

func (s Session) NeedsRefresh(now time.Time, within time.Duration) bool {
	if s.RefreshToken == "" || s.ExpiresAt.IsZero() {
		return false
	}
	return now.Add(within).After(s.ExpiresAt)
}
