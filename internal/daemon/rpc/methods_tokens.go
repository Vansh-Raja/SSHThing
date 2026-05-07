package rpc

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/Vansh-Raja/SSHThing/internal/authtoken"
	"github.com/Vansh-Raja/SSHThing/internal/daemon/service"
)

// RegisterTokens registers tokens.* RPC handlers on s.
func RegisterTokens(s *Server, ts *service.TokensService) {
	s.Register("tokens.list", makeTokensList(ts))
	s.Register("tokens.create", makeTokensCreate(ts))
	s.Register("tokens.revoke", makeTokensRevoke(ts))
	s.Register("tokens.deleteRevoked", makeTokensDeleteRevoked(ts))
}

func makeTokensList(ts *service.TokensService) Handler {
	return func(_ context.Context, _ uint64, _ json.RawMessage) (any, *RPCError) {
		summaries, err := ts.List()
		if err != nil {
			return nil, &RPCError{Code: CodeInternalError, Message: "list tokens failed: " + err.Error()}
		}
		return map[string]any{"tokens": summaries}, nil
	}
}

func makeTokensCreate(ts *service.TokensService) Handler {
	return func(_ context.Context, _ uint64, params json.RawMessage) (any, *RPCError) {
		var p struct {
			Name   string               `json:"name"`
			Grants []authtoken.HostGrant `json:"grants"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		if p.Name == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "name is required"}
		}
		if len(p.Grants) == 0 {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "at least one host grant is required"}
		}
		raw, err := ts.Create(service.CreateTokenParams{
			Name:   p.Name,
			Grants: p.Grants,
		})
		if err != nil {
			if errors.Is(err, service.ErrVaultLocked) {
				return nil, &RPCError{Code: CodeVaultLocked, Message: "vault is locked"}
			}
			return nil, &RPCError{Code: CodeInternalError, Message: "create token failed: " + err.Error()}
		}
		return map[string]string{"rawToken": raw}, nil
	}
}

func makeTokensRevoke(ts *service.TokensService) Handler {
	return func(_ context.Context, _ uint64, params json.RawMessage) (any, *RPCError) {
		var p struct {
			TokenID string `json:"tokenId"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		if p.TokenID == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "tokenId is required"}
		}
		if err := ts.Revoke(p.TokenID); err != nil {
			return nil, &RPCError{Code: CodeInternalError, Message: "revoke token failed: " + err.Error()}
		}
		return map[string]bool{"ok": true}, nil
	}
}

func makeTokensDeleteRevoked(ts *service.TokensService) Handler {
	return func(_ context.Context, _ uint64, params json.RawMessage) (any, *RPCError) {
		var p struct {
			TokenID string `json:"tokenId"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		if p.TokenID == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "tokenId is required"}
		}
		if err := ts.DeleteRevoked(p.TokenID); err != nil {
			return nil, &RPCError{Code: CodeInternalError, Message: "delete revoked token failed: " + err.Error()}
		}
		return map[string]bool{"ok": true}, nil
	}
}
