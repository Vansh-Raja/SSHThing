package rpc

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/Vansh-Raja/SSHThing/internal/daemon/service"
)

// RegisterAuth registers auth.* RPC handlers on s.
func RegisterAuth(s *Server, svc *service.AuthService) {
	s.Register("auth.startSignIn", makeAuthStartSignIn(svc))
	s.Register("auth.openBrowser", makeAuthOpenBrowser(svc))
	s.Register("auth.pollSignIn", makeAuthPollSignIn(svc))
	s.Register("auth.signOut", makeAuthSignOut(svc))
	s.Register("auth.session", makeAuthSession(svc))
	s.Register("auth.tokenForRenderer", makeAuthTokenForRenderer(svc))
}

func handleAuthErr(err error) *RPCError {
	if errors.Is(err, service.ErrNotSignedIn) {
		return &RPCError{Code: CodeNotSignedIn, Message: "not signed in", Data: map[string]string{"kind": "not_signed_in"}}
	}
	return &RPCError{Code: CodeInternalError, Message: err.Error()}
}

func makeAuthStartSignIn(svc *service.AuthService) Handler {
	return func(ctx context.Context, _ uint64, _ json.RawMessage) (any, *RPCError) {
		result, err := svc.StartSignIn(ctx)
		if err != nil {
			return nil, handleAuthErr(err)
		}
		return result, nil
	}
}

func makeAuthOpenBrowser(svc *service.AuthService) Handler {
	return func(ctx context.Context, _ uint64, params json.RawMessage) (any, *RPCError) {
		var p struct {
			URL string `json:"url"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		if p.URL == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "url is required"}
		}
		if err := svc.OpenBrowser(p.URL); err != nil {
			return nil, &RPCError{Code: CodeInternalError, Message: err.Error()}
		}
		return map[string]any{"ok": true}, nil
	}
}

func makeAuthPollSignIn(svc *service.AuthService) Handler {
	return func(ctx context.Context, _ uint64, params json.RawMessage) (any, *RPCError) {
		var p struct {
			SessionID  string `json:"sessionId"`
			PollSecret string `json:"pollSecret"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		if p.SessionID == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "sessionId is required"}
		}
		if p.PollSecret == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "pollSecret is required"}
		}
		result, err := svc.PollSignIn(ctx, p.SessionID, p.PollSecret)
		if err != nil {
			return nil, handleAuthErr(err)
		}
		return result, nil
	}
}

func makeAuthSignOut(svc *service.AuthService) Handler {
	return func(ctx context.Context, _ uint64, _ json.RawMessage) (any, *RPCError) {
		if err := svc.SignOut(ctx); err != nil {
			return nil, handleAuthErr(err)
		}
		return map[string]any{"ok": true}, nil
	}
}

func makeAuthSession(svc *service.AuthService) Handler {
	return func(ctx context.Context, _ uint64, _ json.RawMessage) (any, *RPCError) {
		info, err := svc.Session()
		if err != nil {
			return nil, handleAuthErr(err)
		}
		// info is nil when not signed in — return null to the renderer.
		return map[string]any{"session": info}, nil
	}
}

func makeAuthTokenForRenderer(svc *service.AuthService) Handler {
	return func(ctx context.Context, _ uint64, _ json.RawMessage) (any, *RPCError) {
		token, err := svc.TokenForRenderer()
		if err != nil {
			return nil, handleAuthErr(err)
		}
		return map[string]any{"token": token}, nil
	}
}
