package ui

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
)

const SidebarW = 4

// Renderer holds all context needed to render views.
type Renderer struct {
	Theme          Theme
	Icons          IconSet
	WrapLabels     bool
	W, H           int
	Tick           int
	PageIndicators []PageIndicator
}

// PageIndicator pairs a sidebar icon with its page index.
type PageIndicator struct {
	Icon  string
	Index int
}

// PageIcons returns the sidebar page indicators.
func (r *Renderer) PageIcons() []PageIndicator {
	if len(r.PageIndicators) > 0 {
		return r.PageIndicators
	}
	return []PageIndicator{
		{r.Icons.Home, 0},
		{r.Icons.Profile, 1},
		{r.Icons.Settings, 2},
		{r.Icons.Tokens, 3},
		{r.Icons.Teams, 4},
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

func (r *Renderer) RenderMainFooter() string {
	return r.RenderFooter(r.MainFooterText())
}

func (r *Renderer) MainFooterText() string {
	if r.W < 72 {
		return "\u2191\u2193 \u00B7 enter \u00B7 / \u00B7 : \u00B7 q"
	}
	return "\u2191\u2193 nav \u00B7 enter select \u00B7 / search \u00B7 : commands \u00B7 q quit"
}

type CommandLineItem struct {
	Name           string
	Description    string
	Disabled       bool
	DisabledReason string
	Danger         bool
}

type CommandLineView struct {
	Query  string
	Cursor int
	Items  []CommandLineItem
}

func (r *Renderer) RenderCommandLine(p CommandLineView) string {
	return r.RenderCommandLineWithHeight(p, r.CommandLineHeight(&p))
}

func (r *Renderer) FooterBlockHeight(commandLine *CommandLineView) int {
	if commandLine == nil {
		return 1
	}
	return r.CommandLineHeight(commandLine)
}

func (r *Renderer) CommandLineHeight(p *CommandLineView) int {
	if p == nil {
		return 1
	}
	switch {
	case r.H < 12:
		return 1
	case r.H < 18 || r.W < 72:
		return 2
	case r.H >= 34:
		return 7
	case r.H >= 26:
		return 6
	default:
		return 5
	}
}

func (r *Renderer) RenderFooterBlock(footerText string, commandLine *CommandLineView) string {
	if commandLine != nil {
		return r.RenderCommandLineWithHeight(*commandLine, r.CommandLineHeight(commandLine))
	}
	return r.RenderFooter(footerText)
}

func (r *Renderer) RenderCommandLineWithHeight(p CommandLineView, height int) string {
	if height <= 1 {
		return r.renderCommandPromptLine(p, true)
	}
	if height == 2 {
		return r.renderCommandSuggestionStrip(p) + "\n" + r.renderCommandPromptLine(p, false)
	}

	maxItems := height - 2
	items, offset := visibleCommandLineItems(p.Items, p.Cursor, maxItems)

	var lines []string
	nameW := 13
	for i, item := range items {
		sel := offset+i == p.Cursor
		prefix := "  "
		nameStyle := lipgloss.NewStyle().Foreground(r.Theme.Subtext)
		descStyle := lipgloss.NewStyle().Foreground(r.Theme.Overlay)
		if item.Danger {
			nameStyle = nameStyle.Foreground(r.Theme.Red)
		}
		if item.Disabled {
			nameStyle = nameStyle.Foreground(r.Theme.Overlay)
			descStyle = descStyle.Foreground(r.Theme.Surface0)
		}
		if sel {
			prefix = lipgloss.NewStyle().Foreground(r.Theme.Accent).Render(r.Icons.Selected + " ")
			nameStyle = nameStyle.Foreground(r.Theme.Accent).Bold(true)
			if item.Danger {
				nameStyle = nameStyle.Foreground(r.Theme.Red).Bold(true)
			}
			descStyle = descStyle.Foreground(r.Theme.Subtext)
		}
		desc := item.Description
		if item.Disabled && strings.TrimSpace(item.DisabledReason) != "" {
			desc = "disabled: " + item.DisabledReason
		}
		lines = append(lines, prefix+nameStyle.Width(nameW).Render(":"+r.TruncStr(item.Name, nameW-2))+" "+descStyle.Render(r.TruncStr(desc, max(8, r.PageContentWidth()-nameW-4))))
	}
	if len(lines) == 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render("  no commands"))
	}

	for len(lines) < height-2 {
		lines = append(lines, "")
	}
	if len(lines) > height-2 {
		lines = lines[:height-2]
	}

	lines = append(lines, lipgloss.NewStyle().Foreground(r.Theme.Surface0).Render(strings.Repeat(r.Icons.Rule, min(r.PageContentWidth(), 40))))
	lines = append(lines, r.renderCommandPromptLine(p, false))
	return strings.Join(lines, "\n")
}

