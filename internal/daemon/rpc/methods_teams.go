package rpc

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/Vansh-Raja/SSHThing/internal/daemon/service"
	"github.com/Vansh-Raja/SSHThing/internal/teams"
)

// RegisterTeams registers teams.* RPC handlers on s.
func RegisterTeams(s *Server, ts *service.TeamsService) {
	// Team listing and CRUD.
	s.Register("teams.list", makeTeamsList(ts))
	s.Register("teams.create", makeTeamsCreate(ts))
	s.Register("teams.rename", makeTeamsRename(ts))
	s.Register("teams.delete", makeTeamsDelete(ts))
	s.Register("teams.reorder", makeTeamsReorder(ts))
	s.Register("teams.leave", makeTeamsLeave(ts))

	// Host management.
	s.Register("teams.hosts.list", makeTeamsHostsList(ts))
	s.Register("teams.hosts.create", makeTeamsHostsCreate(ts))
	s.Register("teams.hosts.update", makeTeamsHostsUpdate(ts))
	s.Register("teams.hosts.delete", makeTeamsHostsDelete(ts))

	// Member management.
	s.Register("teams.members.list", makeTeamsMembersList(ts))
	s.Register("teams.members.invite", makeTeamsMembersInvite(ts))
	s.Register("teams.members.updateRole", makeTeamsMembersUpdateRole(ts))
	s.Register("teams.members.remove", makeTeamsMembersRemove(ts))

	// Invites.
	s.Register("teams.invites.list", makeTeamsInvitesList(ts))
	s.Register("teams.invites.accept", makeTeamsInvitesAccept(ts))
	s.Register("teams.invites.revoke", makeTeamsInvitesRevoke(ts))

	// Audit.
	s.Register("teams.audit.list", makeTeamsAuditList(ts))

	// Credentials.
	s.Register("teams.hosts.credentials.revealShared", makeTeamsHostsRevealShared(ts))
	s.Register("teams.hosts.credentials.rosterList", makeTeamsHostsRosterList(ts))
	s.Register("teams.hosts.credentials.revealMember", makeTeamsHostsRevealMember(ts))
	s.Register("teams.hosts.credentials.deleteMember", makeTeamsHostsDeleteMember(ts))
	s.Register("teams.hosts.credentials.upsertMine", makeTeamsHostsUpsertMine(ts))

	// Import personal host.
	s.Register("teams.hosts.importPersonal.preview", makeTeamsHostsImportPersonalPreview(ts))
	s.Register("teams.hosts.importPersonal.commit", makeTeamsHostsImportPersonalCommit(ts))
}

func notSignedInErr() *RPCError {
	return &RPCError{Code: CodeNotSignedIn, Message: "not signed in", Data: map[string]string{"kind": "not_signed_in"}}
}

func handleTeamsErr(err error) *RPCError {
	if errors.Is(err, service.ErrNotSignedIn) {
		return notSignedInErr()
	}
	return &RPCError{Code: CodeInternalError, Message: err.Error()}
}

func makeTeamsList(ts *service.TeamsService) Handler {
	return func(ctx context.Context, _ uint64, _ json.RawMessage) (any, *RPCError) {
		list, err := ts.List(ctx)
		if err != nil {
			return nil, handleTeamsErr(err)
		}
		return map[string]any{"teams": list}, nil
	}
}

func makeTeamsCreate(ts *service.TeamsService) Handler {
	return func(ctx context.Context, _ uint64, params json.RawMessage) (any, *RPCError) {
		var p struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		if p.Name == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "name is required"}
		}
		team, err := ts.CreateTeam(ctx, p.Name)
		if err != nil {
			return nil, handleTeamsErr(err)
		}
		return team, nil
	}
}

func makeTeamsRename(ts *service.TeamsService) Handler {
	return func(ctx context.Context, _ uint64, params json.RawMessage) (any, *RPCError) {
		var p struct {
			TeamID string `json:"teamId"`
			Name   string `json:"name"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		if p.TeamID == "" || p.Name == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "teamId and name are required"}
		}
		team, err := ts.RenameTeam(ctx, p.TeamID, p.Name)
		if err != nil {
			return nil, handleTeamsErr(err)
		}
		return team, nil
	}
}

func makeTeamsDelete(ts *service.TeamsService) Handler {
	return func(ctx context.Context, _ uint64, params json.RawMessage) (any, *RPCError) {
		var p struct {
			TeamID string `json:"teamId"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		if p.TeamID == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "teamId is required"}
		}
		if err := ts.DeleteTeam(ctx, p.TeamID); err != nil {
			return nil, handleTeamsErr(err)
		}
		return map[string]bool{"ok": true}, nil
	}
}

