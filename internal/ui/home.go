package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// SyncActivity holds the state of an ongoing sync operation.
type SyncActivity struct {
	Active   bool
	Frame    int
	Progress float64
	Stage    string
}

// HomeViewParams holds all data needed to render the home page.
type HomeViewParams struct {
	Items        []HomeListItem
	Cursor       int
	Err          error
	SyncActivity *SyncActivity
	Page         int
	HostCount    int
	Connected    int
}

// HomeListItem represents one row in the home list.
type HomeListItem struct {
	IsGroup    bool
	IsNewGroup bool
	GroupName  string
	Collapsed  bool
	HostCount  int
	// Host fields
	Label         string
	Hostname      string
	Username      string
	Port          int
	KeyType       string
	Tags          []string
	Status        int // 0=offline, 1=idle, 2=connected
	LastSSH       string
	Mounted       bool
	MountPath     string
	LastConnected *time.Time
}

func (r *Renderer) renderListEntry(prefix string, label string, style lipgloss.Style, width int) []string {
	label = strings.TrimSpace(label)
	if label == "" {
		return []string{prefix}
	}
	if width <= 0 {
		width = 1
	}
	if !r.WrapLabels {
		return []string{prefix + style.Render(r.TruncStr(label, width))}
	}

	rawLines := wrapPlainTextLines(label, width)
	indent := strings.Repeat(" ", lipgloss.Width(prefix))
	lines := make([]string, 0, len(rawLines))
	for i, line := range rawLines {
		line = style.Render(strings.TrimRight(line, " "))
		if i == 0 {
			lines = append(lines, prefix+line)
		} else {
			lines = append(lines, indent+line)
		}
	}
	return lines
}

type homeFrameLayout struct {
	listW      int
	gapW       int
	detailW    int
	bodyH      int
	narrowMode bool
}

func (r *Renderer) buildHomeFrameLayout(notifCount int) homeFrameLayout {
	cw := r.PageContentWidth()

	listW := cw * 30 / 100
	if listW < 24 {
		listW = 24
	}
	gapW := 4
	detailW := cw - listW - gapW
	if detailW < 20 {
		detailW = 20
	}
	bodyH := r.H - 6
	if bodyH < 4 {
		bodyH = 4
	}

	bodyH -= notifCount
	if bodyH < 4 {
		bodyH = 4
	}

	return homeFrameLayout{
		listW:      listW,
		gapW:       gapW,
		detailW:    detailW,
		bodyH:      bodyH,
		narrowMode: r.W < 70,
	}
}

func (r *Renderer) renderHomeBody(listBlock string, detailBlock string, layout homeFrameLayout, page int) string {
	body := listBlock
	if !layout.narrowMode {
		gapBlock := lipgloss.NewStyle().Width(layout.gapW).Render(strings.Repeat("\n", layout.bodyH))
		body = lipgloss.JoinHorizontal(lipgloss.Top, listBlock, gapBlock, detailBlock)
	}

	if r.ShowSidebar() {
		sidebar := r.RenderSidebar(layout.bodyH, page)
		sideGap := lipgloss.NewStyle().Width(2).Render(strings.Repeat("\n", layout.bodyH))
		body = lipgloss.JoinHorizontal(lipgloss.Top, body, sideGap, sidebar)
	}
	return body
}

func (r *Renderer) renderHomeFrame(header string, body string, notifLines []string, footerText string) string {
	pad := r.LeftPad()
	filtered := make([]string, 0, len(notifLines))
	for _, line := range notifLines {
		line = strings.TrimRight(line, "\n")
		if strings.TrimSpace(line) == "" {
			continue
		}
		filtered = append(filtered, line)
	}

	inner := header + "\n\n" + body + "\n\n"
	if len(filtered) > 0 {
		inner += strings.Join(filtered, "\n") + "\n"
	}
	inner += r.RenderFooter(footerText)
	return r.PadContent(inner, pad)
}

func (r *Renderer) padListLines(lines []string, bodyH int) string {
	for len(lines) < bodyH {
		lines = append(lines, "")
	}
	if len(lines) > bodyH {
		lines = lines[:bodyH]
	}
	return strings.Join(lines, "\n")
}

