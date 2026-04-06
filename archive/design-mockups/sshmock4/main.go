// sshmock4 — "Monolith"
// Single-column card-based dashboard layout. Each host is a compact
// card showing connection string, status, and tags inline. Warm earth
// tones (amber, copper, sand) with a clean, readable hierarchy.
// No split panels — everything flows vertically like a feed.
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
	bg0 = lipgloss.Color("#1c1917") // stone-950
	bg1 = lipgloss.Color("#292524") // stone-900
	bg2 = lipgloss.Color("#44403c") // stone-700
	bg3 = lipgloss.Color("#57534e") // stone-600

	sand   = lipgloss.Color("#a8a29e") // stone-400
	cream  = lipgloss.Color("#d6d3d1") // stone-300
	bone   = lipgloss.Color("#e7e5e4") // stone-200
	pWhite = lipgloss.Color("#fafaf9") // stone-50

	amber    = lipgloss.Color("#f59e0b")
	copper   = lipgloss.Color("#ea580c")
	rust     = lipgloss.Color("#dc2626")
	jade     = lipgloss.Color("#059669")
	sky      = lipgloss.Color("#0ea5e9")
	indigo   = lipgloss.Color("#6366f1")
	dimAmber = lipgloss.Color("#92400e")
)

type host struct {
	label, group, hostname, user, keyType string
	port                                  int
	tags                                  []string
	status                                int
	lastSSH, latency                      string
}

type grp struct {
	name      string
	collapsed bool
}

func mockGroups() []grp {
	return []grp{{name: "Production"}, {name: "Staging"}, {name: "Homelab"}}
}