func makeTeamsReorder(ts *service.TeamsService) Handler {
	return func(ctx context.Context, _ uint64, params json.RawMessage) (any, *RPCError) {
		var p struct {
			TeamIDs []string `json:"teamIds"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		if len(p.TeamIDs) == 0 {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "teamIds is required"}
		}
		if err := ts.ReorderTeams(ctx, p.TeamIDs); err != nil {
			return nil, handleTeamsErr(err)
		}
		return map[string]bool{"ok": true}, nil
	}
}

func makeTeamsLeave(ts *service.TeamsService) Handler {
	return func(ctx context.Context, _ uint64, params json.RawMessage) (any, *RPCError) {
		var p struct {
			TeamID string `json:"teamId"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		if p.TeamID == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "teamId is required"}
		}
		if err := ts.LeaveTeam(ctx, p.TeamID); err != nil {
			return nil, handleTeamsErr(err)
		}
		return map[string]bool{"ok": true}, nil
	}
}

// ── Hosts ─────────────────────────────────────────────────────────────────────

func makeTeamsHostsList(ts *service.TeamsService) Handler {
	return func(ctx context.Context, _ uint64, params json.RawMessage) (any, *RPCError) {
		var p struct {
			TeamID string `json:"teamId"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		if p.TeamID == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "teamId is required"}
		}
		hosts, err := ts.ListHosts(ctx, p.TeamID)
		if err != nil {
			return nil, handleTeamsErr(err)
		}
		return map[string]any{"hosts": hosts}, nil
	}
}

func makeTeamsHostsCreate(ts *service.TeamsService) Handler {
	return func(ctx context.Context, _ uint64, params json.RawMessage) (any, *RPCError) {
		var p struct {
			TeamID string                       `json:"teamId"`
			Req    teams.CreateTeamHostRequest  `json:"host"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		if p.TeamID == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "teamId is required"}
		}
		host, err := ts.CreateHost(ctx, p.TeamID, p.Req)
		if err != nil {
			return nil, handleTeamsErr(err)
		}
		return host, nil
	}
}

func makeTeamsHostsUpdate(ts *service.TeamsService) Handler {
	return func(ctx context.Context, _ uint64, params json.RawMessage) (any, *RPCError) {
		var p struct {
			HostID string                       `json:"hostId"`
			Req    teams.UpdateTeamHostRequest  `json:"host"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		if p.HostID == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "hostId is required"}
		}
		if err := ts.UpdateHost(ctx, p.HostID, p.Req); err != nil {
			return nil, handleTeamsErr(err)
		}
		return map[string]bool{"ok": true}, nil
	}
}

func makeTeamsHostsDelete(ts *service.TeamsService) Handler {
	return func(ctx context.Context, _ uint64, params json.RawMessage) (any, *RPCError) {
		var p struct {
			HostID string `json:"hostId"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		if p.HostID == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "hostId is required"}
		}
		if err := ts.DeleteHost(ctx, p.HostID); err != nil {
			return nil, handleTeamsErr(err)
		}
		return map[string]bool{"ok": true}, nil
	}
}

// ── Members ───────────────────────────────────────────────────────────────────

func makeTeamsMembersList(ts *service.TeamsService) Handler {
	return func(ctx context.Context, _ uint64, params json.RawMessage) (any, *RPCError) {
		var p struct {
			TeamID string `json:"teamId"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		if p.TeamID == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "teamId is required"}
		}
		members, err := ts.ListMembers(ctx, p.TeamID)
		if err != nil {
			return nil, handleTeamsErr(err)
		}
		return map[string]any{"members": members}, nil
	}
}

func makeTeamsMembersInvite(ts *service.TeamsService) Handler {
	return func(ctx context.Context, _ uint64, params json.RawMessage) (any, *RPCError) {
		var p struct {
			TeamID string          `json:"teamId"`
			Email  string          `json:"email"`
			Role   teams.TeamRole  `json:"role"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		if p.TeamID == "" || p.Email == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "teamId and email are required"}
		}
		if p.Role == "" {
			p.Role = teams.TeamRoleMember
		}
		invite, err := ts.InviteMember(ctx, p.TeamID, p.Email, p.Role)
		if err != nil {
			return nil, handleTeamsErr(err)
		}
		return invite, nil
	}
}

