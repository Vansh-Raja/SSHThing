/**
 * HealthCard — shows the latest health probe result for a host.
 * Includes a "Probe now" button that triggers an on-demand probe.
 */
import Button from '../ui/Button';
import Spinner from '../ui/Spinner';

type HealthCardProps = {
  hostId: string;
  hostLabel: string;
  result: HealthResult | undefined;
  probing: boolean;
  onProbe: (hostId: string) => void;
};

function formatBytes(bytes: number | undefined): string {
  if (!bytes) return '—';
  if (bytes >= 1_073_741_824) return `${(bytes / 1_073_741_824).toFixed(1)} GB`;
  if (bytes >= 1_048_576) return `${(bytes / 1_048_576).toFixed(1)} MB`;
  return `${(bytes / 1024).toFixed(1)} KB`;
}

function formatUptime(secs: number | undefined): string {
  if (!secs) return '—';
  const d = Math.floor(secs / 86400);
  const h = Math.floor((secs % 86400) / 3600);
  const m = Math.floor((secs % 3600) / 60);
  if (d > 0) return `${d}d ${h}h`;
  if (h > 0) return `${h}h ${m}m`;
  return `${m}m`;
}

function statusColor(status: string): string {
  switch (status) {
    case 'ok':
    case 'healthy':
      return 'var(--success, var(--accent))';
    case 'degraded':
    case 'slow':
      return 'var(--warn, #d97706)';
    default:
      return 'var(--danger)';
  }
}

export default function HealthCard({ hostId, hostLabel, result, probing, onProbe }: HealthCardProps) {
  return (
    <div
      style={{
        border: '1.5px solid var(--line)',
        borderRadius: 4,
        padding: 12,
        display: 'flex',
        flexDirection: 'column',
        gap: 8,
        background: 'var(--paper-2)',
      }}
    >
      {/* Header */}
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 8 }}>
        <span style={{ fontSize: 12, fontWeight: 700, color: 'var(--ink)' }}>
          Health — {hostLabel}
        </span>
        <Button
          variant="ghost"
          onClick={() => onProbe(hostId)}
          disabled={probing}
          style={{ minHeight: 26, padding: '0 8px', fontSize: 11 }}
        >
          {probing ? <Spinner size={12} /> : null}
          {probing ? 'Probing…' : 'Probe now'}
        </Button>
      </div>

      {/* No result yet */}
      {!result && !probing && (
        <p style={{ fontSize: 12, color: 'var(--muted)', margin: 0 }}>
          No health data yet. Click "Probe now" to run a check.
        </p>
      )}

      {/* Result */}
      {result && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
          {/* Status + latency */}
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <span
              style={{
                display: 'inline-block',
                width: 8,
                height: 8,
                borderRadius: '50%',
                background: statusColor(result.status),
                flexShrink: 0,
              }}
            />
            <span style={{ fontSize: 12, fontWeight: 700, textTransform: 'capitalize', color: 'var(--ink)' }}>
              {result.status}
            </span>
            <span style={{ fontSize: 11, color: 'var(--muted)' }}>
              {result.latencyMs}ms
            </span>
            <span style={{ fontSize: 11, color: 'var(--muted)', marginLeft: 'auto' }}>
              {new Date(result.checkedAt).toLocaleTimeString()}
            </span>
          </div>

          {/* Error message */}
          {result.error && (
            <p style={{ fontSize: 11, color: 'var(--danger)', margin: 0, wordBreak: 'break-word' }}>
              {result.error}
            </p>
          )}

          {/* Resource metrics grid */}
          {(result.cpuPercent !== undefined || result.memTotalBytes !== undefined || result.diskTotalBytes !== undefined) && (
            <div
              style={{
                display: 'grid',
                gridTemplateColumns: '1fr 1fr',
                gap: '4px 12px',
                marginTop: 4,
              }}
            >
              {result.cpuPercent !== undefined && result.cpuPercent > 0 && (
                <MetricRow label="CPU" value={`${result.cpuPercent.toFixed(1)}%`} />
              )}
              {result.uptimeSecs !== undefined && result.uptimeSecs > 0 && (
                <MetricRow label="Uptime" value={formatUptime(result.uptimeSecs)} />
              )}
              {result.memTotalBytes !== undefined && result.memTotalBytes > 0 && (
                <MetricRow
                  label="RAM"
                  value={`${formatBytes(result.memAvailBytes)} / ${formatBytes(result.memTotalBytes)}`}
                />
              )}
              {result.diskTotalBytes !== undefined && result.diskTotalBytes > 0 && (
                <MetricRow
                  label="Disk"
                  value={`${formatBytes(result.diskAvailBytes)} free`}
                />
              )}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

function MetricRow({ label, value }: { label: string; value: string }) {
  return (
    <div style={{ display: 'flex', alignItems: 'baseline', gap: 6 }}>
      <span style={{ fontSize: 10, color: 'var(--muted)', textTransform: 'uppercase', letterSpacing: '0.08em', fontWeight: 700, minWidth: 36 }}>
        {label}
      </span>
      <span style={{ fontSize: 11, color: 'var(--ink)' }}>
        {value}
      </span>
    </div>
  );
}
