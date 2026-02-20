package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// ModalFormData holds the form state for add/edit modals
type ModalFormData struct {
	Label        textinput.Model
	Group        textinput.Model
	GroupOptions []string
	GroupIndex   int
	Hostname     textinput.Model
	Username     textinput.Model
	Port         textinput.Model
	AuthMethod   int // 0=Password, 1=Key File, 2=Generate
	Password     textinput.Model
	KeyOption    string // Legacy
	KeyType      string // "ed25519", "rsa", "ecdsa"
	PastedKey    textarea.Model
	FocusedField int
	TitleSuffix  string // Debug or status info
}

// FormField represents individual form fields
const (
	FieldLabel = iota
	FieldGroup
	FieldHostname
	FieldPort // Moved Port up
	FieldUsername
	FieldAuthMethod
	FieldAuthDetails // Password or Key details
	FieldSubmit
	FieldCancel
	// Legacy constants for compatibility if needed, but we reordered
	FieldKeyOption = 99
	FieldKeyType   = 100
	FieldPastedKey = 101
)

// Auth Methods
const (
	AuthPassword = iota
	AuthKeyPaste
	AuthKeyGen
)

// RenderAddHostModal renders the add/edit host modal
func (s *Styles) RenderAddHostModal(width, height int, form *ModalFormData, isEdit bool) string {
	// Calculate responsive dimensions
	modalWidth := (width * 85) / 100
	if modalWidth > 70 {
		modalWidth = 70
	}
	if modalWidth < 60 {
		modalWidth = 60
	}

	// Layout calculations
	// Modal has padding 1, 2. Border 1.
	// Inner width = modalWidth - 2(border) - 4(padding) = modalWidth - 6
	rowWidth := modalWidth - 6

	var modal strings.Builder

	// Title
	title := "Add New Host"
	if isEdit {
		title = "Edit Host"
	}
	if form.TitleSuffix != "" {
		title += form.TitleSuffix
	}

	modal.WriteString(s.ModalTitle.Render(title))
	modal.WriteString("\n")

	// Row 1: Label
	modal.WriteString(s.renderFormFieldResponsive("Label:", form.Label, rowWidth))
	modal.WriteString(s.renderGroupSelector(form, rowWidth))

	// Row 2: Hostname + Port
	// Host Label (13) + Host Input (flex) + Spacer (2) + Port Label (7) + Port Input (10)
	// Fixed elements width = 13 + 2 + 7 + 10 = 32
	// Host Input Width = rowWidth - 32

	portInputWidth := 6
	portTotalWidth := portInputWidth + 4 // border/padding

	// Custom labels for this row
	hostLabelView := s.FormLabel.Width(12).Align(lipgloss.Right).MarginRight(1).Render("Host*:") // required
	portLabelView := s.FormLabel.Width(6).Align(lipgloss.Right).MarginRight(1).Render("Port*:")  // required

	// Calculate flexible host width
	// Available = rowWidth
	// Used = 13 (Host Label) + 2 (Spacer) + 7 (Port Label) + 10 (Port Input) = 32
	// + 4 (Host Input Border/Padding) = 36 total overhead
	hostInputWidth := rowWidth - 36
	if hostInputWidth < 15 {
		hostInputWidth = 15
	}

	// Configure Host Input
	hostStyle := s.FormInput.Width(hostInputWidth)
	if form.FocusedField == FieldHostname {
		hostStyle = s.FormInputFocused.Width(hostInputWidth)
	}
	form.Hostname.Width = hostInputWidth
	form.Hostname.Prompt = ""
	hostView := hostStyle.Render(form.Hostname.View())

	// Configure Port Input
	portStyle := s.FormInput.Width(portTotalWidth)
	if form.FocusedField == FieldPort {
		portStyle = s.FormInputFocused.Width(portTotalWidth)
	}
	form.Port.Width = portInputWidth
	form.Port.Prompt = ""
	portView := portStyle.Render(form.Port.View())

	row1 := lipgloss.JoinHorizontal(lipgloss.Center,
		hostLabelView,
		hostView,
		lipgloss.NewStyle().Width(2).Render(""), // Spacer
		portLabelView,
		portView,
	)
	modal.WriteString(row1 + "\n")

	// Row 3: Username
	modal.WriteString(s.renderFormFieldResponsive("User*:", form.Username, rowWidth)) // required

	// Row 5: Auth Method Selector
	modal.WriteString("\n")
	modal.WriteString(s.renderAuthSelector(form))
	modal.WriteString("\n")

	// Row 6: Auth Details
	if form.AuthMethod == AuthPassword {
		// Passwords are not stored; SSH will prompt at connect-time.
		label := s.FormLabel.Width(12).Align(lipgloss.Right).MarginRight(1).Render("Pass:")
		note := "Prompt on connect"
		fieldWidth := rowWidth - 14
		fieldStyle := s.FormInput.Width(fieldWidth)
		if form.FocusedField == FieldAuthDetails {
			fieldStyle = s.FormInputFocused.Width(fieldWidth)
		}
		modal.WriteString(lipgloss.JoinHorizontal(lipgloss.Center, label, fieldStyle.Render(note)) + "\n")
	} else if form.AuthMethod == AuthKeyPaste {
		// Paste key area (multi-line)
		labelView := s.FormLabel.Width(12).Align(lipgloss.Right).MarginRight(1).Render("Key:")
		fieldWidth := rowWidth - 14
		if fieldWidth < 20 {
			fieldWidth = 20
		}
		form.PastedKey.SetWidth(fieldWidth)
		pasteStyle := s.FormInput.Width(fieldWidth)
		if form.FocusedField == FieldAuthDetails {
			pasteStyle = s.FormInputFocused.Width(fieldWidth)
		}
		modal.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, labelView, pasteStyle.Render(form.PastedKey.View())) + "\n")
	} else if form.AuthMethod == AuthKeyGen {
		// Key Gen Options
		// Show Key Type selector
		typeLabel := "Type:"
		typeValue := form.KeyType
		typeStyle := s.FormInput.Width(rowWidth - 14) // roughly rowWidth - label
		if form.FocusedField == FieldAuthDetails {
			typeStyle = s.FormInputFocused.Width(rowWidth - 14)
			typeValue += " (Press Space to cycle)"
		}

		modal.WriteString(lipgloss.JoinHorizontal(lipgloss.Center,
			s.FormLabel.Width(12).Align(lipgloss.Right).MarginRight(1).Render(typeLabel),
			typeStyle.Render(typeValue),
		) + "\n")
	}

	// Buttons
	modal.WriteString(s.renderModalButtons(form.FocusedField, isEdit))
	modal.WriteString("\n")

	// Help text
	helpText := s.HelpValue.Foreground(ColorTextDim).Render("[Tab] Next • [◄/►] Cycle selectors • [Enter] Save • [Esc] Cancel")
	modal.WriteString(helpText)

	modalContent := modal.String()

	// Apply styling
	modalBox := s.Modal.
		Width(modalWidth).
		Render(modalContent)

	// Measure actual rendered height
	boxHeight := lipgloss.Height(modalBox)

	// SCROLLING LOGIC:
	if boxHeight > height {
		lines := strings.Split(modalBox, "\n")
		totalLines := len(lines)

		// Find focused line
		focusLine := 0
		foundCursor := false
		for i, line := range lines {
			if strings.Contains(line, "█") { // textinput cursor
				focusLine = i
				foundCursor = true
				break
			}
		}

		// Fallback focus detection
		if !foundCursor {
			if form.FocusedField == FieldSubmit || form.FocusedField == FieldCancel {
				focusLine = totalLines - 3
			} else if form.FocusedField == FieldAuthMethod {
				focusLine = 8 // Approx row for auth method
			}
		}

		// Calculate scroll offset
		scrollOffset := focusLine - (height / 2)
		maxOffset := totalLines - height
		if scrollOffset < 0 {
			scrollOffset = 0
		}
		if scrollOffset > maxOffset {
			scrollOffset = maxOffset
		}

		// Slice lines
		visibleLines := lines[scrollOffset : scrollOffset+height]
		modalBox = strings.Join(visibleLines, "\n")

		return lipgloss.PlaceHorizontal(width, lipgloss.Center, modalBox)
	}

	// Standard centering
	topPadding := (height - boxHeight) / 2
	if topPadding < 0 {
		topPadding = 0
	}

	bottomPadding := height - boxHeight - topPadding
	if bottomPadding < 0 {
		bottomPadding = 0
	}

	centeredModal := lipgloss.NewStyle().
		PaddingTop(topPadding).
		PaddingBottom(bottomPadding).
		Render(lipgloss.PlaceHorizontal(width, lipgloss.Center, modalBox))

	return centeredModal
}

