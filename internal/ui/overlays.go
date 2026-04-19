package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ── Quit overlay ──────────────────────────────────────────────────────

// QuitViewParams holds data needed to render the quit confirmation overlay.
type QuitViewParams struct {
	Mounts     []string // list of active mount labels
	QuitCursor int      // 0,1,2 button selection
}

// RenderQuitOverlay renders the quit confirmation overlay.
func (r *Renderer) RenderQuitOverlay(p QuitViewParams) string {
	bg := r.Theme.Mantle
	title := lipgloss.NewStyle().Foreground(r.Theme.Text).Background(bg).Bold(true).Render("quit sshthing?")

	btnStyle := func(label string, idx int) string {
		s := lipgloss.NewStyle().Foreground(r.Theme.Subtext).Background(bg).Padding(0, 2)
		if p.QuitCursor == idx {
			s = s.Foreground(r.Theme.Base).Background(r.Theme.Accent).Bold(true)
		}
		return s.Render(label)
	}

	var contentParts []string
	contentParts = append(contentParts, title)

	if len(p.Mounts) > 0 {
		contentParts = append(contentParts, "")
		mountLabel := lipgloss.NewStyle().Foreground(r.Theme.Yellow).Background(bg).Render(r.Icons.Warning + " active mounts:")
		contentParts = append(contentParts, mountLabel)
		for _, mt := range p.Mounts {
			contentParts = append(contentParts, "  "+lipgloss.NewStyle().Foreground(r.Theme.Subtext).Background(bg).Render(mt))
		}
		contentParts = append(contentParts, "")
		buttons := btnStyle("unmount & quit", 0) + "  " + btnStyle("leave mounted", 1) + "  " + btnStyle("cancel", 2)
		contentParts = append(contentParts, buttons)
	} else {
		hint := lipgloss.NewStyle().Foreground(r.Theme.Overlay).Background(bg).Render("are you sure you want to exit?")
		contentParts = append(contentParts, "")
		contentParts = append(contentParts, hint)
		contentParts = append(contentParts, "")
		buttons := btnStyle("yes", 0) + "  " + btnStyle("cancel", 1)
		contentParts = append(contentParts, buttons)
	}

	footer := lipgloss.NewStyle().Foreground(r.Theme.Overlay).Background(bg).Render("\u2190\u2192 select \u00B7 enter confirm \u00B7 esc cancel")
	contentParts = append(contentParts, "")
	contentParts = append(contentParts, footer)

	content := strings.Join(contentParts, "\n")

	boxW := 50
	if len(p.Mounts) == 0 {
		boxW = 40
	}

	box := lipgloss.NewStyle().
		Width(boxW).
		Background(bg).
		Padding(1, 2).
		Align(lipgloss.Center).
		Render(content)

	return lipgloss.Place(r.W, r.H, lipgloss.Center, lipgloss.Center,
		box,
		lipgloss.WithWhitespaceBackground(r.Theme.Base))
}

// ── Help overlay ──────────────────────────────────────────────────────

// RenderHelpOverlay renders the keyboard shortcuts help overlay.
func (r *Renderer) RenderHelpOverlay() string {
	title := lipgloss.NewStyle().Foreground(r.Theme.Text).Bold(true).Render("shortcuts")
	pairs := [][2]string{
		{"\u2191 \u2193  j k", "navigate"},
		{"enter", "connect or toggle"},
		{"/", "search"},
		{"S", "sftp"},
		{"M", "mount / unmount"},
		{"Y", "sync now"},
		{",", "settings"},
		{"ctrl+g", "new group"},
		{"a", "add host"},
		{"e", "edit"},
		{"d", "delete"},
		{"shift+tab", "switch page"},
		{"?", "help"},
		{"q", "quit"},
	}

	kS := lipgloss.NewStyle().Foreground(r.Theme.Accent).Width(16).Align(lipgloss.Right)
	vS := lipgloss.NewStyle().Foreground(r.Theme.Subtext)
	var lines []string
	for _, p := range pairs {
		lines = append(lines, kS.Render(p[0])+"    "+vS.Render(p[1]))
	}

	content := title + "\n\n" + strings.Join(lines, "\n") + "\n\n" +
		lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render("any key to close")

	return lipgloss.Place(r.W, r.H, lipgloss.Center, lipgloss.Center,
		lipgloss.NewStyle().Padding(2, 4).Render(content),
		lipgloss.WithWhitespaceBackground(r.Theme.Base))
}

