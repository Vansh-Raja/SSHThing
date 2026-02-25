package authtoken

import "time"

const (
	TokenPrefix  = "stk"
	vaultVersion = 2
)

type Vault struct {
	Version int           `json:"version"`
	Tokens  []StoredToken `json:"tokens"`
}

type StoredToken struct {
	TokenID string `json:"token_id"`
	Name    string `json:"name"`

	Salt string `json:"salt,omitempty"`
	Hash string `json:"hash,omitempty"`

	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	UseCount   int        `json:"use_count"`

	RevokedAt *time.Time `json:"revoked_at,omitempty"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`

	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	MaxUses   int        `json:"max_uses,omitempty"`

	SyncEnabled bool `json:"sync_enabled,omitempty"`

	UnlockSalt  string `json:"unlock_salt,omitempty"`
	UnlockData  string `json:"unlock_data,omitempty"`
	UnlockBound bool   `json:"unlock_bound,omitempty"`

	Hosts []StoredTokenHost `json:"hosts"`
}

type StoredTokenHost struct {
	HostID       int    `json:"host_id"`
	DisplayLabel string `json:"display_label"`

	// Legacy v1 fields retained for backward compatibility.
	PayloadSalt string `json:"payload_salt,omitempty"`
	Payload     string `json:"payload,omitempty"`
}

type HostGrant struct {
	HostID       int
	DisplayLabel string
}

type CreateOptions struct {
	DevicePepper []byte
	BindToDevice bool
	ExpiresAt    *time.Time
	MaxUses      int
	SyncEnabled  bool
}

type ExecPayload struct {
	HostID              int    `json:"host_id"`
	DisplayLabel        string `json:"display_label"`
	Hostname            string `json:"hostname"`
	Username            string `json:"username"`
	Port                int    `json:"port"`
	KeyType             string `json:"key_type"`
	Secret              string `json:"secret"`
	HostKeyPolicy       string `json:"host_key_policy"`
	KeepAliveSeconds    int    `json:"keepalive_seconds"`
	Term                string `json:"term"`
	PasswordBackendUnix string `json:"password_backend_unix"`
}

type ResolveResult struct {
	TokenIndex int
	TokenID    string
	HostID     int
	HostLabel  string

	DBUnlockSecret string
	LegacyPayload  *ExecPayload
}

type TokenSummary struct {
	TokenID    string
	Name       string
	HostCount  int
	CreatedAt  time.Time
	UpdatedAt  time.Time
	LastUsedAt *time.Time
	UseCount   int
	RevokedAt  *time.Time
	DeletedAt  *time.Time
	ExpiresAt  *time.Time
	MaxUses    int

	SyncEnabled bool
	Usable      bool
	Legacy      bool
}

type SyncTokenDef struct {
	TokenID string `json:"token_id"`
	Name    string `json:"name"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	RevokedAt *time.Time `json:"revoked_at,omitempty"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	MaxUses   int        `json:"max_uses,omitempty"`

	SyncEnabled bool            `json:"sync_enabled,omitempty"`
	Hosts       []SyncTokenHost `json:"hosts"`
}

type SyncTokenHost struct {
	HostID       int    `json:"host_id"`
	DisplayLabel string `json:"display_label"`
}