// renderAuthSelector renders the auth method selector as a spinner
func (s *Styles) renderAuthSelector(form *ModalFormData) string {
	var label string
	switch form.AuthMethod {
	case AuthPassword:
		label = "Password"
	case AuthKeyPaste:
		label = "Paste Key"
	case AuthKeyGen:
		label = "Generate Key"
	}

	// Style for the spinner box
	var style lipgloss.Style
	var arrowColor lipgloss.Style

	if form.FocusedField == FieldAuthMethod {
		style = s.FormInputFocused.Width(20).Align(lipgloss.Center)
		arrowColor = lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)
	} else {
		style = s.FormInput.Width(20).Align(lipgloss.Center)
		arrowColor = lipgloss.NewStyle().Foreground(ColorTextDim)
	}

	// Render content: "◄  Label  ►"
	content := fmt.Sprintf("%s  %s  %s", arrowColor.Render("◄"), label, arrowColor.Render("►"))
	spinner := style.Render(content)

	// Label
	labelView := s.FormLabel.Width(12).Align(lipgloss.Right).MarginRight(1).Render("Auth:")

	return lipgloss.JoinHorizontal(lipgloss.Center, labelView, spinner)
}

func (s *Styles) renderGroupSelector(form *ModalFormData, rowWidth int) string {
	label := "Ungrouped"
	idx := 0
	if len(form.GroupOptions) > 0 {
		idx = form.GroupIndex
		if idx < 0 || idx >= len(form.GroupOptions) {
			idx = 0
		}
		if strings.TrimSpace(form.GroupOptions[idx]) != "" {
			label = form.GroupOptions[idx]
		}
	}

	spinnerWidth := rowWidth - 14
	if spinnerWidth < 20 {
		spinnerWidth = 20
	}

	var style lipgloss.Style
	var arrowColor lipgloss.Style
	if form.FocusedField == FieldGroup {
		style = s.FormInputFocused.Width(spinnerWidth).Align(lipgloss.Center)
		arrowColor = lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)
	} else {
		style = s.FormInput.Width(spinnerWidth).Align(lipgloss.Center)
		arrowColor = lipgloss.NewStyle().Foreground(ColorTextDim)
	}

	cue := ""
	if len(form.GroupOptions) > 0 {
		cue = fmt.Sprintf(" [%d/%d]", idx+1, len(form.GroupOptions))
	}
	content := fmt.Sprintf("%s  %s%s  %s", arrowColor.Render("◄"), label, cue, arrowColor.Render("►"))
	spinner := style.Render(content)
	labelView := s.FormLabel.Width(12).Align(lipgloss.Right).MarginRight(1).Render("Group:")

	return lipgloss.JoinHorizontal(lipgloss.Center, labelView, spinner) + "\n"
}

