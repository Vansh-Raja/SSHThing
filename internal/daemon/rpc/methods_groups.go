package rpc

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/Vansh-Raja/SSHThing/internal/daemon/service"
)

// RegisterGroups registers groups.* RPC handlers on s.
func RegisterGroups(s *Server, g *service.Groups) {
	s.Register("groups.list", makeGroupsList(g))
	s.Register("groups.create", makeGroupsCreate(g))
	s.Register("groups.rename", makeGroupsRename(g))
	s.Register("groups.delete", makeGroupsDelete(g))
}

func makeGroupsList(g *service.Groups) Handler {
	return func(_ context.Context, _ uint64, _ json.RawMessage) (any, *RPCError) {
		groups, err := g.List()
		if err != nil {
			if errors.Is(err, service.ErrVaultLocked) {
				return nil, &RPCError{Code: CodeVaultLocked, Message: "vault is locked"}
			}
			return nil, &RPCError{Code: CodeInternalError, Message: "list groups failed: " + err.Error()}
		}
		return map[string]any{"groups": groups}, nil
	}
}

func makeGroupsCreate(g *service.Groups) Handler {
	return func(_ context.Context, _ uint64, params json.RawMessage) (any, *RPCError) {
		var p struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		if p.Name == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "name is required"}
		}
		if err := g.Create(p.Name); err != nil {
			if errors.Is(err, service.ErrVaultLocked) {
				return nil, &RPCError{Code: CodeVaultLocked, Message: "vault is locked"}
			}
			return nil, &RPCError{Code: CodeInternalError, Message: "create group failed: " + err.Error()}
		}
		return map[string]bool{"ok": true}, nil
	}
}

func makeGroupsRename(g *service.Groups) Handler {
	return func(_ context.Context, _ uint64, params json.RawMessage) (any, *RPCError) {
		var p struct {
			Old string `json:"old"`
			New string `json:"new"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		if p.Old == "" || p.New == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "old and new names are required"}
		}
		if err := g.Rename(p.Old, p.New); err != nil {
			if errors.Is(err, service.ErrVaultLocked) {
				return nil, &RPCError{Code: CodeVaultLocked, Message: "vault is locked"}
			}
			return nil, &RPCError{Code: CodeInternalError, Message: "rename group failed: " + err.Error()}
		}
		return map[string]bool{"ok": true}, nil
	}
}

func makeGroupsDelete(g *service.Groups) Handler {
	return func(_ context.Context, _ uint64, params json.RawMessage) (any, *RPCError) {
		var p struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		if p.Name == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "name is required"}
		}
		if err := g.Delete(p.Name); err != nil {
			if errors.Is(err, service.ErrVaultLocked) {
				return nil, &RPCError{Code: CodeVaultLocked, Message: "vault is locked"}
			}
			return nil, &RPCError{Code: CodeInternalError, Message: "delete group failed: " + err.Error()}
		}
		return map[string]bool{"ok": true}, nil
	}
}
