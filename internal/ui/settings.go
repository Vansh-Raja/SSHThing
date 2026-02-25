package ui

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/Vansh-Raja/SSHThing/internal/config"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

type settingsRow struct {
	label       string
	value       string
	description string
	disabled    bool
}

type UpdateSettingsState struct {
	ChannelLabel   string
	VersionLabel   string
	PathHealth     string
	Checking       bool
	Applying       bool
	CanApply       bool
	CanFixPath     bool
	PlatformHint   string
	LastCheckedAgo string
}

type TokenManagerTokenRow struct {
	TokenID  string
	Name     string
	Hosts    int
	LastUsed string
	Status   string
	Synced   bool
}

type TokenManagerHostRow struct {
	HostID  int
	Label   string
	Detail  string
	HasAuth bool
}

func (s *Styles) RenderSettingsView(width, height int, cfg config.Config, updateState UpdateSettingsState, selectedIdx int, editing bool, input textinput.Model, err error) string {
	rows := buildSettingsRows(cfg, updateState)
	if selectedIdx < 0 {
		selectedIdx = 0
	}
	if selectedIdx >= len(rows) {
		selectedIdx = len(rows) - 1
	}

	// Dimensions
	modalWidth := (width * 85) / 100
	if modalWidth > 86 {
		modalWidth = 86
	}
	if modalWidth < 60 {
		modalWidth = 60
	}
	innerWidth := modalWidth - 6 // border(2)+padding(4)
	itemBoxWidth := innerWidth
	contentWidth := itemBoxWidth - 4 // ListItem padding(0,2) consumes 4 columns
	if contentWidth < 24 {
		contentWidth = 24
	}

	var b strings.Builder
	b.WriteString(s.ModalTitle.Render("⚙ Settings"))
	b.WriteString("\n")

	// Optional notice / error line
	if err != nil {
		b.WriteString(s.renderFooterNotice(err.Error()))
		b.WriteString("\n")
	}

	// Render list
	for i, r := range rows {
		left := r.label
		right := r.value
		if r.disabled {
			left = s.DetailValue.Foreground(ColorTextDim).Render(left)
			right = s.DetailValue.Foreground(ColorTextDim).Render(right)
		}

		// NOTE: ListItem styles include left/right padding (0,2). The rendered content must fit
		// in `contentWidth` or Lipgloss will wrap and create multi-line items.
		marker := "  "
		if i == selectedIdx {
			marker = "▸ "
		}
		lineWidth := contentWidth - lipgloss.Width(marker)
		if lineWidth < 20 {
			lineWidth = 20
		}

		valueWidth := 24
		if valueWidth > lineWidth-10 {
			valueWidth = lineWidth - 10
		}
		if valueWidth < 10 {
			valueWidth = 10
		}
		labelWidth := lineWidth - valueWidth - 2 // spacer=2
		if labelWidth < 12 {
			labelWidth = 12
		}
		if labelWidth+2+valueWidth > lineWidth {
			// last-resort: shrink value
			valueWidth = max(10, lineWidth-labelWidth-2)
		}

		labelText := truncateString(left, labelWidth)
		valueText := truncateString(right, valueWidth)
		labelView := lipgloss.NewStyle().Width(labelWidth).Render(labelText)
		valueView := lipgloss.NewStyle().Width(valueWidth).Align(lipgloss.Right).Render(valueText)

		line := lipgloss.JoinHorizontal(lipgloss.Center, labelView, lipgloss.NewStyle().Width(2).Render(""), valueView)
		full := marker + truncateString(line, lineWidth)
		if i == selectedIdx {
			b.WriteString(s.ListItemSelected.Width(itemBoxWidth).Render(full))
		} else {
			b.WriteString(s.ListItem.Width(itemBoxWidth).Render(full))
		}
		b.WriteString("\n")
	}

	// Selected description
	b.WriteString("\n")
	desc := rows[selectedIdx].description
	if strings.TrimSpace(desc) != "" {
		b.WriteString(s.HelpValue.Render(desc))
		b.WriteString("\n")
	}

	// Edit box if editing
	if editing {
		b.WriteString("\n")
		input.Prompt = ""
		inputView := s.FormInputFocused.Width(innerWidth).Render(input.View())
		b.WriteString(inputView)
		b.WriteString("\n")
	}

	// Footer help
	b.WriteString("\n")
	help := "[↑/↓] Navigate • [Enter] Change • [Space] Toggle • [←/→] Adjust • [Esc] Save & Back"
	if editing {
		help = "[Enter] Apply • [Esc] Cancel"
	}
	b.WriteString(s.HelpValue.Foreground(ColorTextDim).Render(help))

	box := s.Modal.Width(modalWidth).Render(b.String())
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}

