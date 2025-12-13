package ssh

import (
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

