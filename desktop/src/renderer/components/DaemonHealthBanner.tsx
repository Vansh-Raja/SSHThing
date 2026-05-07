/**
 * DaemonHealthBanner — a fixed bottom banner that appears when the sshthing
 * daemon process has exited unexpectedly. Prompts the user to reload the app,
 * which re-spawns the daemon via the Electron main process.
 *
 * Mounted once in AppShell so it is always present in authenticated pages.
 */
import { useEffect, useState } from 'react';

export default function DaemonHealthBanner() {
  const [exited, setExited] = useState(false);

  useEffect(() => {
    const unsub = window.sshthing.onDaemonExited?.(() => setExited(true));
    return () => { unsub?.(); };
  }, []);

  if (!exited) return null;

  return (
    <div
      role="alert"
      style={{
        position: 'fixed',
        bottom: 0,
        left: 'var(--rail-width)',
        right: 0,
        zIndex: 300,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        gap: 12,
        padding: '10px 16px',
        background: 'var(--paper-4)',
        borderTop: '1px solid var(--danger)',
        color: 'var(--ink)',
        fontSize: 13,
      }}
    >
      <span>
        <span style={{ color: 'var(--danger)', fontWeight: 600 }}>Daemon disconnected.</span>
        {' '}Reload to reconnect.
      </span>
      <button
        type="button"
        className="btn btn--primary"
        style={{ flexShrink: 0, height: 28, padding: '0 12px', fontSize: 12 }}
        onClick={() => window.location.reload()}
      >
        Reload
      </button>
    </div>
  );
}
