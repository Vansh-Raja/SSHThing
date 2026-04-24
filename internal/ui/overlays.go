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
	ImportMode   bool
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
	} else if p.ImportMode {
		placeholderText = "search personal hosts to import..."
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
	} else if p.ImportMode {
		footerText = "esc back to add host  ·  enter import personal host"
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
	IsEdit               bool
	Fields               []FormField // [label, tags, hostname, port, username, authDetail]
	Focus                int
	Editing              bool
	Groups               []string
	GroupIdx             int
	AuthOptions          []string
	AuthIdx              int
	KeyTypes             []string
	KeyTypeIdx           int
	AllowImport          bool
	AuthLocked           bool
	SecretRevealed       bool
	PrivateKeyEditorView string
	ScrollOffset         int
	Err                  error
}

type PrivateKeyEditorViewParams struct {
	EditorView string
	Width      int
	Height     int
	Err        error
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
	compact := r.H < 28

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

	blink := r.Tick%2 == 0
	useColumns := r.W >= 96 && cw >= 82
	columnGap := 4

	bodyWidth := cw
	if bodyWidth > 104 {
		bodyWidth = 104
	}
	if !useColumns && bodyWidth > 70 {
		bodyWidth = 70
	}

	var bodyLines []string
	if useColumns {
		leftW := (bodyWidth - columnGap) / 2
		rightW := bodyWidth - columnGap - leftW
		leftLines := r.renderAddHostDetailsColumn(p, leftW, blink, compact)
		rightLines := r.renderAddHostCredentialsColumn(p, rightW, blink, compact)
		maxLines := max(len(leftLines), len(rightLines))
		for len(leftLines) < maxLines {
			leftLines = append(leftLines, "")
		}
		for len(rightLines) < maxLines {
			rightLines = append(rightLines, "")
		}
		gap := strings.Repeat(" ", columnGap)
		for i := 0; i < maxLines; i++ {
			left := lipgloss.NewStyle().Width(leftW).Render(leftLines[i])
			bodyLines = append(bodyLines, left+gap+rightLines[i])
		}
	} else {
		bodyLines = append(bodyLines, r.renderAddHostDetailsColumn(p, bodyWidth, blink, compact)...)
		if !compact {
			bodyLines = append(bodyLines, "")
		}
		bodyLines = append(bodyLines, r.renderAddHostCredentialsColumn(p, bodyWidth, blink, compact)...)
	}

	if p.Err != nil {
		bodyLines = append(bodyLines, "")
		bodyLines = append(bodyLines, r.renderErrLine(p.Err))
	}

	saveLabel := "save host"
	if p.IsEdit {
		saveLabel = "update host"
	}
	bodyLines = append(bodyLines, "")
	if p.Focus == FFSave {
		bodyLines = append(bodyLines, "  "+lipgloss.NewStyle().Foreground(r.Theme.Accent).Bold(true).Render(r.Icons.Save+" "+saveLabel))
	} else {
		bodyLines = append(bodyLines, "  "+lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render("  "+saveLabel))
	}

	footer := r.RenderFooter(r.addHostFooterText(p))

	bodyH := r.H - 7
	if bodyH < 6 {
		bodyH = 6
	}
	scroll := p.ScrollOffset
	if scroll < 0 {
		scroll = 0
	}
	if scroll > max(0, len(bodyLines)-bodyH) {
		scroll = max(0, len(bodyLines)-bodyH)
	}
	visible := append([]string(nil), bodyLines...)
	if len(visible) > bodyH {
		visible = bodyLines[scroll:min(len(bodyLines), scroll+bodyH)]
		if scroll > 0 && len(visible) > 0 {
			visible[0] = lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render("  ↑ more")
		}
		if scroll+bodyH < len(bodyLines) && len(visible) > 0 {
			visible[len(visible)-1] = lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render("  ↓ more")
		}
	}

	innerLines := []string{headerLine, rule}
	if !compact {
		innerLines = append(innerLines, "")
	}
	innerLines = append(innerLines, visible...)
	innerLines = append(innerLines, "")
	innerLines = append(innerLines, lipgloss.NewStyle().Foreground(r.Theme.Surface0).Render(strings.Repeat(r.Icons.Rule, ruleW)))
	innerLines = append(innerLines, footer)

	return r.PadContent(strings.Join(innerLines, "\n"), pad)
}

