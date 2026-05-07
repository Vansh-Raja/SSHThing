package rpc

import (
	"context"
	"encoding/json"

	"github.com/Vansh-Raja/SSHThing/internal/config"
	"github.com/Vansh-Raja/SSHThing/internal/daemon/service"
)

// RegisterSettings registers settings.* RPC handlers on s.
func RegisterSettings(s *Server, svc *service.Settings) {
	s.Register("settings.get", makeSettingsGet(svc))
	s.Register("settings.set", makeSettingsSet(svc))
}

func makeSettingsGet(svc *service.Settings) Handler {
	return func(_ context.Context, _ uint64, _ json.RawMessage) (any, *RPCError) {
		cfg := svc.Get()
		return cfg, nil
	}
}

func makeSettingsSet(svc *service.Settings) Handler {
	return func(_ context.Context, _ uint64, params json.RawMessage) (any, *RPCError) {
		var patch config.Config
		if err := json.Unmarshal(params, &patch); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		if err := svc.Set(patch); err != nil {
			return nil, &RPCError{Code: CodeInternalError, Message: "save settings failed: " + err.Error()}
		}
		return map[string]bool{"ok": true}, nil
	}
}
