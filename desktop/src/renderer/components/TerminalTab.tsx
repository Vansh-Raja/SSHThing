/**
 * TerminalTab — owns a single xterm.js Terminal + a daemon sessionId.
 * Must be disposed (tab close) via the onClose callback which calls
 * session.close + term.dispose() to prevent memory leaks.
 */
import { useEffect, useRef, useCallback } from 'react';
import { Terminal } from 'xterm';
import { FitAddon } from '@xterm/addon-fit';
import { toast } from '../ui/toast';

export type TerminalTabData = {
  id: string;
  hostId: string;
  hostLabel: string;
  sessionId: string | null;
  title: string;
};

type TerminalTabProps = {
  data: TerminalTabData;
  active: boolean;
  onTitleChange: (tabId: string, title: string) => void;
  onExit: (tabId: string, exitCode: number) => void;
};

function b64ToBytes(b64: string): Uint8Array {
  const raw = atob(b64);
  const bytes = new Uint8Array(raw.length);
  for (let i = 0; i < raw.length; i++) {
    bytes[i] = raw.charCodeAt(i)!;
  }
  return bytes;
}

export default function TerminalTab({
  data,
  active,
  onTitleChange,
  onExit,
}: TerminalTabProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const termRef = useRef<Terminal | null>(null);
  const fitRef = useRef<FitAddon | null>(null);
  const roRef = useRef<ResizeObserver | null>(null);
  const unsubRef = useRef<(() => void) | null>(null);
  // Captured at mount so the unmount cleanup can close the daemon session
  // even if `data.sessionId` has been blanked (e.g. by handleTabExit). Set
  // to null after a clean exit so we don't fire a redundant sessionClose.
  const sessionIdRef = useRef<string | null>(data.sessionId);

  // Initialize the terminal and open the SSH session.
  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    const term = new Terminal({
      fontSize: 13,
      fontFamily: 'var(--font-mono)',
      theme: {
        background: '#0a0a0a',
        foreground: '#f4efe6',
        cursor: '#4ade80',
        black: '#0a0a0a',
        brightBlack: '#555',
      },
      cursorBlink: true,
      allowProposedApi: true,
    });

    const fitAddon = new FitAddon();
    term.loadAddon(fitAddon);
    term.open(container);
    fitAddon.fit();

    termRef.current = term;
    fitRef.current = fitAddon;

    // Track OSC title changes (OSC 1/2).
    term.onTitleChange((title) => {
      onTitleChange(data.id, title);
    });

    const sessionId = data.sessionId;
    if (!sessionId) {
      term.write('\r\nNo session — close this tab.\r\n');
      return;
    }

    // Input: renderer → daemon.
    term.onData((input) => {
      const bytes = Array.from(new TextEncoder().encode(input));
      window.sshthing.sessionWrite(sessionId, bytes).catch(() => {});
    });

    // Notifications: daemon → renderer.
    const unsub = window.sshthing.onNotification((method, params) => {
      const p = params as Record<string, unknown>;
      if (method === 'session.data' && p['sessionId'] === sessionId) {
        const bytes = b64ToBytes(p['b64'] as string);
        termRef.current?.write(bytes);
      } else if (method === 'session.exit' && p['sessionId'] === sessionId) {
        const code = p['exitCode'] as number;
        termRef.current?.write(`\r\n[session exited with code ${code}]\r\n`);
        // Daemon already cleaned up — don't fire a redundant sessionClose
        // from the unmount cleanup (it would just log "session not found").
        sessionIdRef.current = null;
        onExit(data.id, code ?? 0);
      } else if (method === 'session.titleChanged' && p['sessionId'] === sessionId) {
        const title = p['title'] as string;
        if (title) onTitleChange(data.id, title);
      }
    });
    unsubRef.current = unsub;

    // Resize observer for the container.
    const ro = new ResizeObserver(() => {
      if (!active) return;
      try { fitAddon.fit(); } catch { /* container not visible */ }
      window.sshthing.sessionResize(sessionId, term.cols, term.rows).catch(() => {});
    });
    ro.observe(container);
    roRef.current = ro;

    return () => {
      // Cleanup on unmount (tab close triggers unmount via display:none, not remove).
      // We do NOT dispose here — disposal happens in onTabClose to avoid double-dispose.
    };
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []); // mount only — data.sessionId is stable after creation

  // Fit on visibility change (tab switch).
  useEffect(() => {
    if (!active) return;
    requestAnimationFrame(() => {
      try { fitRef.current?.fit(); } catch { /* ignore */ }
    });
  }, [active]);

  // Dispose everything when the component unmounts. This fires for both
  // user-initiated tab close (handleCloseTab → setTabs filter → unmount) AND
  // route navigation (Hosts → Settings unmounts the whole tab tree). Without
  // a sessionClose here, the second case would orphan the daemon session —
  // the SSH process keeps running with no UI listening to its output. The
  // user-initiated path also calls sessionClose explicitly; the daemon's
  // second close just returns "session not found" which is caught below.
  useEffect(() => {
    return () => {
      unsubRef.current?.();
      roRef.current?.disconnect();
      try { termRef.current?.dispose(); } catch { /* ignore */ }
      termRef.current = null;
      fitRef.current = null;
      const sid = sessionIdRef.current;
      if (sid) {
        sessionIdRef.current = null;
        window.sshthing.sessionClose(sid).catch(() => { /* already gone */ });
      }
    };
  }, []);

  return (
    <div
      ref={containerRef}
      style={{
        position: 'absolute',
        inset: 0,
        padding: 8,
        display: active ? 'block' : 'none',
        overflow: 'hidden',
        background: '#0a0a0a',
      }}
    />
  );
}

/**
 * Opens a new SSH session and returns the session ID.
 * Call this before creating a TerminalTab.
 */
export async function openTerminalSession(
  host: HostSummary,
  cols: number,
  rows: number,
  termType = 'xterm-256color',
): Promise<string> {
  try {
    const result = await window.sshthing.openSession(host.id, cols, rows, termType);
    return result.sessionId;
  } catch (err: unknown) {
    const e = err as Error;
    toast.error(`Failed to connect to ${host.label || host.hostname}: ${e.message}`);
    throw err;
  }
}

/**
 * Opens a new team SSH session and returns the session ID.
 * Call this before creating a TerminalTab.
 */
export async function openTeamTerminalSession(
  host: TeamHost,
  cols: number,
  rows: number,
  termType = 'xterm-256color',
): Promise<string> {
  try {
    const result = await window.sshthing.openTeamSession(host.id, cols, rows, termType);
    return result.sessionId;
  } catch (err: unknown) {
    const e = err as Error;
    toast.error(`Failed to connect to ${host.label || host.hostname}: ${e.message}`);
    throw err;
  }
}
