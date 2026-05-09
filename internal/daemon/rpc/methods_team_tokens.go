package rpc

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/Vansh-Raja/SSHThing/internal/daemon/service"
)

// RegisterTeamTokens registers teams.tokens.* RPC handlers on s.
func RegisterTeamTokens(s *Server, tts *service.TeamTokensService) {
	s.Register("teams.tokens.list", makeTeamTokensList(tts))
	s.Register("teams.tokens.create", makeTeamTokensCreate(tts))
	s.Register("teams.tokens.revoke", makeTeamTokensRevoke(tts))
	s.Register("teams.tokens.deleteRevoked", makeTeamTokensDeleteRevoked(tts))
}

func makeTeamTokensList(tts *service.TeamTokensService) Handler {
	return func(ctx context.Context, _ uint64, params json.RawMessage) (any, *RPCError) {
		var p struct {
			TeamID string `json:"teamId"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		if p.TeamID == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "teamId is required"}
		}
		summaries, err := tts.List(ctx, p.TeamID)
		if err != nil {
			if errors.Is(err, service.ErrNotSignedIn) {
				return nil, &RPCError{Code: CodeNotSignedIn, Message: "not signed in"}
			}
			return nil, &RPCError{Code: CodeInternalError, Message: "list team tokens failed: " + err.Error()}
		}
		return map[string]any{"tokens": summaries}, nil
	}
}

func makeTeamTokensCreate(tts *service.TeamTokensService) Handler {
	return func(ctx context.Context, _ uint64, params json.RawMessage) (any, *RPCError) {
		var p struct {
			TeamID  string   `json:"teamId"`
			Name    string   `json:"name"`
			HostIDs []string `json:"hostIds"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		if p.TeamID == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "teamId is required"}
		}
		if p.Name == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "name is required"}
		}
		if len(p.HostIDs) == 0 {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "at least one host is required"}
		}
		raw, err := tts.Create(ctx, p.TeamID, p.Name, p.HostIDs)
		if err != nil {
			if errors.Is(err, service.ErrNotSignedIn) {
				return nil, &RPCError{Code: CodeNotSignedIn, Message: "not signed in"}
			}
			return nil, &RPCError{Code: CodeInternalError, Message: "create team token failed: " + err.Error()}
		}
		return map[string]string{"rawToken": raw}, nil
	}
}

func makeTeamTokensRevoke(tts *service.TeamTokensService) Handler {
	return func(ctx context.Context, _ uint64, params json.RawMessage) (any, *RPCError) {
		var p struct {
			TeamID     string `json:"teamId"`
			TokenDocID string `json:"tokenDocId"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		if p.TeamID == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "teamId is required"}
		}
		if p.TokenDocID == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "tokenDocId is required"}
		}
		if err := tts.Revoke(ctx, p.TeamID, p.TokenDocID); err != nil {
			if errors.Is(err, service.ErrNotSignedIn) {
				return nil, &RPCError{Code: CodeNotSignedIn, Message: "not signed in"}
			}
			return nil, &RPCError{Code: CodeInternalError, Message: "revoke team token failed: " + err.Error()}
		}
		return map[string]bool{"ok": true}, nil
	}
}

func makeTeamTokensDeleteRevoked(tts *service.TeamTokensService) Handler {
	return func(ctx context.Context, _ uint64, params json.RawMessage) (any, *RPCError) {
		var p struct {
			TeamID     string `json:"teamId"`
			TokenDocID string `json:"tokenDocId"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		if p.TeamID == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "teamId is required"}
		}
		if p.TokenDocID == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "tokenDocId is required"}
		}
		if err := tts.DeleteRevoked(ctx, p.TeamID, p.TokenDocID); err != nil {
			if errors.Is(err, service.ErrNotSignedIn) {
				return nil, &RPCError{Code: CodeNotSignedIn, Message: "not signed in"}
			}
			return nil, &RPCError{Code: CodeInternalError, Message: "delete revoked team token failed: " + err.Error()}
		}
		return map[string]bool{"ok": true}, nil
	}
}
