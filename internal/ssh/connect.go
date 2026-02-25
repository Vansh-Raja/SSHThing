package ssh

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/google/uuid"
)

// Connection represents an SSH connection configuration
type Connection struct {
	Hostname   string
	Username   string
	Port       int
	PrivateKey string // Decrypted private key content
	Password   string // Decrypted password secret for password auth

	PasswordBackendUnix string // "sshpass_first" | "askpass_first"

	// Options
	HostKeyPolicy    string // "accept-new" | "strict" | "off"
	KeepAliveSeconds int
	Term             string // optional TERM override (env SSHTHING_SSH_TERM still wins)
}

// TempKeyFile manages a temporary file for the SSH private key
type TempKeyFile struct {
	path       string
	closers    []io.Closer
	cleanupFns []func() error
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

func (t *TempKeyFile) addCloser(c io.Closer) {
	if t == nil || c == nil {
		return
	}
	t.closers = append(t.closers, c)
}

func (t *TempKeyFile) addCleanup(fn func() error) {
	if t == nil || fn == nil {
		return
	}
	t.cleanupFns = append(t.cleanupFns, fn)
}

func (t *TempKeyFile) merge(other *TempKeyFile) {
	if t == nil || other == nil {
		return
	}
	t.closers = append(t.closers, other.closers...)
	t.cleanupFns = append(t.cleanupFns, other.cleanupFns...)
}

// Cleanup securely removes the temporary key file
func (t *TempKeyFile) Cleanup() error {
	if t == nil {
		return nil
	}
	if t.path != "" {
		// Overwrite the file content before deletion for extra security
		file, err := os.OpenFile(t.path, os.O_WRONLY, 0600)
		if err == nil {
			info, _ := file.Stat()
			if info != nil {
				zeros := make([]byte, info.Size())
				_, _ = file.Write(zeros)
			}
			_ = file.Close()
		}

		_ = os.Remove(t.path)
	}

	for _, c := range t.closers {
		_ = c.Close()
	}
	for _, fn := range t.cleanupFns {
		_ = fn()
	}
	t.closers = nil
	t.cleanupFns = nil
	t.path = ""

	return nil
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

	passwordAuth := conn.PrivateKey == "" && conn.Password != ""
	if passwordAuth {
		args = append(args, "-o", "PreferredAuthentications=password,keyboard-interactive")
		args = append(args, "-o", "PubkeyAuthentication=no")
	}

	// Add target
	target := conn.Username + "@" + conn.Hostname
	args = append(args, target)

	cmd, cleanupHolder, err := prepareClientCommand("ssh", args, conn, tempKey)
	if err != nil {
		if tempKey != nil {
			_ = tempKey.Cleanup()
		}
		return nil, nil, err
	}
	if tempKey == nil {
		tempKey = cleanupHolder
	} else {
		tempKey.merge(cleanupHolder)
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd, tempKey, nil
}

// ConnectExec establishes a non-interactive SSH command execution session.
// The remote command is passed as a single argument to ssh after the target host.
func ConnectExec(conn Connection, remoteCommand string) (*exec.Cmd, *TempKeyFile, error) {
	remoteCommand = strings.TrimSpace(remoteCommand)
	if remoteCommand == "" {
		return nil, nil, fmt.Errorf("remote command is required")
	}

	var tempKey *TempKeyFile
	var args []string

	args = append(args, "-T")
	args = append(args, "-o", "StrictHostKeyChecking="+strictHostKeyChecking(conn.HostKeyPolicy))
	args = append(args, "-o", fmt.Sprintf("ServerAliveInterval=%d", keepAliveSeconds(conn.KeepAliveSeconds)))

	if conn.Port != 22 && conn.Port != 0 {
		args = append(args, "-p", fmt.Sprintf("%d", conn.Port))
	}

	if conn.PrivateKey != "" {
		var err error
		tempKey, err = NewTempKeyFile(conn.PrivateKey)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create temp key file: %w", err)
		}
		args = append(args, "-i", tempKey.Path())
	}

	passwordAuth := conn.PrivateKey == "" && conn.Password != ""
	if passwordAuth {
		args = append(args, "-o", "PreferredAuthentications=password,keyboard-interactive")
		args = append(args, "-o", "PubkeyAuthentication=no")
	}

	target := conn.Username + "@" + conn.Hostname
	args = append(args, target, remoteCommand)

	cmd, cleanupHolder, err := prepareClientCommand("ssh", args, conn, tempKey)
	if err != nil {
		if tempKey != nil {
			_ = tempKey.Cleanup()
		}
		return nil, nil, err
	}
	if tempKey == nil {
		tempKey = cleanupHolder
	} else {
		tempKey.merge(cleanupHolder)
	}
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

	passwordAuth := conn.PrivateKey == "" && conn.Password != ""
	if passwordAuth {
		args = append(args, "-o", "PreferredAuthentications=password,keyboard-interactive")
		args = append(args, "-o", "PubkeyAuthentication=no")
	}

	// Add target
	target := conn.Username + "@" + conn.Hostname
	args = append(args, target)

	cmd, cleanupHolder, err := prepareClientCommand("sftp", args, conn, tempKey)
	if err != nil {
		if tempKey != nil {
			_ = tempKey.Cleanup()
		}
		return nil, nil, err
	}
	if tempKey == nil {
		tempKey = cleanupHolder
	} else {
		tempKey.merge(cleanupHolder)
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd, tempKey, nil
}

func prepareClientCommand(binary string, args []string, conn Connection, holder *TempKeyFile) (*exec.Cmd, *TempKeyFile, error) {
	password := conn.Password
	if password == "" || conn.PrivateKey != "" {
		cmd := exec.Command(binary, args...)
		cmd.Env = sshEnv(conn.Term)
		return cmd, holder, nil
	}

	if runtime.GOOS == "windows" {
		return prepareAskpassCommand(binary, args, conn, holder)
	}

	backend := strings.TrimSpace(strings.ToLower(conn.PasswordBackendUnix))
	if backend == "" {
		backend = "sshpass_first"
	}

	if backend == "askpass_first" {
		if cmd, h, err := prepareAskpassCommand(binary, args, conn, holder); err == nil {
			return cmd, h, nil
		}
		if cmd, h, err, ok := prepareSSHPassCommand(binary, args, conn, holder); ok {
			return cmd, h, err
		}
	} else {
		if cmd, h, err, ok := prepareSSHPassCommand(binary, args, conn, holder); ok {
			return cmd, h, err
		}
		if cmd, h, err := prepareAskpassCommand(binary, args, conn, holder); err == nil {
			return cmd, h, nil
		}
	}

	cmd := exec.Command(binary, args...)
	cmd.Env = sshEnv(conn.Term)
	return cmd, holder, nil
}

func ensureCleanupHolder(holder *TempKeyFile) *TempKeyFile {
	if holder != nil {
		return holder
	}
	return &TempKeyFile{}
}

func prepareSSHPassCommand(binary string, args []string, conn Connection, holder *TempKeyFile) (*exec.Cmd, *TempKeyFile, error, bool) {
	if !HasTool("sshpass") {
		return nil, holder, nil, false
	}

	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		return nil, holder, fmt.Errorf("failed to create sshpass pipe: %w", err), true
	}
	if _, err := writePipe.WriteString(conn.Password + "\n"); err != nil {
		_ = writePipe.Close()
		_ = readPipe.Close()
		return nil, holder, fmt.Errorf("failed to write sshpass password: %w", err), true
	}
	_ = writePipe.Close()

	allArgs := []string{"-d", "3", binary}
	allArgs = append(allArgs, args...)
	cmd := exec.Command("sshpass", allArgs...)
	cmd.Env = sshEnv(conn.Term)
	cmd.ExtraFiles = []*os.File{readPipe}

	holder = ensureCleanupHolder(holder)
	holder.addCloser(readPipe)
	return cmd, holder, nil, true
}

func prepareAskpassCommand(binary string, args []string, conn Connection, holder *TempKeyFile) (*exec.Cmd, *TempKeyFile, error) {
	server, err := startAskpassServer(conn.Password)
	if err != nil {
		return nil, holder, fmt.Errorf("failed to start askpass server: %w", err)
	}

	exePath, err := os.Executable()
	if err != nil {
		_ = server.Close()
		return nil, holder, fmt.Errorf("failed to resolve executable path: %w", err)
	}

	cmd := exec.Command(binary, args...)
	env := sshEnv(conn.Term)
	env = setEnv(env, "SSH_ASKPASS", exePath)
	env = setEnv(env, "SSH_ASKPASS_REQUIRE", "force")
	env = setEnv(env, askpassModeEnv, "1")
	env = setEnv(env, askpassEndpointEnv, server.Endpoint())
	env = setEnv(env, askpassNonceEnv, server.Nonce())
	cmd.Env = env

	holder = ensureCleanupHolder(holder)
	holder.addCleanup(server.Close)
	return cmd, holder, nil
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

// RunSSHExec runs a non-interactive remote command over SSH and waits for completion.
func RunSSHExec(conn Connection, remoteCommand string) error {
	cmd, tempKey, err := ConnectExec(conn, remoteCommand)
	if err != nil {
		return err
	}
	if tempKey != nil {
		defer tempKey.Cleanup()
	}
	return cmd.Run()
}