func (r *Renderer) renderCommandPromptLine(p CommandLineView, compact bool) string {
	cursor := ""
	if r.Tick%2 == 0 {
		cursor = lipgloss.NewStyle().Foreground(r.Theme.Accent).Render(r.Icons.Cursor)
	} else {
		cursor = lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render(r.Icons.Cursor)
	}
	input := lipgloss.NewStyle().Foreground(r.Theme.Accent).Bold(true).Render(":") +
		lipgloss.NewStyle().Foreground(r.Theme.Text).Render(p.Query) + cursor
	if compact {
		hint := ""
		if item, ok := r.commandLineSelectedItem(p); ok {
			desc := item.Description
			if item.Disabled && strings.TrimSpace(item.DisabledReason) != "" {
				desc = "disabled: " + item.DisabledReason
			}
			if strings.TrimSpace(desc) != "" {
				hint = "  " + lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render("→ "+r.TruncStr(desc, max(8, r.PageContentWidth()-lipgloss.Width(p.Query)-8)))
			}
		}
		return input + hint
	}
	return input + "  " + lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render("enter run · tab complete · esc cancel")
}

func (r *Renderer) renderCommandSuggestionStrip(p CommandLineView) string {
	items := p.Items
	if len(items) == 0 {
		return lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render("no commands")
	}

	var parts []string
	availableW := r.PageContentWidth()
	window, offset := visibleCommandLineItems(items, p.Cursor, 4)
	for i, item := range window {
		style := lipgloss.NewStyle().Foreground(r.Theme.Subtext)
		if item.Danger {
			style = style.Foreground(r.Theme.Red)
		}
		if item.Disabled {
			style = style.Foreground(r.Theme.Overlay)
		}
		sel := offset+i == p.Cursor
		if sel {
			style = style.Foreground(r.Theme.Accent).Bold(true)
			if item.Danger {
				style = style.Foreground(r.Theme.Red).Bold(true)
			}
		}
		part := style.Render(":" + item.Name)
		if sel && strings.TrimSpace(item.Description) != "" {
			part += lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render(" " + r.TruncStr(item.Description, 22))
		}
		next := strings.Join(append(parts, part), "  ·  ")
		if lipgloss.Width(next) > availableW {
			if sel && len(parts) == 0 {
				parts = append(parts, r.TruncStr(":"+item.Name+" "+item.Description, availableW))
			}
			break
		}
		parts = append(parts, part)
	}
	if len(parts) == 0 {
		item, _ := r.commandLineSelectedItem(p)
		return r.TruncStr(":"+item.Name, availableW)
	}
	return strings.Join(parts, "  ·  ")
}

func visibleCommandLineItems(items []CommandLineItem, cursor int, limit int) ([]CommandLineItem, int) {
	if limit <= 0 || len(items) == 0 {
		return nil, 0
	}
	if cursor < 0 {
		cursor = 0
	}
	if cursor >= len(items) {
		cursor = len(items) - 1
	}
	if len(items) <= limit {
		return items, 0
	}
	start := cursor - limit + 1
	if start < 0 {
		start = 0
	}
	if start+limit > len(items) {
		start = len(items) - limit
	}
	return items[start : start+limit], start
}

func (r *Renderer) commandLineSelectedItem(p CommandLineView) (CommandLineItem, bool) {
	if p.Cursor < 0 || p.Cursor >= len(p.Items) {
		return CommandLineItem{}, false
	}
	return p.Items[p.Cursor], true
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
	if w <= 0 || utf8.RuneCountInString(s) <= w {
		return s
	}
	if w <= 1 {
		return r.Icons.Truncation
	}
	runes := []rune(s)
	return string(runes[:w-1]) + r.Icons.Truncation
}

func wrapPlainTextLines(text string, width int) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return []string{""}
	}
	if width <= 1 {
		return []string{text}
	}

	paragraphs := strings.Split(text, "\n")
	lines := make([]string, 0, len(paragraphs))
	for _, paragraph := range paragraphs {
		paragraph = strings.TrimSpace(paragraph)
		if paragraph == "" {
			lines = append(lines, "")
			continue
		}

		runes := []rune(paragraph)
		for len(runes) > 0 {
			if len(runes) <= width {
				lines = append(lines, string(runes))
				break
			}

			cut := width
			for i := cut; i > 0; i-- {
				if i < len(runes) && runes[i] == ' ' {
					cut = i
					break
				}
				if i > 0 && runes[i-1] == ' ' {
					cut = i - 1
					break
				}
			}
			if cut <= 0 {
				cut = width
			}

			line := strings.TrimSpace(string(runes[:cut]))
			if line == "" {
				line = string(runes[:min(width, len(runes))])
				cut = len([]rune(line))
			}
			lines = append(lines, line)

			next := cut
			for next < len(runes) && runes[next] == ' ' {
				next++
			}
			runes = runes[next:]
		}
	}

	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}

// RemoveLastRune removes the last rune from a string.
func RemoveLastRune(s string) string {
	if s == "" {
		return ""
	}
	runes := []rune(s)
	return string(runes[:len(runes)-1])
}