// ── Search overlay ────────────────────────────────────────────────────

// SearchResultItem represents one row in the search results.
type SearchResultItem struct {
	Label       string
	Hostname    string
	GroupName   string
	Status      int // 0=offline, 1=idle, 2=connected
	CommandMode bool
}

// SearchViewParams holds data for the search overlay.
type SearchViewParams struct {
	Query        string
	Cursor       int
	Results      []SearchResultItem
	ArmedSFTP    bool
	ArmedMount   bool
	ArmedUnmount bool
	CommandMode  bool
}

// RenderSearchOverlay renders the search/spotlight overlay.
func (r *Renderer) RenderSearchOverlay(p SearchViewParams) string {
	searchW := 56
	if searchW > r.W-8 {
		searchW = r.W - 8
	}
	if searchW < 30 {
		searchW = 30
	}

	bg := r.Theme.Mantle
	inputStyle := lipgloss.NewStyle().Foreground(r.Theme.Text).Background(bg)
	placeholderText := "search hosts..."
	if p.CommandMode {
		placeholderText = "> commands..."
	}
	placeholder := lipgloss.NewStyle().Foreground(r.Theme.Overlay).Background(bg).Render(placeholderText)
	inputText := p.Query
	cursor := ""
	if r.Tick%2 == 0 {
		cursor = lipgloss.NewStyle().Foreground(r.Theme.Accent).Background(bg).Render(r.Icons.Cursor)
	} else {
		cursor = lipgloss.NewStyle().Foreground(r.Theme.Overlay).Background(bg).Render(r.Icons.Cursor)
	}

	var inputLine string
	if inputText == "" {
		inputLine = "  " + placeholder + cursor
	} else {
		inputLine = "  " + inputStyle.Render(inputText) + cursor
	}

	sep := lipgloss.NewStyle().Foreground(r.Theme.Surface0).Background(bg).Render("  " + strings.Repeat(r.Icons.Rule, searchW-4))

	results := p.Results
	maxResults := 8
	if len(results) > maxResults {
		results = results[:maxResults]
	}

	var resultLines []string
	for i, h := range results {
		sel := i == p.Cursor
		if h.CommandMode {
			nameStyle := lipgloss.NewStyle().Foreground(r.Theme.Subtext).Background(bg)
			prefix := "    "
			if sel {
				nameStyle = lipgloss.NewStyle().Foreground(r.Theme.Accent).Background(bg).Bold(true)
				prefix = lipgloss.NewStyle().Foreground(r.Theme.Accent).Background(bg).Render("  " + r.Icons.Selected + " ")
			}
			resultLines = append(resultLines, prefix+nameStyle.Render(h.Label))
			continue
		}
		var dot string
		switch h.Status {
		case 2:
			dot = lipgloss.NewStyle().Foreground(r.Theme.Green).Background(bg).Render(r.Icons.Connected)
		case 1:
			dot = lipgloss.NewStyle().Foreground(r.Theme.Yellow).Background(bg).Render(r.Icons.Idle)
		default:
			dot = lipgloss.NewStyle().Foreground(r.Theme.Surface0).Background(bg).Render(r.Icons.Offline)
		}

		nameStyle := lipgloss.NewStyle().Foreground(r.Theme.Subtext).Background(bg)
		groupHint := lipgloss.NewStyle().Foreground(r.Theme.Overlay).Background(bg).Render(" " + h.GroupName)
		prefix := "    "
		if sel {
			nameStyle = lipgloss.NewStyle().Foreground(r.Theme.Accent).Background(bg).Bold(true)
			prefix = lipgloss.NewStyle().Foreground(r.Theme.Accent).Background(bg).Render("  " + r.Icons.Selected + " ")
			groupHint = lipgloss.NewStyle().Foreground(r.Theme.Subtext).Background(bg).Render(" " + h.GroupName)
		}
		lbl := h.Label
		if lbl == "" {
			lbl = h.Hostname
		}
		resultLines = append(resultLines, prefix+dot+" "+nameStyle.Render(lbl)+groupHint)
	}

	if len(results) == 0 && p.Query != "" {
		resultLines = append(resultLines, "  "+lipgloss.NewStyle().Foreground(r.Theme.Overlay).Background(bg).Render("no matches"))
	}

	var footerText string
	if p.ArmedMount {
		footerText = "esc close  \u00B7  enter mount  \u00B7  M disarm"
	} else if p.ArmedUnmount {
		footerText = "esc close  \u00B7  enter unmount  \u00B7  M disarm"
	} else if p.ArmedSFTP {
		footerText = "esc close  \u00B7  enter sftp  \u00B7  S disarm"
	} else {
		footerText = "esc close  \u00B7  enter connect  \u00B7  S sftp  \u00B7  M mount"
	}
	if p.CommandMode {
		footerText = "esc close  ·  enter run command"
	}
	footer := lipgloss.NewStyle().Foreground(r.Theme.Overlay).Background(bg).Render("  " + footerText)

	var contentParts []string
	contentParts = append(contentParts, inputLine)
	contentParts = append(contentParts, sep)
	if len(resultLines) > 0 {
		contentParts = append(contentParts, "")
		contentParts = append(contentParts, resultLines...)
	}
	contentParts = append(contentParts, "")
	contentParts = append(contentParts, footer)

	overlayContent := strings.Join(contentParts, "\n")

	overlayBox := lipgloss.NewStyle().
		Width(searchW).
		Background(r.Theme.Mantle).
		Padding(1, 0).
		Render(overlayContent)

	return lipgloss.Place(r.W, r.H, lipgloss.Center, lipgloss.Center,
		overlayBox,
		lipgloss.WithWhitespaceBackground(r.Theme.Base))
}

