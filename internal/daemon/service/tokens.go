package service

import (
	"crypto/rand"
	"fmt"

	"github.com/Vansh-Raja/SSHThing/internal/authtoken"
	"github.com/Vansh-Raja/SSHThing/internal/securestore"
	"github.com/Vansh-Raja/SSHThing/internal/unlock"
)

// TokensService provides personal automation token management.
type TokensService struct {
	Vault    *Vault
	CfgStore *CfgStore
}

// CreateTokenParams holds the parameters for token creation.
type CreateTokenParams struct {
	Name   string                `json:"name"`
	Grants []authtoken.HostGrant `json:"grants"`
}

func tokenSummaryFromRecord(t authtoken.TokenSummary) TokenSummary {
	status := "active"
	if t.RevokedAt != nil {
		status = "revoked"
	}
	ts := TokenSummary{
		ID:        t.TokenID,
		Name:      t.Name,
		Status:    status,
		CreatedAt: t.CreatedAt.Unix(),
		UseCount:  t.UseCount,
		HostCount: t.HostCount,
	}
	if t.RevokedAt != nil {
		v := t.RevokedAt.Unix()
		ts.RevokedAt = &v
	}
	if t.LastUsedAt != nil {
		v := t.LastUsedAt.Unix()
		ts.LastUsedAt = &v
	}
	return ts
}

// List returns summaries of all non-deleted personal tokens.
func (ts *TokensService) List() ([]TokenSummary, error) {
	vault, err := authtoken.LoadVault()
	if err != nil {
		return nil, fmt.Errorf("load token vault: %w", err)
	}
	raw := vault.ListSummaries()
	out := make([]TokenSummary, 0, len(raw))
	for _, r := range raw {
		out = append(out, tokenSummaryFromRecord(r))
	}
	return out, nil
}

// Create generates a new automation token and saves it to the token vault.
func (ts *TokensService) Create(params CreateTokenParams) (string, error) {
	if len(params.Grants) == 0 {
		return "", fmt.Errorf("at least one host grant is required")
	}

	// The token must be bound to the vault unlock secret so the daemon can
	// re-derive the DB key when processing token-auth requests.
	dbSecret, _, ok, err := unlock.Load()
	if err != nil {
		return "", fmt.Errorf("load unlock session: %w", err)
	}
	if !ok {
		return "", ErrVaultLocked
	}

	pepper, _ := securestore.GetOrCreateDevicePepper(rand.Reader)
	syncEnabled := false
	if ts.CfgStore != nil {
		syncEnabled = ts.CfgStore.Get().Automation.SyncTokenDefinitions
	}
	opts := authtoken.CreateOptions{
		DevicePepper: pepper,
		BindToDevice: len(pepper) > 0,
		SyncEnabled:  syncEnabled,
	}

	raw, rec, err := authtoken.CreateToken(params.Name, params.Grants, dbSecret, opts)
	if err != nil {
		return "", fmt.Errorf("create token: %w", err)
	}

	vault, err := authtoken.LoadVault()
	if err != nil {
		return "", fmt.Errorf("load token vault: %w", err)
	}
	if err := vault.AddToken(raw, rec); err != nil {
		return "", fmt.Errorf("add token: %w", err)
	}
	if err := authtoken.SaveVault(vault); err != nil {
		return "", fmt.Errorf("save token vault: %w", err)
	}
	return raw, nil
}

// Revoke marks a token as revoked without deleting it.
func (ts *TokensService) Revoke(tokenID string) error {
	vault, err := authtoken.LoadVault()
	if err != nil {
		return fmt.Errorf("load token vault: %w", err)
	}
	if !vault.RevokeToken(tokenID) {
		return fmt.Errorf("token not found")
	}
	return authtoken.SaveVault(vault)
}

// DeleteRevoked permanently deletes a previously-revoked token.
func (ts *TokensService) DeleteRevoked(tokenID string) error {
	vault, err := authtoken.LoadVault()
	if err != nil {
		return fmt.Errorf("load token vault: %w", err)
	}
	deleted, err := vault.DeleteRevokedToken(tokenID)
	if err != nil {
		return err
	}
	if !deleted {
		return fmt.Errorf("token not found or not revoked")
	}
	return authtoken.SaveVault(vault)
}
