package rpc

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/Vansh-Raja/SSHThing/internal/daemon/service"
)

// RegisterSync registers sync.* RPC handlers on s.
func RegisterSync(s *Server, svc *service.SyncService) {
	s.Register("sync.status", makeSyncStatus(svc))
	s.Register("sync.now", makeSyncNow(svc))
	s.Register("sync.configure", makeSyncConfigure(svc))
	s.Register("sync.events", makeSyncEvents(svc))
	s.Register("sync.devices", makeSyncDevices(svc))
	s.Register("sync.forgetDevice", makeSyncForgetDevice(svc))
	s.Register("sync.testGit", makeSyncTestGit(svc))
}

func handleSyncErr(err error) *RPCError {
	if errors.Is(err, service.ErrVaultLocked) {
		return &RPCError{Code: CodeVaultLocked, Message: "vault is locked"}
	}
	if errors.Is(err, service.ErrNotSignedIn) {
		return &RPCError{Code: CodeNotSignedIn, Message: "not signed in"}
	}
	return &RPCError{Code: CodeInternalError, Message: err.Error()}
}

func makeSyncStatus(svc *service.SyncService) Handler {
	return func(ctx context.Context, _ uint64, _ json.RawMessage) (any, *RPCError) {
		result, err := svc.Status(ctx)
		if err != nil {
			return nil, handleSyncErr(err)
		}
		return result, nil
	}
}

func makeSyncNow(svc *service.SyncService) Handler {
	return func(ctx context.Context, _ uint64, _ json.RawMessage) (any, *RPCError) {
		result, err := svc.Now(ctx)
		if err != nil {
			return nil, handleSyncErr(err)
		}
		return map[string]any{
			"success":      result.Success,
			"message":      result.Message,
			"hostsPulled":  result.HostsPulled,
			"hostsPushed":  result.HostsPushed,
			"hostsAdded":   result.HostsAdded,
			"hostsUpdated": result.HostsUpdated,
			"hostsRemoved": result.HostsRemoved,
		}, nil
	}
}

func makeSyncConfigure(svc *service.SyncService) Handler {
	return func(ctx context.Context, _ uint64, params json.RawMessage) (any, *RPCError) {
		var p service.ConfigureParams
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		if err := svc.Configure(p); err != nil {
			return nil, handleSyncErr(err)
		}
		return map[string]any{"ok": true}, nil
	}
}

func makeSyncEvents(svc *service.SyncService) Handler {
	return func(ctx context.Context, _ uint64, _ json.RawMessage) (any, *RPCError) {
		events, err := svc.Events(ctx)
		if err != nil {
			return nil, handleSyncErr(err)
		}
		return map[string]any{"events": events}, nil
	}
}

func makeSyncDevices(svc *service.SyncService) Handler {
	return func(ctx context.Context, _ uint64, _ json.RawMessage) (any, *RPCError) {
		devices := svc.Devices(ctx)
		return map[string]any{"devices": devices}, nil
	}
}

func makeSyncForgetDevice(svc *service.SyncService) Handler {
	return func(ctx context.Context, _ uint64, params json.RawMessage) (any, *RPCError) {
		var p struct {
			DeviceID string `json:"deviceId"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		if err := svc.ForgetDevice(ctx, p.DeviceID); err != nil {
			return nil, handleSyncErr(err)
		}
		return map[string]any{"ok": true}, nil
	}
}

func makeSyncTestGit(svc *service.SyncService) Handler {
	return func(ctx context.Context, _ uint64, params json.RawMessage) (any, *RPCError) {
		var p struct {
			RepoURL    string `json:"repoUrl"`
			SSHKeyPath string `json:"sshKeyPath"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		result := svc.TestGit(ctx, p.RepoURL, p.SSHKeyPath)
		return result, nil
	}
}
