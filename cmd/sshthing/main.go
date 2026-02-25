package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/Vansh-Raja/SSHThing/internal/app"
	"github.com/Vansh-Raja/SSHThing/internal/authtoken"
	"github.com/Vansh-Raja/SSHThing/internal/config"
	"github.com/Vansh-Raja/SSHThing/internal/db"
	"github.com/Vansh-Raja/SSHThing/internal/securestore"
	"github.com/Vansh-Raja/SSHThing/internal/ssh"
	"github.com/Vansh-Raja/SSHThing/internal/unlock"
	"github.com/Vansh-Raja/SSHThing/internal/update"
	tea "github.com/charmbracelet/bubbletea"
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
			fmt.Println("  sshthing exec       Run one token-auth command")
			fmt.Println("  sshthing session    Manage local unlock session cache")
			fmt.Println("  sshthing --version  Print version")
			fmt.Println("  sshthing --help     Show this help")
			fmt.Println()
			fmt.Println("Exec Usage:")
			fmt.Println("  sshthing exec -t <target_label> --auth <token> \"command\"")
			fmt.Println("  sshthing exec -t <target_label> --auth-file <path> \"command\"")
			fmt.Println("  sshthing exec -t <target_label> --auth-stdin \"command\"")
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
}

func runExec(args []string) error {
	target, token, command, authMode, err := parseExecArgs(args)
	if err != nil {
		return err
	}
	if authMode == "direct" {
		fmt.Fprintln(os.Stderr, "warning: --auth may leak via shell history/process args; prefer --auth-file or --auth-stdin")
	}
	vault, err := authtoken.LoadVault()
	if err != nil {
		return fmt.Errorf("failed to load token vault: %w", err)
	}
	pepper, _ := securestore.GetDevicePepper()
	resolved, err := vault.Resolve(token, target, pepper)
	if err != nil {
		return err
	}
	tokenIdx := resolved.TokenIndex

	if resolved.LegacyPayload != nil {
		p := resolved.LegacyPayload
		conn := ssh.Connection{
			Hostname:            p.Hostname,
			Username:            p.Username,
			Port:                p.Port,
			PasswordBackendUnix: p.PasswordBackendUnix,
			HostKeyPolicy:       p.HostKeyPolicy,
			KeepAliveSeconds:    p.KeepAliveSeconds,
			Term:                p.Term,
		}
		if p.KeyType == "password" {
			conn.Password = p.Secret
		} else {
			conn.PrivateKey = p.Secret
		}
		cmd, tempKey, err := ssh.ConnectExec(conn, command)
		if err != nil {
			return err
		}
		if tempKey != nil {
			defer tempKey.Cleanup()
		}
		err = cmd.Run()
		vault.MarkUsed(tokenIdx)
		_ = authtoken.SaveVault(vault)
		if err == nil {
			return nil
		}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ProcessState != nil {
			os.Exit(exitErr.ProcessState.ExitCode())
		}
		return err
	}

	dbUnlock := strings.TrimSpace(resolved.DBUnlockSecret)
	if dbUnlock == "" {
		cached, _, ok, _ := unlock.Load()
		if ok {
			dbUnlock = strings.TrimSpace(cached)
		}
	}
	if dbUnlock == "" {
		return fmt.Errorf("token is not active on this device and no unlock session is available")
	}

	store, err := db.Init(dbUnlock)
	if err != nil {
		return fmt.Errorf("failed to unlock database: %w", err)
	}
	defer store.Close()

	host, err := store.GetHostByID(resolved.HostID)
	if err != nil {
		return fmt.Errorf("failed to load target host: %w", err)
	}
	secret, err := store.GetHostSecret(resolved.HostID)
	if err != nil {
		return fmt.Errorf("failed to decrypt host secret: %w", err)
	}
	cfg, cfgErr := config.Load()
	if cfgErr != nil {
		cfg = config.Default()
	}
	term := ""
	switch cfg.SSH.TermMode {
	case config.TermXterm:
		term = "xterm-256color"
	case config.TermCustom:
		term = strings.TrimSpace(cfg.SSH.TermCustom)
	}

	conn := ssh.Connection{
		Hostname:            host.Hostname,
		Username:            host.Username,
		Port:                host.Port,
		PasswordBackendUnix: string(cfg.SSH.PasswordBackendUnix),
		HostKeyPolicy:       string(cfg.SSH.HostKeyPolicy),
		KeepAliveSeconds:    cfg.SSH.KeepAliveSeconds,
		Term:                term,
	}
	if host.KeyType == "password" {
		conn.Password = secret
	} else {
		conn.PrivateKey = secret
	}

	cmd, tempKey, err := ssh.ConnectExec(conn, command)
	if err != nil {
		return err
	}
	if tempKey != nil {
		defer tempKey.Cleanup()
	}
	err = cmd.Run()
	ttl := time.Duration(cfg.Automation.SessionTTLSeconds) * time.Second
	_ = unlock.Save(dbUnlock, ttl)
	vault.MarkUsed(tokenIdx)
	_ = authtoken.SaveVault(vault)
	if err == nil {
		return nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ProcessState != nil {
		os.Exit(exitErr.ProcessState.ExitCode())
	}
	return err
}

func parseExecArgs(args []string) (target string, token string, command string, authMode string, err error) {
	var authFile string
	remaining := make([]string, 0)

	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "-t", "--target":
			i++
			if i >= len(args) {
				return "", "", "", "", fmt.Errorf("missing value for %s", a)
			}
			target = strings.TrimSpace(args[i])
		case "--auth":
			if authMode != "" {
				return "", "", "", "", fmt.Errorf("only one auth source can be provided")
			}
			i++
			if i >= len(args) {
				return "", "", "", "", fmt.Errorf("missing value for --auth")
			}
			token = strings.TrimSpace(args[i])
			authMode = "direct"
		case "--auth-file":
			if authMode != "" {
				return "", "", "", "", fmt.Errorf("only one auth source can be provided")
			}
			i++
			if i >= len(args) {
				return "", "", "", "", fmt.Errorf("missing value for --auth-file")
			}
			authFile = strings.TrimSpace(args[i])
			authMode = "file"
		case "--auth-stdin":
			if authMode != "" {
				return "", "", "", "", fmt.Errorf("only one auth source can be provided")
			}
			authMode = "stdin"
		case "--":
			if i+1 < len(args) {
				remaining = append(remaining, args[i+1:]...)
			}
			i = len(args)
		default:
			remaining = append(remaining, a)
		}
	}

	if target == "" {
		return "", "", "", "", fmt.Errorf("target label is required (use -t)")
	}
	if authMode == "" {
		return "", "", "", "", fmt.Errorf("auth token is required (--auth, --auth-file, or --auth-stdin)")
	}

	switch authMode {
	case "file":
		b, readErr := os.ReadFile(authFile)
		if readErr != nil {
			return "", "", "", "", fmt.Errorf("failed to read auth file: %w", readErr)
		}
		token = strings.TrimSpace(string(b))
	case "stdin":
		b, readErr := io.ReadAll(os.Stdin)
		if readErr != nil {
			return "", "", "", "", fmt.Errorf("failed to read auth from stdin: %w", readErr)
		}
		token = strings.TrimSpace(string(b))
	}

	if token == "" {
		return "", "", "", "", fmt.Errorf("auth token cannot be empty")
	}

	command = strings.TrimSpace(strings.Join(remaining, " "))
	if command == "" {
		return "", "", "", "", fmt.Errorf("remote command is required")
	}
	return target, token, command, authMode, nil
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
