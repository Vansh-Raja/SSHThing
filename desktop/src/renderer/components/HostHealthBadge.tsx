/**
 * HostHealthBadge — a small status pill shown on host list rows.
 * Shows a colored dot + short label based on the latest probe status.
 */

type HostHealthBadgeProps = {
  result: HealthResult | undefined;
};

function statusColor(status: string): string {
  switch (status) {
    case 'ok':
    case 'healthy':
      return 'var(--success, var(--accent))';
    case 'degraded':
    case 'slow':
      return 'var(--warn, #d97706)';
    case 'unreachable':
    case 'error':
    case 'failed':
      return 'var(--danger)';
    default:
      return 'var(--muted)';
  }
}

function statusLabel(status: string): string {
  switch (status) {
    case 'ok':
    case 'healthy':
      return 'OK';
    case 'degraded':
      return 'Degraded';
    case 'slow':
      return 'Slow';
    case 'unreachable':
      return 'Down';
    case 'error':
    case 'failed':
      return 'Error';
    default:
      return status;
  }
}

export default function HostHealthBadge({ result }: HostHealthBadgeProps) {
  if (!result) return null;

  const color = statusColor(result.status);
  const label = statusLabel(result.status);

  return (
    <span
      title={result.error ?? `${label} — ${result.latencyMs}ms`}
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: 3,
        fontSize: 10,
        fontWeight: 600,
        color: 'var(--muted)',
        padding: '0 4px',
        borderRadius: 2,
        border: '1px solid var(--line)',
        background: 'var(--paper-2)',
        flexShrink: 0,
      }}
    >
      <span
        style={{
          display: 'inline-block',
          width: 6,
          height: 6,
          borderRadius: '50%',
          background: color,
          flexShrink: 0,
        }}
      />
      {label}
    </span>
  );
}
