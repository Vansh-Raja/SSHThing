// sshmock2 — "Brutalist"
// Raw, high-contrast design using block characters, ASCII borders,
// and a bold status-light system. Inspired by brutalist web design
// and retro terminal aesthetics. Heavy use of full-width block
// backgrounds for selection, no rounded corners.
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
	black     = lipgloss.Color("#000000")
	darkGray  = lipgloss.Color("#111111")
	midGray   = lipgloss.Color("#333333")
	gray      = lipgloss.Color("#666666")
	lightGray = lipgloss.Color("#999999")
	offWhite  = lipgloss.Color("#cccccc")
	white     = lipgloss.Color("#ffffff")

	hotPink  = lipgloss.Color("#ff2d6f")
	neonCyan = lipgloss.Color("#00fff5")
	lime     = lipgloss.Color("#b8ff00")
	amber    = lipgloss.Color("#ffaa00")
	errRed   = lipgloss.Color("#ff0033")
)

// ── mock data ──────────────────────────────────────────────────────
type host struct {
	label, group, hostname, user, keyType string
	port                                  int
	tags                                  []string
	status                                int // 0=offline, 1=idle, 2=connected
	lastSSH                               string
}

type grp struct {
	name      string
	collapsed bool
}

func mockGroups() []grp {
	return []grp{
		{name: "PRODUCTION"},
		{name: "STAGING"},
		{name: "LAB", collapsed: true},
	}
}

func mockHosts() []host {
	return []host{
		{label: "api-gateway", group: "PRODUCTION", hostname: "api.prod.io", user: "deploy", port: 22, keyType: "ed25519", tags: []string{"api", "lb"}, status: 2, lastSSH: "active"},
		{label: "db-primary", group: "PRODUCTION", hostname: "10.0.1.50", user: "root", port: 5432, keyType: "rsa", tags: []string{"pg", "primary"}, status: 2, lastSSH: "active"},
		{label: "worker-pool", group: "PRODUCTION", hostname: "10.0.1.100-110", user: "deploy", port: 22, keyType: "ed25519", tags: []string{"worker"}, status: 1, lastSSH: "3h"},
		{label: "cache-01", group: "PRODUCTION", hostname: "10.0.1.75", user: "admin", port: 22, keyType: "ecdsa", tags: []string{"redis"}, status: 0, lastSSH: "12h"},
		{label: "app-staging", group: "STAGING", hostname: "stg.example.com", user: "dev", port: 2222, keyType: "password", tags: []string{"app"}, status: 1, lastSSH: "5h"},
		{label: "db-staging", group: "STAGING", hostname: "stg-db.local", user: "postgres", port: 5432, keyType: "ecdsa", tags: []string{"db"}, status: 0, lastSSH: "1d"},
		{label: "rpi-cluster", group: "LAB", hostname: "192.168.1.200", user: "pi", port: 22, keyType: "rsa", tags: []string{"arm", "k3s"}, status: 0, lastSSH: "2w"},
		{label: "dev-vm", group: "LAB", hostname: "192.168.1.100", user: "vansh", port: 22, keyType: "ed25519", tags: []string{"local"}, status: 0, lastSSH: "now"},
	}
}

type listItem struct {
	isGroup   bool
	grp       grp
	host      host
	hostCount int
}

func buildList(groups []grp, hosts []host) []listItem {
	var items []listItem
	for _, g := range groups {
		count := 0
		for _, h := range hosts {
			if h.group == g.name {
				count++
			}
		}
		items = append(items, listItem{isGroup: true, grp: g, hostCount: count})
		if !g.collapsed {
			for _, h := range hosts {
				if h.group == g.name {
					items = append(items, listItem{host: h})
				}
			}
		}
	}
	return items
}

// ── model ──────────────────────────────────────────────────────────
type model struct {
	items  []listItem
	groups []grp
	hosts  []host
	cursor int
	w, h   int
	view   int // 0=list, 1=help
	tick   int
}

func initialModel() model {
	g := mockGroups()
	h := mockHosts()
	return model{groups: g, hosts: h, items: buildList(g, h)}
}

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m model) Init() tea.Cmd { return tickCmd() }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
	case tickMsg:
		m.tick++
		return m, tickCmd()
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "j", "down":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "enter", " ":
			if m.cursor < len(m.items) && m.items[m.cursor].isGroup {
				name := m.items[m.cursor].grp.name
				for i := range m.groups {
					if m.groups[i].name == name {
						m.groups[i].collapsed = !m.groups[i].collapsed
					}
				}
				m.items = buildList(m.groups, m.hosts)
				if m.cursor >= len(m.items) {
					m.cursor = len(m.items) - 1
				}
			}
		case "?":
			m.view = 1 - m.view
		}
	}
	return m, nil
}