func buildSettingsRows(cfg config.Config, updateState UpdateSettingsState) []settingsRow {
	termCustomDisabled := cfg.SSH.TermMode != config.TermCustom
	unixBackendDisabled := runtime.GOOS == "windows" || !cfg.SSH.PasswordAutoLogin
	syncDisabled := !cfg.Sync.Enabled

	rows := []settingsRow{
		{
			label:       "UI: Vim navigation",
			value:       onOff(cfg.UI.VimMode),
			description: "Enables j/k/h/l navigation in lists and settings.",
		},
		{
			label:       "UI: Show icons",
			value:       onOff(cfg.UI.ShowIcons),
			description: "Toggles extra icons like ⚡ in the host list.",
		},
		{
			label:       "SSH: Host key policy",
			value:       string(cfg.SSH.HostKeyPolicy),
			description: "Controls StrictHostKeyChecking (accept-new, strict, off).",
		},
		{
			label:       "SSH: Keepalive (seconds)",
			value:       fmt.Sprintf("%d", cfg.SSH.KeepAliveSeconds),
			description: "Sets ServerAliveInterval passed to ssh/sftp/sshfs.",
		},
		{
			label:       "SSH: TERM mode",
			value:       string(cfg.SSH.TermMode),
			description: "Controls TERM passed to sessions (auto includes Ghostty fix).",
		},
		{
			label:       "SSH: TERM custom",
			value:       cfg.SSH.TermCustom,
			description: "Editable only when TERM mode is 'custom'.",
			disabled:    termCustomDisabled,
		},
		{
			label:       "SSH: Password auto-login",
			value:       onOff(cfg.SSH.PasswordAutoLogin),
			description: "Stores encrypted SSH passwords and auto-fills for password-auth hosts.",
		},
		{
			label:       "SSH: Unix password backend",
			value:       string(cfg.SSH.PasswordBackendUnix),
			description: "Linux/macOS backend order for password auto-login (sshpass_first or askpass_first).",
			disabled:    unixBackendDisabled,
		},
		{
			label:       "Mounts: Enabled (beta)",
			value:       onOff(cfg.Mount.Enabled),
			description: mountDescription(),
		},
		{
			label:       "Mounts: Default remote path",
			value:       emptyAs(cfg.Mount.DefaultRemotePath, "(home)"),
			description: "Empty mounts remote home. Set to e.g. /var/www if desired.",
		},
		{
			label:       "Mounts: Quit behavior",
			value:       string(cfg.Mount.QuitBehavior),
			description: "Prompt, always unmount, or leave mounted when quitting.",
		},
		{
			label:       "Sync: Enabled",
			value:       onOff(cfg.Sync.Enabled),
			description: "Enables Git-based sync for hosts across devices.",
		},
		{
			label:       "Sync: Repository URL",
			value:       emptyAs(cfg.Sync.RepoURL, "(not set)"),
			description: "Git repository URL (e.g. git@github.com:user/hosts.git).",
			disabled:    syncDisabled,
		},
		{
			label:       "Sync: SSH key path",
			value:       emptyAs(cfg.Sync.SSHKeyPath, "(auto)"),
			description: "Path to SSH key for Git auth. Empty uses default keys.",
			disabled:    syncDisabled,
		},
		{
			label:       "Sync: Branch",
			value:       cfg.Sync.Branch,
			description: "Git branch to sync with (default: main).",
			disabled:    syncDisabled,
		},
		{
			label:       "Sync: Local path",
			value:       emptyAs(cfg.Sync.LocalPath, "(default)"),
			description: "Local directory for sync repo. Empty uses default.",
			disabled:    syncDisabled,
		},
		{
			label:       "Updates: Channel",
			value:       emptyAs(updateState.ChannelLabel, "(unknown)"),
			description: "Detected install/update channel.",
			disabled:    true,
		},
		{
			label:       "Updates: Current -> Latest",
			value:       emptyAs(updateState.VersionLabel, "(not checked)"),
			description: "Latest stable release comparison.",
			disabled:    true,
		},
		{
			label:       "Updates: Check now",
			value:       ternary(updateState.Checking, "Checking...", "Run"),
			description: "Checks GitHub for the latest stable release.",
			disabled:    updateState.Applying,
		},
		{
			label:       "Updates: Apply now",
			value:       ternary(updateState.Applying, "Applying...", "Run"),
			description: "Applies available update using your detected channel.",
			disabled:    !updateState.CanApply || updateState.Checking || updateState.Applying,
		},
		{
			label:       "Updates: PATH health",
			value:       emptyAs(updateState.PathHealth, "(not checked)"),
			description: "Warns when another sshthing binary shadows the intended one.",
			disabled:    true,
		},
		{
			label:       "Updates: Fix PATH",
			value:       "Run",
			description: "Moves the intended install path ahead of stale sshthing entries.",
			disabled:    !updateState.CanFixPath || updateState.Checking || updateState.Applying,
		},
		{
			label:       "Automation: Manage tokens",
			value:       "Open",
			description: "Create/revoke immutable automation tokens scoped to selected hosts.",
		},
		{
			label:       "Automation: Sync token definitions",
			value:       onOff(cfg.Automation.SyncTokenDefinitions),
			description: "Sync token names/scope/revocations across devices (no usable token secrets).",
			disabled:    syncDisabled,
		},
	}
	return rows
}

