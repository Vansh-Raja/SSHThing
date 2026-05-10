package rpc

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/Vansh-Raja/SSHThing/internal/daemon/service"
	"github.com/Vansh-Raja/SSHThing/internal/securestore"
)

// RegisterVault registers vault.* and keyring.* RPC handlers on s.
func RegisterVault(s *Server, v *service.Vault) {
	s.Register("vault.unlock", makeVaultUnlock(v))
	s.Register("vault.status", makeVaultStatus(v))
	s.Register("vault.create", makeVaultCreate(v))
	s.Register("vault.changePassword", makeVaultChangePassword(v))
	s.Register("vault.lock", makeVaultLock(v))
	s.Register("vault.vacuum", makeVaultVacuum(v))
	s.Register("vault.biometricStatus", makeVaultBiometricStatus(v))
	s.Register("vault.enableBiometric", makeVaultEnableBiometric(v))
	s.Register("vault.disableBiometric", makeVaultDisableBiometric(v))
	s.Register("vault.unlockWithBiometric", makeVaultUnlockWithBiometric(v))
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

// ── Biometric (Touch ID) RPCs ──────────────────────────────────────────

func makeVaultBiometricStatus(v *service.Vault) Handler {
	return func(_ context.Context, _ uint64, _ json.RawMessage) (any, *RPCError) {
		return v.BiometricStatus(), nil
	}
}

func makeVaultEnableBiometric(v *service.Vault) Handler {
	return func(_ context.Context, _ uint64, params json.RawMessage) (any, *RPCError) {
		var p struct {
			Password string `json:"password"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		res, err := v.EnableBiometric(p.Password)
		if err != nil {
			switch {
			case errors.Is(err, service.ErrInvalidPassword):
				return nil, &RPCError{Code: CodeInvalidPassword, Message: "invalid password"}
			case errors.Is(err, service.ErrVaultMissing):
				return nil, &RPCError{Code: CodeVaultMissing, Message: "vault missing"}
			case errors.Is(err, securestore.ErrBiometricUnavailable):
				return nil, &RPCError{Code: CodeInternalError, Message: "biometric unavailable", Data: map[string]string{"kind": "biometric_unavailable"}}
			default:
				return nil, &RPCError{Code: CodeInternalError, Message: err.Error()}
			}
		}
		return res, nil
	}
}

func makeVaultDisableBiometric(v *service.Vault) Handler {
	return func(_ context.Context, _ uint64, _ json.RawMessage) (any, *RPCError) {
		if err := v.DisableBiometric(); err != nil {
			return nil, &RPCError{Code: CodeInternalError, Message: err.Error()}
		}
		return map[string]bool{"ok": true}, nil
	}
}

func makeVaultUnlockWithBiometric(v *service.Vault) Handler {
	return func(_ context.Context, _ uint64, _ json.RawMessage) (any, *RPCError) {
		res, err := v.UnlockWithBiometric()
		if err != nil {
			switch {
			case errors.Is(err, securestore.ErrBiometricCancelled):
				return nil, &RPCError{Code: CodeInternalError, Message: "user cancelled", Data: map[string]string{"kind": "biometric_cancelled"}}
			case errors.Is(err, securestore.ErrBiometricAuthFailed):
				return nil, &RPCError{Code: CodeInternalError, Message: "auth failed", Data: map[string]string{"kind": "biometric_auth_failed"}}
			case errors.Is(err, securestore.ErrBiometricNotFound):
				return nil, &RPCError{Code: CodeInternalError, Message: "no stored secret", Data: map[string]string{"kind": "biometric_not_found"}}
			case errors.Is(err, securestore.ErrBiometricUnavailable):
				return nil, &RPCError{Code: CodeInternalError, Message: "biometric unavailable", Data: map[string]string{"kind": "biometric_unavailable"}}
			case errors.Is(err, service.ErrInvalidPassword):
				// The cached password no longer opens the DB (e.g. user changed
				// it via TUI). Forget the cache and ask the user.
				_ = v.DisableBiometric()
				return nil, &RPCError{Code: CodeInvalidPassword, Message: "cached password no longer valid", Data: map[string]string{"kind": "biometric_stale"}}
			default:
				return nil, &RPCError{Code: CodeInternalError, Message: err.Error()}
			}
		}
		return res, nil
	}
}
