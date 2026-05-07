/**
 * DaemonClient — typed JSON-RPC 2.0 client over a Unix socket (macOS/Linux)
 * or named pipe (Windows).
 *
 * Single-client assumption: one Electron window, one persistent connection.
 * Notifications from the daemon are re-emitted as Node EventEmitter events.
 */
import { EventEmitter } from 'events';
import * as net from 'net';
import * as path from 'path';
import * as os from 'os';

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
  /** plaintext private key PEM */
  plainKey?: string;
  /** plaintext password */
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
  releaseChannel?: 'stable' | 'beta';
  autoApplyUpdates?: boolean;
}

export interface SessionInfo {
  sessionId: string;
  hostId: string;
  openedAt: string;
  label: string;
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

export interface UnlockResult {
  unlocked: boolean;
  salt: string;
  sessionTtlSec: number;
}

export interface VaultStatus {
  unlocked: boolean;
  expiresAt: number | null;
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

interface PendingRequest {
  resolve: (result: unknown) => void;
  reject: (err: Error) => void;
}

export class DaemonClient extends EventEmitter {
  private socket: net.Socket | null = null;
  private token: string = '';
  private nextId = 1;
  private pending = new Map<number, PendingRequest>();
  private lineBuffer = '';

  /** Connect to the daemon socket and set the auth token. */
  connect(sockPath: string, token: string): Promise<void> {
    this.token = token;
    return new Promise((resolve, reject) => {
      const sock = net.createConnection(sockPath);
      sock.setEncoding('utf8');

      sock.once('connect', () => {
        this.socket = sock;
        resolve();
      });

      sock.once('error', (err) => {
        if (!this.socket) {
          reject(err);
        } else {
          this.emit('error', err);
        }
      });

      sock.on('data', (chunk: string) => {
        this.lineBuffer += chunk;
        let idx: number;
        while ((idx = this.lineBuffer.indexOf('\n')) !== -1) {
          const line = this.lineBuffer.slice(0, idx).trim();
          this.lineBuffer = this.lineBuffer.slice(idx + 1);
          if (line) {
            this.handleLine(line);
          }
        }
      });

      sock.on('close', () => {
        this.socket = null;
        this.emit('close');
      });
    });
  }

  private handleLine(line: string): void {
    let msg: Record<string, unknown>;
    try {
      msg = JSON.parse(line) as Record<string, unknown>;
    } catch {
      console.error('[daemon] parse error:', line);
      return;
    }

    // Notification: has method, no id.
    if (typeof msg['method'] === 'string' && msg['id'] === undefined) {
      this.emit('notification', msg['method'], msg['params']);
      this.emit(`notification:${msg['method']}`, msg['params']);
      return;
    }

    // Response: has id.
    const id = msg['id'] as number;
    const pending = this.pending.get(id);
    if (!pending) return;
    this.pending.delete(id);

    if (msg['error']) {
      const err = msg['error'] as { code: number; message: string };
      const e = new Error(err.message) as Error & { code: number };
      e.code = err.code;
      pending.reject(e);
    } else {
      pending.resolve(msg['result']);
    }
  }

  private call<T>(method: string, params: unknown): Promise<T> {
    return new Promise((resolve, reject) => {
      if (!this.socket) {
        reject(new Error('not connected'));
        return;
      }
      const id = this.nextId++;
      this.pending.set(id, {
        resolve: resolve as (v: unknown) => void,
        reject,
      });
      const req = JSON.stringify({
        jsonrpc: '2.0',
        id,
        auth: this.token,
        method,
        params,
      }) + '\n';
      this.socket.write(req);
    });
  }

  daemonVersion(): Promise<{ version: string }> {
    return this.call('daemon.version', {});
  }

  unlock(password: string): Promise<UnlockResult> {
    return this.call('vault.unlock', { password });
  }

  vaultStatus(): Promise<VaultStatus> {
    return this.call('vault.status', {});
  }

  listHosts(query?: string): Promise<{ hosts: HostSummary[] }> {
    return this.call('hosts.list', { query: query ?? '' });
  }

  getHost(id: string): Promise<HostSummary> {
    return this.call('hosts.get', { id });
  }

  openSession(hostId: string, cols: number, rows: number, term?: string): Promise<{ sessionId: string }> {
    return this.call('session.open', { hostId, cols, rows, term: term ?? 'xterm-256color' });
  }

