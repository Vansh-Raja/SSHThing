// sshmock3 — "Vapor"
// Synthwave/cyberpunk aesthetic with purple-to-pink gradients, glowing
// borders, and a three-panel layout (sidebar + detail + quick-stats).
// Neon accents on a deep purple-black background. The list uses
// gradient coloring on selected items.
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
	voidBlack  = lipgloss.Color("#0d001a")
	deepPurple = lipgloss.Color("#1a0033")
	darkViolet = lipgloss.Color("#2d004d")
	midPurple  = lipgloss.Color("#4a0080")
	purple     = lipgloss.Color("#7b2fbe")
	violet     = lipgloss.Color("#9b59d0")
	lavender   = lipgloss.Color("#c39bd3")
	softWhite  = lipgloss.Color("#e8d5f5")

	neonPink   = lipgloss.Color("#ff2d95")
	hotMagenta = lipgloss.Color("#ff00ff")
	neonBlue   = lipgloss.Color("#00bbff")
	cyan       = lipgloss.Color("#00ffee")
	neonGreen  = lipgloss.Color("#39ff14")
	sunYellow  = lipgloss.Color("#ffe000")
	warmOrange = lipgloss.Color("#ff6600")
)

// gradient steps for selected item
var gradientPink = []lipgloss.Color{"#ff2d95", "#ff44aa", "#ff55bb", "#ff66cc"}

// ── mock data ──────────────────────────────────────────────────────
type host struct {
	label, group, hostname, user, keyType string
	port                                  int
	tags                                  []string
	status                                int // 0=off, 1=idle, 2=live
	lastSSH, latency                      string
	uptime                                string
}

type grp struct {
	name      string
	collapsed bool
}

func mockGroups() []grp {
	return []grp{{name: "Production"}, {name: "Staging"}, {name: "Homelab", collapsed: true}}
}

