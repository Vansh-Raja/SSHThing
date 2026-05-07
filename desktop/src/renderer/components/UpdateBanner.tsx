/**
 * UpdateBanner — a fixed banner below the topbar that appears when
 * electron-updater detects an available update. Shows version info and
 * allows installing once the download is complete.
 */
import { useEffect, useState } from 'react';

export default function UpdateBanner() {
  const [available, setAvailable] = useState(false);
  const [downloaded, setDownloaded] = useState(false);
  const [version, setVersion] = useState('');
  const [dismissed, setDismissed] = useState(false);

  useEffect(() => {
    const unsubAvailable = window.sshthing.onUpdateAvailable?.((info) => {
      setAvailable(true);
      setVersion(info.version);
    });
    const unsubDownloaded = window.sshthing.onUpdateDownloaded?.((info) => {
      setAvailable(true);
      setDownloaded(true);
      setVersion(info.version);
    });
    return () => {
      unsubAvailable?.();
      unsubDownloaded?.();
    };
  }, []);

  if (!available || dismissed) return null;

  return (
    <div
      role="alert"
      style={{
        position: 'fixed',
        top: 'var(--topbar-height, 48px)',
        left: 'var(--rail-width)',
        right: 0,
        zIndex: 200,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        gap: 12,
        padding: '8px 16px',
        background: 'var(--paper-4)',
        borderBottom: '1px solid var(--accent)',
        color: 'var(--ink)',
        fontSize: 13,
      }}
    >
      <span>
        <span style={{ color: 'var(--accent)', fontWeight: 600 }}>Update available:</span>
        {' '}v{version}
      </span>
      <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexShrink: 0 }}>
        <button
          type="button"
          className="btn btn--ghost"
          style={{ height: 28, padding: '0 12px', fontSize: 12 }}
          onClick={() => setDismissed(true)}
        >
          Later
        </button>
        <button
          type="button"
          className="btn btn--primary"
          style={{ height: 28, padding: '0 12px', fontSize: 12 }}
          disabled={!downloaded}
          onClick={() => window.sshthing.installUpdate()}
        >
          {downloaded ? 'Install & Restart' : 'Downloading…'}
        </button>
      </div>
    </div>
  );
}
