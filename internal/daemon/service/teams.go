package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/Vansh-Raja/SSHThing/internal/ssh"
	"github.com/Vansh-Raja/SSHThing/internal/teams"
	"github.com/Vansh-Raja/SSHThing/internal/teamsclient"
	"github.com/Vansh-Raja/SSHThing/internal/teamssession"
)

// ErrNotSignedIn is returned when no Convex session is active.
var ErrNotSignedIn = fmt.Errorf("not signed in")

// TeamsService provides team management via the Convex REST API.
// Vault is optional; it is only required for operations that read local host
// data (e.g. ImportPersonalHost). If Vault is nil those methods return an error.
type TeamsService struct {
	Client *teamsclient.Client
	Vault  *Vault // optional — needed for ImportPersonalHost
}

func (ts *TeamsService) accessToken(ctx context.Context) (string, error) {
	sess, err := teamssession.Load()
	if err != nil {
		return "", fmt.Errorf("load session: %w", err)
	}
	if sess.AccessToken == "" {
		return "", ErrNotSignedIn
	}
	return sess.AccessToken, nil
}

// ── Team-level ────────────────────────────────────────────────────────────────

func (ts *TeamsService) List(ctx context.Context) ([]teams.TeamSummary, error) {
	tok, err := ts.accessToken(ctx)
	if err != nil {
		return nil, err
	}
	return ts.Client.ListTeams(ctx, tok)
}

// ── Hosts ─────────────────────────────────────────────────────────────────────

func (ts *TeamsService) ListHosts(ctx context.Context, teamID string) ([]teams.TeamHost, error) {
	tok, err := ts.accessToken(ctx)
	if err != nil {
		return nil, err
	}
	return ts.Client.ListTeamHosts(ctx, tok, teamID)
}

func (ts *TeamsService) CreateHost(ctx context.Context, teamID string, req teams.CreateTeamHostRequest) (teams.TeamHost, error) {
	tok, err := ts.accessToken(ctx)
	if err != nil {
		return teams.TeamHost{}, err
	}
	return ts.Client.CreateTeamHost(ctx, tok, teamID, req)
}

func (ts *TeamsService) UpdateHost(ctx context.Context, hostID string, req teams.UpdateTeamHostRequest) error {
	tok, err := ts.accessToken(ctx)
	if err != nil {
		return err
	}
	return ts.Client.UpdateTeamHost(ctx, tok, hostID, req)
}

func (ts *TeamsService) DeleteHost(ctx context.Context, hostID string) error {
	tok, err := ts.accessToken(ctx)
	if err != nil {
		return err
	}
	return ts.Client.DeleteTeamHost(ctx, tok, hostID)
}

// ── Members ───────────────────────────────────────────────────────────────────

func (ts *TeamsService) ListMembers(ctx context.Context, teamID string) ([]teams.TeamMember, error) {
	tok, err := ts.accessToken(ctx)
	if err != nil {
		return nil, err
	}
	return ts.Client.ListTeamMembers(ctx, tok, teamID)
}

func (ts *TeamsService) InviteMember(ctx context.Context, teamID, email string, role teams.TeamRole) (teams.TeamInvite, error) {
	tok, err := ts.accessToken(ctx)
	if err != nil {
		return teams.TeamInvite{}, err
	}
	return ts.Client.InviteTeamMember(ctx, tok, teamID, email, role)
}

func (ts *TeamsService) UpdateMemberRole(ctx context.Context, teamID, memberID string, role teams.TeamRole) error {
	tok, err := ts.accessToken(ctx)
	if err != nil {
		return err
	}
	return ts.Client.UpdateTeamMemberRole(ctx, tok, teamID, memberID, role)
}

func (ts *TeamsService) RemoveMember(ctx context.Context, teamID, memberID string) error {
	tok, err := ts.accessToken(ctx)
	if err != nil {
		return err
	}
	return ts.Client.RemoveTeamMember(ctx, tok, teamID, memberID)
}

// ── Invites ───────────────────────────────────────────────────────────────────

