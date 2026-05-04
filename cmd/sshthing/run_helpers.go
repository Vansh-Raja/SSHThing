package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/Vansh-Raja/SSHThing/internal/authtoken"
	"github.com/Vansh-Raja/SSHThing/internal/config"
	"github.com/Vansh-Raja/SSHThing/internal/db"
	"github.com/Vansh-Raja/SSHThing/internal/securestore"
	"github.com/Vansh-Raja/SSHThing/internal/ssh"
	"github.com/Vansh-Raja/SSHThing/internal/teams"
	"github.com/Vansh-Raja/SSHThing/internal/teamsclient"
	"github.com/Vansh-Raja/SSHThing/internal/unlock"
)

// runContext bundles everything a token-authenticated subcommand needs after
// resolving a target token: a ready-to-use ssh.Connection plus the bookkeeping
// state needed to record token use / refresh the unlock TTL after the command
// runs. Mirrors the legacy-vs-v2 split that runExec used to do inline.
type runContext struct {
	Vault      *authtoken.Vault
	TokenIdx   int
	Conn       ssh.Connection
	DBStore    *db.Store // non-nil only on the v2 path; caller defers Close
	DBUnlock   string    // empty on legacy path
	SessionTTL time.Duration

	TeamClient  *teamsclient.Client
	TeamToken   string
	ExecutionID string
}

// FinalizeAfterRun records the token as used, refreshes the unlock TTL on the
// v2 path, and saves the vault. Safe to call from a deferred wrapper — does
// not touch exit status. Caller is responsible for closing rc.DBStore (it
// returns it open for parity with the original runExec, in case callers want
// to reach into the db after the command runs).
func (rc *runContext) FinalizeAfterRun() {
	if rc == nil {
		return
	}
	if rc.DBUnlock != "" {
		_ = unlock.Save(rc.DBUnlock, rc.SessionTTL)
	}
	if rc.Vault != nil {
		rc.Vault.MarkUsed(rc.TokenIdx)
		_ = authtoken.SaveVault(rc.Vault)
	}
}

func (rc *runContext) FinishTeamExecution(runErr error) {
	if rc == nil || rc.TeamClient == nil || strings.TrimSpace(rc.ExecutionID) == "" || strings.TrimSpace(rc.TeamToken) == "" {
		return
	}
	status := "completed"
	errMsg := ""
	var exitCode *int
	if runErr != nil {
		status = "failed"
		errMsg = runErr.Error()
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) && exitErr.ProcessState != nil {
			code := exitErr.ProcessState.ExitCode()
			exitCode = &code
		}
	} else {
		code := 0
		exitCode = &code
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = rc.TeamClient.FinishTeamTokenExecution(ctx, rc.ExecutionID, teams.TeamTokenExecutionFinishRequest{
		Token:    rc.TeamToken,
		Status:   status,
		ExitCode: exitCode,
		Error:    errMsg,
	})
}

// resolveTokenAndConn translates (target, token) into a fully-populated
// runContext. It walks the same legacy-vs-v2 logic as runExec, looking up the
// device pepper, resolving the token through the vault, and (for v2 tokens)
// unlocking the local DB to fetch host metadata + the encrypted secret.
//
// On success the returned runContext owns an open *db.Store for v2 tokens —
// callers MUST defer rc.DBStore.Close() if it's non-nil.
func resolveTokenAndConn(target, token string) (*runContext, error) {
	return resolveAuthAndConn(authFlags{Target: target, Token: token}, "")
}

func resolveAuthAndConn(af authFlags, auditCommand string) (*runContext, error) {
	if strings.HasPrefix(strings.TrimSpace(af.Token), "stt_") {
		return resolveTeamTokenAndConn(af, auditCommand)
	}
	return resolvePersonalTokenAndConn(af.Target, af.Token)
}

