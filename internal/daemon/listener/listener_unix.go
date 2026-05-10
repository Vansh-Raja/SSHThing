//go:build !windows

// Package listener provides a platform-appropriate net.Listener for the daemon IPC socket.
package listener

import (
	"net"
	"os"
	"syscall"
)

// Listen creates a Unix domain socket listener at the given path.
// Any stale socket file from a previous crash is removed first. The
// socket file's mode is forced to 0600 so foreign users on the same
// host can't even open(2) it — defence in depth on top of the
// per-request auth token. Without this the socket inherits the
// process umask (typically 0022 → world-readable on macOS).
func Listen(sockPath string) (net.Listener, error) {
	_ = os.Remove(sockPath)
	// Tighten umask just for the socket creation so the resulting
	// inode is 0600 regardless of the parent process's umask. The
	// previous umask is restored before we return.
	old := syscall.Umask(0o077)
	defer syscall.Umask(old)

	l, err := net.Listen("unix", sockPath)
	if err != nil {
		return nil, err
	}
	// Belt and braces: explicitly chmod in case the umask trick was
	// undone by a runtime that thinks it knows better.
	if err := os.Chmod(sockPath, 0o600); err != nil {
		_ = l.Close()
		return nil, err
	}
	return l, nil
}
