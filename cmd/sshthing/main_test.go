package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseExecArgsDirect(t *testing.T) {
	af, cmd, inPath, err := parseExecArgs([]string{"-t", "GPU_Server", "--auth", "stk_x_y", "echo", "ok"})
	if err != nil {
		t.Fatalf("parseExecArgs returned error: %v", err)
	}
	if af.Target != "GPU_Server" || af.Token != "stk_x_y" || cmd != "echo ok" || af.AuthMode != "direct" || inPath != "" {
		t.Fatalf("unexpected parse result: %q %q %q %q %q", af.Target, af.Token, cmd, af.AuthMode, inPath)
	}
}

func TestParseExecArgsAuthFile(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "token.txt")
	if err := os.WriteFile(p, []byte("stk_file_secret\n"), 0600); err != nil {
		t.Fatalf("failed to write token file: %v", err)
	}

	af, cmd, inPath, err := parseExecArgs([]string{"-t", "CPU", "--auth-file", p, "uname", "-a"})
	if err != nil {
		t.Fatalf("parseExecArgs returned error: %v", err)
	}
	if af.Target != "CPU" || af.Token != "stk_file_secret" || cmd != "uname -a" || af.AuthMode != "file" || inPath != "" {
		t.Fatalf("unexpected parse result: %q %q %q %q %q", af.Target, af.Token, cmd, af.AuthMode, inPath)
	}
}

func TestParseExecArgsRejectsMultipleAuthSources(t *testing.T) {
	_, _, _, err := parseExecArgs([]string{"-t", "GPU", "--auth", "a", "--auth-stdin", "echo"})
	if err == nil {
		t.Fatalf("expected error for multiple auth sources")
	}
}

func TestParseExecArgs_WithIn(t *testing.T) {
	af, cmd, inPath, err := parseExecArgs([]string{
		"-t", "DB", "--auth", "stk_x", "--in", "/tmp/schema.sql", "psql", "-f", "-",
	})
	if err != nil {
		t.Fatalf("parseExecArgs returned error: %v", err)
	}
	if af.Target != "DB" || af.Token != "stk_x" || cmd != "psql -f -" || af.AuthMode != "direct" || inPath != "/tmp/schema.sql" {
		t.Fatalf("unexpected parse result: %q %q %q %q %q", af.Target, af.Token, cmd, af.AuthMode, inPath)
	}
}

func TestParseExecArgsTeamTokenTargetID(t *testing.T) {
	af, cmd, inPath, err := parseExecArgs([]string{
		"--team-id", "team_123", "--target-id", "host_456", "--auth", "stt_id_secret", "nvidia-smi",
	})
	if err != nil {
		t.Fatalf("parseExecArgs returned error: %v", err)
	}
	if af.TeamID != "team_123" || af.TargetID != "host_456" || af.Token != "stt_id_secret" || cmd != "nvidia-smi" || inPath != "" {
		t.Fatalf("unexpected parse result: %+v cmd=%q in=%q", af, cmd, inPath)
	}
}

func TestParseExecArgs_InMissingValue(t *testing.T) {
	_, _, _, err := parseExecArgs([]string{"-t", "DB", "--auth", "stk", "--in"})
	if err == nil {
		t.Fatal("expected error for --in without value")
	}
}
