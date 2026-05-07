package service

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty"
	"github.com/Vansh-Raja/SSHThing/internal/config"
	"github.com/Vansh-Raja/SSHThing/internal/ssh"
)

// NotifyFunc is called by the session reader goroutine to push PTY bytes and
// exit codes back to connected clients.
type NotifyFunc func(method string, params any)

// Session represents a single active SSH/PTY session.
type Session struct {
	ID        string
	HostID    string
	OpenedAt  time.Time
	ptyMaster *os.File
	cmd       *exec.Cmd
	tempKey   *ssh.TempKeyFile
	cancel    context.CancelFunc
}

// Sessions manages active SSH sessions.
// Single-client assumption: one Electron window, one connection.
type Sessions struct {
	Vault  *Vault
	Notify NotifyFunc
	// CfgStore carries the SSH defaults (password backend, host-key policy, keep-alive,
	// terminal type) used when building ssh.Connection. Accessed via atomic.Pointer so
	// settings.set hot-reload is safe without locks.
	CfgStore *CfgStore

	mu      sync.Mutex
	byID    map[string]*Session
	counter uint64
}

// resolveTerm picks the right TERM string given config + per-call override.
// Mirrors internal/app/backend.go connectToHost lines 451-457.
func (sm *Sessions) resolveTerm(override string) string {
	if strings.TrimSpace(override) != "" {
		return override
	}
	if sm.CfgStore == nil {
		return ""
	}
	cfg := sm.CfgStore.Get()
	switch cfg.SSH.TermMode {
	case config.TermXterm:
		return "xterm-256color"
	case config.TermCustom:
		return strings.TrimSpace(cfg.SSH.TermCustom)
	}
	return ""
}

// OpenParams are the session.open RPC parameters (after JSON decode).
type OpenParams struct {
	HostID string
	Cols   uint16
	Rows   uint16
	Term   string
}

// Open establishes an SSH session attached to a PTY.
// It replicates the TUI's connectToHost flow (internal/app/backend.go connectToHost).
func (sm *Sessions) Open(params OpenParams) (string, error) {
	store := sm.Vault.Store()
	if store == nil {
		return "", ErrVaultLocked
	}

	intID, err := strconv.Atoi(params.HostID)
	if err != nil {
		return "", fmt.Errorf("invalid host id %q: %w", params.HostID, err)
	}

	// Load host metadata — ref: internal/db/db.go GetHostByID.
	model, err := store.GetHostByID(intID)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("host not found: %s", params.HostID)
	}
	if err != nil {
		return "", fmt.Errorf("get host: %w", err)
	}

	// Decrypt the secret — ref: internal/db/db.go GetHostSecret.
	secret, err := store.GetHostSecret(intID)
	if err != nil {
		return "", fmt.Errorf("get host secret: %w", err)
	}

	// Build ssh.Connection — mirrors internal/app/backend.go connectToHost.
	// Settings are taken from the loaded Config; sensible defaults if CfgStore is nil.
	hostKeyPolicy := string(config.HostKeyAcceptNew)
	keepAlive := 60
	passwordBackend := string(config.PasswordBackendSSHPassFirst)
	if sm.CfgStore != nil {
		cfg := sm.CfgStore.Get()
		hostKeyPolicy = string(cfg.SSH.HostKeyPolicy)
		keepAlive = cfg.SSH.KeepAliveSeconds
		passwordBackend = string(cfg.SSH.PasswordBackendUnix)
	}

	conn := ssh.Connection{
		Hostname:            model.Hostname,
		Username:            model.Username,
		Port:                model.Port,
		Term:                sm.resolveTerm(params.Term),
		HostKeyPolicy:       hostKeyPolicy,
		KeepAliveSeconds:    keepAlive,
		PasswordBackendUnix: passwordBackend,
	}
	if model.KeyType == "password" {
		conn.Password = secret
	} else if secret != "" {
		conn.PrivateKey = secret
	}

	// Build the ssh exec.Cmd — ref: internal/ssh/connect.go Connect().
	// Connect sets Stdin/Stdout/Stderr to os.Stdin/os.Stdout/os.Stderr, but we
	// will replace them via pty.StartWithSize before the cmd is run.
	cmd, tempKey, err := ssh.Connect(conn)
	if err != nil {
		return "", fmt.Errorf("ssh connect: %w", err)
	}

	// Detach from the inherited stdio so the PTY owns them.
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil

	// Start the command attached to a PTY.
	ptyMaster, err := pty.StartWithSize(cmd, &pty.Winsize{
		Rows: params.Rows,
		Cols: params.Cols,
	})
	if err != nil {
		_ = tempKey.Cleanup()
		return "", fmt.Errorf("pty start: %w", err)
	}

	sm.mu.Lock()
	sm.counter++
	id := fmt.Sprintf("s_%d", sm.counter)
	if sm.byID == nil {
		sm.byID = make(map[string]*Session)
	}
	ctx, cancel := context.WithCancel(context.Background())
	sess := &Session{
		ID:        id,
		HostID:    params.HostID,
		OpenedAt:  time.Now(),
		ptyMaster: ptyMaster,
		cmd:       cmd,
		tempKey:   tempKey,
		cancel:    cancel,
	}
	sm.byID[id] = sess
	sm.mu.Unlock()

	// Update last-connected timestamp — best effort.
	_ = store.UpdateLastConnected(intID)

	// Start reader goroutine: pumps PTY bytes to the client via session.data notifications.
	go sm.pumpPTY(ctx, sess)

	return id, nil
}

