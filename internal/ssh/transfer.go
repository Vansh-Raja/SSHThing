package ssh

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

// TransferDirection is the direction of a single sftp batch op.
type TransferDirection int

const (
	// TransferPut uploads a local path to a remote path.
	TransferPut TransferDirection = iota
	// TransferGet downloads a remote path to a local path.
	TransferGet
)

// TransferOp is a single put/get inside an sftp batch script.
//
// Local and Remote are filesystem paths (no `-` sentinel — streaming via
// stdin/stdout uses ConnectExec instead, since OpenSSH sftp's batch mode does
// not support `-` for stdin/stdout). Paths are validated to reject characters
// that would break sftp's batch tokenizer.
type TransferOp struct {
	Direction TransferDirection
	Local     string
	Remote    string
	Recursive bool // -r in the batch command
	Preserve  bool // -P in the batch command (timestamps + mode)
}

// ConnectTransfer builds an `sftp -b <tmpbatch>` invocation that runs every op
// in order. It uses the same temp-key + askpass plumbing as ConnectExec /
// ConnectSFTP. The returned TempKeyFile owns both the temp key (if any) and
// the temp batch file; calling Cleanup() unlinks both.
//
// Quiet defaults to true at the caller level — sftp's progress bars on stderr
// are noise for scripted/agent use; pass quiet=false for human-driven CLI
// invocations that want progress.
func ConnectTransfer(conn Connection, ops []TransferOp, quiet bool) (*exec.Cmd, *TempKeyFile, error) {
	if len(ops) == 0 {
		return nil, nil, fmt.Errorf("ConnectTransfer: no transfer ops provided")
	}
	for i, op := range ops {
		if op.Local == "" || op.Remote == "" {
			return nil, nil, fmt.Errorf("ConnectTransfer: op %d has empty Local or Remote", i)
		}
		if op.Local == "-" || op.Remote == "-" {
			return nil, nil, fmt.Errorf("ConnectTransfer: op %d uses `-` sentinel; route to ConnectExec instead", i)
		}
		if err := validateBatchPath(op.Local); err != nil {
			return nil, nil, fmt.Errorf("ConnectTransfer: op %d local path: %w", i, err)
		}
		if err := validateBatchPath(op.Remote); err != nil {
			return nil, nil, fmt.Errorf("ConnectTransfer: op %d remote path: %w", i, err)
		}
	}

	batchPath, err := writeBatchFile(ops)
	if err != nil {
		return nil, nil, fmt.Errorf("ConnectTransfer: write batch: %w", err)
	}

	var tempKey *TempKeyFile
	var args []string

	args = append(args, "-b", batchPath)
	args = append(args, "-o", "StrictHostKeyChecking="+strictHostKeyChecking(conn.HostKeyPolicy))
	args = append(args, "-o", fmt.Sprintf("ServerAliveInterval=%d", keepAliveSeconds(conn.KeepAliveSeconds)))
	if quiet {
		args = append(args, "-q")
	}

	// sftp uses uppercase -P for the SSH port (lowercase -p is "preserve").
	if conn.Port != 22 && conn.Port != 0 {
		args = append(args, "-P", fmt.Sprintf("%d", conn.Port))
	}

	if conn.PrivateKey != "" {
		tempKey, err = NewTempKeyFile(conn.PrivateKey)
		if err != nil {
			_ = os.Remove(batchPath)
			return nil, nil, fmt.Errorf("ConnectTransfer: temp key: %w", err)
		}
		args = append(args, "-i", tempKey.Path())
	}

	passwordAuth := conn.PrivateKey == "" && conn.Password != ""
	if passwordAuth {
		args = append(args, "-o", "PreferredAuthentications=password,keyboard-interactive")
		args = append(args, "-o", "PubkeyAuthentication=no")
	}

	args = append(args, conn.Username+"@"+conn.Hostname)

	cmd, cleanupHolder, prepErr := prepareClientCommand("sftp", args, conn, tempKey)
	if prepErr != nil {
		if tempKey != nil {
			_ = tempKey.Cleanup()
		}
		_ = os.Remove(batchPath)
		return nil, nil, prepErr
	}
	if tempKey == nil {
		tempKey = cleanupHolder
	} else {
		tempKey.merge(cleanupHolder)
	}
	// We need a non-nil holder to register batch-file cleanup on. If neither
	// the PrivateKey path nor prepareClientCommand allocated one (i.e. no auth
	// state at all), fabricate a fresh holder so the batch file still gets
	// removed on Cleanup().
	if tempKey == nil {
		tempKey = &TempKeyFile{}
	}
	tempKey.addCleanup(func() error {
		return os.Remove(batchPath)
	})

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd, tempKey, nil
}

// validateBatchPath rejects characters that would break sftp's batch
// tokenizer or that we don't want to escape carefully in v1. Agents producing
// these paths shouldn't be feeding embedded quotes/newlines anyway.
func validateBatchPath(p string) error {
	if p == "" {
		return fmt.Errorf("path is empty")
	}
	for _, bad := range []string{"\"", "\\", "\n", "\r", "\x00"} {
		if strings.Contains(p, bad) {
			return fmt.Errorf("path contains unsupported character %q (v1 limitation)", bad)
		}
	}
	return nil
}

// writeBatchFile writes an sftp batch script to a 0600 temp file and returns
// the path. Each op becomes one line. Paths are double-quoted; sftp's batch
// tokenizer accepts double quotes for paths containing spaces.
func writeBatchFile(ops []TransferOp) (string, error) {
	tmpDir := filepath.Join(os.TempDir(), "ssh-manager")
	if err := os.MkdirAll(tmpDir, 0o700); err != nil {
		return "", err
	}
	filename := filepath.Join(tmpDir, "transfer_"+uuid.New().String()+".sftp")

	var b strings.Builder
	for _, op := range ops {
		switch op.Direction {
		case TransferPut:
			b.WriteString("put")
		case TransferGet:
			b.WriteString("get")
		default:
			return "", fmt.Errorf("unknown transfer direction %d", op.Direction)
		}
		if op.Recursive {
			b.WriteString(" -r")
		}
		if op.Preserve {
			b.WriteString(" -P")
		}
		if op.Direction == TransferPut {
			fmt.Fprintf(&b, " %s %s\n", quoteBatchPath(op.Local), quoteBatchPath(op.Remote))
		} else {
			fmt.Fprintf(&b, " %s %s\n", quoteBatchPath(op.Remote), quoteBatchPath(op.Local))
		}
	}

	if err := os.WriteFile(filename, []byte(b.String()), 0o600); err != nil {
		return "", err
	}
	return filename, nil
}

func quoteBatchPath(p string) string {
	// Caller has already passed validateBatchPath, so we know there's no
	// backslash, double quote, newline, or NUL inside p. Wrapping in double
	// quotes is therefore safe and makes spaces work without special handling.
	return "\"" + p + "\""
}