// ── Add/Edit host overlay ─────────────────────────────────────────────

// AddHostViewParams holds data for the add/edit host form overlay.
type AddHostViewParams struct {
	IsEdit      bool
	Fields      []FormField // [label, tags, hostname, port, username, authDetail]
	Focus       int
	Editing     bool
	Groups      []string
	GroupIdx    int
	AuthOptions []string
	AuthIdx     int
	KeyTypes    []string
	KeyTypeIdx  int
	Err         error
}

// Form field indices for add host.
const (
	FFLabel    = 0
	FFTags     = 1
	FFHostname = 2
	FFPort     = 3
	FFUsername = 4
	FFAuthDet  = 5
	FFGroup    = 100 // selector, not a text field
	FFAuthMeth = 101 // selector, not a text field
	FFSave     = 102 // button
)

// RenderAddHostOverlay renders the add/edit host form as a full-page overlay.
func (r *Renderer) RenderAddHostOverlay(p AddHostViewParams) string {
	cw := r.PageContentWidth()
	pad := r.LeftPad()
	compact := r.H < 30

	titleText := "add new host"
	if p.IsEdit {
		titleText = r.Icons.Edit + " edit host"
	}
	headerLine := r.RenderHeader(titleText, 0, 0)

	ruleW := cw
	if ruleW > 40 {
		ruleW = 40
	}
	rule := lipgloss.NewStyle().Foreground(r.Theme.Surface0).Render(strings.Repeat(r.Icons.Rule, ruleW))

	formW := cw
	if formW > 70 {
		formW = 70
	}

	blink := r.Tick%2 == 0

	spacer := func() string {
		if compact {
			return ""
		}
		return "\n"
	}

	var lines []string
	lines = append(lines, headerLine)
	lines = append(lines, rule)

	// label
	lines = append(lines, spacer()+r.RenderFormLabel("label", p.Focus == FFLabel))
	lines = append(lines, r.RenderInput(p.Fields[FFLabel], p.Focus == FFLabel, formW-4, blink, p.Editing))

	// group selector
	lines = append(lines, spacer()+r.RenderFormLabel("group", p.Focus == FFGroup))
	gName := ""
	if len(p.Groups) > 0 {
		gName = p.Groups[p.GroupIdx]
	}
	gCount := fmt.Sprintf("[%d/%d]", p.GroupIdx+1, len(p.Groups))
	if p.Focus == FFGroup {
		lines = append(lines, "  "+
			lipgloss.NewStyle().Foreground(r.Theme.Accent).Render(r.Icons.LeftArrow)+
			" "+lipgloss.NewStyle().Foreground(r.Theme.Text).Render(gName)+
			" "+lipgloss.NewStyle().Foreground(r.Theme.Accent).Render(r.Icons.RightArrow)+
			"  "+lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render(gCount))
	} else {
		lines = append(lines, "  "+
			lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render(r.Icons.LeftArrow)+
			" "+lipgloss.NewStyle().Foreground(r.Theme.Subtext).Render(gName)+
			" "+lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render(r.Icons.RightArrow)+
			"  "+lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render(gCount))
	}

	// tags
	lines = append(lines, spacer()+r.RenderFormLabel("tags", p.Focus == FFTags))
	lines = append(lines, r.RenderInput(p.Fields[FFTags], p.Focus == FFTags, formW-4, blink, p.Editing))
	if !compact {
		lines = append(lines, lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render("  comma-separated"))
	}

	// hostname + port side by side
	hostW := formW - 20
	if hostW < 20 {
		hostW = 20
	}
	portW := 12

	hostLabel := r.RenderFormLabel("hostname", p.Focus == FFHostname)
	portLabel := r.RenderFormLabel("port", p.Focus == FFPort)

	if r.W >= 60 {
		if !compact {
			lines = append(lines, "")
		}
		labelRow := lipgloss.NewStyle().Width(hostW).Render(hostLabel) +
			lipgloss.NewStyle().Width(portW).Render(portLabel)
		lines = append(lines, labelRow)

		hostInput := r.RenderInput(p.Fields[FFHostname], p.Focus == FFHostname, hostW-4, blink, p.Editing)
		portInput := r.RenderInput(p.Fields[FFPort], p.Focus == FFPort, portW-4, blink, p.Editing)
		inputRow := lipgloss.NewStyle().Width(hostW).Render(hostInput) +
			lipgloss.NewStyle().Width(portW).Render(portInput)
		lines = append(lines, inputRow)
	} else {
		lines = append(lines, spacer()+hostLabel)
		lines = append(lines, r.RenderInput(p.Fields[FFHostname], p.Focus == FFHostname, formW-4, blink, p.Editing))
		lines = append(lines, spacer()+portLabel)
		lines = append(lines, r.RenderInput(p.Fields[FFPort], p.Focus == FFPort, formW-4, blink, p.Editing))
	}

	// username
	lines = append(lines, spacer()+r.RenderFormLabel("username", p.Focus == FFUsername))
	lines = append(lines, r.RenderInput(p.Fields[FFUsername], p.Focus == FFUsername, formW-4, blink, p.Editing))

	// auth method selector
	lines = append(lines, spacer()+r.RenderFormLabel("authentication", p.Focus == FFAuthMeth))
	aName := ""
	if len(p.AuthOptions) > 0 {
		aName = p.AuthOptions[p.AuthIdx]
	}
	if p.Focus == FFAuthMeth {
		lines = append(lines, "  "+
			lipgloss.NewStyle().Foreground(r.Theme.Accent).Render(r.Icons.LeftArrow)+
			" "+lipgloss.NewStyle().Foreground(r.Theme.Text).Render(aName)+
			" "+lipgloss.NewStyle().Foreground(r.Theme.Accent).Render(r.Icons.RightArrow))
	} else {
		lines = append(lines, "  "+
			lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render(r.Icons.LeftArrow)+
			" "+lipgloss.NewStyle().Foreground(r.Theme.Subtext).Render(aName)+
			" "+lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render(r.Icons.RightArrow))
	}

	// auth detail based on method
	switch p.AuthIdx {
	case 0: // password
		lines = append(lines, spacer()+r.RenderFormLabel("password", p.Focus == FFAuthDet))
		lines = append(lines, r.RenderInput(p.Fields[FFAuthDet], p.Focus == FFAuthDet, formW-4, blink, p.Editing))
	case 1: // paste key
		lines = append(lines, spacer()+r.RenderFormLabel("private key", p.Focus == FFAuthDet))
		lines = append(lines, r.RenderInput(p.Fields[FFAuthDet], p.Focus == FFAuthDet, formW-4, blink, p.Editing))
		if !compact {
			lines = append(lines, lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render("  paste your private key"))
		}
	case 2: // generate
		lines = append(lines, spacer()+r.RenderFormLabel("key type", false))
		kt := ""
		if len(p.KeyTypes) > 0 {
			kt = p.KeyTypes[p.KeyTypeIdx]
		}
		lines = append(lines, "  "+lipgloss.NewStyle().Foreground(r.Theme.Text).Render(kt)+
			"  "+lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render("(space to cycle)"))
	}

	// error line
	if p.Err != nil {
		lines = append(lines, "")
		lines = append(lines, r.renderErrLine(p.Err))
	}

	// save button
	saveLabel := "save host"
	if p.IsEdit {
		saveLabel = "update host"
	}
	lines = append(lines, "")
	if p.Focus == FFSave {
		lines = append(lines, "  "+lipgloss.NewStyle().Foreground(r.Theme.Accent).Bold(true).Render(r.Icons.Save+" "+saveLabel))
	} else {
		lines = append(lines, "  "+lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render("  "+saveLabel))
	}

	// footer
	if !compact {
		lines = append(lines, "")
	}
	var footerText string
	if p.Editing {
		footerText = "type to edit  \u00B7  \u2191\u2193 leave  \u00B7  \u2190\u2192 cursor  \u00B7  esc done  \u00B7  tab next"
	} else {
		footerText = "\u2191\u2193\u2190\u2192 navigate  \u00B7  enter edit  \u00B7  tab next  \u00B7  esc cancel"
	}
	footer := r.RenderFooter(footerText)
	lines = append(lines, footer)

	inner := strings.Join(lines, "\n")
	padded := r.PadContent(inner, pad)

	return padded
}

