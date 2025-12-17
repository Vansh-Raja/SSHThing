package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
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

type MountQuitBehavior string

const (
	MountQuitPrompt        MountQuitBehavior = "prompt"
	MountQuitAlwaysUnmount MountQuitBehavior = "always_unmount"
	MountQuitLeaveMounted  MountQuitBehavior = "leave_mounted"
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
		VimMode   bool `json:"vim_mode"`
		ShowIcons bool `json:"show_icons"`
	} `json:"ui"`

	SSH struct {
		HostKeyPolicy    HostKeyPolicy `json:"host_key_policy"`
		KeepAliveSeconds int           `json:"keepalive_seconds"`
		TermMode         TermMode      `json:"term_mode"`
		TermCustom       string        `json:"term_custom"`
	} `json:"ssh"`

	Mount struct {
		Enabled           bool              `json:"enabled"`
		DefaultRemotePath string            `json:"default_remote_path"`
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
}

func Default() Config {
	var c Config
	c.Version = 1
	c.UI.VimMode = true
	c.UI.ShowIcons = true

	c.SSH.HostKeyPolicy = HostKeyAcceptNew
	c.SSH.KeepAliveSeconds = 60
	c.SSH.TermMode = TermAuto
	c.SSH.TermCustom = ""

	c.Mount.Enabled = true
	c.Mount.DefaultRemotePath = "" // empty means remote home
	c.Mount.QuitBehavior = MountQuitPrompt

	c.Sync.Enabled = false
	c.Sync.RepoURL = ""
	c.Sync.AuthMethod = SyncAuthSSHKey
	c.Sync.SSHKeyPath = ""
	c.Sync.Branch = "main"
	c.Sync.LocalPath = "" // empty means default path
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

	// UI defaults
	// (bools default false, so we only fill if config version missing)
	if c.Version == def.Version {
		// keep as-is
	} else {
		// Future migration hook.
	}
	if !c.UI.VimMode && def.UI.VimMode {
		// If config explicitly set false, keep it; otherwise defaulting from zero-value
		// is ambiguous. We'll assume missing config implies default, but only when the
		// file was absent. Load() returns defaults in that case.
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

	// Keep UI defaults stable when fields are left at zero-values; callers should
	// prefer Default() when config file doesn't exist.
	if c.UI.ShowIcons == false && def.UI.ShowIcons {
		// Same note as above; leave as-is.
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
