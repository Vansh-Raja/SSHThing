/**
 * HealthStats — full resource panel for a host's latest probe.
 * Mirrors the data the TUI renders (status pill, latency, uptime, CPU,
 * memory, disk, GPU). Used inside the HostDetail page.
 */

interface HealthStatsProps {
  result: HealthResult | null;
  probing: boolean;
  onProbe: () => void;
}

function formatBytes(bytes: number | undefined): string {
  if (!bytes || bytes <= 0) return '—';
  if (bytes >= 1_073_741_824) return `${(bytes / 1_073_741_824).toFixed(1)} GB`;
  if (bytes >= 1_048_576) return `${(bytes / 1_048_576).toFixed(1)} MB`;
  return `${(bytes / 1024).toFixed(1)} KB`;
}

function formatUptime(secs: number | undefined): string {
  if (!secs || secs <= 0) return '—';
  const d = Math.floor(secs / 86_400);
  const h = Math.floor((secs % 86_400) / 3_600);
  const m = Math.floor((secs % 3_600) / 60);
  if (d > 0) return `${d}d ${h}h`;
  if (h > 0) return `${h}h ${m}m`;
  return `${m}m`;
}

function statusVariant(status: string | undefined): 'online' | 'offline' | 'warn' | 'unknown' {
  switch (status) {
    case 'online':
    case 'ok':
    case 'healthy':
      return 'online';
    case 'offline':
    case 'error':
      return 'offline';
    case 'timeout':
    case 'unsupported':
    case 'auth_failed':
    case 'degraded':
    case 'slow':
      return 'warn';
    default:
      return 'unknown';
  }
}

function memUsedFraction(r: HealthResult): number | null {
  if (!r.memTotalBytes || r.memTotalBytes <= 0) return null;
  const avail = r.memAvailBytes ?? 0;
  const used = Math.max(0, r.memTotalBytes - avail);
  return Math.min(1, used / r.memTotalBytes);
}

function diskUsedFraction(r: HealthResult): number | null {
  if (!r.diskTotalBytes || r.diskTotalBytes <= 0) return null;
  const avail = r.diskAvailBytes ?? 0;
  const used = Math.max(0, r.diskTotalBytes - avail);
  return Math.min(1, used / r.diskTotalBytes);
}

function Bar({ fraction }: { fraction: number }) {
  // Color shifts amber/red as utilisation rises.
  const color =
    fraction >= 0.9 ? 'var(--status-offline)'
    : fraction >= 0.75 ? 'var(--status-warn)'
    : 'var(--accent)';
  return (
    <div
      style={{
        width: '100%',
        height: 4,
        background: 'var(--paper-3)',
        borderRadius: 999,
        overflow: 'hidden',
        marginTop: 6,
      }}
    >
      <div
        style={{
          width: `${(fraction * 100).toFixed(1)}%`,
          height: '100%',
          background: color,
          transition: 'width 240ms ease',
        }}
      />
    </div>
  );
}

interface StatProps {
  label: string;
  value: string;
  hint?: string;
  fraction?: number | null;
}

function Stat({ label, value, hint, fraction }: StatProps) {
  return (
    <div
      style={{
        background: 'var(--paper-2)',
        border: '1px solid var(--line-2)',
        borderRadius: 'var(--radius)',
        padding: '12px 14px',
        display: 'flex',
        flexDirection: 'column',
        gap: 4,
      }}
    >
      <span style={{ fontSize: 11, color: 'var(--muted)' }}>{label}</span>
      <span style={{ fontSize: 18, fontWeight: 500, color: 'var(--ink)', fontVariantNumeric: 'tabular-nums' }}>
        {value}
      </span>
      {hint && (
        <span style={{ fontSize: 11, color: 'var(--muted-2)' }}>{hint}</span>
      )}
      {typeof fraction === 'number' && <Bar fraction={fraction} />}
    </div>
  );
}