  sessionWrite(sessionId: string, data: Uint8Array): Promise<{ ok: boolean }> {
    // Encode raw bytes to base64 for transport.
    const b64 = Buffer.from(data).toString('base64');
    return this.call('session.write', { sessionId, b64 });
  }

  sessionResize(sessionId: string, cols: number, rows: number): Promise<{ ok: boolean }> {
    return this.call('session.resize', { sessionId, cols, rows });
  }

  sessionClose(sessionId: string): Promise<{ ok: boolean }> {
    return this.call('session.close', { sessionId });
  }

  sessionList(): Promise<{ sessions: SessionInfo[] }> {
    return this.call('session.list', {});
  }

  // ---- Host CRUD ----

  createHost(host: HostCreate): Promise<{ id: string }> {
    return this.call('hosts.create', host);
  }

  updateHost(host: HostUpdate): Promise<{ ok: boolean }> {
    return this.call('hosts.update', host);
  }

  updateHostWithKey(host: HostUpdate & { plainKey: string }): Promise<{ ok: boolean }> {
    return this.call('hosts.updateWithKey', host);
  }

  deleteHost(id: string): Promise<{ ok: boolean }> {
    return this.call('hosts.delete', { id });
  }

  revealCredential(hostId: string): Promise<{ credential: string; authMode: string }> {
    return this.call('hosts.revealCredential', { id: hostId });
  }

  generateKey(keyType: string, comment: string): Promise<GenerateKeyResult> {
    return this.call('hosts.generateKey', { type: keyType, comment });
  }

  importKey(format: string, blob: string, label: string, hostname: string, username: string, port: number): Promise<{ id: string }> {
    return this.call('hosts.import', { format, blob, label, hostname, username, port });
  }

  // ---- Groups ----

  listGroups(): Promise<{ groups: GroupSummary[] }> {
    return this.call('groups.list', {});
  }

  createGroup(name: string): Promise<{ id: string }> {
    return this.call('groups.create', { name });
  }

  renameGroup(oldName: string, newName: string): Promise<{ ok: boolean }> {
    return this.call('groups.rename', { old: oldName, new: newName });
  }

  deleteGroup(name: string): Promise<{ ok: boolean }> {
    return this.call('groups.delete', { name });
  }

  // ---- Vault ----

  createVault(password: string): Promise<{ ok: boolean }> {
    return this.call('vault.create', { password });
  }

  changeVaultPassword(oldPassword: string, newPassword: string): Promise<{ ok: boolean }> {
    return this.call('vault.changePassword', { oldPassword, newPassword });
  }

  lockVault(): Promise<{ ok: boolean }> {
    return this.call('vault.lock', {});
  }

  vacuumVault(): Promise<{ ok: boolean }> {
    return this.call('vault.vacuum', {});
  }

  // ---- Settings ----

  getSettings(): Promise<AppSettings> {
    return this.call('settings.get', {});
  }

  setSettings(patch: Partial<AppSettings>): Promise<{ ok: boolean }> {
    return this.call('settings.set', patch);
  }

  // ---- Teams ----

  teamsList(): Promise<{ teams: TeamSummary[] }> {
    return this.call('teams.list', {});
  }

  teamsHostsList(teamId: string): Promise<{ hosts: TeamHost[] }> {
    return this.call('teams.hosts.list', { teamId });
  }

  teamsHostsCreate(teamId: string, req: CreateTeamHostRequest): Promise<TeamHost> {
    return this.call('teams.hosts.create', { teamId, host: req });
  }

  teamsHostsUpdate(hostId: string, req: UpdateTeamHostRequest): Promise<{ ok: boolean }> {
    return this.call('teams.hosts.update', { hostId, host: req });
  }

  teamsHostsDelete(hostId: string): Promise<{ ok: boolean }> {
    return this.call('teams.hosts.delete', { hostId });
  }

  teamsMembersList(teamId: string): Promise<{ members: TeamMember[] }> {
    return this.call('teams.members.list', { teamId });
  }

  teamsMembersInvite(teamId: string, email: string, role: TeamRole): Promise<TeamInvite> {
    return this.call('teams.members.invite', { teamId, email, role });
  }