// ── rendering ──────────────────────────────────────────────────────

func statusBlock(s int, blink bool) string {
	switch s {
	case 2:
		return lipgloss.NewStyle().Foreground(lime).Render("█")
	case 1:
		if blink {
			return lipgloss.NewStyle().Foreground(amber).Render("█")
		}
		return lipgloss.NewStyle().Foreground(amber).Render("▄")
	default:
		return lipgloss.NewStyle().Foreground(midGray).Render("▪")
	}
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
		return m.renderHelp(w)
	}

	// ── top bar ──
	barBg := lipgloss.NewStyle().Background(hotPink).Foreground(black).Bold(true).Padding(0, 1)
	barDim := lipgloss.NewStyle().Background(darkGray).Foreground(lightGray).Padding(0, 1)

	title := barBg.Render(" SSHTHING ")
	connCount := 0
	for _, h := range m.hosts {
		if h.status == 2 {
			connCount++
		}
	}
	stats := barDim.Width(w - lipgloss.Width(title)).Render(
		fmt.Sprintf(" %d hosts  %d connected  %d groups", len(m.hosts), connCount, len(m.groups)))
	topBar := title + stats

	// ── divider ──
	div := lipgloss.NewStyle().Foreground(midGray).Render(strings.Repeat("━", w))

	// ── layout ──
	listW := w * 35 / 100
	if listW < 30 {
		listW = 30
	}
	detailW := w - listW - 1
	bodyH := m.h - 4

	blink := m.tick%2 == 0

	// ── list ──
	var listLines []string
	for i, item := range m.items {
		sel := i == m.cursor
		if item.isGroup {
			arrow := "▼"
			if item.grp.collapsed {
				arrow = "▶"
			}
			countStr := fmt.Sprintf("[%d]", item.hostCount)
			if sel {
				line := lipgloss.NewStyle().Background(hotPink).Foreground(black).Bold(true).
					Width(listW).Render(fmt.Sprintf(" %s %s %s", arrow, item.grp.name, countStr))
				listLines = append(listLines, line)
			} else {
				ar := lipgloss.NewStyle().Foreground(gray).Render(arrow)
				nm := lipgloss.NewStyle().Foreground(offWhite).Bold(true).Render(item.grp.name)
				ct := lipgloss.NewStyle().Foreground(gray).Render(countStr)
				listLines = append(listLines, fmt.Sprintf("  %s %s %s", ar, nm, ct))
			}
		} else {
			h := item.host
			sb := statusBlock(h.status, blink)
			lbl := h.label
			maxL := listW - 12
			if maxL > 0 && len(lbl) > maxL {
				lbl = lbl[:maxL-1] + "…"
			}
			portStr := lipgloss.NewStyle().Foreground(gray).Render(fmt.Sprintf(":%d", h.port))
			if sel {
				line := lipgloss.NewStyle().Background(neonCyan).Foreground(black).Bold(true).
					Width(listW).Render(fmt.Sprintf("   %s %s%s", "█", lbl, portStr))
				listLines = append(listLines, line)
			} else {
				nm := lipgloss.NewStyle().Foreground(offWhite).Render(lbl)
				listLines = append(listLines, fmt.Sprintf("   %s %s%s", sb, nm, portStr))
			}
		}
	}
	for len(listLines) < bodyH {
		listLines = append(listLines, "")
	}
	if len(listLines) > bodyH {
		listLines = listLines[:bodyH]
	}
	listCol := strings.Join(listLines, "\n")

	// ── vertical divider ──
	vDiv := lipgloss.NewStyle().Foreground(midGray).Render(
		strings.Repeat("┃\n", bodyH))
	vDiv = strings.TrimSuffix(vDiv, "\n")

	// ── detail ──
	detailContent := m.renderDetail(detailW-4, bodyH, blink)
	detailCol := lipgloss.NewStyle().Width(detailW).Padding(0, 2).Render(detailContent)

	body := lipgloss.JoinHorizontal(lipgloss.Top, listCol, vDiv, detailCol)

	// ── bottom bar ──
	bottomLeft := lipgloss.NewStyle().Background(darkGray).Foreground(gray).Padding(0, 1).
		Render("↑↓ nav  ⏎ connect  s sftp  a add  e edit  d del  ? help  q quit")
	bottomBar := lipgloss.NewStyle().Width(w).Background(darkGray).Render(bottomLeft)

	return lipgloss.NewStyle().Background(black).Width(m.w).Height(m.h).Render(
		topBar + "\n" + div + "\n" + body + "\n" + div + "\n" + bottomBar)
}

