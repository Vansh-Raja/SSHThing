package app

import (
	"sort"
	"strings"

	"github.com/Vansh-Raja/SSHThing/internal/ui"
)

const (
	teamsStateLogin = iota
	teamsStateEmpty
	teamsStateHosts
	teamsStateMembers
	teamsStateInviteMember
	teamsStateEditMember
	teamsStateRemoveMember
)

type teamsMockTeam struct {
	Name        string
	Description string
	Hosts       []teamsMockHost
	Members     []teamsMockMember
}

type teamsMockHost struct {
	ID           string
	Label        string
	Group        string
	Tags         []string
	Hostname     string
	Username     string
	Port         int
	ShareMode    string
	LastActivity string
	Notes        []string
}

type teamsMockMember struct {
	ID       string
	Name     string
	Email    string
	Role     string
	Status   string
	LastSeen string
}

var teamsMemberRoles = []string{
	"Owner",
	"Admin",
	"Member",
	"Restricted Member",
}

func teamsMockTeamData() teamsMockTeam {
	return teamsMockTeam{
		Name:        "Acme Team",
		Description: "Shared hosts for a small developer team.",
		Hosts: []teamsMockHost{
			{
				ID:           "prod-api-1",
				Label:        "prod-api-1",
				Group:        "Production",
				Tags:         []string{"api", "prod"},
				Hostname:     "10.42.3.18",
				Username:     "ubuntu",
				Port:         22,
				ShareMode:    "Host + shared credential",
				LastActivity: "5 minutes ago",
				Notes:        []string{"Primary API node for deploy checks and restarts."},
			},
			{
				ID:           "prod-bastion",
				Label:        "prod-bastion",
				Group:        "Production",
				Tags:         []string{"bastion", "entry"},
				Hostname:     "10.42.3.7",
				Username:     "ops",
				Port:         22,
				ShareMode:    "Host only",
				LastActivity: "18 minutes ago",
				Notes:        []string{"Shared host metadata only. Members use their own credentials."},
			},
			{
				ID:           "staging-web-1",
				Label:        "staging-web-1",
				Group:        "Staging",
				Tags:         []string{"web", "staging"},
				Hostname:     "10.52.4.21",
				Username:     "ubuntu",
				Port:         22,
				ShareMode:    "Host + shared credential",
				LastActivity: "40 minutes ago",
				Notes:        []string{"Lower-risk box for testing and onboarding."},
			},
			{
				ID:           "client-jump",
				Label:        "client-jump",
				Group:        "",
				Tags:         []string{"client"},
				Hostname:     "172.16.18.4",
				Username:     "support",
				Port:         22,
				ShareMode:    "Host only",
				LastActivity: "4 hours ago",
				Notes:        []string{"Example of an ungrouped shared host."},
			},
		},
		Members: []teamsMockMember{
			{ID: "maya", Name: "Maya Singh", Email: "maya@acme.dev", Role: "Owner", Status: "active", LastSeen: "online now"},
			{ID: "omar", Name: "Omar Chen", Email: "omar@acme.dev", Role: "Admin", Status: "active", LastSeen: "14 minutes ago"},
			{ID: "lee", Name: "Lee Jordan", Email: "lee@acme.dev", Role: "Member", Status: "active", LastSeen: "2 hours ago"},
			{ID: "nina", Name: "Nina Patel", Email: "nina@acme.dev", Role: "Restricted Member", Status: "active", LastSeen: "1 hour ago"},
		},
	}
}

func (m *Model) resolveTeamsState() {
	m.ensureTeamsMockData()
	if !m.teamsAuthed {
		m.teamsState = teamsStateLogin
	} else if !m.teamsHasTeam {
		m.teamsState = teamsStateEmpty
	} else if m.teamsState != teamsStateMembers &&
		m.teamsState != teamsStateInviteMember &&
		m.teamsState != teamsStateEditMember &&
		m.teamsState != teamsStateRemoveMember {
		m.teamsState = teamsStateHosts
	}
}

func (m *Model) openTeamsPage() {
	m.resolveTeamsState()
	m.teamsActionIdx = 0
	m.err = nil
}

func (m *Model) clampTeamsMockState() {
	m.ensureTeamsMockData()

	if m.teamsActionIdx < 0 {
		m.teamsActionIdx = 0
	}
	if m.teamsActionIdx > 2 {
		m.teamsActionIdx = 2
	}

	items := m.buildTeamsHostItems()
	if len(items) == 0 {
		m.teamsResourceIdx = 0
	} else if m.teamsResourceIdx < 0 || m.teamsResourceIdx >= len(items) {
		m.teamsResourceIdx = 0
	}

	if len(m.teamsTeam.Members) == 0 {
		m.teamsMemberIdx = 0
	} else if m.teamsMemberIdx < 0 || m.teamsMemberIdx >= len(m.teamsTeam.Members) {
		m.teamsMemberIdx = 0
	}

	if m.teamsManageFocus < 0 {
		m.teamsManageFocus = 0
	}
	if m.teamsInviteRole < 0 || m.teamsInviteRole >= len(teamsMemberRoles) {
		m.teamsInviteRole = 2
	}
	if m.teamsEditRole < 0 || m.teamsEditRole >= len(teamsMemberRoles) {
		m.teamsEditRole = 2
	}

	if m.collapsed == nil {
		m.collapsed = map[string]bool{}
	}
}

