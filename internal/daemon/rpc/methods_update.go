package rpc

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/Vansh-Raja/SSHThing/internal/config"
	"github.com/Vansh-Raja/SSHThing/internal/daemon/service"
)

// RegisterUpdate registers update.* RPC handlers. The actual apply
// pathway lives in the `sshthing update` CLI; the daemon only needs to
// surface the dismiss-banner action so the renderer can sticky-suppress
// the once-a-week nudge per-version.
func RegisterUpdate(s *Server, settings *service.Settings) {
	s.Register("update.dismissBanner", makeUpdateDismiss(settings))
}

type updateDismissParams struct {
	Version string `json:"version"`
}

func makeUpdateDismiss(svc *service.Settings) Handler {
	return func(_ context.Context, _ uint64, raw json.RawMessage) (any, *RPCError) {
		var p updateDismissParams
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		v := strings.TrimSpace(p.Version)
		if v == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "version is required"}
		}
		if err := svc.Mutate(func(c *config.Config) error {
			c.Updates.DismissedVersion = v
			return nil
		}); err != nil {
			return nil, &RPCError{Code: CodeInternalError, Message: "save dismissed version: " + err.Error()}
		}
		return map[string]bool{"ok": true}, nil
	}
}
