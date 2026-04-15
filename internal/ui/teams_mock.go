package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type TeamsActionOption struct {
	Label       string
	Description string
}

type TeamsActionParams struct {
	Page        int
	Title       string
	Message     string
	Description string
	Cursor      int
	Options     []TeamsActionOption
	Err         error
}

type TeamsHostListItem struct {
	IsGroup      bool
	GroupName    string
	Collapsed    bool
	HostCount    int
	Label        string
	Hostname     string
	Username     string
	Port         int
	Tags         []string
	ShareMode    string
	LastActivity string
	Notes        []string
}

type TeamsHostsViewParams struct {
	Page         int
	TeamName     string
	HostCount    int
	MembersCount int
	Items        []TeamsHostListItem
	Cursor       int
	Selected     TeamsHostListItem
	Err          error
}

type TeamsMemberListItem struct {
	Name     string
	Email    string
	Role     string
	Status   string
	LastSeen string
	Selected bool
}

type TeamsMembersViewParams struct {
	Page      int
	TeamName  string
	Items     []TeamsMemberListItem
	Cursor    int
	Selected  TeamsMemberListItem
	HostCount int
	Err       error
}

type TeamsInviteMemberParams struct {
	Page        int
	TeamName    string
	Email       string
	RoleOptions []string
	RoleIdx     int
	Focus       int
	Err         error
}

type TeamsEditMemberParams struct {
	Page        int
	TeamName    string
	Member      TeamsMemberListItem
	RoleOptions []string
	RoleIdx     int
	Focus       int
	Err         error
}

type TeamsRemoveMemberParams struct {
	Page     int
	TeamName string
	Member   TeamsMemberListItem
	Cursor   int
	Err      error
}

func (r *Renderer) RenderTeamsActionView(p TeamsActionParams) string {
	cw := r.PageContentWidth()
	pad := r.LeftPad()

	title := lipgloss.NewStyle().Foreground(r.Theme.Text).Bold(true).Render(p.Title)
	message := lipgloss.NewStyle().Foreground(r.Theme.Text).Bold(true).Render(p.Message)
	desc := lipgloss.NewStyle().Foreground(r.Theme.Subtext).Render(p.Description)

	var optionLines []string
	for i, opt := range p.Options {
		prefix := "  "
		labelStyle := lipgloss.NewStyle().Foreground(r.Theme.Subtext)
		descStyle := lipgloss.NewStyle().Foreground(r.Theme.Overlay)
		if i == p.Cursor {
			prefix = lipgloss.NewStyle().Foreground(r.Theme.Accent).Render(r.Icons.Selected + " ")
			labelStyle = lipgloss.NewStyle().Foreground(r.Theme.Accent).Bold(true)
			descStyle = lipgloss.NewStyle().Foreground(r.Theme.Text)
		}
		optionLines = append(optionLines, prefix+labelStyle.Render(opt.Label))
		optionLines = append(optionLines, "  "+descStyle.Render(opt.Description))
		optionLines = append(optionLines, "")
	}

	content := []string{title, "", message, desc, ""}
	if p.Err != nil {
		content = append(content, r.renderErrLine(p.Err), "")
	}
	content = append(content, optionLines...)
	content = append(content, r.RenderFooter("↑↓ choose  enter continue  esc personal  q back"))

	body := lipgloss.NewStyle().Width(min(cw, 72)).Render(strings.Join(content, "\n"))
	if r.ShowSidebar() {
		sidebar := r.RenderSidebar(max(12, r.H-6), p.Page)
		sideGap := lipgloss.NewStyle().Width(2).Render(strings.Repeat("\n", max(12, r.H-6)))
		body = lipgloss.JoinHorizontal(lipgloss.Top, body, sideGap, sidebar)
	}
	return r.PadContent(body, pad)
}