func (s *Styles) RenderTokenManagerView(width, height int, tokens []TokenManagerTokenRow, hosts []TokenManagerHostRow, tokenIdx int, hostIdx int, createNameMode bool, createScopeMode bool, selectedHostIDs map[int]bool, nameInput textinput.Model, revealOpen bool, revealValue string, revealCopied bool, err error) string {
	modalWidth := min(118, max(76, width-4))
	if width < 80 {
		modalWidth = max(60, width-2)
	}

	if revealOpen {
		inner := modalWidth - (s.Modal.GetBorderLeftSize() + s.Modal.GetBorderRightSize() + s.Modal.GetPaddingLeft() + s.Modal.GetPaddingRight())
		if inner < 30 {
			inner = 30
		}
		var b strings.Builder
		b.WriteString(s.ModalTitle.Render("Token Created"))
		b.WriteString("\n")
		b.WriteString(s.HelpValue.Render("This token is shown only once. Save it now."))
		b.WriteString("\n\n")
		tokenBlock := s.PanelBorder.Width(inner).Render(truncateString(revealValue, inner-4))
		b.WriteString(tokenBlock)
		b.WriteString("\n")
		if revealCopied {
			b.WriteString(s.HelpValue.Foreground(ColorAccent).Render("Copied to clipboard."))
			b.WriteString("\n")
		}
		b.WriteString("\n")
		b.WriteString(s.HelpValue.Foreground(ColorTextDim).Render("[C] Copy token • [Esc] Close"))
		box := s.Modal.Width(modalWidth).Render(b.String())
		return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
	}

	modalInner := modalWidth - (s.Modal.GetBorderLeftSize() + s.Modal.GetBorderRightSize() + s.Modal.GetPaddingLeft() + s.Modal.GetPaddingRight())
	if modalInner < 40 {
		modalInner = 40
	}

	gap := 3
	leftOuter := (modalInner - gap) / 2
	rightOuter := modalInner - leftOuter - gap
	if leftOuter < 24 {
		leftOuter = 24
		rightOuter = modalInner - leftOuter - gap
	}

	panelInnerPad := s.PanelBorder.GetBorderLeftSize() + s.PanelBorder.GetBorderRightSize() + s.PanelBorder.GetPaddingLeft() + s.PanelBorder.GetPaddingRight()
	leftInner := max(12, leftOuter-panelInnerPad)
	rightInner := max(12, rightOuter-panelInnerPad)

	bodyHeight := min(18, max(10, height-18))
	leftTextWidth := max(8, leftInner-4)
	rightTextWidth := max(8, rightInner-4)

	var left strings.Builder
	left.WriteString(s.ModalTitle.Render("Tokens"))
	left.WriteString("\n")
	if len(tokens) == 0 {
		left.WriteString(s.HelpValue.Render("No tokens created yet."))
		left.WriteString("\n")
	} else {
		for i, t := range tokens {
			prefix := "  "
			style := s.ListItem
			if i == tokenIdx && !createNameMode && !createScopeMode {
				prefix = "▸ "
				style = s.ListItemSelected
			}
			scope := "local"
			if t.Synced {
				scope = "sync"
			}
			line := fmt.Sprintf("%s%s [%s/%s] h:%d used:%s", prefix, t.Name, t.Status, scope, t.Hosts, t.LastUsed)
			left.WriteString(style.Width(leftInner).Render(truncateString(line, leftTextWidth)))
			left.WriteString("\n")
		}
	}
	leftPanel := s.PanelBorder.Width(leftOuter).Height(bodyHeight).Render(padToHeight(left.String(), bodyHeight))

	var right strings.Builder
	if createNameMode {
		right.WriteString(s.ModalTitle.Render("Create Token: Name"))
		right.WriteString("\n")
		right.WriteString(s.HelpValue.Render("Enter a name, then press Enter."))
		right.WriteString("\n\n")
		nameInput.Prompt = ""
		right.WriteString(s.FormInputFocused.Width(rightInner).Render(nameInput.View()))
		right.WriteString("\n")
	} else if createScopeMode {
		right.WriteString(s.ModalTitle.Render("Create Token: Scope"))
		right.WriteString("\n")
		if len(hosts) == 0 {
			right.WriteString(s.HelpValue.Render("No hosts available."))
			right.WriteString("\n")
		} else {
			for i, h := range hosts {
				marker := "[ ]"
				if selectedHostIDs[h.HostID] {
					marker = "[x]"
				}
				prefix := "  "
				style := s.ListItem
				if i == hostIdx {
					prefix = "▸ "
					style = s.ListItemSelected
				}
				suffix := ""
				if !h.HasAuth {
					suffix = " (no secret)"
				}
				line := fmt.Sprintf("%s%s %s%s", prefix, marker, h.Label, suffix)
				right.WriteString(style.Width(rightInner).Render(truncateString(line, rightTextWidth)))
				right.WriteString("\n")
			}
		}
	} else {
		right.WriteString(s.ModalTitle.Render("Token Actions"))
		right.WriteString("\n")
		right.WriteString(s.HelpValue.Render("Scopes are immutable."))
		right.WriteString("\n")
		right.WriteString(s.HelpValue.Render("Create new token to change host access."))
		right.WriteString("\n\n")
		right.WriteString(s.DetailLabel.Render("Selected token"))
		right.WriteString("\n")
		if len(tokens) == 0 {
			right.WriteString(s.HelpValue.Render("(none)"))
			right.WriteString("\n")
		} else {
			t := tokens[tokenIdx]
			right.WriteString(s.DetailValue.Render(truncateString(t.Name, rightInner)))
			right.WriteString("\n")
			right.WriteString(s.HelpValue.Render("id: " + t.TokenID))
			right.WriteString("\n")
			right.WriteString(s.HelpValue.Render(fmt.Sprintf("hosts: %d  status: %s", t.Hosts, t.Status)))
			right.WriteString("\n")
		}
	}
	rightPanel := s.PanelBorder.Width(rightOuter).Height(bodyHeight).Render(padToHeight(right.String(), bodyHeight))

	body := lipgloss.JoinHorizontal(lipgloss.Top,
		leftPanel,
		lipgloss.NewStyle().Width(gap).Render(""),
		rightPanel,
	)

	footer := "[N] New token • [A] Activate • [R] Revoke • [D] Delete revoked • [Esc] Back"
	if createNameMode {
		footer = "[Enter] Continue • [Esc] Cancel"
	}
	if createScopeMode {
		footer = "[Space] Toggle host • [Enter] Create token • [Esc] Cancel"
	}

	var b strings.Builder
	b.WriteString(s.ModalTitle.Render("Automation Tokens (Immutable scope)"))
	b.WriteString("\n")
	if err != nil {
		b.WriteString(s.renderFooterNotice(err.Error()))
		b.WriteString("\n")
	}
	b.WriteString(body)
	b.WriteString("\n")
	b.WriteString(s.HelpValue.Foreground(ColorTextDim).Render(footer))

	box := s.Modal.Width(modalWidth).Render(b.String())
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}

func padToHeight(content string, target int) string {
	h := lipgloss.Height(content)
	if h >= target {
		return content
	}
	var b strings.Builder
	b.WriteString(content)
	for i := h; i < target; i++ {
		b.WriteString("\n")
	}
	return b.String()
}

func ternary(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}

func onOff(v bool) string {
	if v {
		return "On"
	}
	return "Off"
}

func emptyAs(v, alt string) string {
	if strings.TrimSpace(v) == "" {
		return alt
	}
	return v
}

func mountDescription() string {
	switch runtime.GOOS {
	case "linux":
		return "Enables SSHFS mounts (requires sshfs + FUSE)."
	default:
		return "Enables Finder mounts (requires FUSE-T + sshfs)."
	}
}