func makeTeamsMembersUpdateRole(ts *service.TeamsService) Handler {
	return func(ctx context.Context, _ uint64, params json.RawMessage) (any, *RPCError) {
		var p struct {
			TeamID   string         `json:"teamId"`
			MemberID string         `json:"memberId"`
			Role     teams.TeamRole `json:"role"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		if p.TeamID == "" || p.MemberID == "" || p.Role == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "teamId, memberId and role are required"}
		}
		if err := ts.UpdateMemberRole(ctx, p.TeamID, p.MemberID, p.Role); err != nil {
			return nil, handleTeamsErr(err)
		}
		return map[string]bool{"ok": true}, nil
	}
}

func makeTeamsMembersRemove(ts *service.TeamsService) Handler {
	return func(ctx context.Context, _ uint64, params json.RawMessage) (any, *RPCError) {
		var p struct {
			TeamID   string `json:"teamId"`
			MemberID string `json:"memberId"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		if p.TeamID == "" || p.MemberID == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "teamId and memberId are required"}
		}
		if err := ts.RemoveMember(ctx, p.TeamID, p.MemberID); err != nil {
			return nil, handleTeamsErr(err)
		}
		return map[string]bool{"ok": true}, nil
	}
}

// ── Invites ───────────────────────────────────────────────────────────────────

func makeTeamsInvitesList(ts *service.TeamsService) Handler {
	return func(ctx context.Context, _ uint64, params json.RawMessage) (any, *RPCError) {
		var p struct {
			TeamID string `json:"teamId"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		if p.TeamID == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "teamId is required"}
		}
		list, err := ts.ListInvites(ctx, p.TeamID)
		if err != nil {
			return nil, handleTeamsErr(err)
		}
		return list, nil
	}
}

func makeTeamsInvitesAccept(ts *service.TeamsService) Handler {
	return func(ctx context.Context, _ uint64, params json.RawMessage) (any, *RPCError) {
		var p struct {
			InviteID string `json:"inviteId"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		if p.InviteID == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "inviteId is required"}
		}
		if err := ts.AcceptInvite(ctx, p.InviteID); err != nil {
			return nil, handleTeamsErr(err)
		}
		return map[string]bool{"ok": true}, nil
	}
}

func makeTeamsInvitesRevoke(ts *service.TeamsService) Handler {
	return func(ctx context.Context, _ uint64, params json.RawMessage) (any, *RPCError) {
		var p struct {
			TeamID   string `json:"teamId"`
			InviteID string `json:"inviteId"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		if p.TeamID == "" || p.InviteID == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "teamId and inviteId are required"}
		}
		if err := ts.RevokeInvite(ctx, p.TeamID, p.InviteID); err != nil {
			return nil, handleTeamsErr(err)
		}
		return map[string]bool{"ok": true}, nil
	}
}

// ── Audit ─────────────────────────────────────────────────────────────────────

func makeTeamsAuditList(ts *service.TeamsService) Handler {
	return func(ctx context.Context, _ uint64, params json.RawMessage) (any, *RPCError) {
		var p struct {
			TeamID string `json:"teamId"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		if p.TeamID == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "teamId is required"}
		}
		events, err := ts.ListAuditEvents(ctx, p.TeamID)
		if err != nil {
			return nil, handleTeamsErr(err)
		}
		return map[string]any{"events": events}, nil
	}
}

// ── Credentials ───────────────────────────────────────────────────────────────

func makeTeamsHostsRevealShared(ts *service.TeamsService) Handler {
	return func(ctx context.Context, _ uint64, params json.RawMessage) (any, *RPCError) {
		var p struct {
			HostID string `json:"hostId"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		if p.HostID == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "hostId is required"}
		}
		cred, err := ts.RevealSharedCredential(ctx, p.HostID)
		if err != nil {
			return nil, handleTeamsErr(err)
		}
		return cred, nil
	}
}

func makeTeamsHostsRosterList(ts *service.TeamsService) Handler {
	return func(ctx context.Context, _ uint64, params json.RawMessage) (any, *RPCError) {
		var p struct {
			HostID string `json:"hostId"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		if p.HostID == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "hostId is required"}
		}
		roster, err := ts.ListCredentialRoster(ctx, p.HostID)
		if err != nil {
			return nil, handleTeamsErr(err)
		}
		return map[string]any{"roster": roster}, nil
	}
}