func (m *Model) ensureTeamsMockData() {
	if m.teamsTeam.Name == "" {
		m.teamsTeam = teamsMockTeamData()
	}
}

func (m Model) teamsActionOptions() []ui.TeamsActionOption {
	if !m.teamsAuthed {
		return []ui.TeamsActionOption{
			{Label: "Login", Description: "Sign in to access your shared team hosts"},
			{Label: "Sign up", Description: "Create a Teams account"},
			{Label: "Back to Personal", Description: "Return to your private host list"},
		}
	}
	return []ui.TeamsActionOption{
		{Label: "Create team", Description: "Start a new shared team space"},
		{Label: "Join team", Description: "Join an existing team"},
		{Label: "Back to Personal", Description: "Return to your private host list"},
	}
}

func (m Model) buildTeamsActionViewParams() ui.TeamsActionParams {
	title := "Teams"
	message := "Login / Sign up"
	description := "Shared hosts for your team"
	if m.teamsAuthed && !m.teamsHasTeam {
		message = "You're not part of a team yet"
		description = "Create a team or join one to access shared hosts."
	}
	return ui.TeamsActionParams{
		Page:        m.page,
		Title:       title,
		Message:     message,
		Description: description,
		Cursor:      m.teamsActionIdx,
		Options:     m.teamsActionOptions(),
		Err:         m.err,
	}
}

func (m Model) buildTeamsHostItems() []ui.TeamsHostListItem {
	counts := make(map[string]int)
	hostsByGroup := make(map[string][]teamsMockHost)
	for _, h := range m.teamsTeam.Hosts {
		group := strings.TrimSpace(h.Group)
		hostsByGroup[group] = append(hostsByGroup[group], h)
		counts[group]++
	}

	groups := make([]string, 0, len(hostsByGroup))
	for g := range hostsByGroup {
		if g == "" {
			continue
		}
		groups = append(groups, g)
	}
	sort.Slice(groups, func(i, j int) bool {
		return strings.ToLower(groups[i]) < strings.ToLower(groups[j])
	})
	for group := range hostsByGroup {
		sort.Slice(hostsByGroup[group], func(i, j int) bool {
			return strings.ToLower(hostsByGroup[group][i].Label) < strings.ToLower(hostsByGroup[group][j].Label)
		})
	}

	var items []ui.TeamsHostListItem
	for _, group := range groups {
		items = append(items, ui.TeamsHostListItem{
			IsGroup:   true,
			GroupName: group,
			HostCount: counts[group],
			Collapsed: m.collapsed[group],
		})
		if !m.collapsed[group] {
			for _, host := range hostsByGroup[group] {
				items = append(items, toTeamsHostListItem(host))
			}
		}
	}

	if ungrouped := hostsByGroup[""]; len(ungrouped) > 0 {
		items = append(items, ui.TeamsHostListItem{
			IsGroup:   true,
			GroupName: "Ungrouped",
			HostCount: counts[""],
			Collapsed: m.collapsed["Ungrouped"],
		})
		if !m.collapsed["Ungrouped"] {
			for _, host := range ungrouped {
				items = append(items, toTeamsHostListItem(host))
			}
		}
	}
	return items
}

func toTeamsHostListItem(host teamsMockHost) ui.TeamsHostListItem {
	return ui.TeamsHostListItem{
		Label:        host.Label,
		GroupName:    host.Group,
		Hostname:     host.Hostname,
		Username:     host.Username,
		Port:         host.Port,
		Tags:         append([]string(nil), host.Tags...),
		ShareMode:    host.ShareMode,
		LastActivity: host.LastActivity,
		Notes:        append([]string(nil), host.Notes...),
	}
}

func (m Model) selectedTeamsHostItem() (ui.TeamsHostListItem, bool) {
	items := m.buildTeamsHostItems()
	if m.teamsResourceIdx < 0 || m.teamsResourceIdx >= len(items) {
		return ui.TeamsHostListItem{}, false
	}
	return items[m.teamsResourceIdx], true
}

func (m Model) selectedTeamsMember() (teamsMockMember, bool) {
	if m.teamsMemberIdx < 0 || m.teamsMemberIdx >= len(m.teamsTeam.Members) {
		return teamsMockMember{}, false
	}
	return m.teamsTeam.Members[m.teamsMemberIdx], true
}

func (m *Model) moveTeamsHostSelection(delta int) {
	items := m.buildTeamsHostItems()
	if len(items) == 0 {
		m.teamsResourceIdx = 0
		return
	}
	m.teamsResourceIdx += delta
	if m.teamsResourceIdx < 0 {
		m.teamsResourceIdx = 0
	}
	if m.teamsResourceIdx >= len(items) {
		m.teamsResourceIdx = len(items) - 1
	}
}

