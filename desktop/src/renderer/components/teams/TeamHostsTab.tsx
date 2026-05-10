/**
 * TeamHostsTab — list, add, edit, delete, and CONNECT to team hosts.
 *
 * Mirrors the personal Hosts page layout exactly:
 *   ┌──────────── sidebar ──────────┬──────── detail ───────┐
 *   │ [groups + host list]          │ host detail or        │
 *   │ + Add host                    │ multi-tab terminal    │
 *   └───────────────────────────────┴───────────────────────┘
 *
 * The only differences from Hosts.tsx:
 * - Data comes from useTeamHosts(teamId) instead of window.sshthing.listHosts()
 * - Sessions are opened via openTeamTerminalSession() instead of openTerminalSession()
 * - Health/mount/exec/SFTP/download are UI-present but may show "coming soon" toasts
 *   because the daemon does not yet support these operations on team host IDs.
 */
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useTeamHosts } from '../../hooks/useTeamHosts';
import TeamHostDrawer from './TeamHostDrawer';
import TerminalTab, {
  openTeamTerminalSession,
  type TerminalTabData,
} from '../TerminalTab';
import Tabs from '../../ui/Tabs';
import Dialog from '../../ui/Dialog';
import Modal from '../../ui/Modal';
import DropdownMenu, { type MenuItemDef } from '../../ui/DropdownMenu';
import { toast } from '../../ui/toast';
import RevealTeamCredentialModal from './RevealTeamCredentialModal';
import PerMemberCredentialRoster from './PerMemberCredentialRoster';
import ImportPersonalHostModal from './ImportPersonalHostModal';
import HealthStats from '../HealthStats';
import ExecModal from '../ExecModal';
import UploadModal, { type UploadOptions } from '../UploadModal';
import DownloadModal from '../DownloadModal';
import TransferTray from '../TransferTray';
import { useHealth } from '../../hooks/useHealth';
import { useHealthScheduler } from '../../hooks/useHealthScheduler';
import { useMounts } from '../../hooks/useMounts';
import { useTransfers } from '../../hooks/useTransfers';
import { PlusIcon, ConnectIcon, FolderIcon, MountIcon } from '../icons';
import { SkeletonRows } from '../Skeleton';

type TeamHostsTabProps = {
  teamId: string;
  viewerRole: TeamRole;
};

let tabCounter = 0;
function newTabId(): string {
  return `team-tab-${++tabCounter}`;
}

type CredErrorModalState = {
  kind: 'personal_credential_not_configured' | 'shared_credential_not_configured';
  hostLabel: string;
  hostId: string;
  isPerMember: boolean;
} | null;

function formatLastConnected(ts: number | null | undefined): string {
  if (!ts) return 'Never';
  return new Date(ts * 1000).toLocaleDateString();
}

function relativeTime(input: string | number | null | undefined): string {
  if (!input) return '—';
  const d = typeof input === 'number' ? new Date(input * 1000) : new Date(input);
  const diff = Date.now() - d.getTime();
  if (Number.isNaN(diff)) return '—';
  const mins = Math.floor(diff / 60_000);
  if (mins < 1) return 'just now';
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  const days = Math.floor(hrs / 24);
  if (days < 30) return `${days}d ago`;
  return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' });
}

function shortDate(input: string | number | null | undefined): string {
  if (!input) return '—';
  const d = typeof input === 'number' ? new Date(input * 1000) : new Date(input);
  if (Number.isNaN(d.getTime())) return '—';
  return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' });
}

type SortOption = 'recent' | 'az';

