package teamssession

import (
	"testing"
	"time"
)

func TestSessionValidity(t *testing.T) {
	now := time.Now()
	s := Session{
		AccessToken:  "access",
		RefreshToken: "refresh",
		ExpiresAt:    now.Add(5 * time.Minute),
	}
	if !s.Valid(now) {
		t.Fatalf("expected session to be valid")
	}
	if !s.NeedsRefresh(now.Add(4*time.Minute), 2*time.Minute) {
		t.Fatalf("expected session to need refresh near expiry")
	}
}
