package service

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Vansh-Raja/SSHThing/internal/config"
	"github.com/Vansh-Raja/SSHThing/internal/personalsync"
	syncpkg "github.com/Vansh-Raja/SSHThing/internal/sync"
	"github.com/Vansh-Raja/SSHThing/internal/teamssession"
	"github.com/Vansh-Raja/SSHThing/internal/unlock"
)

// SyncService wraps a sync.Manager with lazy construction so the daemon can
// serve sync RPCs even before the user has configured a sync provider.
// Vault + CfgStore are used at call time (not construction time) so the
// service reflects the current vault and config state.
type SyncService struct {
	Vault    *Vault
	CfgStore *CfgStore
	// Client satisfies sync.PersonalCloudClient for Convex cloud sync.
	// *teamsclient.Client implements this interface.
	Client   syncpkg.PersonalCloudClient
	Notify   func(method string, params any)
}

// SyncStatusResult is returned by sync.status.
type SyncStatusResult struct {
	Provider         string `json:"provider"`
	Status           string `json:"status"`
	Stage            string `json:"stage,omitempty"`
	LastResultAt     int64  `json:"lastResultAt,omitempty"`   // Unix seconds
	LastResultStatus string `json:"lastResultStatus,omitempty"` // "success" | "error" | ""
	LastMessage      string `json:"lastMessage,omitempty"`
	DirtyCount       int    `json:"dirtyCount"` // placeholder — 0 until we compute it
}

// ConfigureParams are the fields accepted by sync.configure.
type ConfigureParams struct {
	Provider   string `json:"provider"`   // "off" | "git" | "cloud"
	RepoURL    string `json:"repoUrl,omitempty"`
	Branch     string `json:"branch,omitempty"`
	SSHKeyPath string `json:"sshKeyPath,omitempty"`
	Enabled    *bool  `json:"enabled,omitempty"`
}

// buildManager constructs a sync.Manager using the current config + vault state.
// Returns an error if the vault is locked.
func (s *SyncService) buildManager(ctx context.Context) (*syncpkg.Manager, error) {
	store := s.Vault.Store()
	if store == nil {
		return nil, ErrVaultLocked
	}

	password, _, ok, err := unlock.Load()
	if err != nil || !ok || password == "" {
		return nil, fmt.Errorf("vault password not available (vault locked or session expired)")
	}

	cfg := s.CfgStore.Get()

	opts := syncpkg.ManagerOptions{}
	if cfg.Sync.Provider == config.SyncProviderConvex && s.Client != nil {
		opts.CloudClient = s.Client
		opts.AccessTokenProvider = func(ctx context.Context) (string, error) {
			sess, err := teamssession.Load()
			if err != nil {
				return "", fmt.Errorf("load session: %w", err)
			}
			if sess.AccessToken == "" {
				return "", ErrNotSignedIn
			}
			return sess.AccessToken, nil
		}
		opts.DeviceID = syncDeviceID()
	}

	mgr, err := syncpkg.NewManagerWithOptions(&cfg, store, password, opts)
	if err != nil {
		return nil, fmt.Errorf("build sync manager: %w", err)
	}
	return mgr, nil
}

// syncDeviceID returns a hostname-based device identifier, prefixed to
// distinguish the desktop from the TUI ("sshthing-tui-<host>").
func syncDeviceID() string {
	host, err := os.Hostname()
	if err != nil || strings.TrimSpace(host) == "" {
		return "sshthing-desktop"
	}
	return "sshthing-desktop-" + strings.TrimSpace(host)
}

// Status returns the current sync status without triggering a sync.
func (s *SyncService) Status(ctx context.Context) (*SyncStatusResult, error) {
	mgr, err := s.buildManager(ctx)
	if err != nil {
		// Return a disabled status if vault is locked — not an error to the renderer.
		if err == ErrVaultLocked {
			cfg := s.CfgStore.Get()
			return &SyncStatusResult{
				Provider: string(cfg.Sync.Provider),
				Status:   "disabled",
			}, nil
		}
		return nil, err
	}

	cfg := s.CfgStore.Get()
	result := &SyncStatusResult{
		Provider: string(cfg.Sync.Provider),
		Status:   mgr.StatusString(),
		Stage:    mgr.StageString(),
	}

	if last := mgr.GetLastResult(); last != nil {
		result.LastResultAt = last.Timestamp.Unix()
		if last.Success {
			result.LastResultStatus = "success"
		} else {
			result.LastResultStatus = "error"
		}
		result.LastMessage = last.Message
	}

	return result, nil
}

