package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// SettingsItem represents one row in the settings view.
type SettingsItem struct {
	Category string
	Label    string
	Value    string
	Kind     int      // 0=toggle, 1=enum, 2=action
	Options  []string // for enum kind
	OptIdx   int
	Disabled bool
}

// SettingsViewParams holds data for the settings page view.
type SettingsViewParams struct {
	Items        []SettingsItem
	Cursor       int
	Filter       string
	Searching    bool
	FilteredIdxs []int // indices into Items that match filter
	Page         int
	Err          error
	CommandLine  *CommandLineView
}

// RenderSettingsView renders the settings page.
func (r *Renderer) RenderSettingsView(p SettingsViewParams) string {
	cw := r.PageContentWidth()
	pad := r.LeftPad()

	headerLine := r.RenderHeader("settings", 0, 0)

	ruleW := cw
	if ruleW > 40 {
		ruleW = 40
	}
	rule := lipgloss.NewStyle().Foreground(r.Theme.Surface0).Render(strings.Repeat(r.Icons.Rule, ruleW))

	valueW := cw - 40
	if valueW < 16 {
		valueW = 16
	}
	if valueW > 28 {
		valueW = 28
	}
	labelW := cw - valueW - 8
	if labelW < 20 {
		labelW = 20
	}

	footerH := r.FooterBlockHeight(p.CommandLine)
	bodyH := r.H - 7 - footerH

	var lines []string
	lines = append(lines, headerLine)

	// search/filter bar
	if p.Searching || p.Filter != "" {
		filterStyle := lipgloss.NewStyle().Foreground(r.Theme.Overlay)
		inputStyle := lipgloss.NewStyle().Foreground(r.Theme.Text)
		cursor := ""
		if p.Searching && r.Tick%2 == 0 {
			cursor = lipgloss.NewStyle().Foreground(r.Theme.Accent).Render(r.Icons.Cursor)
		} else if p.Searching {
			cursor = lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render(r.Icons.Cursor)
		}
		var filterLine string
		if p.Filter == "" {
			filterLine = "  " + filterStyle.Render("filter...") + cursor
		} else {
			filterLine = "  " + inputStyle.Render(p.Filter) + cursor
		}
		filterSep := lipgloss.NewStyle().Foreground(r.Theme.Surface0).Render("  " + strings.Repeat(r.Icons.Rule, ruleW-4))
		lines = append(lines, filterLine)
		lines = append(lines, filterSep)
	} else {
		lines = append(lines, rule)
	}

	if p.Err != nil {
		lines = append(lines, r.renderErrLine(p.Err))
	}

	filtered := p.FilteredIdxs
	lastCat := ""

	// Build scrollable body lines separately from header/footer
	var bodyLines []string
	cursorLine := 0 // track which body line the cursor is on

	for _, i := range filtered {
		if i >= len(p.Items) {
			continue
		}
		s := p.Items[i]
		if s.Category != lastCat {
			bodyLines = append(bodyLines, "")
			catStyle := lipgloss.NewStyle().Foreground(r.Theme.Subtext)
			bodyLines = append(bodyLines, "  "+catStyle.Render(s.Category))
			lastCat = s.Category
		}

		if i == p.Cursor {
			cursorLine = len(bodyLines)
		}

		sel := i == p.Cursor

		marker := "    "
		lblStyle := lipgloss.NewStyle().Foreground(r.Theme.Subtext)
		valStyle := lipgloss.NewStyle().Foreground(r.Theme.Text)

		if sel {
			marker = lipgloss.NewStyle().Foreground(r.Theme.Accent).Render("  " + r.Icons.Selected + " ")
			lblStyle = lipgloss.NewStyle().Foreground(r.Theme.Accent)
			valStyle = lipgloss.NewStyle().Foreground(r.Theme.Accent).Bold(true)
		}

		// value display
		valDisplay := s.Value
		if s.Kind == 0 { // toggle
			if s.Value == "on" {
				valDisplay = lipgloss.NewStyle().Foreground(r.Theme.Green).Render("on")
			} else {
				valDisplay = valStyle.Render("off")
			}
		} else if s.Kind == 1 && sel { // enum, selected
			arrowW := valueW - 4
			if arrowW < 4 {
				arrowW = 4
			}
			valDisplay = lipgloss.NewStyle().Foreground(r.Theme.Accent).Render(r.Icons.LeftArrow) +
				" " + valStyle.Render(r.TruncStr(s.Value, arrowW)) +
				" " + lipgloss.NewStyle().Foreground(r.Theme.Accent).Render(r.Icons.RightArrow)
		} else {
			valDisplay = valStyle.Render(r.TruncStr(s.Value, valueW))
		}

		// build the row with right-aligned value
		label := lblStyle.Render(r.TruncStr(s.Label, labelW))
		gap := strings.Repeat(" ", max(1, labelW-lipgloss.Width(s.Label)))
		row := marker + label + gap + valDisplay
		bodyLines = append(bodyLines, row)
	}

	if len(filtered) == 0 && p.Filter != "" {
		bodyLines = append(bodyLines, "")
		bodyLines = append(bodyLines, "  "+lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render("no matching settings"))
	}

	// Scroll the body to keep cursor visible
	// Reserve lines for: header(1) + rule/filter(1-3) + bottom rule(1) + blank(1) + footer(1) + padding(2)
	maxBodyLines := bodyH
	if maxBodyLines < 4 {
		maxBodyLines = 4
	}

	scrollOffset := 0
	if cursorLine >= maxBodyLines {
		scrollOffset = cursorLine - maxBodyLines + 3
	}
	if scrollOffset > 0 && scrollOffset < len(bodyLines) {
		bodyLines = bodyLines[scrollOffset:]
	}
	if len(bodyLines) > maxBodyLines {
		bodyLines = bodyLines[:maxBodyLines]
	}

	lines = append(lines, bodyLines...)

	// Fixed footer — always visible
	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Foreground(r.Theme.Surface0).Render(strings.Repeat(r.Icons.Rule, ruleW)))
	footerHint := r.MainFooterText()
	if p.CommandLine == nil {
		if p.Searching {
			footerHint = "type to filter  enter confirm  esc clear"
		} else if p.Filter != "" {
			footerHint = "↑↓ navigate  / filter  : commands  esc clear  q home"
		}
	}
	lines = append(lines, r.RenderFooterBlock(footerHint, p.CommandLine))

	inner := strings.Join(lines, "\n")

	// Attach sidebar
	if r.ShowSidebar() {
		sBodyH := r.H - 4
		if sBodyH < 4 {
			sBodyH = 4
		}
		sidebar := r.RenderSidebar(sBodyH, p.Page)
		innerBlock := lipgloss.NewStyle().Width(cw).Render(inner)
		sideGap := lipgloss.NewStyle().Width(2).Render(strings.Repeat("\n", sBodyH))
		inner = lipgloss.JoinHorizontal(lipgloss.Top, innerBlock, sideGap, sidebar)
	}

	padded := r.PadContent(inner, pad)
	return padded
}

