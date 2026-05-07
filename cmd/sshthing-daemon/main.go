// Command sshthing-daemon is the Go sidecar that provides the RPC back-end
// for the SSHThing Electron desktop app.
//
// Wire format: newline-delimited JSON-RPC 2.0 over a Unix socket (macOS/Linux)
// or a Windows named pipe.
//
// Auth: a 32-byte hex token is generated at startup and written to
// ${dataDir}/daemon.token (mode 0600). Every request must carry it as the
// "auth" top-level field.
//
// Single-client assumption: the spike is written for exactly one Electron
// window owning one connection. Notifications broadcast to all connected
// clients, but only one is expected.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Vansh-Raja/SSHThing/internal/cloud"
	"github.com/Vansh-Raja/SSHThing/internal/config"
	"github.com/Vansh-Raja/SSHThing/internal/daemon/listener"
	"github.com/Vansh-Raja/SSHThing/internal/daemon/paths"
	"github.com/Vansh-Raja/SSHThing/internal/daemon/rpc"
	"github.com/Vansh-Raja/SSHThing/internal/daemon/service"
	mountpkg "github.com/Vansh-Raja/SSHThing/internal/mount"
	"github.com/Vansh-Raja/SSHThing/internal/ssh"
	"github.com/Vansh-Raja/SSHThing/internal/teamsclient"
)

func main() {
	// SSH may spawn this binary as an askpass helper (SSH_ASKPASS) when a
	// host uses password auth and sshpass isn't available. Detect that case
	// before doing any daemon setup so the helper exits cleanly after
	// printing the password.
	if ssh.IsAskpassInvocation() {
		if err := ssh.RunAskpassHelper(); err != nil {
			fmt.Fprintf(os.Stderr, "askpass error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Set up structured log output to the daemon log file.
	logPath, err := paths.LogPath()
	if err != nil {
		log.Fatalf("resolve log path: %v", err)
	}
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		log.Fatalf("open log file: %v", err)
	}
	defer logFile.Close()
	// Log to both stderr (visible in dev) and the log file.
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	// Keep logging to stderr for Electron to capture; also tee to file.
	// (Production builds would redirect stderr to /dev/null; log file is the canonical sink.)
	log.SetOutput(os.Stderr)
	_ = logFile // used by future structured logger; spike sends to stderr

	// Generate auth token and write to disk.
	tokenPath, err := paths.TokenPath()
	if err != nil {
		log.Fatalf("resolve token path: %v", err)
	}
	token, err := rpc.GenerateToken()
	if err != nil {
		log.Fatalf("generate token: %v", err)
	}
	if err := rpc.WriteToken(tokenPath, token); err != nil {
		log.Fatalf("write token: %v", err)
	}
	log.Printf("daemon token written to %s", tokenPath)

	// Resolve socket path.
	sockPath, err := paths.SocketPath()
	if err != nil {
		log.Fatalf("resolve socket path: %v", err)
	}

	// Start listening (stale socket is removed inside listener.Listen).
	l, err := listener.Listen(sockPath)
	if err != nil {
		log.Fatalf("listen %s: %v", sockPath, err)
	}
	log.Printf("listening on %s", sockPath)

	// Load user config. Fall back to defaults on error so the daemon still
	// starts in a fresh install.
	cfg, cfgErr := config.Load()
	if cfgErr != nil {
		log.Printf("config load failed (using defaults): %v", cfgErr)
		cfg = config.Default()
	}

	// cfgStore is the shared atomic config holder. All services read from it;
	// settings.set swaps the value atomically so session.open hot-reloads work.
	cfgStore := service.NewCfgStore(cfg)

	// ── Build services ────────────────────────────────────────────────────────
	vault := &service.Vault{}
	hosts := &service.Hosts{Vault: vault} // SyncSvc is wired below after syncSvc is constructed
	groups := &service.Groups{Vault: vault}
	settings := &service.Settings{Store: cfgStore}

	healthSvc := &service.HealthService{
		Vault:    vault,
		CfgStore: cfgStore,
	}

	mountMgr := mountpkg.NewManager()
	mountSvc := &service.MountService{
		Vault:    vault,
		CfgStore: cfgStore,
		Manager:  mountMgr,
	}

	// Create the RPC server.
	srv := rpc.NewServer(token)

	transferSvc := &service.TransferService{
		Vault:    vault,
		CfgStore: cfgStore,
		Notify: func(method string, params any) {
			srv.Notify(method, params)
		},
	}

	tokensSvc := &service.TokensService{
		Vault:    vault,
		CfgStore: cfgStore,
	}

	// Build the teams client.
	// URL resolution: cfg.Teams.APIBaseURL > SSHTHING_CLOUD_BASE_URL >
	// ldflags default > http://localhost:3000. Same logic as the TUI's
	// internal/app/pages.go cloudServiceBaseURL() — see internal/cloud.BaseURL.
	teamsBaseURL := strings.TrimRight(strings.TrimSpace(cfg.Teams.APIBaseURL), "/")
	if teamsBaseURL == "" {
		teamsBaseURL = cloud.BaseURL()
	}
	log.Printf("teams API base URL: %s", teamsBaseURL)
	teamsClient := teamsclient.New(teamsBaseURL)
	teamsSvc := &service.TeamsService{Client: teamsClient, Vault: vault}

	authSvc := &service.AuthService{
		Client: teamsClient,
		Notify: func(method string, params any) {
			srv.Notify(method, params)
		},
	}

	syncSvc := &service.SyncService{
		Vault:    vault,
		CfgStore: cfgStore,
		Client:   teamsClient,
		Notify: func(method string, params any) {
			srv.Notify(method, params)
		},
	}

	// Wire SyncSvc into hosts for auto-sync-after-CRUD support.
	hosts.SyncSvc = syncSvc

	// After vault unlock: restore any mounts that were active in the previous session.
	vault.OnUnlock = func() {
		mountSvc.RestoreFromDB(context.Background())
	}

	// Sessions needs a Notify callback that routes through the server.
	sessions := &service.Sessions{
		Vault:    vault,
		CfgStore: cfgStore,
		Notify: func(method string, params any) {
			srv.Notify(method, params)
		},
	}

	startedAt := time.Now()

	// Handle SIGINT / SIGTERM for graceful shutdown.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ── Register handlers ─────────────────────────────────────────────────────
	rpc.RegisterMeta(srv, sessions, cancel, startedAt)
	rpc.RegisterVault(srv, vault)
	rpc.RegisterHosts(srv, hosts)
	rpc.RegisterGroups(srv, groups)
	rpc.RegisterSession(srv, sessions)
	rpc.RegisterSettings(srv, settings)
	rpc.RegisterHealth(srv, healthSvc)
	rpc.RegisterMount(srv, mountSvc)
	rpc.RegisterTransfer(srv, transferSvc)
	rpc.RegisterTokens(srv, tokensSvc)
	rpc.RegisterTeams(srv, teamsSvc)
	rpc.RegisterAuth(srv, authSvc)
	rpc.RegisterSync(srv, syncSvc)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		log.Printf("received signal %v, shutting down", sig)
		sessions.CloseAll()
		cancel()
		// Remove the Unix socket file on clean shutdown.
		if sockPath != "" {
			_ = os.Remove(sockPath)
		}
	}()

	log.Printf("sshthing-daemon 0.0.1-spike started")
	if err := srv.Serve(ctx, l); err != nil {
		log.Printf("server exited: %v", err)
	}
	log.Printf("daemon stopped")
}