func (r *Renderer) renderAddHostDetailsColumn(p AddHostViewParams, width int, blink bool, compact bool) []string {
	fieldW := width - 4
	if fieldW < 12 {
		fieldW = 12
	}
	var lines []string
	lines = append(lines, lipgloss.NewStyle().Foreground(r.Theme.Text).Bold(true).Render("host details"))
	r.appendFormGap(&lines, compact)

	lines = append(lines, r.RenderFormLabel("label", p.Focus == FFLabel))
	lines = append(lines, r.RenderInput(p.Fields[FFLabel], p.Focus == FFLabel, fieldW, blink, p.Editing))
	r.appendFormGap(&lines, compact)

	lines = append(lines, r.RenderFormLabel("group", p.Focus == FFGroup))
	lines = append(lines, r.renderAddHostGroupSelector(p))
	r.appendFormGap(&lines, compact)

	lines = append(lines, r.RenderFormLabel("tags", p.Focus == FFTags))
	lines = append(lines, r.RenderInput(p.Fields[FFTags], p.Focus == FFTags, fieldW, blink, p.Editing))
	if !compact {
		lines = append(lines, lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render("  comma-separated"))
	}
	r.appendFormGap(&lines, compact)

	if width >= 42 {
		hostW := max(20, width-18)
		portW := 12
		labelRow := lipgloss.NewStyle().Width(hostW).Render(r.RenderFormLabel("hostname", p.Focus == FFHostname)) +
			lipgloss.NewStyle().Width(portW).Render(r.RenderFormLabel("port", p.Focus == FFPort))
		inputRow := lipgloss.NewStyle().Width(hostW).Render(r.RenderInput(p.Fields[FFHostname], p.Focus == FFHostname, hostW-4, blink, p.Editing)) +
			lipgloss.NewStyle().Width(portW).Render(r.RenderInput(p.Fields[FFPort], p.Focus == FFPort, portW-4, blink, p.Editing))
		lines = append(lines, labelRow, inputRow)
	} else {
		lines = append(lines, r.RenderFormLabel("hostname", p.Focus == FFHostname))
		lines = append(lines, r.RenderInput(p.Fields[FFHostname], p.Focus == FFHostname, fieldW, blink, p.Editing))
		r.appendFormGap(&lines, compact)
		lines = append(lines, r.RenderFormLabel("port", p.Focus == FFPort))
		lines = append(lines, r.RenderInput(p.Fields[FFPort], p.Focus == FFPort, fieldW, blink, p.Editing))
	}
	r.appendFormGap(&lines, compact)
	return lines
}

func (r *Renderer) renderAddHostCredentialsColumn(p AddHostViewParams, width int, blink bool, compact bool) []string {
	fieldW := width - 4
	if fieldW < 12 {
		fieldW = 12
	}
	var lines []string
	lines = append(lines, lipgloss.NewStyle().Foreground(r.Theme.Text).Bold(true).Render("credentials"))
	r.appendFormGap(&lines, compact)

	lines = append(lines, r.RenderFormLabel("username", p.Focus == FFUsername))
	lines = append(lines, r.RenderInput(p.Fields[FFUsername], p.Focus == FFUsername, fieldW, blink, p.Editing))
	r.appendFormGap(&lines, compact)

	lines = append(lines, r.RenderFormLabel("authentication", p.Focus == FFAuthMeth))
	lines = append(lines, r.renderAddHostAuthSelector(p))
	r.appendFormGap(&lines, compact)

	switch p.AuthIdx {
	case 0:
		lines = append(lines, r.RenderFormLabel("password", p.Focus == FFAuthDet))
		lines = append(lines, r.RenderInput(p.Fields[FFAuthDet], p.Focus == FFAuthDet, fieldW, blink, p.Editing))
	case 1:
		lines = append(lines, r.RenderFormLabel("private key", p.Focus == FFAuthDet))
		lines = append(lines, r.RenderSecretPreview(p.Fields[FFAuthDet].Value, fieldW, false, 2)...)
		hint := "  enter edit key  ·  c copy key"
		lines = append(lines, lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render(hint))
	case 2:
		lines = append(lines, r.RenderFormLabel("key type", false))
		kt := ""
		if len(p.KeyTypes) > 0 {
			kt = p.KeyTypes[p.KeyTypeIdx]
		}
		lines = append(lines, "  "+lipgloss.NewStyle().Foreground(r.Theme.Text).Render(kt)+
			"  "+lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render("(space to cycle)"))
	}

	return lines
}

