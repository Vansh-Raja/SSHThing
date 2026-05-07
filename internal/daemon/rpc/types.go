// Package rpc implements the newline-delimited JSON-RPC 2.0 transport used by
// the sshthing-daemon IPC socket.
package rpc

import "encoding/json"

// Request is an inbound JSON-RPC 2.0 message with the daemon-specific auth field.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Auth    string          `json:"auth"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response is an outbound JSON-RPC 2.0 response.
type Response struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id"`
	Result  any              `json:"result,omitempty"`
	Error   *RPCError        `json:"error,omitempty"`
}

// Notification is an outbound JSON-RPC 2.0 notification (no id).
// Used for server-push events: session.data, session.exit, vault.locked.
type Notification struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
}

// RPCError is the standard JSON-RPC error object.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func (e *RPCError) Error() string { return e.Message }

// Standard + app-specific error codes.
const (
	CodeParseError    = -32700
	CodeInvalidReq    = -32600
	CodeMethodNotFound = -32601
	CodeInvalidParams = -32602
	CodeInternalError = -32603

	// App-specific
	CodeUnauthorized    = -32001
	CodeInvalidPassword = -32010
	CodeVaultMissing    = -32011
	CodeVaultLocked     = -32021
	CodeHostNotFound    = -32020
	CodeSSHSpawnFailed  = -32022
	CodeNotSignedIn     = -32030
	CodeNotImplemented  = -32031
)

func errResp(id *json.RawMessage, code int, msg string, data any) Response {
	return Response{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &RPCError{Code: code, Message: msg, Data: data},
	}
}
