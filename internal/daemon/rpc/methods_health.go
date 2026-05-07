package rpc

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/Vansh-Raja/SSHThing/internal/daemon/service"
)

// RegisterHealth registers health.* RPC handlers on s.
func RegisterHealth(s *Server, hs *service.HealthService) {
	s.Register("health.probe", makeHealthProbe(hs))
	s.Register("health.list", makeHealthList(hs))
}

func makeHealthProbe(hs *service.HealthService) Handler {
	return func(ctx context.Context, _ uint64, params json.RawMessage) (any, *RPCError) {
		var p struct {
			HostID string `json:"hostId"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		if p.HostID == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "hostId is required"}
		}
		result, err := hs.Probe(ctx, p.HostID)
		if err != nil {
			if errors.Is(err, service.ErrVaultLocked) {
				return nil, &RPCError{Code: CodeVaultLocked, Message: "vault is locked"}
			}
			return nil, &RPCError{Code: CodeInternalError, Message: "probe failed: " + err.Error()}
		}
		return result, nil
	}
}

func makeHealthList(hs *service.HealthService) Handler {
	return func(_ context.Context, _ uint64, _ json.RawMessage) (any, *RPCError) {
		results, err := hs.List()
		if err != nil {
			if errors.Is(err, service.ErrVaultLocked) {
				return nil, &RPCError{Code: CodeVaultLocked, Message: "vault is locked"}
			}
			return nil, &RPCError{Code: CodeInternalError, Message: "list health failed: " + err.Error()}
		}
		return map[string]any{"results": results}, nil
	}
}
