package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/Vansh-Raja/SSHThing/internal/ssh"
)

// cpFlags captures the parsed argv for `sshthing cp`.
//
// At most one path on either side may be remote (leading `:`). At most one
// path on either side may be the streaming sentinel `-`. If any side is `-`,
// the transfer is routed through ConnectExec with stdin/stdout overrides
// instead of sftp batch mode (sftp doesn't natively support `-`).
type cpFlags struct {
	Auth      authFlags
	Sources   []string // positional args except the last
	Dest      string   // last positional
	Recursive bool
	Preserve  bool
	Quiet     bool
}

func parseCpArgs(args []string) (cpFlags, error) {
	var cf cpFlags
	cf.Quiet = true // sftp progress bars off by default; --progress to opt in

	// Pull cp-specific flags first; pass the rest through the shared auth
	// extractor.
	rest := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "-r", "--recursive":
			cf.Recursive = true
		case "-p", "--preserve":
			cf.Preserve = true
		case "-q", "--quiet":
			cf.Quiet = true
		case "--progress":
			cf.Quiet = false
		default:
			rest = append(rest, a)
		}
	}

	af, leftover, err := extractAuthFlags(rest)
	if err != nil {
		return cpFlags{}, err
	}
	cf.Auth = af

	if len(leftover) < 2 {
		return cpFlags{}, fmt.Errorf("cp requires at least one source and one destination")
	}
	cf.Sources = leftover[:len(leftover)-1]
	cf.Dest = leftover[len(leftover)-1]
	return cf, nil
}

// pathKind classifies a positional path. Leading `:` denotes remote (we strip
// it for the sftp / ssh consumer). Exactly `-` is the streaming sentinel.
type pathKind int

const (
	pathLocal pathKind = iota
	pathRemote
	pathStream
)

func classify(p string) (pathKind, string) {
	if p == "-" {
		return pathStream, ""
	}
	if strings.HasPrefix(p, ":") {
		return pathRemote, p[1:]
	}
	return pathLocal, p
}

func runCp(args []string) error {
	cf, err := parseCpArgs(args)
	if err != nil {
		return err
	}
	if cf.Auth.AuthMode == "direct" {
		fmt.Fprintln(os.Stderr, "warning: --auth may leak via shell history/process args; prefer --auth-file or --auth-stdin")
	}

	destKind, destPath := classify(cf.Dest)

	// Classify all sources up-front so we can reject mixed semantics before
	// touching the network or unlocking the DB.
	var srcs []cpSource
	streamCount := 0
	if destKind == pathStream {
		streamCount++
	}
	for _, raw := range cf.Sources {
		k, p := classify(raw)
		if k == pathStream {
			streamCount++
		}
		srcs = append(srcs, cpSource{Kind: k, Path: p, Raw: raw})
	}
	if streamCount > 1 {
		return fmt.Errorf("at most one path may be `-` (streaming sentinel)")
	}

	// Mixed-semantics rejection.
	if destKind == pathRemote {
		// Upload — every source must be local or the single stream sentinel.
		for _, s := range srcs {
			if s.Kind == pathRemote {
				return fmt.Errorf("upload destination is remote; sources must be local (got remote source %q)", s.Raw)
			}
		}
	} else if destKind == pathLocal {
		// Download — exactly one remote source, no others.
		if len(srcs) != 1 {
			return fmt.Errorf("download supports a single remote source")
		}
		if srcs[0].Kind != pathRemote {
			return fmt.Errorf("download requires a remote source (prefix with `:`); got %q", srcs[0].Raw)
		}
	} else {
		// destKind == pathStream — getting from remote to stdout.
		if len(srcs) != 1 || srcs[0].Kind != pathRemote {
			return fmt.Errorf("`cp <remote> -` requires a single remote source (prefix with `:`)")
		}
	}

	rc, err := resolveTokenAndConn(cf.Auth.Target, cf.Auth.Token)
	if err != nil {
		return err
	}
	if rc.DBStore != nil {
		defer rc.DBStore.Close()
	}
	defer rc.FinalizeAfterRun()

	// Streaming path: routes through ConnectExec with cat-style remote command.
	if streamCount == 1 {
		return runCpStreaming(rc, srcs, destKind, destPath)
	}

	// Filesystem path: build TransferOps and run sftp batch.
	var ops []ssh.TransferOp
	if destKind == pathRemote {
		for _, s := range srcs {
			ops = append(ops, ssh.TransferOp{
				Direction: ssh.TransferPut,
				Local:     s.Path,
				Remote:    destPath,
				Recursive: cf.Recursive,
				Preserve:  cf.Preserve,
			})
		}
	} else {
		// destKind == pathLocal, single remote source verified above.
		ops = append(ops, ssh.TransferOp{
			Direction: ssh.TransferGet,
			Local:     destPath,
			Remote:    srcs[0].Path,
			Recursive: cf.Recursive,
			Preserve:  cf.Preserve,
		})
	}

	cmd, holder, err := ssh.ConnectTransfer(rc.Conn, ops, cf.Quiet)
	if err != nil {
		return err
	}
	if holder != nil {
		defer holder.Cleanup()
	}
	return propagateExitCode(cmd.Run())
}

// cpSource is one classified positional path on the cp argv.
type cpSource struct {
	Kind pathKind
	Path string
	Raw  string
}

