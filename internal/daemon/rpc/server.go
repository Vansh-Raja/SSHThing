package rpc

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"log"
	"net"
	"sync"
	"sync/atomic"
)

// Handler is the function signature for RPC method handlers.
// It receives the raw JSON params and returns a result or an *RPCError.
type Handler func(ctx context.Context, connID uint64, params json.RawMessage) (any, *RPCError)

// Server dispatches newline-delimited JSON-RPC 2.0 requests and sends notifications.
// Single-client assumption is documented; for the spike one Electron window owns one connection.
type Server struct {
	token    string
	handlers map[string]Handler

	mu      sync.Mutex
	conns   map[uint64]*connState
	nextID  atomic.Uint64
}

type connState struct {
	id   uint64
	conn net.Conn
	enc  *json.Encoder
	mu   sync.Mutex // guards writes to enc
}

func (cs *connState) writeJSON(v any) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return cs.enc.Encode(v)
}

// NewServer creates a server that validates requests against token.
func NewServer(token string) *Server {
	return &Server{
		token:    token,
		handlers: make(map[string]Handler),
		conns:    make(map[uint64]*connState),
	}
}

// Register adds a handler for the given method name.
func (s *Server) Register(method string, h Handler) {
	s.handlers[method] = h
}

// Serve accepts connections from l until ctx is cancelled.
func (s *Server) Serve(ctx context.Context, l net.Listener) error {
	go func() {
		<-ctx.Done()
		l.Close()
	}()

	for {
		conn, err := l.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				return err
			}
		}
		id := s.nextID.Add(1)
		cs := &connState{
			id:   id,
			conn: conn,
			enc:  json.NewEncoder(conn),
		}
		s.mu.Lock()
		s.conns[id] = cs
		s.mu.Unlock()

		go s.handleConn(ctx, cs)
	}
}

// Notify sends a notification to all connected clients.
// For the spike, single-client is assumed; this broadcasts to all connections.
func (s *Server) Notify(method string, params any) {
	notif := Notification{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}
	s.mu.Lock()
	conns := make([]*connState, 0, len(s.conns))
	for _, cs := range s.conns {
		conns = append(conns, cs)
	}
	s.mu.Unlock()

	for _, cs := range conns {
		if err := cs.writeJSON(notif); err != nil {
			log.Printf("notify %s to conn %d: %v", method, cs.id, err)
		}
	}
}

// NotifyTo sends a notification only to the connection with the given ID.
func (s *Server) NotifyTo(connID uint64, method string, params any) {
	notif := Notification{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}
	s.mu.Lock()
	cs, ok := s.conns[connID]
	s.mu.Unlock()
	if !ok {
		return
	}
	if err := cs.writeJSON(notif); err != nil {
		log.Printf("notifyTo %s to conn %d: %v", method, connID, err)
	}
}

func (s *Server) handleConn(ctx context.Context, cs *connState) {
	defer func() {
		s.mu.Lock()
		delete(s.conns, cs.id)
		s.mu.Unlock()
		cs.conn.Close()
	}()

	scanner := bufio.NewScanner(cs.conn)
	// Allow up to 4 MiB lines (large session.write payloads).
	buf := make([]byte, 4*1024*1024)
	scanner.Buffer(buf, cap(buf))

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}

		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req Request
		if err := json.Unmarshal(line, &req); err != nil {
			resp := errResp(nil, CodeParseError, "parse error", nil)
			_ = cs.writeJSON(resp)
			continue
		}

		// Auth check — every request must carry the correct token.
		if req.Auth != s.token {
			resp := errResp(req.ID, CodeUnauthorized, "unauthorized", nil)
			_ = cs.writeJSON(resp)
			continue
		}

		h, ok := s.handlers[req.Method]
		if !ok {
			resp := errResp(req.ID, CodeMethodNotFound, "method not found: "+req.Method, nil)
			_ = cs.writeJSON(resp)
			continue
		}

		// Dispatch the handler in a goroutine so a slow request (e.g. a
		// network-heavy teams.list call) can't block other in-flight RPCs
		// on the same connection. Writes are still serialised via
		// cs.writeJSON's mutex so responses interleave safely. Most
		// handlers either don't share state or have their own locking
		// (Vault.mu, db.Store, etc.), so parallel dispatch is safe.
		reqCopy := req
		go func() {
			result, rpcErr := h(ctx, cs.id, reqCopy.Params)
			var resp Response
			if rpcErr != nil {
				resp = errResp(reqCopy.ID, rpcErr.Code, rpcErr.Message, rpcErr.Data)
			} else {
				resp = Response{
					JSONRPC: "2.0",
					ID:      reqCopy.ID,
					Result:  result,
				}
			}
			if err := cs.writeJSON(resp); err != nil {
				if err != io.EOF {
					log.Printf("write response to conn %d: %v", cs.id, err)
				}
			}
		}()
	}
}
