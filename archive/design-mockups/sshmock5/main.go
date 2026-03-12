// sshmock5 — "Terminal"
// Retro green-on-black hacker aesthetic. Table-based layout showing
// all hosts in a compact, data-dense table with inline status
// indicators. No panels — pure information. Inspired by htop and
// classic terminal dashboards. Monospace-optimized.
package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── palette ────────────────────────────────────────────────────────
var (
	termBlack   = lipgloss.Color("#0a0a0a")
	termDark    = lipgloss.Color("#0d1a0d")
	termDim     = lipgloss.Color("#1a331a")
	termMid     = lipgloss.Color("#2d5a2d")
	termGreen   = lipgloss.Color("#33ff33")
	termBright  = lipgloss.Color("#66ff66")
	termDull    = lipgloss.Color("#339933")
	termFade    = lipgloss.Color("#226622")
	termWhite   = lipgloss.Color("#ccffcc")

	termRed     = lipgloss.Color("#ff3333")
	termYellow  = lipgloss.Color("#ffff33")
	termCyan    = lipgloss.Color("#33ffff")
	termAmber   = lipgloss.Color("#ffaa00")
)

type host struct {
	label, group, hostname, user, keyType string
	port                                  int
	tags                                  []string
	status                                int
	lastSSH, latency, pid                 string
}

func mockHosts() []host {
	return []host{
		{label: "api-gateway", group: "prod", hostname: "api.prod.io", user: "deploy", port: 22, keyType: "ed25519", tags: []string{"api"}, status: 2, lastSSH: "2m", latency: "23ms", pid: "14523"},
		{label: "db-primary", group: "prod", hostname: "10.0.1.50", user: "root", port: 5432, keyType: "rsa", tags: []string{"pg"}, status: 2, lastSSH: "act", latency: "8ms", pid: "14601"},
		{label: "redis-01", group: "prod", hostname: "10.0.1.75", user: "admin", port: 6379, keyType: "ecdsa", tags: []string{"redis"}, status: 1, lastSSH: "6h", latency: "3ms", pid: "—"},
		{label: "worker-pool", group: "prod", hostname: "10.0.1.100", user: "deploy", port: 22, keyType: "ed25519", tags: []string{"worker"}, status: 0, lastSSH: "3d", latency: "—", pid: "—"},
		{label: "lb-nginx", group: "prod", hostname: "10.0.1.10", user: "root", port: 22, keyType: "ed25519", tags: []string{"lb"}, status: 2, lastSSH: "act", latency: "2ms", pid: "14590"},
		{label: "monitor", group: "prod", hostname: "10.0.1.200", user: "grafana", port: 22, keyType: "password", tags: []string{"mon"}, status: 1, lastSSH: "12h", latency: "15ms", pid: "—"},
		{label: "stg-app", group: "stg", hostname: "stg.example.com", user: "dev", port: 2222, keyType: "password", tags: []string{"app"}, status: 1, lastSSH: "1h", latency: "45ms", pid: "—"},
		{label: "stg-db", group: "stg", hostname: "stg-db.local", user: "postgres", port: 5432, keyType: "ecdsa", tags: []string{"db"}, status: 0, lastSSH: "1d", latency: "—", pid: "—"},
		{label: "dev-vm", group: "dev", hostname: "192.168.1.100", user: "vansh", port: 22, keyType: "ed25519", tags: []string{"local"}, status: 0, lastSSH: "now", latency: "1ms", pid: "—"},
		{label: "pi-k3s", group: "dev", hostname: "192.168.1.200", user: "pi", port: 22, keyType: "rsa", tags: []string{"arm"}, status: 0, lastSSH: "2w", latency: "—", pid: "—"},
		{label: "nas-box", group: "dev", hostname: "192.168.1.50", user: "admin", port: 22, keyType: "ed25519", tags: []string{"nas"}, status: 0, lastSSH: "5d", latency: "2ms", pid: "—"},
	}
}

type model struct {
	hosts  []host
	cursor int
	w, h   int
	view   int
	tick   int
	sortBy int // 0=name, 1=group, 2=status
}

