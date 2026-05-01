package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/Vansh-Raja/SSHThing/internal/app"
	"github.com/Vansh-Raja/SSHThing/internal/ssh"
	"github.com/Vansh-Raja/SSHThing/internal/unlock"
	"github.com/Vansh-Raja/SSHThing/internal/update"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

var version = "dev"

func main() {
	if ssh.IsAskpassInvocation() {
		if err := ssh.RunAskpassHelper(); err != nil {
			fmt.Fprintf(os.Stderr, "askpass error: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if len(os.Args) > 2 && os.Args[1] == update.HandoffArg {
		if err := update.RunHandoffFromFile(os.Args[2]); err != nil {
			fmt.Fprintf(os.Stderr, "update handoff error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if len(os.Args) > 1 && os.Args[1] == "exec" {
		if err := runExec(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "exec error: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "cp" {
		if err := runCp(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "cp error: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "put" {
		if err := runPut(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "put error: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "get" {
		if err := runGet(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "get error: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "session" {
		if err := runSession(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "session error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "version":
			fmt.Printf("sshthing %s\n", version)
			return
		case "--help", "-h", "help":
			fmt.Println("sshthing — SSHThing TUI")
			fmt.Println()
			fmt.Println("Usage:")
			fmt.Println("  sshthing            Run the TUI")
			fmt.Println("  sshthing exec       Run one token-auth command on a remote host")
			fmt.Println("  sshthing cp         Copy files to/from a remote host (scp-style)")
			fmt.Println("  sshthing put        Upload stdin (or --in <file>) to a remote path")
			fmt.Println("  sshthing get        Download a remote file to stdout (or --out <file>)")
			fmt.Println("  sshthing session    Manage local unlock session cache")
			fmt.Println("  sshthing --version  Print version")
			fmt.Println("  sshthing --help     Show this help")
			fmt.Println()
			fmt.Println("Exec Usage:")
			fmt.Println("  sshthing exec -t <target_label> --auth <token> \"command\"")
			fmt.Println("  sshthing exec -t <target_label> --auth-file <path> \"command\"")
			fmt.Println("  sshthing exec -t <target_label> --auth-stdin \"command\"")
			fmt.Println("  sshthing exec --in <local_file> -t <target> --auth-file <path> \"cmd reading stdin\"")
			fmt.Println()
			fmt.Println("File Transfer:")
			fmt.Println("  # Upload (scp-style; leading ':' marks the remote side)")
			fmt.Println("  sshthing cp -t <target> --auth-file <path> ./local :/remote/dir/")
			fmt.Println("  sshthing cp -t <target> --auth-file <path> -r ./dist/ :/srv/www/")
			fmt.Println()
			fmt.Println("  # Download")
			fmt.Println("  sshthing cp -t <target> --auth-file <path> :/var/log/app.log ./logs/")
			fmt.Println()
			fmt.Println("  # Streaming (verb form is cleaner for pipelines)")
			fmt.Println("  echo \"hello\" | sshthing put -t <target> --auth-file <path> /tmp/hello.txt")
			fmt.Println("  sshthing get -t <target> --auth-file <path> /tmp/hello.txt > ./local.txt")
			fmt.Println()
			fmt.Println("Session Usage:")
			fmt.Println("  printf 'MASTER_PASSWORD' | sshthing session unlock --password-stdin --ttl 15m")
			fmt.Println("  sshthing session status")
			fmt.Println("  sshthing session lock")
			return
		}
	}

	// Check for required OpenSSH tools before starting the TUI
	if err := ssh.CheckPrereqs(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Force TrueColor so all themes render correctly
	lipgloss.SetColorProfile(termenv.TrueColor)

	// Create the initial model
	m := app.NewModelWithVersion(version)

	// Create the Bubble Tea program with alternate screen
	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),       // Use alternate screen buffer
		tea.WithMouseCellMotion(), // Enable mouse support
	)

	// Run the program
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}

	// OSC 111: restore the terminal's original default background on exit
	fmt.Fprint(os.Stdout, "\x1b]111\x1b\\")
}

func runExec(args []string) error {
	target, token, command, authMode, inPath, err := parseExecArgs(args)
	if err != nil {
		return err
	}
	if authMode == "direct" {
		fmt.Fprintln(os.Stderr, "warning: --auth may leak via shell history/process args; prefer --auth-file or --auth-stdin")
	}

	rc, err := resolveTokenAndConn(target, token)
	if err != nil {
		return err
	}
	if rc.DBStore != nil {
		defer rc.DBStore.Close()
	}
	defer rc.FinalizeAfterRun()

	cmd, tempKey, err := ssh.ConnectExec(rc.Conn, command)
	if err != nil {
		return err
	}
	if tempKey != nil {
		defer tempKey.Cleanup()
	}

	// --in <file> overrides the inherited stdin so the local file's contents
	// are piped to the remote command (e.g. `psql -f -`, `kubectl apply -f -`).
	if inPath != "" {
		f, openErr := os.Open(inPath)
		if openErr != nil {
			return fmt.Errorf("--in: %w", openErr)
		}
		defer f.Close()
		cmd.Stdin = f
	}

	return propagateExitCode(cmd.Run())
}

// parseExecArgs walks `sshthing exec` argv. Returns the resolved target,
// token, remote command, auth source mode, and an optional --in path that
// overrides stdin. The auth flag parsing is delegated to extractAuthFlags so
// cp/put/get can share it.
func parseExecArgs(args []string) (target string, token string, command string, authMode string, inPath string, err error) {
	// Pull --in out before passing the rest to extractAuthFlags so it doesn't
	// land in the leftover positional args.
	filtered := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--in" {
			i++
			if i >= len(args) {
				return "", "", "", "", "", fmt.Errorf("missing value for --in")
			}
			inPath = strings.TrimSpace(args[i])
			continue
		}
		filtered = append(filtered, a)
	}

	af, leftover, perr := extractAuthFlags(filtered)
	if perr != nil {
		return "", "", "", "", "", perr
	}

	command = strings.TrimSpace(strings.Join(leftover, " "))
	if command == "" {
		return "", "", "", "", "", fmt.Errorf("remote command is required")
	}
	return af.Target, af.Token, command, af.AuthMode, inPath, nil
}

func runSession(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: sshthing session <unlock|lock|status>")
	}
	switch args[0] {
	case "lock":
		return unlock.Clear()
	case "status":
		_, exp, ok, err := unlock.Load()
		if err != nil {
			fmt.Println("session: unavailable")
			return nil
		}
		if !ok {
			fmt.Println("session: locked")
			return nil
		}
		fmt.Printf("session: unlocked until %s\n", exp.Local().Format(time.RFC3339))
		return nil
	case "unlock":
		ttl := 15 * time.Minute
		readStdin := false
		for i := 1; i < len(args); i++ {
			a := args[i]
			switch a {
			case "--password-stdin":
				readStdin = true
			case "--ttl":
				i++
				if i >= len(args) {
					return fmt.Errorf("missing value for --ttl")
				}
				d, err := time.ParseDuration(args[i])
				if err != nil {
					return fmt.Errorf("invalid ttl: %w", err)
				}
				ttl = d
			default:
				return fmt.Errorf("unknown session flag: %s", a)
			}
		}
		if !readStdin {
			return fmt.Errorf("unlock requires --password-stdin")
		}
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("failed to read password from stdin: %w", err)
		}
		pw := strings.TrimSpace(string(b))
		if pw == "" {
			return fmt.Errorf("empty password")
		}
		if err := unlock.Save(pw, ttl); err != nil {
			return err
		}
		fmt.Println("session: unlocked")
		return nil
	default:
		return fmt.Errorf("unknown session command: %s", args[0])
	}
}
