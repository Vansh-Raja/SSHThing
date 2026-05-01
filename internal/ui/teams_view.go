package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type TeamsHomeStatusLine struct {
	Kind    string
	Message string
}

type TeamsHomeTeamSummary struct {
	Name      string
	Slug      string
	Role      string
	HostCount int
	TeamCount int
}

type TeamsHomeListItem struct {
	IsGroup            bool
	GroupName          string
	HostCount          int
	Selected           bool
	Label              string
	Hostname           string
	Username           string
	Port               int
	Group              string
	CredentialMode     string
	CredentialType     string
	LastConnectedLabel string
	Tags               []string
	Notes              string
	Health             *HostHealthView
}

type TeamsHomeViewParams struct {
	Page              int
	SessionValid      bool
	HasTeams          bool
	CurrentTeam       TeamsHomeTeamSummary
	Items             []TeamsHomeListItem
	StatusLines       []TeamsHomeStatusLine
	FooterText        string
	HealthDisplayMode string
}

func (r *Renderer) RenderTeamsView(p TeamsHomeViewParams) string {
	layout := r.buildHomeFrameLayout(len(p.StatusLines))
	listBlock := lipgloss.NewStyle().Width(layout.listW).Render(r.padListLines(r.renderTeamsListLines(p, layout), layout.bodyH))
	detail := clampBlockHeight(r.renderTeamsDetail(p, layout.detailW, layout.bodyH), layout.bodyH, r.Icons.Truncation+" more")
	detailBlock := lipgloss.NewStyle().Width(layout.detailW).Foreground(r.Theme.Subtext).
		Render(detail)
	body := r.renderHomeBody(listBlock, detailBlock, layout, p.Page)

	headerLine := r.RenderHeader(r.renderTeamsHeaderSummary(p), 0, 0)
	return r.renderHomeFrame(headerLine, body, r.renderTeamsStatusLines(p.StatusLines), r.teamsFooterText(p))
}

func (r *Renderer) renderTeamsHeaderSummary(p TeamsHomeViewParams) string {
	return fmt.Sprintf("%d teams  %d hosts", p.CurrentTeam.TeamCount, p.CurrentTeam.HostCount)
}

func (r *Renderer) teamsFooterText(p TeamsHomeViewParams) string {
	if r.W < 95 && p.SessionValid && p.HasTeams {
		return "\u2191\u2193 nav · enter connect · R health · r refresh · q quit"
	}
	if strings.TrimSpace(p.FooterText) != "" {
		return p.FooterText
	}
	if !p.SessionValid || !p.HasTeams {
		return "enter create team  / search  ? commands  , settings  shift+tab cycle  T personal mode  q quit"
	}
	return "\u2191\u2193 nav  \u23CE connect  R health  a add  e edit  d del  ctrl+1..9 switch team  / search  r refresh  , settings  shift+tab cycle  T personal mode  q quit"
}

func (r *Renderer) renderTeamsStatusLines(lines []TeamsHomeStatusLine) []string {
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		message := strings.TrimSpace(line.Message)
		if message == "" {
			continue
		}
		switch line.Kind {
		case "success":
			out = append(out, lipgloss.NewStyle().Foreground(r.Theme.Green).Render(message))
		case "warning":
			out = append(out, lipgloss.NewStyle().Foreground(r.Theme.Yellow).Render(message))
		case "info":
			out = append(out, lipgloss.NewStyle().Foreground(r.Theme.Sky).Render(message))
		default:
			out = append(out, lipgloss.NewStyle().Foreground(r.Theme.Red).Render(message))
		}
	}
	return out
}