func mockHosts() []host {
	return []host{
		{label: "api-prod-01", group: "Production", hostname: "api.neon.io", user: "deploy", port: 22, keyType: "ed25519", tags: []string{"api", "nginx"}, status: 2, lastSSH: "2m ago", latency: "23ms", uptime: "47d"},
		{label: "db-master", group: "Production", hostname: "10.0.1.50", user: "root", port: 5432, keyType: "rsa", tags: []string{"pg"}, status: 2, lastSSH: "active", latency: "8ms", uptime: "120d"},
		{label: "redis-cache", group: "Production", hostname: "10.0.1.75", user: "admin", port: 6379, keyType: "ecdsa", tags: []string{"redis", "cache"}, status: 1, lastSSH: "6h", latency: "3ms", uptime: "90d"},
		{label: "worker-fleet", group: "Production", hostname: "10.0.1.100-110", user: "deploy", port: 22, keyType: "ed25519", tags: []string{"worker"}, status: 0, lastSSH: "3d", latency: "—", uptime: "—"},
		{label: "stg-app", group: "Staging", hostname: "staging.neon.io", user: "dev", port: 2222, keyType: "password", tags: []string{"app"}, status: 1, lastSSH: "1h", latency: "45ms", uptime: "12d"},
		{label: "stg-db", group: "Staging", hostname: "stg-db.local", user: "postgres", port: 5432, keyType: "ecdsa", tags: []string{"db"}, status: 0, lastSSH: "1d", latency: "—", uptime: "—"},
		{label: "pi-k3s", group: "Homelab", hostname: "192.168.1.200", user: "pi", port: 22, keyType: "rsa", tags: []string{"arm"}, status: 0, lastSSH: "2w", latency: "1ms", uptime: "—"},
		{label: "nas-box", group: "Homelab", hostname: "192.168.1.50", user: "admin", port: 22, keyType: "ed25519", tags: []string{"storage"}, status: 0, lastSSH: "5d", latency: "2ms", uptime: "—"},
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

func tickCmd() tea.Cmd {
	return tea.Tick(400*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })
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

func statusGlyph(s int, blink bool) string {
	switch s {
	case 2:
		return lipgloss.NewStyle().Foreground(neonGreen).Render("◆")
	case 1:
		if blink {
			return lipgloss.NewStyle().Foreground(sunYellow).Render("◇")
		}
		return lipgloss.NewStyle().Foreground(sunYellow).Render("◆")
	default:
		return lipgloss.NewStyle().Foreground(darkViolet).Render("◇")
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
		return m.helpView(w)
	}

	blink := m.tick%2 == 0

	// ── top bar ──
	glowBar := ""
	barChars := "━"
	for i := 0; i < w; i++ {
		pct := float64(i) / float64(w)
		var c lipgloss.Color
		if pct < 0.5 {
			c = neonPink
		} else {
			c = neonBlue
		}
		glowBar += lipgloss.NewStyle().Foreground(c).Render(barChars)
	}

	titleLeft := lipgloss.NewStyle().Foreground(neonPink).Bold(true).Render("⚡ SSH") +
		lipgloss.NewStyle().Foreground(hotMagenta).Bold(true).Render("THING")
	live := 0
	for _, h := range m.hosts {
		if h.status == 2 {
			live++
		}
	}
	statsR := lipgloss.NewStyle().Foreground(lavender).Render(
		fmt.Sprintf("%d hosts  %d live  %d groups", len(m.hosts), live, len(m.groups)))
	hintR := lipgloss.NewStyle().Foreground(violet).Render("? help  / search  q quit")
	gap := strings.Repeat(" ", max(0, w-lipgloss.Width(titleLeft)-lipgloss.Width(statsR)-lipgloss.Width(hintR)-6))
	headerLine := "  " + titleLeft + "  " + statsR + gap + hintR + "  "
	headerBg := lipgloss.NewStyle().Background(deepPurple).Width(w).Render(headerLine)

	// ── three-column layout ──
	sideW := w * 28 / 100
	if sideW < 26 {
		sideW = 26
	}
	statsW := 22
	if w < 100 {
		statsW = 0 // hide stats panel on narrow terminals
	}
	mainW := w - sideW - statsW - 2
	bodyH := m.h - 5

	// ── sidebar ──
	var listLines []string
	for i, item := range m.items {
		sel := i == m.cursor
		if item.isGroup {
			arrow := "▾"
			if item.grp.collapsed {
				arrow = "▸"
			}
			name := item.grp.name
			cnt := fmt.Sprintf("(%d)", item.hostCount)
			if sel {
				line := lipgloss.NewStyle().Foreground(neonPink).Bold(true).Render(
					fmt.Sprintf(" %s %s %s", arrow, name, cnt))
				listLines = append(listLines, line)
			} else {
				ar := lipgloss.NewStyle().Foreground(purple).Render(arrow)
				nm := lipgloss.NewStyle().Foreground(lavender).Bold(true).Render(name)
				ct := lipgloss.NewStyle().Foreground(violet).Render(cnt)
				listLines = append(listLines, fmt.Sprintf(" %s %s %s", ar, nm, ct))
			}
		} else {
			h := item.host
			sg := statusGlyph(h.status, blink)
			lbl := h.label
			maxL := sideW - 8
			if maxL > 0 && len(lbl) > maxL {
				lbl = lbl[:maxL-1] + "…"
			}
			if sel {
				// gradient selection bar
				nameR := lipgloss.NewStyle().Foreground(voidBlack).Bold(true).Render(lbl)
				line := lipgloss.NewStyle().Background(neonPink).Width(sideW - 2).
					Render(fmt.Sprintf("  %s %s", "◆", nameR))
				listLines = append(listLines, line)
			} else {
				nm := lipgloss.NewStyle().Foreground(softWhite).Render(lbl)
				listLines = append(listLines, fmt.Sprintf("   %s %s", sg, nm))
			}
		}
	}
	for len(listLines) < bodyH-2 {
		listLines = append(listLines, "")
	}
	if len(listLines) > bodyH-2 {
		listLines = listLines[:bodyH-2]
	}

	sideTitle := lipgloss.NewStyle().Foreground(neonPink).Bold(true).Render("SERVERS")
	sidePanel := lipgloss.NewStyle().
		Width(sideW).Height(bodyH).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(darkViolet).
		Render(sideTitle + "\n" + strings.Join(listLines, "\n"))

	// ── main detail ──
	detailContent := m.renderDetail(mainW-4, bodyH-2, blink)
	detailPanel := lipgloss.NewStyle().
		Width(mainW).Height(bodyH).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(darkViolet).
		Render(detailContent)

	// ── quick stats sidebar ──
	var statsPanel string
	if statsW > 0 {
		statsContent := m.renderQuickStats(statsW-4, bodyH-2)
		statsPanel = lipgloss.NewStyle().
			Width(statsW).Height(bodyH).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(darkViolet).
			Render(statsContent)
	}

	body := lipgloss.JoinHorizontal(lipgloss.Top, sidePanel, detailPanel, statsPanel)

	// ── bottom glow bar + footer ──
	footerKeys := lipgloss.NewStyle().Foreground(violet).Background(deepPurple).Width(w).Padding(0, 2).
		Render("↑↓ nav   ⏎ ssh   s sftp   a add   e edit   d del   , settings")

	return lipgloss.NewStyle().Background(voidBlack).Width(m.w).Height(m.h).Render(
		headerBg + "\n" + glowBar + "\n" + body + "\n" + glowBar + "\n" + footerKeys)
}

func (m model) renderDetail(w, h int, blink bool) string {
	if m.cursor >= len(m.items) {
		return ""
	}
	item := m.items[m.cursor]

	if item.isGroup {
		t := lipgloss.NewStyle().Foreground(hotMagenta).Bold(true).Render("⚡ " + item.grp.name)
		sub := lipgloss.NewStyle().Foreground(lavender).Render(fmt.Sprintf("%d servers in this group", item.hostCount))
		sep := lipgloss.NewStyle().Foreground(darkViolet).Render(strings.Repeat("─", min(w, 40)))
		hint := lipgloss.NewStyle().Foreground(violet).Render("enter toggle  ·  a add  ·  e rename  ·  d delete")
		return t + "\n" + sub + "\n\n" + sep + "\n\n" + hint
	}

	ho := item.host
	lbl := lipgloss.NewStyle().Foreground(neonPink).Bold(true).Render(ho.label)
	stat := statusGlyph(ho.status, blink) + " "
	switch ho.status {
	case 2:
		stat += lipgloss.NewStyle().Foreground(neonGreen).Bold(true).Render("LIVE")
	case 1:
		stat += lipgloss.NewStyle().Foreground(sunYellow).Render("IDLE")
	default:
		stat += lipgloss.NewStyle().Foreground(violet).Render("OFFLINE")
	}

	kS := lipgloss.NewStyle().Foreground(violet).Width(14).Align(lipgloss.Right)
	vS := lipgloss.NewStyle().Foreground(softWhite)
	dS := lipgloss.NewStyle().Foreground(lavender)

	connStr := fmt.Sprintf("%s@%s:%d", ho.user, ho.hostname, ho.port)

	// connection box with glow
	connBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(midPurple).
		Foreground(cyan).
		Padding(0, 2).
		Render(connStr)

	tagStr := ""
	for _, t := range ho.tags {
		tagStr += lipgloss.NewStyle().Foreground(neonBlue).Render("#"+t) + " "
	}
	if tagStr == "" {
		tagStr = dS.Render("—")
	}

	sep := lipgloss.NewStyle().Foreground(darkViolet).Render(strings.Repeat("─", min(w, 50)))

	lines := []string{
		lbl,
		stat,
		"",
		connBox,
		"",
		kS.Render("auth") + "  " + vS.Render(ho.keyType),
		kS.Render("group") + "  " + dS.Render(ho.group),
		kS.Render("latency") + "  " + dS.Render(ho.latency),
		kS.Render("uptime") + "  " + dS.Render(ho.uptime),
		kS.Render("last") + "  " + dS.Render(ho.lastSSH),
		kS.Render("tags") + "  " + tagStr,
		"",
		sep,
		"",
		lipgloss.NewStyle().Foreground(violet).Render("⏎ connect  ·  s sftp  ·  e edit  ·  d delete"),
	}
	return strings.Join(lines, "\n")
}

func (m model) renderQuickStats(w, h int) string {
	title := lipgloss.NewStyle().Foreground(neonBlue).Bold(true).Render("STATS")
	live, idle, off := 0, 0, 0
	for _, h := range m.hosts {
		switch h.status {
		case 2:
			live++
		case 1:
			idle++
		default:
			off++
		}
	}

	liveR := lipgloss.NewStyle().Foreground(neonGreen).Bold(true).Render(fmt.Sprintf(" %d", live))
	idleR := lipgloss.NewStyle().Foreground(sunYellow).Render(fmt.Sprintf(" %d", idle))
	offR := lipgloss.NewStyle().Foreground(violet).Render(fmt.Sprintf(" %d", off))

	bar := func(n, total int, c lipgloss.Color) string {
		barW := w - 2
		if total == 0 {
			return ""
		}
		filled := barW * n / total
		if filled < 0 {
			filled = 0
		}
		return lipgloss.NewStyle().Foreground(c).Render(strings.Repeat("█", filled)) +
			lipgloss.NewStyle().Foreground(darkViolet).Render(strings.Repeat("░", barW-filled))
	}

	total := len(m.hosts)
	sep := lipgloss.NewStyle().Foreground(darkViolet).Render(strings.Repeat("─", w))

	lines := []string{
		title,
		"",
		lipgloss.NewStyle().Foreground(lavender).Render("live") + liveR,
		bar(live, total, neonGreen),
		"",
		lipgloss.NewStyle().Foreground(lavender).Render("idle") + idleR,
		bar(idle, total, sunYellow),
		"",
		lipgloss.NewStyle().Foreground(lavender).Render("off") + offR,
		bar(off, total, violet),
		"",
		sep,
		"",
		lipgloss.NewStyle().Foreground(violet).Render(fmt.Sprintf("total: %d", total)),
		lipgloss.NewStyle().Foreground(violet).Render(fmt.Sprintf("groups: %d", len(m.groups))),
	}
	return strings.Join(lines, "\n")
}

func (m model) helpView(w int) string {
	title := lipgloss.NewStyle().Foreground(neonPink).Bold(true).Render("⚡ Keyboard Shortcuts")
	pairs := [][2]string{
		{"↑ / ↓ / j / k", "Navigate"},
		{"enter", "Connect / Toggle"},
		{"s → enter", "SFTP"},
		{"a", "Add host"},
		{"e", "Edit"},
		{"d", "Delete"},
		{"/", "Search"},
		{",", "Settings"},
		{"?", "Help"},
		{"q", "Quit"},
	}
	kS := lipgloss.NewStyle().Foreground(cyan).Width(20).Align(lipgloss.Right)
	vS := lipgloss.NewStyle().Foreground(softWhite)
	var lines []string
	for _, p := range pairs {
		lines = append(lines, kS.Render(p[0])+"  "+vS.Render(p[1]))
	}
	content := title + "\n\n" + strings.Join(lines, "\n") + "\n\n" +
		lipgloss.NewStyle().Foreground(violet).Render("? or esc to close")

	box := lipgloss.NewStyle().
		Width(60).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(midPurple).
		Padding(1, 3).
		Render(content)

	return lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, box,
		lipgloss.WithWhitespaceBackground(voidBlack))
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
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
