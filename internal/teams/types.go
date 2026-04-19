package teams

type VisibilityMode string

const (
	VisibilityFull     VisibilityMode = "full"
	VisibilityMasked   VisibilityMode = "masked"
	VisibilityReadOnly VisibilityMode = "read_only"
)

type ShareMode string

const (
	ShareModeHostOnly     ShareMode = "host_only"
	ShareModeSharedSecret ShareMode = "shared_secret"
	ShareModeMetadataOnly ShareMode = "metadata_only"
	ShareModeUnavailable  ShareMode = "unavailable"
)

type WorkspaceRole string

const (
	WorkspaceRoleOwner WorkspaceRole = "owner"
	WorkspaceRoleAdmin WorkspaceRole = "admin"
)

type VaultRole string

const (
	VaultRoleAdmin              VaultRole = "vault_admin"
	VaultRoleEditor             VaultRole = "editor"
	VaultRoleOperator           VaultRole = "operator"
	VaultRoleRestrictedOperator VaultRole = "restricted_operator"
	VaultRoleViewer             VaultRole = "viewer"
)

type UserSummary struct {
	ID    string `json:"id"`
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
}

type TeamSummary struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Slug         string `json:"slug"`
	DisplayOrder int    `json:"displayOrder"`
}

type TeamHost struct {
	ID              string   `json:"id"`
	TeamID          string   `json:"teamId"`
	Label           string   `json:"label"`
	Hostname        string   `json:"hostname"`
	Username        string   `json:"username"`
	Port            int      `json:"port"`
	Group           string   `json:"group,omitempty"`
	Tags            []string `json:"tags,omitempty"`
	AuthMode        string   `json:"authMode,omitempty"`
	LastConnectedAt *int64   `json:"lastConnectedAt,omitempty"`
}

type WorkspaceSummary struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	Slug                string `json:"slug"`
	ClerkOrganizationID string `json:"clerkOrganizationId,omitempty"`
}

type Vault struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspaceId"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description,omitempty"`
}

type Resource struct {
	ID             string         `json:"id"`
	VaultID        string         `json:"vaultId"`
	Label          string         `json:"label"`
	Group          string         `json:"group,omitempty"`
	Tags           []string       `json:"tags,omitempty"`
	Hostname       string         `json:"hostname,omitempty"`
	Username       string         `json:"username,omitempty"`
	Port           int            `json:"port,omitempty"`
	ShareMode      ShareMode      `json:"shareMode"`
	VisibilityMode VisibilityMode `json:"visibilityMode,omitempty"`
	Notes          string         `json:"notes,omitempty"`
	CreatedBy      string         `json:"createdBy,omitempty"`
	UpdatedBy      string         `json:"updatedBy,omitempty"`
}

type Member struct {
	ID            string        `json:"id"`
	WorkspaceID   string        `json:"workspaceId,omitempty"`
	VaultID       string        `json:"vaultId,omitempty"`
	ClerkUserID   string        `json:"clerkUserId,omitempty"`
	Email         string        `json:"email,omitempty"`
	DisplayName   string        `json:"displayName,omitempty"`
	WorkspaceRole WorkspaceRole `json:"workspaceRole,omitempty"`
	VaultRole     VaultRole     `json:"vaultRole,omitempty"`
	Status        string        `json:"status,omitempty"`
}

type AuthState struct {
	Authenticated   bool              `json:"authenticated"`
	HasWorkspace    bool              `json:"hasWorkspace"`
	UserID          string            `json:"userId,omitempty"`
	ActiveWorkspace *WorkspaceSummary `json:"activeWorkspace,omitempty"`
}

type MeResponse struct {
	Auth AuthState `json:"auth"`
}

type CliAuthStartResponse struct {
	AuthURL             string `json:"authUrl"`
	DeviceCode          string `json:"deviceCode"`
	SessionID           string `json:"sessionId"`
	PollSecret          string `json:"pollSecret"`
	PollIntervalSeconds int    `json:"pollIntervalSeconds"`
	ExpiresAt           int64  `json:"expiresAt"`
}

type CliAuthPollResponse struct {
	Status       string            `json:"status"`
	ExpiresAt    int64             `json:"expiresAt,omitempty"`
	AccessToken  string            `json:"accessToken,omitempty"`
	RefreshToken string            `json:"refreshToken,omitempty"`
	Workspace    *WorkspaceSummary `json:"workspace,omitempty"`
	User         *UserSummary      `json:"user,omitempty"`
}

type RefreshResponse struct {
	AccessToken string `json:"accessToken"`
	ExpiresAt   int64  `json:"expiresAt"`
}

type ConnectResponse struct {
	Resource Resource `json:"resource"`
	Message  string   `json:"message,omitempty"`
}

type InviteRequest struct {
	Email         string    `json:"email"`
	WorkspaceRole string    `json:"workspaceRole,omitempty"`
	VaultID       string    `json:"vaultId"`
	VaultRole     VaultRole `json:"vaultRole"`
}

type UpdateMemberRoleRequest struct {
	VaultID   string    `json:"vaultId"`
	VaultRole VaultRole `json:"vaultRole"`
}
