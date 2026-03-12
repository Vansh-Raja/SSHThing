// sshmock1 — "Noir"
// OpenCode-inspired minimal dark design with subtle gradient accents,
// monochrome panels, and a single pop of electric blue for focus states.
// Philosophy: information density through restraint.
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
	base00 = lipgloss.Color("#0a0a0f")
	base01 = lipgloss.Color("#12121a")
	base02 = lipgloss.Color("#1a1a26")
	base03 = lipgloss.Color("#25253a")
	base04 = lipgloss.Color("#3a3a55")
	base05 = lipgloss.Color("#555577")
	base06 = lipgloss.Color("#8888aa")
	base07 = lipgloss.Color("#aaaabb")
	base08 = lipgloss.Color("#ccccdd")
	base09 = lipgloss.Color("#e8e8f0")

	accent    = lipgloss.Color("#00d4ff")
	accentDim = lipgloss.Color("#0088aa")
	green     = lipgloss.Color("#00e88f")
	red       = lipgloss.Color("#ff4466")
	yellow    = lipgloss.Color("#ffcc00")
	orange    = lipgloss.Color("#ff8844")
)

// ── mock data ──────────────────────────────────────────────────────
type host struct {
	label, group, hostname, user, keyType string
	port                                  int
	tags                                  []string
	connected                             bool
	lastSSH                               string
	latency                               string
}

type group struct {
	name      string
	collapsed bool
}

func mockGroups() []group {
	return []group{
		{name: "Production", collapsed: false},
		{name: "Staging", collapsed: false},
		{name: "Development", collapsed: true},
	}
}

func mockHosts() []host {
	return []host{
		{label: "api-gateway", group: "Production", hostname: "api.prod.example.com", user: "deploy", port: 22, keyType: "ed25519", tags: []string{"api", "nginx"}, connected: true, lastSSH: "2m ago", latency: "23ms"},
		{label: "db-primary", group: "Production", hostname: "10.0.1.50", user: "root", port: 5432, keyType: "rsa", tags: []string{"postgres", "primary"}, connected: true, lastSSH: "1h ago", latency: "12ms"},
		{label: "worker-01", group: "Production", hostname: "10.0.1.100", user: "deploy", port: 22, keyType: "ed25519", tags: []string{"worker", "redis"}, connected: false, lastSSH: "3d ago", latency: "—"},
		{label: "cache-node", group: "Production", hostname: "10.0.1.75", user: "admin", port: 22, keyType: "ecdsa", tags: []string{"redis", "cache"}, connected: false, lastSSH: "12h ago", latency: "8ms"},
		{label: "staging-app", group: "Staging", hostname: "staging.example.com", user: "dev", port: 2222, keyType: "password", tags: []string{"app"}, connected: false, lastSSH: "5h ago", latency: "45ms"},
		{label: "staging-db", group: "Staging", hostname: "staging-db.local", user: "postgres", port: 5432, keyType: "ecdsa", tags: []string{"db"}, connected: false, lastSSH: "1d ago", latency: "—"},
		{label: "dev-box", group: "Development", hostname: "192.168.1.100", user: "vansh", port: 22, keyType: "ed25519", tags: []string{"local"}, connected: false, lastSSH: "Just now", latency: "1ms"},
		{label: "pi-cluster", group: "Development", hostname: "192.168.1.200", user: "pi", port: 22, keyType: "rsa", tags: []string{"arm", "k3s"}, connected: false, lastSSH: "2w ago", latency: "—"},
	}
}

// ── list item (group or host) ──────────────────────────────────────
type listItem struct {
	isGroup   bool
	group     group
	host      host
	hostCount int
}

