package ssh

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestConnectSFTP_WithPortAndKey_AddsArgs(t *testing.T) {
	cmd, tempKey, err := ConnectSFTP(Connection{
		Hostname:   "example.com",
		Username:   "ubuntu",
		Port:       2222,
		PrivateKey: "dummy-key",
	})
	if err != nil {
		t.Fatalf("ConnectSFTP returned error: %v", err)
	}
	if tempKey == nil || tempKey.Path() == "" {
		t.Fatalf("expected temp key file to be created")
	}
	defer tempKey.Cleanup()

	args := strings.Join(cmd.Args, " ")
	if !strings.Contains(args, " -P 2222 ") {
		t.Fatalf("expected -P 2222 in args, got: %q", args)
	}
	if !strings.Contains(args, " -i ") {
		t.Fatalf("expected -i <path> in args, got: %q", args)
	}
	if !strings.HasSuffix(args, " ubuntu@example.com") {
		t.Fatalf("expected target at end, got: %q", args)
	}
}

func TestConnectSFTP_DefaultPort_NoKey(t *testing.T) {
	cmd, tempKey, err := ConnectSFTP(Connection{
		Hostname: "example.com",
		Username: "ubuntu",
		Port:     22,
	})
	if err != nil {
		t.Fatalf("ConnectSFTP returned error: %v", err)
	}
	if tempKey != nil {
		t.Fatalf("did not expect temp key file, got: %v", tempKey.Path())
	}

	args := strings.Join(cmd.Args, " ")
	if strings.Contains(args, " -P ") {
		t.Fatalf("did not expect -P when port is 22, got: %q", args)
	}
	if strings.Contains(args, " -i ") {
		t.Fatalf("did not expect -i without key, got: %q", args)
	}
	if !strings.HasSuffix(args, " ubuntu@example.com") {
		t.Fatalf("expected target at end, got: %q", args)
	}
}

func TestConnect_WithPassword_UsesAskpassEnv(t *testing.T) {
	cmd, tempKey, err := Connect(Connection{
		Hostname:            "example.com",
		Username:            "ubuntu",
		Password:            "super-secret",
		PasswordBackendUnix: "askpass_first",
	})
	if err != nil {
		t.Fatalf("Connect returned error: %v", err)
	}
	if tempKey == nil {
		t.Fatalf("expected cleanup handle for askpass session")
	}
	defer tempKey.Cleanup()

	if got := cmd.Args[0]; got != "ssh" {
		t.Fatalf("expected ssh command, got %q", got)
	}
	env := strings.Join(cmd.Env, "\n")
	if !strings.Contains(env, "SSH_ASKPASS=") {
		t.Fatalf("expected SSH_ASKPASS env, got: %q", env)
	}
	if !strings.Contains(env, "SSH_ASKPASS_REQUIRE=force") {
		t.Fatalf("expected SSH_ASKPASS_REQUIRE=force, got: %q", env)
	}
	if !strings.Contains(env, "SSHTHING_ASKPASS_MODE=1") {
		t.Fatalf("expected SSHTHING_ASKPASS_MODE=1, got: %q", env)
	}
}

func TestConnectExec_AppendsRemoteCommand(t *testing.T) {
	cmd, tempKey, err := ConnectExec(Connection{
		Hostname: "example.com",
		Username: "ubuntu",
		Port:     22,
	}, "echo hello")
	if err != nil {
		t.Fatalf("ConnectExec returned error: %v", err)
	}
	if tempKey != nil {
		defer tempKey.Cleanup()
	}

	args := strings.Join(cmd.Args, " ")
	if !strings.Contains(args, " -T ") {
		t.Fatalf("expected -T in args, got: %q", args)
	}
	if !strings.HasSuffix(args, " ubuntu@example.com echo hello") {
		t.Fatalf("expected remote command at end, got: %q", args)
	}
}

func TestConnectExecCaptured_UsesNonInteractiveOptionsAndCapturesOutput(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "ssh-args.txt")
	sshPath := filepath.Join(dir, "ssh")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > " + shellQuote(logPath) + "\nprintf 'probe-ok\\n'\n"
	if runtime.GOOS == "windows" {
		sshPath += ".bat"
		script = "@echo off\r\necho %* > " + logPath + "\r\necho probe-ok\r\n"
	}
	if err := os.WriteFile(sshPath, []byte(script), 0700); err != nil {
		t.Fatalf("write fake ssh: %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	result, err := ConnectExecCaptured(context.Background(), Connection{
		Hostname: "example.com",
		Username: "ubuntu",
		Port:     2222,
	}, "echo hello", ExecOptions{
		Timeout:        2 * time.Second,
		ConnectTimeout: time.Second,
		BatchMode:      true,
	})
	if err != nil {
		t.Fatalf("ConnectExecCaptured returned error: %v", err)
	}
	if strings.TrimSpace(result.Stdout) != "probe-ok" {
		t.Fatalf("expected captured stdout, got %q", result.Stdout)
	}
	argsData, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read fake ssh args: %v", err)
	}
	args := string(argsData)
	for _, want := range []string{"-T", "BatchMode=yes", "ConnectTimeout=1", "ServerAliveCountMax=1", "-p", "2222", "ubuntu@example.com", "echo hello"} {
		if !strings.Contains(args, want) {
			t.Fatalf("expected %q in args, got %q", want, args)
		}
	}
}

func TestConnectSFTP_WithPassword_SSHPassFirstWhenAvailable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sshpass backend not used on windows")
	}
	if !HasTool("sshpass") {
		t.Skip("sshpass not installed in test environment")
	}

	cmd, tempKey, err := ConnectSFTP(Connection{
		Hostname:            "example.com",
		Username:            "ubuntu",
		Password:            "super-secret",
		PasswordBackendUnix: "sshpass_first",
	})
	if err != nil {
		t.Fatalf("ConnectSFTP returned error: %v", err)
	}
	if tempKey == nil {
		t.Fatalf("expected cleanup handle for sshpass session")
	}
	defer tempKey.Cleanup()

	args := strings.Join(cmd.Args, " ")
	if !strings.HasPrefix(args, "sshpass -d 3 sftp ") {
		t.Fatalf("expected sshpass command prefix, got: %q", args)
	}
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