func makeTeamsHostsRevealMember(ts *service.TeamsService) Handler {
	return func(ctx context.Context, _ uint64, params json.RawMessage) (any, *RPCError) {
		var p struct {
			HostID   string `json:"hostId"`
			MemberID string `json:"memberId"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		if p.HostID == "" || p.MemberID == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "hostId and memberId are required"}
		}
		cred, err := ts.RevealMemberCredential(ctx, p.HostID, p.MemberID)
		if err != nil {
			return nil, handleTeamsErr(err)
		}
		return cred, nil
	}
}

func makeTeamsHostsDeleteMember(ts *service.TeamsService) Handler {
	return func(ctx context.Context, _ uint64, params json.RawMessage) (any, *RPCError) {
		var p struct {
			HostID   string `json:"hostId"`
			MemberID string `json:"memberId"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		if p.HostID == "" || p.MemberID == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "hostId and memberId are required"}
		}
		if err := ts.DeleteMemberCredential(ctx, p.HostID, p.MemberID); err != nil {
			return nil, handleTeamsErr(err)
		}
		return map[string]bool{"ok": true}, nil
	}
}

func makeTeamsHostsUpsertMine(ts *service.TeamsService) Handler {
	return func(ctx context.Context, _ uint64, params json.RawMessage) (any, *RPCError) {
		var p struct {
			HostID         string `json:"hostId"`
			CredentialType string `json:"credentialType"`
			Secret         string `json:"secret"`
			Username       string `json:"username,omitempty"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		if p.HostID == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "hostId is required"}
		}
		if p.Secret == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "secret is required"}
		}
		if p.CredentialType == "" {
			p.CredentialType = "password"
		}
		req := teams.UpsertMyCredentialRequest{
			CredentialType: p.CredentialType,
			Secret:         p.Secret,
			Username:       p.Username,
		}
		if err := ts.UpsertMyCredential(ctx, p.HostID, req); err != nil {
			return nil, handleTeamsErr(err)
		}
		return map[string]bool{"ok": true}, nil
	}
}

// ── Import personal host ──────────────────────────────────────────────────────

func makeTeamsHostsImportPersonalPreview(ts *service.TeamsService) Handler {
	return func(ctx context.Context, _ uint64, params json.RawMessage) (any, *RPCError) {
		var p struct {
			PersonalHostID string `json:"personalHostId"`
			TeamID         string `json:"teamId"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		if p.PersonalHostID == "" || p.TeamID == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "personalHostId and teamId are required"}
		}
		if ts.Vault == nil {
			return nil, &RPCError{Code: CodeInternalError, Message: "vault not available for import"}
		}
		result, err := ts.ImportPersonalHostPreview(ctx, p.PersonalHostID, p.TeamID)
		if err != nil {
			if errors.Is(err, service.ErrVaultLocked) {
				return nil, &RPCError{Code: CodeVaultLocked, Message: "vault is locked", Data: map[string]string{"kind": "locked"}}
			}
			return nil, handleTeamsErr(err)
		}
		return result, nil
	}
}

func makeTeamsHostsImportPersonalCommit(ts *service.TeamsService) Handler {
	return func(ctx context.Context, _ uint64, params json.RawMessage) (any, *RPCError) {
		var p teams.ImportPersonalHostCommitRequest
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()}
		}
		if p.PersonalHostID == "" || p.TeamID == "" || p.Action == "" {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "personalHostId, teamId and action are required"}
		}
		if ts.Vault == nil {
			return nil, &RPCError{Code: CodeInternalError, Message: "vault not available for import"}
		}
		if err := ts.ImportPersonalHostCommit(ctx, p); err != nil {
			if errors.Is(err, service.ErrVaultLocked) {
				return nil, &RPCError{Code: CodeVaultLocked, Message: "vault is locked", Data: map[string]string{"kind": "locked"}}
			}
			return nil, handleTeamsErr(err)
		}
		return map[string]bool{"ok": true}, nil
	}
}
