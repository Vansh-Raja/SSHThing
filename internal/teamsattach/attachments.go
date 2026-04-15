package teamsattach

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/Vansh-Raja/SSHThing/internal/config"
)

type AttachmentType string

const (
	AttachmentTypeExistingPersonalHost  AttachmentType = "existing_personal_host"
	AttachmentTypeDirectLocalCredential AttachmentType = "direct_local_credential"
)

type Attachment struct {
	TeamHostID      string         `json:"teamHostId"`
	AttachmentType  AttachmentType `json:"attachmentType"`
	PersonalHostID  int            `json:"personalHostId,omitempty"`
	DirectReference string         `json:"directReference,omitempty"`
}

type Store struct {
	Attachments []Attachment `json:"attachments"`
}

func Path() (string, error) {
	dir, err := config.DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "teams_attachments.json"), nil
}

func Load() (Store, error) {
	path, err := Path()
	if err != nil {
		return Store{}, err
	}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Store{}, nil
	}
	if err != nil {
		return Store{}, err
	}
	var s Store
	if err := json.Unmarshal(b, &s); err != nil {
		return Store{}, err
	}
	return s, nil
}

func Save(s Store) error {
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

func (s Store) Find(teamHostID string) (Attachment, bool) {
	for _, a := range s.Attachments {
		if a.TeamHostID == teamHostID {
			return a, true
		}
	}
	return Attachment{}, false
}

func (s *Store) Upsert(attachment Attachment) {
	for i, a := range s.Attachments {
		if a.TeamHostID == attachment.TeamHostID {
			s.Attachments[i] = attachment
			return
		}
	}
	s.Attachments = append(s.Attachments, attachment)
}

func (s *Store) Delete(teamHostID string) {
	next := s.Attachments[:0]
	for _, a := range s.Attachments {
		if a.TeamHostID != teamHostID {
			next = append(next, a)
		}
	}
	s.Attachments = next
}