func buildList(groups []group, hosts []host) []listItem {
	var items []listItem
	for _, g := range groups {
		count := 0
		for _, h := range hosts {
			if h.group == g.name {
				count++
			}
		}
		items = append(items, listItem{isGroup: true, group: g, hostCount: count})
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
type view int

const (
	viewList view = iota
	viewHelp
	viewSearch
)

type model struct {
	items    []listItem
	groups   []group
	hosts    []host
	cursor   int
	w, h     int
	view     view
	search   string
	tick     int
}

func initialModel() model {
	g := mockGroups()
	h := mockHosts()
	return model{
		groups: g,
		hosts:  h,
		items:  buildList(g, h),
	}
}

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
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
		if m.view == viewSearch {
			switch msg.String() {
			case "esc":
				m.view = viewList
				m.search = ""
			case "backspace":
				if len(m.search) > 0 {
					m.search = m.search[:len(m.search)-1]
				}
			case "enter":
				m.view = viewList
			default:
				if len(msg.String()) == 1 {
					m.search += msg.String()
				}
			}
			return m, nil
		}
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
				name := m.items[m.cursor].group.name
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
			if m.view == viewHelp {
				m.view = viewList
			} else {
				m.view = viewHelp
			}
		case "/":
			m.view = viewSearch
			m.search = ""
		}
	}
	return m, nil
}

// ── rendering ──────────────────────────────────────────────────────

func (m model) View() string {
	if m.w == 0 {
		return ""
	}

	w := m.w
	if w > 160 {
		w = 160
	}

	switch m.view {
	case viewHelp:
		return m.renderHelp(w)
	case viewSearch:
		return m.renderSearch(w)
	default:
		return m.renderMain(w)
	}
}

func (m model) renderMain(maxW int) string {
	// ── header ──
	headerStyle := lipgloss.NewStyle().
		Foreground(base06).
		Background(base01).
		Width(maxW).
		Padding(0, 2)

	titlePart := lipgloss.NewStyle().Foreground(accent).Bold(true).Render("ssh")
	titlePart2 := lipgloss.NewStyle().Foreground(base08).Bold(true).Render("thing")
	connCount := 0
	for _, h := range m.hosts {
		if h.connected {
			connCount++
		}
	}
	connBadge := lipgloss.NewStyle().Foreground(green).Render(fmt.Sprintf(" %d active", connCount))
	rightSide := lipgloss.NewStyle().Foreground(base05).Render("? help  / search  q quit")
	gap := strings.Repeat(" ", max(0, maxW-4-lipgloss.Width(titlePart+titlePart2)-lipgloss.Width(connBadge)-lipgloss.Width(rightSide)))
	headerContent := titlePart + titlePart2 + connBadge + gap + rightSide
	header := headerStyle.Render(headerContent)

	// ── panels ──
	listW := maxW * 30 / 100
	if listW < 28 {
		listW = 28
	}
	detailW := maxW - listW - 3 // 3 for divider
	availH := m.h - 4           // header + footer

	// ── host list ──
	var listLines []string
	for i, item := range m.items {
		selected := i == m.cursor
		var line string
		if item.isGroup {
			arrow := "▾"
			if item.group.collapsed {
				arrow = "▸"
			}
			countStr := lipgloss.NewStyle().Foreground(base05).Render(fmt.Sprintf(" %d", item.hostCount))
			nameStr := lipgloss.NewStyle().Foreground(base08).Bold(true).Render(item.group.name)
			if selected {
				nameStr = lipgloss.NewStyle().Foreground(accent).Bold(true).Render(item.group.name)
			}
			arrowStyle := lipgloss.NewStyle().Foreground(base05)
			if selected {
				arrowStyle = arrowStyle.Foreground(accent)
			}
			line = fmt.Sprintf("  %s %s%s", arrowStyle.Render(arrow), nameStr, countStr)
		} else {
			prefix := "    "
			dot := lipgloss.NewStyle().Foreground(base04).Render("○")
			if item.host.connected {
				dot = lipgloss.NewStyle().Foreground(green).Render("●")
			}
			nameStyle := lipgloss.NewStyle().Foreground(base07)
			if selected {
				nameStyle = lipgloss.NewStyle().Foreground(accent).Bold(true)
				prefix = lipgloss.NewStyle().Foreground(accent).Render(" ▸") + "  "
			}
			lbl := item.host.label
			maxLbl := listW - 10
			if maxLbl > 0 && len(lbl) > maxLbl {
				lbl = lbl[:maxLbl-1] + "…"
			}
			line = fmt.Sprintf("%s%s %s", prefix, dot, nameStyle.Render(lbl))
		}
		// pad to width
		lineW := lipgloss.Width(line)
		if lineW < listW-2 {
			line += strings.Repeat(" ", listW-2-lineW)
		}
		listLines = append(listLines, line)
	}
	for len(listLines) < availH-2 {
		listLines = append(listLines, strings.Repeat(" ", listW-2))
	}
	if len(listLines) > availH-2 {
		listLines = listLines[:availH-2]
	}

	listTitle := lipgloss.NewStyle().Foreground(base05).Render("SERVERS")
	listPanel := lipgloss.NewStyle().
		Width(listW).
		Height(availH).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(base03).
		Render(listTitle + "\n" + strings.Join(listLines, "\n"))

	// ── detail panel ──
	detailContent := m.renderDetail(detailW-4, availH-2)
	detailPanel := lipgloss.NewStyle().
		Width(detailW).
		Height(availH).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(base03).
		Render(detailContent)

	body := lipgloss.JoinHorizontal(lipgloss.Top, listPanel, " ", detailPanel)

	// ── footer ──
	footerStyle := lipgloss.NewStyle().
		Foreground(base05).
		Background(base01).
		Width(maxW).
		Padding(0, 2)

	keys := []struct{ k, v string }{
		{"↑/↓", "nav"},
		{"enter", "connect"},
		{"s", "sftp"},
		{"a", "add"},
		{"e", "edit"},
		{"d", "del"},
	}
	var parts []string
	for _, kv := range keys {
		k := lipgloss.NewStyle().Foreground(base07).Render(kv.k)
		v := lipgloss.NewStyle().Foreground(base05).Render(kv.v)
		parts = append(parts, k+" "+v)
	}
	footer := footerStyle.Render(strings.Join(parts, "   "))

	// ── assemble ──
	out := lipgloss.NewStyle().Background(base00).Width(maxW).Height(m.h).Render(
		header + "\n" + body + "\n" + footer,
	)
	return out
}

