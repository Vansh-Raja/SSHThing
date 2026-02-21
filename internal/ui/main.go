package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

type SyncActivity struct {
	Active   bool
	Frame    int
	Progress float64
	Stage    string
}

// RenderView renders the main view based on the current view mode
func (s *Styles) RenderView(viewMode string, width, height int, hosts interface{}, selectedIdx int, searchQuery string, isSearching bool, err error) string {
	switch viewMode {
	case "list":
		return s.RenderListView(width, height, hosts, selectedIdx, searchQuery, isSearching, err)
	case "help":
		return s.RenderHelpView(width, height)
	default:
		return "Unknown view mode"
	}
}

// RenderListView renders the main two-panel layout
func (s *Styles) RenderListView(width, height int, hostsInterface interface{}, selectedIdx int, searchQuery string, isSearching bool, err error) string {
	return s.RenderListViewWithSync(width, height, hostsInterface, selectedIdx, searchQuery, isSearching, err, "", nil)
}

// RenderListViewWithSync renders the main two-panel layout with sync status
func (s *Styles) RenderListViewWithSync(width, height int, hostsInterface interface{}, selectedIdx int, searchQuery string, isSearching bool, err error, syncStatus string, syncActivity *SyncActivity) string {
	// Type assertion for hosts
	hosts, ok := hostsInterface.([]interface{})
	if !ok {
		return "Error: Invalid hosts data"
	}

	var view strings.Builder

	// Header
	header := s.RenderHeader(width, searchQuery, isSearching)
	view.WriteString(header)
	view.WriteString("\n")

	// Calculate available space
	headerHeight := lipgloss.Height(header)
	footerHeight := 3
	if err != nil {
		footerHeight++
	}
	if syncActivity != nil && syncActivity.Active {
		footerHeight++
	}
	availableHeight := height - headerHeight - footerHeight - 2

	// Two-panel layout
	leftPanelWidth := width / 3
	rightPanelWidth := width - leftPanelWidth - 3

	leftPanel := s.RenderHostList(hosts, selectedIdx, leftPanelWidth, availableHeight)
	rightPanel := s.RenderHostDetails(hosts, selectedIdx, rightPanelWidth, availableHeight)

	// Join panels side by side
	panels := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftPanel,
		rightPanel,
	)
	view.WriteString(panels)
	view.WriteString("\n")

	// Footer
	footer := s.RenderFooterWithSync(width, err, syncStatus, syncActivity)
	view.WriteString(footer)

	return view.String()
}

// RenderHeader renders the header bar
func (s *Styles) RenderHeader(width int, searchQuery string, isSearching bool) string {
	title := s.Title.Render("🔐 SSH Manager")

	var right string
	if isSearching {
		right = s.HelpValue.Render(fmt.Sprintf("Search: %s_", searchQuery))
	} else if searchQuery != "" {
		right = s.HelpValue.Render(fmt.Sprintf("Filter: %s", searchQuery))
	} else {
		right = s.HelpKey.Render("[?]") + " " + s.HelpValue.Render("Help")
	}

	// Calculate spacing
	titleWidth := lipgloss.Width(title)
	rightWidth := lipgloss.Width(right)
	spacing := width - titleWidth - rightWidth - 4
	if spacing < 0 {
		spacing = 0
	}

	header := lipgloss.JoinHorizontal(
		lipgloss.Center,
		title,
		strings.Repeat(" ", spacing),
		right,
	)

	return s.Header.Width(width - 2).Render(header)
}