func (r *Renderer) renderAddHostGroupSelector(p AddHostViewParams) string {
	gName := ""
	if len(p.Groups) > 0 && p.GroupIdx >= 0 && p.GroupIdx < len(p.Groups) {
		gName = p.Groups[p.GroupIdx]
	}
	gCount := fmt.Sprintf("[%d/%d]", p.GroupIdx+1, len(p.Groups))
	if len(p.Groups) == 0 {
		gCount = "[0/0]"
	}
	if p.Focus == FFGroup && p.Editing {
		return "  " +
			lipgloss.NewStyle().Foreground(r.Theme.Accent).Render(r.Icons.LeftArrow) +
			" " + lipgloss.NewStyle().Foreground(r.Theme.Text).Render(gName) +
			" " + lipgloss.NewStyle().Foreground(r.Theme.Accent).Render(r.Icons.RightArrow) +
			"  " + lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render(gCount)
	}
	if p.Focus == FFGroup {
		return "  " +
			lipgloss.NewStyle().Foreground(r.Theme.Text).Bold(true).Render(gName) +
			"  " + lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render(gCount+"  enter change")
	}
	return "  " +
		lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render(r.Icons.LeftArrow) +
		" " + lipgloss.NewStyle().Foreground(r.Theme.Subtext).Render(gName) +
		" " + lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render(r.Icons.RightArrow) +
		"  " + lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render(gCount)
}

func (r *Renderer) renderAddHostAuthSelector(p AddHostViewParams) string {
	aName := ""
	if len(p.AuthOptions) > 0 {
		aName = p.AuthOptions[p.AuthIdx]
	}
	if p.AuthLocked && aName != "" {
		aName += " (locked)"
	}
	if p.Focus == FFAuthMeth && p.Editing {
		if p.AuthLocked {
			return "  " + lipgloss.NewStyle().Foreground(r.Theme.Text).Render(aName)
		}
		return "  " +
			lipgloss.NewStyle().Foreground(r.Theme.Accent).Render(r.Icons.LeftArrow) +
			" " + lipgloss.NewStyle().Foreground(r.Theme.Text).Render(aName) +
			" " + lipgloss.NewStyle().Foreground(r.Theme.Accent).Render(r.Icons.RightArrow)
	}
	if p.Focus == FFAuthMeth {
		return "  " + lipgloss.NewStyle().Foreground(r.Theme.Text).Bold(true).Render(aName) +
			"  " + lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render("enter change")
	}
	if p.AuthLocked {
		return "  " + lipgloss.NewStyle().Foreground(r.Theme.Subtext).Render(aName)
	}
	return "  " +
		lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render(r.Icons.LeftArrow) +
		" " + lipgloss.NewStyle().Foreground(r.Theme.Subtext).Render(aName) +
		" " + lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render(r.Icons.RightArrow)
}

func (r *Renderer) addHostFooterText(p AddHostViewParams) string {
	var footerText string
	if p.Editing {
		footerText = "editing  ·  enter done  ·  esc done  ·  ←→ cursor/change  ·  tab next"
	} else {
		footerText = "↑↓ rows  ·  ←→ columns  ·  enter edit  ·  tab next  ·  esc cancel"
		if p.AllowImport {
			footerText += "  ·  I import personal host"
		}
	}
	if p.AuthIdx == 1 {
		footerText += "  ·  enter edit key  ·  c copy key"
	}
	return footerText
}

