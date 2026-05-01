package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

type HostKeyPolicy string

const (
	HostKeyAcceptNew HostKeyPolicy = "accept-new"
	HostKeyStrict    HostKeyPolicy = "strict"
	HostKeyOff       HostKeyPolicy = "off"
)

type TermMode string

const (
	TermAuto   TermMode = "auto"
	TermXterm  TermMode = "xterm-256color"
	TermCustom TermMode = "custom"
)

type PasswordBackendUnix string

const (
	PasswordBackendSSHPassFirst PasswordBackendUnix = "sshpass_first"
	PasswordBackendAskpassFirst PasswordBackendUnix = "askpass_first"
)

type MountQuitBehavior string

const (
	MountQuitPrompt        MountQuitBehavior = "prompt"
	MountQuitAlwaysUnmount MountQuitBehavior = "always_unmount"
	MountQuitLeaveMounted  MountQuitBehavior = "leave_mounted"
)

type HealthDisplayMode string

const (
	HealthDisplayMinimal     HealthDisplayMode = "minimal"
	HealthDisplayValues      HealthDisplayMode = "values"
	HealthDisplayGraphValues HealthDisplayMode = "graph_values"
)

// SyncAuthMethod represents the authentication method for Git sync
type SyncAuthMethod string

const (
	SyncAuthSSHKey SyncAuthMethod = "ssh_key"
	SyncAuthNone   SyncAuthMethod = "none"
)

type Config struct {
	Version int `json:"version"`

	UI struct {
		VimMode           bool              `json:"vim_mode"`
		ShowIcons         bool              `json:"show_icons"`
		WrapLabels        bool              `json:"wrap_labels"`
		ShowHealthDetails bool              `json:"show_health_details,omitempty"`
		HealthDisplayMode HealthDisplayMode `json:"health_display_mode,omitempty"`
		Theme             string            `json:"theme"`
		IconSet           string            `json:"icon_set"`
	} `json:"ui"`

	TeamsUI struct {
		WrapLabels bool   `json:"wrap_labels"`
		Theme      string `json:"theme"`
		IconSet    string `json:"icon_set"`
	} `json:"teams_ui"`

	SSH struct {
		HostKeyPolicy       HostKeyPolicy       `json:"host_key_policy"`
		KeepAliveSeconds    int                 `json:"keepalive_seconds"`
		TermMode            TermMode            `json:"term_mode"`
		TermCustom          string              `json:"term_custom"`
		PasswordAutoLogin   bool                `json:"password_auto_login"`
		PasswordNoticeShown bool                `json:"password_notice_shown,omitempty"`
		PasswordBackendUnix PasswordBackendUnix `json:"password_backend_unix"`
	} `json:"ssh"`

	Mount struct {
		Enabled           bool              `json:"enabled"`
		DefaultRemotePath string            `json:"default_remote_path"`
		LocalMountPath    string            `json:"local_mount_path,omitempty"`
		QuitBehavior      MountQuitBehavior `json:"quit_behavior"`
	} `json:"mount"`

	Sync struct {
		Enabled    bool           `json:"enabled"`
		RepoURL    string         `json:"repo_url"`
		AuthMethod SyncAuthMethod `json:"auth_method"`
		SSHKeyPath string         `json:"ssh_key_path"`
		Branch     string         `json:"branch"`
		LocalPath  string         `json:"local_path"`
	} `json:"sync"`

	Updates struct {
		LastCheckedAt    string `json:"last_checked_at,omitempty"`
		LastSeenVersion  string `json:"last_seen_version,omitempty"`
		LastSeenTag      string `json:"last_seen_tag,omitempty"`
		ETagLatest       string `json:"etag_latest,omitempty"`
		ReleaseChannel   string `json:"release_channel,omitempty"`
		AutoApplyUpdates bool   `json:"auto_apply_updates,omitempty"`
		ETagStable       string `json:"etag_stable,omitempty"`
		ETagBeta         string `json:"etag_beta,omitempty"`
	} `json:"updates"`

	Automation struct {
		SyncTokenDefinitions bool `json:"sync_token_definitions"`
		SessionTTLSeconds    int  `json:"session_ttl_seconds"`
	} `json:"automation"`

	Teams struct {
		Enabled             bool   `json:"enabled"`
		APIBaseURL          string `json:"api_base_url"`
		BrowserBaseURL      string `json:"browser_base_url"`
		SessionCacheEnabled bool   `json:"session_cache_enabled"`
		LastTeamID          string `json:"last_team_id,omitempty"`
	} `json:"teams"`
}