func (ts *TeamsService) ListInvites(ctx context.Context, teamID string) (teams.TeamInviteList, error) {
	tok, err := ts.accessToken(ctx)
	if err != nil {
		return teams.TeamInviteList{}, err
	}
	return ts.Client.ListTeamInvites(ctx, tok, teamID)
}

func (ts *TeamsService) AcceptInvite(ctx context.Context, inviteID string) error {
	tok, err := ts.accessToken(ctx)
	if err != nil {
		return err
	}
	return ts.Client.AcceptTeamInvite(ctx, tok, inviteID)
}

func (ts *TeamsService) RevokeInvite(ctx context.Context, teamID, inviteID string) error {
	tok, err := ts.accessToken(ctx)
	if err != nil {
		return err
	}
	return ts.Client.RevokeTeamInvite(ctx, tok, teamID, inviteID)
}

// ── Team CRUD ─────────────────────────────────────────────────────────────────

func (ts *TeamsService) CreateTeam(ctx context.Context, name string) (teams.TeamSummary, error) {
	tok, err := ts.accessToken(ctx)
	if err != nil {
		return teams.TeamSummary{}, err
	}
	return ts.Client.CreateTeam(ctx, tok, name)
}

func (ts *TeamsService) RenameTeam(ctx context.Context, teamID, name string) (teams.TeamSummary, error) {
	tok, err := ts.accessToken(ctx)
	if err != nil {
		return teams.TeamSummary{}, err
	}
	return ts.Client.RenameTeam(ctx, tok, teamID, name)
}

func (ts *TeamsService) DeleteTeam(ctx context.Context, teamID string) error {
	tok, err := ts.accessToken(ctx)
	if err != nil {
		return err
	}
	return ts.Client.DeleteTeam(ctx, tok, teamID)
}

func (ts *TeamsService) ReorderTeams(ctx context.Context, teamIDs []string) error {
	tok, err := ts.accessToken(ctx)
	if err != nil {
		return err
	}
	return ts.Client.ReorderTeams(ctx, tok, teamIDs)
}

// LeaveTeam removes the current user from the team using their member record.
// It looks up the current user's clerk user ID from the stored session, finds
// their member entry in the team, and removes it.
func (ts *TeamsService) LeaveTeam(ctx context.Context, teamID string) error {
	sess, err := teamssession.Load()
	if err != nil {
		return fmt.Errorf("load session: %w", err)
	}
	if sess.AccessToken == "" || sess.UserID == "" {
		return ErrNotSignedIn
	}
	members, err := ts.Client.ListTeamMembers(ctx, sess.AccessToken, teamID)
	if err != nil {
		return fmt.Errorf("list members: %w", err)
	}
	for _, m := range members {
		if m.ClerkUserID == sess.UserID {
			return ts.Client.RemoveTeamMember(ctx, sess.AccessToken, teamID, m.ID)
		}
	}
	return fmt.Errorf("current user is not a member of team %s", teamID)
}

// ── Audit ─────────────────────────────────────────────────────────────────────

func (ts *TeamsService) ListAuditEvents(ctx context.Context, teamID string) ([]teams.TeamAuditEvent, error) {
	tok, err := ts.accessToken(ctx)
	if err != nil {
		return nil, err
	}
	return ts.Client.ListTeamAuditEvents(ctx, tok, teamID)
}

// ── Credentials ───────────────────────────────────────────────────────────────

// RevealSharedCredential reveals the shared credential for a team host.
func (ts *TeamsService) RevealSharedCredential(ctx context.Context, hostID string) (teams.RevealedTeamHostCredential, error) {
	tok, err := ts.accessToken(ctx)
	if err != nil {
		return teams.RevealedTeamHostCredential{}, err
	}
	return ts.Client.RevealSharedCredential(ctx, tok, hostID)
}

// ListCredentialRoster returns the per-member credential roster for a host.
func (ts *TeamsService) ListCredentialRoster(ctx context.Context, hostID string) ([]teams.TeamHostCredentialRosterEntry, error) {
	tok, err := ts.accessToken(ctx)
	if err != nil {
		return nil, err
	}
	return ts.Client.ListHostCredentialRoster(ctx, tok, hostID)
}

