package teams

type TeamRole string

const (
	TeamRoleOwner  TeamRole = "owner"
	TeamRoleAdmin  TeamRole = "admin"
	TeamRoleMember TeamRole = "member"
)

type UserSummary struct {
	ID    string `json:"id"`
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
}

type TeamSummary struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Slug         string   `json:"slug"`
	DisplayOrder int      `json:"displayOrder"`
	Role         TeamRole `json:"role,omitempty"`
}

type TeamMember struct {
	ID          string   `json:"id"`
	TeamID      string   `json:"teamId"`
	ClerkUserID string   `json:"clerkUserId"`
	Email       string   `json:"email"`
	DisplayName string   `json:"displayName"`
	Role        TeamRole `json:"role"`
	Status      string   `json:"status"`
	JoinedAt    *int64   `json:"joinedAt,omitempty"`
}

type TeamInvite struct {
	ID        string   `json:"id"`
	TeamID    string   `json:"teamId"`
	TeamName  string   `json:"teamName"`
	TeamSlug  string   `json:"teamSlug"`
	Email     string   `json:"email"`
	Role      TeamRole `json:"role"`
	Status    string   `json:"status"`
	ExpiresAt int64    `json:"expiresAt"`
	CreatedAt int64    `json:"createdAt"`
	ShareURL  string   `json:"shareUrl,omitempty"`
}

type TeamInviteList struct {
	Incoming []TeamInvite `json:"incoming"`
	Sent     []TeamInvite `json:"sent"`
}

type TeamHost struct {
	ID               string   `json:"id"`
	TeamID           string   `json:"teamId"`
	Label            string   `json:"label"`
	Hostname         string   `json:"hostname"`
	Username         string   `json:"username"`
	Port             int      `json:"port"`
	Group            string   `json:"group,omitempty"`
	Tags             []string `json:"tags,omitempty"`
	Notes            string   `json:"notes,omitempty"`
	AuthMode         string   `json:"authMode,omitempty"`
	CredentialMode   string   `json:"credentialMode,omitempty"`
	CredentialType   string   `json:"credentialType,omitempty"`
	SecretVisibility string   `json:"secretVisibility,omitempty"`
	LastConnectedAt  *int64   `json:"lastConnectedAt,omitempty"`
	CreatedAt        int64    `json:"createdAt,omitempty"`
	UpdatedAt        int64    `json:"updatedAt,omitempty"`
	CanManageHosts   bool     `json:"canManageHosts,omitempty"`
	CanRevealSecrets bool     `json:"canRevealSecrets,omitempty"`
	CanEditNotes     bool     `json:"canEditNotes,omitempty"`
}

type TeamHostDetail struct {
	TeamHost
	SharedCredential           string `json:"sharedCredential,omitempty"`
	SharedCredentialConfigured bool   `json:"sharedCredentialConfigured,omitempty"`
}

type TeamHostConnectConfig struct {
	HostID         string `json:"hostId"`
	TeamID         string `json:"teamId"`
	Label          string `json:"label"`
	Hostname       string `json:"hostname"`
	Username       string `json:"username"`
	Port           int    `json:"port"`
	CredentialMode string `json:"credentialMode"`
	CredentialType string `json:"credentialType"`
	Secret         string `json:"secret"`
}

type CreateTeamHostRequest struct {
	Label            string   `json:"label"`
	Hostname         string   `json:"hostname"`
	Username         string   `json:"username"`
	Port             int      `json:"port"`
	Group            string   `json:"group,omitempty"`
	Tags             []string `json:"tags,omitempty"`
	Notes            string   `json:"notes,omitempty"`
	CredentialMode   string   `json:"credentialMode"`
	CredentialType   string   `json:"credentialType"`
	SecretVisibility string   `json:"secretVisibility"`
	SharedCredential string   `json:"sharedCredential,omitempty"`
}

type UpdateTeamHostRequest struct {
	Label                 string   `json:"label"`
	Hostname              string   `json:"hostname"`
	Username              string   `json:"username"`
	Port                  int      `json:"port"`
	Group                 string   `json:"group,omitempty"`
	Tags                  []string `json:"tags,omitempty"`
	Notes                 string   `json:"notes,omitempty"`
	CredentialMode        string   `json:"credentialMode"`
	CredentialType        string   `json:"credentialType"`
	SecretVisibility      string   `json:"secretVisibility"`
	SharedCredential      string   `json:"sharedCredential,omitempty"`
	ClearSharedCredential bool     `json:"clearSharedCredential,omitempty"`
}

type TeamHostCredential struct {
	HostID         string `json:"hostId"`
	CredentialMode string `json:"credentialMode"`
	CredentialType string `json:"credentialType"`
	Username       string `json:"username,omitempty"`
	HasCredential  bool   `json:"hasCredential"`
	Secret         string `json:"secret,omitempty"`
	UpdatedAt      *int64 `json:"updatedAt,omitempty"`
	ViewerCanEdit  bool   `json:"viewerCanEdit,omitempty"`
}

type TeamHostCredentialRosterEntry struct {
	MemberID       string   `json:"memberId"`
	DisplayName    string   `json:"displayName"`
	Email          string   `json:"email"`
	Role           TeamRole `json:"role"`
	IsOwner        bool     `json:"isOwner"`
	IsCurrentUser  bool     `json:"isCurrentUser"`
	HasCredential  bool     `json:"hasCredential"`
	CredentialType string   `json:"credentialType"`
	Username       string   `json:"username,omitempty"`
	UpdatedAt      *int64   `json:"updatedAt,omitempty"`
}

type RevealedTeamHostCredential struct {
	HostID            string `json:"hostId"`
	MemberClerkUserID string `json:"memberClerkUserId,omitempty"`
	CredentialType    string `json:"credentialType"`
	Username          string `json:"username,omitempty"`
	Secret            string `json:"secret,omitempty"`
	UpdatedAt         *int64 `json:"updatedAt,omitempty"`
}

type TeamAuditEvent struct {
	ID                string `json:"id"`
	TeamID            string `json:"teamId"`
	ActorClerkUserID  string `json:"actorClerkUserId"`
	ActorDisplayName  string `json:"actorDisplayName"`
	EntityType        string `json:"entityType"`
	EntityID          string `json:"entityId"`
	EventType         string `json:"eventType"`
	TargetClerkUserID string `json:"targetClerkUserId,omitempty"`
	TargetDisplayName string `json:"targetDisplayName,omitempty"`
	Summary           string `json:"summary"`
	CreatedAt         int64  `json:"createdAt"`
}

type AuthState struct {
	Authenticated bool   `json:"authenticated"`
	UserID        string `json:"userId,omitempty"`
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
	Status       string       `json:"status"`
	ExpiresAt    int64        `json:"expiresAt,omitempty"`
	AccessToken  string       `json:"accessToken,omitempty"`
	RefreshToken string       `json:"refreshToken,omitempty"`
	User         *UserSummary `json:"user,omitempty"`
}

type RefreshResponse struct {
	AccessToken string `json:"accessToken"`
	ExpiresAt   int64  `json:"expiresAt"`
}