// ── Login overlay ─────────────────────────────────────────────────────

// LoginViewParams holds data for the login overlay.
type LoginViewParams struct {
	Password FormField
	Err      string
}

// RenderLoginOverlay renders the password-unlock login overlay.
func (r *Renderer) RenderLoginOverlay(p LoginViewParams) string {
	bg := r.Theme.Mantle
	blink := r.Tick%2 == 0

	title := lipgloss.NewStyle().Foreground(r.Theme.Accent).Background(bg).Bold(true).
		Render(r.Icons.Lock + " sshthing")

	label := lipgloss.NewStyle().Foreground(r.Theme.Overlay).Background(bg).Render("  password")
	input := r.RenderModalField(p.Password.Value, p.Password.Cursor, true, true, blink, bg)

	var errLine string
	if p.Err != "" {
		errLine = lipgloss.NewStyle().Foreground(r.Theme.Red).Background(bg).
			Render("  " + r.Icons.ErrorIcon + " " + p.Err)
	}

	footer := lipgloss.NewStyle().Foreground(r.Theme.Overlay).Background(bg).
		Render("enter unlock  \u00B7  esc quit")

	var contentParts []string
	contentParts = append(contentParts, title, "", label, input)
	if errLine != "" {
		contentParts = append(contentParts, "", errLine)
	}
	contentParts = append(contentParts, "", footer)

	content := strings.Join(contentParts, "\n")

	box := lipgloss.NewStyle().
		Width(40).
		Background(bg).
		Padding(1, 2).
		Align(lipgloss.Center).
		Render(content)

	return lipgloss.Place(r.W, r.H, lipgloss.Center, lipgloss.Center, box,
		lipgloss.WithWhitespaceBackground(r.Theme.Base))
}

