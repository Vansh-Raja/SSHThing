/**
 * Hosts — main page. Three regions (the AppShell handles rail/topbar/
 * bottombar around us):
 *
 *   ┌──────────── sidebar ──────────┬──────── detail ───────┐
 *   │ [groups + host list]          │ host detail or        │
 *   │ + Add group                   │ multi-tab terminal    │
 *   └───────────────────────────────┴───────────────────────┘
 *
 * - Selecting a host shows its detail (label, address, props, actions).
 * - Connecting opens a session that takes over the right pane in tabs.
 * - Closing all tabs returns to the detail view.
 */
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import HostDrawer from '../components/HostDrawer';
import HostDetail from '../components/HostDetail';
import RevealCredentialModal from '../components/RevealCredentialModal';
import TerminalTab, {
  openTerminalSession,
  type TerminalTabData,
} from '../components/TerminalTab';
import Tabs from '../ui/Tabs';
import Dialog from '../ui/Dialog';
import Modal from '../ui/Modal';
import DropdownMenu, { type MenuItemDef } from '../ui/DropdownMenu';
import { toast } from '../ui/toast';
import { useNotifications } from '../hooks/useNotifications';
import { useHealth } from '../hooks/useHealth';
import { useHealthScheduler } from '../hooks/useHealthScheduler';
import { useMounts } from '../hooks/useMounts';
import { useTransfers } from '../hooks/useTransfers';
import MountDrawer from '../components/MountDrawer';
import TransferTray from '../components/TransferTray';
import UploadModal, { type UploadOptions } from '../components/UploadModal';
import DownloadModal from '../components/DownloadModal';
import ExecModal from '../components/ExecModal';
import { PlusIcon } from '../components/icons';
import { SkeletonRows } from '../components/Skeleton';

let tabCounter = 0;
function newTabId(): string {
  return `tab-${++tabCounter}`;
}

interface AdoptSessionDetail {
  hostId: string;
  sessionId: string;
  label: string;
}

type SortOption = 'recent' | 'az';