// renderFormFieldResponsive renders a form field with responsive width
func (s *Styles) renderFormFieldResponsive(label string, input textinput.Model, totalWidth int) string {
	// Fixed dimensions
	labelWidth := 14 // 12 chars + 1 margin + 1 extra space

	// Calculate available width for the input box
	inputBoxWidth := totalWidth - labelWidth
	if inputBoxWidth < 10 {
		inputBoxWidth = 10
	}

	// Calculate inner text width (accounting for border/padding)
	// Border (2) + Padding (2) = 4
	input.Width = inputBoxWidth - 4
	input.Prompt = ""

	// Reset styles
	input.TextStyle = lipgloss.NewStyle()
	input.Cursor.Style = lipgloss.NewStyle().Foreground(ColorPrimary)

	// Determine style
	var style lipgloss.Style
	if input.Focused() {
		style = s.FormInputFocused.Width(inputBoxWidth)
	} else {
		style = s.FormInput.Width(inputBoxWidth)
	}

	// Render input
	inputView := style.Render(input.View())

	// Render Label
	labelView := s.FormLabel.Width(12).Align(lipgloss.Right).MarginRight(1).Render(label)

	// Join
	row := lipgloss.JoinHorizontal(
		lipgloss.Center,
		labelView,
		inputView,
	)

	return row + "\n"
}