// ── Setup overlay ─────────────────────────────────────────────────────

// SetupViewParams holds data for the first-time setup overlay.
type SetupViewParams struct {
	Password FormField
	Confirm  FormField
	Focus    int // 0=password, 1=confirm, 2=submit
	Err      string
}

// RenderSetupOverlay renders the first-time setup overlay.
func (r *Renderer) RenderSetupOverlay(p SetupViewParams) string {
	bg := r.Theme.Mantle
	blink := r.Tick%2 == 0

	title := lipgloss.NewStyle().Foreground(r.Theme.Accent).Background(bg).Bold(true).
		Render(r.Icons.Shield + " first-time setup")

	pwLabel := lipgloss.NewStyle().Foreground(r.Theme.Overlay).Background(bg).Render("  master password")
	pwInput := r.RenderModalField(p.Password.Value, p.Password.Cursor, true, p.Focus == 0, blink, bg)

	cfLabel := lipgloss.NewStyle().Foreground(r.Theme.Overlay).Background(bg).Render("  confirm password")
	cfInput := r.RenderModalField(p.Confirm.Value, p.Confirm.Cursor, true, p.Focus == 1, blink, bg)

	var submitLine string
	if p.Focus == 2 {
		submitLine = "  " + lipgloss.NewStyle().Foreground(r.Theme.Accent).Background(bg).Bold(true).
			Render(r.Icons.Save+" create vault")
	} else {
		submitLine = "  " + lipgloss.NewStyle().Foreground(r.Theme.Overlay).Background(bg).
			Render("  create vault")
	}

	var errLine string
	if p.Err != "" {
		errLine = lipgloss.NewStyle().Foreground(r.Theme.Red).Background(bg).
			Render("  " + r.Icons.ErrorIcon + " " + p.Err)
	}

	footer := lipgloss.NewStyle().Foreground(r.Theme.Overlay).Background(bg).
		Render("tab next  \u00B7  enter submit  \u00B7  esc quit")

	var contentParts []string
	contentParts = append(contentParts, title, "", pwLabel, pwInput, "", cfLabel, cfInput)
	if errLine != "" {
		contentParts = append(contentParts, "", errLine)
	}
	contentParts = append(contentParts, "", submitLine, "", footer)

	content := strings.Join(contentParts, "\n")

	box := lipgloss.NewStyle().
		Width(50).
		Background(bg).
		Padding(1, 2).
		Align(lipgloss.Center).
		Render(content)

	return lipgloss.Place(r.W, r.H, lipgloss.Center, lipgloss.Center, box,
		lipgloss.WithWhitespaceBackground(r.Theme.Base))
}

