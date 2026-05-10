package ssh

import (
	"os/exec"
	"runtime"
	"strings"
)

// WrapForPasswordAuth wraps an arbitrary binary invocation (e.g. ssh, sshfs,
// sftp, scp) with the same sshpass / askpass plumbing the ssh.Connect path
// uses. This lets non-interactive callers run binaries that prompt for a
// password when no terminal is attached.
//
// If conn.Password is empty (or a private key is set), the returned command
// is just `exec.Command(binary, args...)` with the standard ssh env applied
// and a no-op cleanup. Otherwise the command is rewritten through sshpass or
// configured with askpass env vars per conn.PasswordBackendUnix (Unix only —
// Windows always uses askpass).
//
// The cleanup function MUST be called when the command exits. It closes the
// askpass server and any pipe file descriptors used by sshpass.
//
// This mirrors the unexported prepareClientCommand path inside connect.go,
// but exposed for callers in other packages (internal/mount, etc.).
func WrapForPasswordAuth(binary string, args []string, conn Connection) (*exec.Cmd, func(), error) {
	noop := func() {}

	// No password, or a private key is set — fall through to a plain command.
	if conn.Password == "" || conn.PrivateKey != "" {
		cmd := exec.Command(binary, args...)
		cmd.Env = sshEnv(conn.Term)
		return cmd, noop, nil
	}

	if runtime.GOOS == "windows" {
		return wrapAskpass(binary, args, conn)
	}

	backend := strings.TrimSpace(strings.ToLower(conn.PasswordBackendUnix))
	if backend == "" {
		backend = "sshpass_first"
	}

	if backend == "askpass_first" {
		if cmd, cleanup, err := wrapAskpass(binary, args, conn); err == nil {
			return cmd, cleanup, nil
		}
		if cmd, cleanup, err, ok := wrapSSHPass(binary, args, conn); ok {
			return cmd, cleanup, err
		}
	} else {
		if cmd, cleanup, err, ok := wrapSSHPass(binary, args, conn); ok {
			return cmd, cleanup, err
		}
		if cmd, cleanup, err := wrapAskpass(binary, args, conn); err == nil {
			return cmd, cleanup, nil
		}
	}

	// Last resort: bare command (will prompt on stdin and fail in daemon ctx).
	cmd := exec.Command(binary, args...)
	cmd.Env = sshEnv(conn.Term)
	return cmd, noop, nil
}

// wrapSSHPass mirrors prepareSSHPassCommand for arbitrary binaries.
func wrapSSHPass(binary string, args []string, conn Connection) (*exec.Cmd, func(), error, bool) {
	if !HasTool("sshpass") {
		return nil, nil, nil, false
	}
	holder := &TempKeyFile{}
	allArgs := append([]string{"-d", "3", binary}, args...)
	cmd := exec.Command("sshpass", allArgs...)
	cmd.Env = sshEnv(conn.Term)

	// The connect.go helper writes the password to a pipe and passes the
	// read-end as fd 3. Reuse the same approach via prepareSSHPassCommand's
	// holder.addCloser pattern.
	innerCmd, innerHolder, err, ok := prepareSSHPassCommand(binary, args, conn, holder)
	if !ok || err != nil {
		return nil, nil, err, true
	}
	cleanup := func() { _ = innerHolder.Cleanup() }
	return innerCmd, cleanup, nil, true
}

// wrapAskpass mirrors prepareAskpassCommand for arbitrary binaries.
func wrapAskpass(binary string, args []string, conn Connection) (*exec.Cmd, func(), error) {
	holder := &TempKeyFile{}
	cmd, h, err := prepareAskpassCommand(binary, args, conn, holder)
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() { _ = h.Cleanup() }
	return cmd, cleanup, nil
}