export default function Hosts() {
  const [hosts, setHosts] = useState<HostSummary[]>([]);
  const [groups, setGroups] = useState<GroupSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [sortBy, setSortBy] = useState<SortOption>('recent');

  // Selected host (left side selection — shown in the detail pane when no
  // terminal sessions are active for that host).
  const [selectedHostId, setSelectedHostId] = useState<string | null>(null);

  // Drawer state
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [editingHost, setEditingHost] = useState<HostSummary | null>(null);

  // Reveal modal
  const [revealHostId, setRevealHostId] = useState<string | null>(null);
  const [revealHostLabel, setRevealHostLabel] = useState('');

  // Tabs (multi-terminal)
  const [tabs, setTabs] = useState<TerminalTabData[]>([]);
  const [activeTabId, setActiveTabId] = useState<string>('');

  // Group rename / delete state
  const [renameGroupTarget, setRenameGroupTarget] = useState<string | null>(null);
  const [renameGroupValue, setRenameGroupValue] = useState('');
  const [renameGroupLoading, setRenameGroupLoading] = useState(false);
  const [deleteGroupTarget, setDeleteGroupTarget] = useState<string | null>(null);
  const [deleteGroupLoading, setDeleteGroupLoading] = useState(false);

  // Phase 6 — health, mounts, transfers, exec
  const health = useHealth();
  useHealthScheduler({ hosts, health });
  const mounts = useMounts();
  const transfers = useTransfers();

  const [mountDrawerOpen, setMountDrawerOpen] = useState(false);
  const [mountTarget, setMountTarget] = useState<HostSummary | null>(null);
  const [execModalOpen, setExecModalOpen] = useState(false);
  const [execTarget, setExecTarget] = useState<HostSummary | null>(null);
  const [uploadModalOpen, setUploadModalOpen] = useState(false);
  const [uploadHost, setUploadHost] = useState<HostSummary | null>(null);
  const [uploadLocalPath, setUploadLocalPath] = useState('');
  const [downloadModalOpen, setDownloadModalOpen] = useState(false);
  const [downloadHost, setDownloadHost] = useState<HostSummary | null>(null);

  useEffect(() => { void health.loadAll(); }, []); // eslint-disable-line react-hooks/exhaustive-deps
  useEffect(() => { void mounts.loadMounts(); }, []); // eslint-disable-line react-hooks/exhaustive-deps

  // ── Data load ───────────────────────────────────────────────────────────
  const loadHosts = useCallback(async () => {
    setLoading(true);
    try {
      const result = await window.sshthing.listHosts();
      setHosts(result.hosts);
      setSelectedHostId((current) => {
        if (current && result.hosts.some((h) => h.id === current)) return current;
        return result.hosts[0]?.id ?? null;
      });
    } catch (err) {
      toast.error((err as Error).message ?? 'Failed to load hosts');
    } finally {
      setLoading(false);
    }
  }, []);

  const loadGroups = useCallback(async () => {
    try {
      const result = await window.sshthing.listGroups();
      setGroups(result.groups);
    } catch {
      // groups RPC may not be ready
    }
  }, []);

  useEffect(() => { void loadHosts(); void loadGroups(); }, [loadHosts, loadGroups]);

  // ── Vault locked / notifications ────────────────────────────────────────
  // NOTE: We keep a hard redirect here rather than showing an inline banner.
  // When the vault locks, all active terminal sessions lose their crypto
  // context and cannot safely resume after re-unlock. A full navigation to
  // /unlock is the safest escape hatch; it tears down the terminal pane and
  // forces the user to reconnect their sessions after unlocking.
  const handleNotification = useCallback((method: string) => {
    if (method === 'vault.locked') {
      window.location.hash = '/unlock';
    }
  }, []);
  useNotifications(handleNotification);

  // ── Connect a host ──────────────────────────────────────────────────────
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

  const connectHost = useCallback(async (host: HostSummary) => {
    let sessionId: string | null = null;
    try {
      sessionId = await openTerminalSession(host, 80, 24);
    } catch {
      return;
    }
    if (!sessionId) return;
    adoptSession(host.id, sessionId, host.label.trim() || host.hostname);
  }, [adoptSession]);

  // External adopt-session (from command bar / shell-level palette).
  useEffect(() => {
    const onAdopt = (e: Event) => {
      const ev = e as CustomEvent<AdoptSessionDetail>;
      adoptSession(ev.detail.hostId, ev.detail.sessionId, ev.detail.label);
    };
    window.addEventListener('sshthing:adopt-session', onAdopt);
    return () => window.removeEventListener('sshthing:adopt-session', onAdopt);
  }, [adoptSession]);

  // ── Tab management ──────────────────────────────────────────────────────
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
    // Mark the session closed so we don't attempt another sessionClose.
    setTabs((prev) => prev.map((t) => (t.id === tabId ? { ...t, sessionId: null } : t)));
    // Auto-close on a clean exit (typing `exit`, `logout`, ssh quit ~. , etc.).
    // Non-zero codes usually mean the connection dropped or the remote shell
    // crashed — leave the tab open so the user can read the error.
    if (exitCode === 0) {
      // Brief delay so the "Connection closed" line is visible before the tab disappears.
      window.setTimeout(() => {
        void handleCloseTab(tabId, null);
      }, 600);
    }
  }, [handleCloseTab]);

  // ── Keyboard shortcuts (only those scoped to the Hosts page) ────────────
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

  // ── Sidebar groups ──────────────────────────────────────────────────────
  const groupedHosts = useMemo(() => {
    const map = new Map<string, HostSummary[]>();
    const orderedGroupNames: string[] = [];
    for (const host of hosts) {
      const key = host.group?.trim() || 'Ungrouped';
      if (!map.has(key)) {
        map.set(key, []);
        orderedGroupNames.push(key);
      }
      map.get(key)!.push(host);
    }
    // Sort hosts inside each group.
    for (const list of map.values()) {
      if (sortBy === 'az') {
        list.sort((a, b) => (a.label || a.hostname).localeCompare(b.label || b.hostname));
      } else {
        // recent: sort by lastConnectedAt desc, null/missing goes to the end
        list.sort((a, b) => {
          const ta = a.lastConnectedAt ? new Date(a.lastConnectedAt).getTime() : 0;
          const tb = b.lastConnectedAt ? new Date(b.lastConnectedAt).getTime() : 0;
          if (tb !== ta) return tb - ta;
          return (a.label || a.hostname).localeCompare(b.label || b.hostname);
        });
      }
    }
    return orderedGroupNames.map((name) => ({ name, hosts: map.get(name)! }));
  }, [hosts, sortBy]);

  const selectedHost = hosts.find((h) => h.id === selectedHostId) ?? null;

  // ── Drag-drop file upload ───────────────────────────────────────────────
  const handleDropFile = useCallback((host: HostSummary, filePath: string) => {
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

  const handleMountMounted = useCallback((summary: MountSummary) => {
    toast.success(`Mounted ${summary.hostname} at ${summary.localPath}`);
    void mounts.loadMounts();
  }, [mounts]);

  const handleAddGroup = useCallback(async () => {
    const name = prompt('Group name?');
    if (!name?.trim()) return;
    try {
      await window.sshthing.createGroup(name.trim());
      await loadGroups();
    } catch (err) {
      toast.error((err as Error).message ?? 'Could not create group');
    }
  }, [loadGroups]);

  // ── Group rename / delete handlers ──────────────────────────────────────
  const openRenameGroup = useCallback((groupName: string) => {
    setRenameGroupTarget(groupName);
    setRenameGroupValue(groupName);
  }, []);

  const handleRenameGroupSubmit = useCallback(async () => {
    if (!renameGroupTarget) return;
    const newName = renameGroupValue.trim();
    if (!newName || newName === renameGroupTarget) {
      setRenameGroupTarget(null);
      return;
    }
    setRenameGroupLoading(true);
    try {
      await window.sshthing.renameGroup(renameGroupTarget, newName);
      toast.success(`Group renamed to "${newName}"`);
      setRenameGroupTarget(null);
      await loadGroups();
      await loadHosts();
    } catch (err) {
      toast.error((err as Error).message ?? 'Could not rename group');
    } finally {
      setRenameGroupLoading(false);
    }
  }, [renameGroupTarget, renameGroupValue, loadGroups, loadHosts]);

  const handleDeleteGroup = useCallback(async () => {
    if (!deleteGroupTarget) return;
    setDeleteGroupLoading(true);
    try {
      await window.sshthing.deleteGroup(deleteGroupTarget);
      toast.success(`Group "${deleteGroupTarget}" deleted`);
      setDeleteGroupTarget(null);
      await loadGroups();
      await loadHosts();
    } catch (err) {
      toast.error((err as Error).message ?? 'Could not delete group');
    } finally {
      setDeleteGroupLoading(false);
    }
  }, [deleteGroupTarget, loadGroups, loadHosts]);

  const groupMenuItems = useCallback((groupName: string): MenuItemDef[] => [
    {
      kind: 'item',
      label: 'Rename group',
      onClick: () => openRenameGroup(groupName),
    },
    { kind: 'separator' },
    {
      kind: 'item',
      label: 'Delete group',
      danger: true,
      onClick: () => setDeleteGroupTarget(groupName),
    },
  ], [openRenameGroup]);

  const sortOptions: { value: SortOption; label: string }[] = [
    { value: 'recent', label: 'Recent' },
    { value: 'az', label: 'A→Z' },
  ];

  // ── Render ──────────────────────────────────────────────────────────────
  return (
    <div className="pane-shell" onDragOver={(e) => e.preventDefault()}>
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
          {loading && (
            <div style={{ padding: '12px 8px' }}>
              <SkeletonRows count={4} />
            </div>
          )}
          {!loading && groupedHosts.length === 0 && (
            <div style={{ padding: 24 }}>
              <div className="empty-state">
                <div className="empty-state__title">No hosts yet</div>
                <div>Click + Add host to get started.</div>
              </div>
            </div>
          )}
          {!loading && groupedHosts.map((g, i) => {
            // "Ungrouped" is a synthetic group — don't offer rename/delete on it.
            const isReal = g.name !== 'Ungrouped';
            return (
              <div key={g.name} className="sidebar__group">
                <div className="sidebar__group-title" style={{ justifyContent: 'space-between' }}>
                  <span style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                    {g.name}
                    <span className="sidebar__group-count">·</span>
                    <span className="sidebar__group-count">{g.hosts.length}</span>
                  </span>
                  {isReal && (
                    <DropdownMenu
                      trigger={
                        <button
                          type="button"
                          aria-label={`Group actions for ${g.name}`}
                          style={{
                            background: 'transparent',
                            border: 'none',
                            color: 'var(--muted-2)',
                            fontSize: 14,
                            cursor: 'pointer',
                            padding: '0 4px',
                            lineHeight: 1,
                            borderRadius: 'var(--radius-sm)',
                          }}
                        >
                          ⋯
                        </button>
                      }
                      items={groupMenuItems(g.name)}
                    />
                  )}
                </div>
                {g.hosts.map((h) => {
                  const status = health.healthMap.get(h.id)?.status ?? 'unknown';
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
                          // Electron's File extends the browser File with a non-standard `path`.
                          const electronFile = file as File & { path?: string };
                          handleDropFile(h, electronFile.path ?? file.name);
                        }
                      }}
                    >
                      <span className={`status-dot ${dotClass}`} />
                      <span className="host-row__label">{h.label.trim() || h.hostname}</span>
                    </button>
                  );
                })}
                {i < groupedHosts.length - 1 && <div className="sidebar__divider" />}
              </div>
            );
          })}
        </div>
        <button type="button" className="sidebar__add" onClick={handleAddGroup}>
          <PlusIcon /> Add group
        </button>
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
          <HostDetail
            host={selectedHost}
            health={health.healthMap.get(selectedHost.id) ?? null}
            probing={health.probing.has(selectedHost.id)}
            mount={mounts.mounts.find((m) => m.hostId === selectedHost.id) ?? null}
            onConnect={() => void connectHost(selectedHost)}
            onSFTP={() => toast.info('SFTP browser coming in v1.1')}
            onMount={() => { setMountTarget(selectedHost); setMountDrawerOpen(true); }}
            onEdit={() => { setEditingHost(selectedHost); setDrawerOpen(true); }}
            onReveal={() => {
              setRevealHostId(selectedHost.id);
              setRevealHostLabel(selectedHost.label.trim() || selectedHost.hostname);
            }}
            onProbe={() => void health.probe(selectedHost.id)}
            onExec={() => { setExecTarget(selectedHost); setExecModalOpen(true); }}
            onDownload={() => { setDownloadHost(selectedHost); setDownloadModalOpen(true); }}
          />
        ) : (
          <div className="detail-empty">
            <div className="detail-empty__title">No host selected</div>
            <div className="detail-empty__hint">Pick a host from the sidebar to see its details.</div>
            <span className="detail-empty__kbd">⌘K to search</span>
          </div>
        )}
      </section>

      {/* Overlays */}
      <HostDrawer
        open={drawerOpen}
        onClose={() => setDrawerOpen(false)}
        groups={groups}
        host={editingHost}
        onSaved={() => { void loadHosts(); void loadGroups(); }}
      />

      <RevealCredentialModal
        open={!!revealHostId}
        hostId={revealHostId}
        hostLabel={revealHostLabel}
        onClose={() => { setRevealHostId(null); setRevealHostLabel(''); }}
      />

      <MountDrawer
        open={mountDrawerOpen}
        host={mountTarget}
        onClose={() => setMountDrawerOpen(false)}
        onMounted={handleMountMounted}
      />

      <ExecModal
        open={execModalOpen}
        host={execTarget}
        allHosts={hosts}
        onClose={() => setExecModalOpen(false)}
      />

      <UploadModal
        open={uploadModalOpen}
        host={uploadHost}
        localPath={uploadLocalPath}
        onClose={() => { setUploadModalOpen(false); setUploadLocalPath(''); setUploadHost(null); }}
        onConfirm={(local, remote, opts) => { void handleUploadConfirm(local, remote, opts); }}
      />

      <DownloadModal
        open={downloadModalOpen}
        host={downloadHost}
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

      {/* Group rename modal */}
      <Modal
        open={!!renameGroupTarget}
        onClose={() => setRenameGroupTarget(null)}
        title="Rename group"
        maxWidth={360}
        footer={
          <div className="modal__actions">
            <button
              type="button"
              className="btn btn--ghost"
              onClick={() => setRenameGroupTarget(null)}
              disabled={renameGroupLoading}
            >
              Cancel
            </button>
            <button
              type="button"
              className="btn btn--primary"
              onClick={() => void handleRenameGroupSubmit()}
              disabled={renameGroupLoading || !renameGroupValue.trim()}
            >
              {renameGroupLoading ? <span className="spinner" /> : 'Rename'}
            </button>
          </div>
        }
      >
        <div className="field">
          <label className="field__label">New name</label>
          <input
            className="field__input"
            type="text"
            autoFocus
            value={renameGroupValue}
            onChange={(e) => setRenameGroupValue(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') void handleRenameGroupSubmit();
              if (e.key === 'Escape') setRenameGroupTarget(null);
            }}
          />
        </div>
      </Modal>

      {/* Group delete confirm */}
      <Dialog
        open={!!deleteGroupTarget}
        onClose={() => setDeleteGroupTarget(null)}
        title="Delete group"
        message={`Delete group "${deleteGroupTarget ?? ''}"? Hosts in this group will become ungrouped.`}
        confirmLabel="Delete"
        confirmVariant="danger"
        onConfirm={() => void handleDeleteGroup()}
        loading={deleteGroupLoading}
      />
    </div>
  );
}
