//go:build !windows

package ssh

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type unixAskpassServer struct {
	ln       net.Listener
	endpoint string
	nonce    string
	password string
	once     sync.Once
}

func startAskpassServer(password string) (askpassServer, error) {
	tmpDir := filepath.Join(os.TempDir(), "ssh-manager", "askpass")
	if err := os.MkdirAll(tmpDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create askpass temp dir: %w", err)
	}

	endpoint := filepath.Join(tmpDir, fmt.Sprintf("sock_%s", uuid.NewString()))
	_ = os.Remove(endpoint)
	ln, err := net.Listen("unix", endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on askpass socket: %w", err)
	}
	if err := os.Chmod(endpoint, 0600); err != nil {
		_ = ln.Close()
		_ = os.Remove(endpoint)
		return nil, fmt.Errorf("failed to chmod askpass socket: %w", err)
	}

	s := &unixAskpassServer{
		ln:       ln,
		endpoint: endpoint,
		nonce:    uuid.NewString(),
		password: password,
	}
	go s.serve()
	return s, nil
}

func (s *unixAskpassServer) Endpoint() string {
	return s.endpoint
}

func (s *unixAskpassServer) Nonce() string {
	return s.nonce
}

func (s *unixAskpassServer) Close() error {
	s.once.Do(func() {
		if s.ln != nil {
			_ = s.ln.Close()
		}
		if s.endpoint != "" {
			_ = os.Remove(s.endpoint)
		}
	})
	return nil
}

func (s *unixAskpassServer) serve() {
	defer s.Close()
	_ = s.ln.(*net.UnixListener).SetDeadline(time.Now().Add(2 * time.Minute))

	conn, err := s.ln.Accept()
	if err != nil {
		return
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)
	token, err := reader.ReadString('\n')
	if err != nil {
		return
	}
	if strings.TrimSpace(token) != s.nonce {
		return
	}
	_, _ = conn.Write([]byte(s.password + "\n"))
}

func requestAskpassSecret(endpoint, nonce string) (string, error) {
	conn, err := net.DialTimeout("unix", endpoint, 5*time.Second)
	if err != nil {
		return "", fmt.Errorf("failed to dial askpass socket: %w", err)
	}
	defer conn.Close()

	if _, err := conn.Write([]byte(nonce + "\n")); err != nil {
		return "", fmt.Errorf("failed to write askpass nonce: %w", err)
	}

	line, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read askpass secret: %w", err)
	}
	return strings.TrimRight(line, "\r\n"), nil
}