// RenderHostList renders the left panel with the list of hosts
func (s *Styles) RenderHostList(hostsInterface []interface{}, selectedIdx int, width int, height int) string {
	var list strings.Builder

	// Header
	listHeader := s.ListHeader.Render("HOSTS")
	list.WriteString(listHeader)
	list.WriteString("\n")

	if len(hostsInterface) == 0 {
		emptyMsg := s.DetailValue.Foreground(ColorTextDim).Render("No hosts found")
		list.WriteString("\n")
		list.WriteString(emptyMsg)
	} else {
		// PanelBorder is rendered at (width-4) and has border(2) + padding(4),
		// so usable inner width is ~width-10.
		itemBoxWidth := width - 10
		if itemBoxWidth < 12 {
			itemBoxWidth = 12
		}
		textWidth := itemBoxWidth - 4 // ListItem padding(0,2) consumes 4 columns
		if textWidth < 6 {
			textWidth = 6
		}

		// Render hosts
		for i, hostInterface := range hostsInterface {
			host, ok := hostInterface.(map[string]interface{})
			if !ok {
				continue
			}
			kind, _ := host["Kind"].(string)
			if kind == "group" {
				groupName, _ := host["GroupName"].(string)
				count, _ := host["Count"].(int)
				collapsed, _ := host["Collapsed"].(bool)
				isSelected := i == selectedIdx
				prefix := "▾ "
				if collapsed {
					prefix = "▸ "
				}
				line := truncateString(prefix+groupName+fmt.Sprintf(" (%d)", count), textWidth)
				style := s.ListItem
				if isSelected {
					style = s.ListItemSelected
				}
				list.WriteString(style.Width(itemBoxWidth).Render(line))
				list.WriteString("\n")
				continue
			}
			if kind == "new_group" {
				isSelected := i == selectedIdx
				line := truncateString("+ New Group", textWidth)
				style := s.ListItem
				if isSelected {
					style = s.ListItemSelected
				}
				list.WriteString(style.Width(itemBoxWidth).Render(line))
				list.WriteString("\n")
				continue
			}

			label, _ := host["Label"].(string)
			hostname, _ := host["Hostname"].(string)
			hasKey, _ := host["HasKey"].(bool)
			indent, _ := host["Indent"].(int)
			showIcons := true
			if v, ok := host["ShowIcons"].(bool); ok {
				showIcons = v
			}

			isSelected := i == selectedIdx
			_ = hostname // keep host in details only; list shows label/host display name only.

			display := strings.TrimSpace(label)
			if display == "" {
				display = strings.TrimSpace(hostname)
			}

			icon := "  "
			if isSelected {
				icon = "▸ "
			} else if showIcons && hasKey {
				icon = "⚡ "
			}

			line := truncateString(strings.Repeat(" ", indent)+icon+display, textWidth)
			if isSelected {
				item := s.ListItemSelected.Width(itemBoxWidth).Render(line)
				list.WriteString(item)
				list.WriteString("\n")
				continue
			}
			item := s.ListItem.Width(itemBoxWidth).Render(line)
			list.WriteString(item)
			list.WriteString("\n")
		}
	}

	// Pad remaining space
	currentHeight := lipgloss.Height(list.String())
	for currentHeight < height {
		list.WriteString("\n")
		currentHeight++
	}

	return s.PanelBorder.
		Width(width - 4).
		Height(height).
		Render(list.String())
}

func truncateString(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= width {
		return s
	}
	r := []rune(s)
	if len(r) <= 1 {
		return ""
	}
	out := string(r)
	for lipgloss.Width(out) > width-1 && len(r) > 0 {
		r = r[:len(r)-1]
		out = string(r)
	}
	return out + "…"
}