func (m model) renderDetail(w, h int) string {
	if m.cursor >= len(m.items) {
		return ""
	}
	item := m.items[m.cursor]

	if item.isGroup {
		title := lipgloss.NewStyle().Foreground(accent).Bold(true).Render(item.group.name)
		sub := lipgloss.NewStyle().Foreground(base06).Render(fmt.Sprintf("%d servers", item.hostCount))
		state := "expanded"
		if item.group.collapsed {
			state = "collapsed"
		}
		stateR := lipgloss.NewStyle().Foreground(base05).Render(state)
		return fmt.Sprintf("%s\n%s  •  %s\n\n%s",
			title, sub, stateR,
			lipgloss.NewStyle().Foreground(base05).Render("enter toggle  •  a add host  •  e rename  •  d delete"))
	}

	ho := item.host
	labelStyle := lipgloss.NewStyle().Foreground(base05).Width(14).Align(lipgloss.Right)
	valStyle := lipgloss.NewStyle().Foreground(base08)
	accentVal := lipgloss.NewStyle().Foreground(accent)
	dimVal := lipgloss.NewStyle().Foreground(base06)

	statusDot := lipgloss.NewStyle().Foreground(red).Render("● offline")
	if ho.connected {
		statusDot = lipgloss.NewStyle().Foreground(green).Render("● connected")
	}

	connStr := fmt.Sprintf("%s@%s:%d", ho.user, ho.hostname, ho.port)

	tagStr := ""
	for _, t := range ho.tags {
		tagStr += lipgloss.NewStyle().Foreground(accentDim).Render("#"+t) + " "
	}
	if tagStr == "" {
		tagStr = dimVal.Render("none")
	}

	lines := []string{
		accentVal.Render(ho.label),
		statusDot + "  " + dimVal.Render(ho.latency),
		"",
		labelStyle.Render("connection") + "  " + valStyle.Render(connStr),
		labelStyle.Render("auth") + "  " + valStyle.Render(ho.keyType),
		labelStyle.Render("group") + "  " + dimVal.Render(ho.group),
		labelStyle.Render("last session") + "  " + dimVal.Render(ho.lastSSH),
		labelStyle.Render("tags") + "  " + tagStr,
		"",
		lipgloss.NewStyle().Foreground(base04).Render(strings.Repeat("─", min(w, 50))),
		"",
		lipgloss.NewStyle().Foreground(base05).Render("enter connect  •  s sftp  •  e edit  •  d delete"),
	}

	return strings.Join(lines, "\n")
}

