package teams

import "time"

type Role string

const (
	RoleOwner            Role = "owner"
	RoleAdmin            Role = "admin"
	RoleMember           Role = "member"
	RoleRestrictedMember Role = "restricted_member"
)

type ShareMode string

const (
	ShareModeHostOnly                 ShareMode = "host_only"
	ShareModeHostPlusSharedCredential ShareMode = "host_plus_shared_credential"
)

type MemberStatus string

const (
	MemberStatusActive  MemberStatus = "active"
	MemberStatusInvited MemberStatus = "invited"
	MemberStatusRemoved MemberStatus = "removed"
)

type TeamSummary struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Slug          string    `json:"slug"`
	Description   string    `json:"description,omitempty"`
	MemberCount   int       `json:"memberCount"`
	HostCount     int       `json:"hostCount"`
	BillingStatus string    `json:"billingStatus,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
}

type Host struct {
	ID                  string     `json:"id"`
	TeamID              string     `json:"teamId"`
	Label               string     `json:"label"`
	Group               string     `json:"group,omitempty"`
	Tags                []string   `json:"tags,omitempty"`
	Hostname            string     `json:"hostname"`
	Username            string     `json:"username"`
	Port                int        `json:"port"`
	ShareMode           ShareMode  `json:"shareMode"`
	Notes               []string   `json:"notes,omitempty"`
	LastActivityAt      *time.Time `json:"lastActivityAt,omitempty"`
	RotationRecommended bool       `json:"rotationRecommended,omitempty"`
	RotationReason      string     `json:"rotationReason,omitempty"`
	CreatedAt           time.Time  `json:"createdAt"`
	UpdatedAt           time.Time  `json:"updatedAt"`
}

type Member struct {
	ID          string       `json:"id"`
	TeamID      string       `json:"teamId"`
	UserID      string       `json:"userId"`
	Email       string       `json:"email"`
	DisplayName string       `json:"displayName"`
	Role        Role         `json:"role"`
	Status      MemberStatus `json:"status"`
	JoinedAt    *time.Time   `json:"joinedAt,omitempty"`
	LastSeenAt  *time.Time   `json:"lastSeenAt,omitempty"`
}

type Invite struct {
	ID        string    `json:"id"`
	TeamID    string    `json:"teamId"`
	Email     string    `json:"email"`
	Role      Role      `json:"role"`
	InvitedBy string    `json:"invitedBy"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

type AuthState struct {
	Authenticated bool         `json:"authenticated"`
	HasTeam       bool         `json:"hasTeam"`
	UserID        string       `json:"userId,omitempty"`
	UserEmail     string       `json:"userEmail,omitempty"`
	ActiveTeam    *TeamSummary `json:"activeTeam,omitempty"`
}

type MembershipState struct {
	Team    *TeamSummary `json:"team,omitempty"`
	Members []Member     `json:"members,omitempty"`
	Hosts   []Host       `json:"hosts,omitempty"`
}

type MeResponse struct {
	Auth AuthState `json:"auth"`
}

type ConnectPayload struct {
	GrantID             string    `json:"grantId,omitempty"`
	TeamHostID          string    `json:"teamHostId"`
	Hostname            string    `json:"hostname"`
	Username            string    `json:"username"`
	Port                int       `json:"port"`
	KeyType             string    `json:"keyType"`
	Secret              string    `json:"secret"`
	HostKeyPolicy       string    `json:"hostKeyPolicy,omitempty"`
	KeepAliveSeconds    int       `json:"keepAliveSeconds,omitempty"`
	PasswordBackendUnix string    `json:"passwordBackendUnix,omitempty"`
	ExpiresAt           time.Time `json:"expiresAt"`
}