export default function HealthStats({ result, probing, onProbe }: HealthStatsProps) {
  const variant = statusVariant(result?.status);
  const onlineLabel =
    variant === 'online' ? 'Healthy'
    : variant === 'offline' ? 'Offline'
    : variant === 'warn' ? result?.status ?? 'Unreachable'
    : 'Unknown';

  return (
    <section style={{ marginTop: 32 }}>
      {/* Header */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 14, marginBottom: 14 }}>
        <h2 style={{ fontSize: 14, fontWeight: 600, color: 'var(--ink)', margin: 0 }}>
          Health
        </h2>
        {result ? (
          <span className={`status-pill status-pill--${variant}`}>
            <span className="status-pill__dot" />
            {onlineLabel}
          </span>
        ) : (
          <span className="status-pill status-pill--unknown">
            <span className="status-pill__dot" />
            No data
          </span>
        )}
        <span style={{ flex: 1 }} />
        <button
          type="button"
          className="btn btn--ghost"
          onClick={onProbe}
          disabled={probing}
          style={{ height: 28, padding: '0 12px', fontSize: 12 }}
        >
          {probing ? 'Probing…' : 'Probe now'}
        </button>
      </div>

      {result?.error && (
        <p style={{ color: 'var(--danger)', fontSize: 12, marginBottom: 12 }}>{result.error}</p>
      )}

      {result && (
        <>
          {/* Stats grid */}
          <div
            style={{
              display: 'grid',
              gridTemplateColumns: 'repeat(auto-fill, minmax(180px, 1fr))',
              gap: 12,
            }}
          >
            <Stat
              label="Latency"
              value={result.latencyMs >= 0 ? `${result.latencyMs} ms` : '—'}
            />
            <Stat
              label="Uptime"
              value={formatUptime(result.uptimeSecs)}
            />
            {typeof result.cpuPercent === 'number' && result.cpuPercent > 0 && (
              <Stat
                label="CPU"
                value={`${result.cpuPercent.toFixed(1)}%`}
                fraction={Math.min(1, result.cpuPercent / 100)}
              />
            )}
            {result.memTotalBytes && result.memTotalBytes > 0 && (
              <Stat
                label="Memory"
                value={(() => {
                  const avail = result.memAvailBytes ?? 0;
                  const used = Math.max(0, result.memTotalBytes - avail);
                  return `${formatBytes(used)} / ${formatBytes(result.memTotalBytes)}`;
                })()}
                hint={(() => {
                  const f = memUsedFraction(result);
                  return f === null ? undefined : `${(f * 100).toFixed(0)}% used`;
                })()}
                fraction={memUsedFraction(result)}
              />
            )}
            {result.diskTotalBytes && result.diskTotalBytes > 0 && (
              <Stat
                label="Disk"
                value={(() => {
                  const avail = result.diskAvailBytes ?? 0;
                  const used = Math.max(0, result.diskTotalBytes - avail);
                  return `${formatBytes(used)} / ${formatBytes(result.diskTotalBytes)}`;
                })()}
                hint={(() => {
                  const f = diskUsedFraction(result);
                  return f === null ? undefined : `${(f * 100).toFixed(0)}% used`;
                })()}
                fraction={diskUsedFraction(result)}
              />
            )}
            {result.gpuPresent && (
              <Stat
                label="GPU"
                value={result.gpuName?.trim() || 'Present'}
              />
            )}
          </div>

          <div style={{ marginTop: 10, fontSize: 11, color: 'var(--muted-2)' }}>
            Last checked {new Date(result.checkedAt).toLocaleString()}
          </div>
        </>
      )}

      {!result && !probing && (
        <p style={{ color: 'var(--muted)', fontSize: 13, marginTop: 4 }}>
          No health data yet. Click <strong style={{ color: 'var(--ink)' }}>Probe now</strong> to run a check.
        </p>
      )}
    </section>
  );
}
