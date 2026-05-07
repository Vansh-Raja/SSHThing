/**
 * Ambient type declarations for the window.sshthing API exposed by the preload
 * via contextBridge. These must stay in sync with src/preload/index.ts.
 */

export {};

declare global {
  interface HostSummary {
    id: string;
    syncId: string;
    label: string;
    hostname: string;
    username: string;
    port: number;
    group: string;
    tags: string[];
    lastConnectedAt: string | null;
    authMode: 'key' | 'password' | 'none';
  }

  interface HostCreate {
    label?: string;
    hostname: string;
    username: string;
    port?: number;
    group?: string;
    tags?: string[];
    authMode: 'key' | 'password' | 'none';
    plainKey?: string;
    plainPassword?: string;
  }

  interface HostUpdate {
    id: string;
    label?: string;
    hostname?: string;
    username?: string;
    port?: number;
    group?: string;
    tags?: string[];
    authMode?: 'key' | 'password' | 'none';
  }

  interface GroupSummary {
    id: string;
    name: string;
  }

  interface GenerateKeyResult {
    publicKey: string;
    privateKey: string;
    keyType: string;
    comment: string;
  }

  interface AppSettings {
    theme: 'light' | 'dark' | 'system';
    fontSize: number;
    termType: string;
    keepAliveSeconds: number;
    hostKeyPolicy: string;
    passwordBackend: string;
    syncProvider: 'off' | 'git' | 'cloud';
    releaseChannel?: 'stable' | 'beta';
    autoApplyUpdates?: boolean;
  }

  interface SessionInfo {
    sessionId: string;
    hostId: string;
    openedAt: string;
    label: string;
  }

  interface UnlockResult {
    unlocked: boolean;
    salt: string;
    sessionTtlSec: number;
  }

  interface VaultStatus {
    unlocked: boolean;
    expiresAt: number | null;
  }

  // ---- Teams types ----
  type TeamRole = 'owner' | 'admin' | 'member';

  interface TeamSummary {
    id: string;
    name: string;
    slug: string;
    displayOrder: number;
    role?: TeamRole;
  }

  interface TeamMember {
    id: string;
    teamId: string;
    clerkUserId: string;
    email: string;
    displayName: string;
    role: TeamRole;
    status: string;
    joinedAt?: number | null;
  }

  interface TeamInvite {
    id: string;
    teamId: string;
    teamName: string;
    teamSlug: string;
    email: string;
    role: TeamRole;
    status: string;
    expiresAt: number;
    createdAt: number;
    shareUrl?: string;
  }

  interface TeamInviteList {
    incoming: TeamInvite[];
    sent: TeamInvite[];
  }

  interface TeamHost {
    id: string;
    teamId: string;
    label: string;
    hostname: string;
    username: string;
    port: number;
    group?: string;
    tags?: string[];
    notes?: string;
    authMode?: string;
    credentialMode?: string;
    credentialType?: string;
    secretVisibility?: string;
    lastConnectedAt?: number | null;
    createdAt?: number;
    updatedAt?: number;
    canManageHosts?: boolean;
    canRevealSecrets?: boolean;
    canEditNotes?: boolean;
  }

  interface CreateTeamHostRequest {
    label: string;
    hostname: string;
    username: string;
    port: number;
    group?: string;
    tags?: string[];
    notes?: string;
    credentialMode: string;
    credentialType: string;
    secretVisibility: string;
    sharedCredential?: string;
  }

  interface UpdateTeamHostRequest {
    label: string;
    hostname: string;
    username: string;
    port: number;
    group?: string;
    tags?: string[];
    notes?: string;
    credentialMode: string;
    credentialType: string;
    secretVisibility: string;
    sharedCredential?: string;
    clearSharedCredential?: boolean;
  }

  interface TeamAuditEvent {
    id: string;
    teamId: string;
    actorClerkUserId: string;
    actorDisplayName: string;
    entityType: string;
    entityId: string;
    eventType: string;
    targetClerkUserId?: string;
    targetDisplayName?: string;
    summary: string;
    metadata?: Record<string, unknown>;
    createdAt: number;
  }

  interface RevealedTeamHostCredential {
    hostId: string;
    memberClerkUserId?: string;
    credentialType: string;
    username?: string;
    secret?: string;
    updatedAt?: number;
  }

  interface TeamHostCredentialRosterEntry {
    memberId: string;
    displayName: string;
    email: string;
    role: TeamRole;
    isOwner: boolean;
    isCurrentUser: boolean;
    hasCredential: boolean;
    credentialType: string;
    username?: string;
    updatedAt?: number;
  }

  interface UpsertMyCredentialRequest {
    credentialType: string;
    secret: string;
    username?: string;
  }

  interface ImportPersonalHostPreviewResult {
    hasConflict: boolean;
    isIdentical?: boolean;
    existingHostId?: string;
    existingLabel?: string;
    proposed: CreateTeamHostRequest;
  }

  type ImportPersonalHostAction = 'create' | 'update' | 'duplicate';

  interface ImportPersonalHostCommitRequest {
    personalHostId: string;
    teamId: string;
    action: ImportPersonalHostAction;
    existingHostId?: string;
  }

  // ---- Token types ----
  interface TokenHostGrant {
    hostId: string;
  }

  interface TokenSummary {
    id: string;
    name: string;
    status: string;
    createdAt: number;
    revokedAt?: number | null;
    lastUsedAt?: number | null;
    useCount?: number;
  }

  // ---- Phase 6: health, mount, transfer, exec types ----
  interface HealthResult {
    hostId: string;
    status: string;
    checkedAt: string;
    latencyMs: number;
    error?: string;
    uptimeSecs?: number;
    cpuPercent?: number;
    memTotalBytes?: number;
    memAvailBytes?: number;
    diskTotalBytes?: number;
    diskAvailBytes?: number;
  }

  interface MountSummary {
    hostId: string;
    hostname: string;
    localPath: string;
    remotePath: string;
  }

  interface TransferParams {
    hostId: string;
    local: string;
    remote: string;
    recursive?: boolean;
    preserve?: boolean;
  }

  interface ExecResult {
    stdout: string;
    stderr: string;
    exitCode: number;
    durationMs: number;
  }

  interface TransferProgress {
    transferId: string;
    hostId: string;
    status: 'started' | 'finished' | 'failed';
    direction: 'upload' | 'download';
    local: string;
    remote: string;
    error?: string;
  }

  // ---- Phase 4: auth + sync types ----
  interface AuthSessionInfo {
    userId: string;
    userName: string;
    userEmail: string;
    currentTeamId?: string;
    expiresAt: number;
  }

  interface AuthStartSignInResult {
    url: string;
    sessionId: string;
    deviceCode: string;
    pollSecret: string;
    pollIntervalSeconds: number;
    expiresAt: number;
  }

  interface AuthPollSignInResult {
    status: 'pending' | 'completed' | 'expired';
    session?: AuthSessionInfo;
  }

  interface SyncStatusResult {
    provider: string;
    status: string;
    stage?: string;
    lastResultAt?: number;
    lastResultStatus?: string;
    lastMessage?: string;
    dirtyCount: number;
  }

  interface SyncNowResult {
    success: boolean;
    message: string;
    hostsPulled: number;
    hostsPushed: number;
    hostsAdded: number;
    hostsUpdated: number;
    hostsRemoved: number;
  }

  interface SyncConfigureParams {
    provider?: 'off' | 'git' | 'cloud';
    repoUrl?: string;
    branch?: string;
    sshKeyPath?: string;
    enabled?: boolean;
  }

  interface SSHThingAPI {
    daemonVersion: () => Promise<{ version: string }>;

    // Vault
    unlock: (password: string) => Promise<UnlockResult>;
    vaultStatus: () => Promise<VaultStatus>;
    createVault: (password: string) => Promise<{ ok: boolean }>;
    changeVaultPassword: (oldPassword: string, newPassword: string) => Promise<{ ok: boolean }>;
    lockVault: () => Promise<{ ok: boolean }>;
    vacuumVault: () => Promise<{ ok: boolean }>;

    // Hosts
    listHosts: (query?: string) => Promise<{ hosts: HostSummary[] }>;
    getHost: (id: string) => Promise<HostSummary>;
    createHost: (host: HostCreate) => Promise<{ id: string }>;
    updateHost: (host: HostUpdate) => Promise<{ ok: boolean }>;
    updateHostWithKey: (host: HostUpdate & { plainKey: string }) => Promise<{ ok: boolean }>;
    deleteHost: (id: string) => Promise<{ ok: boolean }>;
    revealCredential: (hostId: string) => Promise<{ credential: string; authMode: string }>;
    generateKey: (keyType: string, comment: string) => Promise<GenerateKeyResult>;
    importKey: (format: string, blob: string, label: string, hostname: string, username: string, port: number) => Promise<{ id: string }>;

    // Groups
    listGroups: () => Promise<{ groups: GroupSummary[] }>;
    createGroup: (name: string) => Promise<{ id: string }>;
    renameGroup: (oldName: string, newName: string) => Promise<{ ok: boolean }>;
    deleteGroup: (name: string) => Promise<{ ok: boolean }>;

    // Sessions
    openSession: (hostId: string, cols: number, rows: number, term?: string) => Promise<{ sessionId: string }>;
    sessionWrite: (sessionId: string, data: number[]) => Promise<{ ok: boolean }>;
    sessionResize: (sessionId: string, cols: number, rows: number) => Promise<{ ok: boolean }>;
    sessionClose: (sessionId: string) => Promise<{ ok: boolean }>;
    sessionList: () => Promise<{ sessions: SessionInfo[] }>;

    // Settings
    getSettings: () => Promise<AppSettings>;
    setSettings: (patch: Partial<AppSettings>) => Promise<{ ok: boolean }>;

    // Teams
    teamsList: () => Promise<{ teams: TeamSummary[] }>;
    teamsHostsList: (teamId: string) => Promise<{ hosts: TeamHost[] }>;
    teamsHostsCreate: (teamId: string, req: CreateTeamHostRequest) => Promise<TeamHost>;
    teamsHostsUpdate: (hostId: string, req: UpdateTeamHostRequest) => Promise<{ ok: boolean }>;
    teamsHostsDelete: (hostId: string) => Promise<{ ok: boolean }>;
    teamsMembersList: (teamId: string) => Promise<{ members: TeamMember[] }>;
    teamsMembersInvite: (teamId: string, email: string, role: TeamRole) => Promise<TeamInvite>;
    teamsMembersUpdateRole: (teamId: string, memberId: string, role: TeamRole) => Promise<{ ok: boolean }>;
    teamsMembersRemove: (teamId: string, memberId: string) => Promise<{ ok: boolean }>;
    teamsInvitesList: (teamId: string) => Promise<TeamInviteList>;
    teamsInvitesAccept: (inviteId: string) => Promise<{ ok: boolean }>;
    teamsInvitesRevoke: (teamId: string, inviteId: string) => Promise<{ ok: boolean }>;
    teamsAuditList: (teamId: string) => Promise<{ events: TeamAuditEvent[] }>;
    teamsCreate: (name: string) => Promise<TeamSummary>;
    teamsRename: (teamId: string, name: string) => Promise<TeamSummary>;
    teamsDelete: (teamId: string) => Promise<{ ok: boolean }>;
    teamsReorder: (teamIds: string[]) => Promise<{ ok: boolean }>;
    teamsLeave: (teamId: string) => Promise<{ ok: boolean }>;

    // Team credentials
    teamsHostsRevealShared: (hostId: string) => Promise<RevealedTeamHostCredential>;
    teamsHostsRosterList: (hostId: string) => Promise<{ roster: TeamHostCredentialRosterEntry[] }>;
    teamsHostsRevealMember: (hostId: string, memberId: string) => Promise<RevealedTeamHostCredential>;
    teamsHostsDeleteMemberCredential: (hostId: string, memberId: string) => Promise<{ ok: boolean }>;
    teamsHostsUpsertMyCredential: (hostId: string, req: UpsertMyCredentialRequest) => Promise<{ ok: boolean }>;
    teamsHostsImportPersonalPreview: (personalHostId: string, teamId: string) => Promise<ImportPersonalHostPreviewResult>;
    teamsHostsImportPersonalCommit: (req: ImportPersonalHostCommitRequest) => Promise<{ ok: boolean }>;

    // Tokens
    tokensList: () => Promise<{ tokens: TokenSummary[] }>;
    tokensCreate: (name: string, grants: TokenHostGrant[]) => Promise<{ rawToken: string }>;
    tokensRevoke: (tokenId: string) => Promise<{ ok: boolean }>;
    tokensDeleteRevoked: (tokenId: string) => Promise<{ ok: boolean }>;

    // Health
    healthProbe: (hostId: string) => Promise<HealthResult>;
    healthList: () => Promise<{ results: HealthResult[] }>;

    // Mounts
    mountStart: (hostId: string, remotePath: string) => Promise<MountSummary>;
    mountStop: (hostId: string) => Promise<{ ok: boolean }>;
    mountList: () => Promise<{ mounts: MountSummary[] }>;
    mountCheckPrereqs: () => Promise<{ ok: boolean; platform: string; missing: string[]; hints: string[] }>;

    // Transfer
    transferUpload: (params: TransferParams) => Promise<{ transferId: string }>;
    transferDownload: (params: TransferParams) => Promise<{ transferId: string }>;
    transferCancel: (transferId: string) => Promise<{ ok: boolean }>;

    // Exec
    sessionExec: (hostId: string, cmd: string, timeoutMs?: number) => Promise<ExecResult>;

    // Auth
    authStartSignIn: () => Promise<AuthStartSignInResult>;
    authOpenBrowser: (url: string) => Promise<{ ok: boolean }>;
    authPollSignIn: (sessionId: string, pollSecret: string) => Promise<AuthPollSignInResult>;
    authSignOut: () => Promise<{ ok: boolean }>;
    authSession: () => Promise<{ session: AuthSessionInfo | null }>;
    authTokenForRenderer: () => Promise<{ token: string }>;

    // Sync
    syncStatus: () => Promise<SyncStatusResult>;
    syncNow: () => Promise<SyncNowResult>;
    syncConfigure: (params: SyncConfigureParams) => Promise<{ ok: boolean }>;
    syncEvents: () => Promise<{ events: Array<{ source: string; action: string; itemType?: string; itemCount?: number; createdAt: number }> }>;
    syncDevices: () => Promise<{ devices: Array<Record<string, unknown>> }>;
    syncForgetDevice: (deviceId: string) => Promise<{ ok: boolean }>;
    syncTestGit: (repoUrl: string, sshKeyPath: string) => Promise<{ ok: boolean; message?: string }>;
    keyringHealthCheck: () => Promise<{ ok: boolean; error?: string }>;

    // Updates
    installUpdate: () => Promise<void>;
    checkForUpdates: () => Promise<void>;

    // System
    openPath: (filePath: string) => Promise<string>;

    // Dialog
    chooseDirectory: () => Promise<{ canceled: boolean; path: string | null }>;

    // Notifications
    onNotification: (cb: (method: string, params: unknown) => void) => () => void;

    // App-menu commands forwarded from Electron main process
    onMenuCommand: (cb: (cmd: string) => void) => () => void;

    // Fired when the daemon process exits unexpectedly
    onDaemonExited: (cb: () => void) => () => void;

    // Fired when an app update is available
    onUpdateAvailable: (cb: (info: { version: string; releaseDate: string; releaseNotes?: string }) => void) => () => void;

    // Fired when an app update has been downloaded
    onUpdateDownloaded: (cb: (info: { version: string; releaseDate: string; releaseNotes?: string }) => void) => () => void;
  }

  interface Window {
    sshthing: SSHThingAPI;
  }
}
