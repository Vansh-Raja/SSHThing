package rpc

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/Vansh-Raja/SSHThing/internal/daemon/service"
)

// RegisterHosts registers hosts.* RPC handlers on s.
func RegisterHosts(s *Server, h *service.Hosts) {
	s.Register("hosts.list", makeHostsList(h))
	s.Register("hosts.get", makeHostsGet(h))
	s.Register("hosts.create", makeHostsCreate(h))
	s.Register("hosts.update", makeHostsUpdate(h))
	s.Register("hosts.updateWithKey", makeHostsUpdateWithKey(h))
	s.Register("hosts.delete", makeHostsDelete(h))
	s.Register("hosts.revealCredential", makeHostsRevealCredential(h))
	s.Register("hosts.import", makeHostsImport(h))
	s.Register("hosts.generateKey", makeHostsGenerateKey(h))
}

func makeHostsList(h *service.Hosts) Handler {
	return func(_ context.Context, _ uint64, params json.RawMessage) (any, *RPCError) {
		var p struct {
			Query string `json:"query"`
		}
		if params != nil {
			_ = json.Unmarshal(params, &p)
		}

		hosts, err := h.List(p.Query)
		if err != nil {
			if errors.Is(err, service.ErrVaultLocked) {
				return nil, &RPCError{Code: CodeVaultLocked, Message: "vault is locked", Data: map[string]string{"kind": "locked"}}
			}
			return nil, &RPCError{Code: CodeInternalError, Message: "list hosts failed"}
		}
		return map[string]any{"hosts": hosts}, nil
	}
}

func makeHostsGet(h *service.Hosts) Handler {
	return func(_ context.Context, _ uint64, params json.RawMessage) (any, *RPCError) {
		var p struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		if p.ID == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "id is required"}
		}

		summary, err := h.Get(p.ID)
		if err != nil {
			if errors.Is(err, service.ErrVaultLocked) {
				return nil, &RPCError{Code: CodeVaultLocked, Message: "vault is locked", Data: map[string]string{"kind": "locked"}}
			}
			return nil, &RPCError{Code: CodeInternalError, Message: "get host failed"}
		}
		if summary == nil {
			return nil, &RPCError{Code: CodeHostNotFound, Message: "host not found", Data: map[string]string{"kind": "not_found"}}
		}
		return summary, nil
	}
}

func makeHostsCreate(h *service.Hosts) Handler {
	return func(_ context.Context, _ uint64, params json.RawMessage) (any, *RPCError) {
		var p service.CreateParams
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		if p.Hostname == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "hostname is required"}
		}

		summary, err := h.Create(p)
		if err != nil {
			if errors.Is(err, service.ErrVaultLocked) {
				return nil, &RPCError{Code: CodeVaultLocked, Message: "vault is locked"}
			}
			return nil, &RPCError{Code: CodeInternalError, Message: "create host failed: " + err.Error()}
		}
		return summary, nil
	}
}

func makeHostsUpdate(h *service.Hosts) Handler {
	return func(_ context.Context, _ uint64, params json.RawMessage) (any, *RPCError) {
		var p service.UpdateParams
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		if p.ID == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "id is required"}
		}
		if err := h.Update(p); err != nil {
			if errors.Is(err, service.ErrVaultLocked) {
				return nil, &RPCError{Code: CodeVaultLocked, Message: "vault is locked"}
			}
			return nil, &RPCError{Code: CodeInternalError, Message: "update host failed: " + err.Error()}
		}
		return map[string]bool{"ok": true}, nil
	}
}

func makeHostsUpdateWithKey(h *service.Hosts) Handler {
	return func(_ context.Context, _ uint64, params json.RawMessage) (any, *RPCError) {
		var p struct {
			service.UpdateParams
			PlainKey string `json:"plainKey"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		if p.ID == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "id is required"}
		}
		if err := h.UpdateWithKey(p.UpdateParams, p.PlainKey); err != nil {
			if errors.Is(err, service.ErrVaultLocked) {
				return nil, &RPCError{Code: CodeVaultLocked, Message: "vault is locked"}
			}
			return nil, &RPCError{Code: CodeInternalError, Message: "update host with key failed: " + err.Error()}
		}
		return map[string]bool{"ok": true}, nil
	}
}

func makeHostsDelete(h *service.Hosts) Handler {
	return func(_ context.Context, _ uint64, params json.RawMessage) (any, *RPCError) {
		var p struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		if p.ID == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "id is required"}
		}
		if err := h.Delete(p.ID); err != nil {
			if errors.Is(err, service.ErrVaultLocked) {
				return nil, &RPCError{Code: CodeVaultLocked, Message: "vault is locked"}
			}
			return nil, &RPCError{Code: CodeInternalError, Message: "delete host failed: " + err.Error()}
		}
		return map[string]bool{"ok": true}, nil
	}
}

func makeHostsRevealCredential(h *service.Hosts) Handler {
	return func(_ context.Context, connID uint64, params json.RawMessage) (any, *RPCError) {
		var p struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		if p.ID == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "id is required"}
		}
		// RevealCredential returns the secret + auth mode so the renderer
		// can label the reveal as "Private key" or "Password" without a
		// second roundtrip. The renderer (RevealCredentialModal) reads
		// `result.credential` and `result.authMode`.
		revealed, err := h.RevealCredential(p.ID, connID)
		if err != nil {
			if errors.Is(err, service.ErrVaultLocked) {
				return nil, &RPCError{Code: CodeVaultLocked, Message: "vault is locked"}
			}
			return nil, &RPCError{Code: CodeInternalError, Message: "reveal credential failed: " + err.Error()}
		}
		return revealed, nil
	}
}

func makeHostsImport(h *service.Hosts) Handler {
	return func(_ context.Context, _ uint64, params json.RawMessage) (any, *RPCError) {
		var p service.ImportParams
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		if p.KeyBlob == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "keyBlob is required"}
		}

		summary, err := h.Import(p)
		if err != nil {
			if errors.Is(err, service.ErrVaultLocked) {
				return nil, &RPCError{Code: CodeVaultLocked, Message: "vault is locked"}
			}
			return nil, &RPCError{Code: CodeInternalError, Message: "import failed: " + err.Error()}
		}
		return summary, nil
	}
}

func makeHostsGenerateKey(h *service.Hosts) Handler {
	return func(_ context.Context, _ uint64, params json.RawMessage) (any, *RPCError) {
		var p struct {
			Type    string `json:"type"`
			Comment string `json:"comment"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		if p.Type == "" {
			p.Type = "ed25519"
		}

		result, err := h.GenerateKey(p.Type, p.Comment)
		if err != nil {
			return nil, &RPCError{Code: CodeInternalError, Message: "generate key failed: " + err.Error()}
		}
		return result, nil
	}
}
