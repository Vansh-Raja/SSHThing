/**
 * Preload script — exposes the daemon RPC surface to the renderer via
 * contextBridge. The renderer only sees window.sshthing; it never touches
 * Node or Electron APIs directly.
 */
import { contextBridge, ipcRenderer } from 'electron';

export interface HostSummary {
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

export interface HostCreate {
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

export interface HostUpdate {
  id: string;
  label?: string;
  hostname?: string;
  username?: string;
  port?: number;
  group?: string;
  tags?: string[];
  authMode?: 'key' | 'password' | 'none';
}

export interface GroupSummary {
  id: string;
  name: string;
}

export interface GenerateKeyResult {
  publicKey: string;
  privateKey: string;
  keyType: string;
  comment: string;
}

export interface AppSettings {
  theme: 'light' | 'dark' | 'system';
  fontSize: number;
  termType: string;
  keepAliveSeconds: number;
  hostKeyPolicy: string;
  passwordBackend: string;
  syncProvider: 'off' | 'git' | 'cloud';
}

export interface SessionInfo {
  sessionId: string;
  hostId: string;
  openedAt: string;
  label: string;
}

export interface UnlockResult {
  unlocked: boolean;
  salt: string;
  sessionTtlSec: number;
}

export interface VaultStatus {
  unlocked: boolean;
  expiresAt: number | null;
}

// ---- Teams types ----

export type TeamRole = 'owner' | 'admin' | 'member';

export interface TeamSummary {
  id: string;
  name: string;
  slug: string;
  displayOrder: number;
  role?: TeamRole;
}

export interface TeamMember {
  id: string;
  teamId: string;
  clerkUserId: string;
  email: string;
  displayName: string;
  role: TeamRole;
  status: string;
  joinedAt?: number | null;
}

export interface TeamInvite {
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

export interface TeamInviteList {
  incoming: TeamInvite[];
  sent: TeamInvite[];
}

export interface TeamHost {
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

export interface CreateTeamHostRequest {
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

export interface UpdateTeamHostRequest {
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

export interface TeamAuditEvent {
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

export interface RevealedTeamHostCredential {
  hostId: string;
  memberClerkUserId?: string;
  credentialType: string;
  username?: string;
  secret?: string;
  updatedAt?: number;
}

export interface TeamHostCredentialRosterEntry {
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

export interface UpsertMyCredentialRequest {
  credentialType: string;
  secret: string;
  username?: string;
}

export interface ImportPersonalHostPreviewResult {
  hasConflict: boolean;
  isIdentical?: boolean;
  existingHostId?: string;
  existingLabel?: string;
  proposed: CreateTeamHostRequest;
}

export type ImportPersonalHostAction = 'create' | 'update' | 'duplicate';

export interface ImportPersonalHostCommitRequest {
  personalHostId: string;
  teamId: string;
  action: ImportPersonalHostAction;
  existingHostId?: string;
}

// ---- Token types ----

export interface TokenHostGrant {
  hostId: string;
}

export interface TokenSummary {
  id: string;
  name: string;
  status: string;
  createdAt: number;
  revokedAt?: number | null;
  lastUsedAt?: number | null;
  useCount?: number;
}

// ---- Phase 6: health, mount, transfer, exec types ----

export interface HealthResult {
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

export interface MountSummary {
  hostId: string;
  hostname: string;
  localPath: string;
  remotePath: string;
}

export interface TransferParams {
  hostId: string;
  local: string;
  remote: string;
  recursive?: boolean;
  preserve?: boolean;
}

export interface ExecResult {
  stdout: string;
  stderr: string;
  exitCode: number;
  durationMs: number;
}

export interface TransferProgress {
  transferId: string;
  hostId: string;
  status: 'started' | 'finished' | 'failed';
  direction: 'upload' | 'download';
  local: string;
  remote: string;
  error?: string;
}

export interface AuthSessionInfo {
  userId: string;
  userName: string;
  userEmail: string;
  currentTeamId?: string;
  expiresAt: number;
}

export interface AuthStartSignInResult {
  url: string;
  sessionId: string;
  deviceCode: string;
  pollSecret: string;
  pollIntervalSeconds: number;
  expiresAt: number;
}

export interface AuthPollSignInResult {
  status: 'pending' | 'completed' | 'expired';
  session?: AuthSessionInfo;
}

export interface SyncStatusResult {
  provider: string;
  status: string;
  stage?: string;
  lastResultAt?: number;
  lastResultStatus?: string;
  lastMessage?: string;
  dirtyCount: number;
}

export interface SyncNowResult {
  success: boolean;
  message: string;
  hostsPulled: number;
  hostsPushed: number;
  hostsAdded: number;
  hostsUpdated: number;
  hostsRemoved: number;
}

export interface SyncConfigureParams {
  provider?: 'off' | 'git' | 'cloud';
  repoUrl?: string;
  branch?: string;
  sshKeyPath?: string;
  enabled?: boolean;
}

const sshthing = {
  daemonVersion: (): Promise<{ version: string }> =>
    ipcRenderer.invoke('daemon:version'),

  // ---- Vault ----
  unlock: (password: string): Promise<UnlockResult> =>
    ipcRenderer.invoke('vault:unlock', password),

  vaultStatus: (): Promise<VaultStatus> =>
    ipcRenderer.invoke('vault:status'),

  createVault: (password: string): Promise<{ ok: boolean }> =>
    ipcRenderer.invoke('vault:create', password),

  changeVaultPassword: (oldPassword: string, newPassword: string): Promise<{ ok: boolean }> =>
    ipcRenderer.invoke('vault:changePassword', oldPassword, newPassword),

  lockVault: (): Promise<{ ok: boolean }> =>
    ipcRenderer.invoke('vault:lock'),

  vacuumVault: (): Promise<{ ok: boolean }> =>
    ipcRenderer.invoke('vault:vacuum'),

  // ---- Hosts ----
  listHosts: (query?: string): Promise<{ hosts: HostSummary[] }> =>
    ipcRenderer.invoke('hosts:list', query),

  getHost: (id: string): Promise<HostSummary> =>
    ipcRenderer.invoke('hosts:get', id),

  createHost: (host: HostCreate): Promise<{ id: string }> =>
    ipcRenderer.invoke('hosts:create', host),

  updateHost: (host: HostUpdate): Promise<{ ok: boolean }> =>
    ipcRenderer.invoke('hosts:update', host),

  updateHostWithKey: (host: HostUpdate & { plainKey: string }): Promise<{ ok: boolean }> =>
    ipcRenderer.invoke('hosts:updateWithKey', host),

  deleteHost: (id: string): Promise<{ ok: boolean }> =>
    ipcRenderer.invoke('hosts:delete', id),

  revealCredential: (hostId: string): Promise<{ credential: string; authMode: string }> =>
    ipcRenderer.invoke('hosts:revealCredential', hostId),

  generateKey: (keyType: string, comment: string): Promise<GenerateKeyResult> =>
    ipcRenderer.invoke('hosts:generateKey', keyType, comment),

  importKey: (format: string, blob: string, label: string, hostname: string, username: string, port: number): Promise<{ id: string }> =>
    ipcRenderer.invoke('hosts:importKey', format, blob, label, hostname, username, port),

  // ---- Groups ----
  listGroups: (): Promise<{ groups: GroupSummary[] }> =>
    ipcRenderer.invoke('groups:list'),

  createGroup: (name: string): Promise<{ id: string }> =>
    ipcRenderer.invoke('groups:create', name),

  renameGroup: (oldName: string, newName: string): Promise<{ ok: boolean }> =>
    ipcRenderer.invoke('groups:rename', oldName, newName),

  deleteGroup: (name: string): Promise<{ ok: boolean }> =>
    ipcRenderer.invoke('groups:delete', name),

  // ---- Sessions ----
  openSession: (hostId: string, cols: number, rows: number, term?: string): Promise<{ sessionId: string }> =>
    ipcRenderer.invoke('session:open', hostId, cols, rows, term),

  /** data must be a plain Array of byte values (not Uint8Array — that can't cross IPC). */
  sessionWrite: (sessionId: string, data: number[]): Promise<{ ok: boolean }> =>
    ipcRenderer.invoke('session:write', sessionId, data),

  sessionResize: (sessionId: string, cols: number, rows: number): Promise<{ ok: boolean }> =>
    ipcRenderer.invoke('session:resize', sessionId, cols, rows),

  sessionClose: (sessionId: string): Promise<{ ok: boolean }> =>
    ipcRenderer.invoke('session:close', sessionId),

  sessionList: (): Promise<{ sessions: SessionInfo[] }> =>
    ipcRenderer.invoke('session:list'),

  // ---- Settings ----
  getSettings: (): Promise<AppSettings> =>
    ipcRenderer.invoke('settings:get'),

  setSettings: (patch: Partial<AppSettings>): Promise<{ ok: boolean }> =>
    ipcRenderer.invoke('settings:set', patch),

  // ---- Teams ----
  teamsList: (): Promise<{ teams: TeamSummary[] }> =>
    ipcRenderer.invoke('teams:list'),

  teamsHostsList: (teamId: string): Promise<{ hosts: TeamHost[] }> =>
    ipcRenderer.invoke('teams:hosts:list', teamId),

  teamsHostsCreate: (teamId: string, req: CreateTeamHostRequest): Promise<TeamHost> =>
    ipcRenderer.invoke('teams:hosts:create', teamId, req),

  teamsHostsUpdate: (hostId: string, req: UpdateTeamHostRequest): Promise<{ ok: boolean }> =>
    ipcRenderer.invoke('teams:hosts:update', hostId, req),

  teamsHostsDelete: (hostId: string): Promise<{ ok: boolean }> =>
    ipcRenderer.invoke('teams:hosts:delete', hostId),

  teamsMembersList: (teamId: string): Promise<{ members: TeamMember[] }> =>
    ipcRenderer.invoke('teams:members:list', teamId),

  teamsMembersInvite: (teamId: string, email: string, role: TeamRole): Promise<TeamInvite> =>
    ipcRenderer.invoke('teams:members:invite', teamId, email, role),

  teamsMembersUpdateRole: (teamId: string, memberId: string, role: TeamRole): Promise<{ ok: boolean }> =>
    ipcRenderer.invoke('teams:members:updateRole', teamId, memberId, role),

  teamsMembersRemove: (teamId: string, memberId: string): Promise<{ ok: boolean }> =>
    ipcRenderer.invoke('teams:members:remove', teamId, memberId),

  teamsInvitesList: (teamId: string): Promise<TeamInviteList> =>
    ipcRenderer.invoke('teams:invites:list', teamId),

  teamsInvitesAccept: (inviteId: string): Promise<{ ok: boolean }> =>
    ipcRenderer.invoke('teams:invites:accept', inviteId),

  teamsInvitesRevoke: (teamId: string, inviteId: string): Promise<{ ok: boolean }> =>
    ipcRenderer.invoke('teams:invites:revoke', teamId, inviteId),

  teamsAuditList: (teamId: string): Promise<{ events: TeamAuditEvent[] }> =>
    ipcRenderer.invoke('teams:audit:list', teamId),

  teamsCreate: (name: string): Promise<TeamSummary> =>
    ipcRenderer.invoke('teams:create', name),

  teamsRename: (teamId: string, name: string): Promise<TeamSummary> =>
    ipcRenderer.invoke('teams:rename', teamId, name),

  teamsDelete: (teamId: string): Promise<{ ok: boolean }> =>
    ipcRenderer.invoke('teams:delete', teamId),

  teamsReorder: (teamIds: string[]): Promise<{ ok: boolean }> =>
    ipcRenderer.invoke('teams:reorder', teamIds),

  teamsLeave: (teamId: string): Promise<{ ok: boolean }> =>
    ipcRenderer.invoke('teams:leave', teamId),

  // ---- Team credentials ----
  teamsHostsRevealShared: (hostId: string): Promise<RevealedTeamHostCredential> =>
    ipcRenderer.invoke('teams:hosts:credentials:revealShared', hostId),

  teamsHostsRosterList: (hostId: string): Promise<{ roster: TeamHostCredentialRosterEntry[] }> =>
    ipcRenderer.invoke('teams:hosts:credentials:rosterList', hostId),

  teamsHostsRevealMember: (hostId: string, memberId: string): Promise<RevealedTeamHostCredential> =>
    ipcRenderer.invoke('teams:hosts:credentials:revealMember', hostId, memberId),

  teamsHostsDeleteMemberCredential: (hostId: string, memberId: string): Promise<{ ok: boolean }> =>
    ipcRenderer.invoke('teams:hosts:credentials:deleteMember', hostId, memberId),

  teamsHostsUpsertMyCredential: (hostId: string, req: UpsertMyCredentialRequest): Promise<{ ok: boolean }> =>
    ipcRenderer.invoke('teams:hosts:credentials:upsertMine', hostId, req),

  teamsHostsImportPersonalPreview: (personalHostId: string, teamId: string): Promise<ImportPersonalHostPreviewResult> =>
    ipcRenderer.invoke('teams:hosts:importPersonal:preview', personalHostId, teamId),

  teamsHostsImportPersonalCommit: (req: ImportPersonalHostCommitRequest): Promise<{ ok: boolean }> =>
    ipcRenderer.invoke('teams:hosts:importPersonal:commit', req),

  // ---- Tokens ----
  tokensList: (): Promise<{ tokens: TokenSummary[] }> =>
    ipcRenderer.invoke('tokens:list'),

  tokensCreate: (name: string, grants: TokenHostGrant[]): Promise<{ rawToken: string }> =>
    ipcRenderer.invoke('tokens:create', name, grants),

  tokensRevoke: (tokenId: string): Promise<{ ok: boolean }> =>
    ipcRenderer.invoke('tokens:revoke', tokenId),

  tokensDeleteRevoked: (tokenId: string): Promise<{ ok: boolean }> =>
    ipcRenderer.invoke('tokens:deleteRevoked', tokenId),

  // ---- Health ----
  healthProbe: (hostId: string): Promise<HealthResult> =>
    ipcRenderer.invoke('health:probe', hostId),

  healthList: (): Promise<{ results: HealthResult[] }> =>
    ipcRenderer.invoke('health:list'),

  // ---- Mounts ----
  mountStart: (hostId: string, remotePath: string): Promise<MountSummary> =>
    ipcRenderer.invoke('mount:start', hostId, remotePath),

  mountStop: (hostId: string): Promise<{ ok: boolean }> =>
    ipcRenderer.invoke('mount:stop', hostId),

  mountList: (): Promise<{ mounts: MountSummary[] }> =>
    ipcRenderer.invoke('mount:list'),

  mountCheckPrereqs: (): Promise<{ ok: boolean; platform: string; missing: string[]; hints: string[] }> =>
    ipcRenderer.invoke('mount:checkPrereqs'),

  // ---- Transfer ----
  transferUpload: (params: TransferParams): Promise<{ transferId: string }> =>
    ipcRenderer.invoke('transfer:upload', params),

  transferDownload: (params: TransferParams): Promise<{ transferId: string }> =>
    ipcRenderer.invoke('transfer:download', params),

  transferCancel: (transferId: string): Promise<{ ok: boolean }> =>
    ipcRenderer.invoke('transfer:cancel', transferId),

  // ---- Dialog ----
  /** Opens a native directory picker. Returns null path if the user cancelled. */
  chooseDirectory: (): Promise<{ canceled: boolean; path: string | null }> =>
    ipcRenderer.invoke('dialog:open-directory'),

  // ---- Exec ----
  sessionExec: (hostId: string, cmd: string, timeoutMs?: number): Promise<ExecResult> =>
    ipcRenderer.invoke('session:exec', hostId, cmd, timeoutMs),

  // ---- Auth ----
  authStartSignIn: (): Promise<AuthStartSignInResult> =>
    ipcRenderer.invoke('auth:startSignIn'),

  authOpenBrowser: (url: string): Promise<{ ok: boolean }> =>
    ipcRenderer.invoke('auth:openBrowser', url),

  authPollSignIn: (sessionId: string, pollSecret: string): Promise<AuthPollSignInResult> =>
    ipcRenderer.invoke('auth:pollSignIn', sessionId, pollSecret),

  authSignOut: (): Promise<{ ok: boolean }> =>
    ipcRenderer.invoke('auth:signOut'),

  authSession: (): Promise<{ session: AuthSessionInfo | null }> =>
    ipcRenderer.invoke('auth:session'),

  authTokenForRenderer: (): Promise<{ token: string }> =>
    ipcRenderer.invoke('auth:tokenForRenderer'),

  // ---- Sync ----
  syncStatus: (): Promise<SyncStatusResult> =>
    ipcRenderer.invoke('sync:status'),

  syncNow: (): Promise<SyncNowResult> =>
    ipcRenderer.invoke('sync:now'),

  syncConfigure: (params: SyncConfigureParams): Promise<{ ok: boolean }> =>
    ipcRenderer.invoke('sync:configure', params),

  syncEvents: (): Promise<{ events: Array<{ source: string; action: string; itemType?: string; itemCount?: number; createdAt: number }> }> =>
    ipcRenderer.invoke('sync:events'),

  syncDevices: (): Promise<{ devices: Array<Record<string, unknown>> }> =>
    ipcRenderer.invoke('sync:devices'),

  syncForgetDevice: (deviceId: string): Promise<{ ok: boolean }> =>
    ipcRenderer.invoke('sync:forgetDevice', deviceId),

  syncTestGit: (repoUrl: string, sshKeyPath: string): Promise<{ ok: boolean; message?: string }> =>
    ipcRenderer.invoke('sync:testGit', repoUrl, sshKeyPath),

  keyringHealthCheck: (): Promise<{ ok: boolean; error?: string }> =>
    ipcRenderer.invoke('keyring:healthCheck'),

  // ---- System ----
  /** Opens a path in the system file manager (Finder on macOS). Returns empty string on success. */
  openPath: (filePath: string): Promise<string> =>
    ipcRenderer.invoke('system:openPath', filePath),

  /**
   * Subscribe to daemon notifications (session.data, session.exit, vault.locked, etc.).
   * Returns an unsubscribe function.
   */
  onNotification: (cb: (method: string, params: unknown) => void): (() => void) => {
    const handler = (_event: Electron.IpcRendererEvent, method: string, params: unknown) => {
      cb(method, params);
    };
    ipcRenderer.on('daemon:notification', handler);
    return () => {
      ipcRenderer.removeListener('daemon:notification', handler);
    };
  },

  /**
   * Subscribe to app-menu commands forwarded from the Electron main process.
   * Known commands: 'open-settings' | 'lock-vault' | 'sign-out' |
   *                 'open-help' | 'new-tab' | 'open-account'
   * Returns an unsubscribe function.
   */
  onMenuCommand: (cb: (cmd: string) => void): (() => void) => {
    const handler = (_event: Electron.IpcRendererEvent, cmd: string) => {
      cb(cmd);
    };
    ipcRenderer.on('app:menu-command', handler);
    return () => {
      ipcRenderer.removeListener('app:menu-command', handler);
    };
  },

  /**
   * Subscribe to daemon-exit notifications so the renderer can surface a
   * reconnect banner. Returns an unsubscribe function.
   */
  onDaemonExited: (cb: () => void): (() => void) => {
    const handler = () => { cb(); };
    ipcRenderer.on('app:daemon-exited', handler);
    return () => {
      ipcRenderer.removeListener('app:daemon-exited', handler);
    };
  },
};

contextBridge.exposeInMainWorld('sshthing', sshthing);