  teamsMembersUpdateRole(teamId: string, memberId: string, role: TeamRole): Promise<{ ok: boolean }> {
    return this.call('teams.members.updateRole', { teamId, memberId, role });
  }

  teamsMembersRemove(teamId: string, memberId: string): Promise<{ ok: boolean }> {
    return this.call('teams.members.remove', { teamId, memberId });
  }

  teamsInvitesList(teamId: string): Promise<TeamInviteList> {
    return this.call('teams.invites.list', { teamId });
  }

  teamsInvitesAccept(inviteId: string): Promise<{ ok: boolean }> {
    return this.call('teams.invites.accept', { inviteId });
  }

  teamsInvitesRevoke(teamId: string, inviteId: string): Promise<{ ok: boolean }> {
    return this.call('teams.invites.revoke', { teamId, inviteId });
  }

  teamsAuditList(teamId: string): Promise<{ events: TeamAuditEvent[] }> {
    return this.call('teams.audit.list', { teamId });
  }

  teamsCreate(name: string): Promise<TeamSummary> {
    return this.call('teams.create', { name });
  }

  teamsRename(teamId: string, name: string): Promise<TeamSummary> {
    return this.call('teams.rename', { teamId, name });
  }

  teamsDelete(teamId: string): Promise<{ ok: boolean }> {
    return this.call('teams.delete', { teamId });
  }

  teamsReorder(teamIds: string[]): Promise<{ ok: boolean }> {
    return this.call('teams.reorder', { teamIds });
  }

  teamsLeave(teamId: string): Promise<{ ok: boolean }> {
    return this.call('teams.leave', { teamId });
  }

  // ---- Team credentials ----

  teamsHostsRevealShared(hostId: string): Promise<RevealedTeamHostCredential> {
    return this.call('teams.hosts.credentials.revealShared', { hostId });
  }

  teamsHostsRosterList(hostId: string): Promise<{ roster: TeamHostCredentialRosterEntry[] }> {
    return this.call('teams.hosts.credentials.rosterList', { hostId });
  }

  teamsHostsRevealMember(hostId: string, memberId: string): Promise<RevealedTeamHostCredential> {
    return this.call('teams.hosts.credentials.revealMember', { hostId, memberId });
  }

  teamsHostsDeleteMemberCredential(hostId: string, memberId: string): Promise<{ ok: boolean }> {
    return this.call('teams.hosts.credentials.deleteMember', { hostId, memberId });
  }

  teamsHostsUpsertMyCredential(hostId: string, req: UpsertMyCredentialRequest): Promise<{ ok: boolean }> {
    return this.call('teams.hosts.credentials.upsertMine', { hostId, ...req });
  }

  teamsHostsImportPersonalPreview(personalHostId: string, teamId: string): Promise<ImportPersonalHostPreviewResult> {
    return this.call('teams.hosts.importPersonal.preview', { personalHostId, teamId });
  }

  teamsHostsImportPersonalCommit(req: ImportPersonalHostCommitRequest): Promise<{ ok: boolean }> {
    return this.call('teams.hosts.importPersonal.commit', req);
  }

  // ---- Tokens ----

  tokensList(): Promise<{ tokens: TokenSummary[] }> {
    return this.call('tokens.list', {});
  }

  tokensCreate(name: string, grants: TokenHostGrant[]): Promise<{ rawToken: string }> {
    return this.call('tokens.create', { name, grants });
  }

  tokensRevoke(tokenId: string): Promise<{ ok: boolean }> {
    return this.call('tokens.revoke', { tokenId });
  }

  tokensDeleteRevoked(tokenId: string): Promise<{ ok: boolean }> {
    return this.call('tokens.deleteRevoked', { tokenId });
  }

  // ---- Health ----

  healthProbe(hostId: string): Promise<HealthResult> {
    return this.call('health.probe', { hostId });
  }

  healthList(): Promise<{ results: HealthResult[] }> {
    return this.call('health.list', {});
  }

  // ---- Mounts ----

  mountStart(hostId: string, remotePath: string): Promise<MountSummary> {
    return this.call('mount.start', { hostId, remotePath });
  }

  mountStop(hostId: string): Promise<{ ok: boolean }> {
    return this.call('mount.stop', { hostId });
  }