// RenderHostDetails renders the right panel with selected host details
func (s *Styles) RenderHostDetails(hostsInterface []interface{}, selectedIdx int, width int, height int) string {
	var details strings.Builder

	// Header
	detailHeader := s.ListHeader.Render("DETAILS")
	details.WriteString(detailHeader)
	details.WriteString("\n\n")

	if len(hostsInterface) == 0 || selectedIdx >= len(hostsInterface) {
		emptyMsg := s.DetailValue.Foreground(ColorTextDim).Render("No host selected")
		details.WriteString(emptyMsg)
	} else {
		host, ok := hostsInterface[selectedIdx].(map[string]interface{})
		if !ok {
			details.WriteString("Error: Invalid host data")
		} else {
			kind, _ := host["Kind"].(string)
			if kind == "group" {
				groupName, _ := host["GroupName"].(string)
				count, _ := host["Count"].(int)
				details.WriteString(s.renderDetailRow("Group:", groupName))
				details.WriteString(s.renderDetailRow("Hosts:", fmt.Sprintf("%d", count)))
				details.WriteString("\n")
				details.WriteString(s.HelpValue.Foreground(ColorTextDim).Render("Enter: collapse/expand • a: add host • e: rename • d: delete"))
				goto padDetails
			}
			if kind == "new_group" {
				details.WriteString(s.renderDetailRow("Action:", "Create a new group"))
				details.WriteString("\n")
				details.WriteString(s.HelpValue.Foreground(ColorTextDim).Render("Press Enter or a"))
				goto padDetails
			}

			// Extract host fields
			label, _ := host["Label"].(string)
			groupName, _ := host["GroupName"].(string)
			hostname, _ := host["Hostname"].(string)
			username, _ := host["Username"].(string)
			port, _ := host["Port"].(int)
			hasKey, _ := host["HasKey"].(bool)
			keyType, _ := host["KeyType"].(string)
			mounted, _ := host["Mounted"].(bool)
			mountPath, _ := host["MountPath"].(string)
			lastConnected, _ := host["LastConnected"].(*time.Time)

			// Render details
			if strings.TrimSpace(label) != "" {
				details.WriteString(s.renderDetailRow("Label:", label))
			}
			if strings.TrimSpace(groupName) != "" {
				details.WriteString(s.renderDetailRow("Group:", groupName))
			}
			details.WriteString(s.renderDetailRow("Host:", hostname))
			details.WriteString(s.renderDetailRow("Username:", username))
			details.WriteString(s.renderDetailRow("Port:", fmt.Sprintf("%d", port)))

			// Status
			var status string
			if hasKey {
				status = s.StatusReady.Render("Ready ✓")
			} else {
				status = s.StatusWarning.Render("No key")
			}
			details.WriteString(s.renderDetailRow("Status:", status))

			// Key type
			if keyType != "" {
				details.WriteString(s.renderDetailRow("Key Type:", keyType))
			}

			// Mount status
			if mounted {
				details.WriteString(s.renderDetailRow("Mount:", s.StatusReady.Render("Mounted ✓")))
				if strings.TrimSpace(mountPath) != "" {
					details.WriteString(s.renderDetailRow("Local:", mountPath))
				}
			} else {
				details.WriteString(s.renderDetailRow("Mount:", s.DetailValue.Foreground(ColorTextDim).Render("Not mounted")))
			}

			// Last connected
			if lastConnected != nil {
				timeAgo := formatTimeAgo(*lastConnected)
				details.WriteString(s.renderDetailRow("Last SSH:", timeAgo))
			} else {
				details.WriteString(s.renderDetailRow("Last SSH:", s.DetailValue.Foreground(ColorTextDim).Render("Never")))
			}
		}
	}

padDetails:

	// Pad remaining space
	currentHeight := lipgloss.Height(details.String())
	for currentHeight < height {
		details.WriteString("\n")
		currentHeight++
	}

	return s.PanelBorder.
		Width(width - 4).
		Height(height).
		Render(details.String())
}

// renderDetailRow renders a single detail row
func (s *Styles) renderDetailRow(label, value string) string {
	row := lipgloss.JoinHorizontal(
		lipgloss.Top,
		s.DetailLabel.Render(label),
		s.DetailValue.Render(value),
	)
	return s.DetailRow.Render(row) + "\n"
}

// RenderFooter renders the footer with keybindings
func (s *Styles) RenderFooter(width int, err error) string {
	return s.RenderFooterWithSync(width, err, "", nil)
}