// renderKeyOptionsCompact renders SSH key options in ultra-compact mode
func (s *Styles) renderKeyOptionsCompact(form *ModalFormData, width int) string {
	var opts strings.Builder

	// Show key option inline with label
	label := s.FormLabel.Render("SSH Key:")
	opts.WriteString(label)

	if form.KeyOption == "generate" {
		// Show inline: "SSH Key: [●] Generate (Ed25519)"
		genStyle := s.FormInput.Width(width - 5)
		if form.FocusedField == FieldKeyOption {
			genStyle = s.FormInputFocused.Width(width - 5)
		}

		keyTypeShort := form.KeyType
		if form.KeyType == "ed25519" {
			keyTypeShort = "Ed25519"
		} else if form.KeyType == "rsa" {
			keyTypeShort = "RSA"
		} else if form.KeyType == "ecdsa" {
			keyTypeShort = "ECDSA"
		}

		generateOpt := genStyle.Render("[●] Gen (" + keyTypeShort + ")")
		opts.WriteString("\n")
		opts.WriteString(strings.Repeat(" ", 17))
		opts.WriteString(generateOpt)
		opts.WriteString("\n")
	} else {
		// Show compact paste option
		pasteStyle := s.FormInput.Width(width - 5)
		if form.FocusedField == FieldKeyOption {
			pasteStyle = s.FormInputFocused.Width(width - 5)
		}

		pasteOpt := pasteStyle.Render("[●] Paste")
		opts.WriteString("\n")
		opts.WriteString(strings.Repeat(" ", 17))
		opts.WriteString(pasteOpt)
		opts.WriteString("\n")
	}

	return opts.String()
}

// renderKeyOptionsResponsive renders SSH key options with responsive width
func (s *Styles) renderKeyOptionsResponsive(form *ModalFormData, width int) string {
	var opts strings.Builder

	opts.WriteString("\n")

	// Key option toggle
	label := s.FormLabel.Render("SSH Key:")
	opts.WriteString(label)
	opts.WriteString("\n")

	// Show current selection inline
	if form.KeyOption == "generate" {
		// Generate option selected
		genFocused := form.FocusedField == FieldKeyOption
		genStyle := s.FormInput.Width(width)
		if genFocused {
			genStyle = s.FormInputFocused.Width(width)
		}

		generateOpt := genStyle.Render("[●] Generate new key")
		opts.WriteString(strings.Repeat(" ", 17))
		opts.WriteString(generateOpt)
		opts.WriteString("\n")

		// Show selected key type inline
		keyTypeFocused := form.FocusedField == FieldKeyType
		keyTypeStyle := s.FormInput.Width(width)
		if keyTypeFocused {
			keyTypeStyle = s.FormInputFocused.Width(width)
		}

		keyTypeLabel := ""
		switch form.KeyType {
		case "ed25519":
			keyTypeLabel = "  Ed25519"
		case "rsa":
			keyTypeLabel = "  RSA 4096"
		case "ecdsa":
			keyTypeLabel = "  ECDSA P-256"
		}

		keyTypeOpt := keyTypeStyle.Render(keyTypeLabel)
		opts.WriteString(strings.Repeat(" ", 17))
		opts.WriteString(keyTypeOpt)
		if keyTypeFocused {
			hint := s.HelpValue.Foreground(ColorTextDim).Render(" (Enter=cycle)")
			opts.WriteString(" ")
			opts.WriteString(hint)
		}
		opts.WriteString("\n")
	} else {
		// Paste option selected
		pasteFocused := form.FocusedField == FieldKeyOption
		pasteStyle := s.FormInput.Width(width)
		if pasteFocused {
			pasteStyle = s.FormInputFocused.Width(width)
		}

		pasteOpt := pasteStyle.Render("[●] Paste existing key")
		opts.WriteString(strings.Repeat(" ", 17))
		opts.WriteString(pasteOpt)
		opts.WriteString("\n")

		// Show compact paste area (2 lines instead of 3)
		pasteAreaFocused := form.FocusedField == FieldPastedKey
		pasteAreaStyle := s.FormInput.Width(width).Height(2)
		if pasteAreaFocused {
			pasteAreaStyle = s.FormInputFocused.Width(width).Height(2)
		}

		// Update pasted key input width
		form.PastedKey.SetWidth(width - 5) // Adjust for padding

		// Render textarea view
		content := form.PastedKey.View()
		if form.PastedKey.Value() == "" && !pasteAreaFocused {
			content = s.DetailValue.Foreground(ColorTextDim).Render("Paste key...")
		}

		textArea := pasteAreaStyle.Render(content)
		opts.WriteString(strings.Repeat(" ", 17))
		opts.WriteString(textArea)
		opts.WriteString("\n")
	}

	// Compact hint
	if form.FocusedField == FieldKeyOption {
		toggleHint := s.HelpValue.Foreground(ColorTextDim).Render("  Space=toggle")
		opts.WriteString(strings.Repeat(" ", 17))
		opts.WriteString(toggleHint)
		opts.WriteString("\n")
	}

	return opts.String()
}

