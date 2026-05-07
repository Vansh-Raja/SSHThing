package rpc

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/Vansh-Raja/SSHThing/internal/daemon/service"
)

// RegisterMount registers mount.* RPC handlers on s.
func RegisterMount(s *Server, ms *service.MountService) {
	s.Register("mount.start", makeMountStart(ms))
	s.Register("mount.stop", makeMountStop(ms))
	s.Register("mount.list", makeMountList(ms))
	s.Register("mount.checkPrereqs", makeMountCheckPrereqs(ms))
}

func makeMountStart(ms *service.MountService) Handler {
	return func(ctx context.Context, _ uint64, params json.RawMessage) (any, *RPCError) {
		var p struct {
			HostID     string `json:"hostId"`
			RemotePath string `json:"remotePath"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		if p.HostID == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "hostId is required"}
		}
		summary, err := ms.Start(ctx, p.HostID, p.RemotePath)
		if err != nil {
			if errors.Is(err, service.ErrVaultLocked) {
				return nil, &RPCError{Code: CodeVaultLocked, Message: "vault is locked"}
			}
			return nil, &RPCError{Code: CodeInternalError, Message: "mount failed: " + err.Error()}
		}
		return summary, nil
	}
}

func makeMountStop(ms *service.MountService) Handler {
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
		if err := ms.Stop(ctx, p.HostID); err != nil {
			if errors.Is(err, service.ErrVaultLocked) {
				return nil, &RPCError{Code: CodeVaultLocked, Message: "vault is locked"}
			}
			return nil, &RPCError{Code: CodeInternalError, Message: "unmount failed: " + err.Error()}
		}
		return map[string]bool{"ok": true}, nil
	}
}

func makeMountList(ms *service.MountService) Handler {
	return func(_ context.Context, _ uint64, _ json.RawMessage) (any, *RPCError) {
		mounts := ms.List()
		return map[string]any{"mounts": mounts}, nil
	}
}

func makeMountCheckPrereqs(ms *service.MountService) Handler {
	return func(_ context.Context, _ uint64, _ json.RawMessage) (any, *RPCError) {
		result := ms.CheckPrereqs()
		return result, nil
	}
}