func (r *Renderer) RenderTeamsHostsView(p TeamsHostsViewParams) string {
	cw := r.PageContentWidth()
	pad := r.LeftPad()

	listW := cw * 30 / 100
	if listW < 24 {
		listW = 24
	}
	gapW := 4
	detailW := cw - listW - gapW
	if detailW < 20 {
		detailW = 20
	}
	bodyH := r.H - 7
	if p.Err != nil {
		bodyH--
	}
	if bodyH < 4 {
		bodyH = 4
	}

	header := r.RenderHeader(fmt.Sprintf("team  ·  %s  ·  %d hosts  %d members", p.TeamName, p.HostCount, p.MembersCount), 0, 0)

	var listLines []string
	for i, item := range p.Items {
		sel := i == p.Cursor
		if item.IsGroup {
			arrow := r.Icons.Expanded
			if item.Collapsed {
				arrow = r.Icons.Collapsed
			}
			nameStyle := lipgloss.NewStyle().Foreground(r.Theme.Subtext)
			arrowStyle := lipgloss.NewStyle().Foreground(r.Theme.Overlay)
			if sel {
				nameStyle = lipgloss.NewStyle().Foreground(r.Theme.Accent)
				arrowStyle = lipgloss.NewStyle().Foreground(r.Theme.Accent)
			}
			listLines = append(listLines, "")
			listLines = append(listLines, arrowStyle.Render(arrow)+" "+nameStyle.Render(item.GroupName)+lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render(fmt.Sprintf(" %d", item.HostCount)))
			continue
		}

		prefix := "    "
		nameStyle := lipgloss.NewStyle().Foreground(r.Theme.Subtext)
		dot := lipgloss.NewStyle().Foreground(r.Theme.Green).Render(r.Icons.Connected)
		if sel {
			prefix = lipgloss.NewStyle().Foreground(r.Theme.Accent).Render("  " + r.Icons.Focused + " ")
			nameStyle = lipgloss.NewStyle().Foreground(r.Theme.Accent).Bold(true)
		}
		listLines = append(listLines, prefix+dot+" "+nameStyle.Render(r.TruncStr(item.Label, listW-10)))
	}

	for len(listLines) < bodyH {
		listLines = append(listLines, "")
	}
	if len(listLines) > bodyH {
		listLines = listLines[:bodyH]
	}

	listBlock := lipgloss.NewStyle().Width(listW).Render(strings.Join(listLines, "\n"))
	detailBlock := lipgloss.NewStyle().Width(detailW).Render(r.renderTeamsHostDetail(p.Selected))
	gapBlock := lipgloss.NewStyle().Width(gapW).Render(strings.Repeat("\n", bodyH))
	body := lipgloss.JoinHorizontal(lipgloss.Top, listBlock, gapBlock, detailBlock)

	if r.ShowSidebar() {
		sidebar := r.RenderSidebar(bodyH, p.Page)
		sideGap := lipgloss.NewStyle().Width(2).Render(strings.Repeat("\n", bodyH))
		body = lipgloss.JoinHorizontal(lipgloss.Top, body, sideGap, sidebar)
	}

	lines := []string{header, ""}
	if p.Err != nil {
		lines = append(lines, r.renderErrLine(p.Err))
	}
	lines = append(lines, body, "", r.RenderFooter("↑↓ nav  enter connect  / search  a add  e edit  d delete  m members  q back"))
	return r.PadContent(strings.Join(lines, "\n"), pad)
}

func (r *Renderer) renderTeamsHostDetail(item TeamsHostListItem) string {
	if item.IsGroup {
		name := lipgloss.NewStyle().Foreground(r.Theme.Text).Bold(true).Render(item.GroupName)
		sub := lipgloss.NewStyle().Foreground(r.Theme.Subtext).Render(fmt.Sprintf("%d shared hosts", item.HostCount))
		hint := lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render("enter toggle")
		return name + "\n" + sub + "\n\n" + hint
	}

	title := lipgloss.NewStyle().Foreground(r.Theme.Text).Bold(true).Render(item.Label)
	connStr := fmt.Sprintf("%s@%s", item.Username, item.Hostname)
	if item.Port != 22 && item.Port != 0 {
		connStr += fmt.Sprintf(":%d", item.Port)
	}
	tagStr := "no tags"
	if len(item.Tags) > 0 {
		tagStr = strings.Join(item.Tags, "  ")
	}

	lines := []string{
		title,
		lipgloss.NewStyle().Foreground(r.Theme.Accent).Render(connStr),
		"",
		lipgloss.NewStyle().Foreground(r.Theme.Subtext).Render("group       ") + detailGroupName(item.GroupName),
		lipgloss.NewStyle().Foreground(r.Theme.Subtext).Render("share mode  ") + lipgloss.NewStyle().Foreground(r.Theme.Text).Render(item.ShareMode),
		lipgloss.NewStyle().Foreground(r.Theme.Subtext).Render("last seen   ") + lipgloss.NewStyle().Foreground(r.Theme.Text).Render(item.LastActivity),
		"",
		lipgloss.NewStyle().Foreground(r.Theme.Subtext).Render("tags        ") + lipgloss.NewStyle().Foreground(r.Theme.Pink).Render(tagStr),
		"",
		lipgloss.NewStyle().Foreground(r.Theme.Subtext).Render("notes"),
	}
	for _, note := range item.Notes {
		lines = append(lines, lipgloss.NewStyle().Foreground(r.Theme.Subtext).Render("• "+note))
	}
	lines = append(lines, "", lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render("shared host  ·  enter connect  ·  e edit  ·  d delete"))
	return strings.Join(lines, "\n")
}