func resolvePersonalTokenAndConn(target, token string) (*runContext, error) {
	vault, err := authtoken.LoadVault()
	if err != nil {
		return nil, fmt.Errorf("failed to load token vault: %w", err)
	}
	pepper, _ := securestore.GetDevicePepper()
	resolved, err := vault.Resolve(token, target, pepper)
	if err != nil {
		return nil, err
	}
	rc := &runContext{
		Vault:    vault,
		TokenIdx: resolved.TokenIndex,
	}

	if resolved.LegacyPayload != nil {
		p := resolved.LegacyPayload
		rc.Conn = ssh.Connection{
			Hostname:            p.Hostname,
			Username:            p.Username,
			Port:                p.Port,
			PasswordBackendUnix: p.PasswordBackendUnix,
			HostKeyPolicy:       p.HostKeyPolicy,
			KeepAliveSeconds:    p.KeepAliveSeconds,
			Term:                p.Term,
		}
		if p.KeyType == "password" {
			rc.Conn.Password = p.Secret
		} else {
			rc.Conn.PrivateKey = p.Secret
		}
		return rc, nil
	}

	dbUnlock := strings.TrimSpace(resolved.DBUnlockSecret)
	if dbUnlock == "" {
		cached, _, ok, _ := unlock.Load()
		if ok {
			dbUnlock = strings.TrimSpace(cached)
		}
	}
	if dbUnlock == "" {
		return nil, fmt.Errorf("token is not active on this device and no unlock session is available")
	}

	store, err := db.Init(dbUnlock)
	if err != nil {
		return nil, fmt.Errorf("failed to unlock database: %w", err)
	}

	host, err := store.GetHostByID(resolved.HostID)
	if err != nil {
		store.Close()
		return nil, fmt.Errorf("failed to load target host: %w", err)
	}
	secret, err := store.GetHostSecret(resolved.HostID)
	if err != nil {
		store.Close()
		return nil, fmt.Errorf("failed to decrypt host secret: %w", err)
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

	rc.Conn = ssh.Connection{
		Hostname:            host.Hostname,
		Username:            host.Username,
		Port:                host.Port,
		PasswordBackendUnix: string(cfg.SSH.PasswordBackendUnix),
		HostKeyPolicy:       string(cfg.SSH.HostKeyPolicy),
		KeepAliveSeconds:    cfg.SSH.KeepAliveSeconds,
		Term:                term,
	}
	if host.KeyType == "password" {
		rc.Conn.Password = secret
	} else {
		rc.Conn.PrivateKey = secret
	}
	rc.DBStore = store
	rc.DBUnlock = dbUnlock
	rc.SessionTTL = time.Duration(cfg.Automation.SessionTTLSeconds) * time.Second
	return rc, nil
}

func resolveTeamTokenAndConn(af authFlags, auditCommand string) (*runContext, error) {
	cfg, cfgErr := config.Load()
	if cfgErr != nil {
		cfg = config.Default()
	}
	client := teamsclient.New(cliTeamsAPIBaseURL(cfg))
	deviceName, _ := os.Hostname()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	resolved, err := client.ResolveTeamToken(ctx, teams.TeamTokenResolveRequest{
		Token:        af.Token,
		TeamID:       af.TeamID,
		Target:       af.Target,
		TargetID:     af.TargetID,
		Command:      auditCommand,
		ClientDevice: deviceName,
	})
	if err != nil {
		return nil, err
	}

	term := ""
	switch cfg.SSH.TermMode {
	case config.TermXterm:
		term = "xterm-256color"
	case config.TermCustom:
		term = strings.TrimSpace(cfg.SSH.TermCustom)
	}

	conn := ssh.Connection{
		Hostname:            resolved.Host.Hostname,
		Username:            resolved.Host.Username,
		Port:                resolved.Host.Port,
		PasswordBackendUnix: string(cfg.SSH.PasswordBackendUnix),
		HostKeyPolicy:       string(cfg.SSH.HostKeyPolicy),
		KeepAliveSeconds:    cfg.SSH.KeepAliveSeconds,
		Term:                term,
	}
	switch resolved.Host.CredentialType {
	case "private_key":
		conn.PrivateKey = resolved.Host.Secret
	case "password":
		conn.Password = resolved.Host.Secret
	}

	return &runContext{
		Conn:        conn,
		TeamClient:  client,
		TeamToken:   af.Token,
		ExecutionID: resolved.ExecutionID,
	}, nil
}

func cliTeamsAPIBaseURL(cfg config.Config) string {
	if value := strings.TrimRight(strings.TrimSpace(cfg.Teams.APIBaseURL), "/"); value != "" {
		return value
	}
	if value := strings.TrimRight(strings.TrimSpace(os.Getenv("SSHTHING_CLOUD_BASE_URL")), "/"); value != "" {
		return value
	}
	return "http://localhost:3000"
}

// authFlags captures the auth-source flags shared by exec / cp / put / get.
type authFlags struct {
	Target   string
	TargetID string
	Team     string
	TeamID   string
	Token    string // resolved (possibly read from file/stdin)
	AuthMode string // "direct" | "file" | "stdin"
}

// extractAuthFlags walks args, extracts -t / --auth* flags, and returns the
// remaining positional args. Callers can chain on additional flag parsing
// (e.g. cp/put/get specifics) via the leftover slice.
//
// authMode resolution (file → read file, stdin → read stdin) happens inline
// so the caller gets back a usable Token regardless of source.
func extractAuthFlags(args []string) (af authFlags, leftover []string, err error) {
	var authFile string
	leftover = make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "-t", "--target":
			i++
			if i >= len(args) {
				return af, nil, fmt.Errorf("missing value for %s", a)
			}
			af.Target = strings.TrimSpace(args[i])
		case "--target-id":
			i++
			if i >= len(args) {
				return af, nil, fmt.Errorf("missing value for %s", a)
			}
			af.TargetID = strings.TrimSpace(args[i])
		case "--team":
			i++
			if i >= len(args) {
				return af, nil, fmt.Errorf("missing value for %s", a)
			}
			af.Team = strings.TrimSpace(args[i])
		case "--team-id":
			i++
			if i >= len(args) {
				return af, nil, fmt.Errorf("missing value for %s", a)
			}
			af.TeamID = strings.TrimSpace(args[i])
		case "--auth":
			if af.AuthMode != "" {
				return af, nil, fmt.Errorf("only one auth source can be provided")
			}
			i++
			if i >= len(args) {
				return af, nil, fmt.Errorf("missing value for --auth")
			}
			af.Token = strings.TrimSpace(args[i])
			af.AuthMode = "direct"
		case "--auth-file":
			if af.AuthMode != "" {
				return af, nil, fmt.Errorf("only one auth source can be provided")
			}
			i++
			if i >= len(args) {
				return af, nil, fmt.Errorf("missing value for --auth-file")
			}
			authFile = strings.TrimSpace(args[i])
			af.AuthMode = "file"
		case "--auth-stdin":
			if af.AuthMode != "" {
				return af, nil, fmt.Errorf("only one auth source can be provided")
			}
			af.AuthMode = "stdin"
		case "--":
			if i+1 < len(args) {
				leftover = append(leftover, args[i+1:]...)
			}
			i = len(args)
		default:
			leftover = append(leftover, a)
		}
	}

	if af.Target == "" && af.TargetID == "" {
		return af, nil, fmt.Errorf("target label or id is required (use -t or --target-id)")
	}
	if af.AuthMode == "" {
		return af, nil, fmt.Errorf("auth token is required (--auth, --auth-file, or --auth-stdin)")
	}

	switch af.AuthMode {
	case "file":
		b, readErr := os.ReadFile(authFile)
		if readErr != nil {
			return af, nil, fmt.Errorf("failed to read auth file: %w", readErr)
		}
		af.Token = strings.TrimSpace(string(b))
	case "stdin":
		b, readErr := io.ReadAll(os.Stdin)
		if readErr != nil {
			return af, nil, fmt.Errorf("failed to read auth from stdin: %w", readErr)
		}
		af.Token = strings.TrimSpace(string(b))
	}

	if af.Token == "" {
		return af, nil, fmt.Errorf("auth token cannot be empty")
	}

	return af, leftover, nil
}

// propagateExitCode mimics runExec's behavior: if the underlying cmd returned
// a non-nil error wrapping an *exec.ExitError, exit the process with the
// remote process's exit code. Otherwise return the error to the caller.
func propagateExitCode(err error) error {
	if err == nil {
		return nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ProcessState != nil {
		os.Exit(exitErr.ProcessState.ExitCode())
	}
	return err
}
