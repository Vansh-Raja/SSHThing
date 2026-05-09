/**
 * TransferTray — bottom-right file-transfer monitor.
 * Collapses to a compact bar when empty; expands automatically when
 * transfers are active. User can manually expand/collapse at any time.
 */
import { useState } from 'react';
import type { TransferEntry } from '../hooks/useTransfers';

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
  const [manuallyExpanded, setManuallyExpanded] = useState(false);
  const hasTransfers = transfers.length > 0;
  const isExpanded = hasTransfers || manuallyExpanded;

  if (!isExpanded) {
    /* Compact collapsed bar */
    return (
      <button
        type="button"
        onClick={() => setManuallyExpanded(true)}
        style={{
          position: 'fixed',
          bottom: 0,
          right: 0,
          background: 'var(--paper-2)',
          border: '1.5px solid var(--line)',
          borderBottom: 'none',
          borderRight: 'none',
          borderTopLeftRadius: 8,
          zIndex: 200,
          padding: '6px 14px',
          display: 'flex',
          alignItems: 'center',
          gap: 8,
          cursor: 'pointer',
          fontSize: 11,
          fontWeight: 600,
          color: 'var(--muted)',
          textTransform: 'uppercase',
          letterSpacing: '0.06em',
        }}
      >
        <span style={{ width: 6, height: 6, borderRadius: '50%', background: 'var(--muted-2)' }} />
        Transfers
      </button>
    );
  }

  const hasFinished = transfers.some((t) => t.status !== 'started');
  const activeCount = transfers.filter((t) => t.status === 'started').length;

  return (
    <div
      style={{
        position: 'fixed',
        bottom: 0,
        right: 0,
        width: 360,
        maxHeight: 320,
        overflowY: 'auto',
        background: 'var(--paper-2)',
        border: '1.5px solid var(--line)',
        borderBottom: 'none',
        borderRight: 'none',
        borderTopLeftRadius: 8,
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
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <span style={{ fontSize: 11, fontWeight: 700, color: 'var(--ink)', textTransform: 'uppercase', letterSpacing: '0.08em' }}>
            Transfers
          </span>
          {hasTransfers && (
            <span style={{
              fontSize: 10,
              fontWeight: 600,
              color: activeCount > 0 ? 'var(--accent)' : 'var(--muted)',
              background: activeCount > 0 ? 'var(--accent-soft)' : 'var(--paper-3)',
              padding: '1px 6px',
              borderRadius: 999,
            }}>
              {activeCount > 0 ? `${activeCount} active` : `${transfers.length} done`}
            </span>
          )}
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          {hasFinished && (
            <button
              type="button"
              onClick={onClearFinished}
              style={{ fontSize: 11, background: 'none', border: 'none', color: 'var(--muted)', cursor: 'pointer' }}
            >
              Clear done
            </button>
          )}
          <button
            type="button"
            onClick={() => setManuallyExpanded(false)}
            style={{
              fontSize: 14,
              background: 'none',
              border: 'none',
              color: 'var(--muted)',
              cursor: 'pointer',
              lineHeight: 1,
              padding: '0 2px',
            }}
            aria-label="Collapse transfers"
            title="Collapse"
          >
            −
          </button>
        </div>
      </div>

      {/* Transfer list or empty state */}
      {transfers.length === 0 ? (
        <div style={{ padding: '24px 16px', textAlign: 'center' }}>
          <span style={{ fontSize: 12, color: 'var(--muted)' }}>No active transfers</span>
          <p style={{ fontSize: 11, color: 'var(--muted-2)', marginTop: 4, marginBottom: 0 }}>
            Uploads and downloads will appear here.
          </p>
        </div>
      ) : (
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
      )}

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