// ── Settings edit overlay ─────────────────────────────────────────────

// SettingsEditParams holds data for the settings text edit popup.
type SettingsEditParams struct {
	Label string
	Value string // current edit buffer
	Err   string
}

// RenderSettingsEditOverlay renders a centered text-edit modal for a settings field.
func (r *Renderer) RenderSettingsEditOverlay(p SettingsEditParams) string {
	bg := r.Theme.Mantle
	title := lipgloss.NewStyle().Foreground(r.Theme.Text).Background(bg).Bold(true).Render("edit setting")

	label := lipgloss.NewStyle().Foreground(r.Theme.Subtext).Background(bg).Render(p.Label)

	cursor := ""
	if r.Tick%2 == 0 {
		cursor = lipgloss.NewStyle().Foreground(r.Theme.Accent).Background(bg).Render(r.Icons.Cursor)
	} else {
		cursor = lipgloss.NewStyle().Foreground(r.Theme.Overlay).Background(bg).Render(r.Icons.Cursor)
	}
	inputLine := lipgloss.NewStyle().Foreground(r.Theme.Text).Background(bg).Render(p.Value) + cursor

	barW := 36
	bar := lipgloss.NewStyle().Foreground(r.Theme.Accent).Background(bg).Render(strings.Repeat("─", barW))

	var parts []string
	parts = append(parts, title, "", label, inputLine, bar)

	if p.Err != "" {
		parts = append(parts, "")
		parts = append(parts, lipgloss.NewStyle().Foreground(r.Theme.Yellow).Background(bg).Render(p.Err))
	}

	hint := lipgloss.NewStyle().Foreground(r.Theme.Overlay).Background(bg).Render("enter save · esc cancel")
	parts = append(parts, "", hint)

	content := strings.Join(parts, "\n")

	box := lipgloss.NewStyle().
		Width(42).
		Background(bg).
		Padding(1, 2).
		Render(content)

	return lipgloss.Place(r.W, r.H, lipgloss.Center, lipgloss.Center,
		box,
		lipgloss.WithWhitespaceBackground(r.Theme.Base))
}

