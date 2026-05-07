package rpc

import (
	"context"
	"encoding/json"
	"runtime"
	"time"

	"github.com/Vansh-Raja/SSHThing/internal/daemon/service"
)

// RegisterMeta registers meta RPC handlers on s.
// Handlers: daemon.version, daemon.health, daemon.shutdown.
func RegisterMeta(s *Server, sessions *service.Sessions, cancel context.CancelFunc, startedAt time.Time) {
	s.Register("daemon.version", handleDaemonVersion)
	s.Register("daemon.health", makeDaemonHealth(sessions, startedAt))
	s.Register("daemon.shutdown", makeDaemonShutdown(sessions, cancel))
}

func handleDaemonVersion(_ context.Context, _ uint64, _ json.RawMessage) (any, *RPCError) {
	return map[string]string{"version": "0.0.1-spike"}, nil
}

func makeDaemonHealth(sessions *service.Sessions, startedAt time.Time) Handler {
	return func(_ context.Context, _ uint64, _ json.RawMessage) (any, *RPCError) {
		return map[string]any{
			"uptime":         int64(time.Since(startedAt).Seconds()),
			"version":        "0.0.1-spike",
			"activeSessions": sessions.Count(),
			"goVersion":      runtime.Version(),
			"os":             runtime.GOOS,
			"arch":           runtime.GOARCH,
		}, nil
	}
}

func makeDaemonShutdown(sessions *service.Sessions, cancel context.CancelFunc) Handler {
	return func(_ context.Context, _ uint64, _ json.RawMessage) (any, *RPCError) {
		sessions.CloseAll()
		cancel()
		return map[string]bool{"ok": true}, nil
	}
}