func Default() Config {
	var c Config
	c.Version = 8
	c.UI.VimMode = true
	c.UI.ShowIcons = true
	c.UI.WrapLabels = false
	c.UI.ShowHealthDetails = true
	c.UI.HealthDisplayMode = HealthDisplayGraphValues
	c.UI.Theme = "Catppuccin Mocha"
	c.UI.IconSet = "Unicode"
	c.TeamsUI.WrapLabels = false
	c.TeamsUI.Theme = "Catppuccin Latte"
	c.TeamsUI.IconSet = "ASCII"

	c.SSH.HostKeyPolicy = HostKeyAcceptNew
	c.SSH.KeepAliveSeconds = 60
	c.SSH.TermMode = TermAuto
	c.SSH.TermCustom = ""
	c.SSH.PasswordAutoLogin = true
	c.SSH.PasswordBackendUnix = PasswordBackendSSHPassFirst

	c.Mount.Enabled = true
	c.Mount.DefaultRemotePath = "" // empty means remote home
	c.Mount.QuitBehavior = MountQuitPrompt

	c.Sync.Enabled = false
	c.Sync.RepoURL = ""
	c.Sync.AuthMethod = SyncAuthSSHKey
	c.Sync.SSHKeyPath = ""
	c.Sync.Branch = "main"
	c.Sync.LocalPath = "" // empty means default path

	c.Automation.SyncTokenDefinitions = false
	c.Automation.SessionTTLSeconds = 900

	c.Teams.Enabled = false
	c.Teams.APIBaseURL = ""
	c.Teams.BrowserBaseURL = ""
	c.Teams.SessionCacheEnabled = true
	c.Updates.ReleaseChannel = "stable"
	c.Updates.AutoApplyUpdates = false
	c.Updates.ETagStable = ""
	c.Updates.ETagBeta = ""
	return c
}

// DataDir returns the base data directory for SSHThing.
// Respects SSHTHING_DATA_DIR env var for testing/custom setups.
func DataDir() (string, error) {
	if dir := os.Getenv("SSHTHING_DATA_DIR"); dir != "" {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return "", err
		}
		return dir, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "sshthing"), nil
}

func Path() (string, error) {
	dir, err := DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func Load() (Config, error) {
	path, err := Path()
	if err != nil {
		return Default(), err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Default(), nil
		}
		return Default(), err
	}
	var c Config
	if err := json.Unmarshal(b, &c); err != nil {
		return Default(), err
	}
	c = withDefaults(c)
	return c, nil
}