// ── Token overlay types ───────────────────────────────────────────────

// TokenHostItem represents one host row in the token host picker.
type TokenHostItem struct {
	ID       string
	Label    string
	Detail   string // user@host:port
	Selected bool
}

// TokenCreateNameParams holds data for the token name input overlay.
type TokenCreateNameParams struct {
	NameValue string
	Err       string
}

// TokenSelectHostsParams holds data for the host picker overlay.
type TokenSelectHostsParams struct {
	Hosts  []TokenHostItem
	Cursor int
	Err    string
}

// TokenRevealParams holds data for the token reveal/copy overlay.
type TokenRevealParams struct {
	TokenValue string
	Copied     bool
}

// ── Tokens view ───────────────────────────────────────────────────────

// TokenViewItem represents one token in the tokens page.
type TokenViewItem struct {
	Name    string
	Scope   string
	Created string
	LastUse string
}

// TokensViewParams holds data for the tokens page view.
type TokensViewParams struct {
	Tokens      []TokenViewItem
	Cursor      int
	Page        int
	Err         error
	CommandLine *CommandLineView
}

// RenderTokensView renders the tokens page.
func (r *Renderer) RenderTokensView(p TokensViewParams) string {
	cw := r.PageContentWidth()
	pad := r.LeftPad()

	headerLine := r.RenderHeader("tokens", 0, 0)

	ruleW := cw
	if ruleW > 40 {
		ruleW = 40
	}

	footerH := r.FooterBlockHeight(p.CommandLine)
	bodyH := r.H - 5 - footerH
	if bodyH < 4 {
		bodyH = 4
	}

	var lines []string
	lines = append(lines, headerLine)
	lines = append(lines, "")

	if p.Err != nil {
		lines = append(lines, r.renderErrLine(p.Err))
	}

	if len(p.Tokens) == 0 {
		lines = append(lines, "  "+lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render("no tokens"))
		lines = append(lines, "")
		lines = append(lines, "  "+lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render("press a to create one"))
	} else {
		for i, tok := range p.Tokens {
			sel := i == p.Cursor

			nameStyle := lipgloss.NewStyle().Foreground(r.Theme.Text).Bold(true)
			prefix := "  "
			if sel {
				nameStyle = lipgloss.NewStyle().Foreground(r.Theme.Accent).Bold(true)
				prefix = lipgloss.NewStyle().Foreground(r.Theme.Accent).Render(r.Icons.Selected + " ")
			}

			lines = append(lines, prefix+nameStyle.Render(tok.Name))

			// scope + created on same line
			scopeStr := lipgloss.NewStyle().Foreground(r.Theme.Subtext).Render("scope: " + tok.Scope)
			createdStr := lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render("created " + tok.Created)
			scopeW := lipgloss.Width("scope: " + tok.Scope)
			gapN := ruleW - scopeW - lipgloss.Width("created "+tok.Created) - 4
			if gapN < 2 {
				gapN = 2
			}
			lines = append(lines, "  "+scopeStr+strings.Repeat(" ", gapN)+createdStr)

			// last use
			if tok.LastUse == "never" {
				lines = append(lines, "  "+lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render("never used"))
			} else {
				lines = append(lines, "  "+lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render("last used "+tok.LastUse))
			}

			// separator
			sepW := ruleW
			if sepW > 20 {
				sepW = 20
			}
			lines = append(lines, "  "+lipgloss.NewStyle().Foreground(r.Theme.Surface0).Render(strings.Repeat(r.Icons.Rule, sepW)))
			lines = append(lines, "")
		}
	}

	maxContentLines := bodyH + 2
	if len(lines) > maxContentLines {
		lines = lines[:maxContentLines]
	}
	lines = append(lines, r.RenderFooterBlock(r.MainFooterText(), p.CommandLine))

	inner := strings.Join(lines, "\n")

	// Attach sidebar
	if r.ShowSidebar() {
		sidebar := r.RenderSidebar(bodyH, p.Page)
		innerBlock := lipgloss.NewStyle().Width(cw).Render(inner)
		sideGap := lipgloss.NewStyle().Width(2).Render(strings.Repeat("\n", bodyH))
		inner = lipgloss.JoinHorizontal(lipgloss.Top, innerBlock, sideGap, sidebar)
	}

	padded := r.PadContent(inner, pad)
	return padded
}