// renderModalButtons renders submit/cancel buttons
func (s *Styles) renderModalButtons(focusedField int, isEdit bool) string {
	submitLabel := "Add Host"
	if isEdit {
		submitLabel = "Save"
	}

	submitStyle := s.FormButton
	cancelStyle := s.FormButton

	if focusedField == FieldSubmit {
		submitStyle = s.FormButtonFocused
	} else if focusedField == FieldCancel {
		cancelStyle = s.FormButtonFocused
	}

	buttons := lipgloss.JoinHorizontal(
		lipgloss.Top,
		submitStyle.Render(submitLabel),
		cancelStyle.Render("Cancel"),
	)

	// Reduced indentation for compactness
	return "\n" + strings.Repeat(" ", 15) + buttons + "\n"
}

// getBoolChar returns a checkmark or space based on boolean
func (s *Styles) getBoolChar(checked bool) string {
	if checked {
		return "●"
	}
	return " "
}

// RenderDeleteModal renders the delete confirmation modal
func (s *Styles) RenderDeleteModal(width, height int, hostname, username string, confirmed bool) string {
	// Responsive width (70% of terminal, max 50)
	modalWidth := (width * 70) / 100
	if modalWidth > 50 {
		modalWidth = 50
	}
	if modalWidth < 40 {
		modalWidth = 40
	}

	var modal strings.Builder

	modal.WriteString(s.ModalTitle.Foreground(ColorDanger).Render("⚠️  Delete Host"))
	modal.WriteString("\n\n")

	modal.WriteString("Delete this host?\n\n")

	modal.WriteString(s.DetailLabel.Render("Host:"))
	modal.WriteString(" ")
	modal.WriteString(s.DetailValue.Render(hostname))
	modal.WriteString("\n")

	modal.WriteString(s.DetailLabel.Render("User:"))
	modal.WriteString(" ")
	modal.WriteString(s.DetailValue.Render(username))
	modal.WriteString("\n\n")

	modal.WriteString(s.Error.Render("Cannot be undone!"))
	modal.WriteString("\n\n")

	// Buttons
	yesStyle := s.FormButton
	noStyle := s.FormButton

	if confirmed {
		yesStyle = s.FormButtonFocused.Background(ColorDanger)
	} else {
		noStyle = s.FormButtonFocused
	}

	buttons := lipgloss.JoinHorizontal(
		lipgloss.Top,
		yesStyle.Render("Delete"),
		noStyle.Render("Cancel"),
	)

	modal.WriteString(buttons)
	modal.WriteString("\n")

	helpText := s.HelpValue.Foreground(ColorTextDim).Render("[◄/►] Select • [Enter] Confirm • [Esc] Cancel")
	modal.WriteString(helpText)

	// Apply styling
	modalBox := s.Modal.
		BorderForeground(ColorDanger).
		Width(modalWidth - 4).
		Render(modal.String())

	// Manual centering logic to avoid cutoff at top
	// If boxHeight > height, topPadding will be negative, clamped to 0
	topPadding := (height - lipgloss.Height(modalBox)) / 2
	if topPadding < 0 {
		topPadding = 0
	}

	// Calculate bottom padding
	boxHeight := lipgloss.Height(modalBox)
	bottomPadding := height - boxHeight - topPadding
	if bottomPadding < 0 {
		bottomPadding = 0
	}

	// Horizontal centering
	centeredHorizontal := lipgloss.PlaceHorizontal(width, lipgloss.Center, modalBox)

	// Apply vertical padding and render
	centeredModal := lipgloss.NewStyle().
		PaddingTop(topPadding).
		PaddingBottom(bottomPadding).
		Render(centeredHorizontal)

	return centeredModal
}