func (r *Renderer) renderTeamsListLines(p TeamsHomeViewParams, layout homeFrameLayout) []string {
	if !p.SessionValid {
		return []string{
			lipgloss.NewStyle().Foreground(r.Theme.Text).Bold(true).Render("Teams"),
			"",
			lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render("Cloud session unavailable."),
			lipgloss.NewStyle().Foreground(r.Theme.Subtext).Render("Return to Profile to sign in again."),
		}
	}

	if !p.HasTeams {
		return []string{
			lipgloss.NewStyle().Foreground(r.Theme.Text).Bold(true).Render("Teams"),
			"",
			lipgloss.NewStyle().Foreground(r.Theme.Text).Render("No teams yet."),
			lipgloss.NewStyle().Foreground(r.Theme.Subtext).Render("Create a team when you want a shared shell."),
		}
	}

	if len(p.Items) == 0 {
		return []string{
			lipgloss.NewStyle().Foreground(r.Theme.Text).Bold(true).Render("Hosts"),
			"",
			lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render("No hosts in this team yet."),
		}
	}

	var lines []string
	for _, item := range p.Items {
		if item.IsGroup {
			lines = append(lines, "")
			arrowR := lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render(r.Icons.Expanded)
			nameStyle := lipgloss.NewStyle().Foreground(r.Theme.Subtext)
			countStr := lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render(fmt.Sprintf(" %d", item.HostCount))
			groupPrefix := arrowR + " "
			groupLines := r.renderListEntry(groupPrefix, item.GroupName, nameStyle, layout.listW-lipgloss.Width(groupPrefix)-4)
			if len(groupLines) > 0 {
				groupLines[0] += countStr
			}
			lines = append(lines, groupLines...)
			continue
		}

		prefix := "    "
		nameStyle := lipgloss.NewStyle().Foreground(r.Theme.Subtext)
		if item.Selected {
			prefix = lipgloss.NewStyle().Foreground(r.Theme.Accent).Render("  " + r.Icons.Focused + " ")
			nameStyle = lipgloss.NewStyle().Foreground(r.Theme.Accent).Bold(true)
		}

		maxLblW := layout.listW - 10
		if layout.narrowMode {
			maxLblW = r.W - 16
		}
		label := strings.TrimSpace(item.Label)
		if label == "" {
			label = item.Hostname
		}
		entryPrefix := prefix + r.renderHostStatusDot(0, item.Health) + " "
		lines = append(lines, r.renderListEntry(entryPrefix, label, nameStyle, maxLblW)...)
	}

	return lines
}

