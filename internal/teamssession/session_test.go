package teamssession

import (
	"testing"
	"time"
)

func TestSessionSaveLoadClear(t *testing.T) {
	t.Setenv("SSHTHING_DATA_DIR", t.TempDir())

	session := Session{
		AccessToken:  "access",
		RefreshToken: "refresh",
		ExpiresAt:    time.Now().Add(5 * time.Minute).UnixMilli(),
		UserID:       "user_123",
		UserName:     "Test User",
		UserEmail:    "user@example.com",
	}
	if err := Save(session); err != nil {
		t.Fatalf("save session: %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("load session: %v", err)
	}
	if got.AccessToken != session.AccessToken || got.RefreshToken != session.RefreshToken || got.UserName != session.UserName {
		t.Fatalf("expected saved session values to round-trip")
	}

	if err := Clear(); err != nil {
		t.Fatalf("clear session: %v", err)
	}
	empty, err := Load()
	if err != nil {
		t.Fatalf("load cleared session: %v", err)
	}
	if empty.Valid() {
		t.Fatalf("expected cleared session to be invalid")
	}
}

func TestSessionExpiry(t *testing.T) {
	valid := Session{AccessToken: "a", RefreshToken: "b", ExpiresAt: time.Now().Add(time.Minute).UnixMilli()}
	if !valid.Valid() {
		t.Fatalf("expected session to be valid")
	}
	if valid.Expired(time.Now()) {
		t.Fatalf("expected session to be unexpired")
	}

	expired := Session{AccessToken: "a", RefreshToken: "b", ExpiresAt: time.Now().Add(-time.Minute).UnixMilli()}
	if !expired.Expired(time.Now()) {
		t.Fatalf("expected session to be expired")
	}
}
