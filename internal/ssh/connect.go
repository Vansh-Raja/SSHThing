package ssh

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

// Connection represents an SSH connection configuration
type Connection struct {
	Hostname   string
	Username   string
	Port       int
	PrivateKey string // Decrypted private key content
	Password   string // For password auth (not recommended)

	// Options
	HostKeyPolicy    string // "accept-new" | "strict" | "off"
	KeepAliveSeconds int
	Term             string // optional TERM override (env SSHTHING_SSH_TERM still wins)
}

// TempKeyFile manages a temporary file for the SSH private key
type TempKeyFile struct {
	path string
}

// NewTempKeyFile creates a secure temporary file for the private key.
// The file is created with 600 permissions (owner read/write only).
func NewTempKeyFile(privateKey string) (*TempKeyFile, error) {
	privateKey = strings.ReplaceAll(privateKey, "\r\n", "\n")
	privateKey = strings.ReplaceAll(privateKey, "\r", "\n")
	if privateKey != "" && !strings.HasSuffix(privateKey, "\n") {
		privateKey += "\n"
	}

	// Create temp directory if it doesn't exist
	tmpDir := filepath.Join(os.TempDir(), "ssh-manager")
	if err := os.MkdirAll(tmpDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}

	// Generate unique filename
	filename := fmt.Sprintf("key_%s", uuid.New().String())
	keyPath := filepath.Join(tmpDir, filename)

	// Create file with secure permissions
	file, err := os.OpenFile(keyPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp key file: %w", err)
	}
	defer file.Close()

	// Write the private key
	if _, err := file.WriteString(privateKey); err != nil {
		os.Remove(keyPath)
		return nil, fmt.Errorf("failed to write private key: %w", err)
	}

	return &TempKeyFile{path: keyPath}, nil
}

// Path returns the path to the temporary key file
func (t *TempKeyFile) Path() string {
	return t.path
}

// Cleanup securely removes the temporary key file
func (t *TempKeyFile) Cleanup() error {
	if t.path == "" {
		return nil
	}

	// Overwrite the file content before deletion for extra security
	file, err := os.OpenFile(t.path, os.O_WRONLY, 0600)
	if err == nil {
		info, _ := file.Stat()
		if info != nil {
			zeros := make([]byte, info.Size())
			file.Write(zeros)
		}
		file.Close()
	}

	return os.Remove(t.path)
}

// Connect establishes an SSH connection.
// It returns the exec.Cmd that can be used to run the SSH session.
// The caller is responsible for cleaning up the temp key file after the session ends.
func Connect(conn Connection) (*exec.Cmd, *TempKeyFile, error) {
	var tempKey *TempKeyFile
	var args []string

	// Build SSH command arguments
	args = append(args, "-o", "StrictHostKeyChecking="+strictHostKeyChecking(conn.HostKeyPolicy))
	args = append(args, "-o", fmt.Sprintf("ServerAliveInterval=%d", keepAliveSeconds(conn.KeepAliveSeconds)))

	// Add port if not default
	if conn.Port != 22 && conn.Port != 0 {
		args = append(args, "-p", fmt.Sprintf("%d", conn.Port))
	}

	// Handle authentication
	if conn.PrivateKey != "" {
		// Key-based authentication
		var err error
		tempKey, err = NewTempKeyFile(conn.PrivateKey)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create temp key file: %w", err)
		}
		args = append(args, "-i", tempKey.Path())
	}

	// Add target
	target := conn.Username + "@" + conn.Hostname
	args = append(args, target)

	// Create the command
	cmd := exec.Command("ssh", args...)
	cmd.Env = sshEnv(conn.Term)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd, tempKey, nil
}

// ConnectSFTP establishes an interactive SFTP session using the system `sftp` client.
// It returns the exec.Cmd that can be used to run the session and any temp key file
// that must be cleaned up after exit.
func ConnectSFTP(conn Connection) (*exec.Cmd, *TempKeyFile, error) {
	var tempKey *TempKeyFile
	var args []string

	// Pass SSH options through to the underlying transport.
	args = append(args, "-o", "StrictHostKeyChecking="+strictHostKeyChecking(conn.HostKeyPolicy))
	args = append(args, "-o", fmt.Sprintf("ServerAliveInterval=%d", keepAliveSeconds(conn.KeepAliveSeconds)))

	// sftp uses -P (uppercase) for port.
	if conn.Port != 22 && conn.Port != 0 {
		args = append(args, "-P", fmt.Sprintf("%d", conn.Port))
	}

	// Handle authentication
	if conn.PrivateKey != "" {
		var err error
		tempKey, err = NewTempKeyFile(conn.PrivateKey)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create temp key file: %w", err)
		}
		args = append(args, "-i", tempKey.Path())
	}

	// Add target
	target := conn.Username + "@" + conn.Hostname
	args = append(args, target)

	cmd := exec.Command("sftp", args...)
	cmd.Env = sshEnv(conn.Term)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd, tempKey, nil
}

func sshEnv(termOverride string) []string {
	env := os.Environ()

	term := os.Getenv("SSHTHING_SSH_TERM")
	if term == "" {
		if strings.TrimSpace(termOverride) != "" {
			term = termOverride
		} else {
			term = os.Getenv("TERM")
		}
	}
	if term == "xterm-ghostty" {
		// Many servers don't have Ghostty's terminfo entry installed yet, so
		// force a widely-supported TERM for sessions launched via this app.
		// See Ghostty docs: ssh-env / ssh-terminfo.
		term = "xterm-256color"
	}
	if term != "" {
		env = setEnv(env, "TERM", term)
	}
	return env
}

func setEnv(env []string, key, value string) []string {
	prefix := key + "="
	for i, kv := range env {
		if strings.HasPrefix(kv, prefix) {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}

func strictHostKeyChecking(policy string) string {
	switch strings.TrimSpace(strings.ToLower(policy)) {
	case "strict", "yes":
		return "yes"
	case "off", "no":
		return "no"
	default:
		return "accept-new"
	}
}

func keepAliveSeconds(v int) int {
	if v <= 0 {
		return 60
	}
	if v > 600 {
		return 600
	}
	return v
}

// RunSSH runs an SSH session and waits for it to complete.
// It handles cleanup of the temporary key file automatically.
func RunSSH(conn Connection) error {
	cmd, tempKey, err := Connect(conn)
	if err != nil {
		return err
	}

	// Ensure cleanup happens
	if tempKey != nil {
		defer tempKey.Cleanup()
	}

	// Run the SSH session
	return cmd.Run()
}
