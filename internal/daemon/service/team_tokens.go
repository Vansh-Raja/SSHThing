package service

import (
	"context"
	"fmt"

	"github.com/Vansh-Raja/SSHThing/internal/teams"
	"github.com/Vansh-Raja/SSHThing/internal/teamsclient"
)

// TeamTokensService provides team automation token management via the cloud API.
type TeamTokensService struct {
	Client *teamsclient.Client
}

func teamTokenSummaryFromCloud(t teams.TeamAutomationToken) TokenSummary {
	ts := TokenSummary{
		ID:        t.ID,
		Name:      t.Name,
		Status:    t.Status,
		CreatedAt: t.CreatedAt,
		UseCount:  t.UseCount,
		HostCount: t.HostCount,
	}
	if t.RevokedAt != nil {
		v := *t.RevokedAt
		ts.RevokedAt = &v
	}
	if t.LastUsedAt != nil {
		v := *t.LastUsedAt
		ts.LastUsedAt = &v
	}
	return ts
}

// List returns team automation tokens for the given team.
func (tts *TeamTokensService) List(ctx context.Context, teamID string) ([]TokenSummary, error) {
	if tts.Client == nil || !tts.Client.Enabled() {
		return nil, fmt.Errorf("teams client not configured")
	}
	token, err := accessToken(ctx, tts.Client)
	if err != nil {
		return nil, err
	}
	raw, err := tts.Client.ListTeamTokens(ctx, token, teamID)
	if err != nil {
		return nil, fmt.Errorf("list team tokens: %w", err)
	}
	out := make([]TokenSummary, 0, len(raw))
	for _, t := range raw {
		out = append(out, teamTokenSummaryFromCloud(t))
	}
	return out, nil
}

// Create creates a new team automation token.
func (tts *TeamTokensService) Create(ctx context.Context, teamID string, name string, hostIDs []string) (string, error) {
	if tts.Client == nil || !tts.Client.Enabled() {
		return "", fmt.Errorf("teams client not configured")
	}
	token, err := accessToken(ctx, tts.Client)
	if err != nil {
		return "", err
	}
	res, err := tts.Client.CreateTeamToken(ctx, token, teamID, teams.CreateTeamAutomationTokenRequest{
		Name:    name,
		HostIDs: hostIDs,
	})
	if err != nil {
		return "", fmt.Errorf("create team token: %w", err)
	}
	return res.RawToken, nil
}

// Revoke revokes a team automation token.
func (tts *TeamTokensService) Revoke(ctx context.Context, teamID, tokenDocID string) error {
	if tts.Client == nil || !tts.Client.Enabled() {
		return fmt.Errorf("teams client not configured")
	}
	token, err := accessToken(ctx, tts.Client)
	if err != nil {
		return err
	}
	if err := tts.Client.RevokeTeamToken(ctx, token, teamID, tokenDocID); err != nil {
		return fmt.Errorf("revoke team token: %w", err)
	}
	return nil
}

// DeleteRevoked permanently deletes a revoked team automation token.
func (tts *TeamTokensService) DeleteRevoked(ctx context.Context, teamID, tokenDocID string) error {
	if tts.Client == nil || !tts.Client.Enabled() {
		return fmt.Errorf("teams client not configured")
	}
	token, err := accessToken(ctx, tts.Client)
	if err != nil {
		return err
	}
	if err := tts.Client.DeleteRevokedTeamToken(ctx, token, teamID, tokenDocID); err != nil {
		return fmt.Errorf("delete revoked team token: %w", err)
	}
	return nil
}