  mountList(): Promise<{ mounts: MountSummary[] }> {
    return this.call('mount.list', {});
  }

  mountCheckPrereqs(): Promise<{ ok: boolean; platform: string; missing: string[]; hints: string[] }> {
    return this.call('mount.checkPrereqs', {});
  }

  // ---- Transfer ----

  transferUpload(params: TransferParams): Promise<{ transferId: string }> {
    return this.call('transfer.upload', params);
  }

  transferDownload(params: TransferParams): Promise<{ transferId: string }> {
    return this.call('transfer.download', params);
  }

  transferCancel(transferId: string): Promise<{ ok: boolean }> {
    return this.call('transfer.cancel', { transferId });
  }

  // ---- Exec ----

  sessionExec(hostId: string, cmd: string, timeoutMs?: number): Promise<ExecResult> {
    return this.call('session.exec', { hostId, cmd, timeoutMs: timeoutMs ?? 0 });
  }

  // ---- Auth ----

  authStartSignIn(): Promise<AuthStartSignInResult> {
    return this.call('auth.startSignIn', {});
  }

  authOpenBrowser(url: string): Promise<{ ok: boolean }> {
    return this.call('auth.openBrowser', { url });
  }

  authPollSignIn(sessionId: string, pollSecret: string): Promise<AuthPollSignInResult> {
    return this.call('auth.pollSignIn', { sessionId, pollSecret });
  }

  authSignOut(): Promise<{ ok: boolean }> {
    return this.call('auth.signOut', {});
  }

  authSession(): Promise<{ session: AuthSessionInfo | null }> {
    return this.call('auth.session', {});
  }

  authTokenForRenderer(): Promise<{ token: string }> {
    return this.call('auth.tokenForRenderer', {});
  }

  // ---- Sync ----

  syncStatus(): Promise<SyncStatusResult> {
    return this.call('sync.status', {});
  }

  syncNow(): Promise<SyncNowResult> {
    return this.call('sync.now', {});
  }

  syncConfigure(params: SyncConfigureParams): Promise<{ ok: boolean }> {
    return this.call('sync.configure', params);
  }

  syncEvents(): Promise<{ events: Array<{ source: string; action: string; itemType?: string; itemCount?: number; createdAt: number }> }> {
    return this.call('sync.events', {});
  }

  syncDevices(): Promise<{ devices: Array<Record<string, unknown>> }> {
    return this.call('sync.devices', {});
  }

  syncForgetDevice(deviceId: string): Promise<{ ok: boolean }> {
    return this.call('sync.forgetDevice', { deviceId });
  }

  syncTestGit(repoUrl: string, sshKeyPath: string): Promise<{ ok: boolean; message?: string }> {
    return this.call('sync.testGit', { repoUrl, sshKeyPath });
  }

  keyringHealthCheck(): Promise<{ ok: boolean; error?: string }> {
    return this.call('keyring.healthCheck', {});
  }

  disconnect(): void {
    this.socket?.destroy();
    this.socket = null;
  }
}

/** Returns the platform-appropriate socket path for the daemon. */
export function defaultSocketPath(): string {
  if (process.platform === 'win32') {
    const username = os.userInfo().username;
    return `\\\\.\\pipe\\sshthing-${username}`;
  }
  if (process.platform === 'darwin') {
    return path.join(os.homedir(), 'Library', 'Application Support', 'SSHThing', 'sshthing.sock');
  }
  // Linux: XDG_RUNTIME_DIR or /tmp/sshthing-<user>.sock
  const xdg = process.env['XDG_RUNTIME_DIR'];
  if (xdg) return path.join(xdg, 'sshthing.sock');
  return `/tmp/sshthing-${os.userInfo().username}.sock`;
}

/** Returns the path to the daemon auth token file. */
export function defaultTokenPath(): string {
  if (process.platform === 'darwin') {
    return path.join(os.homedir(), 'Library', 'Application Support', 'SSHThing', 'daemon.token');
  }
  if (process.platform === 'win32') {
    return path.join(os.homedir(), 'AppData', 'Roaming', 'SSHThing', 'daemon.token');
  }
  const xdgData = process.env['XDG_DATA_HOME'] ?? path.join(os.homedir(), '.local', 'share');
  return path.join(xdgData, 'SSHThing', 'daemon.token');
}