// ── Token overlays ────────────────────────────────────────────────────

// RenderTokenCreateNameOverlay renders the token name input modal.
func (r *Renderer) RenderTokenCreateNameOverlay(p TokenCreateNameParams) string {
	bg := r.Theme.Mantle
	title := lipgloss.NewStyle().Foreground(r.Theme.Text).Background(bg).Bold(true).Render("create token")

	label := lipgloss.NewStyle().Foreground(r.Theme.Subtext).Background(bg).Render("name")

	inputVal := p.NameValue
	cursor := ""
	if r.Tick%2 == 0 {
		cursor = lipgloss.NewStyle().Foreground(r.Theme.Accent).Background(bg).Render(r.Icons.Cursor)
	} else {
		cursor = lipgloss.NewStyle().Foreground(r.Theme.Overlay).Background(bg).Render(r.Icons.Cursor)
	}
	inputLine := lipgloss.NewStyle().Foreground(r.Theme.Text).Background(bg).Render(inputVal) + cursor

	barW := 36
	bar := lipgloss.NewStyle().Foreground(r.Theme.Accent).Background(bg).Render(strings.Repeat("─", barW))

	var parts []string
	parts = append(parts, title, "", label, inputLine, bar)

	if p.Err != "" {
		parts = append(parts, "")
		parts = append(parts, lipgloss.NewStyle().Foreground(r.Theme.Yellow).Background(bg).Render(p.Err))
	}

	hint := lipgloss.NewStyle().Foreground(r.Theme.Overlay).Background(bg).Render("enter confirm · esc cancel")
	parts = append(parts, "", hint)

	content := strings.Join(parts, "\n")

	box := lipgloss.NewStyle().
		Width(42).
		Background(bg).
		Padding(1, 2).
		Render(content)

	return lipgloss.Place(r.W, r.H, lipgloss.Center, lipgloss.Center,
		box,
		lipgloss.WithWhitespaceBackground(r.Theme.Base))
}