func (m *Model) moveTeamsMemberSelection(delta int) {
	if len(m.teamsTeam.Members) == 0 {
		m.teamsMemberIdx = 0
		return
	}
	m.teamsMemberIdx += delta
	if m.teamsMemberIdx < 0 {
		m.teamsMemberIdx = 0
	}
	if m.teamsMemberIdx >= len(m.teamsTeam.Members) {
		m.teamsMemberIdx = len(m.teamsTeam.Members) - 1
	}
}

func (m Model) buildTeamsHostsViewParams() ui.TeamsHostsViewParams {
	selected, _ := m.selectedTeamsHostItem()
	return ui.TeamsHostsViewParams{
		Page:         m.page,
		TeamName:     m.teamsTeam.Name,
		HostCount:    len(m.teamsTeam.Hosts),
		MembersCount: len(m.teamsTeam.Members),
		Items:        m.buildTeamsHostItems(),
		Cursor:       m.teamsResourceIdx,
		Selected:     selected,
		Err:          m.err,
	}
}

func (m Model) buildTeamsMembersViewParams() ui.TeamsMembersViewParams {
	items := make([]ui.TeamsMemberListItem, 0, len(m.teamsTeam.Members))
	for i, member := range m.teamsTeam.Members {
		items = append(items, ui.TeamsMemberListItem{
			Name:     member.Name,
			Email:    member.Email,
			Role:     member.Role,
			Status:   member.Status,
			LastSeen: member.LastSeen,
			Selected: i == m.teamsMemberIdx,
		})
	}
	selected, _ := m.selectedTeamsMember()
	return ui.TeamsMembersViewParams{
		Page:      m.page,
		TeamName:  m.teamsTeam.Name,
		Items:     items,
		Cursor:    m.teamsMemberIdx,
		Selected:  ui.TeamsMemberListItem{Name: selected.Name, Email: selected.Email, Role: selected.Role, Status: selected.Status, LastSeen: selected.LastSeen},
		HostCount: len(m.teamsTeam.Hosts),
		Err:       m.err,
	}
}

func (m Model) buildTeamsInviteMemberParams() ui.TeamsInviteMemberParams {
	return ui.TeamsInviteMemberParams{
		Page:        m.page,
		TeamName:    m.teamsTeam.Name,
		Email:       m.teamsInviteEmail,
		RoleOptions: append([]string(nil), teamsMemberRoles...),
		RoleIdx:     m.teamsInviteRole,
		Focus:       m.teamsManageFocus,
		Err:         m.err,
	}
}

func (m Model) buildTeamsEditMemberParams() ui.TeamsEditMemberParams {
	selected, _ := m.selectedTeamsMember()
	return ui.TeamsEditMemberParams{
		Page:        m.page,
		TeamName:    m.teamsTeam.Name,
		Member:      ui.TeamsMemberListItem{Name: selected.Name, Email: selected.Email, Role: selected.Role, Status: selected.Status, LastSeen: selected.LastSeen},
		RoleOptions: append([]string(nil), teamsMemberRoles...),
		RoleIdx:     m.teamsEditRole,
		Focus:       m.teamsManageFocus,
		Err:         m.err,
	}
}

func (m Model) buildTeamsRemoveMemberParams() ui.TeamsRemoveMemberParams {
	selected, _ := m.selectedTeamsMember()
	return ui.TeamsRemoveMemberParams{
		Page:     m.page,
		TeamName: m.teamsTeam.Name,
		Member:   ui.TeamsMemberListItem{Name: selected.Name, Email: selected.Email, Role: selected.Role, Status: selected.Status, LastSeen: selected.LastSeen},
		Cursor:   m.teamsManageFocus,
		Err:      m.err,
	}
}

func (m *Model) openTeamsInviteMember() {
	m.teamsState = teamsStateInviteMember
	m.teamsManageFocus = 0
	m.teamsInviteEmail = ""
	m.teamsInviteRole = 2
	m.err = nil
}

func (m *Model) openTeamsEditMember() {
	member, ok := m.selectedTeamsMember()
	if !ok {
		return
	}
	m.teamsState = teamsStateEditMember
	m.teamsManageFocus = 0
	m.teamsEditRole = teamsRoleIndex(member.Role)
	m.err = nil
}

func (m *Model) openTeamsRemoveMember() {
	if _, ok := m.selectedTeamsMember(); !ok {
		return
	}
	m.teamsState = teamsStateRemoveMember
	m.teamsManageFocus = 1
	m.err = nil
}

func (m *Model) leaveTeamsMemberManagement() {
	m.teamsState = teamsStateMembers
	m.teamsManageFocus = 0
	m.err = nil
}

func teamsRoleIndex(role string) int {
	for i, candidate := range teamsMemberRoles {
		if strings.EqualFold(candidate, role) {
			return i
		}
	}
	return 2
}

func teamsMockDisplayName(raw string) string {
	parts := strings.Fields(strings.ReplaceAll(raw, ".", " "))
	for i := range parts {
		runes := []rune(parts[i])
		if len(runes) == 0 {
			continue
		}
		parts[i] = strings.ToUpper(string(runes[0])) + strings.ToLower(string(runes[1:]))
	}
	if len(parts) == 0 {
		return "New Member"
	}
	return strings.Join(parts, " ")
}
