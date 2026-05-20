package app

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Vansh-Raja/SSHThing/internal/health"
	syncpkg "github.com/Vansh-Raja/SSHThing/internal/sync"
	"github.com/Vansh-Raja/SSHThing/internal/teams"
	"github.com/Vansh-Raja/SSHThing/internal/teamsclient"
	tea "github.com/charmbracelet/bubbletea"
)

// ── Message types ─────────────────────────────────────────────────────

type sshFinishedMsg struct {
	err      error
	hostname string
	proto    string
	keyType  string
}

type mountFinishedMsg struct {
	action string // "mount" | "unmount"
	hostID int
	local  string
	err    error
	stderr string
}

type syncFinishedMsg struct {
	runID  int
	result *syncpkg.SyncResult
}

type syncAnimTickMsg struct {
	runID int
}

type quitFinishedMsg struct{}

type clearErrMsg struct {
	seq int
}

type tickMsg struct{}

type profileAuthPolledMsg struct {
	runID  int
	result teams.CliAuthPollResponse
	err    error
}

// profileClaimFinishedMsg carries the result of a headless paste-back
// claim (the user pasted the browser-shown code into the TUI).
type profileClaimFinishedMsg struct {
	runID  int
	result teams.CliAuthPollResponse
	err    error
}

type healthRefreshStartedMsg struct {
	runID int
	total int
}

type hostHealthResultMsg struct {
	runID     int
	targetKey string
	hostID    int
	result    health.Result
}

type healthRefreshFinishedMsg struct {
	runID int
}

// ── Command constructors ──────────────────────────────────────────────

func tickCmd() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(time.Time) tea.Msg {
		return tickMsg{}
	})
}

func pollProfileAuthCmd(runID int, client *teamsclient.Client, sessionID, pollSecret string, interval time.Duration) tea.Cmd {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	return tea.Tick(interval, func(time.Time) tea.Msg {
		if client == nil {
			return profileAuthPolledMsg{runID: runID, err: fmt.Errorf("teams client is not configured")}
		}
		result, err := client.PollCLIAuth(context.Background(), sessionID, pollSecret)
		return profileAuthPolledMsg{runID: runID, result: result, err: err}
	})
}

// claimProfileAuthCmd runs the headless paste-back exchange: it sends the
// pasted claim code (with the TUI-held pollSecret) and reports the result.
func claimProfileAuthCmd(runID int, client *teamsclient.Client, sessionID, pollSecret, claimCode string) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return profileClaimFinishedMsg{runID: runID, err: fmt.Errorf("teams client is not configured")}
		}
		result, err := client.ClaimCLIAuth(context.Background(), sessionID, pollSecret, claimCode)
		return profileClaimFinishedMsg{runID: runID, result: result, err: err}
	}
}

// copyToTerminalClipboardCmd copies text to the user's *local* terminal
// clipboard via the OSC 52 escape sequence. Unlike a system-clipboard
// library, OSC 52 is interpreted by the terminal emulator itself, so it
// works over SSH on a headless server. Support varies by terminal;
// manual select-copy is the fallback.
func copyToTerminalClipboardCmd(text string) tea.Cmd {
	return func() tea.Msg {
		if strings.TrimSpace(text) == "" {
			return clipboardCopiedMsg{ok: false}
		}
		enc := base64.StdEncoding.EncodeToString([]byte(text))
		// OSC 52: ESC ] 52 ; c ; <base64> BEL
		fmt.Fprintf(os.Stdout, "\x1b]52;c;%s\x07", enc)
		return clipboardCopiedMsg{ok: true}
	}
}

type clipboardCopiedMsg struct{ ok bool }

func runSyncCmd(runID int, mgr *syncpkg.Manager) tea.Cmd {
	return func() tea.Msg {
		if mgr == nil {
			return syncFinishedMsg{runID: runID, result: &syncpkg.SyncResult{Success: false, Message: "sync manager is nil", Timestamp: time.Now()}}
		}
		return syncFinishedMsg{runID: runID, result: mgr.Sync()}
	}
}

func syncAnimTickCmd(runID int) tea.Cmd {
	return tea.Tick(120*time.Millisecond, func(time.Time) tea.Msg {
		return syncAnimTickMsg{runID: runID}
	})
}

