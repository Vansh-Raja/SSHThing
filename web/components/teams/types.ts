export type TeamRole = "owner" | "admin" | "member";

export type DashboardTab = "members" | "hosts" | "invites" | "tokens" | "audit";

export type TeamSummary = {
  id: string;
  name: string;
  slug: string;
  displayOrder: number;
  role: TeamRole;
};

export type TeamMember = {
  id: string;
  teamId: string;
  clerkUserId: string;
  email: string;
  displayName: string;
  role: TeamRole;
  status: string;
  joinedAt: number | null;
};

export type TeamInvite = {
  id: string;
  teamId: string;
  teamName: string;
  teamSlug: string;
  email: string;
  role: TeamRole;
  status: string;
  expiresAt: number;
  createdAt: number;
  shareUrl?: string | null;
};

export type InviteResponse = {
  incoming: TeamInvite[];
  sent: TeamInvite[];
};

export type TeamHost = {
  id: string;
  teamId: string;
  label: string;
  hostname: string;
  username: string;
  port: number;
  group: string;
  tags: string[];
  notes: string;
  authMode?: string;
  credentialMode: "shared" | "per_member";
  credentialType: "none" | "password" | "private_key";
  secretVisibility: string;
  lastConnectedAt: number | null;
  createdAt: number;
  updatedAt: number;
  canManageHosts?: boolean;
  canRevealSecrets?: boolean;
  canEditNotes?: boolean;
};

export type TeamHostDetail = TeamHost & {
  sharedCredential: string | null;
  sharedCredentialConfigured?: boolean;
};

export type PersonalCredential = {
  hostId: string;
  credentialMode: "shared" | "per_member";
  credentialType: "none" | "password" | "private_key";
  username: string | null;
  hasCredential: boolean;
  secret: string;
  updatedAt?: number | null;
  viewerCanEdit?: boolean;
};

export type CredentialRosterEntry = {
  memberId: string;
  displayName: string;
  email: string;
  role: TeamRole;
  isOwner: boolean;
  isCurrentUser: boolean;
  hasCredential: boolean;
  credentialType: "none" | "password" | "private_key";
  username: string | null;
  updatedAt: number | null;
  /** True when the member has no personal credential but the host stores a
   * shared credential that acts as the fallback on connect. */
  usingSharedFallback?: boolean;
};

export type RevealedCredential = {
  hostId: string;
  memberClerkUserId?: string;
  credentialType: "none" | "password" | "private_key";
  username?: string | null;
  secret: string;
  updatedAt?: number | null;
};

export type TeamAuditEvent = {
  id: string;
  teamId: string;
  actorClerkUserId: string;
  actorDisplayName: string;
  entityType: string;
  entityId: string;
  eventType: string;
  targetClerkUserId?: string | null;
  targetDisplayName?: string | null;
  summary: string;
  metadata?: {
    hostLabel?: string;
    credentialMode?: string;
    credentialType?: string;
  } | null;
  createdAt: number;
};

export type TeamAutomationToken = {
  id: string;
  teamId: string;
  tokenId: string;
  name: string;
  status: string;
  hostCount: number;
  hosts?: Array<{ hostId: string; hostLabel: string }>;
  createdByClerkUserId?: string;
  createdByDisplayName?: string;
  createdAt: number;
  updatedAt: number;
  lastUsedAt?: number | null;
  useCount: number;
  expiresAt?: number | null;
  maxUses?: number | null;
  revokedAt?: number | null;
};

export type HostFormState = {
  label: string;
  hostname: string;
  username: string;
  port: string;
  group: string;
  tags: string;
  notes: string;
  credentialMode: "shared" | "per_member";
  credentialType: "none" | "password" | "private_key";
  sharedCredential: string;
};

export type PersonalCredentialFormState = {
  username: string;
  credentialType: "password" | "private_key";
  secret: string;
};