// RenderFooterWithSync renders the footer with keybindings and optional sync status
func (s *Styles) RenderFooterWithSync(width int, err error, syncStatus string, syncActivity *SyncActivity) string {
	var footer strings.Builder

	// Show notice if present
	if err != nil {
		footer.WriteString(s.renderFooterNotice(err.Error()))
		footer.WriteString("\n")
	}

	if syncActivity != nil && syncActivity.Active {
		footer.WriteString(s.renderSyncActivityLine(width-4, syncActivity))
		footer.WriteString("\n")
	}

	// Keybindings
	bindings := []string{
		s.HelpKey.Render("[↑/↓]") + " " + s.HelpValue.Render("Navigate"),
		s.HelpKey.Render("[Enter]") + " " + s.HelpValue.Render("SSH"),
		s.HelpKey.Render("[S]") + " " + s.HelpValue.Render("Arm SFTP"),
		s.HelpKey.Render("[M]") + " " + s.HelpValue.Render("Arm Mount"),
		s.HelpKey.Render("[Y]") + " " + s.HelpValue.Render("Sync"),
		s.HelpKey.Render("[,]") + " " + s.HelpValue.Render("Settings"),
		s.HelpKey.Render("[a]") + " " + s.HelpValue.Render("Add"),
		s.HelpKey.Render("[e]") + " " + s.HelpValue.Render("Edit"),
		s.HelpKey.Render("[d]") + " " + s.HelpValue.Render("Delete"),
		s.HelpKey.Render("[Ctrl+G]") + " " + s.HelpValue.Render("New Group"),
		s.HelpKey.Render("[/]") + " " + s.HelpValue.Render("Search"),
		s.HelpKey.Render("[q]") + " " + s.HelpValue.Render("Quit"),
	}

	footerText := strings.Join(bindings, s.HelpSep.String())

	// Add sync status if provided
	if syncStatus != "" && (syncActivity == nil || !syncActivity.Active) {
		footerText += s.HelpSep.String() + s.HelpValue.Foreground(ColorTextDim).Render("Sync: "+syncStatus)
	}

	footer.WriteString(footerText)

	return s.Footer.Width(width - 2).Render(footer.String())
}

func (s *Styles) renderSyncActivityLine(width int, activity *SyncActivity) string {
	if width < 16 {
		return s.HelpValue.Render("Syncing...")
	}

	frames := []string{"|", "/", "-", "\\"}
	icon := frames[activity.Frame%len(frames)]
	stage := strings.TrimSpace(activity.Stage)
	label := icon + " Syncing"
	if stage != "" {
		label += " (" + stage + ")"
	}

	labelStyled := s.HelpValue.Foreground(ColorSecondary).Render(label)
	barWidth := width - lipgloss.Width(label) - 1
	if barWidth < 10 {
		return labelStyled
	}

	return lipgloss.JoinHorizontal(
		lipgloss.Center,
		labelStyled,
		" ",
		s.HelpValue.Foreground(ColorPrimary).Render(renderSyncBar(barWidth, activity.Frame, activity.Progress)),
	)
}

func renderSyncBar(width int, frame int, progress float64) string {
	if width <= 2 {
		return "[]"
	}
	inner := width - 2
	if inner < 1 {
		inner = 1
	}
	if progress < 0 {
		progress = 0
	}
	if progress > 1 {
		progress = 1
	}

	filled := int(progress * float64(inner))
	if filled > inner {
		filled = inner
	}

	bar := make([]rune, inner)
	for i := range bar {
		bar[i] = '-'
	}
	for i := 0; i < filled; i++ {
		bar[i] = '='
	}

	if filled < inner {
		head := frame % inner
		if head < filled {
			head = filled
		}
		if head >= inner {
			head = inner - 1
		}
		bar[head] = '>'
	}

	return "[" + string(bar) + "]"
}

type footerNoticeKind int

const (
	footerNoticeInfo footerNoticeKind = iota
	footerNoticeSuccess
	footerNoticeWarning
	footerNoticeError
)

func (s *Styles) renderFooterNotice(message string) string {
	kind := classifyFooterNotice(message)
	message = strings.TrimSpace(message)

	stripPrefix := func(prefixes ...string) {
		for _, p := range prefixes {
			if strings.HasPrefix(message, p) {
				message = strings.TrimSpace(strings.TrimPrefix(message, p))
				return
			}
		}
	}

	switch kind {
	case footerNoticeSuccess:
		stripPrefix("✓", "✔")
		return s.Success.Render("✓ " + message)
	case footerNoticeInfo:
		stripPrefix("ℹ", "i", "info:")
		return s.Info.Render("ℹ " + message)
	case footerNoticeWarning:
		stripPrefix("⚠", "!", "warning:")
		return s.Warning.Render("! " + message)
	default:
		stripPrefix("✗", "×", "error:")
		return s.Error.Render("✗ " + message)
	}
}

