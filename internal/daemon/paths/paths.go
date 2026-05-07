// Package paths resolves platform-specific filesystem paths for the daemon:
// socket, auth token, and log file.
package paths

import (
	"os"
	"path/filepath"
)

// DataDir returns the platform-specific data directory for SSHThing.
// macOS: ~/Library/Application Support/SSHThing
// Linux: $XDG_DATA_HOME/SSHThing (fallback ~/.local/share/SSHThing)
// Windows: handled in paths_windows.go
func DataDir() (string, error) {
	dir, err := dataDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return dir, nil
}

// SocketPath returns the platform-appropriate IPC socket path.
// On Unix this is a filesystem path; on Windows it is a named pipe path.
// Callers use this as the address for net.Listen / net.Dial.
func SocketPath() (string, error) {
	return socketPath()
}

// TokenPath returns the path to the daemon auth token file (mode 0600).
func TokenPath() (string, error) {
	dir, err := DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "daemon.token"), nil
}

// LogPath returns the path to the daemon log file.
func LogPath() (string, error) {
	dir, err := DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "daemon.log"), nil
}
