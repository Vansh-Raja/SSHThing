package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Vansh-Raja/SSHThing/internal/app"
	"github.com/Vansh-Raja/SSHThing/internal/config"
	"github.com/Vansh-Raja/SSHThing/internal/sync"
	tea "github.com/charmbracelet/bubbletea"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "version":
			fmt.Printf("sshthing-test %s\n", version)
			return
		case "--help", "-h", "help":
			fmt.Println("sshthing-test — SSHThing test binary (isolated DB path)")
			fmt.Println()
			fmt.Println("Usage:")
			fmt.Println("  sshthing-test                 Run using ~/.sshthing-test")
			fmt.Println("  sshthing-test --db-path PATH  Run using an explicit DB path")
			fmt.Println("  sshthing-test --data-dir DIR  Run using an explicit data dir")
			fmt.Println("  sshthing-test --version       Print version")
			fmt.Println("  sshthing-test debug-sync      Debug sync initialization")
			return
		case "debug-sync":
			debugSync()
			return
		}
	}

	dbPath := flag.String("db-path", "", "Override DB path (testing only)")
	dataDir := flag.String("data-dir", "", "Override data dir (testing only). DB stored at <dir>/hosts.db")
	flag.Parse()

	switch {
	case *dbPath != "":
		_ = os.Setenv("SSHTHING_DB_PATH", *dbPath)
	case *dataDir != "":
		_ = os.Setenv("SSHTHING_DATA_DIR", *dataDir)
	default:
		home, err := os.UserHomeDir()
		if err == nil {
			_ = os.Setenv("SSHTHING_DATA_DIR", filepath.Join(home, ".sshthing-test"))
		}
	}

	m := app.NewModel()
	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}
}

func debugSync() {
	fmt.Println("=== SSHThing Sync Debug ===")
	fmt.Println()

	// Load config
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("❌ Config load error: %v\n", err)
		return
	}

	fmt.Printf("Sync enabled: %v\n", cfg.Sync.Enabled)
	fmt.Printf("Repo URL: %s\n", cfg.Sync.RepoURL)
	fmt.Printf("SSH Key: %s\n", cfg.Sync.SSHKeyPath)
	fmt.Printf("Branch: %s\n", cfg.Sync.Branch)

	syncPath, err := cfg.SyncPath()
	if err != nil {
		fmt.Printf("❌ Sync path error: %v\n", err)
		return
	}
	fmt.Printf("Sync path: %s\n", syncPath)
	fmt.Println()

	if !cfg.Sync.Enabled {
		fmt.Println("❌ Sync is disabled in config")
		return
	}

	// Check SSH key
	if cfg.Sync.SSHKeyPath != "" {
		if _, err := os.Stat(cfg.Sync.SSHKeyPath); err != nil {
			fmt.Printf("❌ SSH key not found: %s\n", cfg.Sync.SSHKeyPath)
			return
		}
		fmt.Printf("✓ SSH key exists: %s\n", cfg.Sync.SSHKeyPath)
	}

	// Try git init
	fmt.Println()
	fmt.Println("Initializing git...")
	git := sync.NewGitManager(syncPath, cfg.Sync.RepoURL, cfg.Sync.Branch, cfg.Sync.SSHKeyPath)

	err = git.Init()
	if err != nil {
		fmt.Printf("❌ Git init error: %v\n", err)
		return
	}
	fmt.Println("✓ Git initialized")

	// Check sync file
	syncFile := git.GetSyncFilePath()
	fmt.Printf("Sync file: %s\n", syncFile)
	if _, err := os.Stat(syncFile); err == nil {
		fmt.Println("✓ Sync file exists")
	} else {
		fmt.Println("⚠ Sync file does not exist (will be created)")
	}

	// Try push
	fmt.Println()
	fmt.Println("Attempting push...")
	if err := git.Push(); err != nil {
		fmt.Printf("❌ Push error: %v\n", err)
		return
	}
	fmt.Println("✓ Push successful!")
}