// RenderHomeView renders the three-panel home layout (list + detail + sidebar).
func (r *Renderer) RenderHomeView(p HomeViewParams) string {
	notifCount := 0
	if p.Err != nil {
		notifCount++
	}
	if p.SyncActivity != nil && p.SyncActivity.Active {
		notifCount++
	}
	layout := r.buildHomeFrameLayout(notifCount)

	// list column
	var listLines []string
	for i, item := range p.Items {
		sel := i == p.Cursor
		if item.IsNewGroup {
			nameStyle := lipgloss.NewStyle().Foreground(r.Theme.Overlay)
			prefix := "    "
			if sel {
				nameStyle = lipgloss.NewStyle().Foreground(r.Theme.Accent)
				prefix = lipgloss.NewStyle().Foreground(r.Theme.Accent).Render("  " + r.Icons.Focused + " ")
			}
			listLines = append(listLines, "")
			listLines = append(listLines, r.renderListEntry(prefix, r.Icons.Add+" new group", nameStyle, layout.listW-6)...)
		} else if item.IsGroup {
			arrow := r.Icons.Expanded
			if item.Collapsed {
				arrow = r.Icons.Collapsed
			}
			nameStyle := lipgloss.NewStyle().Foreground(r.Theme.Subtext)
			if sel {
				nameStyle = lipgloss.NewStyle().Foreground(r.Theme.Accent)
			}
			countStr := lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render(fmt.Sprintf(" %d", item.HostCount))
			arrowR := lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render(arrow)
			if sel {
				arrowR = lipgloss.NewStyle().Foreground(r.Theme.Accent).Render(arrow)
			}
			listLines = append(listLines, "")
			groupPrefix := arrowR + " "
			groupLines := r.renderListEntry(groupPrefix, item.GroupName, nameStyle, layout.listW-lipgloss.Width(groupPrefix)-4)
			if len(groupLines) > 0 {
				groupLines[0] += countStr
			}
			listLines = append(listLines, groupLines...)
		} else {
			prefix := "    "
			nameStyle := lipgloss.NewStyle().Foreground(r.Theme.Subtext)

			var dot string
			switch item.Status {
			case 2:
				dot = lipgloss.NewStyle().Foreground(r.Theme.Green).Render(r.Icons.Connected)
			case 1:
				dot = lipgloss.NewStyle().Foreground(r.Theme.Yellow).Render(r.Icons.Idle)
			default:
				dot = lipgloss.NewStyle().Foreground(r.Theme.Surface0).Render(r.Icons.Offline)
			}

			if sel {
				nameStyle = lipgloss.NewStyle().Foreground(r.Theme.Accent).Bold(true)
				prefix = lipgloss.NewStyle().Foreground(r.Theme.Accent).Render("  " + r.Icons.Focused + " ")
			}

			maxLblW := layout.listW - 10
			if layout.narrowMode {
				maxLblW = r.W - 16
			}
			lbl := strings.TrimSpace(item.Label)
			if lbl == "" {
				lbl = item.Hostname
			}
			entryPrefix := prefix + dot + " "
			listLines = append(listLines, r.renderListEntry(entryPrefix, lbl, nameStyle, maxLblW)...)
		}
	}

	listBlock := lipgloss.NewStyle().Width(layout.listW).Render(r.padListLines(listLines, layout.bodyH))
	detailBlock := lipgloss.NewStyle().Width(layout.detailW).Foreground(r.Theme.Subtext).
		Render(r.renderDetail(p, layout.detailW, layout.bodyH))
	body := r.renderHomeBody(listBlock, detailBlock, layout, p.Page)

	// notification area above footer (err + sync status)
	var notifLines []string
	if p.Err != nil {
		notifLines = append(notifLines, r.renderErrLine(p.Err))
	}
	if p.SyncActivity != nil && p.SyncActivity.Active {
		notifLines = append(notifLines, r.renderSyncFooter(p.SyncActivity))
	}

	headerLine := r.RenderHeader("", p.HostCount, p.Connected)
	footerText := "\u2191\u2193 nav  \u23CE connect  S sftp  M mount  Y sync  / search  a add  e edit  d del  , settings  ? help  q quit"
	return r.renderHomeFrame(headerLine, body, notifLines, footerText)
}