// ── Delete host overlay ───────────────────────────────────────────────

// DeleteHostViewParams holds data for the delete host confirmation overlay.
type DeleteHostViewParams struct {
	Label        string
	Hostname     string
	Username     string
	DeleteCursor int // 0=delete, 1=cancel
}

// RenderDeleteHostOverlay renders the delete host confirmation overlay.
func (r *Renderer) RenderDeleteHostOverlay(p DeleteHostViewParams) string {
	bg := r.Theme.Mantle

	title := lipgloss.NewStyle().Foreground(r.Theme.Red).Background(bg).Bold(true).
		Render(r.Icons.Warning + " delete host")

	hostLabel := lipgloss.NewStyle().Foreground(r.Theme.Text).Background(bg).Bold(true).Render(p.Label)
	connStr := lipgloss.NewStyle().Foreground(r.Theme.Subtext).Background(bg).
		Render(p.Username + "@" + p.Hostname)
	warn := lipgloss.NewStyle().Foreground(r.Theme.Red).Background(bg).
		Render("cannot be undone!")

	delStyle := lipgloss.NewStyle().Foreground(r.Theme.Subtext).Background(bg).Padding(0, 2)
	cancelStyle := lipgloss.NewStyle().Foreground(r.Theme.Subtext).Background(bg).Padding(0, 2)
	if p.DeleteCursor == 0 {
		delStyle = delStyle.Foreground(r.Theme.Base).Background(r.Theme.Red).Bold(true)
	} else {
		cancelStyle = cancelStyle.Foreground(r.Theme.Base).Background(r.Theme.Accent).Bold(true)
	}
	buttons := delStyle.Render("delete") + "  " + cancelStyle.Render("cancel")

	footer := lipgloss.NewStyle().Foreground(r.Theme.Overlay).Background(bg).
		Render("\u2190\u2192 select \u00B7 enter confirm \u00B7 esc cancel")

	content := strings.Join([]string{title, "", hostLabel, connStr, "", warn, "", buttons, "", footer}, "\n")

	box := lipgloss.NewStyle().
		Width(42).
		Background(bg).
		Padding(1, 2).
		Align(lipgloss.Center).
		Render(content)

	return lipgloss.Place(r.W, r.H, lipgloss.Center, lipgloss.Center, box,
		lipgloss.WithWhitespaceBackground(r.Theme.Base))
}