func mockHosts() []host {
	return []host{
		{label: "api-gateway", group: "Production", hostname: "api.prod.io", user: "deploy", port: 22, keyType: "ed25519", tags: []string{"api", "nginx", "lb"}, status: 2, lastSSH: "2m ago", latency: "23ms"},
		{label: "db-primary", group: "Production", hostname: "10.0.1.50", user: "root", port: 5432, keyType: "rsa", tags: []string{"postgres", "primary"}, status: 2, lastSSH: "active", latency: "8ms"},
		{label: "redis-node", group: "Production", hostname: "10.0.1.75", user: "admin", port: 6379, keyType: "ecdsa", tags: []string{"redis"}, status: 1, lastSSH: "6h ago", latency: "3ms"},
		{label: "worker-01", group: "Production", hostname: "10.0.1.100", user: "deploy", port: 22, keyType: "ed25519", tags: []string{"worker", "sidekiq"}, status: 0, lastSSH: "3d ago", latency: "—"},
		{label: "staging-app", group: "Staging", hostname: "staging.example.com", user: "dev", port: 2222, keyType: "password", tags: []string{"app"}, status: 1, lastSSH: "1h ago", latency: "45ms"},
		{label: "staging-db", group: "Staging", hostname: "stg-db.local", user: "postgres", port: 5432, keyType: "ecdsa", tags: []string{"db"}, status: 0, lastSSH: "1d ago", latency: "—"},
		{label: "pi-cluster", group: "Homelab", hostname: "192.168.1.200", user: "pi", port: 22, keyType: "rsa", tags: []string{"arm", "k3s"}, status: 0, lastSSH: "2w ago", latency: "1ms"},
		{label: "nas-synology", group: "Homelab", hostname: "192.168.1.50", user: "admin", port: 22, keyType: "ed25519", tags: []string{"storage", "backup"}, status: 0, lastSSH: "5d ago", latency: "2ms"},
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
		c := 0
		for _, h := range hosts {
			if h.group == g.name {
				c++
			}
		}
		items = append(items, listItem{isGroup: true, grp: g, hostCount: c})
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

type model struct {
	items  []listItem
	groups []grp
	hosts  []host
	cursor int
	scroll int
	w, h   int
	view   int
	tick   int
}

func initialModel() model {
	g := mockGroups()
	h := mockHosts()
	return model{groups: g, hosts: h, items: buildList(g, h)}
}

type tickMsg time.Time

func (m model) Init() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
	case tickMsg:
		m.tick++
		return m, tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "j", "down":
			if m.cursor < len(m.items)-1 {
				m.cursor++
				m.adjustScroll()
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
				m.adjustScroll()
			}
		case "enter", " ":
			if m.cursor < len(m.items) && m.items[m.cursor].isGroup {
				n := m.items[m.cursor].grp.name
				for i := range m.groups {
					if m.groups[i].name == n {
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

func (m *model) adjustScroll() {
	// Each group header = 1 line, each host card = 3 lines
	visibleLines := m.h - 6
	linePos := 0
	for i := 0; i < m.cursor; i++ {
		if m.items[i].isGroup {
			linePos += 2
		} else {
			linePos += 4
		}
	}
	if linePos < m.scroll {
		m.scroll = linePos
	}
	if linePos >= m.scroll+visibleLines {
		m.scroll = linePos - visibleLines + 4
	}
}

func statusPill(s int) string {
	switch s {
	case 2:
		return lipgloss.NewStyle().Background(jade).Foreground(pWhite).Bold(true).Padding(0, 1).Render("LIVE")
	case 1:
		return lipgloss.NewStyle().Background(dimAmber).Foreground(amber).Padding(0, 1).Render("IDLE")
	default:
		return lipgloss.NewStyle().Background(bg2).Foreground(sand).Padding(0, 1).Render("OFF")
	}
}

func (m model) View() string {
	if m.w == 0 {
		return ""
	}

	contentW := m.w
	if contentW > 100 {
		contentW = 100
	}

	if m.view == 1 {
		return m.helpView()
	}

	// ── header ──
	title := lipgloss.NewStyle().Foreground(amber).Bold(true).Render("ssh") +
		lipgloss.NewStyle().Foreground(copper).Bold(true).Render("thing")

	live := 0
	for _, h := range m.hosts {
		if h.status == 2 {
			live++
		}
	}
	badge := lipgloss.NewStyle().Foreground(jade).Render(fmt.Sprintf("● %d live", live))
	total := lipgloss.NewStyle().Foreground(sand).Render(fmt.Sprintf("  %d hosts", len(m.hosts)))
	hints := lipgloss.NewStyle().Foreground(bg3).Render("? help  / search  q quit")
	gap := strings.Repeat(" ", max(0, contentW-lipgloss.Width(title)-lipgloss.Width(badge)-lipgloss.Width(total)-lipgloss.Width(hints)-4))
	header := lipgloss.NewStyle().Width(contentW).Padding(0, 2).Background(bg1).
		Render(title + "  " + badge + total + gap + hints)

	sep := lipgloss.NewStyle().Foreground(bg2).Render(strings.Repeat("─", contentW))

	// ── card list ──
	var cards []string
	for i, item := range m.items {
		sel := i == m.cursor
		if item.isGroup {
			arrow := "▾"
			if item.grp.collapsed {
				arrow = "▸"
			}
			arC := lipgloss.NewStyle().Foreground(sand)
			nameC := lipgloss.NewStyle().Foreground(cream).Bold(true)
			countC := lipgloss.NewStyle().Foreground(bg3)
			if sel {
				arC = arC.Foreground(amber)
				nameC = nameC.Foreground(amber)
			}
			cards = append(cards, "")
			cards = append(cards, fmt.Sprintf("  %s %s %s",
				arC.Render(arrow), nameC.Render(item.grp.name), countC.Render(fmt.Sprintf("(%d)", item.hostCount))))
		} else {
			cards = append(cards, m.renderCard(item.host, sel, contentW-4))
		}
	}

	cardContent := strings.Join(cards, "\n")
	// Simple scroll: split by lines
	allLines := strings.Split(cardContent, "\n")
	visibleH := m.h - 5
	start := m.scroll
	if start > len(allLines) {
		start = 0
	}
	end := start + visibleH
	if end > len(allLines) {
		end = len(allLines)
	}
	visible := strings.Join(allLines[start:end], "\n")

	body := lipgloss.NewStyle().Width(contentW).Height(visibleH).Render(visible)

	// ── footer ──
	footerParts := []string{
		lipgloss.NewStyle().Foreground(cream).Render("↑↓") + lipgloss.NewStyle().Foreground(sand).Render(" nav"),
		lipgloss.NewStyle().Foreground(cream).Render("⏎") + lipgloss.NewStyle().Foreground(sand).Render(" ssh"),
		lipgloss.NewStyle().Foreground(cream).Render("s") + lipgloss.NewStyle().Foreground(sand).Render(" sftp"),
		lipgloss.NewStyle().Foreground(cream).Render("a") + lipgloss.NewStyle().Foreground(sand).Render(" add"),
		lipgloss.NewStyle().Foreground(cream).Render("e") + lipgloss.NewStyle().Foreground(sand).Render(" edit"),
		lipgloss.NewStyle().Foreground(cream).Render("d") + lipgloss.NewStyle().Foreground(sand).Render(" del"),
	}
	footer := lipgloss.NewStyle().Width(contentW).Padding(0, 2).Background(bg1).
		Render(strings.Join(footerParts, "   "))

	full := header + "\n" + sep + "\n" + body + "\n" + sep + "\n" + footer

	// center horizontally if terminal is wider than content
	return lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Top, full,
		lipgloss.WithWhitespaceBackground(bg0))
}

func (m model) renderCard(h host, sel bool, maxW int) string {
	borderColor := bg2
	if sel {
		borderColor = amber
	}

	// ── line 1: label + status pill + latency ──
	nameStyle := lipgloss.NewStyle().Foreground(bone).Bold(true)
	if sel {
		nameStyle = nameStyle.Foreground(amber)
	}
	pill := statusPill(h.status)
	lat := lipgloss.NewStyle().Foreground(sand).Render(h.latency)
	l1 := nameStyle.Render(h.label) + "  " + pill + "  " + lat

	// ── line 2: connection string + auth ──
	connStr := fmt.Sprintf("%s@%s:%d", h.user, h.hostname, h.port)
	connR := lipgloss.NewStyle().Foreground(sky).Render(connStr)
	authR := lipgloss.NewStyle().Foreground(sand).Render("[" + h.keyType + "]")
	lastR := lipgloss.NewStyle().Foreground(bg3).Render(h.lastSSH)
	l2 := connR + "  " + authR + "  " + lastR

	// ── line 3: tags ──
	tagStr := ""
	for _, t := range h.tags {
		tagStr += lipgloss.NewStyle().Foreground(indigo).Render("#"+t) + " "
	}
	if tagStr == "" {
		tagStr = lipgloss.NewStyle().Foreground(bg3).Render("no tags")
	}

	cardW := maxW
	if cardW > 90 {
		cardW = 90
	}

	card := lipgloss.NewStyle().
		Width(cardW).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 2).
		Render(l1 + "\n" + l2 + "\n" + tagStr)

	return card
}

func (m model) helpView() string {
	title := lipgloss.NewStyle().Foreground(amber).Bold(true).Render("Keyboard Shortcuts")
	pairs := [][2]string{
		{"↑/↓ j/k", "Navigate"},
		{"enter", "Connect / Toggle"},
		{"s", "SFTP"},
		{"a", "Add host"},
		{"e", "Edit"},
		{"d", "Delete"},
		{"/", "Search"},
		{",", "Settings"},
		{"?", "Help"},
		{"q", "Quit"},
	}
	kS := lipgloss.NewStyle().Foreground(amber).Width(16).Align(lipgloss.Right)
	vS := lipgloss.NewStyle().Foreground(cream)
	var lines []string
	for _, p := range pairs {
		lines = append(lines, kS.Render(p[0])+"  "+vS.Render(p[1]))
	}
	content := title + "\n\n" + strings.Join(lines, "\n") + "\n\n" +
		lipgloss.NewStyle().Foreground(sand).Render("? or esc to close")

	box := lipgloss.NewStyle().
		Width(50).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(bg2).
		Padding(1, 3).
		Render(content)

	return lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, box,
		lipgloss.WithWhitespaceBackground(bg0))
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