func (r *Renderer) renderDetail(p HomeViewParams, w, h int) string {
	if p.Cursor >= len(p.Items) {
		return ""
	}
	item := p.Items[p.Cursor]

	if item.IsNewGroup {
		hint := lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render("press enter or a to create a new group")
		return lipgloss.NewStyle().Foreground(r.Theme.Text).Bold(true).Render("new group") + "\n\n" + hint
	}

	if item.IsGroup {
		name := lipgloss.NewStyle().Foreground(r.Theme.Text).Bold(true).Render(item.GroupName)
		sub := lipgloss.NewStyle().Foreground(r.Theme.Subtext).Render(fmt.Sprintf("%d servers", item.HostCount))
		hint := lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render("enter toggle  \u00B7  e rename  \u00B7  d delete")
		return name + "\n" + sub + "\n\n" + hint
	}

	kStyle := lipgloss.NewStyle().Foreground(r.Theme.Subtext)
	vStyle := lipgloss.NewStyle().Foreground(r.Theme.Text)
	dimStyle := lipgloss.NewStyle().Foreground(r.Theme.Subtext)

	var statusR string
	switch item.Status {
	case 2:
		statusR = lipgloss.NewStyle().Foreground(r.Theme.Green).Render("connected")
	case 1:
		statusR = lipgloss.NewStyle().Foreground(r.Theme.Yellow).Render("idle")
	default:
		statusR = lipgloss.NewStyle().Foreground(r.Theme.Subtext).Render("offline")
	}

	connStr := fmt.Sprintf("%s@%s", item.Username, item.Hostname)
	if item.Port != 22 && item.Port != 0 {
		connStr += fmt.Sprintf(":%d", item.Port)
	}

	tagStr := ""
	for _, t := range item.Tags {
		tagStr += lipgloss.NewStyle().Foreground(r.Theme.Pink).Render(t) + "  "
	}
	if tagStr == "" {
		tagStr = lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render("no tags")
	}

	displayLabel := item.Label
	if displayLabel == "" {
		displayLabel = item.Hostname
	}
	title := lipgloss.NewStyle().Foreground(r.Theme.Text).Bold(true).Render(displayLabel)

	lastSeen := item.LastSSH
	if lastSeen == "" && item.LastConnected != nil {
		lastSeen = FormatTimeAgo(*item.LastConnected)
	}
	if lastSeen == "" {
		lastSeen = "never"
	}

	mountLine := ""
	if item.Mounted {
		mountLine = kStyle.Render("mount       ") + lipgloss.NewStyle().Foreground(r.Theme.Green).Render(item.MountPath)
	}

	lines := []string{
		title,
		statusR,
		"",
		lipgloss.NewStyle().Foreground(r.Theme.Accent).Render(connStr),
		"",
		kStyle.Render("auth        ") + vStyle.Render(item.KeyType),
		kStyle.Render("group       ") + dimStyle.Render(item.GroupName),
		kStyle.Render("last seen   ") + dimStyle.Render(lastSeen),
	}
	if mountLine != "" {
		lines = append(lines, mountLine)
	}
	lines = append(lines, "", kStyle.Render("tags        ")+tagStr)
	lines = append(lines, "", "")
	lines = append(lines, lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render("enter connect  \u00B7  S sftp  \u00B7  M mount  \u00B7  e edit  \u00B7  d delete"))

	return strings.Join(lines, "\n")
}

func (r *Renderer) renderSyncFooter(activity *SyncActivity) string {
	if activity == nil || !activity.Active {
		return ""
	}
	frames := []string{"|", "/", "-", "\\"}
	icon := frames[activity.Frame%len(frames)]
	stage := strings.TrimSpace(activity.Stage)
	label := icon + " Syncing"
	if stage != "" {
		label += " (" + stage + ")"
	}
	return lipgloss.NewStyle().Foreground(r.Theme.Sky).Render(label)
}

func (r *Renderer) renderErrLine(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	if strings.HasPrefix(msg, "\u2713") {
		return lipgloss.NewStyle().Foreground(r.Theme.Green).Render(msg)
	}
	if strings.HasPrefix(msg, "\u26A0") {
		return lipgloss.NewStyle().Foreground(r.Theme.Yellow).Render(msg)
	}
	if strings.HasPrefix(msg, "\u2139") {
		return lipgloss.NewStyle().Foreground(r.Theme.Sky).Render(msg)
	}
	return lipgloss.NewStyle().Foreground(r.Theme.Red).Render(msg)
}

// FormatTimeAgo formats a time duration as a human-readable string.
func FormatTimeAgo(t time.Time) string {
	duration := time.Since(t)
	if duration < time.Minute {
		return "just now"
	} else if duration < time.Hour {
		mins := int(duration.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	} else if duration < 24*time.Hour {
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	} else if duration < 7*24*time.Hour {
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
	weeks := int(duration.Hours() / 24 / 7)
	if weeks == 1 {
		return "1 week ago"
	}
	return fmt.Sprintf("%d weeks ago", weeks)
}