// ── Create group overlay ──────────────────────────────────────────────

// GroupInputViewParams holds data for create/rename group overlays.
type GroupInputViewParams struct {
	Title       string // e.g. "+ new group" or "edit rename group"
	InputValue  string
	InputCursor int
	Focus       int    // 0=input, 1=action button, 2=cancel
	ActionLabel string // "create" or "rename"
}

// RenderCreateGroupOverlay renders the create group overlay.
func (r *Renderer) RenderCreateGroupOverlay(p GroupInputViewParams) string {
	bg := r.Theme.Mantle
	blink := r.Tick%2 == 0

	title := lipgloss.NewStyle().Foreground(r.Theme.Accent).Background(bg).Bold(true).
		Render(r.Icons.Add + " new group")

	label := lipgloss.NewStyle().Foreground(r.Theme.Overlay).Background(bg).Render("  name")
	input := r.RenderModalField(p.InputValue, p.InputCursor, false, p.Focus == 0, blink, bg)

	createStyle := lipgloss.NewStyle().Foreground(r.Theme.Subtext).Background(bg).Padding(0, 2)
	cancelStyle := lipgloss.NewStyle().Foreground(r.Theme.Subtext).Background(bg).Padding(0, 2)
	if p.Focus == 1 {
		createStyle = createStyle.Foreground(r.Theme.Base).Background(r.Theme.Accent).Bold(true)
	} else if p.Focus == 2 {
		cancelStyle = cancelStyle.Foreground(r.Theme.Base).Background(r.Theme.Accent).Bold(true)
	}
	buttons := createStyle.Render("create") + "  " + cancelStyle.Render("cancel")

	footer := lipgloss.NewStyle().Foreground(r.Theme.Overlay).Background(bg).
		Render("tab nav \u00B7 enter submit \u00B7 esc cancel")

	content := strings.Join([]string{title, "", label, input, "", buttons, "", footer}, "\n")

	box := lipgloss.NewStyle().
		Width(40).
		Background(bg).
		Padding(1, 2).
		Align(lipgloss.Center).
		Render(content)

	return lipgloss.Place(r.W, r.H, lipgloss.Center, lipgloss.Center, box,
		lipgloss.WithWhitespaceBackground(r.Theme.Base))
}

// ── Rename group overlay ──────────────────────────────────────────────

