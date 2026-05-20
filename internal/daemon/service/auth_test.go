package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Vansh-Raja/SSHThing/internal/daemon/service"
	"github.com/Vansh-Raja/SSHThing/internal/teams"
	"github.com/Vansh-Raja/SSHThing/internal/teamssession"
)

// mockCLIAuthClient is an in-memory mock that satisfies service.CLIAuthClient.
type mockCLIAuthClient struct {
	startResp teams.CliAuthStartResponse
	startErr  error
	pollResp  teams.CliAuthPollResponse
	pollErr   error
	logoutErr error
	enabled   bool
}

func (m *mockCLIAuthClient) StartCLIAuth(_ context.Context, _ string, _ bool) (teams.CliAuthStartResponse, error) {
	return m.startResp, m.startErr
}

func (m *mockCLIAuthClient) PollCLIAuth(_ context.Context, _, _ string) (teams.CliAuthPollResponse, error) {
	return m.pollResp, m.pollErr
}

func (m *mockCLIAuthClient) Logout(_ context.Context, _ string) error {
	return m.logoutErr
}

func (m *mockCLIAuthClient) Enabled() bool {
	return m.enabled
}

// TestAuthStartSignIn_ShapeAndAuthURL verifies that StartSignIn returns the
// server-provided AuthURL directly (not reformatted) and includes all fields.
func TestAuthStartSignIn_ShapeAndAuthURL(t *testing.T) {
	mock := &mockCLIAuthClient{
		enabled: true,
		startResp: teams.CliAuthStartResponse{
			AuthURL:             "https://sshthing.com/cli-auth/complete?session=abc&code=xyz",
			SessionID:           "sess-1",
			DeviceCode:          "device-code-1",
			PollSecret:          "poll-secret-1",
			PollIntervalSeconds: 3,
			ExpiresAt:           time.Now().Add(10 * time.Minute).UnixMilli(),
		},
	}

	var notified []string
	svc := &service.AuthService{
		Client: mock,
		Notify: func(method string, _ any) {
			notified = append(notified, method)
		},
	}

	result, err := svc.StartSignIn(context.Background())
	if err != nil {
		t.Fatalf("StartSignIn returned error: %v", err)
	}

	// URL must be exactly the server URL — not re-formatted.
	if result.URL != mock.startResp.AuthURL {
		t.Errorf("URL = %q; want %q", result.URL, mock.startResp.AuthURL)
	}
	if result.SessionID != "sess-1" {
		t.Errorf("SessionID = %q; want sess-1", result.SessionID)
	}
	if result.PollIntervalSeconds != 3 {
		t.Errorf("PollIntervalSeconds = %d; want 3", result.PollIntervalSeconds)
	}
	if result.ExpiresAt == 0 {
		t.Error("ExpiresAt should be non-zero")
	}
	// No notification on startSignIn.
	if len(notified) != 0 {
		t.Errorf("unexpected notifications: %v", notified)
	}
}

// TestAuthStartSignIn_DefaultPollInterval verifies the fallback when server
// returns PollIntervalSeconds == 0.
func TestAuthStartSignIn_DefaultPollInterval(t *testing.T) {
	mock := &mockCLIAuthClient{
		enabled: true,
		startResp: teams.CliAuthStartResponse{
			AuthURL:             "https://sshthing.com/cli-auth/complete",
			SessionID:           "sess-2",
			PollIntervalSeconds: 0, // server didn't set it
			ExpiresAt:           time.Now().Add(10 * time.Minute).UnixMilli(),
		},
	}
	svc := &service.AuthService{Client: mock, Notify: func(string, any) {}}
	result, err := svc.StartSignIn(context.Background())
	if err != nil {
		t.Fatalf("StartSignIn: %v", err)
	}
	if result.PollIntervalSeconds != 2 {
		t.Errorf("PollIntervalSeconds = %d; want 2 (default)", result.PollIntervalSeconds)
	}
}

// TestAuthStartSignIn_ClientDisabled verifies that a disabled client returns an error.
func TestAuthStartSignIn_ClientDisabled(t *testing.T) {
	svc := &service.AuthService{
		Client: &mockCLIAuthClient{enabled: false},
		Notify: func(string, any) {},
	}
	_, err := svc.StartSignIn(context.Background())
	if err == nil {
		t.Fatal("expected error when client disabled, got nil")
	}
}

// TestAuthPollSignIn_PendingStatus verifies pending response passes through.
func TestAuthPollSignIn_PendingStatus(t *testing.T) {
	mock := &mockCLIAuthClient{
		enabled:  true,
		pollResp: teams.CliAuthPollResponse{Status: "pending"},
	}
	svc := &service.AuthService{Client: mock, Notify: func(string, any) {}}
	result, err := svc.PollSignIn(context.Background(), "sess", "secret")
	if err != nil {
		t.Fatalf("PollSignIn: %v", err)
	}
	if result.Status != "pending" {
		t.Errorf("Status = %q; want pending", result.Status)
	}
	if result.Session != nil {
		t.Error("Session should be nil on pending")
	}
}

