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
			description: "Enables macOS Finder mounts (requires FUSE-T + sshfs).",
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
	}
	return rows
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
