//go:build windows

// Package listener provides a platform-appropriate net.Listener for the daemon IPC socket.
package listener

import (
	"net"

	"github.com/Microsoft/go-winio"
)

// Listen creates a Windows named pipe listener at the given path.
// The security descriptor grants access only to the current user (SDDL: owner full control).
func Listen(sockPath string) (net.Listener, error) {
	cfg := &winio.PipeConfig{
		// SDDL that restricts the pipe to the current user only.
		SecurityDescriptor: "D:P(A;;GA;;;OW)",
	}
	return winio.ListenPipe(sockPath, cfg)
}
