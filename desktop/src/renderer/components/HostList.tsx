/**
 * HostList — sidebar list of hosts. Supports group filtering, click-to-connect,
 * drag-drop upload, and a context menu (Edit, Reveal, Health probe, Mount,
 * Run command, Delete).
 */
import { useState } from 'react';
import Tag from '../ui/Tag';
import DropdownMenu, { type MenuItemDef } from '../ui/DropdownMenu';
import Dialog from '../ui/Dialog';
import { toast } from '../ui/toast';
import HostHealthBadge from './HostHealthBadge';
import type { HealthMap } from '../hooks/useHealth';

type HostListProps = {
  hosts: HostSummary[];
  activeHostId: string | null;
  selectedGroup: string;
  healthMap?: HealthMap;
  probingSet?: Set<string>;
  onConnect: (host: HostSummary) => void;
  onEdit: (host: HostSummary) => void;
  onReveal: (host: HostSummary) => void;
  onRefresh: () => void;
  onProbe?: (host: HostSummary) => void;
  onMount?: (host: HostSummary) => void;
  onExec?: (host: HostSummary) => void;
  onDropFile?: (host: HostSummary, filePath: string) => void;
};

function relativeTime(isoString: string | null): string {
  if (!isoString) return 'Never';
  const diff = Date.now() - new Date(isoString).getTime();
  const mins = Math.floor(diff / 60_000);
  if (mins < 1) return 'Just now';
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  const days = Math.floor(hrs / 24);
  return `${days}d ago`;
}

export default function HostList({
  hosts,
  activeHostId,
  selectedGroup,
  healthMap,
  probingSet,
  onConnect,
  onEdit,
  onReveal,
  onRefresh,
  onProbe,
  onMount,
  onExec,
  onDropFile,
}: HostListProps) {
  const [deleteTarget, setDeleteTarget] = useState<HostSummary | null>(null);
  const [deleteLoading, setDeleteLoading] = useState(false);
  // dragOverHostId tracks which row is being dragged over for visual feedback.
  const [dragOverHostId, setDragOverHostId] = useState<string | null>(null);

  const filtered = selectedGroup
    ? hosts.filter((h) => h.group === selectedGroup)
    : hosts;

  const handleDelete = async () => {
    if (!deleteTarget) return;
    setDeleteLoading(true);
    try {
      await window.sshthing.deleteHost(deleteTarget.id);
      toast.success('Host deleted');
      setDeleteTarget(null);
      onRefresh();
    } catch (err: unknown) {
      const e = err as Error & { code?: number };
      if (e.code === -32601) {
        toast.error('Delete requires a newer daemon version.');
      } else {
        toast.error(e.message ?? 'Failed to delete host');
      }
    } finally {
      setDeleteLoading(false);
    }
  };

  const menuItems = (host: HostSummary): MenuItemDef[] => {
    const items: MenuItemDef[] = [
      {
        kind: 'item',
        label: 'Edit',
        onClick: () => onEdit(host),
      },
      {
        kind: 'item',
        label: 'Reveal credential',
        onClick: () => onReveal(host),
      },
    ];

    if (onProbe) {
      items.push({
        kind: 'item',
        label: probingSet?.has(host.id) ? 'Probing…' : 'Probe health',
        onClick: () => onProbe(host),
      });
    }

    if (onMount) {
      items.push({
        kind: 'item',
        label: 'Mount',
        onClick: () => onMount(host),
      });
    }

    if (onExec) {
      items.push({
        kind: 'item',
        label: 'Run command',
        onClick: () => onExec(host),
      });
    }

    items.push(
      { kind: 'separator' },
      {
        kind: 'item',
        label: 'Delete',
        danger: true,
        onClick: () => setDeleteTarget(host),
      },
    );
    return items;
  };

  if (filtered.length === 0) {
    return (
      <div className="empty-state" style={{ margin: '8px 0' }}>
        <span className="empty-state__title">No hosts</span>
        <span style={{ fontSize: 12 }}>
          {selectedGroup
            ? `No hosts in group "${selectedGroup}". Add one with the + button.`
            : 'Add your first host with the + button above.'}
        </span>
      </div>
    );
  }

  return (
    <>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 0 }}>
        {filtered.map((host) => {
          const displayName = host.label.trim() || host.hostname;
          const isActive = host.id === activeHostId;
          const isDragOver = dragOverHostId === host.id;
          const healthResult = healthMap?.get(host.id);

          return (
            // Wrap in a div to support drag-drop while keeping button for keyboard/click.
            <div
              key={host.id}
              onDragOver={(e) => {
                e.preventDefault();
                e.stopPropagation();
                setDragOverHostId(host.id);
              }}
              onDragLeave={() => setDragOverHostId(null)}
              onDrop={(e) => {
                e.preventDefault();
                e.stopPropagation();
                setDragOverHostId(null);
                if (onDropFile) {
                  const files = Array.from(e.dataTransfer.files);
                  const first = files[0];
                  if (first) {
                    // Electron exposes the real path via webkitRelativePath or path property.
                    const filePath = (first as File & { path?: string }).path ?? first.name;
                    onDropFile(host, filePath);
                  }
                }
              }}
              style={{
                outline: isDragOver ? '2px solid var(--accent)' : undefined,
                borderRadius: 2,
                marginBottom: 6,
              }}
            >
              <button
                type="button"
                className={`data-row data-row--clickable${isActive ? ' data-row--active' : ''}`}
                onClick={() => onConnect(host)}
                style={{ margin: 0, borderRadius: 2, width: '100%' }}
              >
                <div className="data-row__primary">
                  <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                    <span className="data-row__title">{displayName}</span>
                    {healthResult && (
                      <HostHealthBadge result={healthResult} />
                    )}
                  </div>
                  <span className="data-row__meta">
                    {host.username}@{host.hostname}:{host.port}
                  </span>
                  <div className="data-row__chips">
                    {host.tags.map((tag) => (
                      <Tag key={tag} variant="muted">{tag}</Tag>
                    ))}
                    {host.lastConnectedAt && (
                      <span
                        style={{ fontSize: 10, color: 'var(--muted)', alignSelf: 'center' }}
                      >
                        {relativeTime(host.lastConnectedAt)}
                      </span>
                    )}
                  </div>
                </div>

                <div className="data-row__trail">
                  <DropdownMenu
                    trigger={
                      <button
                        type="button"
                        style={{
                          display: 'inline-flex',
                          alignItems: 'center',
                          justifyContent: 'center',
                          width: 24,
                          height: 24,
                          background: 'transparent',
                          border: '1.5px solid transparent',
                          borderRadius: 2,
                          color: 'var(--muted)',
                          fontSize: 16,
                          cursor: 'pointer',
                          lineHeight: 1,
                        }}
                        aria-label="Host actions"
                      >
                        ⋯
                      </button>
                    }
                    items={menuItems(host)}
                  />
                </div>
              </button>
            </div>
          );
        })}
      </div>

      <Dialog
        open={!!deleteTarget}
        onClose={() => setDeleteTarget(null)}
        title="Delete host"
        message={`Are you sure you want to delete "${deleteTarget?.label || deleteTarget?.hostname}"? This cannot be undone.`}
        confirmLabel="Delete"
        confirmVariant="danger"
        onConfirm={handleDelete}
        loading={deleteLoading}
      />
    </>
  );
}
