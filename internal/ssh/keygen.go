package ssh

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	xssh "golang.org/x/crypto/ssh"
)

// KeyType represents the type of SSH key to generate
type KeyType string

const (
	KeyTypeEd25519 KeyType = "ed25519"
	KeyTypeRSA     KeyType = "rsa"
	KeyTypeECDSA   KeyType = "ecdsa"
)

// GenerateKey generates a new SSH key pair using ssh-keygen.
// Returns the private key content and the public key content.
func GenerateKey(keyType KeyType, comment string) (privateKey, publicKey string, err error) {
	// Create a temporary directory for key generation
	tmpDir, err := os.MkdirTemp("", "ssh-manager-keygen-")
	if err != nil {
		return "", "", fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	keyPath := filepath.Join(tmpDir, "id_key")

	// Build ssh-keygen command
	args := []string{
		"-t", string(keyType),
		"-f", keyPath,
		"-N", "", // No passphrase (we encrypt at app level)
		"-q", // Quiet mode
	}

	// Add key-specific options
	switch keyType {
	case KeyTypeRSA:
		args = append(args, "-b", "4096")
	case KeyTypeECDSA:
		args = append(args, "-b", "256") // P-256 curve
	}

	// Add comment if provided
	if comment != "" {
		args = append(args, "-C", comment)
	}

	// Run ssh-keygen
	cmd := exec.Command("ssh-keygen", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", "", fmt.Errorf("ssh-keygen failed: %w, output: %s", err, string(output))
	}

	// Read the generated keys
	privateKeyBytes, err := os.ReadFile(keyPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to read private key: %w", err)
	}

	publicKeyBytes, err := os.ReadFile(keyPath + ".pub")
	if err != nil {
		return "", "", fmt.Errorf("failed to read public key: %w", err)
	}

	return string(privateKeyBytes), strings.TrimSpace(string(publicKeyBytes)), nil
}

// ValidatePrivateKey parses a PEM/OpenSSH private key using x/crypto/ssh.
func ValidatePrivateKey(key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("private key is empty")
	}
	if _, err := xssh.ParsePrivateKey([]byte(key)); err != nil {
		var passphraseErr *xssh.PassphraseMissingError
		if errors.As(err, &passphraseErr) {
			return fmt.Errorf("encrypted private keys are not supported yet")
		}
		return fmt.Errorf("invalid SSH private key: %w", err)
	}
	return nil
}

// GetPublicKeyFromPrivate extracts the public key from a private key using ssh-keygen.
func GetPublicKeyFromPrivate(privateKey string) (string, error) {
	// Create a temporary file for the private key
	tmpFile, err := os.CreateTemp("", "ssh-manager-key-")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write private key with secure permissions
	if err := os.Chmod(tmpFile.Name(), 0600); err != nil {
		return "", fmt.Errorf("failed to set permissions: %w", err)
	}

	if _, err := tmpFile.WriteString(privateKey); err != nil {
		tmpFile.Close()
		return "", fmt.Errorf("failed to write private key: %w", err)
	}
	tmpFile.Close()

	// Run ssh-keygen -y to extract public key
	cmd := exec.Command("ssh-keygen", "-y", "-f", tmpFile.Name())
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to extract public key: %w, output: %s", err, string(output))
	}

	return strings.TrimSpace(string(output)), nil
}
