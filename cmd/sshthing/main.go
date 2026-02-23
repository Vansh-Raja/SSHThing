package main

import (
	"fmt"
	"os"

	"github.com/Vansh-Raja/SSHThing/internal/app"
	"github.com/Vansh-Raja/SSHThing/internal/ssh"
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
			fmt.Println("  sshthing --version  Print version")
			fmt.Println("  sshthing --help     Show this help")
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