func Save(c Config) error {
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	c = withDefaults(c)
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func withDefaults(c Config) Config {
	def := Default()
	if c.Version == 0 {
		c.Version = def.Version
	}

	// Migrate from v1 → v2: PasswordAutoLogin default changed to true
	if c.Version < 2 {
		c.SSH.PasswordAutoLogin = true
		c.Version = 2
	}
	if c.Version < 3 {
		c.Teams.SessionCacheEnabled = true
		c.Version = 3
	}
	if c.Version < 4 {
		if c.TeamsUI.Theme == "" {
			c.TeamsUI.Theme = "Catppuccin Latte"
		}
		if c.TeamsUI.IconSet == "" {
			c.TeamsUI.IconSet = "ASCII"
		}
		c.Version = 4
	}
	if c.Version < 5 {
		c.UI.WrapLabels = false
		c.TeamsUI.WrapLabels = false
		c.Version = 5
	}
	if c.Version < 6 {
		if strings.TrimSpace(c.Updates.ReleaseChannel) == "" {
			c.Updates.ReleaseChannel = "stable"
		}
		c.Updates.AutoApplyUpdates = false
		if c.Updates.ETagStable == "" {
			c.Updates.ETagStable = c.Updates.ETagLatest
		}
		c.Version = 6
	}
	if c.Version < 7 {
		c.UI.ShowHealthDetails = true
		c.Version = 7
	}
	if c.Version < 8 {
		if c.UI.ShowHealthDetails {
			c.UI.HealthDisplayMode = HealthDisplayGraphValues
		} else {
			c.UI.HealthDisplayMode = HealthDisplayMinimal
		}
		c.Version = 8
	}

	// Enums / ints: normalize invalid values.
	switch c.SSH.HostKeyPolicy {
	case HostKeyAcceptNew, HostKeyStrict, HostKeyOff:
	default:
		c.SSH.HostKeyPolicy = def.SSH.HostKeyPolicy
	}
	if c.SSH.KeepAliveSeconds <= 0 || c.SSH.KeepAliveSeconds > 600 {
		c.SSH.KeepAliveSeconds = def.SSH.KeepAliveSeconds
	}
	switch c.SSH.TermMode {
	case TermAuto, TermXterm, TermCustom:
	default:
		c.SSH.TermMode = def.SSH.TermMode
	}
	switch c.SSH.PasswordBackendUnix {
	case PasswordBackendSSHPassFirst, PasswordBackendAskpassFirst:
	default:
		c.SSH.PasswordBackendUnix = def.SSH.PasswordBackendUnix
	}

	switch c.Mount.QuitBehavior {
	case MountQuitPrompt, MountQuitAlwaysUnmount, MountQuitLeaveMounted:
	default:
		c.Mount.QuitBehavior = def.Mount.QuitBehavior
	}

	// Sync defaults
	switch c.Sync.AuthMethod {
	case SyncAuthSSHKey, SyncAuthNone:
	default:
		c.Sync.AuthMethod = def.Sync.AuthMethod
	}
	if c.Sync.Branch == "" {
		c.Sync.Branch = def.Sync.Branch
	}

	if c.Automation.SessionTTLSeconds <= 0 || c.Automation.SessionTTLSeconds > 86400 {
		c.Automation.SessionTTLSeconds = def.Automation.SessionTTLSeconds
	}
	switch strings.ToLower(strings.TrimSpace(c.Updates.ReleaseChannel)) {
	case "", "stable":
		c.Updates.ReleaseChannel = "stable"
	case "beta":
		c.Updates.ReleaseChannel = "beta"
	default:
		c.Updates.ReleaseChannel = def.Updates.ReleaseChannel
	}
	if c.Updates.ETagStable == "" && c.Updates.ETagLatest != "" {
		c.Updates.ETagStable = c.Updates.ETagLatest
	}
	switch c.UI.HealthDisplayMode {
	case HealthDisplayMinimal, HealthDisplayValues, HealthDisplayGraphValues:
	default:
		c.UI.HealthDisplayMode = def.UI.HealthDisplayMode
	}

	if c.Teams.APIBaseURL == "" {
		c.Teams.APIBaseURL = def.Teams.APIBaseURL
	}
	if c.Teams.BrowserBaseURL == "" {
		c.Teams.BrowserBaseURL = def.Teams.BrowserBaseURL
	}
	if c.TeamsUI.Theme == "" {
		c.TeamsUI.Theme = def.TeamsUI.Theme
	}
	if c.TeamsUI.IconSet == "" {
		c.TeamsUI.IconSet = def.TeamsUI.IconSet
	}

	return c
}

// SyncPath returns the path to the sync repository directory.
// If LocalPath is set, it returns that; otherwise returns the default path.
func (c *Config) SyncPath() (string, error) {
	if c.Sync.LocalPath != "" {
		return c.Sync.LocalPath, nil
	}
	dir, err := DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "sync"), nil
}