// Now triggers a full sync (pull → import → export → push) and emits
// sync.progress notifications per stage.
func (s *SyncService) Now(ctx context.Context) (*syncpkg.SyncResult, error) {
	mgr, err := s.buildManager(ctx)
	if err != nil {
		return nil, err
	}

	// Init the provider (e.g. git clone / Convex handshake) before syncing.
	if initErr := mgr.Init(); initErr != nil {
		return nil, fmt.Errorf("sync init: %w", initErr)
	}

	if s.Notify != nil {
		s.Notify("sync.progress", map[string]any{"stage": "syncing"})
	}

	result := mgr.Sync()

	if s.Notify != nil {
		status := "success"
		if !result.Success {
			status = "error"
		}
		s.Notify("sync.progress", map[string]any{
			"stage":   "done",
			"status":  status,
			"message": result.Message,
		})
	}

	if !result.Success {
		msg := result.Message
		if msg == "" {
			msg = "sync failed"
		}
		return result, fmt.Errorf("%s", msg)
	}

	return result, nil
}

// Events returns the last 50 sync events for the authenticated user from the
// personal cloud. Only meaningful when provider == "convex".
func (s *SyncService) Events(ctx context.Context) ([]personalsync.SyncEvent, error) {
	cfg := s.CfgStore.Get()
	if cfg.Sync.Provider != config.SyncProviderConvex || s.Client == nil {
		return []personalsync.SyncEvent{}, nil
	}

	sess, err := teamssession.Load()
	if err != nil || sess.AccessToken == "" {
		return nil, ErrNotSignedIn
	}

	events, err := s.Client.ListPersonalSyncEvents(ctx, sess.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("list sync events: %w", err)
	}
	if events == nil {
		events = []personalsync.SyncEvent{}
	}
	return events, nil
}

// Devices returns the list of devices that have synced with this vault.
// NOTE: The backend does not yet expose a list-devices endpoint, so this
// always returns an empty slice.  The UI handles the empty state gracefully.
// TODO: implement when convex personalVaults.listDevices exists.
func (s *SyncService) Devices(_ context.Context) []map[string]any {
	return []map[string]any{}
}

// ForgetDevice removes a device from the sync device registry.
// NOTE: Stubbed — no backend endpoint exists yet.
// TODO: implement when convex personalVaults.forgetDevice exists.
func (s *SyncService) ForgetDevice(_ context.Context, _ string) error {
	return nil
}

// TestGitResult is the result of a git connectivity test.
type TestGitResult struct {
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
}

// TestGit validates a git repository URL and SSH key path by attempting
// a lightweight remote references listing via go-git.
// Useful for validating the git sync wizard input before saving config.
func (s *SyncService) TestGit(ctx context.Context, repoURL, sshKeyPath string) TestGitResult {
	if strings.TrimSpace(repoURL) == "" {
		return TestGitResult{OK: false, Message: "repoUrl is required"}
	}

	if err := syncpkg.TestGitConnectivity(ctx, repoURL, sshKeyPath); err != nil {
		return TestGitResult{OK: false, Message: err.Error()}
	}
	return TestGitResult{OK: true}
}

// Configure updates the sync configuration and persists it.
func (s *SyncService) Configure(params ConfigureParams) error {
	cfg := s.CfgStore.Get()

	if params.Provider != "" {
		cfg.Sync.Provider = config.SyncProvider(params.Provider)
		// Enabling/disabling follows from provider selection.
		cfg.Sync.Enabled = cfg.Sync.Provider != config.SyncProviderOff
	}
	if params.Enabled != nil {
		cfg.Sync.Enabled = *params.Enabled
	}
	if params.RepoURL != "" {
		cfg.Sync.RepoURL = params.RepoURL
	}
	if params.Branch != "" {
		cfg.Sync.Branch = params.Branch
	}
	if params.SSHKeyPath != "" {
		cfg.Sync.SSHKeyPath = params.SSHKeyPath
	}

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	s.CfgStore.Set(cfg)
	return nil
}
