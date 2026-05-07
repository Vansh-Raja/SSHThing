/**
 * HostDetail — the canonical right-pane view for a selected host.
 * Mirrors the design mockup: title row, address, online status, property
 * list, action buttons.
 */
import { useState } from 'react';
import { CopyIcon, StarIcon, ConnectIcon, FolderIcon, MountIcon } from './icons';
import HealthStats from './HealthStats';
import { toast } from '../ui/toast';

interface HostDetailProps {
  host: HostSummary;
  health: HealthResult | null;
  probing: boolean;
  mount: MountSummary | null;
  onConnect: () => void;
  onSFTP: () => void;
  onMount: () => void;
  onEdit: () => void;
  onReveal: () => void;
  onProbe: () => void;
  onExec: () => void;
  onDownload: () => void;
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

function authLabel(mode: HostSummary['authMode']): string {
  switch (mode) {
    case 'key': return 'SSH key';
    case 'password': return 'Password';
    case 'none': return 'None';
    default: return '—';
  }
}

function statusToVariant(status: string | undefined): 'online' | 'offline' | 'warn' | 'unknown' {
  switch (status) {
    case 'online': return 'online';
    case 'offline':
    case 'error': return 'offline';
    case 'timeout':
    case 'unsupported':
    case 'auth_failed': return 'warn';
    default: return 'unknown';
  }
}

export default function HostDetail({
  host,
  health,
  probing,
  mount,
  onConnect,
  onSFTP,
  onMount,
  onEdit,
  onReveal,
  onProbe,
  onExec,
  onDownload,
}: HostDetailProps) {
  const [favorited, setFavorited] = useState(false); // local-only for now

  const statusVariant = statusToVariant(health?.status);
  const onlineLabel =
    statusVariant === 'online' ? 'Online'
    : statusVariant === 'offline' ? 'Offline'
    : statusVariant === 'warn' ? 'Unreachable'
    : 'Unknown';

  const address = `${host.username}@${host.hostname}${host.port && host.port !== 22 ? `:${host.port}` : ''}`;

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(address);
      toast.success('Address copied');
    } catch {
      toast.error('Could not copy');
    }
  };

  const healthSummary = (() => {
    if (!health) return null;
    const parts: string[] = [];
    parts.push(statusVariant === 'online' ? 'healthy' : (health.status ?? 'unknown'));
    if (typeof health.latencyMs === 'number' && health.latencyMs > 0) parts.push(`${health.latencyMs}ms`);
    if (mount) parts.push(`mounted ${mount.localPath}`);
    return parts.join(' · ');
  })();

  return (
    <div className="detail__scroll">
      <div className="detail__inner">
        {/* Title row */}
        <div className="detail__title-row">
          <h1 className="detail__title">{host.label.trim() || host.hostname}</h1>
          <button
            type="button"
            className={`detail__star${favorited ? ' detail__star--active' : ''}`}
            title={favorited ? 'Unfavorite' : 'Favorite'}
            onClick={() => setFavorited((v) => !v)}
          >
            <StarIcon filled={favorited} width={18} height={18} />
          </button>
        </div>

        {/* Address + copy */}
        <div className="detail__address">
          <span style={{ fontFamily: 'var(--font-mono)' }}>{address}</span>
          <button type="button" className="detail__copy" title="Copy address" onClick={handleCopy}>
            <CopyIcon />
          </button>
        </div>

        {/* Online status pill */}
        <div className="detail__status">
          <span className={`status-pill status-pill--${statusVariant}`}>
            <span className="status-pill__dot" />
            {onlineLabel}
          </span>
        </div>

        {/* Primary actions — kept above the fold so they're always visible. */}
        <div className="detail__actions" style={{ marginTop: 18 }}>
          <button type="button" className="btn btn--primary btn--lg" onClick={onConnect}>
            <ConnectIcon /> Connect
          </button>
          <button type="button" className="btn btn--lg" onClick={onSFTP}>
            <FolderIcon /> SFTP
          </button>
          <button type="button" className="btn btn--lg" onClick={onMount}>
            <MountIcon /> {mount ? 'Unmount' : 'Mount'}
          </button>
          <span style={{ flex: 1 }} />
          <button type="button" className="btn btn--ghost" onClick={onEdit}>Edit</button>
          <button type="button" className="btn btn--ghost" onClick={onReveal}>Reveal credential</button>
        </div>

        <div className="detail__divider" />

        {/* Property list */}
        <div className="detail__props">
          <div className="detail__prop-label">Group</div>
          <div className="detail__prop-value">{host.group?.trim() || '—'}</div>

          <div className="detail__prop-label">Authentication</div>
          <div className="detail__prop-value">{authLabel(host.authMode)}</div>

          <div className="detail__prop-label">Last seen</div>
          <div className="detail__prop-value">{relativeTime(health?.checkedAt ?? host.lastConnectedAt)}</div>

          <div className="detail__prop-label">Created</div>
          <div className="detail__prop-value">{shortDate(host.lastConnectedAt)}</div>

          <div className="detail__prop-label">Tags</div>
          <div className="detail__prop-value">
            {host.tags.length > 0
              ? host.tags.map((t) => <span key={t} className="chip">{t}</span>)
              : <span style={{ color: 'var(--muted-2)' }}>—</span>}
          </div>

          <div className="detail__prop-label">Health</div>
          <div className="detail__prop-value">
            <span className={`status-pill status-pill--${statusVariant}`}>
              <span className="status-pill__dot" />
              {healthSummary ?? 'unknown'}
            </span>
          </div>
        </div>

        {/* Resource stats */}
        <HealthStats result={health} probing={probing} onProbe={onProbe} />

        {/* Tertiary actions — diagnostic/scripting flows live below the
            detail content where they don't compete with Connect. */}
        <div style={{ marginTop: 28, display: 'flex', gap: 10, flexWrap: 'wrap' }}>
          <button type="button" className="btn btn--ghost" onClick={onExec}>Run command…</button>
          <button type="button" className="btn btn--ghost" onClick={onDownload}>Download file…</button>
        </div>
      </div>
    </div>
  );
}