// RenderRenameGroupOverlay renders the rename group overlay.
func (r *Renderer) RenderRenameGroupOverlay(p GroupInputViewParams) string {
	bg := r.Theme.Mantle
	blink := r.Tick%2 == 0

	title := lipgloss.NewStyle().Foreground(r.Theme.Accent).Background(bg).Bold(true).
		Render(r.Icons.Edit + " rename group")

	label := lipgloss.NewStyle().Foreground(r.Theme.Overlay).Background(bg).Render("  name")
	input := r.RenderModalField(p.InputValue, p.InputCursor, false, p.Focus == 0, blink, bg)

	renameStyle := lipgloss.NewStyle().Foreground(r.Theme.Subtext).Background(bg).Padding(0, 2)
	cancelStyle := lipgloss.NewStyle().Foreground(r.Theme.Subtext).Background(bg).Padding(0, 2)
	if p.Focus == 1 {
		renameStyle = renameStyle.Foreground(r.Theme.Base).Background(r.Theme.Accent).Bold(true)
	} else if p.Focus == 2 {
		cancelStyle = cancelStyle.Foreground(r.Theme.Base).Background(r.Theme.Accent).Bold(true)
	}
	buttons := renameStyle.Render("rename") + "  " + cancelStyle.Render("cancel")

	footer := lipgloss.NewStyle().Foreground(r.Theme.Overlay).Background(bg).
		Render("tab nav \u00B7 enter submit \u00B7 esc cancel")

	content := strings.Join([]string{title, "", label, input, "", buttons, "", footer}, "\n")

	box := lipgloss.NewStyle().
		Width(40).
		Background(bg).
		Padding(1, 2).
		Align(lipgloss.Center).
		Render(content)

	return lipgloss.Place(r.W, r.H, lipgloss.Center, lipgloss.Center, box,
		lipgloss.WithWhitespaceBackground(r.Theme.Base))
}

// ── Delete group overlay ──────────────────────────────────────────────

// DeleteGroupViewParams holds data for the delete group confirmation overlay.
type DeleteGroupViewParams struct {
	GroupName    string
	HostCount    int
	TargetGroup  string // where hosts will be moved
	DeleteCursor int    // 0=delete, 1=cancel
}

// RenderDeleteGroupOverlay renders the delete group confirmation overlay.
func (r *Renderer) RenderDeleteGroupOverlay(p DeleteGroupViewParams) string {
	bg := r.Theme.Mantle

	title := lipgloss.NewStyle().Foreground(r.Theme.Red).Background(bg).Bold(true).
		Render(r.Icons.Warning + " delete group")

	info := lipgloss.NewStyle().Foreground(r.Theme.Text).Background(bg).Bold(true).
		Render(fmt.Sprintf("%q (%d hosts)", p.GroupName, p.HostCount))

	hint := lipgloss.NewStyle().Foreground(r.Theme.Subtext).Background(bg).
		Render(fmt.Sprintf("hosts will move to %q", p.TargetGroup))

	delStyle := lipgloss.NewStyle().Foreground(r.Theme.Subtext).Background(bg).Padding(0, 2)
	cancelStyle := lipgloss.NewStyle().Foreground(r.Theme.Subtext).Background(bg).Padding(0, 2)
	if p.DeleteCursor == 0 {
		delStyle = delStyle.Foreground(r.Theme.Base).Background(r.Theme.Red).Bold(true)
	} else {
		cancelStyle = cancelStyle.Foreground(r.Theme.Base).Background(r.Theme.Accent).Bold(true)
	}
	buttons := delStyle.Render("delete") + "  " + cancelStyle.Render("cancel")

	footer := lipgloss.NewStyle().Foreground(r.Theme.Overlay).Background(bg).
		Render("\u2190\u2192 select \u00B7 enter confirm \u00B7 esc cancel")

	content := strings.Join([]string{title, "", info, hint, "", buttons, "", footer}, "\n")

	box := lipgloss.NewStyle().
		Width(42).
		Background(bg).
		Padding(1, 2).
		Align(lipgloss.Center).
		Render(content)

	return lipgloss.Place(r.W, r.H, lipgloss.Center, lipgloss.Center, box,
		lipgloss.WithWhitespaceBackground(r.Theme.Base))
}