func (m model) renderHelp(w int) string {
	title := lipgloss.NewStyle().Foreground(accent).Bold(true).Render("Keyboard Shortcuts")
	pairs := [][2]string{
		{"↑ / ↓  or  j / k", "Navigate"},
		{"enter", "Connect SSH / Toggle group"},
		{"s → enter", "Connect SFTP"},
		{"a", "Add new host"},
		{"e", "Edit selected"},
		{"d", "Delete selected"},
		{"/", "Search hosts"},
		{",", "Settings"},
		{"?", "Toggle help"},
		{"q", "Quit"},
	}
	kStyle := lipgloss.NewStyle().Foreground(accent).Width(22).Align(lipgloss.Right)
	vStyle := lipgloss.NewStyle().Foreground(base07)
	var lines []string
	for _, p := range pairs {
		lines = append(lines, kStyle.Render(p[0])+"  "+vStyle.Render(p[1]))
	}
	content := title + "\n\n" + strings.Join(lines, "\n") + "\n\n" +
		lipgloss.NewStyle().Foreground(base05).Render("press ? or esc to close")

	box := lipgloss.NewStyle().
		Width(60).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(base03).
		Padding(1, 3).
		Render(content)

	return lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, box,
		lipgloss.WithWhitespaceBackground(base00))
}

func (m model) renderSearch(w int) string {
	cursor := lipgloss.NewStyle().Foreground(accent).Render("▌")
	prompt := lipgloss.NewStyle().Foreground(base05).Render("search: ")
	input := lipgloss.NewStyle().Foreground(base09).Render(m.search) + cursor

	// filter results
	var results []host
	for _, h := range m.hosts {
		if m.search == "" || strings.Contains(strings.ToLower(h.label+h.hostname+h.user+h.group), strings.ToLower(m.search)) {
			results = append(results, h)
		}
	}

	var lines []string
	for i, h := range results {
		if i >= 10 {
			break
		}
		dot := lipgloss.NewStyle().Foreground(base04).Render("○")
		if h.connected {
			dot = lipgloss.NewStyle().Foreground(green).Render("●")
		}
		grp := lipgloss.NewStyle().Foreground(base05).Render("[" + h.group + "]")
		name := lipgloss.NewStyle().Foreground(base08).Render(h.label)
		conn := lipgloss.NewStyle().Foreground(base06).Render(h.user + "@" + h.hostname)
		lines = append(lines, fmt.Sprintf("  %s %s %s  %s", dot, grp, name, conn))
	}

	sep := lipgloss.NewStyle().Foreground(base03).Render(strings.Repeat("─", 56))
	body := prompt + input + "\n" + sep + "\n" + strings.Join(lines, "\n") +
		"\n\n" + lipgloss.NewStyle().Foreground(base05).Render("esc close  •  enter select")

	box := lipgloss.NewStyle().
		Width(60).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(accent).
		Padding(1, 2).
		Render(body)

	return lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, box,
		lipgloss.WithWhitespaceBackground(base00))
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
