package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/Vansh-Raja/SSHThing/internal/teams"
	"github.com/charmbracelet/lipgloss"
)

type TeamsViewParams struct {
	Page         int
	ModeLabel    string
	State        int
	Err          error
	SessionValid bool
	CurrentTeam  teams.TeamSummary
	Teams        []teams.TeamSummary
	HostCursor   int
	Hosts        []teams.TeamHost
}

func (r *Renderer) RenderTeamsView(p TeamsViewParams) string {
	cw := r.PageContentWidth()
	pad := r.LeftPad()

	var lines []string
	subtitle := "teams"
	if strings.TrimSpace(p.ModeLabel) != "" {
		subtitle = p.ModeLabel
	}
	lines = append(lines, r.RenderHeader(subtitle, len(p.Hosts), 0))
	lines = append(lines, "")

	switch {
	case !p.SessionValid:
		lines = append(lines, lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render("Cloud session unavailable. Return to Profile to sign in again."))
	case p.State == 0:
		lines = append(lines,
			lipgloss.NewStyle().Foreground(r.Theme.Text).Bold(true).Render("Teams"),
			"",
			lipgloss.NewStyle().Foreground(r.Theme.Text).Render("No teams yet."),
			lipgloss.NewStyle().Foreground(r.Theme.Subtext).Render("SSHThing stays fully local in personal mode. Create a team when you want a shared shell."),
			"",
			lipgloss.NewStyle().Foreground(r.Theme.Accent).Render("Press Enter to create a team."),
		)
	default:
		lines = append(lines, r.renderCurrentTeamHeader(p)...)
		lines = append(lines, "")
		lines = append(lines, r.renderTeamHosts(p)...)
	}

	if p.Err != nil {
		lines = append(lines, "")
		lines = append(lines, r.renderErrLine(p.Err))
	}

	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Foreground(r.Theme.Surface0).Render(strings.Repeat(r.Icons.Rule, min(cw, 40))))
	footer := "enter create team  / search  ? commands  , settings  shift+tab cycle  T personal mode"
	if p.SessionValid && p.State != 0 {
		footer = "↑↓ hosts  ctrl+1..9 switch team  / search  ? commands  r refresh  , settings  shift+tab cycle  T personal mode"
	}
	lines = append(lines, r.RenderFooter(footer))

	inner := strings.Join(lines, "\n")
	if r.ShowSidebar() {
		bodyH := max(8, len(lines))
		sidebar := r.RenderSidebar(bodyH, p.Page)
		sideGap := lipgloss.NewStyle().Width(2).Render(strings.Repeat("\n", bodyH))
		inner = lipgloss.JoinHorizontal(lipgloss.Top, lipgloss.NewStyle().Width(cw).Render(inner), sideGap, sidebar)
	}
	return r.PadContent(inner, pad)
}

func (r *Renderer) renderCurrentTeamHeader(p TeamsViewParams) []string {
	teamName := strings.TrimSpace(p.CurrentTeam.Name)
	if teamName == "" {
		teamName = "Current Team"
	}
	lines := []string{
		lipgloss.NewStyle().Foreground(r.Theme.Text).Bold(true).Render(teamName),
	}
	if slug := strings.TrimSpace(p.CurrentTeam.Slug); slug != "" {
		lines = append(lines, lipgloss.NewStyle().Foreground(r.Theme.Subtext).Render("slug: "+slug))
	}
	if len(p.Teams) > 0 {
		teamNames := make([]string, 0, len(p.Teams))
		for _, team := range p.Teams {
			teamNames = append(teamNames, team.Name)
		}
		lines = append(lines, lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render("teams: "+strings.Join(teamNames, "  •  ")))
	}
	return lines
}

func (r *Renderer) renderTeamHosts(p TeamsViewParams) []string {
	if len(p.Hosts) == 0 {
		return []string{
			lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render("No hosts in this team yet."),
			lipgloss.NewStyle().Foreground(r.Theme.Subtext).Render("Host management for teams is the next piece of the shell."),
		}
	}

	lines := []string{lipgloss.NewStyle().Foreground(r.Theme.Text).Bold(true).Render("Hosts"), ""}
	for i, host := range p.Hosts {
		prefix := "    "
		nameStyle := lipgloss.NewStyle().Foreground(r.Theme.Subtext)
		if i == p.HostCursor {
			prefix = lipgloss.NewStyle().Foreground(r.Theme.Accent).Render("  " + r.Icons.Focused + " ")
			nameStyle = lipgloss.NewStyle().Foreground(r.Theme.Accent).Bold(true)
		}
		label := strings.TrimSpace(host.Label)
		if label == "" {
			label = host.Hostname
		}
		lines = append(lines, prefix+nameStyle.Render(label))
		if i == p.HostCursor {
			detail := []string{}
			if host.Username != "" {
				detail = append(detail, host.Username+"@"+host.Hostname)
			} else {
				detail = append(detail, host.Hostname)
			}
			if host.Port > 0 {
				detail = append(detail, fmt.Sprintf("port:%d", host.Port))
			}
			if host.Group != "" {
				detail = append(detail, "group:"+host.Group)
			}
			if host.LastConnectedAt != nil {
				detail = append(detail, "last:"+FormatTimeAgo(time.UnixMilli(*host.LastConnectedAt)))
			}
			if len(detail) > 0 {
				lines = append(lines, "      "+lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render(strings.Join(detail, "  ")))
			}
		}
	}
	return lines
}
