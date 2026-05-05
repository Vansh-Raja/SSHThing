package sync

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Vansh-Raja/SSHThing/internal/authtoken"
	"github.com/Vansh-Raja/SSHThing/internal/crypto"
	"github.com/Vansh-Raja/SSHThing/internal/personalsync"
)

type Provider interface {
	Init(ctx context.Context) error
	Pull(ctx context.Context, password string) (*SyncData, *RemoteState, error)
	Push(ctx context.Context, data *SyncData, password string, state *RemoteState) error
	Name() string
}

type RemoteState struct {
	Revision       string
	LatestRevision int64
	UpdatedAt      time.Time
	ItemRevisions  map[string]string
}

type GitProvider struct {
	git *GitManager
}

func NewGitProvider(git *GitManager) *GitProvider {
	return &GitProvider{git: git}
}

func (p *GitProvider) Name() string { return "git" }

func (p *GitProvider) Init(ctx context.Context) error {
	if p.git == nil {
		return fmt.Errorf("git manager not initialized")
	}
	return p.git.Init()
}

func (p *GitProvider) Pull(ctx context.Context, password string) (*SyncData, *RemoteState, error) {
	if p.git == nil {
		return nil, nil, fmt.Errorf("git manager not initialized")
	}
	if p.git.HasRemote() {
		if err := p.git.Pull(); err != nil {
			return nil, nil, err
		}
	}
	data, err := LoadFromFile(p.git.GetSyncFilePath(), password)
	if err != nil {
		return nil, nil, err
	}
	return data, &RemoteState{}, nil
}

func (p *GitProvider) Push(ctx context.Context, data *SyncData, password string, state *RemoteState) error {
	if p.git == nil {
		return fmt.Errorf("git manager not initialized")
	}
	if err := ExportDataToFile(data, p.git.GetSyncFilePath(), password); err != nil {
		return err
	}
	commitMsg := fmt.Sprintf("Sync: %s", time.Now().Format(time.RFC3339))
	if err := p.git.CommitChanges(commitMsg); err != nil {
		return err
	}
	if p.git.HasRemote() {
		return p.git.Push()
	}
	return nil
}

type PersonalCloudClient interface {
	GetPersonalVault(ctx context.Context, accessToken string) (personalsync.VaultSummary, error)
	ListPersonalVaultItems(ctx context.Context, accessToken, since string) (personalsync.ListItemsResponse, error)
	ListPersonalVaultChanges(ctx context.Context, accessToken, sinceRevision string) (personalsync.ChangesResponse, error)
	UpsertPersonalVaultItems(ctx context.Context, accessToken string, req personalsync.UpsertRequest) (personalsync.UpsertResponse, error)
	MarkPersonalVaultDeviceSeen(ctx context.Context, accessToken string, req personalsync.DeviceSeenRequest) error
	RecordPersonalSyncEvent(ctx context.Context, accessToken string, req personalsync.SyncEventRequest) error
}

type ConvexProvider struct {
	client              PersonalCloudClient
	accessTokenProvider func(context.Context) (string, error)
	deviceID            string
}

func NewConvexProvider(client PersonalCloudClient, accessTokenProvider func(context.Context) (string, error), deviceID string) *ConvexProvider {
	return &ConvexProvider{client: client, accessTokenProvider: accessTokenProvider, deviceID: strings.TrimSpace(deviceID)}
}

func (p *ConvexProvider) Name() string { return "convex" }

func (p *ConvexProvider) Init(ctx context.Context) error {
	if p.client == nil {
		return fmt.Errorf("personal cloud client is not configured")
	}
	if p.accessTokenProvider == nil {
		return fmt.Errorf("sign in from Profile to use SSHThing Cloud sync")
	}
	return nil
}

func (p *ConvexProvider) Pull(ctx context.Context, password string) (*SyncData, *RemoteState, error) {
	return p.PullChanges(ctx, password, "")
}

