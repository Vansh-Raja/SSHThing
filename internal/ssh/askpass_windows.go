//go:build windows

package ssh

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/Microsoft/go-winio"
	"github.com/google/uuid"
)

type windowsAskpassServer struct {
	ln       net.Listener
	endpoint string
	nonce    string
	password string
	once     sync.Once
}

func startAskpassServer(password string) (askpassServer, error) {
	endpoint := `\\.\pipe\sshthing-askpass-` + uuid.NewString()
	ln, err := winio.ListenPipe(endpoint, &winio.PipeConfig{
		SecurityDescriptor: "D:P(A;;GA;;;SY)(A;;GA;;;OW)",
		MessageMode:        true,
		InputBufferSize:    4096,
		OutputBufferSize:   4096,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to listen on askpass pipe: %w", err)
	}

	s := &windowsAskpassServer{
		ln:       ln,
		endpoint: endpoint,
		nonce:    uuid.NewString(),
		password: password,
	}
	go s.serve()
	return s, nil
}

func (s *windowsAskpassServer) Endpoint() string {
	return s.endpoint
}

func (s *windowsAskpassServer) Nonce() string {
	return s.nonce
}

func (s *windowsAskpassServer) Close() error {
	s.once.Do(func() {
		if s.ln != nil {
			_ = s.ln.Close()
		}
	})
	return nil
}

func (s *windowsAskpassServer) serve() {
	defer s.Close()
	go func() {
		time.Sleep(2 * time.Minute)
		_ = s.Close()
	}()

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
	timeout := 5 * time.Second
	conn, err := winio.DialPipe(endpoint, &timeout)
	if err != nil {
		return "", fmt.Errorf("failed to dial askpass pipe: %w", err)
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
