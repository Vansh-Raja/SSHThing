package rpc

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
)

// GenerateToken creates a cryptographically random 32-byte hex token.
func GenerateToken() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return hex.EncodeToString(b[:]), nil
}

// WriteToken writes the token to path with mode 0600.
func WriteToken(path, token string) error {
	return os.WriteFile(path, []byte(token), 0600)
}

// ReadToken reads a token from path and trims whitespace.
func ReadToken(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	// Trim any trailing newline written by external tools.
	for len(b) > 0 && (b[len(b)-1] == '\n' || b[len(b)-1] == '\r') {
		b = b[:len(b)-1]
	}
	return string(b), nil
}
