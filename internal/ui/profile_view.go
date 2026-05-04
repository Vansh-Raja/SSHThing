package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type ProfileViewParams struct {
	SignedIn         bool
	SigningIn        bool
	DisplayName      string
	Email            string
	ShowOpenTeamsCTA bool
	AppModeLabel     string
	Err              error
	Page             int
	CommandLine      *CommandLineView
}

func (r *Renderer) RenderProfileView(p ProfileViewParams) string {
	cw := r.PageContentWidth()
	pad := r.LeftPad()
	footerH := r.FooterBlockHeight(p.CommandLine)
	bodyH := r.H - 7 - footerH
	if bodyH < 6 {
		bodyH = 6
	}

	var lines []string
	lines = append(lines, r.RenderHeader("profile", 0, 0))
	lines = append(lines, "")

	title := lipgloss.NewStyle().Foreground(r.Theme.Text).Bold(true).Render("Cloud Profile")
	lines = append(lines, title)
	if strings.TrimSpace(p.AppModeLabel) != "" {
		lines = append(lines, lipgloss.NewStyle().Foreground(r.Theme.Subtext).Render(p.AppModeLabel))
	}
	lines = append(lines, "")

	switch {
	case p.SigningIn:
		lines = append(lines,
			lipgloss.NewStyle().Foreground(r.Theme.Text).Render("Signing in..."),
			lipgloss.NewStyle().Foreground(r.Theme.Subtext).Render("SSHThing opened your browser and is waiting for the cloud sign-in to complete."),
			lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render("Keep the browser open until the terminal updates."),
		)
	case p.SignedIn:
		displayName := strings.TrimSpace(p.DisplayName)
		if displayName == "" {
			displayName = "Signed-in user"
		}
		lines = append(lines,
			"Name: "+lipgloss.NewStyle().Foreground(r.Theme.Text).Render(displayName),
			"Email: "+lipgloss.NewStyle().Foreground(r.Theme.Text).Render(strings.TrimSpace(p.Email)),
			"",
			lipgloss.NewStyle().Foreground(r.Theme.Green).Render("Signed in"),
		)
		if p.ShowOpenTeamsCTA {
			lines = append(lines, "", lipgloss.NewStyle().Foreground(r.Theme.Accent).Render("Press Shift+T to enter Teams mode. Press S to sign out."))
		}
	default:
		lines = append(lines,
			lipgloss.NewStyle().Foreground(r.Theme.Text).Render("SSHThing works fully offline without sign-in."),
			lipgloss.NewStyle().Foreground(r.Theme.Subtext).Render("Sign in only if you want to use cloud and team features."),
			"",
			lipgloss.NewStyle().Foreground(r.Theme.Accent).Render("Press Enter to sign in."),
		)
	}

	if p.Err != nil {
		lines = append(lines, "")
		lines = append(lines, r.renderErrLine(p.Err))
	}

	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Foreground(r.Theme.Surface0).Render(strings.Repeat(r.Icons.Rule, min(cw, 40))))
	lines = append(lines, r.RenderFooterBlock(r.MainFooterText(), p.CommandLine))

	inner := strings.Join(lines, "\n")
	if r.ShowSidebar() {
		sidebar := r.RenderSidebar(max(bodyH, len(lines)), p.Page)
		sideGap := lipgloss.NewStyle().Width(2).Render(strings.Repeat("\n", max(bodyH, len(lines))))
		inner = lipgloss.JoinHorizontal(lipgloss.Top, lipgloss.NewStyle().Width(cw).Render(inner), sideGap, sidebar)
	}
	return r.PadContent(inner, pad)
}