// Write decodes b64 and writes to the session's PTY master.
func (sm *Sessions) Write(sessionID, b64 string) error {
	sess, err := sm.get(sessionID)
	if err != nil {
		return err
	}
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return fmt.Errorf("decode b64: %w", err)
	}
	_, err = sess.ptyMaster.Write(data)
	return err
}

// Resize sets the PTY window size.
func (sm *Sessions) Resize(sessionID string, cols, rows uint16) error {
	sess, err := sm.get(sessionID)
	if err != nil {
		return err
	}
	return pty.Setsize(sess.ptyMaster, &pty.Winsize{
		Rows: rows,
		Cols: cols,
	})
}

// Close kills the session and performs cleanup in the correct order:
// 1. close PTY master (reader goroutine sees EOF and exits)
// 2. kill cmd
// 3. wait for cmd to exit (harvest exit code)
// 4. cleanup temp key
// 5. emit session.exit notification
func (sm *Sessions) Close(sessionID string) error {
	sm.mu.Lock()
	sess, ok := sm.byID[sessionID]
	if ok {
		delete(sm.byID, sessionID)
	}
	sm.mu.Unlock()

	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	return sm.closeSession(sess, "")
}

func (sm *Sessions) closeSession(sess *Session, reason string) error {
	// 1. Cancel reader goroutine context.
	sess.cancel()

	// 2. Close PTY master — reader goroutine gets EOF.
	_ = sess.ptyMaster.Close()

	// 3. Kill the ssh process.
	if sess.cmd.Process != nil {
		_ = sess.cmd.Process.Kill()
	}

	// 4. Wait for the process to exit and capture exit code.
	exitCode := 0
	if err := sess.cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}

	// 5. Cleanup temp key — must happen after Wait to avoid removing the key
	//    while ssh is still using it.
	if sess.tempKey != nil {
		_ = sess.tempKey.Cleanup()
	}

	// 6. Emit session.exit notification.
	if sm.Notify != nil {
		sm.Notify("session.exit", map[string]any{
			"sessionId": sess.ID,
			"exitCode":  exitCode,
		})
	}

	if reason != "" {
		log.Printf("session %s closed: %s (exit %d)", sess.ID, reason, exitCode)
	}
	return nil
}

func (sm *Sessions) get(sessionID string) (*Session, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sess, ok := sm.byID[sessionID]
	if !ok {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}
	return sess, nil
}

// pumpPTY reads from the PTY master and emits session.data notifications.
// It runs until the PTY is closed or ctx is cancelled.
// Cleanup is always performed regardless of exit path.
func (sm *Sessions) pumpPTY(ctx context.Context, sess *Session) {
	buf := make([]byte, 4096)
	for {
		select {
		case <-ctx.Done():
			// Session was closed externally; close() will handle cleanup.
			return
		default:
		}

		n, err := sess.ptyMaster.Read(buf)
		if n > 0 && sm.Notify != nil {
			chunk := make([]byte, n)
			copy(chunk, buf[:n])
			sm.Notify("session.data", map[string]any{
				"sessionId": sess.ID,
				"stream":    "stdout",
				"b64":       base64.StdEncoding.EncodeToString(chunk),
			})
		}
		if err != nil {
			if err == io.EOF {
				// PTY closed normally.
			} else {
				log.Printf("session %s pty read error: %v", sess.ID, err)
			}
			// Remove from map if not already removed by Close().
			sm.mu.Lock()
			_, stillPresent := sm.byID[sess.ID]
			if stillPresent {
				delete(sm.byID, sess.ID)
			}
			sm.mu.Unlock()

			if stillPresent {
				// We own the cleanup since Close() hasn't run.
				_ = sm.closeSession(sess, "pty eof")
			}
			return
		}
	}
}