func initialModel() model {
	return model{hosts: mockHosts()}
}

type tickMsg time.Time

func (m model) Init() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
	case tickMsg:
		m.tick++
		return m, tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })
	case tea.KeyMsg:
		if m.view == 1 {
			m.view = 0
			return m, nil
		}
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "j", "down":
			if m.cursor < len(m.hosts)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "?":
			m.view = 1
		case "tab":
			m.sortBy = (m.sortBy + 1) % 3
		}
	}
	return m, nil
}

func padRight(s string, w int) string {
	if len(s) >= w {
		return s[:w]
	}
	return s + strings.Repeat(" ", w-len(s))
}

func (m model) View() string {
	if m.w == 0 {
		return ""
	}
	w := m.w
	if w > 160 {
		w = 160
	}

	if m.view == 1 {
		return m.helpView()
	}

	blink := m.tick%2 == 0

	// ── top system bar ──
	now := time.Now().Format("15:04:05")
	live := 0
	for _, h := range m.hosts {
		if h.status == 2 {
			live++
		}
	}
	sysLeft := lipgloss.NewStyle().Foreground(termGreen).Bold(true).Render("sshthing")
	sysInfo := lipgloss.NewStyle().Foreground(termDull).Render(
		fmt.Sprintf("  hosts: %d  live: %d  groups: 3  ", len(m.hosts), live))
	sysClock := lipgloss.NewStyle().Foreground(termFade).Render(now)
	sysHint := lipgloss.NewStyle().Foreground(termFade).Render("  F1:help  TAB:sort  q:quit")
	gap := strings.Repeat(" ", max(0, w-lipgloss.Width(sysLeft)-lipgloss.Width(sysInfo)-lipgloss.Width(sysClock)-lipgloss.Width(sysHint)))
	topBar := lipgloss.NewStyle().Background(termDim).Width(w).
		Render(sysLeft + sysInfo + gap + sysClock + sysHint)

	// ── column headers ──
	colST := 4
	colName := 18
	colGroup := 8
	colConn := 30
	remaining := w - colST - colName - colGroup - colConn - 20
	if remaining < 0 {
		remaining = 0
	}
	colAuth := 10
	colLat := 8
	colLast := 6
	colPID := 7

	hdrStyle := lipgloss.NewStyle().Foreground(termBright).Bold(true).Background(termDark)
	sortMark := func(col int) string {
		if m.sortBy == col {
			return "▼"
		}
		return ""
	}
	headers := hdrStyle.Width(w).Render(
		padRight("ST", colST) +
			padRight("NAME"+sortMark(0), colName) +
			padRight("GRP"+sortMark(1), colGroup) +
			padRight("CONNECTION", colConn) +
			padRight("AUTH", colAuth) +
			padRight("LAT", colLat) +
			padRight("LAST", colLast) +
			padRight("PID", colPID))

	hSep := lipgloss.NewStyle().Foreground(termDim).Render(strings.Repeat("─", w))

	// ── rows ──
	bodyH := m.h - 6
	var rows []string
	for i, h := range m.hosts {
		sel := i == m.cursor

		// status indicator
		var st string
		switch h.status {
		case 2:
			st = lipgloss.NewStyle().Foreground(termGreen).Render("●")
			if blink {
				st = lipgloss.NewStyle().Foreground(termBright).Render("●")
			}
		case 1:
			st = lipgloss.NewStyle().Foreground(termYellow).Render("◐")
		default:
			st = lipgloss.NewStyle().Foreground(termFade).Render("○")
		}

		nameS := lipgloss.NewStyle().Foreground(termWhite)
		grpS := lipgloss.NewStyle().Foreground(termDull)
		connS := lipgloss.NewStyle().Foreground(termCyan)
		authS := lipgloss.NewStyle().Foreground(termDull)
		latS := lipgloss.NewStyle().Foreground(termDull)
		lastS := lipgloss.NewStyle().Foreground(termFade)
		pidS := lipgloss.NewStyle().Foreground(termFade)

		conn := fmt.Sprintf("%s@%s:%d", h.user, h.hostname, h.port)
		if len(conn) > colConn-1 {
			conn = conn[:colConn-2] + "…"
		}

		row := padRight(" "+st+" ", colST) +
			nameS.Render(padRight(h.label, colName)) +
			grpS.Render(padRight(h.group, colGroup)) +
			connS.Render(padRight(conn, colConn)) +
			authS.Render(padRight(h.keyType, colAuth)) +
			latS.Render(padRight(h.latency, colLat)) +
			lastS.Render(padRight(h.lastSSH, colLast)) +
			pidS.Render(padRight(h.pid, colPID))

		if sel {
			row = lipgloss.NewStyle().Background(termDim).Foreground(termBright).Bold(true).Width(w).
				Render(padRight(" ▸ ", colST) +
					padRight(h.label, colName) +
					padRight(h.group, colGroup) +
					padRight(conn, colConn) +
					padRight(h.keyType, colAuth) +
					padRight(h.latency, colLat) +
					padRight(h.lastSSH, colLast) +
					padRight(h.pid, colPID))
		}

		rows = append(rows, row)
	}

	for len(rows) < bodyH {
		rows = append(rows, "")
	}
	if len(rows) > bodyH {
		rows = rows[:bodyH]
	}

	// ── detail bar (bottom) ──
	var detailBar string
	if m.cursor < len(m.hosts) {
		h := m.hosts[m.cursor]
		tagStr := ""
		for _, t := range h.tags {
			tagStr += lipgloss.NewStyle().Foreground(termGreen).Render("#"+t) + " "
		}
		detailBar = lipgloss.NewStyle().Background(termDark).Width(w).Foreground(termDull).
			Render(fmt.Sprintf(" %s  %s@%s:%d  auth:%s  %s  %s",
				lipgloss.NewStyle().Foreground(termBright).Bold(true).Render(h.label),
				h.user, h.hostname, h.port, h.keyType, tagStr,
				lipgloss.NewStyle().Foreground(termFade).Render("⏎:ssh  s:sftp  a:add  e:edit  d:del")))
	}

	// ── bottom keys ──
	bottomBar := lipgloss.NewStyle().Background(termDim).Width(w).Foreground(termFade).
		Render(" ↑↓:nav  ⏎:connect  s:sftp  a:add  e:edit  d:del  /:search  ,:settings  ?:help  q:quit")

	out := topBar + "\n" + headers + "\n" + hSep + "\n" +
		strings.Join(rows, "\n") + "\n" + hSep + "\n" + detailBar + "\n" + bottomBar

	return lipgloss.NewStyle().Background(termBlack).Width(m.w).Height(m.h).Render(out)
}

