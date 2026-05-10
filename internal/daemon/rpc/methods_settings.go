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

// errInvalidSettingsPatch is a sentinel returned from the settings.set
// closure when the renderer sends a malformed JSON body. CfgStore.Mutate
// sees a non-nil error and aborts before Save / pointer-swap, so we
// never persist a partially-decoded config (json.Unmarshal is documented
// to leave the destination half-populated on failure). The outer handler
// type-asserts to map this to CodeInvalidParams.
type errInvalidSettingsPatch struct{ err error }

func (e *errInvalidSettingsPatch) Error() string { return "invalid params: " + e.err.Error() }

func makeSettingsSet(svc *service.Settings) Handler {
	return func(_ context.Context, _ uint64, params json.RawMessage) (any, *RPCError) {
		err := svc.Mutate(func(cfg *config.Config) error {
			if uerr := json.Unmarshal(params, cfg); uerr != nil {
				return &errInvalidSettingsPatch{err: uerr}
			}
			return nil
		})
		if err != nil {
			if invalid, ok := err.(*errInvalidSettingsPatch); ok {
				return nil, &RPCError{Code: CodeInvalidParams, Message: invalid.Error()}
			}
			return nil, &RPCError{Code: CodeInternalError, Message: "save settings failed: " + err.Error()}
		}
		return map[string]bool{"ok": true}, nil
	}
}