// runCpStreaming handles the `cp -` (or `cp <remote> -`) path by piping
// stdin/stdout through `ssh "cat …"` instead of sftp.
func runCpStreaming(rc *runContext, srcs []cpSource, destKind pathKind, destPath string) error {
	if destKind == pathRemote {
		// Upload from stdin.
		quoted, err := singleQuoteRemote(destPath)
		if err != nil {
			return err
		}
		remoteCmd := "cat > " + quoted
		cmd, tempKey, err := ssh.ConnectExec(rc.Conn, remoteCmd)
		if err != nil {
			return err
		}
		if tempKey != nil {
			defer tempKey.Cleanup()
		}
		// stdin is os.Stdin (default); leave it alone.
		return propagateExitCode(cmd.Run())
	}

	// destKind == pathStream — download remote → stdout.
	quoted, err := singleQuoteRemote(srcs[0].Path)
	if err != nil {
		return err
	}
	remoteCmd := "cat " + quoted
	cmd, tempKey, err := ssh.ConnectExec(rc.Conn, remoteCmd)
	if err != nil {
		return err
	}
	if tempKey != nil {
		defer tempKey.Cleanup()
	}
	// stdout is os.Stdout (default).
	return propagateExitCode(cmd.Run())
}

// singleQuoteRemote wraps a remote path in single quotes for the bourne shell
// `cat` command we send over ssh. We disallow embedded single quotes in v1 to
// avoid shell-escape gymnastics — agents producing pathological filenames can
// fall back to `sshthing exec`.
func singleQuoteRemote(p string) (string, error) {
	if p == "" {
		return "", fmt.Errorf("remote path is empty")
	}
	if strings.ContainsAny(p, "'\n\r\x00") {
		return "", fmt.Errorf("remote path contains unsupported character (single quote, newline, NUL); use sshthing exec for this case")
	}
	return "'" + p + "'", nil
}

// putFlags / getFlags share the auth + remote target, plus optional file
// override for stdin (put) or stdout (get).
type putFlags struct {
	Auth   authFlags
	Remote string // remote destination
	InPath string // optional --in <local>; stdin if empty
}

type getFlags struct {
	Auth    authFlags
	Remote  string // remote source
	OutPath string // optional --out <local>; stdout if empty
}

func parsePutArgs(args []string) (putFlags, error) {
	var pf putFlags
	rest := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "--in":
			i++
			if i >= len(args) {
				return putFlags{}, fmt.Errorf("missing value for --in")
			}
			pf.InPath = strings.TrimSpace(args[i])
		default:
			rest = append(rest, a)
		}
	}
	af, leftover, err := extractAuthFlags(rest)
	if err != nil {
		return putFlags{}, err
	}
	pf.Auth = af
	if len(leftover) != 1 {
		return putFlags{}, fmt.Errorf("put requires exactly one remote path argument")
	}
	pf.Remote = strings.TrimSpace(leftover[0])
	if pf.Remote == "" {
		return putFlags{}, fmt.Errorf("remote path is empty")
	}
	return pf, nil
}

func parseGetArgs(args []string) (getFlags, error) {
	var gf getFlags
	rest := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "--out":
			i++
			if i >= len(args) {
				return getFlags{}, fmt.Errorf("missing value for --out")
			}
			gf.OutPath = strings.TrimSpace(args[i])
		default:
			rest = append(rest, a)
		}
	}
	af, leftover, err := extractAuthFlags(rest)
	if err != nil {
		return getFlags{}, err
	}
	gf.Auth = af
	if len(leftover) != 1 {
		return getFlags{}, fmt.Errorf("get requires exactly one remote path argument")
	}
	gf.Remote = strings.TrimSpace(leftover[0])
	if gf.Remote == "" {
		return getFlags{}, fmt.Errorf("remote path is empty")
	}
	return gf, nil
}

func runPut(args []string) error {
	pf, err := parsePutArgs(args)
	if err != nil {
		return err
	}
	if pf.Auth.AuthMode == "direct" {
		fmt.Fprintln(os.Stderr, "warning: --auth may leak via shell history/process args; prefer --auth-file or --auth-stdin")
	}
	rc, err := resolveTokenAndConn(pf.Auth.Target, pf.Auth.Token)
	if err != nil {
		return err
	}
	if rc.DBStore != nil {
		defer rc.DBStore.Close()
	}
	defer rc.FinalizeAfterRun()

	quoted, err := singleQuoteRemote(pf.Remote)
	if err != nil {
		return err
	}
	cmd, tempKey, err := ssh.ConnectExec(rc.Conn, "cat > "+quoted)
	if err != nil {
		return err
	}
	if tempKey != nil {
		defer tempKey.Cleanup()
	}
	if pf.InPath != "" {
		f, openErr := os.Open(pf.InPath)
		if openErr != nil {
			return fmt.Errorf("--in: %w", openErr)
		}
		defer f.Close()
		cmd.Stdin = f
	}
	return propagateExitCode(cmd.Run())
}

func runGet(args []string) error {
	gf, err := parseGetArgs(args)
	if err != nil {
		return err
	}
	if gf.Auth.AuthMode == "direct" {
		fmt.Fprintln(os.Stderr, "warning: --auth may leak via shell history/process args; prefer --auth-file or --auth-stdin")
	}
	rc, err := resolveTokenAndConn(gf.Auth.Target, gf.Auth.Token)
	if err != nil {
		return err
	}
	if rc.DBStore != nil {
		defer rc.DBStore.Close()
	}
	defer rc.FinalizeAfterRun()

	quoted, err := singleQuoteRemote(gf.Remote)
	if err != nil {
		return err
	}
	cmd, tempKey, err := ssh.ConnectExec(rc.Conn, "cat "+quoted)
	if err != nil {
		return err
	}
	if tempKey != nil {
		defer tempKey.Cleanup()
	}
	if gf.OutPath != "" {
		f, openErr := os.Create(gf.OutPath)
		if openErr != nil {
			return fmt.Errorf("--out: %w", openErr)
		}
		defer f.Close()
		cmd.Stdout = f
	}
	return propagateExitCode(cmd.Run())
}
