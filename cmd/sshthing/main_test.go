package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseExecArgsDirect(t *testing.T) {
	target, token, cmd, mode, err := parseExecArgs([]string{"-t", "GPU_Server", "--auth", "stk_x_y", "echo", "ok"})
	if err != nil {
		t.Fatalf("parseExecArgs returned error: %v", err)
	}
	if target != "GPU_Server" || token != "stk_x_y" || cmd != "echo ok" || mode != "direct" {
		t.Fatalf("unexpected parse result: %q %q %q %q", target, token, cmd, mode)
	}
}

func TestParseExecArgsAuthFile(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "token.txt")
	if err := os.WriteFile(p, []byte("stk_file_secret\n"), 0600); err != nil {
		t.Fatalf("failed to write token file: %v", err)
	}

	target, token, cmd, mode, err := parseExecArgs([]string{"-t", "CPU", "--auth-file", p, "uname", "-a"})
	if err != nil {
		t.Fatalf("parseExecArgs returned error: %v", err)
	}
	if target != "CPU" || token != "stk_file_secret" || cmd != "uname -a" || mode != "file" {
		t.Fatalf("unexpected parse result: %q %q %q %q", target, token, cmd, mode)
	}
}

func TestParseExecArgsRejectsMultipleAuthSources(t *testing.T) {
	_, _, _, _, err := parseExecArgs([]string{"-t", "GPU", "--auth", "a", "--auth-stdin", "echo"})
	if err == nil {
		t.Fatalf("expected error for multiple auth sources")
	}
}
