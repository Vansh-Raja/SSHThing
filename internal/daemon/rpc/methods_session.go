package rpc

import (
	"context"
	"encoding/json"

	"github.com/Vansh-Raja/SSHThing/internal/daemon/service"
)

// RegisterSession registers session.* RPC handlers on s.
func RegisterSession(s *Server, sm *service.Sessions) {
	s.Register("session.open", makeSessionOpen(sm))
	s.Register("session.write", makeSessionWrite(sm))
	s.Register("session.resize", makeSessionResize(sm))
	s.Register("session.close", makeSessionClose(sm))
	s.Register("session.exec", makeSessionExec(sm))
	s.Register("session.list", makeSessionList(sm))
	s.Register("session.titleChanged", makeSessionTitleChanged(sm))
}

func makeSessionOpen(sm *service.Sessions) Handler {
	return func(_ context.Context, _ uint64, params json.RawMessage) (any, *RPCError) {
		var p struct {
			HostID string `json:"hostId"`
			Cols   uint16 `json:"cols"`
			Rows   uint16 `json:"rows"`
			Term   string `json:"term"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		if p.HostID == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "hostId is required"}
		}
		if p.Cols == 0 {
			p.Cols = 80
		}
		if p.Rows == 0 {
			p.Rows = 24
		}

		sessionID, err := sm.Open(service.OpenParams{
			HostID: p.HostID,
			Cols:   p.Cols,
			Rows:   p.Rows,
			Term:   p.Term,
		})
		if err != nil {
			switch err {
			case service.ErrVaultLocked:
				return nil, &RPCError{Code: CodeVaultLocked, Message: "vault is locked", Data: map[string]string{"kind": "locked"}}
			default:
				// Check for host not found
				if isHostNotFound(err) {
					return nil, &RPCError{Code: CodeHostNotFound, Message: "host not found", Data: map[string]string{"kind": "not_found"}}
				}
				return nil, &RPCError{Code: CodeSSHSpawnFailed, Message: "ssh spawn failed: " + err.Error(), Data: map[string]string{"kind": "ssh_spawn_failed"}}
			}
		}
		return map[string]string{"sessionId": sessionID}, nil
	}
}

func makeSessionWrite(sm *service.Sessions) Handler {
	return func(_ context.Context, _ uint64, params json.RawMessage) (any, *RPCError) {
		var p struct {
			SessionID string `json:"sessionId"`
			B64       string `json:"b64"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		if err := sm.Write(p.SessionID, p.B64); err != nil {
			return nil, &RPCError{Code: CodeInternalError, Message: err.Error()}
		}
		return map[string]bool{"ok": true}, nil
	}
}

func makeSessionResize(sm *service.Sessions) Handler {
	return func(_ context.Context, _ uint64, params json.RawMessage) (any, *RPCError) {
		var p struct {
			SessionID string `json:"sessionId"`
			Cols      uint16 `json:"cols"`
			Rows      uint16 `json:"rows"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		if err := sm.Resize(p.SessionID, p.Cols, p.Rows); err != nil {
			return nil, &RPCError{Code: CodeInternalError, Message: err.Error()}
		}
		return map[string]bool{"ok": true}, nil
	}
}

func makeSessionClose(sm *service.Sessions) Handler {
	return func(_ context.Context, _ uint64, params json.RawMessage) (any, *RPCError) {
		var p struct {
			SessionID string `json:"sessionId"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		if err := sm.Close(p.SessionID); err != nil {
			return nil, &RPCError{Code: CodeInternalError, Message: err.Error()}
		}
		return map[string]bool{"ok": true}, nil
	}
}

func makeSessionExec(sm *service.Sessions) Handler {
	return func(ctx context.Context, _ uint64, params json.RawMessage) (any, *RPCError) {
		var p struct {
			HostID    string `json:"hostId"`
			Cmd       string `json:"cmd"`
			TimeoutMs int    `json:"timeoutMs"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		if p.HostID == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "hostId is required"}
		}
		if p.Cmd == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "cmd is required"}
		}
		result, err := sm.Exec(ctx, service.ExecParams{
			HostID:    p.HostID,
			Cmd:       p.Cmd,
			TimeoutMs: p.TimeoutMs,
		})
		if err != nil {
			switch err {
			case service.ErrVaultLocked:
				return nil, &RPCError{Code: CodeVaultLocked, Message: "vault is locked"}
			default:
				if isHostNotFound(err) {
					return nil, &RPCError{Code: CodeHostNotFound, Message: "host not found"}
				}
				return nil, &RPCError{Code: CodeInternalError, Message: "exec failed: " + err.Error()}
			}
		}
		return result, nil
	}
}

func makeSessionList(sm *service.Sessions) Handler {
	return func(_ context.Context, _ uint64, _ json.RawMessage) (any, *RPCError) {
		sessions := sm.List()
		return map[string]any{"sessions": sessions}, nil
	}
}

func makeSessionTitleChanged(sm *service.Sessions) Handler {
	return func(_ context.Context, _ uint64, params json.RawMessage) (any, *RPCError) {
		var p struct {
			SessionID string `json:"sessionId"`
			Title     string `json:"title"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		if p.SessionID == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "sessionId is required"}
		}
		sm.TitleChanged(p.SessionID, p.Title)
		return map[string]bool{"ok": true}, nil
	}
}

func isHostNotFound(err error) bool {
	if err == nil {
		return false
	}
	return contains(err.Error(), "host not found")
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