func (r *Renderer) renderTeamsDetail(p TeamsHomeViewParams, w int, h int) string {
	if !p.SessionValid {
		return strings.Join([]string{
			lipgloss.NewStyle().Foreground(r.Theme.Text).Bold(true).Render("Teams"),
			lipgloss.NewStyle().Foreground(r.Theme.Subtext).Render("Sign in to browse shared hosts."),
			"",
			lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render("Use the Profile page to authenticate and return to Teams mode."),
		}, "\n")
	}

	if !p.HasTeams {
		return strings.Join([]string{
			lipgloss.NewStyle().Foreground(r.Theme.Text).Bold(true).Render("Create your first team"),
			lipgloss.NewStyle().Foreground(r.Theme.Subtext).Render("SSHThing stays local in personal mode until you want shared access."),
			"",
			lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render("Press Enter to create a team."),
		}, "\n")
	}

	selected, ok := r.selectedTeamsHost(p.Items)
	if !ok {
		return r.renderTeamsSummaryDetail(p, "Select a host to see connection details.")
	}

	kStyle := lipgloss.NewStyle().Foreground(r.Theme.Subtext)
	vStyle := lipgloss.NewStyle().Foreground(r.Theme.Text)
	dimStyle := lipgloss.NewStyle().Foreground(r.Theme.Subtext)

	label := strings.TrimSpace(selected.Label)
	if label == "" {
		label = selected.Hostname
	}
	connStr := selected.Hostname
	if selected.Username != "" {
		connStr = selected.Username + "@" + selected.Hostname
	}
	if selected.Port > 0 && selected.Port != 22 {
		connStr += fmt.Sprintf(":%d", selected.Port)
	}

	tagStr := lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render("no tags")
	if len(selected.Tags) > 0 {
		parts := make([]string, 0, len(selected.Tags))
		for _, tag := range selected.Tags {
			tag = strings.TrimSpace(tag)
			if tag == "" {
				continue
			}
			parts = append(parts, lipgloss.NewStyle().Foreground(r.Theme.Pink).Render(tag))
		}
		if len(parts) > 0 {
			tagStr = strings.Join(parts, "  ")
		}
	}

	lastSeen := selected.LastConnectedLabel
	if lastSeen == "" {
		lastSeen = "never"
	}

	lines := []string{
		lipgloss.NewStyle().Foreground(r.Theme.Text).Bold(true).Render(label),
	}
	short := h <= 9
	medium := h <= 15
	if short {
		lines = append(lines, r.renderHostStatusLabel(0, selected.Health)+" "+lipgloss.NewStyle().Foreground(r.Theme.Accent).Render(connStr))
		lines = append(lines, dimStyle.Render("auth "+orFallback(selected.CredentialType, "none")+" · mode "+orFallback(selected.CredentialMode, "shared")))
	} else {
		lines = append(lines, r.renderHostStatusLabel(0, selected.Health), lipgloss.NewStyle().Foreground(r.Theme.Accent).Render(connStr), "")
		if medium {
			lines = append(lines,
				dimStyle.Render("team "+orFallback(p.CurrentTeam.Name, "(none)")+" · role "+orFallback(p.CurrentTeam.Role, "(unknown)")),
				dimStyle.Render("auth "+orFallback(selected.CredentialType, "none")+" · mode "+orFallback(selected.CredentialMode, "shared")),
			)
		} else {
			lines = append(lines,
				kStyle.Render("team        ")+vStyle.Render(p.CurrentTeam.Name),
				kStyle.Render("slug        ")+dimStyle.Render(orFallback(p.CurrentTeam.Slug, "(none)")),
				kStyle.Render("role        ")+dimStyle.Render(orFallback(p.CurrentTeam.Role, "(unknown)")),
				kStyle.Render("auth        ")+vStyle.Render(orFallback(selected.CredentialType, "none")),
				kStyle.Render("mode        ")+dimStyle.Render(orFallback(selected.CredentialMode, "shared")),
				kStyle.Render("group       ")+dimStyle.Render(orFallback(selected.Group, "Ungrouped")),
				kStyle.Render("last seen   ")+dimStyle.Render(lastSeen),
			)
		}
	}

	if healthLines := r.renderHostHealthBlock(selected.Health, w, h, p.HealthDisplayMode); len(healthLines) > 0 {
		lines = append(lines, "")
		lines = append(lines, healthLines...)
	}

	if !short {
		lines = append(lines, "", kStyle.Render("tags        ")+tagStr)
	}

	if notes := strings.TrimSpace(selected.Notes); notes != "" && !short {
		if medium {
			lines = append(lines, kStyle.Render("notes       ")+dimStyle.Render(r.TruncStr(notes, max(8, w-12))))
		} else {
			lines = append(lines,
				kStyle.Render("notes       ")+dimStyle.Render(r.TruncStr(notes, max(8, w-12))),
				"",
			)
		}
	}

	if !short {
		lines = append(lines,
			lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render("enter connect  ·  a add host  ·  e edit  ·  d delete"),
		)
	}
	return strings.Join(lines, "\n")
}

func (r *Renderer) renderTeamsSummaryDetail(p TeamsHomeViewParams, hint string) string {
	lines := []string{
		lipgloss.NewStyle().Foreground(r.Theme.Text).Bold(true).Render(orFallback(p.CurrentTeam.Name, "Current Team")),
		lipgloss.NewStyle().Foreground(r.Theme.Subtext).Render(fmt.Sprintf("%d hosts  \u00B7  %d teams", p.CurrentTeam.HostCount, p.CurrentTeam.TeamCount)),
	}
	if slug := strings.TrimSpace(p.CurrentTeam.Slug); slug != "" {
		lines = append(lines, lipgloss.NewStyle().Foreground(r.Theme.Subtext).Render("slug: "+slug))
	}
	if role := strings.TrimSpace(p.CurrentTeam.Role); role != "" {
		lines = append(lines, lipgloss.NewStyle().Foreground(r.Theme.Subtext).Render("role: "+role))
	}
	lines = append(lines, "", lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render(hint))
	if p.CurrentTeam.HostCount == 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render("Press 'a' to add a team host here."))
	}
	return strings.Join(lines, "\n")
}

func (r *Renderer) selectedTeamsHost(items []TeamsHomeListItem) (TeamsHomeListItem, bool) {
	for _, item := range items {
		if item.IsGroup || !item.Selected {
			continue
		}
		return item, true
	}
	return TeamsHomeListItem{}, false
}

func orFallback(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