func (r *Renderer) RenderTeamsMembersView(p TeamsMembersViewParams) string {
	cw := r.PageContentWidth()
	pad := r.LeftPad()
	leftW := cw * 38 / 100
	if leftW < 26 {
		leftW = 26
	}
	gapW := 4
	rightW := cw - leftW - gapW
	if rightW < 20 {
		rightW = 20
	}
	bodyH := r.H - 7
	if p.Err != nil {
		bodyH--
	}
	if bodyH < 4 {
		bodyH = 4
	}

	header := r.RenderHeader(fmt.Sprintf("team members  ·  %s  ·  %d shared hosts", p.TeamName, p.HostCount), 0, 0)

	var listLines []string
	for _, item := range p.Items {
		prefix := "  "
		nameStyle := lipgloss.NewStyle().Foreground(r.Theme.Subtext)
		metaStyle := lipgloss.NewStyle().Foreground(r.Theme.Overlay)
		if item.Selected {
			prefix = lipgloss.NewStyle().Foreground(r.Theme.Accent).Render(r.Icons.Selected + " ")
			nameStyle = lipgloss.NewStyle().Foreground(r.Theme.Accent).Bold(true)
			metaStyle = lipgloss.NewStyle().Foreground(r.Theme.Text)
		}
		listLines = append(listLines, prefix+nameStyle.Render(item.Name))
		listLines = append(listLines, "  "+metaStyle.Render(item.Email))
		listLines = append(listLines, "  "+lipgloss.NewStyle().Foreground(r.Theme.Sky).Render(item.Role))
		listLines = append(listLines, "")
	}
	for len(listLines) < bodyH {
		listLines = append(listLines, "")
	}
	if len(listLines) > bodyH {
		listLines = listLines[:bodyH]
	}

	listBlock := lipgloss.NewStyle().Width(leftW).Render(strings.Join(listLines, "\n"))
	detailLines := []string{
		lipgloss.NewStyle().Foreground(r.Theme.Text).Bold(true).Render(p.Selected.Name),
		lipgloss.NewStyle().Foreground(r.Theme.Surface0).Render(strings.Repeat(r.Icons.Rule, min(rightW-2, 28))),
		"",
		lipgloss.NewStyle().Foreground(r.Theme.Subtext).Render(p.Selected.Email),
		lipgloss.NewStyle().Foreground(r.Theme.Subtext).Render("role        ") + lipgloss.NewStyle().Foreground(r.Theme.Text).Render(p.Selected.Role),
		lipgloss.NewStyle().Foreground(r.Theme.Subtext).Render("status      ") + lipgloss.NewStyle().Foreground(r.Theme.Text).Render(p.Selected.Status),
		lipgloss.NewStyle().Foreground(r.Theme.Subtext).Render("last seen   ") + lipgloss.NewStyle().Foreground(r.Theme.Text).Render(p.Selected.LastSeen),
		"",
		lipgloss.NewStyle().Foreground(r.Theme.Subtext).Render("member management"),
		lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render("invite, edit, and remove stay lightweight in the first pass"),
		"",
		lipgloss.NewStyle().Foreground(r.Theme.Overlay).Render("a invite  ·  e edit  ·  d remove"),
	}
	detailBlock := lipgloss.NewStyle().Width(rightW).Render(strings.Join(detailLines, "\n"))
	gapBlock := lipgloss.NewStyle().Width(gapW).Render(strings.Repeat("\n", bodyH))
	body := lipgloss.JoinHorizontal(lipgloss.Top, listBlock, gapBlock, detailBlock)

	if r.ShowSidebar() {
		sidebar := r.RenderSidebar(bodyH, p.Page)
		sideGap := lipgloss.NewStyle().Width(2).Render(strings.Repeat("\n", bodyH))
		body = lipgloss.JoinHorizontal(lipgloss.Top, body, sideGap, sidebar)
	}

	lines := []string{header, ""}
	if p.Err != nil {
		lines = append(lines, r.renderErrLine(p.Err))
	}
	lines = append(lines, body, "", r.RenderFooter("↑↓ nav  a invite  e edit  d remove  h hosts  q back"))
	return r.PadContent(strings.Join(lines, "\n"), pad)
}

func (r *Renderer) RenderTeamsInviteMemberView(p TeamsInviteMemberParams) string {
	role := p.RoleOptions[p.RoleIdx]
	lines := []string{
		lipgloss.NewStyle().Foreground(r.Theme.Text).Bold(true).Render("Invite member"),
		lipgloss.NewStyle().Foreground(r.Theme.Subtext).Render(p.TeamName + "  ·  send a mock team invite"),
		"",
		r.renderTeamsManageLine("email", p.Email, p.Focus == 0),
		r.renderTeamsManageLine("role", role, p.Focus == 1),
		"",
		r.renderTeamsManageActions("Send invite", "Cancel", p.Focus, 2, 3),
	}
	if p.Err != nil {
		lines = append(lines[:3], append([]string{r.renderErrLine(p.Err), ""}, lines[3:]...)...)
	}
	lines = append(lines, "", r.RenderFooter("↑↓ focus  ←→ role/button  type email  enter continue  q back"))
	return r.renderTeamsManageView(p.Page, strings.Join(lines, "\n"))
}

