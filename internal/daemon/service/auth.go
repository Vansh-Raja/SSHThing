package service

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"os/exec"
	"strings"
	"time"

	"github.com/Vansh-Raja/SSHThing/internal/teams"
	"github.com/Vansh-Raja/SSHThing/internal/teamsclient"
	"github.com/Vansh-Raja/SSHThing/internal/teamssession"
)

// CLIAuthClient is the minimal interface required by AuthService for sign-in.
// Using an interface (not *teamsclient.Client directly) makes the service
// testable without a live HTTP server.
type CLIAuthClient interface {
	StartCLIAuth(ctx context.Context, deviceName string, headless bool) (teams.CliAuthStartResponse, error)
	PollCLIAuth(ctx context.Context, sessionID, pollSecret string) (teams.CliAuthPollResponse, error)
	Logout(ctx context.Context, refreshToken string) error
	Enabled() bool
}

// Compile-time check that *teamsclient.Client satisfies CLIAuthClient.
var _ CLIAuthClient = (*teamsclient.Client)(nil)

// AuthService provides Convex device-code sign-in / sign-out RPCs.
// Notify is wired by main.go to srv.Notify so auth events reach the renderer.
type AuthService struct {
	Client CLIAuthClient
	Notify func(method string, params any)
}

// StartSignInResult is returned by auth.startSignIn.
type StartSignInResult struct {
	URL                 string `json:"url"`
	SessionID           string `json:"sessionId"`
	DeviceCode          string `json:"deviceCode"`
	PollSecret          string `json:"pollSecret"`
	PollIntervalSeconds int    `json:"pollIntervalSeconds"`
	ExpiresAt           int64  `json:"expiresAt"`
}

// PollSignInResult is returned by auth.pollSignIn.
// Session is non-nil only when Status == "completed".
// Tokens are NEVER included — only safe user info is returned.
type PollSignInResult struct {
	Status  string       `json:"status"` // "pending" | "completed" | "expired"
	Session *SessionInfo `json:"session,omitempty"`
}

// SessionInfo is the safe subset of teamssession.Session exposed to the renderer.
// Access/refresh tokens are never included.
type SessionInfo struct {
	UserID        string `json:"userId"`
	UserName      string `json:"userName"`
	UserEmail     string `json:"userEmail"`
	CurrentTeamID string `json:"currentTeamId,omitempty"`
	ExpiresAt     int64  `json:"expiresAt"`
}

// StartSignIn calls the server's CLI auth start endpoint and returns the
// auth URL and polling parameters.
func (a *AuthService) StartSignIn(ctx context.Context) (*StartSignInResult, error) {
	if a.Client == nil || !a.Client.Enabled() {
		return nil, fmt.Errorf("teams client not configured; set Teams.APIBaseURL in config")
	}
	resp, err := a.Client.StartCLIAuth(ctx, "SSHThing Desktop", false)
	if err != nil {
		return nil, fmt.Errorf("start CLI auth: %w", err)
	}
	pollInterval := resp.PollIntervalSeconds
	if pollInterval <= 0 {
		pollInterval = 2
	}
	return &StartSignInResult{
		URL:                 resp.AuthURL,
		SessionID:           resp.SessionID,
		DeviceCode:          resp.DeviceCode,
		PollSecret:          resp.PollSecret,
		PollIntervalSeconds: pollInterval,
		ExpiresAt:           resp.ExpiresAt,
	}, nil
}

// OpenBrowser shells out to the platform browser opener.
func (a *AuthService) OpenBrowser(url string) error {
	url = strings.TrimSpace(url)
	if url == "" {
		return fmt.Errorf("url is required")
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/C", "start", url)
	default: // linux, etc.
		cmd = exec.Command("xdg-open", url)
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// PollSignIn checks the CLI auth session status.
// On "completed": saves the session to disk BEFORE returning so the subsequent
// auth.session() call always finds a valid session.
func (a *AuthService) PollSignIn(ctx context.Context, sessionID, pollSecret string) (*PollSignInResult, error) {
	if a.Client == nil || !a.Client.Enabled() {
		return nil, fmt.Errorf("teams client not configured")
	}
	resp, err := a.Client.PollCLIAuth(ctx, sessionID, pollSecret)
	if err != nil {
		return nil, fmt.Errorf("poll CLI auth: %w", err)
	}

	if resp.Status != "completed" {
		return &PollSignInResult{Status: resp.Status}, nil
	}

	// Build and persist the session synchronously before returning so
	// auth.session() always finds a valid session after status=="completed".
	sess := teamssession.Session{
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
		ExpiresAt:    resp.ExpiresAt,
	}
	if resp.User != nil {
		sess.UserID = resp.User.ID
		sess.UserName = resp.User.Name
		sess.UserEmail = resp.User.Email
	}
	if err := teamssession.Save(sess); err != nil {
		return nil, fmt.Errorf("save session: %w", err)
	}

	// Emit sign-in notification to all renderer windows.
	if a.Notify != nil {
		a.Notify("auth.signedIn", map[string]any{
			"userId":    sess.UserID,
			"userName":  sess.UserName,
			"userEmail": sess.UserEmail,
		})
	}

	return &PollSignInResult{
		Status: "completed",
		Session: &SessionInfo{
			UserID:        sess.UserID,
			UserName:      sess.UserName,
			UserEmail:     sess.UserEmail,
			CurrentTeamID: sess.CurrentTeamID,
			ExpiresAt:     sess.ExpiresAt,
		},
	}, nil
}

// SignOut clears the local session and, if the client is available and we have
// a refresh token, revokes the server-side token.
func (a *AuthService) SignOut(ctx context.Context) error {
	sess, err := teamssession.Load()
	if err != nil {
		// Can't load — still clear.
		_ = teamssession.Clear()
		return nil
	}

	// Attempt server-side logout (best-effort).
	if a.Client != nil && a.Client.Enabled() && sess.RefreshToken != "" {
		_ = a.Client.Logout(ctx, sess.RefreshToken)
	}

	if err := teamssession.Clear(); err != nil {
		return fmt.Errorf("clear session: %w", err)
	}

	if a.Notify != nil {
		a.Notify("auth.signedOut", map[string]any{})
	}
	return nil
}

// Session returns the current session info (no tokens).
// Returns nil, nil when no session is active.
func (a *AuthService) Session() (*SessionInfo, error) {
	sess, err := teamssession.Load()
	if err != nil {
		return nil, fmt.Errorf("load session: %w", err)
	}
	if sess.AccessToken == "" {
		return nil, nil
	}
	return &SessionInfo{
		UserID:        sess.UserID,
		UserName:      sess.UserName,
		UserEmail:     sess.UserEmail,
		CurrentTeamID: sess.CurrentTeamID,
		ExpiresAt:     sess.ExpiresAt,
	}, nil
}

// TokenForRenderer returns the current access token string only.
// This is the only place the access token leaves the daemon.
func (a *AuthService) TokenForRenderer() (string, error) {
	sess, err := teamssession.Load()
	if err != nil {
		return "", fmt.Errorf("load session: %w", err)
	}
	if sess.AccessToken == "" {
		return "", ErrNotSignedIn
	}
	// Check expiry.
	if sess.Expired(time.Now()) {
		return "", fmt.Errorf("session expired")
	}
	return sess.AccessToken, nil
}
