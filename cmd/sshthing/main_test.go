package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseExecArgsDirect(t *testing.T) {
	target, token, cmd, mode, inPath, err := parseExecArgs([]string{"-t", "GPU_Server", "--auth", "stk_x_y", "echo", "ok"})
	if err != nil {
		t.Fatalf("parseExecArgs returned error: %v", err)
	}
	if target != "GPU_Server" || token != "stk_x_y" || cmd != "echo ok" || mode != "direct" || inPath != "" {
		t.Fatalf("unexpected parse result: %q %q %q %q %q", target, token, cmd, mode, inPath)
	}
}

func TestParseExecArgsAuthFile(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "token.txt")
	if err := os.WriteFile(p, []byte("stk_file_secret\n"), 0600); err != nil {
		t.Fatalf("failed to write token file: %v", err)
	}

	target, token, cmd, mode, inPath, err := parseExecArgs([]string{"-t", "CPU", "--auth-file", p, "uname", "-a"})
	if err != nil {
		t.Fatalf("parseExecArgs returned error: %v", err)
	}
	if target != "CPU" || token != "stk_file_secret" || cmd != "uname -a" || mode != "file" || inPath != "" {
		t.Fatalf("unexpected parse result: %q %q %q %q %q", target, token, cmd, mode, inPath)
	}
}

func TestParseExecArgsRejectsMultipleAuthSources(t *testing.T) {
	_, _, _, _, _, err := parseExecArgs([]string{"-t", "GPU", "--auth", "a", "--auth-stdin", "echo"})
	if err == nil {
		t.Fatalf("expected error for multiple auth sources")
	}
}

func TestParseExecArgs_WithIn(t *testing.T) {
	target, token, cmd, mode, inPath, err := parseExecArgs([]string{
		"-t", "DB", "--auth", "stk_x", "--in", "/tmp/schema.sql", "psql", "-f", "-",
	})
	if err != nil {
		t.Fatalf("parseExecArgs returned error: %v", err)
	}
	if target != "DB" || token != "stk_x" || cmd != "psql -f -" || mode != "direct" || inPath != "/tmp/schema.sql" {
		t.Fatalf("unexpected parse result: %q %q %q %q %q", target, token, cmd, mode, inPath)
	}
}

func TestParseExecArgs_InMissingValue(t *testing.T) {
	_, _, _, _, _, err := parseExecArgs([]string{"-t", "DB", "--auth", "stk", "--in"})
	if err == nil {
		t.Fatal("expected error for --in without value")
	}
}