export default function TeamHostsTab({ teamId, viewerRole }: TeamHostsTabProps) {
  const { hosts, loading, error, reload } = useTeamHosts(teamId);

  const [hostsLoaded, setHostsLoaded] = useState(false);
  const [sortBy, setSortBy] = useState<SortOption>('recent');
  const [selectedHostId, setSelectedHostId] = useState<string | null>(null);

  // Drawer state
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [editingHost, setEditingHost] = useState<TeamHost | null>(null);

  // Tabs (multi-terminal)
  const [tabs, setTabs] = useState<TerminalTabData[]>([]);
  const [activeTabId, setActiveTabId] = useState<string>('');

  // Collapsible groups
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>(() => {
    try {
      return JSON.parse(localStorage.getItem('sshthing:teamCollapsedGroups') ?? '{}') as Record<string, boolean>;
    } catch {
      return {};
    }
  });
  const toggleGroupCollapse = useCallback((name: string) => {
    setCollapsed((prev) => {
      const next = { ...prev, [name]: !prev[name] };
      localStorage.setItem('sshthing:teamCollapsedGroups', JSON.stringify(next));
      return next;
    });
  }, []);

  // Delete confirm
  const [deleteHostTarget, setDeleteHostTarget] = useState<TeamHost | null>(null);
  const [deleteHostLoading, setDeleteHostLoading] = useState(false);

  // Reveal modal
  const [revealTarget, setRevealTarget] = useState<TeamHost | null>(null);
  const [revealedCred, setRevealedCred] = useState<RevealedTeamHostCredential | null>(null);
  const [revealing, setRevealing] = useState(false);

  // Roster panel state
  const [rosterHost, setRosterHost] = useState<TeamHost | null>(null);

  // Import modal
  const [importOpen, setImportOpen] = useState(false);

  // Cred error modal
  const [credErrorModal, setCredErrorModal] = useState<CredErrorModalState>(null);

  // Phase 6 UI state (health, mounts, transfers, exec)
  const health = useHealth();
  useHealthScheduler({ hosts: hosts.map((h): HostSummary => ({ id: h.id, syncId: h.id, label: h.label || h.hostname, hostname: h.hostname, username: h.username, port: h.port, group: h.group ?? '', tags: h.tags ?? [], authMode: (h.authMode as 'key' | 'password' | 'none') ?? 'none', lastConnectedAt: h.lastConnectedAt != null ? new Date(h.lastConnectedAt * 1000).toISOString() : null })), health, scopeKey: `team:${teamId}` });
  const mounts = useMounts();
  const transfers = useTransfers();

  const [execModalOpen, setExecModalOpen] = useState(false);
  const [execTarget, setExecTarget] = useState<TeamHost | null>(null);
  const [uploadModalOpen, setUploadModalOpen] = useState(false);
  const [uploadHost, setUploadHost] = useState<TeamHost | null>(null);
  const [uploadLocalPath, setUploadLocalPath] = useState('');
  const [downloadModalOpen, setDownloadModalOpen] = useState(false);
  const [downloadHost, setDownloadHost] = useState<TeamHost | null>(null);

  useEffect(() => { void health.loadAll(); }, []); // eslint-disable-line react-hooks/exhaustive-deps
  useEffect(() => { void mounts.loadMounts(); }, []); // eslint-disable-line react-hooks/exhaustive-deps

  const canAdd = viewerRole === 'owner' || viewerRole === 'admin';

  // ── Data load ──
  useEffect(() => {
    setHostsLoaded(true);
    if (hosts.length > 0 && !selectedHostId) {
      setSelectedHostId(hosts[0]!.id);
    }
  }, [hosts, selectedHostId]);

  // ── Connect a host ──
  const adoptSession = useCallback((hostId: string, sessionId: string, label: string) => {
    const tabId = newTabId();
    const newTab: TerminalTabData = {
      id: tabId,
      hostId,
      hostLabel: label,
      sessionId,
      title: label,
    };
    setTabs((prev) => [...prev, newTab]);
    setActiveTabId(tabId);
  }, []);

  const connectHost = useCallback(async (host: TeamHost) => {
    let sessionId: string | null = null;
    try {
      sessionId = await openTeamTerminalSession(host, 80, 24);
    } catch (err: unknown) {
      const e = err as Error;
      const msg = e.message ?? 'Failed to connect';
      if (msg.includes('personal_credential_not_configured')) {
        setCredErrorModal({
          kind: 'personal_credential_not_configured',
          hostLabel: host.label || host.hostname,
          hostId: host.id,
          isPerMember: host.credentialMode === 'per_member',
        });
      } else if (msg.includes('shared_credential_not_configured')) {
        setCredErrorModal({
          kind: 'shared_credential_not_configured',
          hostLabel: host.label || host.hostname,
          hostId: host.id,
          isPerMember: false,
        });
      }
      return;
    }
    if (!sessionId) return;
    adoptSession(host.id, sessionId, host.label.trim() || host.hostname);
  }, [adoptSession]);

  // External adopt-session (from command bar)
  useEffect(() => {
    const onAdopt = (e: Event) => {
      const ev = e as CustomEvent<{ hostId: string; sessionId: string; label: string }>;
      adoptSession(ev.detail.hostId, ev.detail.sessionId, ev.detail.label);
    };
    window.addEventListener('sshthing:adopt-session', onAdopt);
    return () => window.removeEventListener('sshthing:adopt-session', onAdopt);
  }, [adoptSession]);

  // ── Tab management ──
  const handleCloseTab = useCallback(async (tabId: string, sessionId: string | null) => {
    if (sessionId) {
      try { await window.sshthing.sessionClose(sessionId); } catch { /* ignore */ }
    }
    setTabs((prev) => {
      const idx = prev.findIndex((t) => t.id === tabId);
      const next = prev.filter((t) => t.id !== tabId);
      if (activeTabId === tabId && next.length > 0) {
        const newActive = next[Math.min(idx, next.length - 1)]!;
        setActiveTabId(newActive.id);
      } else if (next.length === 0) {
        setActiveTabId('');
      }
      return next;
    });
  }, [activeTabId]);

  const handleTabTitleChange = useCallback((tabId: string, title: string) => {
    setTabs((prev) => prev.map((t) => (t.id === tabId ? { ...t, title } : t)));
  }, []);

  const handleTabExit = useCallback((tabId: string, exitCode: number) => {
    setTabs((prev) => prev.map((t) => (t.id === tabId ? { ...t, sessionId: null } : t)));
    if (exitCode === 0) {
      window.setTimeout(() => {
        void handleCloseTab(tabId, null);
      }, 600);
    }
  }, [handleCloseTab]);

  // ── Keyboard shortcuts ──
  const tabsRef = useRef(tabs);
  tabsRef.current = tabs;
  const activeTabIdRef = useRef(activeTabId);
  activeTabIdRef.current = activeTabId;

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (!(e.metaKey || e.ctrlKey)) return;
      if (e.key === 'w') {
        const tab = tabsRef.current.find((t) => t.id === activeTabIdRef.current);
        if (tab) {
          e.preventDefault();
          void handleCloseTab(tab.id, tab.sessionId);
        }
        return;
      }
      const num = parseInt(e.key, 10);
      if (!isNaN(num) && num >= 1 && num <= 9) {
        const target = tabsRef.current[num - 1];
        if (target) {
          e.preventDefault();
          setActiveTabId(target.id);
        }
      }
    };
    document.addEventListener('keydown', onKey);
    return () => document.removeEventListener('keydown', onKey);
  }, [handleCloseTab]);

  // ── Sidebar groups ──
  const groupedHosts = useMemo(() => {
    const map = new Map<string, TeamHost[]>();
    const orderedGroupNames: string[] = [];
    for (const host of hosts) {
      const key = host.group?.trim() || 'Ungrouped';
      if (!map.has(key)) {
        map.set(key, []);
        orderedGroupNames.push(key);
      }
      map.get(key)!.push(host);
    }
    for (const list of map.values()) {
      if (sortBy === 'az') {
        list.sort((a, b) => (a.label || a.hostname).localeCompare(b.label || b.hostname));
      } else {
        list.sort((a, b) => {
          const ta = a.lastConnectedAt ? new Date(a.lastConnectedAt * 1000).getTime() : 0;
          const tb = b.lastConnectedAt ? new Date(b.lastConnectedAt * 1000).getTime() : 0;
          if (tb !== ta) return tb - ta;
          return (a.label || a.hostname).localeCompare(b.label || b.hostname);
        });
      }
    }
    const sortedNames = orderedGroupNames.filter((n) => n !== 'Ungrouped');
    if (orderedGroupNames.includes('Ungrouped')) sortedNames.push('Ungrouped');
    return sortedNames.map((name) => ({ name, hosts: map.get(name)! }));
  }, [hosts, sortBy]);

  const selectedHost = hosts.find((h) => h.id === selectedHostId) ?? null;

  // ── Host actions ──
  const handleDeleteHost = useCallback(async () => {
    if (!deleteHostTarget) return;
    setDeleteHostLoading(true);
    try {
      await window.sshthing.teamsHostsDelete(deleteHostTarget.id);
      toast.success(`Host "${deleteHostTarget.label || deleteHostTarget.hostname}" deleted`);
      setDeleteHostTarget(null);
      reload();
    } catch (err) {
      toast.error((err as Error).message ?? 'Failed to delete host');
    } finally {
      setDeleteHostLoading(false);
    }
  }, [deleteHostTarget, reload]);

  const handleRevealShared = useCallback(async (host: TeamHost) => {
    setRevealTarget(host);
    setRevealing(true);
    try {
      const cred = await window.sshthing.teamsHostsRevealShared(host.id);
      setRevealedCred(cred);
    } catch (err: unknown) {
      toast.error((err as Error).message ?? 'Failed to reveal credential');
      setRevealTarget(null);
    } finally {
      setRevealing(false);
    }
  }, []);

  // ── Phase 6 actions (health, mount, exec, transfer) ──
  const handleProbe = useCallback(async (host: TeamHost) => {
    await health.probe(host.id);
  }, [health]);

  const handleMount = useCallback(async (host: TeamHost) => {
    toast.info('SSHFS mounts for team hosts are coming soon');
  }, []);

  const handleExec = useCallback((host: TeamHost) => {
    setExecTarget(host);
    setExecModalOpen(true);
  }, []);

  const handleDownload = useCallback((host: TeamHost) => {
    setDownloadHost(host);
    setDownloadModalOpen(true);
  }, []);

  const handleDropFile = useCallback((host: TeamHost, filePath: string) => {
    setUploadHost(host);
    setUploadLocalPath(filePath);
    setUploadModalOpen(true);
  }, []);

  const handleUploadConfirm = useCallback(async (localPath: string, remotePath: string, options: UploadOptions) => {
    if (!uploadHost) return;
    const hostLabel = uploadHost.label.trim() || uploadHost.hostname;
    try {
      await transfers.startUpload(uploadHost.id, hostLabel, localPath, remotePath, options.recursive, options.preserve);
    } catch (err) {
      toast.error((err as Error).message ?? 'Upload failed');
    }
  }, [uploadHost, transfers]);

  // ── Command palette integration ──
  useEffect(() => {
    const onNew = () => { setEditingHost(null); setDrawerOpen(true); };
    const onEdit = () => {
      const h = hosts.find((h_) => h_.id === selectedHostId);
      if (h) { setEditingHost(h); setDrawerOpen(true); }
    };
    const onDelete = () => {
      const h = hosts.find((h_) => h_.id === selectedHostId);
      if (h) setDeleteHostTarget(h);
    };
    const onConnect = () => {
      const h = hosts.find((h_) => h_.id === selectedHostId);
      if (h) void connectHost(h);
    };
    const onSort = () => {
      setSortBy((prev) => (prev === 'recent' ? 'az' : 'recent'));
    };
    const onHealth = () => {
      const h = hosts.find((h_) => h_.id === selectedHostId);
      if (h) void handleProbe(h);
    };
    const onMount = () => {
      const h = hosts.find((h_) => h_.id === selectedHostId);
      if (h) void handleMount(h);
    };
    const onExec = () => {
      const h = hosts.find((h_) => h_.id === selectedHostId);
      if (h) handleExec(h);
    };

    window.addEventListener('sshthing:cmd-new-host', onNew);
    window.addEventListener('sshthing:cmd-edit-selected', onEdit);
    window.addEventListener('sshthing:cmd-delete-selected', onDelete);
    window.addEventListener('sshthing:cmd-connect-selected', onConnect);
    window.addEventListener('sshthing:cmd-sort', onSort);
    window.addEventListener('sshthing:cmd-health-selected', onHealth);
    window.addEventListener('sshthing:cmd-mount-selected', onMount);
    window.addEventListener('sshthing:cmd-exec-selected', onExec);
    return () => {
      window.removeEventListener('sshthing:cmd-new-host', onNew);
      window.removeEventListener('sshthing:cmd-edit-selected', onEdit);
      window.removeEventListener('sshthing:cmd-delete-selected', onDelete);
      window.removeEventListener('sshthing:cmd-connect-selected', onConnect);
      window.removeEventListener('sshthing:cmd-sort', onSort);
      window.removeEventListener('sshthing:cmd-health-selected', onHealth);
      window.removeEventListener('sshthing:cmd-mount-selected', onMount);
      window.removeEventListener('sshthing:cmd-exec-selected', onExec);
    };
  }, [hosts, selectedHostId, connectHost, handleProbe, handleMount, handleExec]);

  const sortOptions: { value: SortOption; label: string }[] = [
    { value: 'recent', label: 'Recent' },
    { value: 'az', label: 'A→Z' },
  ];

  if (loading) {
    return (
      <div style={{ padding: 32, textAlign: 'center', color: 'var(--muted)', fontSize: 13 }}>
        Loading hosts…
      </div>
    );
  }

  if (error) {
    return (
      <div style={{ padding: 32, textAlign: 'center', color: 'var(--danger)', fontSize: 13 }}>
        {error}
      </div>
    );
  }

  return (
    <div className="pane-shell">
      {/* ── Sidebar ── */}
      <aside className="sidebar">
        {/* Sort control header */}
        <div style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'flex-end',
          padding: '8px 14px 0',
          gap: 6,
        }}>
          <span style={{ fontSize: 11, color: 'var(--muted)', flexShrink: 0 }}>Sort:</span>
          <select
            className="field__input"
            value={sortBy}
            onChange={(e) => setSortBy(e.target.value as SortOption)}
            style={{ fontSize: 11, height: 26, padding: '0 6px', minWidth: 80, cursor: 'pointer' }}
          >
            {sortOptions.map((opt) => (
              <option key={opt.value} value={opt.value}>{opt.label}</option>
            ))}
          </select>
        </div>

        <div className="sidebar__scroll">
          {!hostsLoaded && (
            <div style={{ padding: '12px 8px' }}>
              <SkeletonRows count={4} />
            </div>
          )}
          {hostsLoaded && groupedHosts.length === 0 && (
            <div style={{ padding: 24 }}>
              <div className="empty-state">
                <div className="empty-state__title">No hosts yet</div>
                <div>Click + Add host to get started.</div>
              </div>
            </div>
          )}
          {!loading && groupedHosts.map((g, i) => {
            const isCollapsed = !!collapsed[g.name];
            return (
              <div key={g.name} className="sidebar__group">
                <button
                  type="button"
                  className="sidebar__group-title"
                  style={{ justifyContent: 'space-between', width: '100%', background: 'transparent', border: 'none', cursor: 'pointer' }}
                  onClick={() => toggleGroupCollapse(g.name)}
                >
                  <span style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                    <span style={{ fontSize: 10, color: 'var(--muted)', width: 12, display: 'inline-block' }}>
                      {isCollapsed ? '▸' : '▾'}
                    </span>
                    {g.name}
                    <span className="sidebar__group-count">·</span>
                    <span className="sidebar__group-count">{g.hosts.length}</span>
                  </span>
                </button>
                {!isCollapsed && g.hosts.map((h) => {
                  const status = health.healthMap.get(h.id)?.status ?? 'unknown';
                  const isProbing = health.probing.has(h.id);
                  const dotClass =
                    status === 'online' ? 'status-dot--online'
                    : status === 'offline' || status === 'error' ? 'status-dot--offline'
                    : status === 'timeout' || status === 'unsupported' ? 'status-dot--warn'
                    : 'status-dot--unknown';
                  const isActive = selectedHostId === h.id;
                  return (
                    <button
                      key={h.id}
                      type="button"
                      className={`host-row${isActive ? ' host-row--active' : ''}`}
                      onClick={() => setSelectedHostId(h.id)}
                      onDoubleClick={() => void connectHost(h)}
                      onDragOver={(e) => {
                        if (e.dataTransfer.types.includes('Files')) {
                          e.preventDefault();
                          e.dataTransfer.dropEffect = 'copy';
                        }
                      }}
                      onDrop={(e) => {
                        e.preventDefault();
                        const file = e.dataTransfer.files[0];
                        if (file) {
                          const electronFile = file as File & { path?: string };
                          handleDropFile(h, electronFile.path ?? file.name);
                        }
                      }}
                    >
                      <span className={`status-dot ${dotClass}${isProbing ? ' status-dot--probing' : ''}`} />
                      <span className="host-row__label">{h.label.trim() || h.hostname}</span>
                    </button>
                  );
                })}
                {i < groupedHosts.length - 1 && <div className="sidebar__divider" />}
              </div>
            );
          })}
          {!loading && canAdd && (
            <button
              type="button"
              className="sidebar__add"
              style={{ marginTop: 4 }}
              onClick={() => { setEditingHost(null); setDrawerOpen(true); }}
            >
              <PlusIcon /> Add host
            </button>
          )}
          {!loading && canAdd && (
            <button
              type="button"
              className="sidebar__add"
              style={{ marginTop: 2 }}
              onClick={() => setImportOpen(true)}
            >
              <PlusIcon /> Import from vault
            </button>
          )}
        </div>
      </aside>

      {/* ── Detail / terminal pane ── */}
      <section className="detail">
        {tabs.length > 0 ? (
          <Tabs
            tabs={tabs.map((t) => ({
              id: t.id,
              label: t.title,
              onClose: () => void handleCloseTab(t.id, t.sessionId),
            }))}
            active={activeTabId}
            onTabChange={setActiveTabId}
            onNewTab={() => {
              if (selectedHost) void connectHost(selectedHost);
            }}
          >
            <div style={{ position: 'absolute', inset: 0 }}>
              {tabs.map((tab) => (
                <TerminalTab
                  key={tab.id}
                  data={tab}
                  active={tab.id === activeTabId}
                  onTitleChange={handleTabTitleChange}
                  onExit={handleTabExit}
                />
              ))}
            </div>
          </Tabs>
        ) : selectedHost ? (
          <div className="detail__scroll">
            <div className="detail__inner">
              {/* Title row */}
              <div className="detail__title-row">
                <h1 className="detail__title">{selectedHost.label.trim() || selectedHost.hostname}</h1>
              </div>

              {/* Address */}
              <div className="detail__address">
                <span style={{ fontFamily: 'var(--font-mono)' }}>
                  {selectedHost.username}@{selectedHost.hostname}{selectedHost.port !== 22 ? `:${selectedHost.port}` : ''}
                </span>
              </div>

              {/* Credential mode badge */}
              <div className="detail__status">
                {selectedHost.credentialMode && (
                  <span className="chip" style={{ fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.06em' }}>
                    {selectedHost.credentialMode}
                  </span>
                )}
                {selectedHost.credentialType && selectedHost.credentialType !== 'none' && (
                  <span className="chip" style={{ fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.06em', marginLeft: 6 }}>
                    {selectedHost.credentialType}
                  </span>
                )}
              </div>

              {/* Primary actions */}
              <div className="detail__actions" style={{ marginTop: 18 }}>
                <button type="button" className="btn btn--primary btn--lg" onClick={() => void connectHost(selectedHost)}>
                  <ConnectIcon /> Connect
                </button>
                <button type="button" className="btn btn--lg" onClick={() => toast.info('SFTP browser coming in v1.1')}>
                  <FolderIcon /> SFTP
                </button>
                <button type="button" className="btn btn--lg" onClick={() => void handleMount(selectedHost)}>
                  <MountIcon /> Mount
                </button>
                <span style={{ flex: 1 }} />
                {selectedHost.canManageHosts && (
                  <button
                    type="button"
                    className="btn btn--ghost"
                    onClick={() => { setEditingHost(selectedHost); setDrawerOpen(true); }}
                  >
                    Edit
                  </button>
                )}
                {selectedHost.canRevealSecrets && selectedHost.credentialMode === 'shared' && selectedHost.credentialType !== 'none' && (
                  <button
                    type="button"
                    className="btn btn--ghost"
                    onClick={() => void handleRevealShared(selectedHost)}
                  >
                    Reveal credential
                  </button>
                )}
                {selectedHost.canManageHosts && (
                  <button
                    type="button"
                    className="btn btn--ghost"
                    onClick={() => setDeleteHostTarget(selectedHost)}
                  >
                    Delete
                  </button>
                )}
              </div>

              <div className="detail__divider" />

              {/* Property list */}
              <div className="detail__props">
                <div className="detail__prop-label">Group</div>
                <div className="detail__prop-value">{selectedHost.group?.trim() || '—'}</div>

                <div className="detail__prop-label">Credential mode</div>
                <div className="detail__prop-value">{selectedHost.credentialMode ?? '—'}</div>

                <div className="detail__prop-label">Credential type</div>
                <div className="detail__prop-value">{selectedHost.credentialType ?? '—'}</div>

                <div className="detail__prop-label">Last connected</div>
                <div className="detail__prop-value">{relativeTime(selectedHost.lastConnectedAt)}</div>

                <div className="detail__prop-label">Created</div>
                <div className="detail__prop-value">{shortDate(selectedHost.createdAt)}</div>

                <div className="detail__prop-label">Tags</div>
                <div className="detail__prop-value">
                  {selectedHost.tags && selectedHost.tags.length > 0
                    ? selectedHost.tags.map((t) => <span key={t} className="chip">{t}</span>)
                    : <span style={{ color: 'var(--muted-2)' }}>—</span>}
                </div>

                {selectedHost.notes && (
                  <>
                    <div className="detail__prop-label">Notes</div>
                    <div className="detail__prop-value" style={{ whiteSpace: 'pre-wrap' }}>{selectedHost.notes}</div>
                  </>
                )}
              </div>

              {/* Health stats */}
              <HealthStats
                result={health.healthMap.get(selectedHost.id) ?? null}
                probing={health.probing.has(selectedHost.id)}
                onProbe={() => void handleProbe(selectedHost)}
              />

              {/* Per-member roster button */}
              {selectedHost.credentialMode === 'per_member' && (
                <div style={{ marginTop: 20 }}>
                  <button
                    type="button"
                    className="btn btn--ghost"
                    onClick={() => setRosterHost(rosterHost?.id === selectedHost.id ? null : selectedHost)}
                  >
                    {rosterHost?.id === selectedHost.id ? 'Hide credential roster' : 'View credential roster'}
                  </button>
                  {rosterHost?.id === selectedHost.id && (
                    <div style={{ marginTop: 12 }}>
                      <PerMemberCredentialRoster
                        hostId={selectedHost.id}
                        hostLabel={selectedHost.label || selectedHost.hostname}
                        canManage={!!selectedHost.canManageHosts}
                        credentialType={selectedHost.credentialType ?? 'password'}
                      />
                    </div>
                  )}
                </div>
              )}

              {/* Tertiary actions */}
              <div style={{ marginTop: 28, display: 'flex', gap: 10, flexWrap: 'wrap' }}>
                <button type="button" className="btn btn--ghost" onClick={() => handleExec(selectedHost)}>
                  Run command…
                </button>
                <button type="button" className="btn btn--ghost" onClick={() => handleDownload(selectedHost)}>
                  Download file…
                </button>
              </div>
            </div>
          </div>
        ) : (
          <div className="detail-empty">
            <div className="detail-empty__title">No host selected</div>
            <div className="detail-empty__hint">Pick a host from the sidebar to see its details.</div>
            <span className="detail-empty__kbd">⌘K to search</span>
          </div>
        )}
      </section>

      {/* Overlays */}
      <TeamHostDrawer
        open={drawerOpen}
        onClose={() => setDrawerOpen(false)}
        teamId={teamId}
        host={editingHost}
        onSaved={reload}
      />

      <Dialog
        open={!!deleteHostTarget}
        onClose={() => setDeleteHostTarget(null)}
        title="Delete host"
        message={`Delete host "${deleteHostTarget?.label || deleteHostTarget?.hostname || ''}"? This cannot be undone.`}
        confirmLabel="Delete"
        confirmVariant="danger"
        onConfirm={() => void handleDeleteHost()}
        loading={deleteHostLoading}
      />

      <RevealTeamCredentialModal
        open={!!revealedCred}
        onClose={() => { setRevealedCred(null); setRevealTarget(null); }}
        hostLabel={revealTarget?.label || revealTarget?.hostname || ''}
        credentialScope="shared"
        credentialType={revealedCred?.credentialType ?? ''}
        secret={revealedCred?.secret ?? ''}
      />

      <ImportPersonalHostModal
        open={importOpen}
        onClose={() => setImportOpen(false)}
        teamId={teamId}
        onImported={reload}
      />

      <ExecModal
        open={execModalOpen}
        host={execTarget ? { id: execTarget.id, syncId: execTarget.id, label: execTarget.label || execTarget.hostname, hostname: execTarget.hostname, username: execTarget.username, port: execTarget.port, group: execTarget.group ?? '', tags: execTarget.tags ?? [], authMode: (execTarget.authMode as 'key' | 'password' | 'none') ?? 'none', lastConnectedAt: execTarget.lastConnectedAt != null ? new Date(execTarget.lastConnectedAt * 1000).toISOString() : null } : null}
        onClose={() => setExecModalOpen(false)}
      />

      <UploadModal
        open={uploadModalOpen}
        host={uploadHost ? { id: uploadHost.id, syncId: uploadHost.id, label: uploadHost.label || uploadHost.hostname, hostname: uploadHost.hostname, username: uploadHost.username, port: uploadHost.port, group: uploadHost.group ?? '', tags: uploadHost.tags ?? [], authMode: (uploadHost.authMode as 'key' | 'password' | 'none') ?? 'none', lastConnectedAt: uploadHost.lastConnectedAt != null ? new Date(uploadHost.lastConnectedAt * 1000).toISOString() : null } : null}
        localPath={uploadLocalPath}
        onClose={() => { setUploadModalOpen(false); setUploadLocalPath(''); setUploadHost(null); }}
        onConfirm={(local, remote, opts) => { void handleUploadConfirm(local, remote, opts); }}
      />

      <DownloadModal
        open={downloadModalOpen}
        host={downloadHost ? { id: downloadHost.id, syncId: downloadHost.id, label: downloadHost.label || downloadHost.hostname, hostname: downloadHost.hostname, username: downloadHost.username, port: downloadHost.port, group: downloadHost.group ?? '', tags: downloadHost.tags ?? [], authMode: (downloadHost.authMode as 'key' | 'password' | 'none') ?? 'none', lastConnectedAt: downloadHost.lastConnectedAt != null ? new Date(downloadHost.lastConnectedAt * 1000).toISOString() : null } : null}
        onClose={() => { setDownloadModalOpen(false); setDownloadHost(null); }}
        onConfirm={(hostId, remotePath, localPath, opts) => {
          const label = downloadHost ? (downloadHost.label.trim() || downloadHost.hostname) : hostId;
          void transfers.startDownload(hostId, label, localPath, remotePath, opts.recursive, opts.preserve);
        }}
      />

      <TransferTray
        transfers={transfers.transfers}
        onDismiss={transfers.dismiss}
        onCancel={(id) => { void transfers.cancelTransfer(id); }}
        onClearFinished={transfers.clearFinished}
      />

      {/* Connect-config error modal */}
      <Modal
        open={!!credErrorModal}
        onClose={() => setCredErrorModal(null)}
        title="Credential not configured"
        maxWidth={440}
        footer={
          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
            <button type="button" className="btn btn--ghost" onClick={() => setCredErrorModal(null)}>
              Close
            </button>
          </div>
        }
      >
        {credErrorModal && (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 10, fontSize: 13 }}>
            {credErrorModal.kind === 'personal_credential_not_configured' ? (
              <>
                <p style={{ margin: 0, lineHeight: 1.55 }}>
                  You haven&apos;t set a personal credential for{' '}
                  <strong>{credErrorModal.hostLabel}</strong>.
                </p>
                <p style={{ margin: 0, lineHeight: 1.55, color: 'var(--muted)', fontSize: 12 }}>
                  Open the credential roster (View credential roster) and click{' '}
                  <strong>Set credential</strong> on your row to add your credential.
                </p>
              </>
            ) : (
              <p style={{ margin: 0, lineHeight: 1.55 }}>
                The shared credential for <strong>{credErrorModal.hostLabel}</strong> has
                not been configured. An admin needs to edit the host and provide the shared
                credential.
              </p>
            )}
          </div>
        )}
      </Modal>
    </div>
  );
}
