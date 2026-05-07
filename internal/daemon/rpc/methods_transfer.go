package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"

	"github.com/Vansh-Raja/SSHThing/internal/daemon/service"
)

// transferSeq is a process-wide counter used to generate unique transfer IDs.
var transferSeq atomic.Uint64

// RegisterTransfer registers transfer.* RPC handlers on s.
func RegisterTransfer(s *Server, ts *service.TransferService) {
	s.Register("transfer.upload", makeTransferUpload(ts))
	s.Register("transfer.download", makeTransferDownload(ts))
	s.Register("transfer.cancel", makeTransferCancel(ts))
}

func makeTransferUpload(ts *service.TransferService) Handler {
	return func(_ context.Context, connID uint64, params json.RawMessage) (any, *RPCError) {
		var p service.TransferParams
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		if p.HostID == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "hostId is required"}
		}
		if p.Local == "" || p.Remote == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "local and remote paths are required"}
		}
		// Validate that the vault is unlocked synchronously before returning the
		// transferId — avoids the renderer waiting for a failed goroutine.
		if ts.Vault.Store() == nil {
			return nil, &RPCError{Code: CodeVaultLocked, Message: "vault is locked"}
		}
		transferID := fmt.Sprintf("xfr_%d_%d", connID, transferSeq.Add(1))
		go func() {
			// Errors surface as transfer.progress{status:"failed"} notifications.
			_ = ts.Upload(p, transferID)
		}()
		return map[string]string{"transferId": transferID}, nil
	}
}

func makeTransferCancel(ts *service.TransferService) Handler {
	return func(_ context.Context, _ uint64, params json.RawMessage) (any, *RPCError) {
		var p struct {
			TransferID string `json:"transferId"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		if p.TransferID == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "transferId is required"}
		}
		if err := ts.Cancel(p.TransferID); err != nil {
			return nil, &RPCError{Code: CodeInternalError, Message: "cancel transfer failed: " + err.Error()}
		}
		return map[string]bool{"ok": true}, nil
	}
}

func makeTransferDownload(ts *service.TransferService) Handler {
	return func(_ context.Context, connID uint64, params json.RawMessage) (any, *RPCError) {
		var p service.TransferParams
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		if p.HostID == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "hostId is required"}
		}
		if p.Local == "" || p.Remote == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "local and remote paths are required"}
		}
		// Validate that the vault is unlocked synchronously before returning the
		// transferId — avoids the renderer waiting for a failed goroutine.
		if ts.Vault.Store() == nil {
			return nil, &RPCError{Code: CodeVaultLocked, Message: "vault is locked"}
		}
		transferID := fmt.Sprintf("xfr_%d_%d", connID, transferSeq.Add(1))
		go func() {
			// Errors surface as transfer.progress{status:"failed"} notifications.
			_ = ts.Download(p, transferID)
		}()
		return map[string]string{"transferId": transferID}, nil
	}
}