// RenderQuitModal renders a quit confirmation modal when mounts are active.
// focus: 0=Unmount&Quit, 1=Leave Mounted&Quit, 2=Cancel
func (s *Styles) RenderQuitModal(width, height int, mounts []string, focus int) string {
	modalWidth := (width * 75) / 100
	if modalWidth > 72 {
		modalWidth = 72
	}
	if modalWidth < 50 {
		modalWidth = 50
	}

	var b strings.Builder
	b.WriteString(s.ModalTitle.Foreground(ColorWarning).Render("⚠️  Active Mounts"))
	b.WriteString("\n\n")

	b.WriteString(s.DetailValue.Render("You have active Finder mounts."))
	b.WriteString("\n")
	b.WriteString(s.HelpValue.Foreground(ColorTextDim).Render("Unmounting is recommended. Leaving mounts open may keep a mount key file on disk for reconnect."))
	b.WriteString("\n\n")

	max := 6
	if len(mounts) > 0 {
		for i := 0; i < len(mounts) && i < max; i++ {
			b.WriteString(s.DetailValue.Foreground(ColorText).Bold(false).Render("• " + mounts[i]))
			b.WriteString("\n")
		}
		if len(mounts) > max {
			b.WriteString(s.HelpValue.Foreground(ColorTextDim).Render(fmt.Sprintf("…and %d more", len(mounts)-max)))
			b.WriteString("\n")
		}
	} else {
		b.WriteString(s.HelpValue.Foreground(ColorTextDim).Render("No mounts found."))
		b.WriteString("\n")
	}

	b.WriteString("\n")

	unmountStyle := s.FormButton
	leaveStyle := s.FormButton
	cancelStyle := s.FormButton
	switch focus {
	case 0:
		unmountStyle = s.FormButtonFocused.Background(ColorDanger)
	case 1:
		leaveStyle = s.FormButtonFocused
	default:
		cancelStyle = s.FormButtonFocused
	}

	buttons := lipgloss.JoinHorizontal(
		lipgloss.Top,
		unmountStyle.Render("Unmount & Quit"),
		leaveStyle.Render("Leave Mounted & Quit"),
		cancelStyle.Render("Cancel"),
	)
	b.WriteString(buttons)
	b.WriteString("\n")
	b.WriteString(s.HelpValue.Foreground(ColorTextDim).Render("[◄/►] Select • [Enter] Confirm • [Esc] Cancel"))

	modalBox := s.Modal.
		BorderForeground(ColorWarning).
		Width(modalWidth - 4).
		Render(b.String())

	topPadding := (height - lipgloss.Height(modalBox)) / 2
	if topPadding < 0 {
		topPadding = 0
	}
	boxHeight := lipgloss.Height(modalBox)
	bottomPadding := height - boxHeight - topPadding
	if bottomPadding < 0 {
		bottomPadding = 0
	}

	centeredHorizontal := lipgloss.PlaceHorizontal(width, lipgloss.Center, modalBox)
	return lipgloss.NewStyle().
		PaddingTop(topPadding).
		PaddingBottom(bottomPadding).
		Render(centeredHorizontal)
}