func (r *Renderer) RenderTeamsEditMemberView(p TeamsEditMemberParams) string {
	role := p.RoleOptions[p.RoleIdx]
	lines := []string{
		lipgloss.NewStyle().Foreground(r.Theme.Text).Bold(true).Render("Edit member"),
		lipgloss.NewStyle().Foreground(r.Theme.Subtext).Render(p.TeamName + "  ·  lightweight role management"),
		"",
		lipgloss.NewStyle().Foreground(r.Theme.Text).Bold(true).Render(p.Member.Name),
		lipgloss.NewStyle().Foreground(r.Theme.Subtext).Render(p.Member.Email),
		"",
		r.renderTeamsManageLine("role", role, p.Focus == 0),
		"",
		r.renderTeamsManageActions("Save", "Cancel", p.Focus, 1, 2),
	}
	if p.Err != nil {
		lines = append(lines[:3], append([]string{r.renderErrLine(p.Err), ""}, lines[3:]...)...)
	}
	lines = append(lines, "", r.RenderFooter("↑↓ focus  ←→ role/button  enter continue  q back"))
	return r.renderTeamsManageView(p.Page, strings.Join(lines, "\n"))
}

func (r *Renderer) RenderTeamsRemoveMemberView(p TeamsRemoveMemberParams) string {
	lines := []string{
		lipgloss.NewStyle().Foreground(r.Theme.Text).Bold(true).Render("Remove member"),
		lipgloss.NewStyle().Foreground(r.Theme.Subtext).Render(p.TeamName + "  ·  confirm mock member removal"),
		"",
		lipgloss.NewStyle().Foreground(r.Theme.Text).Bold(true).Render(p.Member.Name),
		lipgloss.NewStyle().Foreground(r.Theme.Subtext).Render(p.Member.Email),
		lipgloss.NewStyle().Foreground(r.Theme.Subtext).Render("role        ") + lipgloss.NewStyle().Foreground(r.Theme.Text).Render(p.Member.Role),
		"",
		r.renderTeamsManageActions("Remove member", "Cancel", p.Cursor, 0, 1),
	}
	if p.Err != nil {
		lines = append(lines[:3], append([]string{r.renderErrLine(p.Err), ""}, lines[3:]...)...)
	}
	lines = append(lines, "", r.RenderFooter("←→ choose  enter continue  q back"))
	return r.renderTeamsManageView(p.Page, strings.Join(lines, "\n"))
}

func (r *Renderer) renderTeamsManageView(page int, body string) string {
	cw := r.PageContentWidth()
	pad := r.LeftPad()
	content := lipgloss.NewStyle().Width(min(cw, 72)).Render(body)
	if r.ShowSidebar() {
		sidebar := r.RenderSidebar(max(12, r.H-6), page)
		sideGap := lipgloss.NewStyle().Width(2).Render(strings.Repeat("\n", max(12, r.H-6)))
		content = lipgloss.JoinHorizontal(lipgloss.Top, content, sideGap, sidebar)
	}
	return r.PadContent(content, pad)
}

func (r *Renderer) renderTeamsManageLine(label, value string, selected bool) string {
	prefix := "  "
	labelStyle := lipgloss.NewStyle().Foreground(r.Theme.Subtext)
	valueStyle := lipgloss.NewStyle().Foreground(r.Theme.Text)
	if selected {
		prefix = lipgloss.NewStyle().Foreground(r.Theme.Accent).Render(r.Icons.Selected + " ")
		labelStyle = lipgloss.NewStyle().Foreground(r.Theme.Accent)
		valueStyle = lipgloss.NewStyle().Foreground(r.Theme.Accent).Bold(true)
	}
	return prefix + labelStyle.Render(label+"  ") + valueStyle.Render(value)
}

func (r *Renderer) renderTeamsManageActions(primary, secondary string, focus, primaryFocus, secondaryFocus int) string {
	left := lipgloss.NewStyle().Foreground(r.Theme.Subtext)
	right := lipgloss.NewStyle().Foreground(r.Theme.Subtext)
	if focus == primaryFocus {
		left = lipgloss.NewStyle().Foreground(r.Theme.Accent).Bold(true)
	}
	if focus == secondaryFocus {
		right = lipgloss.NewStyle().Foreground(r.Theme.Accent).Bold(true)
	}
	return left.Render("[ "+primary+" ]") + "   " + right.Render("[ "+secondary+" ]")
}

func detailGroupName(group string) string {
	group = strings.TrimSpace(group)
	if group == "" {
		return "Ungrouped"
	}
	return group
}
