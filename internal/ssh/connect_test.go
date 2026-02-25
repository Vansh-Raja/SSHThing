package ssh

import (
	"runtime"
	"strings"
	"testing"
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
