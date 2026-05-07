//go:build !windows

// Package listener provides a platform-appropriate net.Listener for the daemon IPC socket.
package listener

import (
	"net"
	"os"
)

// Listen creates a Unix domain socket listener at the given path.
// Any stale socket file from a previous crash is removed first.
func Listen(sockPath string) (net.Listener, error) {
	// Remove stale socket from a previous crash so we don't get "address already in use".
	_ = os.Remove(sockPath)
	return net.Listen("unix", sockPath)
}