// RevealMemberCredential reveals the per-member credential for a specific member (admin action).
func (ts *TeamsService) RevealMemberCredential(ctx context.Context, hostID, memberID string) (teams.RevealedTeamHostCredential, error) {
	tok, err := ts.accessToken(ctx)
	if err != nil {
		return teams.RevealedTeamHostCredential{}, err
	}
	return ts.Client.RevealMemberCredential(ctx, tok, hostID, memberID)
}

// DeleteMemberCredential deletes a member's credential (admin action).
func (ts *TeamsService) DeleteMemberCredential(ctx context.Context, hostID, memberID string) error {
	tok, err := ts.accessToken(ctx)
	if err != nil {
		return err
	}
	return ts.Client.DeleteMemberCredentialAsAdmin(ctx, tok, hostID, memberID)
}

// UpsertMyCredential sets the current user's per-member credential for a host.
func (ts *TeamsService) UpsertMyCredential(ctx context.Context, hostID string, req teams.UpsertMyCredentialRequest) error {
	tok, err := ts.accessToken(ctx)
	if err != nil {
		return err
	}
	return ts.Client.UpsertMyCredential(ctx, tok, hostID, req)
}

// ── Import personal host ──────────────────────────────────────────────────────

// normalizeText lower-cases and strips extra whitespace (mirrors TUI normalizeTeamText).
func normalizeText(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// ImportPersonalHostPreview checks whether importing a personal host into a
// team would produce a conflict. It mirrors the TUI's findImportConflict logic.
func (ts *TeamsService) ImportPersonalHostPreview(ctx context.Context, personalHostID string, teamID string) (teams.ImportPersonalHostPreviewResult, error) {
	store := ts.Vault.Store()
	if store == nil {
		return teams.ImportPersonalHostPreviewResult{}, ErrVaultLocked
	}

	intID := 0
	if _, err := fmt.Sscanf(personalHostID, "%d", &intID); err != nil || intID == 0 {
		return teams.ImportPersonalHostPreviewResult{}, fmt.Errorf("invalid personalHostId %q", personalHostID)
	}

	host, err := store.GetHostByID(intID)
	if err != nil {
		return teams.ImportPersonalHostPreviewResult{}, fmt.Errorf("get personal host: %w", err)
	}

	tok, err := ts.accessToken(ctx)
	if err != nil {
		return teams.ImportPersonalHostPreviewResult{}, err
	}

	// Build the proposed create request (mirrors TUI buildTeamHostRequestFromPersonalHost).
	req := teams.CreateTeamHostRequest{
		Label:            strings.TrimSpace(host.Label),
		Hostname:         host.Hostname,
		Username:         host.Username,
		Port:             host.Port,
		Group:            strings.TrimSpace(host.GroupName),
		Tags:             append([]string(nil), host.Tags...),
		CredentialMode:   "shared",
		CredentialType:   "none",
		SecretVisibility: "revealed_to_access_holders",
	}
	if host.KeyData != "" {
		secret, sErr := store.GetHostSecret(intID)
		if sErr != nil {
			return teams.ImportPersonalHostPreviewResult{}, fmt.Errorf("read host secret: %w", sErr)
		}
		switch host.KeyType {
		case "password":
			req.CredentialType = "password"
			req.SharedCredential = secret
		default:
			if vErr := ssh.ValidatePrivateKey(secret); vErr != nil {
				return teams.ImportPersonalHostPreviewResult{}, fmt.Errorf("local private key is invalid: %w", vErr)
			}
			req.CredentialType = "private_key"
			req.SharedCredential = normalizePrivateKey(secret)
		}
	}

	// Find existing hosts in the team with the same hostname.
	teamHosts, err := ts.Client.ListTeamHosts(ctx, tok, teamID)
	if err != nil {
		return teams.ImportPersonalHostPreviewResult{}, fmt.Errorf("list team hosts: %w", err)
	}

	var bestConflict *teams.TeamHostDetail
	for _, th := range teamHosts {
		if normalizeText(th.Hostname) != normalizeText(req.Hostname) {
			continue
		}
		detail, dErr := ts.Client.GetTeamHost(ctx, tok, th.ID)
		if dErr != nil {
			continue
		}
		// Identical: same label + hostname + username + port
		if normalizeText(detail.Label) == normalizeText(req.Label) &&
			normalizeText(detail.Hostname) == normalizeText(req.Hostname) &&
			normalizeText(detail.Username) == normalizeText(req.Username) &&
			detail.Port == req.Port {
			return teams.ImportPersonalHostPreviewResult{
				HasConflict:    true,
				IsIdentical:    true,
				ExistingHostID: detail.ID,
				ExistingLabel:  detail.Label,
				Proposed:       req,
			}, nil
		}
		if bestConflict == nil {
			copy := detail
			bestConflict = &copy
		}
	}

	if bestConflict != nil {
		return teams.ImportPersonalHostPreviewResult{
			HasConflict:    true,
			IsIdentical:    false,
			ExistingHostID: bestConflict.ID,
			ExistingLabel:  bestConflict.Label,
			Proposed:       req,
		}, nil
	}

	return teams.ImportPersonalHostPreviewResult{
		HasConflict: false,
		Proposed:    req,
	}, nil
}

// ImportPersonalHostCommit performs the actual import after the user has
// resolved any conflict. Action is "create", "update", or "duplicate".
func (ts *TeamsService) ImportPersonalHostCommit(ctx context.Context, req teams.ImportPersonalHostCommitRequest) error {
	store := ts.Vault.Store()
	if store == nil {
		return ErrVaultLocked
	}

	intID := 0
	if _, err := fmt.Sscanf(req.PersonalHostID, "%d", &intID); err != nil || intID == 0 {
		return fmt.Errorf("invalid personalHostId %q", req.PersonalHostID)
	}

	host, err := store.GetHostByID(intID)
	if err != nil {
		return fmt.Errorf("get personal host: %w", err)
	}

	tok, err := ts.accessToken(ctx)
	if err != nil {
		return err
	}

	// Build the create request.
	createReq := teams.CreateTeamHostRequest{
		Label:            strings.TrimSpace(host.Label),
		Hostname:         host.Hostname,
		Username:         host.Username,
		Port:             host.Port,
		Group:            strings.TrimSpace(host.GroupName),
		Tags:             append([]string(nil), host.Tags...),
		CredentialMode:   "shared",
		CredentialType:   "none",
		SecretVisibility: "revealed_to_access_holders",
	}
	if host.KeyData != "" {
		secret, sErr := store.GetHostSecret(intID)
		if sErr != nil {
			return fmt.Errorf("read host secret: %w", sErr)
		}
		switch host.KeyType {
		case "password":
			createReq.CredentialType = "password"
			createReq.SharedCredential = secret
		default:
			if vErr := ssh.ValidatePrivateKey(secret); vErr != nil {
				return fmt.Errorf("local private key is invalid: %w", vErr)
			}
			createReq.CredentialType = "private_key"
			createReq.SharedCredential = normalizePrivateKey(secret)
		}
	}

	switch req.Action {
	case teams.ImportActionCreate, teams.ImportActionDuplicate:
		_, err = ts.Client.CreateTeamHost(ctx, tok, req.TeamID, createReq)
		return err

	case teams.ImportActionUpdate:
		if req.ExistingHostID == "" {
			return fmt.Errorf("existingHostId is required for update action")
		}
		updateReq := teams.UpdateTeamHostRequest{
			Label:            createReq.Label,
			Hostname:         createReq.Hostname,
			Username:         createReq.Username,
			Port:             createReq.Port,
			Group:            createReq.Group,
			Tags:             createReq.Tags,
			CredentialMode:   createReq.CredentialMode,
			CredentialType:   createReq.CredentialType,
			SecretVisibility: createReq.SecretVisibility,
			SharedCredential: createReq.SharedCredential,
		}
		return ts.Client.UpdateTeamHost(ctx, tok, req.ExistingHostID, updateReq)

	default:
		return fmt.Errorf("unknown action %q; expected create, update, or duplicate", req.Action)
	}
}

// normalizePrivateKey ensures a PEM private key ends with a newline.
func normalizePrivateKey(s string) string {
	s = strings.TrimSpace(s)
	if s != "" && !strings.HasSuffix(s, "\n") {
		s += "\n"
	}
	return s
}