// TestAuthPollSignIn_CompletedSavesSession verifies that on "completed":
//   - teamssession is saved synchronously
//   - the returned Session is populated
//   - auth.signedIn notification is emitted
//   - tokens are NOT in the response
//
// This test writes to the real teamssession file and cleans up.
func TestAuthPollSignIn_CompletedSavesSession(t *testing.T) {
	// Clear any pre-existing session to avoid interference.
	_ = teamssession.Clear()
	t.Cleanup(func() { _ = teamssession.Clear() })

	expiresAt := time.Now().Add(1 * time.Hour).UnixMilli()
	mock := &mockCLIAuthClient{
		enabled: true,
		pollResp: teams.CliAuthPollResponse{
			Status:       "completed",
			AccessToken:  "access-tok-abc",
			RefreshToken: "refresh-tok-xyz",
			ExpiresAt:    expiresAt,
			User: &teams.UserSummary{
				ID:    "user-1",
				Name:  "Alice",
				Email: "alice@example.com",
			},
		},
	}

	var notified []string
	svc := &service.AuthService{
		Client: mock,
		Notify: func(method string, _ any) {
			notified = append(notified, method)
		},
	}

	result, err := svc.PollSignIn(context.Background(), "sess", "secret")
	if err != nil {
		t.Fatalf("PollSignIn: %v", err)
	}

	// Status must be "completed".
	if result.Status != "completed" {
		t.Errorf("Status = %q; want completed", result.Status)
	}

	// Session must be populated with safe user info.
	if result.Session == nil {
		t.Fatal("Session must not be nil on completed")
	}
	if result.Session.UserID != "user-1" {
		t.Errorf("Session.UserID = %q; want user-1", result.Session.UserID)
	}
	if result.Session.UserName != "Alice" {
		t.Errorf("Session.UserName = %q; want Alice", result.Session.UserName)
	}
	if result.Session.UserEmail != "alice@example.com" {
		t.Errorf("Session.UserEmail = %q; want alice@example.com", result.Session.UserEmail)
	}

	// auth.signedIn notification must have been emitted.
	if len(notified) == 0 || notified[0] != "auth.signedIn" {
		t.Errorf("notifications = %v; want [auth.signedIn]", notified)
	}

	// Session must be persisted to disk.
	saved, loadErr := teamssession.Load()
	if loadErr != nil {
		t.Fatalf("teamssession.Load after poll: %v", loadErr)
	}
	if saved.AccessToken != "access-tok-abc" {
		t.Errorf("saved.AccessToken = %q; want access-tok-abc", saved.AccessToken)
	}
	if saved.UserID != "user-1" {
		t.Errorf("saved.UserID = %q; want user-1", saved.UserID)
	}
}

// TestAuthSession_NilWhenNoSession verifies that Session() returns nil, nil
// when no session exists.
func TestAuthSession_NilWhenNoSession(t *testing.T) {
	_ = teamssession.Clear()
	t.Cleanup(func() { _ = teamssession.Clear() })

	svc := &service.AuthService{Client: &mockCLIAuthClient{enabled: true}, Notify: func(string, any) {}}
	info, err := svc.Session()
	if err != nil {
		t.Fatalf("Session: %v", err)
	}
	if info != nil {
		t.Errorf("Session() = %+v; want nil", info)
	}
}

// TestAuthSignOut_EmitsNotification verifies sign-out clears session and notifies.
func TestAuthSignOut_EmitsNotification(t *testing.T) {
	// Plant a session.
	_ = teamssession.Save(teamssession.Session{
		AccessToken:  "tok",
		RefreshToken: "ref",
		ExpiresAt:    time.Now().Add(time.Hour).UnixMilli(),
		UserID:       "u1",
	})
	t.Cleanup(func() { _ = teamssession.Clear() })

	var notified []string
	mock := &mockCLIAuthClient{enabled: true}
	svc := &service.AuthService{
		Client: mock,
		Notify: func(method string, _ any) {
			notified = append(notified, method)
		},
	}

	if err := svc.SignOut(context.Background()); err != nil {
		t.Fatalf("SignOut: %v", err)
	}

	// Session must be cleared.
	sess, _ := teamssession.Load()
	if sess.AccessToken != "" {
		t.Error("session not cleared after SignOut")
	}

	// auth.signedOut notification must be emitted.
	if len(notified) == 0 || notified[0] != "auth.signedOut" {
		t.Errorf("notifications = %v; want [auth.signedOut]", notified)
	}
}

// TestAuthTokenForRenderer_NotSignedIn verifies ErrNotSignedIn is returned.
func TestAuthTokenForRenderer_NotSignedIn(t *testing.T) {
	_ = teamssession.Clear()
	t.Cleanup(func() { _ = teamssession.Clear() })

	svc := &service.AuthService{Client: &mockCLIAuthClient{enabled: true}, Notify: func(string, any) {}}
	_, err := svc.TokenForRenderer()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, service.ErrNotSignedIn) {
		t.Errorf("err = %v; want ErrNotSignedIn", err)
	}
}