// RenderSpotlight renders the Raycast-like search overlay
func (s *Styles) RenderSpotlight(width, height int, input textinput.Model, results []interface{}, selectedIdx int, armedSFTP bool, armedMount bool, armedUnmount bool) string {
	var modal strings.Builder

	// 1. Render Search Input (Top)
	// Apply custom styling to input
	// input.TextStyle = s.SpotlightInput // We can't easily override style inside model without updating it.
	// So we wrap it.

	// Create a clear input view without border, we will wrap it
	inputView := input.View()
	styledInput := s.SpotlightInput.Width(58).Render(inputView)
	modal.WriteString(styledInput)
	modal.WriteString("\n")

	// 2. Render Results List
	// Limit results to 5-8 items to fit in the box
	maxItems := 8
	displayCount := 0

	if len(results) == 0 {
		modal.WriteString(s.SpotlightItem.Render("No results found"))
	} else {
		// Calculate window for scrolling if needed, but for now let's just show top items or simplistic window
		// Simple window logic: keep selected in view
		startIdx := 0
		if selectedIdx >= maxItems {
			startIdx = selectedIdx - maxItems + 1
		}
		endIdx := startIdx + maxItems
		if endIdx > len(results) {
			endIdx = len(results)
		}

		for i := startIdx; i < endIdx; i++ {
			// Extract host data
			hostMap, ok := results[i].(map[string]interface{})
			if !ok {
				continue
			}
			kind, _ := hostMap["Kind"].(string)
			itemText := ""
			if kind == "group" {
				groupName, _ := hostMap["GroupName"].(string)
				count, _ := hostMap["Count"].(int)
				itemText = fmt.Sprintf("# %s (%d)", groupName, count)
			} else {
				label, _ := hostMap["Label"].(string)
				hostname, _ := hostMap["Hostname"].(string)
				username, _ := hostMap["Username"].(string)
				mounted, _ := hostMap["Mounted"].(bool)
				groupName, _ := hostMap["GroupName"].(string)

				itemText = fmt.Sprintf("%s @ %s", username, hostname)
				displayLabel := strings.TrimSpace(label)
				if displayLabel != "" && displayLabel != strings.TrimSpace(hostname) {
					itemText = fmt.Sprintf("%s — %s", displayLabel, itemText)
				}
				if strings.TrimSpace(groupName) != "" {
					itemText = fmt.Sprintf("[%s] %s", groupName, itemText)
				}
				if mounted {
					itemText = "📁 " + itemText
				}
				if indent, ok := hostMap["Indent"].(int); ok && indent > 0 {
					itemText = strings.Repeat("  ", indent) + itemText
				}
			}

			// Truncate if too long
			if len(itemText) > 50 {
				itemText = itemText[:47] + "..."
			}

			// Render item
			if i == selectedIdx {
				modal.WriteString(s.SpotlightSelected.Width(56).Render(itemText))
			} else {
				modal.WriteString(s.SpotlightItem.Width(56).Render(itemText))
			}
			modal.WriteString("\n")
			displayCount++
		}
	}

	// Pad remaining space to keep box stable size
	for displayCount < maxItems {
		modal.WriteString("\n")
		displayCount++
	}

	// Footer hint
	modal.WriteString("\n")
	if armedUnmount {
		modal.WriteString(s.HelpValue.Foreground(ColorTextDim).Padding(0, 2).Render("[Esc] Close • [M] Cancel • [Enter] Unmount"))
	} else if armedMount {
		modal.WriteString(s.HelpValue.Foreground(ColorTextDim).Padding(0, 2).Render("[Esc] Close • [M] Cancel • [Enter] Mount"))
	} else if armedSFTP {
		modal.WriteString(s.HelpValue.Foreground(ColorTextDim).Padding(0, 2).Render("[Esc] Close • [S] Cancel • [Enter] SFTP • [M] Mount"))
	} else {
		modal.WriteString(s.HelpValue.Foreground(ColorTextDim).Padding(0, 2).Render("[Esc] Close • [Enter] SSH/Jump Group • [S] Arm SFTP • [M] Mount"))
	}

	modalContent := modal.String()

	// Apply container styling
	modalBox := s.Spotlight.Render(modalContent)

	// Center in terminal
	centeredModal := lipgloss.Place(
		width,
		height,
		lipgloss.Center,
		lipgloss.Center, // Vertical center for Spotlight
		modalBox,
	)

	return centeredModal
}

