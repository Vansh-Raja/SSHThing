/**
 * TransferTray — bottom drawer showing active and recently completed file transfers.
 * Uses an indeterminate progress indicator since the daemon does not emit byte counts.
 */
import type { TransferEntry } from '../hooks/useTransfers';
import EmptyState from './EmptyState';

type TransferTrayProps = {
  transfers: TransferEntry[];
  onDismiss: (transferId: string) => void;
  onCancel: (transferId: string) => void;
  onClearFinished: () => void;
};

function statusLabel(status: TransferEntry['status']): string {
  switch (status) {
    case 'started': return 'In progress';
    case 'finished': return 'Done';
    case 'failed': return 'Failed';
  }
}

function statusColor(status: TransferEntry['status']): string {
  switch (status) {
    case 'started': return 'var(--accent)';
    case 'finished': return 'var(--success, var(--accent))';
    case 'failed': return 'var(--danger)';
  }
}

export default function TransferTray({ transfers, onDismiss, onCancel, onClearFinished }: TransferTrayProps) {
  if (transfers.length === 0) {
    return (
      <div
        style={{
          position: 'fixed',
          bottom: 0,
          right: 0,
          width: 340,
          background: 'var(--paper-2)',
          border: '1.5px solid var(--line)',
          borderBottom: 'none',
          borderRight: 'none',
          borderTopLeftRadius: 6,
          boxShadow: 'var(--shadow-md, 0 -2px 12px rgba(0,0,0,0.15))',
          zIndex: 200,
          padding: '20px 16px',
        }}
      >
        <EmptyState
          title="No active transfers"
          description="Uploads and downloads will appear here."
        />
      </div>
    );
  }

  const hasFinished = transfers.some((t) => t.status !== 'started');

  return (
    <div
      style={{
        position: 'fixed',
        bottom: 0,
        right: 0,
        width: 340,
        maxHeight: 320,
        overflowY: 'auto',
        background: 'var(--paper-2)',
        border: '1.5px solid var(--line)',
        borderBottom: 'none',
        borderRight: 'none',
        borderTopLeftRadius: 6,
        boxShadow: 'var(--shadow-md, 0 -2px 12px rgba(0,0,0,0.15))',
        zIndex: 200,
      }}
    >
      {/* Tray header */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          padding: '8px 12px',
          borderBottom: '1px solid var(--line)',
          position: 'sticky',
          top: 0,
          background: 'var(--paper-2)',
          zIndex: 1,
        }}
      >
        <span style={{ fontSize: 11, fontWeight: 700, color: 'var(--ink)', textTransform: 'uppercase', letterSpacing: '0.08em' }}>
          Transfers
        </span>
        {hasFinished && (
          <button
            type="button"
            onClick={onClearFinished}
            style={{ fontSize: 11, background: 'none', border: 'none', color: 'var(--muted)', cursor: 'pointer' }}
          >
            Clear done
          </button>
        )}
      </div>

      {/* Transfer list */}
      <div style={{ padding: '6px 0' }}>
        {transfers.map((t) => (
          <div
            key={t.transferId}
            style={{
              padding: '6px 12px',
              display: 'flex',
              flexDirection: 'column',
              gap: 4,
              borderBottom: '1px solid var(--line)',
            }}
          >
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <span style={{ fontSize: 11, fontWeight: 600, color: 'var(--ink)' }}>
                {t.direction === 'upload' ? '↑ Upload' : '↓ Download'} — {t.hostLabel}
              </span>
              <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                <span style={{ fontSize: 10, color: statusColor(t.status), fontWeight: 600 }}>
                  {statusLabel(t.status)}
                </span>
                {t.status === 'started' ? (
                  <button
                    type="button"
                    onClick={() => onCancel(t.transferId)}
                    style={{
                      background: 'none',
                      border: 'none',
                      color: 'var(--muted)',
                      cursor: 'pointer',
                      fontSize: 11,
                      lineHeight: 1,
                      padding: '2px 6px',
                      borderRadius: 3,
                    }}
                    aria-label="Cancel transfer"
                  >
                    Cancel
                  </button>
                ) : (
                  <button
                    type="button"
                    onClick={() => onDismiss(t.transferId)}
                    style={{
                      background: 'none',
                      border: 'none',
                      color: 'var(--muted)',
                      cursor: 'pointer',
                      fontSize: 14,
                      lineHeight: 1,
                      padding: 0,
                    }}
                    aria-label="Dismiss"
                  >
                    ×
                  </button>
                )}
              </div>
            </div>

            {/* File path */}
            <span
              style={{
                fontSize: 10,
                color: 'var(--muted)',
                fontFamily: 'var(--font-mono)',
                overflow: 'hidden',
                textOverflow: 'ellipsis',
                whiteSpace: 'nowrap',
              }}
              title={`${t.local} → ${t.remote}`}
            >
              {t.direction === 'upload' ? t.local : t.remote}
            </span>

            {/* Indeterminate progress bar (in progress) */}
            {t.status === 'started' && (
              <div
                style={{
                  height: 3,
                  background: 'var(--line)',
                  borderRadius: 2,
                  overflow: 'hidden',
                }}
              >
                <div
                  style={{
                    height: '100%',
                    background: 'var(--accent)',
                    animation: 'transferIndeterminate 1.4s ease-in-out infinite',
                    borderRadius: 2,
                  }}
                />
              </div>
            )}

            {/* Error */}
            {t.error && (
              <span style={{ fontSize: 10, color: 'var(--danger)' }}>{t.error}</span>
            )}
          </div>
        ))}
      </div>

      {/* CSS animation injected once */}
      <style>{`
        @keyframes transferIndeterminate {
          0% { transform: translateX(-100%) scaleX(0.3); }
          50% { transform: translateX(50%) scaleX(0.5); }
          100% { transform: translateX(200%) scaleX(0.3); }
        }
      `}</style>
    </div>
  );
}
