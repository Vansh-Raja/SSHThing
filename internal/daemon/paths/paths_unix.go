//go:build !windows

package paths

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
)

func dataDir() (string, error) {
	// Allow tests and smoke scripts to redirect all data files to a temp dir.
	if override := os.Getenv("SSHTHING_DATA_DIR"); override != "" {
		return override, nil
	}
	if runtime.GOOS == "darwin" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "Library", "Application Support", "SSHThing"), nil
	}

	// Linux: XDG_DATA_HOME or ~/.local/share
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "SSHThing"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "SSHThing"), nil
}

func socketPath() (string, error) {
	// When SSHTHING_DATA_DIR is set (e.g. smoke tests), put the socket there.
	if override := os.Getenv("SSHTHING_DATA_DIR"); override != "" {
		if err := os.MkdirAll(override, 0700); err != nil {
			return "", err
		}
		return filepath.Join(override, "daemon.sock"), nil
	}
	if runtime.GOOS == "darwin" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dir := filepath.Join(home, "Library", "Application Support", "SSHThing")
		if err := os.MkdirAll(dir, 0700); err != nil {
			return "", err
		}
		return filepath.Join(dir, "sshthing.sock"), nil
	}

	// Linux: prefer XDG_RUNTIME_DIR
	if xdg := os.Getenv("XDG_RUNTIME_DIR"); xdg != "" {
		return filepath.Join(xdg, "sshthing.sock"), nil
	}
	u, err := user.Current()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("/tmp/sshthing-%s.sock", u.Username), nil
}
