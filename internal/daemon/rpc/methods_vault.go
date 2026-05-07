package rpc

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/Vansh-Raja/SSHThing/internal/daemon/service"
)

// RegisterVault registers vault.* and keyring.* RPC handlers on s.
func RegisterVault(s *Server, v *service.Vault) {
	s.Register("vault.unlock", makeVaultUnlock(v))
	s.Register("vault.status", makeVaultStatus(v))
	s.Register("vault.create", makeVaultCreate(v))
	s.Register("vault.changePassword", makeVaultChangePassword(v))
	s.Register("vault.lock", makeVaultLock(v))
	s.Register("vault.vacuum", makeVaultVacuum(v))
	s.Register("keyring.healthCheck", handleKeyringHealthCheck)
}

func makeVaultUnlock(v *service.Vault) Handler {
	return func(_ context.Context, _ uint64, params json.RawMessage) (any, *RPCError) {
		var p struct {
			Password string `json:"password"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		if p.Password == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "password is required"}
		}

		result, err := v.Unlock(p.Password)
		if err != nil {
			if errors.Is(err, service.ErrVaultMissing) {
				return nil, &RPCError{Code: CodeVaultMissing, Message: "vault not found"}
			}
			if errors.Is(err, service.ErrInvalidPassword) {
				return nil, &RPCError{Code: CodeInvalidPassword, Message: "invalid password"}
			}
			return nil, &RPCError{Code: CodeInternalError, Message: "unlock failed"}
		}
		return result, nil
	}
}

func makeVaultStatus(v *service.Vault) Handler {
	return func(_ context.Context, _ uint64, _ json.RawMessage) (any, *RPCError) {
		return v.Status(), nil
	}
}

func makeVaultCreate(v *service.Vault) Handler {
	return func(_ context.Context, _ uint64, params json.RawMessage) (any, *RPCError) {
		var p struct {
			Password string `json:"password"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		if p.Password == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "password is required"}
		}

		result, err := v.Create(p.Password)
		if err != nil {
			return nil, &RPCError{Code: CodeInternalError, Message: "create vault failed: " + err.Error()}
		}
		return result, nil
	}
}

func makeVaultChangePassword(v *service.Vault) Handler {
	return func(_ context.Context, _ uint64, params json.RawMessage) (any, *RPCError) {
		var p struct {
			Old string `json:"old"`
			New string `json:"new"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		if p.Old == "" || p.New == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "old and new passwords are required"}
		}
		if err := v.ChangePassword(p.Old, p.New); err != nil {
			if errors.Is(err, service.ErrVaultLocked) {
				return nil, &RPCError{Code: CodeVaultLocked, Message: "vault is locked"}
			}
			if errors.Is(err, service.ErrInvalidPassword) {
				return nil, &RPCError{Code: CodeInvalidPassword, Message: "invalid old password"}
			}
			return nil, &RPCError{Code: CodeInternalError, Message: "change password failed: " + err.Error()}
		}
		return map[string]bool{"ok": true}, nil
	}
}

func makeVaultLock(v *service.Vault) Handler {
	return func(_ context.Context, _ uint64, _ json.RawMessage) (any, *RPCError) {
		v.Lock()
		return map[string]bool{"ok": true}, nil
	}
}

func makeVaultVacuum(v *service.Vault) Handler {
	return func(_ context.Context, _ uint64, _ json.RawMessage) (any, *RPCError) {
		if err := v.Vacuum(); err != nil {
			if errors.Is(err, service.ErrVaultLocked) {
				return nil, &RPCError{Code: CodeVaultLocked, Message: "vault is locked"}
			}
			return nil, &RPCError{Code: CodeInternalError, Message: "vacuum failed: " + err.Error()}
		}
		return map[string]bool{"ok": true}, nil
	}
}

func handleKeyringHealthCheck(_ context.Context, _ uint64, _ json.RawMessage) (any, *RPCError) {
	return service.KeyringHealthCheck(), nil
}
