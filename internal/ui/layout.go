package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const SidebarW = 4

// Renderer holds all context needed to render views.
type Renderer struct {
	Theme Theme
	Icons IconSet
	W, H  int
	Tick  int
}

// PageIndicator pairs a sidebar icon with its page index.
type PageIndicator struct {
	Icon  string
	Index int
}

// PageIcons returns the sidebar page indicators.
func (r *Renderer) PageIcons() []PageIndicator {
	return []PageIndicator{
		{r.Icons.Home, 0},
		{r.Icons.Settings, 1},
		{r.Icons.Tokens, 2},
		{r.Icons.Shield, 3},
	}
}

// ContentWidth returns the total usable inner width (padding each side).
func (r *Renderer) ContentWidth() int {
	cw := r.W - 8
	if cw > 160 {
		cw = 160
	}
	if cw < 40 {
		cw = 40
	}
	return cw
}

// ShowSidebar returns true when the terminal is wide enough for the sidebar.
func (r *Renderer) ShowSidebar() bool {
	return r.W >= 60
}

// PageContentWidth returns the content width for page views (minus sidebar + gap).
func (r *Renderer) PageContentWidth() int {
	cw := r.ContentWidth()
	if r.ShowSidebar() {
		cw -= SidebarW + 2
	}
	if cw < 36 {
		cw = 36
	}
	return cw
}

// LeftPad returns the left padding to center the total content block.
func (r *Renderer) LeftPad() int {
	cw := r.ContentWidth()
	total := r.W - cw
	if total < 0 {
		return 0
	}
	return total / 2
}

// PadContent adds bg-colored left padding and a top blank line.
func (r *Renderer) PadContent(inner string, leftPadN int) string {
	bg := lipgloss.NewStyle().Background(r.Theme.Base)
	padStr := bg.Render(strings.Repeat(" ", leftPadN))
	lines := strings.Split(inner, "\n")
	for i, l := range lines {
		lines[i] = padStr + l
	}
	return "\n" + strings.Join(lines, "\n")
}

// WrapFull fills every cell with the theme background, guaranteeing no terminal bg leaks.
func (r *Renderer) WrapFull(content string) string {
	bg := lipgloss.NewStyle().Background(r.Theme.Base)
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		visualW := lipgloss.Width(line)
		if visualW < r.W {
			lines[i] = line + bg.Render(strings.Repeat(" ", r.W-visualW))
		}
	}

	emptyLine := bg.Render(strings.Repeat(" ", r.W))
	for len(lines) < r.H {
		lines = append(lines, emptyLine)
	}
	if len(lines) > r.H {
		lines = lines[:r.H]
	}

	out := strings.Join(lines, "\n")
	return ApplyBaseBg(out, r.Theme.Base)
}

// RenderHeader returns the standard header with optional subtitle.
func (r *Renderer) RenderHeader(subtitle string, hostCount, connectedCount int) string {
	title := lipgloss.NewStyle().Foreground(r.Theme.Text).Bold(true).Render("sshthing")
	if subtitle == "" {
		subtitle = fmt.Sprintf("%d hosts  %d connected", hostCount, connectedCount)
	}
	meta := lipgloss.NewStyle().Foreground(r.Theme.Subtext).Render(subtitle)
	return title + "    " + meta
}

// RenderFooter returns a dimmed footer hint line.
func (r *Renderer) RenderFooter(text string) string {
	return lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render(text)
}

// RenderSidebarItem renders a single sidebar icon with active/inactive state.
func (r *Renderer) RenderSidebarItem(icon string, active bool) string {
	dotStyle := lipgloss.NewStyle().Foreground(r.Theme.Surface0)
	iconStyle := lipgloss.NewStyle().Foreground(r.Theme.Overlay)
	dot := r.Icons.InactiveMarker
	if active {
		dot = r.Icons.ActiveMarker
		dotStyle = lipgloss.NewStyle().Foreground(r.Theme.Accent)
		iconStyle = lipgloss.NewStyle().Foreground(r.Theme.Text)
	}

	rendered := dotStyle.Render(dot) + iconStyle.Render(icon)
	visualW := lipgloss.Width(rendered)
	pad := SidebarW - visualW
	if pad < 0 {
		pad = 0
	}
	return strings.Repeat(" ", pad) + rendered
}

// RenderSidebar renders the vertical sidebar for page navigation.
func (r *Renderer) RenderSidebar(bodyH int, activePage int) string {
	var items []string
	for _, pi := range r.PageIcons() {
		items = append(items, r.RenderSidebarItem(pi.Icon, activePage == pi.Index))
	}

	topPad := bodyH/3 - len(items)/2
	if topPad < 0 {
		topPad = 0
	}

	var lines []string
	for i := 0; i < bodyH; i++ {
		idx := i - topPad
		if idx >= 0 && idx < len(items) {
			lines = append(lines, items[idx])
		} else {
			lines = append(lines, strings.Repeat(" ", SidebarW))
		}
	}
	return lipgloss.NewStyle().Width(SidebarW).Render(strings.Join(lines, "\n"))
}

// TruncStr truncates a string to width, adding an ellipsis if needed.
func (r *Renderer) TruncStr(s string, w int) string {
	if w <= 0 || len(s) <= w {
		return s
	}
	if w <= 1 {
		return r.Icons.Truncation
	}
	return s[:w-1] + r.Icons.Truncation
}

// RemoveLastRune removes the last rune from a string.
func RemoveLastRune(s string) string {
	if s == "" {
		return ""
	}
	runes := []rune(s)
	return string(runes[:len(runes)-1])
}
