package unlock

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Vansh-Raja/SSHThing/internal/securestore"
)

type cacheRecord struct {
	Version   int       `json:"version"`
	ExpiresAt time.Time `json:"expires_at"`
	Secret    string    `json:"secret"`
}

func Save(secret string, ttl time.Duration) error {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return fmt.Errorf("missing session secret")
	}
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	rec := cacheRecord{
		Version:   1,
		ExpiresAt: time.Now().UTC().Add(ttl),
		Secret:    secret,
	}
	b, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	return securestore.StoreSessionUnlock(string(b))
}

func Load() (secret string, expiresAt time.Time, ok bool, err error) {
	v, err := securestore.LoadSessionUnlock()
	if err != nil {
		return "", time.Time{}, false, err
	}
	var rec cacheRecord
	if err := json.Unmarshal([]byte(v), &rec); err != nil {
		return "", time.Time{}, false, err
	}
	if time.Now().UTC().After(rec.ExpiresAt) {
		_ = securestore.ClearSessionUnlock()
		return "", rec.ExpiresAt, false, nil
	}
	return rec.Secret, rec.ExpiresAt, strings.TrimSpace(rec.Secret) != "", nil
}

func Clear() error {
	return securestore.ClearSessionUnlock()
}
