package authtoken

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Vansh-Raja/SSHThing/internal/config"
)

func VaultPath() (string, error) {
	dir, err := config.DataDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return filepath.Join(dir, "tokens.json"), nil
}

func LoadVault() (*Vault, error) {
	path, err := VaultPath()
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return &Vault{Version: vaultVersion, Tokens: []StoredToken{}}, nil
	}
	if err != nil {
		return nil, err
	}
	var v Vault
	if err := json.Unmarshal(b, &v); err != nil {
		return nil, err
	}
	if v.Version == 0 {
		v.Version = 1
	}
	if v.Tokens == nil {
		v.Tokens = []StoredToken{}
	}
	if v.Version < vaultVersion {
		migrateVault(&v)
	}
	return &v, nil
}

func migrateVault(v *Vault) {
	if v == nil {
		return
	}
	for i := range v.Tokens {
		t := &v.Tokens[i]
		if t.CreatedAt.IsZero() {
			t.CreatedAt = time.Now().UTC()
		}
		if t.UpdatedAt.IsZero() {
			t.UpdatedAt = t.CreatedAt
		}
	}
	v.Version = vaultVersion
}

func SaveVault(v *Vault) error {
	if v == nil {
		return fmt.Errorf("vault is nil")
	}
	path, err := VaultPath()
	if err != nil {
		return err
	}
	v.Version = vaultVersion
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (v *Vault) AddToken(rawToken string, rec StoredToken) error {
	if v == nil {
		return fmt.Errorf("vault is nil")
	}
	id, _, err := Parse(rawToken)
	if err != nil {
		return err
	}
	if id != rec.TokenID {
		return fmt.Errorf("token record mismatch")
	}
	for _, t := range v.Tokens {
		if t.TokenID == rec.TokenID {
			return fmt.Errorf("token id already exists")
		}
	}
	v.Tokens = append(v.Tokens, rec)
	sort.SliceStable(v.Tokens, func(i, j int) bool {
		return v.Tokens[i].CreatedAt.After(v.Tokens[j].CreatedAt)
	})
	return nil
}

func (v *Vault) ListSummaries() []TokenSummary {
	if v == nil {
		return nil
	}
	out := make([]TokenSummary, 0, len(v.Tokens))
	for _, t := range v.Tokens {
		if t.DeletedAt != nil {
			continue
		}
		out = append(out, TokenSummary{
			TokenID:     t.TokenID,
			Name:        t.Name,
			HostCount:   len(t.Hosts),
			CreatedAt:   t.CreatedAt,
			UpdatedAt:   t.UpdatedAt,
			LastUsedAt:  t.LastUsedAt,
			UseCount:    t.UseCount,
			RevokedAt:   t.RevokedAt,
			DeletedAt:   t.DeletedAt,
			ExpiresAt:   t.ExpiresAt,
			MaxUses:     t.MaxUses,
			SyncEnabled: t.SyncEnabled,
			Usable:      t.IsUsable(),
			Legacy:      t.IsLegacyPayload(),
		})
	}
	return out
}

func (v *Vault) RevokeToken(tokenID string) bool {
	if v == nil {
		return false
	}
	tokenID = strings.TrimSpace(tokenID)
	if tokenID == "" {
		return false
	}
	for i := range v.Tokens {
		if v.Tokens[i].TokenID != tokenID || v.Tokens[i].DeletedAt != nil {
			continue
		}
		now := time.Now().UTC()
		v.Tokens[i].RevokedAt = &now
		v.Tokens[i].UpdatedAt = now
		return true
	}
	return false
}

func (v *Vault) DeleteRevokedToken(tokenID string) (bool, error) {
	if v == nil {
		return false, fmt.Errorf("vault is nil")
	}
	tokenID = strings.TrimSpace(tokenID)
	if tokenID == "" {
		return false, nil
	}
	for i := range v.Tokens {
		t := &v.Tokens[i]
		if t.TokenID != tokenID {
			continue
		}
		if t.RevokedAt == nil {
			return false, fmt.Errorf("token must be revoked before deletion")
		}
		if t.SyncEnabled {
			now := time.Now().UTC()
			t.DeletedAt = &now
			t.UpdatedAt = now
			t.Salt = ""
			t.Hash = ""
			t.UnlockSalt = ""
			t.UnlockData = ""
			t.UnlockBound = false
			return true, nil
		}
		v.Tokens = append(v.Tokens[:i], v.Tokens[i+1:]...)
		return true, nil
	}
	return false, nil
}

func (v *Vault) SyncHostLabels(labels map[int]string) bool {
	if v == nil {
		return false
	}
	changed := false
	for ti := range v.Tokens {
		tokenChanged := false
		for hi := range v.Tokens[ti].Hosts {
			h := &v.Tokens[ti].Hosts[hi]
			newLabel, ok := labels[h.HostID]
			if !ok {
				continue
			}
			newLabel = strings.TrimSpace(newLabel)
			if newLabel == "" || newLabel == h.DisplayLabel {
				continue
			}
			h.DisplayLabel = newLabel
			tokenChanged = true
			changed = true
		}
		if tokenChanged {
			v.Tokens[ti].UpdatedAt = time.Now().UTC()
		}
	}
	return changed
}

func (v *Vault) Resolve(rawToken string, targetLabel string, devicePepper []byte) (ResolveResult, error) {
	if v == nil {
		return ResolveResult{}, fmt.Errorf("vault is nil")
	}
	tokenID, _, err := Parse(rawToken)
	if err != nil {
		return ResolveResult{}, err
	}
	for i := range v.Tokens {
		if v.Tokens[i].TokenID != tokenID {
			continue
		}
		result, err := resolveRecord(rawToken, v.Tokens[i], targetLabel, devicePepper)
		if err != nil {
			return ResolveResult{}, err
		}
		result.TokenIndex = i
		return result, nil
	}
	return ResolveResult{}, fmt.Errorf("token not found")
}

func (v *Vault) MarkUsed(tokenIndex int) {
	if v == nil || tokenIndex < 0 || tokenIndex >= len(v.Tokens) {
		return
	}
	now := time.Now().UTC()
	v.Tokens[tokenIndex].UseCount++
	v.Tokens[tokenIndex].LastUsedAt = &now
	v.Tokens[tokenIndex].UpdatedAt = now
}

func (v *Vault) ActivateToken(tokenID string, dbUnlockSecret string, devicePepper []byte) (string, error) {
	if v == nil {
		return "", fmt.Errorf("vault is nil")
	}
	tokenID = strings.TrimSpace(tokenID)
	if tokenID == "" {
		return "", fmt.Errorf("missing token id")
	}
	for i := range v.Tokens {
		t := &v.Tokens[i]
		if t.TokenID != tokenID || t.DeletedAt != nil {
			continue
		}
		grants := make([]HostGrant, 0, len(t.Hosts))
		for _, h := range t.Hosts {
			grants = append(grants, HostGrant{HostID: h.HostID, DisplayLabel: h.DisplayLabel})
		}
		opts := CreateOptions{DevicePepper: devicePepper, BindToDevice: len(devicePepper) > 0, ExpiresAt: t.ExpiresAt, MaxUses: t.MaxUses, SyncEnabled: t.SyncEnabled}
		raw, rec, err := createTokenWithID(t.TokenID, t.Name, grants, dbUnlockSecret, opts)
		if err != nil {
			return "", err
		}
		rec.CreatedAt = t.CreatedAt
		rec.RevokedAt = t.RevokedAt
		rec.DeletedAt = t.DeletedAt
		rec.UseCount = t.UseCount
		rec.LastUsedAt = t.LastUsedAt
		v.Tokens[i] = rec
		return raw, nil
	}
	return "", fmt.Errorf("token not found")
}

func (v *Vault) ExportSyncDefinitions() []SyncTokenDef {
	if v == nil {
		return nil
	}
	out := make([]SyncTokenDef, 0)
	for _, t := range v.Tokens {
		if !t.SyncEnabled {
			continue
		}
		hosts := make([]SyncTokenHost, 0, len(t.Hosts))
		for _, h := range t.Hosts {
			hosts = append(hosts, SyncTokenHost{HostID: h.HostID, DisplayLabel: h.DisplayLabel})
		}
		out = append(out, SyncTokenDef{
			TokenID:     t.TokenID,
			Name:        t.Name,
			CreatedAt:   t.CreatedAt,
			UpdatedAt:   t.UpdatedAt,
			RevokedAt:   t.RevokedAt,
			DeletedAt:   t.DeletedAt,
			ExpiresAt:   t.ExpiresAt,
			MaxUses:     t.MaxUses,
			SyncEnabled: t.SyncEnabled,
			Hosts:       hosts,
		})
	}
	return out
}

func (v *Vault) MergeSyncDefinitions(defs []SyncTokenDef) bool {
	if v == nil {
		return false
	}
	if len(defs) == 0 {
		return false
	}
	changed := false
	for _, d := range defs {
		if strings.TrimSpace(d.TokenID) == "" {
			continue
		}
		idx := -1
		for i := range v.Tokens {
			if v.Tokens[i].TokenID == d.TokenID {
				idx = i
				break
			}
		}
		if idx < 0 {
			hosts := make([]StoredTokenHost, 0, len(d.Hosts))
			for _, h := range d.Hosts {
				hosts = append(hosts, StoredTokenHost{HostID: h.HostID, DisplayLabel: strings.TrimSpace(h.DisplayLabel)})
			}
			v.Tokens = append(v.Tokens, StoredToken{
				TokenID:     d.TokenID,
				Name:        strings.TrimSpace(d.Name),
				CreatedAt:   d.CreatedAt,
				UpdatedAt:   d.UpdatedAt,
				RevokedAt:   d.RevokedAt,
				DeletedAt:   d.DeletedAt,
				ExpiresAt:   d.ExpiresAt,
				MaxUses:     d.MaxUses,
				SyncEnabled: true,
				Hosts:       hosts,
			})
			changed = true
			continue
		}
		local := &v.Tokens[idx]
		if local.UpdatedAt.After(d.UpdatedAt) {
			continue
		}
		local.Name = strings.TrimSpace(d.Name)
		local.UpdatedAt = d.UpdatedAt
		local.CreatedAt = d.CreatedAt
		local.ExpiresAt = d.ExpiresAt
		local.MaxUses = d.MaxUses
		local.SyncEnabled = true
		local.RevokedAt = d.RevokedAt
		local.DeletedAt = d.DeletedAt
		local.Hosts = make([]StoredTokenHost, 0, len(d.Hosts))
		for _, h := range d.Hosts {
			local.Hosts = append(local.Hosts, StoredTokenHost{HostID: h.HostID, DisplayLabel: strings.TrimSpace(h.DisplayLabel)})
		}
		changed = true
	}
	sort.SliceStable(v.Tokens, func(i, j int) bool {
		return v.Tokens[i].CreatedAt.After(v.Tokens[j].CreatedAt)
	})
	return changed
}