func (p *ConvexProvider) PullChanges(ctx context.Context, password, sinceRevision string) (*SyncData, *RemoteState, error) {
	accessToken, err := p.accessTokenProvider(ctx)
	if err != nil {
		return nil, nil, err
	}
	vault, err := p.client.GetPersonalVault(ctx, accessToken)
	if err != nil {
		return nil, nil, fmt.Errorf("personal cloud vault unavailable: %w", err)
	}
	var resp personalsync.ChangesResponse
	if strings.TrimSpace(sinceRevision) != "" {
		resp, err = p.client.ListPersonalVaultChanges(ctx, accessToken, sinceRevision)
	} else {
		resp, err = p.client.ListPersonalVaultItems(ctx, accessToken, "")
	}
	if err != nil {
		return nil, nil, fmt.Errorf("personal cloud items unavailable: %w", err)
	}
	data, err := decryptVaultItems(resp.Items, password, vault.KDF.Salt)
	if err != nil {
		return nil, nil, err
	}
	_ = p.client.RecordPersonalSyncEvent(ctx, accessToken, personalsync.SyncEventRequest{DeviceID: p.deviceID, Source: "tui", Action: "pull", ItemCount: len(resp.Items)})
	latest := resp.LatestRevision
	if latest == 0 {
		latest = vault.LatestRevision
	}
	_ = p.client.MarkPersonalVaultDeviceSeen(ctx, accessToken, personalsync.DeviceSeenRequest{DeviceID: p.deviceID, LastPulledRevision: latest})
	return data, &RemoteState{Revision: resp.Revision, LatestRevision: latest, UpdatedAt: time.UnixMilli(vault.UpdatedAt), ItemRevisions: itemRevisions(resp.Items)}, nil
}

func (p *ConvexProvider) Push(ctx context.Context, data *SyncData, password string, state *RemoteState) error {
	accessToken, err := p.accessTokenProvider(ctx)
	if err != nil {
		return err
	}
	vault, err := p.client.GetPersonalVault(ctx, accessToken)
	if err != nil {
		return fmt.Errorf("personal cloud vault unavailable: %w", err)
	}
	items, err := encryptVaultItems(data, password, vault.KDF.Salt)
	if err != nil {
		return err
	}
	baseRevision := ""
	if state != nil {
		baseRevision = state.Revision
		for i := range items {
			itemRevision := state.ItemRevisions[items[i].ItemType+":"+items[i].SyncID]
			if itemRevision == "" {
				itemRevision = baseRevision
			}
			items[i].BaseRevision = itemRevision
		}
	}
	resp, err := p.client.UpsertPersonalVaultItems(ctx, accessToken, personalsync.UpsertRequest{
		BaseRevision: baseRevision,
		DeviceID:     p.deviceID,
		Items:        items,
	})
	if err != nil {
		if !isUnsupportedBaseRevisionError(err) {
			return fmt.Errorf("personal cloud items save failed: %w", err)
		}
		compatItems := clearItemBaseRevisions(items)
		resp, err = p.client.UpsertPersonalVaultItems(ctx, accessToken, personalsync.UpsertRequest{
			BaseRevision: baseRevision,
			DeviceID:     p.deviceID,
			Items:        compatItems,
		})
		if err != nil {
			return fmt.Errorf("personal cloud items save failed: %w", err)
		}
	}
	if len(resp.Conflicts) > 0 {
		return fmt.Errorf("personal cloud sync conflict: pull and retry")
	}
	if state != nil {
		state.Revision = resp.Revision
		state.LatestRevision = resp.LatestRevision
		state.UpdatedAt = time.Now().UTC()
	}
	_ = p.client.RecordPersonalSyncEvent(ctx, accessToken, personalsync.SyncEventRequest{DeviceID: p.deviceID, Source: "tui", Action: "push", ItemCount: len(items)})
	_ = p.client.MarkPersonalVaultDeviceSeen(ctx, accessToken, personalsync.DeviceSeenRequest{DeviceID: p.deviceID, LastPushedRevision: resp.LatestRevision})
	return nil
}

func isUnsupportedBaseRevisionError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "baseRevision") &&
		(strings.Contains(msg, "extra field") || strings.Contains(msg, "not in the validator"))
}

func clearItemBaseRevisions(items []personalsync.VaultItem) []personalsync.VaultItem {
	out := make([]personalsync.VaultItem, len(items))
	copy(out, items)
	for i := range out {
		out[i].BaseRevision = ""
	}
	return out
}

type personalPlainItem struct {
	Type      string          `json:"type"`
	SyncID    string          `json:"syncId"`
	Data      json.RawMessage `json:"data"`
	UpdatedAt time.Time       `json:"updatedAt"`
	DeletedAt *time.Time      `json:"deletedAt,omitempty"`
}

func deriveVaultKey(password, saltHex string) ([]byte, error) {
	salt, err := hex.DecodeString(strings.TrimSpace(saltHex))
	if err != nil {
		return nil, fmt.Errorf("invalid vault salt: %w", err)
	}
	key, _, err := crypto.DeriveKey(password, salt)
	return key, err
}