func (m model) helpView() string {
	title := lipgloss.NewStyle().Foreground(termGreen).Bold(true).Render("=== HELP ===")
	pairs := [][2]string{
		{"↑/↓/j/k", "navigate"},
		{"enter", "ssh connect"},
		{"s", "sftp connect"},
		{"a", "add host"},
		{"e", "edit host"},
		{"d", "delete host"},
		{"/", "search filter"},
		{"tab", "cycle sort column"},
		{",", "settings"},
		{"?", "this help"},
		{"q", "quit"},
	}
	kS := lipgloss.NewStyle().Foreground(termBright).Width(14).Align(lipgloss.Right)
	vS := lipgloss.NewStyle().Foreground(termDull)
	var lines []string
	for _, p := range pairs {
		lines = append(lines, kS.Render(p[0])+"  "+vS.Render(p[1]))
	}
	content := title + "\n\n" + strings.Join(lines, "\n") + "\n\n" +
		lipgloss.NewStyle().Foreground(termFade).Render("any key to close")

	box := lipgloss.NewStyle().
		Width(46).
		Border(lipgloss.NormalBorder()).
		BorderForeground(termDim).
		Padding(1, 2).
		Render(content)

	return lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, box,
		lipgloss.WithWhitespaceBackground(termBlack))
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
