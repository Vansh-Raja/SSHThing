package main

import (
	"strings"
	"testing"
)

func TestParseCpArgs_UploadOneFile(t *testing.T) {
	cf, err := parseCpArgs([]string{
		"-t", "Server", "--auth", "stk_x", "./local.txt", ":/remote/path",
	})
	if err != nil {
		t.Fatalf("parseCpArgs: %v", err)
	}
	if cf.Auth.Target != "Server" {
		t.Errorf("target = %q, want Server", cf.Auth.Target)
	}
	if len(cf.Sources) != 1 || cf.Sources[0] != "./local.txt" {
		t.Errorf("sources = %v, want [./local.txt]", cf.Sources)
	}
	if cf.Dest != ":/remote/path" {
		t.Errorf("dest = %q, want :/remote/path", cf.Dest)
	}
	if cf.Recursive || cf.Preserve {
		t.Errorf("flags = recursive %v preserve %v, want both false", cf.Recursive, cf.Preserve)
	}
	if !cf.Quiet {
		t.Error("quiet should default to true")
	}
}

func TestParseCpArgs_DownloadOneFile(t *testing.T) {
	cf, err := parseCpArgs([]string{
		"-t", "Server", "--auth", "stk_x", ":/var/log/app.log", "./pulled.log",
	})
	if err != nil {
		t.Fatalf("parseCpArgs: %v", err)
	}
	if len(cf.Sources) != 1 || cf.Sources[0] != ":/var/log/app.log" {
		t.Errorf("sources = %v", cf.Sources)
	}
	if cf.Dest != "./pulled.log" {
		t.Errorf("dest = %q", cf.Dest)
	}
}

func TestParseCpArgs_MultiSourceUpload(t *testing.T) {
	cf, err := parseCpArgs([]string{
		"-t", "S", "--auth", "x", "a.txt", "b.txt", "c.txt", ":/tmp/",
	})
	if err != nil {
		t.Fatalf("parseCpArgs: %v", err)
	}
	if len(cf.Sources) != 3 {
		t.Errorf("sources len = %d, want 3 (%v)", len(cf.Sources), cf.Sources)
	}
	if cf.Dest != ":/tmp/" {
		t.Errorf("dest = %q", cf.Dest)
	}
}

func TestParseCpArgs_RecursiveAndPreserve(t *testing.T) {
	cf, err := parseCpArgs([]string{
		"-r", "-p", "-t", "S", "--auth", "x", "./dist/", ":/srv/www/",
	})
	if err != nil {
		t.Fatalf("parseCpArgs: %v", err)
	}
	if !cf.Recursive {
		t.Error("recursive flag not parsed")
	}
	if !cf.Preserve {
		t.Error("preserve flag not parsed")
	}
}

func TestParseCpArgs_ProgressFlagDisablesQuiet(t *testing.T) {
	cf, err := parseCpArgs([]string{"--progress", "-t", "S", "--auth", "x", "a.txt", ":/tmp/"})
	if err != nil {
		t.Fatalf("parseCpArgs: %v", err)
	}
	if cf.Quiet {
		t.Error("--progress should disable quiet")
	}
}

func TestParseCpArgs_RejectsTooFewArgs(t *testing.T) {
	_, err := parseCpArgs([]string{"-t", "S", "--auth", "x", ":/only-one"})
	if err == nil {
		t.Fatal("expected error for missing source/dest")
	}
}

func TestParseCpArgs_RejectsMissingAuth(t *testing.T) {
	_, err := parseCpArgs([]string{"-t", "S", "a.txt", ":/tmp/"})
	if err == nil {
		t.Fatal("expected error for missing auth")
	}
}

func TestParseCpArgs_RejectsMultipleAuthSources(t *testing.T) {
	_, err := parseCpArgs([]string{"-t", "S", "--auth", "x", "--auth-stdin", "a.txt", ":/tmp/"})
	if err == nil {
		t.Fatal("expected error for multiple auth sources")
	}
}

func TestClassify(t *testing.T) {
	cases := []struct {
		in       string
		wantKind pathKind
		wantPath string
	}{
		{"./local", pathLocal, "./local"},
		{"/abs/local", pathLocal, "/abs/local"},
		{":/remote/path", pathRemote, "/remote/path"},
		{":relative/remote", pathRemote, "relative/remote"},
		{"-", pathStream, ""},
	}
	for _, c := range cases {
		k, p := classify(c.in)
		if k != c.wantKind || p != c.wantPath {
			t.Errorf("classify(%q) = (%v,%q), want (%v,%q)", c.in, k, p, c.wantKind, c.wantPath)
		}
	}
}

func TestSingleQuoteRemote_Allows(t *testing.T) {
	got, err := singleQuoteRemote("/var/log/app.log")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got != "'/var/log/app.log'" {
		t.Errorf("got %q", got)
	}
}

func TestSingleQuoteRemote_Rejects(t *testing.T) {
	for _, p := range []string{
		"",
		"path with 'single' quote",
		"path with \nnewline",
		"path with \rcr",
	} {
		if _, err := singleQuoteRemote(p); err == nil {
			t.Errorf("expected reject for %q", p)
		}
	}
}

func TestParsePutArgs(t *testing.T) {
	pf, err := parsePutArgs([]string{"-t", "S", "--auth", "x", "/tmp/foo"})
	if err != nil {
		t.Fatalf("parsePutArgs: %v", err)
	}
	if pf.Auth.Target != "S" || pf.Remote != "/tmp/foo" || pf.InPath != "" {
		t.Errorf("got %+v", pf)
	}
}

func TestParsePutArgs_WithIn(t *testing.T) {
	pf, err := parsePutArgs([]string{"-t", "S", "--auth", "x", "--in", "./local", "/tmp/foo"})
	if err != nil {
		t.Fatalf("parsePutArgs: %v", err)
	}
	if pf.InPath != "./local" {
		t.Errorf("InPath = %q, want ./local", pf.InPath)
	}
}

func TestParsePutArgs_RejectsExtraArgs(t *testing.T) {
	_, err := parsePutArgs([]string{"-t", "S", "--auth", "x", "/tmp/foo", "extra"})
	if err == nil || !strings.Contains(err.Error(), "exactly one") {
		t.Fatalf("expected 'exactly one' error, got %v", err)
	}
}

func TestParsePutArgs_RejectsMissingRemote(t *testing.T) {
	_, err := parsePutArgs([]string{"-t", "S", "--auth", "x"})
	if err == nil {
		t.Fatal("expected error for missing remote path")
	}
}

func TestParseGetArgs(t *testing.T) {
	gf, err := parseGetArgs([]string{"-t", "S", "--auth", "x", "/tmp/foo"})
	if err != nil {
		t.Fatalf("parseGetArgs: %v", err)
	}
	if gf.Auth.Target != "S" || gf.Remote != "/tmp/foo" || gf.OutPath != "" {
		t.Errorf("got %+v", gf)
	}
}

func TestParseGetArgs_WithOut(t *testing.T) {
	gf, err := parseGetArgs([]string{"-t", "S", "--auth", "x", "--out", "./local", "/tmp/foo"})
	if err != nil {
		t.Fatalf("parseGetArgs: %v", err)
	}
	if gf.OutPath != "./local" {
		t.Errorf("OutPath = %q, want ./local", gf.OutPath)
	}
}

func TestParseGetArgs_RejectsExtraArgs(t *testing.T) {
	_, err := parseGetArgs([]string{"-t", "S", "--auth", "x", "/tmp/foo", "extra"})
	if err == nil {
		t.Fatal("expected error for too many positional args")
	}
}