func (m model) renderDetail(w, h int, blink bool) string {
	if m.cursor >= len(m.items) {
		return ""
	}
	item := m.items[m.cursor]

	if item.isGroup {
		title := lipgloss.NewStyle().Foreground(hotPink).Bold(true).Render("┃ " + item.grp.name)
		info := lipgloss.NewStyle().Foreground(lightGray).Render(fmt.Sprintf("  %d servers", item.hostCount))
		sep := lipgloss.NewStyle().Foreground(midGray).Render(strings.Repeat("━", min(w, 40)))
		hint := lipgloss.NewStyle().Foreground(gray).Render("ENTER toggle  A add  E rename  D delete")
		return title + info + "\n" + sep + "\n\n" + hint
	}

	ho := item.host
	keyStyle := lipgloss.NewStyle().Foreground(gray).Width(16).Align(lipgloss.Right)
	valStyle := lipgloss.NewStyle().Foreground(offWhite)
	dimStyle := lipgloss.NewStyle().Foreground(lightGray)

	statusStr := statusBlock(ho.status, blink) + " "
	switch ho.status {
	case 2:
		statusStr += lipgloss.NewStyle().Foreground(lime).Bold(true).Render("CONNECTED")
	case 1:
		statusStr += lipgloss.NewStyle().Foreground(amber).Render("IDLE")
	default:
		statusStr += lipgloss.NewStyle().Foreground(gray).Render("OFFLINE")
	}

	connBox := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(midGray).
		Padding(0, 2).
		Render(fmt.Sprintf("%s@%s:%d", ho.user, ho.hostname, ho.port))

	tagStr := ""
	for _, t := range ho.tags {
		tagStr += lipgloss.NewStyle().Background(midGray).Foreground(offWhite).Padding(0, 1).Render(t) + " "
	}
	if tagStr == "" {
		tagStr = dimStyle.Render("—")
	}

	title := lipgloss.NewStyle().Foreground(neonCyan).Bold(true).Render(ho.label)
	sep := lipgloss.NewStyle().Foreground(midGray).Render(strings.Repeat("━", min(w, 50)))

	lines := []string{
		title,
		statusStr,
		"",
		connBox,
		"",
		keyStyle.Render("AUTH") + "  " + valStyle.Render(strings.ToUpper(ho.keyType)),
		keyStyle.Render("GROUP") + "  " + dimStyle.Render(ho.group),
		keyStyle.Render("LAST") + "  " + dimStyle.Render(ho.lastSSH),
		keyStyle.Render("TAGS") + "  " + tagStr,
		"",
		sep,
		"",
		lipgloss.NewStyle().Foreground(gray).Render("ENTER connect  S sftp  E edit  D delete"),
	}

	return strings.Join(lines, "\n")
}

func (m model) renderHelp(w int) string {
	title := lipgloss.NewStyle().Background(hotPink).Foreground(black).Bold(true).Padding(0, 2).
		Render(" KEYBOARD SHORTCUTS ")

	pairs := [][2]string{
		{"↑ / ↓ / j / k", "Navigate list"},
		{"ENTER", "Connect SSH / Toggle group"},
		{"S then ENTER", "Connect SFTP"},
		{"A", "Add new host"},
		{"E", "Edit selected"},
		{"D", "Delete selected"},
		{"/", "Search"},
		{",", "Settings"},
		{"?", "Toggle help"},
		{"Q", "Quit"},
	}

	kStyle := lipgloss.NewStyle().Foreground(neonCyan).Bold(true).Width(20).Align(lipgloss.Right)
	vStyle := lipgloss.NewStyle().Foreground(offWhite)
	var lines []string
	for _, p := range pairs {
		lines = append(lines, kStyle.Render(p[0])+"  "+vStyle.Render(p[1]))
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(midGray).
		Padding(1, 3).
		Render(title + "\n\n" + strings.Join(lines, "\n") + "\n\n" +
			lipgloss.NewStyle().Foreground(gray).Render("? or ESC to close"))

	return lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, box,
		lipgloss.WithWhitespaceBackground(black))
}

func min(a, b int) int {
	if a < b {
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
