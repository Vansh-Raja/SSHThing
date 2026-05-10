/**
 * UpdateBanner — fixed banner below the topbar that appears when the
 * daemon's once-a-week nudge finds a new release.
 *
 * The banner intentionally has no "install" button. Updates run via the
 * `sshthing update` CLI now (so the user can read the plan + confirm
 * outside of a busy app window), so the only actions here are "got it"
 * (sticky-dismiss this version via the daemon's `update.dismissBanner`
 * RPC) and "release notes" (open the GitHub release page).
 */
import { useEffect, useState } from 'react';

interface UpdateAvailable {
  currentVersion: string;
  latestVersion: string;
  latestTag: string;
  releaseUrl?: string;
}

export default function UpdateBanner() {
  const [info, setInfo] = useState<UpdateAvailable | null>(null);

  useEffect(() => {
    const unsub = window.sshthing.onNotification?.((method, params) => {
      if (method !== 'update.available') return;
      const p = params as Partial<UpdateAvailable> | null | undefined;
      if (!p || !p.latestVersion) return;
      setInfo({
        currentVersion: p.currentVersion ?? '',
        latestVersion: p.latestVersion,
        latestTag: p.latestTag ?? '',
        releaseUrl: p.releaseUrl,
      });
    });
    return () => unsub?.();
  }, []);

  if (!info) return null;

  const handleDismiss = () => {
    // Sticky-dismiss in the daemon so this version doesn't re-banner on
    // the next 6h tick. The next *new* release will banner again.
    void window.sshthing.dismissUpdateBanner(info.latestVersion).catch(() => {
      /* fail silently — UI dismisses regardless */
    });
    setInfo(null);
  };

  const handleReleaseNotes = () => {
    if (info.releaseUrl) {
      void window.sshthing.openPath(info.releaseUrl);
    }
  };

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
        {' '}v{info.latestVersion} — run{' '}
        <code
          style={{
            padding: '1px 6px',
            background: 'var(--paper-2)',
            border: '1px solid var(--line)',
            borderRadius: 3,
            fontFamily: 'var(--font-mono)',
          }}
        >
          sshthing update
        </code>
        {' '}from your terminal to install.
      </span>
      <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexShrink: 0 }}>
        {info.releaseUrl && (
          <button
            type="button"
            className="btn btn--ghost"
            style={{ height: 28, padding: '0 12px', fontSize: 12 }}
            onClick={handleReleaseNotes}
          >
            Release notes
          </button>
        )}
        <button
          type="button"
          className="btn btn--ghost"
          style={{ height: 28, padding: '0 12px', fontSize: 12 }}
          onClick={handleDismiss}
        >
          Got it
        </button>
      </div>
    </div>
  );
}