// RenderGroupInputModal renders a simple text-input modal for creating/renaming groups.
// focus: 0=input, 1=submit, 2=cancel
func (s *Styles) RenderGroupInputModal(width, height int, title string, input textinput.Model, submitLabel string, focus int) string {
	modalWidth := (width * 70) / 100
	if modalWidth > 64 {
		modalWidth = 64
	}
	if modalWidth < 36 {
		modalWidth = 36
	}
	if modalWidth > width-2 {
		modalWidth = width - 2
	}
	if modalWidth < 24 {
		modalWidth = 24
	}

	rowWidth := modalWidth - 6
	if rowWidth < 18 {
		rowWidth = 18
	}

	var b strings.Builder
	b.WriteString(s.ModalTitle.Render(title))
	b.WriteString("\n\n")
	b.WriteString(s.renderFormFieldResponsive("Name*:", input, rowWidth))
	b.WriteString("\n")

	submitStyle := s.FormButton
	cancelStyle := s.FormButton
	if focus == 1 {
		submitStyle = s.FormButtonFocused
	} else if focus == 2 {
		cancelStyle = s.FormButtonFocused
	}
	buttons := lipgloss.JoinHorizontal(lipgloss.Top, submitStyle.Render(submitLabel), cancelStyle.Render("Cancel"))
	b.WriteString(lipgloss.PlaceHorizontal(rowWidth, lipgloss.Center, buttons))
	b.WriteString("\n\n")
	help := "[Tab] Next • [Shift+Tab] Prev • [Enter] Confirm • [Esc] Cancel"
	b.WriteString(s.HelpValue.Foreground(ColorTextDim).Width(rowWidth).Align(lipgloss.Center).Render(help))

	modalBox := s.Modal.Width(modalWidth).Render(b.String())
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, modalBox)
}

// RenderDeleteGroupModal renders group delete confirmation.
func (s *Styles) RenderDeleteGroupModal(width, height int, groupName string, hostCount int, deleteFocused bool) string {
	modalWidth := 54
	if modalWidth > width-4 {
		modalWidth = width - 4
	}
	if modalWidth < 40 {
		modalWidth = 40
	}

	var b strings.Builder
	b.WriteString(s.ModalTitle.Foreground(ColorDanger).Render("⚠️  Delete Group"))
	b.WriteString("\n\n")
	b.WriteString("Delete this group? Hosts will be ungrouped.\n\n")
	b.WriteString(s.DetailLabel.Render("Group:"))
	b.WriteString(" ")
	b.WriteString(s.DetailValue.Render(groupName))
	b.WriteString("\n")
	b.WriteString(s.DetailLabel.Render("Hosts:"))
	b.WriteString(" ")
	b.WriteString(s.DetailValue.Render(fmt.Sprintf("%d", hostCount)))
	b.WriteString("\n\n")

	deleteStyle := s.FormButton
	cancelStyle := s.FormButton
	if deleteFocused {
		deleteStyle = s.FormButtonFocused.Background(ColorDanger)
	} else {
		cancelStyle = s.FormButtonFocused
	}
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, deleteStyle.Render("Delete"), cancelStyle.Render("Cancel")))
	b.WriteString("\n")
	b.WriteString(s.HelpValue.Foreground(ColorTextDim).Render("[◄/►] Select • [Enter] Confirm • [Esc] Cancel"))

	modalBox := s.Modal.BorderForeground(ColorDanger).Width(modalWidth - 4).Render(b.String())
	topPadding := (height - lipgloss.Height(modalBox)) / 2
	if topPadding < 0 {
		topPadding = 0
	}
	centeredHorizontal := lipgloss.PlaceHorizontal(width, lipgloss.Center, modalBox)
	return lipgloss.NewStyle().PaddingTop(topPadding).Render(centeredHorizontal)
}