// RenderTokenSelectHostsOverlay renders the host picker modal.
func (r *Renderer) RenderTokenSelectHostsOverlay(p TokenSelectHostsParams) string {
	bg := r.Theme.Mantle
	title := lipgloss.NewStyle().Foreground(r.Theme.Text).Background(bg).Bold(true).Render("select hosts")
	subtitle := lipgloss.NewStyle().Foreground(r.Theme.Subtext).Background(bg).Render("space to toggle, enter to create")

	var parts []string
	parts = append(parts, title, subtitle, "")

	maxVisible := r.H - 14
	if maxVisible < 4 {
		maxVisible = 4
	}

	scrollOff := 0
	if p.Cursor > maxVisible-1 {
		scrollOff = p.Cursor - maxVisible + 1
	}

	displayed := 0
	for i, h := range p.Hosts {
		if i < scrollOff {
			continue
		}
		if displayed >= maxVisible {
			break
		}

		sel := i == p.Cursor
		check := r.Icons.Offline
		checkStyle := lipgloss.NewStyle().Foreground(r.Theme.Surface0).Background(bg)
		if h.Selected {
			check = r.Icons.Connected
			checkStyle = lipgloss.NewStyle().Foreground(r.Theme.Green).Background(bg)
		}

		nameStyle := lipgloss.NewStyle().Foreground(r.Theme.Subtext).Background(bg)
		detailStyle := lipgloss.NewStyle().Foreground(r.Theme.Overlay).Background(bg)
		prefix := "  "
		if sel {
			nameStyle = lipgloss.NewStyle().Foreground(r.Theme.Accent).Background(bg).Bold(true)
			detailStyle = lipgloss.NewStyle().Foreground(r.Theme.Accent).Background(bg)
			prefix = lipgloss.NewStyle().Foreground(r.Theme.Accent).Background(bg).Render(r.Icons.Focused + " ")
		}

		line := prefix + checkStyle.Render(check) + " " + nameStyle.Render(h.Label)
		if h.Detail != "" {
			line += "  " + detailStyle.Render(h.Detail)
		}
		parts = append(parts, line)
		displayed++
	}

	if len(p.Hosts) == 0 {
		parts = append(parts, "  "+lipgloss.NewStyle().Foreground(r.Theme.Overlay).Background(bg).Render("no hosts available"))
	}

	if p.Err != "" {
		parts = append(parts, "")
		parts = append(parts, lipgloss.NewStyle().Foreground(r.Theme.Yellow).Background(bg).Render(p.Err))
	}

	hint := lipgloss.NewStyle().Foreground(r.Theme.Overlay).Background(bg).Render("↑↓ navigate · space toggle · enter create · esc cancel")
	parts = append(parts, "", hint)

	content := strings.Join(parts, "\n")

	boxW := 56
	if r.W < 60 {
		boxW = r.W - 6
	}
	if boxW < 30 {
		boxW = 30
	}

	box := lipgloss.NewStyle().
		Width(boxW).
		Background(bg).
		Padding(1, 2).
		Render(content)

	return lipgloss.Place(r.W, r.H, lipgloss.Center, lipgloss.Center,
		box,
		lipgloss.WithWhitespaceBackground(r.Theme.Base))
}

// RenderTokenRevealOverlay renders the token reveal/copy modal.
func (r *Renderer) RenderTokenRevealOverlay(p TokenRevealParams) string {
	bg := r.Theme.Mantle
	title := lipgloss.NewStyle().Foreground(r.Theme.Text).Background(bg).Bold(true).Render("token created")

	warn := lipgloss.NewStyle().Foreground(r.Theme.Yellow).Background(bg).Render(r.Icons.Warning + " copy this token now — it won't be shown again")

	boxW := 56
	if r.W < 60 {
		boxW = r.W - 6
	}
	if boxW < 30 {
		boxW = 30
	}
	innerW := boxW - 6

	tokenVal := p.TokenValue
	var tokenLines []string
	for len(tokenVal) > innerW {
		tokenLines = append(tokenLines, tokenVal[:innerW])
		tokenVal = tokenVal[innerW:]
	}
	if len(tokenVal) > 0 {
		tokenLines = append(tokenLines, tokenVal)
	}

	tokenDisplay := lipgloss.NewStyle().Foreground(r.Theme.Accent).Background(bg).Render(strings.Join(tokenLines, "\n"))

	var parts []string
	parts = append(parts, title, "", warn, "", tokenDisplay, "")

	if p.Copied {
		parts = append(parts, lipgloss.NewStyle().Foreground(r.Theme.Green).Background(bg).Render("✓ copied to clipboard"))
	} else {
		parts = append(parts, lipgloss.NewStyle().Foreground(r.Theme.Overlay).Background(bg).Render("press c to copy to clipboard"))
	}

	hint := lipgloss.NewStyle().Foreground(r.Theme.Overlay).Background(bg).Render("c copy · esc close")
	parts = append(parts, "", hint)

	content := strings.Join(parts, "\n")

	box := lipgloss.NewStyle().
		Width(boxW).
		Background(bg).
		Padding(1, 2).
		Render(content)

	return lipgloss.Place(r.W, r.H, lipgloss.Center, lipgloss.Center,
		box,
		lipgloss.WithWhitespaceBackground(r.Theme.Base))
}