func (r *Renderer) appendFormGap(lines *[]string, compact bool) {
	if !compact {
		*lines = append(*lines, "")
	}
}

func (r *Renderer) RenderPrivateKeyEditorOverlay(p PrivateKeyEditorViewParams) string {
	bg := r.Theme.Mantle
	width := p.Width
	if width < 40 {
		width = 40
	}
	if width > r.W-4 {
		width = r.W - 4
	}
	height := p.Height
	if height < 10 {
		height = 10
	}
	if height > r.H-4 {
		height = r.H - 4
	}

	innerW := max(8, width-4)
	contentH := max(6, height-2)
	title := lipgloss.NewStyle().Foreground(r.Theme.Text).Background(bg).Bold(true).Render("edit private key")
	rule := lipgloss.NewStyle().Foreground(r.Theme.Surface0).Background(bg).Render(strings.Repeat(r.Icons.Rule, innerW))
	footer := lipgloss.NewStyle().Foreground(r.Theme.Overlay).Background(bg).Render("ctrl+s save  ·  esc cancel  ·  ctrl+y copy")

	lines := []string{title, rule}
	errLines := []string{}
	if p.Err != nil {
		errLines = append(errLines, r.renderErrLine(p.Err))
	}
	editorH := contentH - len(lines) - len(errLines) - 2
	if editorH < 3 {
		editorH = 3
	}
	editorLines := strings.Split(strings.TrimRight(p.EditorView, "\n"), "\n")
	if len(editorLines) > editorH {
		editorLines = editorLines[:editorH]
	}
	blank := lipgloss.NewStyle().Background(bg).Render(strings.Repeat(" ", innerW))
	for len(editorLines) < editorH {
		editorLines = append(editorLines, blank)
	}
	lines = append(lines, editorLines...)
	lines = append(lines, errLines...)
	lines = append(lines, rule, footer)

	box := lipgloss.NewStyle().
		Width(width).
		Height(height).
		Background(bg).
		Padding(1, 2).
		Render(strings.Join(lines, "\n"))

	return lipgloss.Place(r.W, r.H, lipgloss.Center, lipgloss.Center, box, lipgloss.WithWhitespaceBackground(r.Theme.Base))
}

type ImportConflictViewParams struct {
	ExistingLabel string
	ExistingConn  string
	Cursor        int
}

func (r *Renderer) RenderImportConflictOverlay(p ImportConflictViewParams) string {
	bg := r.Theme.Mantle
	title := lipgloss.NewStyle().Foreground(r.Theme.Yellow).Background(bg).Bold(true).
		Render(r.Icons.Warning + " host already exists in this team")
	body := lipgloss.NewStyle().Foreground(r.Theme.Subtext).Background(bg).
		Render("A host with the same hostname already exists, but its settings or credentials differ.")
	existing := lipgloss.NewStyle().Foreground(r.Theme.Text).Background(bg).Bold(true).Render(p.ExistingLabel)
	conn := lipgloss.NewStyle().Foreground(r.Theme.Subtext).Background(bg).Render(p.ExistingConn)

	button := func(label string, idx int, danger bool) string {
		style := lipgloss.NewStyle().Foreground(r.Theme.Subtext).Background(bg).Padding(0, 2)
		if p.Cursor == idx {
			color := r.Theme.Accent
			if danger {
				color = r.Theme.Red
			}
			style = style.Foreground(r.Theme.Base).Background(color).Bold(true)
		}
		return style.Render(label)
	}

	buttons := button("update existing", 0, false) + "  " + button("create duplicate", 1, true) + "  " + button("cancel", 2, false)
	footer := lipgloss.NewStyle().Foreground(r.Theme.Overlay).Background(bg).
		Render("←→ select · enter confirm · esc cancel")
	content := strings.Join([]string{title, "", body, "", existing, conn, "", buttons, "", footer}, "\n")
	box := lipgloss.NewStyle().Width(64).Background(bg).Padding(1, 2).Align(lipgloss.Center).Render(content)
	return lipgloss.Place(r.W, r.H, lipgloss.Center, lipgloss.Center, box, lipgloss.WithWhitespaceBackground(r.Theme.Base))
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