func encryptVaultItems(data *SyncData, password, saltHex string) ([]personalsync.VaultItem, error) {
	if data == nil {
		return nil, nil
	}
	key, err := deriveVaultKey(password, saltHex)
	if err != nil {
		return nil, err
	}
	out := make([]personalsync.VaultItem, 0, len(data.Groups)+len(data.Hosts)+len(data.TokenDefs)+1)
	meta := map[string]any{"salt": data.Salt, "updated_at": data.UpdatedAt}
	metaItem, err := encryptOneVaultItem("meta", "sync-meta", data.UpdatedAt, nil, meta, key)
	if err != nil {
		return nil, err
	}
	out = append(out, metaItem)
	for _, group := range data.Groups {
		item, err := encryptOneVaultItem("group", group.SyncID, group.UpdatedAt, group.DeletedAt, group, key)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	for _, host := range data.Hosts {
		item, err := encryptOneVaultItem("host", host.SyncID, host.UpdatedAt, nil, host, key)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	for _, token := range data.TokenDefs {
		item, err := encryptOneVaultItem("token_def", token.TokenID, token.UpdatedAt, token.DeletedAt, token, key)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, nil
}

func encryptOneVaultItem(itemType, syncID string, updatedAt time.Time, deletedAt *time.Time, data any, key []byte) (personalsync.VaultItem, error) {
	payload, err := json.Marshal(data)
	if err != nil {
		return personalsync.VaultItem{}, err
	}
	ciphertext, err := crypto.Encrypt(payload, key)
	if err != nil {
		return personalsync.VaultItem{}, err
	}
	if updatedAt.IsZero() {
		updatedAt = time.Now()
	}
	var deletedAtMS *int64
	if deletedAt != nil {
		ms := deletedAt.UnixMilli()
		deletedAtMS = &ms
	}
	return personalsync.VaultItem{ItemType: itemType, SyncID: strings.TrimSpace(syncID), Ciphertext: ciphertext, Nonce: "", UpdatedAt: updatedAt.UnixMilli(), DeletedAt: deletedAtMS, SchemaVersion: CurrentSyncVersion}, nil
}

func itemRevisions(items []personalsync.VaultItem) map[string]string {
	out := make(map[string]string, len(items))
	for _, item := range items {
		rev := item.ItemRevision
		if rev == 0 {
			rev = item.UpdatedAt
		}
		if rev == 0 {
			continue
		}
		out[item.ItemType+":"+item.SyncID] = fmt.Sprintf("%d", rev)
	}
	return out
}

func decryptVaultItems(items []personalsync.VaultItem, password, saltHex string) (*SyncData, error) {
	key, err := deriveVaultKey(password, saltHex)
	if err != nil {
		return nil, err
	}
	out := &SyncData{Version: CurrentSyncVersion, UpdatedAt: time.Now(), Groups: []SyncGroup{}, Hosts: []SyncHost{}}
	for _, item := range items {
		deletedAt := int64ToTimePtr(item.DeletedAt)
		if deletedAt != nil {
			switch item.ItemType {
			case "group":
				out.Groups = append(out.Groups, SyncGroup{SyncID: item.SyncID, UpdatedAt: time.UnixMilli(item.UpdatedAt), DeletedAt: deletedAt})
			}
			continue
		}
		plain, err := crypto.Decrypt(item.Ciphertext, key)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt personal cloud item %s/%s: %w", item.ItemType, item.SyncID, err)
		}
		switch item.ItemType {
		case "meta":
			var meta struct {
				Salt      string    `json:"salt"`
				UpdatedAt time.Time `json:"updated_at"`
			}
			if err := json.Unmarshal(plain, &meta); err != nil {
				return nil, err
			}
			out.Salt = meta.Salt
			if !meta.UpdatedAt.IsZero() {
				out.UpdatedAt = meta.UpdatedAt
			}
		case "group":
			var group SyncGroup
			if err := json.Unmarshal(plain, &group); err != nil {
				return nil, err
			}
			if group.SyncID == "" {
				group.SyncID = item.SyncID
			}
			out.Groups = append(out.Groups, group)
		case "host":
			var host SyncHost
			if err := json.Unmarshal(plain, &host); err != nil {
				return nil, err
			}
			if host.SyncID == "" {
				host.SyncID = item.SyncID
			}
			out.Hosts = append(out.Hosts, host)
		case "token_def":
			var token authtoken.SyncTokenDef
			if err := json.Unmarshal(plain, &token); err != nil {
				return nil, err
			}
			out.TokenDefs = append(out.TokenDefs, token)
		}
	}
	return out, nil
}

func int64ToTimePtr(v *int64) *time.Time {
	if v == nil {
		return nil
	}
	t := time.UnixMilli(*v)
	return &t
}
