package service

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/Vansh-Raja/SSHThing/internal/db"
	"github.com/Vansh-Raja/SSHThing/internal/ssh"
)

// Hosts provides host-listing operations on top of an unlocked Vault.
type Hosts struct {
	Vault   *Vault
	SyncSvc *SyncService // optional; if set and AutoSyncAfterCRUD is enabled, triggers sync after mutations
}

// triggerAutoSync fires a background sync if AutoSyncAfterCRUD is configured.
// It never blocks the calling goroutine and never returns an error.
func (h *Hosts) triggerAutoSync() {
	if h.SyncSvc == nil {
		return
	}
	cfg := h.SyncSvc.CfgStore.Get()
	if !cfg.Sync.AutoSyncAfterCRUD || !cfg.Sync.Enabled {
		return
	}
	go func() {
		ctx := context.Background()
		if _, err := h.SyncSvc.Now(ctx); err != nil {
			log.Printf("hosts: auto-sync after CRUD failed: %v", err)
		}
	}()
}

// List returns all hosts, optionally filtered by query (case-insensitive substring match
// across label, hostname, username, group, and tags).
func (h *Hosts) List(query string) ([]HostSummary, error) {
	store := h.Vault.Store()
	if store == nil {
		return nil, ErrVaultLocked
	}

	models, err := store.GetHosts()
	if err != nil {
		return nil, fmt.Errorf("get hosts: %w", err)
	}

	summaries := make([]HostSummary, 0, len(models))
	for _, m := range models {
		summaries = append(summaries, hostSummaryFromModel(m))
	}

	if q := strings.TrimSpace(query); q != "" {
		summaries = Filter(summaries, q)
	}

	return summaries, nil
}

// Get returns a single host by stringified int ID.
func (h *Hosts) Get(id string) (*HostSummary, error) {
	store := h.Vault.Store()
	if store == nil {
		return nil, ErrVaultLocked
	}

	intID, err := strconv.Atoi(id)
	if err != nil {
		return nil, fmt.Errorf("invalid host id %q: %w", id, err)
	}

	model, err := store.GetHostByID(intID)
	if err == sql.ErrNoRows {
		return nil, nil // caller checks nil → host_not_found
	}
	if err != nil {
		return nil, fmt.Errorf("get host: %w", err)
	}

	s := hostSummaryFromModel(*model)
	return &s, nil
}

// CreateParams holds the fields for creating a new host.
type CreateParams struct {
	Label     string   `json:"label"`
	GroupName string   `json:"group"`
	Tags      []string `json:"tags"`
	Hostname  string   `json:"hostname"`
	Username  string   `json:"username"`
	Port      int      `json:"port"`
	KeyType   string   `json:"keyType"` // "password" | "pasted" | "ed25519" | "rsa" | "ecdsa"
	PlainKey  string   `json:"plainKey"`
}

// Create adds a new host. Mirrors: internal/app/handlers.go submitAndClose → store.CreateHost.
func (h *Hosts) Create(p CreateParams) (*HostSummary, error) {
	store := h.Vault.Store()
	if store == nil {
		return nil, ErrVaultLocked
	}
	model := &db.HostModel{
		Label:     strings.TrimSpace(p.Label),
		GroupName: p.GroupName,
		Tags:      p.Tags,
		Hostname:  p.Hostname,
		Username:  p.Username,
		Port:      p.Port,
		KeyType:   p.KeyType,
	}
	if err := store.CreateHost(model, p.PlainKey); err != nil {
		return nil, fmt.Errorf("create host: %w", err)
	}
	// Re-fetch to get the assigned ID.
	hosts, err := store.GetHosts()
	if err != nil {
		return nil, fmt.Errorf("reload hosts: %w", err)
	}
	// The new host is the one with the matching hostname+username+label — grab last match.
	var created *HostSummary
	for _, m := range hosts {
		if m.Hostname == p.Hostname && m.Username == p.Username &&
			strings.TrimSpace(m.Label) == model.Label {
			s := hostSummaryFromModel(m)
			created = &s
		}
	}
	h.triggerAutoSync()
	return created, nil
}

// UpdateParams holds the fields for updating host metadata (no key change).
type UpdateParams struct {
	ID        string   `json:"id"`
	Label     string   `json:"label"`
	GroupName string   `json:"group"`
	Tags      []string `json:"tags"`
	Hostname  string   `json:"hostname"`
	Username  string   `json:"username"`
	Port      int      `json:"port"`
	KeyType   string   `json:"keyType"`
}

// Update updates a host's metadata without touching the stored credential.
// Mirrors: internal/app/handlers.go submitAndClose → store.UpdateHost.
func (h *Hosts) Update(p UpdateParams) error {
	store := h.Vault.Store()
	if store == nil {
		return ErrVaultLocked
	}
	intID, err := strconv.Atoi(p.ID)
	if err != nil {
		return fmt.Errorf("invalid host id %q: %w", p.ID, err)
	}
	model := &db.HostModel{
		ID:        intID,
		Label:     strings.TrimSpace(p.Label),
		GroupName: p.GroupName,
		Tags:      p.Tags,
		Hostname:  p.Hostname,
		Username:  p.Username,
		Port:      p.Port,
		KeyType:   p.KeyType,
	}
	if err := store.UpdateHost(model); err != nil {
		return err
	}
	h.triggerAutoSync()
	return nil
}

