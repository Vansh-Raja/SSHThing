package ssh

import (
	"errors"
	"os"
	"strings"
	"testing"
)

func TestWriteBatchFile_PutGet(t *testing.T) {
	ops := []TransferOp{
		{Direction: TransferPut, Local: "./local", Remote: "/remote"},
		{Direction: TransferGet, Local: "./pulled", Remote: "/var/log/app.log", Preserve: true},
		{Direction: TransferPut, Local: "./dist", Remote: "/srv/www", Recursive: true, Preserve: true},
	}
	path, err := writeBatchFile(ops)
	if err != nil {
		t.Fatalf("writeBatchFile: %v", err)
	}
	defer os.Remove(path)

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read batch: %v", err)
	}
	got := string(body)

	want := "put \"./local\" \"/remote\"\n" +
		"get -P \"/var/log/app.log\" \"./pulled\"\n" +
		"put -r -P \"./dist\" \"/srv/www\"\n"
	if got != want {
		t.Fatalf("batch content mismatch\n  got: %q\n want: %q", got, want)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	// On Windows os.Stat reports 0666 — we don't enforce there. Unix should
	// show 0600.
	if mode := info.Mode().Perm(); mode != 0o600 && mode != 0o666 {
		t.Errorf("batch file perms = %o, want 0600 (or 0666 on Windows)", mode)
	}
}

func TestValidateBatchPath_RejectsBadChars(t *testing.T) {
	cases := []struct {
		path    string
		wantErr bool
	}{
		{"/normal/path", false},
		{"./relative", false},
		{"path with spaces", false},
		{"", true},
		{"path\"with\"quote", true},
		{"path\\with\\backslash", true},
		{"path\nwith\nnewline", true},
		{"path\rwith\rcr", true},
		{"path\x00null", true},
	}
	for _, c := range cases {
		err := validateBatchPath(c.path)
		if (err != nil) != c.wantErr {
			t.Errorf("validateBatchPath(%q) err=%v, wantErr=%v", c.path, err, c.wantErr)
		}
	}
}

func TestConnectTransfer_RejectsEmptyOps(t *testing.T) {
	_, _, err := ConnectTransfer(Connection{Hostname: "h", Username: "u"}, nil, true)
	if err == nil {
		t.Fatal("expected error for empty ops")
	}
	if !strings.Contains(err.Error(), "no transfer ops") {
		t.Errorf("error = %v, want 'no transfer ops'", err)
	}
}

func TestConnectTransfer_RejectsDashSentinel(t *testing.T) {
	ops := []TransferOp{{Direction: TransferPut, Local: "-", Remote: "/remote"}}
	_, _, err := ConnectTransfer(Connection{Hostname: "h", Username: "u"}, ops, true)
	if err == nil {
		t.Fatal("expected error for `-` sentinel")
	}
	if !strings.Contains(err.Error(), "-") {
		t.Errorf("error = %v, want mention of `-`", err)
	}
}

func TestConnectTransfer_RejectsBadPath(t *testing.T) {
	ops := []TransferOp{{Direction: TransferPut, Local: "ok", Remote: "bad\"quote"}}
	_, _, err := ConnectTransfer(Connection{Hostname: "h", Username: "u"}, ops, true)
	if err == nil {
		t.Fatal("expected error for unsupported character")
	}
}

func TestConnectTransfer_ArgvShape(t *testing.T) {
	ops := []TransferOp{{Direction: TransferPut, Local: "./a", Remote: "/b"}}
	conn := Connection{
		Hostname:         "example.com",
		Username:         "alice",
		Port:             2222,
		Password:         "hunter2",
		HostKeyPolicy:    "strict",
		KeepAliveSeconds: 60,
	}
	cmd, holder, err := ConnectTransfer(conn, ops, true)
	if err != nil {
		t.Fatalf("ConnectTransfer: %v", err)
	}
	defer func() {
		_ = holder.Cleanup()
	}()

	args := cmd.Args
	if len(args) == 0 || args[0] != "sftp" && !strings.HasSuffix(args[0], "sftp") &&
		!strings.HasSuffix(args[0], "/sh") /* askpass wrapper */ {
		// On password-auth paths the binary may be wrapped in a shell — the
		// underlying sftp argv still appears later. We just check that "sftp"
		// shows up somewhere in the argv chain.
	}

	full := strings.Join(args, " ")
	if !strings.Contains(full, "sftp") {
		t.Errorf("argv missing sftp: %v", args)
	}
	if !strings.Contains(full, "-b ") {
		t.Errorf("argv missing -b batchfile flag: %v", args)
	}
	if !strings.Contains(full, "-q") {
		t.Errorf("argv missing -q (quiet): %v", args)
	}
	if !strings.Contains(full, "-P 2222") {
		t.Errorf("argv missing port flag: %v", args)
	}
	if !strings.Contains(full, "alice@example.com") {
		t.Errorf("argv missing user@host: %v", args)
	}
	// Password auth requires opting out of pubkey
	if !strings.Contains(full, "PubkeyAuthentication=no") {
		t.Errorf("argv missing PubkeyAuthentication=no for password auth: %v", args)
	}
}

func TestConnectTransfer_NoQuietFlag(t *testing.T) {
	ops := []TransferOp{{Direction: TransferGet, Local: "./a", Remote: "/b"}}
	cmd, holder, err := ConnectTransfer(Connection{Hostname: "h", Username: "u"}, ops, false)
	if err != nil {
		t.Fatalf("ConnectTransfer: %v", err)
	}
	defer func() {
		_ = holder.Cleanup()
	}()
	for _, a := range cmd.Args {
		if a == "-q" {
			t.Fatalf("argv should not contain -q when quiet=false: %v", cmd.Args)
		}
	}
}

func TestConnectTransfer_BatchFileGetsCleanedUp(t *testing.T) {
	ops := []TransferOp{{Direction: TransferPut, Local: "./a", Remote: "/b"}}
	cmd, holder, err := ConnectTransfer(Connection{Hostname: "h", Username: "u"}, ops, true)
	if err != nil {
		t.Fatalf("ConnectTransfer: %v", err)
	}

	// Pull the batch path straight out of the argv (token after -b).
	var batchPath string
	for i, a := range cmd.Args {
		if a == "-b" && i+1 < len(cmd.Args) {
			batchPath = cmd.Args[i+1]
			break
		}
	}
	if batchPath == "" {
		t.Fatalf("could not find -b <batchfile> in argv: %v", cmd.Args)
	}

	if _, err := os.Stat(batchPath); errors.Is(err, os.ErrNotExist) {
		t.Fatalf("batch file should exist before cleanup, got %v", err)
	}

	if err := holder.Cleanup(); err != nil {
		t.Fatalf("Cleanup: %v", err)
	}
	if _, err := os.Stat(batchPath); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("batch file should be removed after Cleanup, stat err=%v", err)
	}
}