func classifyFooterNotice(message string) footerNoticeKind {
	msg := strings.TrimSpace(message)
	lower := strings.ToLower(msg)

	if strings.HasPrefix(msg, "✓") || strings.HasPrefix(msg, "✔") {
		return footerNoticeSuccess
	}
	if strings.HasPrefix(msg, "ℹ") || strings.HasPrefix(lower, "info") {
		return footerNoticeInfo
	}
	if strings.Contains(lower, "disconnected") ||
		strings.Contains(lower, "connected") ||
		strings.Contains(lower, "armed") ||
		strings.Contains(lower, "syncing") ||
		strings.Contains(lower, "database deleted") ||
		strings.Contains(lower, "db deleted") {
		return footerNoticeInfo
	}
	if strings.HasPrefix(msg, "⚠") ||
		strings.HasPrefix(lower, "warning") ||
		strings.Contains(lower, "must be") ||
		strings.Contains(lower, "cannot be") ||
		strings.Contains(lower, "invalid private key") {
		return footerNoticeWarning
	}
	return footerNoticeError
}

// RenderHelpView renders the help screen
func (s *Styles) RenderHelpView(width, height int) string {
	var help strings.Builder

	title := s.ModalTitle.Render("⌨️  Keyboard Shortcuts")
	help.WriteString(title)
	help.WriteString("\n\n")

	shortcuts := []struct {
		key  string
		desc string
	}{
		{"↑/↓ or j/k", "Navigate up/down"},
		{"Ctrl+U/D", "Page up/down"},
		{"Home/End or g/G", "Jump to top/bottom"},
		{"Enter", "Connect host or toggle group"},
		{"S then Enter", "Connect via SFTP"},
		{"M then Enter", "Mount/unmount in Finder (beta)"},
		{",", "Open settings"},
		{"Ctrl+G", "Create group"},
		{"a or Ctrl+N", "Add new host"},
		{"e", "Edit selected host"},
		{"d or Delete", "Delete selected host"},
		{"/ or Ctrl+F", "Search/filter hosts"},
		{"?", "Toggle this help"},
		{"q or Ctrl+C", "Quit application"},
	}

	for _, sc := range shortcuts {
		row := lipgloss.JoinHorizontal(
			lipgloss.Top,
			s.FormLabel.Width(20).Render(sc.key),
			s.DetailValue.Render(sc.desc),
		)
		help.WriteString(row)
		help.WriteString("\n")
	}

	help.WriteString("\n")
	footer := s.HelpValue.Foreground(ColorTextDim).Render("Press ? or Esc to close")
	help.WriteString(footer)

	// Center the help modal
	helpContent := help.String()
	modalWidth := 60
	modalHeight := lipgloss.Height(helpContent) + 4

	modal := s.Modal.
		Width(modalWidth).
		Height(modalHeight).
		Render(helpContent)

	// Center in terminal
	verticalPadding := (height - modalHeight) / 2
	horizontalPadding := (width - modalWidth) / 2

	if verticalPadding < 0 {
		verticalPadding = 0
	}
	if horizontalPadding < 0 {
		horizontalPadding = 0
	}

	centeredModal := lipgloss.NewStyle().
		Padding(verticalPadding, horizontalPadding).
		Render(modal)

	return centeredModal
}

// formatTimeAgo formats a time duration as a human-readable string
func formatTimeAgo(t time.Time) string {
	duration := time.Since(t)

	if duration < time.Minute {
		return "Just now"
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
	} else if duration < 30*24*time.Hour {
		weeks := int(duration.Hours() / 24 / 7)
		if weeks == 1 {
			return "1 week ago"
		}
		return fmt.Sprintf("%d weeks ago", weeks)
	} else {
		months := int(duration.Hours() / 24 / 30)
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	}
}