// UpdateWithKey updates a host's metadata and replaces the stored credential.
// Mirrors: internal/app/handlers.go submitAndClose → store.UpdateHostWithKey.
func (h *Hosts) UpdateWithKey(p UpdateParams, plainKey string) error {
	store := h.Vault.Store()
	if store == nil {
		return ErrVaultLocked
	}
	intID, err := strconv.Atoi(p.ID)
	if err != nil {
		return fmt.Errorf("invalid host id %q: %w", p.ID, err)
	}
	model := &db.HostModel{
		ID:        intID,
		Label:     strings.TrimSpace(p.Label),
		GroupName: p.GroupName,
		Tags:      p.Tags,
		Hostname:  p.Hostname,
		Username:  p.Username,
		Port:      p.Port,
		KeyType:   p.KeyType,
	}
	if err := store.UpdateHostWithKey(model, plainKey); err != nil {
		return err
	}
	h.triggerAutoSync()
	return nil
}

// Delete removes a host by stringified int ID.
// Mirrors: internal/app/handlers.go handleDeleteHostKeys → store.DeleteHost.
func (h *Hosts) Delete(id string) error {
	store := h.Vault.Store()
	if store == nil {
		return ErrVaultLocked
	}
	intID, err := strconv.Atoi(id)
	if err != nil {
		return fmt.Errorf("invalid host id %q: %w", id, err)
	}
	if err := store.DeleteHost(intID); err != nil {
		return err
	}
	h.triggerAutoSync()
	return nil
}

// RevealCredential decrypts and returns the stored credential for the given host ID.
// Audit-logs the access (connID is the RPC connection that requested it).
// Mirrors: internal/app/backend.go connectToHost → store.GetHostSecret.
func (h *Hosts) RevealCredential(id string, connID uint64) (string, error) {
	store := h.Vault.Store()
	if store == nil {
		return "", ErrVaultLocked
	}
	intID, err := strconv.Atoi(id)
	if err != nil {
		return "", fmt.Errorf("invalid host id %q: %w", id, err)
	}
	secret, err := store.GetHostSecret(intID)
	if err != nil {
		return "", fmt.Errorf("get secret: %w", err)
	}
	// Audit log — never log the secret value itself.
	// TODO: persist to audit log when audit table lands.
	log.Printf("audit: hosts.revealCredential host=%d connID=%d", intID, connID)
	return secret, nil
}

// ImportParams holds the fields for importing a host from a key file / PEM blob.
type ImportParams struct {
	Label    string   `json:"label"`
	Hostname string   `json:"hostname"`
	Username string   `json:"username"`
	Port     int      `json:"port"`
	Group    string   `json:"group"`
	Tags     []string `json:"tags"`
	KeyType  string   `json:"keyType"` // "pasted" or empty
	KeyBlob  string   `json:"keyBlob"` // raw PEM / OpenSSH key text
}

// Import validates a private key blob and creates a host with it.
// Mirrors: internal/app/handlers.go normalizePrivateKey + ssh.ValidatePrivateKey + store.CreateHost.
func (h *Hosts) Import(p ImportParams) (*HostSummary, error) {
	if strings.TrimSpace(p.KeyBlob) == "" {
		return nil, fmt.Errorf("keyBlob is required for import")
	}
	if err := ssh.ValidatePrivateKey(p.KeyBlob); err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}
	keyType := strings.TrimSpace(p.KeyType)
	if keyType == "" {
		keyType = "pasted"
	}
	return h.Create(CreateParams{
		Label:     p.Label,
		GroupName: p.Group,
		Tags:      p.Tags,
		Hostname:  p.Hostname,
		Username:  p.Username,
		Port:      p.Port,
		KeyType:   keyType,
		PlainKey:  p.KeyBlob,
	})
}

// GenerateKeyResult holds the result of key generation.
type GenerateKeyResult struct {
	PrivateKey string `json:"privateKey"`
	PublicKey  string `json:"publicKey"`
	KeyType    string `json:"keyType"`
	Comment    string `json:"comment"`
}

// GenerateKey wraps ssh.GenerateKey.
// Mirrors: internal/app/handlers.go submitAndClose case formAuthIdx==2 → ssh.GenerateKey.
func (h *Hosts) GenerateKey(keyType, comment string) (*GenerateKeyResult, error) {
	privKey, pubKey, err := ssh.GenerateKey(ssh.KeyType(keyType), comment)
	if err != nil {
		return nil, fmt.Errorf("generate key: %w", err)
	}
	return &GenerateKeyResult{
		PrivateKey: privKey,
		PublicKey:  pubKey,
		KeyType:    keyType,
		Comment:    comment,
	}, nil
}

// Filter performs a simple case-insensitive substring search over HostSummary fields.
// Equivalent to the TUI's buildSearchResults helper (internal/app/).
func Filter(hosts []HostSummary, query string) []HostSummary {
	q := strings.ToLower(query)
	var out []HostSummary
	for _, h := range hosts {
		if matchesQuery(h, q) {
			out = append(out, h)
		}
	}
	return out
}

func matchesQuery(h HostSummary, q string) bool {
	fields := []string{
		strings.ToLower(h.Label),
		strings.ToLower(h.Hostname),
		strings.ToLower(h.Username),
		strings.ToLower(h.Group),
	}
	for _, f := range fields {
		if strings.Contains(f, q) {
			return true
		}
	}
	for _, tag := range h.Tags {
		if strings.Contains(strings.ToLower(tag), q) {
			return true
		}
	}
	return false
}
