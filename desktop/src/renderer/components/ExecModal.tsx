/**
 * ExecModal — runs a one-shot command on one or more remote hosts and shows output.
 *
 * Modes:
 *  - Single host: shows the result in place with collapsible history panel.
 *  - Multi-host: select multiple hosts, run concurrently (max 4 in-flight),
 *    display a result table.
 */
import { useEffect, useRef, useState } from 'react';
import Modal from '../ui/Modal';
import Button from '../ui/Button';
import Spinner from '../ui/Spinner';
import { useExec } from '../hooks/useExec';
import { useExecHistory, type ExecHistoryEntry } from '../hooks/useExecHistory';

const MAX_CONCURRENT = 4;

type ExecModalProps = {
  open: boolean;
  host: HostSummary | null;
  onClose: () => void;
  /** Optional: all available hosts for multi-select mode. */
  allHosts?: HostSummary[];
};

interface BatchResult {
  hostId: string;
  hostLabel: string;
  exitCode: number | null;
  stdout: string;
  stderr: string;
  durationMs: number;
  error?: string;
}

function formatTime(ts: number): string {
  return new Date(ts).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
}

export default function ExecModal({ open, host, onClose, allHosts }: ExecModalProps) {
  const [cmd, setCmd] = useState('');
  const { loading, result, error, run, reset } = useExec();
  const { history, addEntry, clearHistory } = useExecHistory();
  const outputRef = useRef<HTMLPreElement | null>(null);
  const [historyOpen, setHistoryOpen] = useState(false);

  // Multi-host mode
  const [multiMode, setMultiMode] = useState(false);
  const [selectedHostIds, setSelectedHostIds] = useState<Set<string>>(new Set());
  const [stopOnError, setStopOnError] = useState(false);
  const [batchResults, setBatchResults] = useState<BatchResult[]>([]);
  const [batchRunning, setBatchRunning] = useState(false);

  // Reset on open.
  useEffect(() => {
    if (open) {
      setCmd('');
      reset();
      setBatchResults([]);
      setMultiMode(false);
      setSelectedHostIds(new Set());
      setHistoryOpen(false);
    }
  }, [open, reset]);

  // Scroll output to bottom when result arrives.
  useEffect(() => {
    if (result && outputRef.current) {
      outputRef.current.scrollTop = outputRef.current.scrollHeight;
    }
  }, [result]);

  // When single-host run completes, save to history.
  useEffect(() => {
    if (result && host) {
      addEntry(host.id, host.label.trim() || host.hostname, cmd, result);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [result]);

  const handleRun = () => {
    if (!host || !cmd.trim()) return;
    void run(host.id, cmd.trim());
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
      e.preventDefault();
      if (multiMode) {
        void handleBatchRun();
      } else {
        handleRun();
      }
    }
  };

  const handleClose = () => {
    reset();
    onClose();
  };

  const handleLoadFromHistory = (entry: ExecHistoryEntry) => {
    setCmd(entry.command);
    setHistoryOpen(false);
  };

  const toggleHostSelection = (id: string) => {
    setSelectedHostIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };

  const handleBatchRun = async () => {
    if (!cmd.trim() || selectedHostIds.size === 0) return;
    setBatchRunning(true);
    setBatchResults([]);

    const targets = (allHosts ?? []).filter((h) => selectedHostIds.has(h.id));
    const results: BatchResult[] = [];
    let aborted = false;

    // Process in chunks of MAX_CONCURRENT.
    for (let i = 0; i < targets.length && !aborted; i += MAX_CONCURRENT) {
      const chunk = targets.slice(i, i + MAX_CONCURRENT);
      const chunkResults = await Promise.allSettled(
        chunk.map(async (h): Promise<BatchResult> => {
          const start = Date.now();
          try {
            const res = await window.sshthing.sessionExec(h.id, cmd.trim());
            return {
              hostId: h.id,
              hostLabel: h.label.trim() || h.hostname,
              exitCode: res.exitCode,
              stdout: res.stdout,
              stderr: res.stderr,
              durationMs: res.durationMs,
            };
          } catch (err: unknown) {
            return {
              hostId: h.id,
              hostLabel: h.label.trim() || h.hostname,
              exitCode: null,
              stdout: '',
              stderr: '',
              durationMs: Date.now() - start,
              error: (err as Error).message ?? 'Failed',
            };
          }
        }),
      );

      for (const settled of chunkResults) {
        const r = settled.status === 'fulfilled' ? settled.value : {
          hostId: '',
          hostLabel: '',
          exitCode: null,
          stdout: '',
          stderr: '',
          durationMs: 0,
          error: 'Unexpected rejection',
        };
        results.push(r);
        if (stopOnError && (r.exitCode !== 0 || r.error)) {
          aborted = true;
          break;
        }
      }
    }

    setBatchResults(results);
    setBatchRunning(false);
  };

  const displayName = host ? (host.label.trim() || host.hostname) : '';
  const modalTitle = multiMode ? 'Run command — multiple hosts' : `Run command — ${displayName}`;

  const runDisabled = multiMode
    ? batchRunning || !cmd.trim() || selectedHostIds.size === 0
    : loading || !cmd.trim();

  return (
    <Modal
      open={open}
      onClose={handleClose}
      title={modalTitle}
      maxWidth={600}
      footer={
        <div className="modal__actions">
          <Button variant="ghost" onClick={handleClose}>
            Close
          </Button>
          {allHosts && allHosts.length > 1 && (
            <Button
              variant="ghost"
              onClick={() => { setMultiMode((m) => !m); setBatchResults([]); }}
              disabled={loading || batchRunning}
            >
              {multiMode ? 'Single host' : 'Multi-host…'}
            </Button>
          )}
          <Button
            variant="primary"
            onClick={multiMode ? () => { void handleBatchRun(); } : handleRun}
            disabled={runDisabled}
          >
            {(loading || batchRunning) ? <Spinner size={14} /> : null}
            {(loading || batchRunning) ? 'Running…' : 'Run'}
          </Button>
        </div>
      }
    >
      <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
        {/* Command input */}
        <div>
          <label
            style={{
              display: 'block',
              fontSize: 11,
              color: 'var(--muted)',
              textTransform: 'uppercase',
              letterSpacing: '0.08em',
              fontWeight: 700,
              marginBottom: 4,
            }}
          >
            Command
          </label>
          <textarea
            className="field__input"
            value={cmd}
            onChange={(e) => setCmd(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="e.g. uptime"
            disabled={loading || batchRunning}
            rows={3}
            style={{
              width: '100%',
              resize: 'vertical',
              fontFamily: 'var(--font-mono)',
              fontSize: 12,
              padding: '6px 8px',
              boxSizing: 'border-box',
            }}
          />
          <p style={{ fontSize: 10, color: 'var(--muted)', margin: '2px 0 0' }}>
            Press Cmd+Enter to run
          </p>
        </div>

        {/* Multi-host selection */}
        {multiMode && allHosts && (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <span style={{ fontSize: 11, color: 'var(--muted)', textTransform: 'uppercase', letterSpacing: '0.08em', fontWeight: 700 }}>
                Hosts ({selectedHostIds.size} selected)
              </span>
              <label style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 11 }}>
                <input
                  type="checkbox"
                  checked={stopOnError}
                  onChange={(e) => setStopOnError(e.target.checked)}
                />
                Stop on first error
              </label>
            </div>
            <div style={{ maxHeight: 140, overflowY: 'auto', display: 'flex', flexDirection: 'column', gap: 2 }}>
              {allHosts.map((h) => (
                <label
                  key={h.id}
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 8,
                    padding: '4px 6px',
                    borderRadius: 3,
                    cursor: 'pointer',
                    background: selectedHostIds.has(h.id) ? 'var(--paper-3, var(--paper-2))' : 'transparent',
                  }}
                >
                  <input
                    type="checkbox"
                    checked={selectedHostIds.has(h.id)}
                    onChange={() => toggleHostSelection(h.id)}
                    disabled={batchRunning}
                  />
                  <span style={{ fontSize: 12, fontWeight: 500 }}>{h.label.trim() || h.hostname}</span>
                  <span style={{ fontSize: 11, color: 'var(--muted)' }}>{h.hostname}</span>
                </label>
              ))}
            </div>
          </div>
        )}

        {/* Batch results table */}
        {multiMode && batchResults.length > 0 && (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
            <span style={{ fontSize: 11, color: 'var(--muted)', textTransform: 'uppercase', letterSpacing: '0.08em', fontWeight: 700 }}>
              Results
            </span>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 4, maxHeight: 220, overflowY: 'auto' }}>
              {batchResults.map((r) => (
                <div
                  key={r.hostId}
                  style={{
                    padding: '6px 8px',
                    borderRadius: 4,
                    background: 'var(--paper-2)',
                    border: '1px solid var(--line)',
                    fontSize: 12,
                  }}
                >
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: r.stdout || r.stderr || r.error ? 4 : 0 }}>
                    <span style={{ fontWeight: 600, flex: 1 }}>{r.hostLabel}</span>
                    <span
                      style={{
                        fontSize: 11,
                        fontWeight: 600,
                        color: r.error || r.exitCode !== 0 ? 'var(--danger)' : 'var(--success, var(--accent))',
                      }}
                    >
                      {r.error ? 'error' : `exit ${r.exitCode}`}
                    </span>
                    <span style={{ fontSize: 10, color: 'var(--muted)' }}>{r.durationMs}ms</span>
                  </div>
                  {(r.stdout || r.stderr || r.error) && (
                    <pre
                      style={{
                        margin: 0,
                        fontFamily: 'var(--font-mono)',
                        fontSize: 10,
                        color: r.error ? 'var(--danger)' : 'var(--ink)',
                        whiteSpace: 'pre-wrap',
                        wordBreak: 'break-all',
                        maxHeight: 80,
                        overflowY: 'auto',
                      }}
                    >
                      {r.error ?? (r.stdout || r.stderr)}
                    </pre>
                  )}
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Single-host: error */}
        {!multiMode && error && (
          <p style={{ fontSize: 12, color: 'var(--danger)', margin: 0 }}>{error}</p>
        )}

        {/* Single-host: output */}
        {!multiMode && result && (
          <div>
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 4 }}>
              <span style={{ fontSize: 11, color: 'var(--muted)', textTransform: 'uppercase', letterSpacing: '0.08em', fontWeight: 700 }}>
                Output
              </span>
              <span style={{ fontSize: 10, color: result.exitCode === 0 ? 'var(--success, var(--accent))' : 'var(--danger)' }}>
                exit {result.exitCode} · {result.durationMs}ms
              </span>
            </div>
            <pre
              ref={outputRef}
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: 11,
                background: 'var(--paper-2)',
                border: '1.5px solid var(--line)',
                borderRadius: 2,
                padding: 8,
                margin: 0,
                maxHeight: 240,
                overflowY: 'auto',
                whiteSpace: 'pre-wrap',
                wordBreak: 'break-all',
                userSelect: 'text',
              }}
            >
              {result.stdout || result.stderr || '(no output)'}
              {result.stdout && result.stderr && (
                <span style={{ color: 'var(--danger)' }}>{'\n--- stderr ---\n'}{result.stderr}</span>
              )}
            </pre>
          </div>
        )}

        {/* History panel (single-host mode only) */}
        {!multiMode && history.length > 0 && (
          <div>
            <button
              type="button"
              onClick={() => setHistoryOpen((o) => !o)}
              style={{
                background: 'none',
                border: 'none',
                cursor: 'pointer',
                fontSize: 11,
                color: 'var(--muted)',
                padding: '2px 0',
                textTransform: 'uppercase',
                letterSpacing: '0.08em',
                fontWeight: 700,
                display: 'flex',
                alignItems: 'center',
                gap: 4,
              }}
            >
              <span>{historyOpen ? '▾' : '▸'}</span>
              History ({history.length})
              <button
                type="button"
                onClick={(e) => { e.stopPropagation(); clearHistory(); }}
                style={{
                  background: 'none',
                  border: 'none',
                  cursor: 'pointer',
                  fontSize: 10,
                  color: 'var(--muted)',
                  marginLeft: 4,
                  padding: 0,
                  textTransform: 'none',
                  letterSpacing: 0,
                  fontWeight: 400,
                }}
              >
                Clear
              </button>
            </button>

            {historyOpen && (
              <div style={{ display: 'flex', flexDirection: 'column', gap: 3, marginTop: 4, maxHeight: 180, overflowY: 'auto' }}>
                {history.map((entry) => (
                  <button
                    key={entry.id}
                    type="button"
                    onClick={() => handleLoadFromHistory(entry)}
                    style={{
                      background: 'var(--paper-2)',
                      border: '1px solid var(--line)',
                      borderRadius: 3,
                      padding: '6px 8px',
                      textAlign: 'left',
                      cursor: 'pointer',
                      display: 'flex',
                      flexDirection: 'column',
                      gap: 2,
                    }}
                  >
                    <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                      <span style={{ fontFamily: 'var(--font-mono)', fontSize: 11, fontWeight: 600, flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                        {entry.command}
                      </span>
                      <span style={{ fontSize: 10, color: entry.exitCode === 0 ? 'var(--success, var(--accent))' : 'var(--danger)', flexShrink: 0 }}>
                        exit {entry.exitCode}
                      </span>
                    </div>
                    <span style={{ fontSize: 10, color: 'var(--muted)' }}>
                      {entry.hostLabel} · {formatTime(entry.executedAt)}
                    </span>
                  </button>
                ))}
              </div>
            )}
          </div>
        )}
      </div>
    </Modal>
  );
}