// Count returns the number of currently active sessions.
func (sm *Sessions) Count() int {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return len(sm.byID)
}

// ExecParams holds the parameters for session.exec.
type ExecParams struct {
	HostID    string
	Cmd       string
	TimeoutMs int
}

// ExecResult is returned by session.exec.
type ExecResult struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exitCode"`
	DurationMs int64  `json:"durationMs"`
}

// Exec runs a non-interactive command on the remote host and captures output.
// Mirrors: internal/ssh/connect.go ConnectExecCaptured.
func (sm *Sessions) Exec(ctx context.Context, params ExecParams) (*ExecResult, error) {
	store := sm.Vault.Store()
	if store == nil {
		return nil, ErrVaultLocked
	}
	intID, err := strconv.Atoi(params.HostID)
	if err != nil {
		return nil, fmt.Errorf("invalid host id %q: %w", params.HostID, err)
	}
	model, err := store.GetHostByID(intID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("host not found: %s", params.HostID)
	}
	if err != nil {
		return nil, fmt.Errorf("get host: %w", err)
	}
	secret, err := store.GetHostSecret(intID)
	if err != nil {
		return nil, fmt.Errorf("get secret: %w", err)
	}

	hostKeyPolicy := string(config.HostKeyAcceptNew)
	keepAlive := 60
	passwordBackend := string(config.PasswordBackendSSHPassFirst)
	if sm.CfgStore != nil {
		cfg := sm.CfgStore.Get()
		hostKeyPolicy = string(cfg.SSH.HostKeyPolicy)
		keepAlive = cfg.SSH.KeepAliveSeconds
		passwordBackend = string(cfg.SSH.PasswordBackendUnix)
	}

	conn := ssh.Connection{
		Hostname:            model.Hostname,
		Username:            model.Username,
		Port:                model.Port,
		HostKeyPolicy:       hostKeyPolicy,
		KeepAliveSeconds:    keepAlive,
		PasswordBackendUnix: passwordBackend,
	}
	if model.KeyType == "password" {
		conn.Password = secret
	} else if secret != "" {
		conn.PrivateKey = secret
	}

	timeout := time.Duration(params.TimeoutMs) * time.Millisecond
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	res, err := ssh.ConnectExecCaptured(ctx, conn, params.Cmd, ssh.ExecOptions{
		Timeout:           timeout,
		ConnectTimeout:    5 * time.Second,
		AllowPasswordAuth: model.KeyType == "password",
	})
	if err != nil {
		return nil, fmt.Errorf("exec: %w", err)
	}
	return &ExecResult{
		Stdout:     res.Stdout,
		Stderr:     res.Stderr,
		ExitCode:   res.ExitCode,
		DurationMs: res.Duration.Milliseconds(),
	}, nil
}

// SessionSummary is the wire-safe description of an active session.
type SessionSummary struct {
	ID       string    `json:"id"`
	HostID   string    `json:"hostId"`
	OpenedAt time.Time `json:"openedAt"`
}

// List returns summaries of all currently active sessions.
func (sm *Sessions) List() []SessionSummary {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	out := make([]SessionSummary, 0, len(sm.byID))
	for _, s := range sm.byID {
		out = append(out, SessionSummary{
			ID:       s.ID,
			HostID:   s.HostID,
			OpenedAt: s.OpenedAt,
		})
	}
	return out
}

// TitleChanged broadcasts a session.titleChanged notification (from renderer OSC parsing).
func (sm *Sessions) TitleChanged(sessionID, title string) {
	if sm.Notify != nil {
		sm.Notify("session.titleChanged", map[string]any{
			"sessionId": sessionID,
			"title":     title,
		})
	}
}

// CloseAll closes all active sessions — called on daemon shutdown.
func (sm *Sessions) CloseAll() {
	sm.mu.Lock()
	sessions := make([]*Session, 0, len(sm.byID))
	for _, sess := range sm.byID {
		sessions = append(sessions, sess)
	}
	sm.byID = make(map[string]*Session)
	sm.mu.Unlock()

	for _, sess := range sessions {
		_ = sm.closeSession(sess, "daemon shutdown")
	}
}
