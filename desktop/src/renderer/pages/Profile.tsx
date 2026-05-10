/**
 * Profile — profile + cloud-account info for the signed-in user.
 *
 * - Pre-sign-in: prompt to sign in (deep-links to /sign-in).
 * - Post-sign-in: avatar, name, email, expiry, current team, sync state,
 *   sign-out button.
 */
import { useCallback, useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../hooks/useAuth';
import { useSyncStatus } from '../hooks/useSyncStatus';
import { useTeams } from '../hooks/useTeams';
import { toast } from '../ui/toast';
import Button from '../ui/Button';
import Dialog from '../ui/Dialog';
import { CheckIcon, CloudOffIcon, SignOutIcon } from '../components/icons';
import { Skeleton } from '../components/Skeleton';

function relativeFromNow(unix?: number): string {
  if (!unix) return '—';
  const now = Date.now() / 1000;
  const diff = unix - now;
  if (diff <= 0) return 'expired';
  const mins = Math.floor(diff / 60);
  if (mins < 60) return `in ${mins}m`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `in ${hrs}h`;
  const days = Math.floor(hrs / 24);
  return `in ${days}d`;
}

function initialsFor(name: string, email: string): string {
  const source = name.trim() || email.split('@')[0] || '·';
  return source
    .split(/[\s._-]+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((s) => s[0]!.toUpperCase())
    .join('');
}

export default function Profile() {
  const navigate = useNavigate();
  const { session, loading, refresh } = useAuth();
  const sync = useSyncStatus();
  // Use the shared teams hook (with localStorage cache) instead of a
  // page-local fetch. This eliminates the brief "you're not part of any
  // team" flash on cold loads.
  const { teams, loading: teamsLoading } = useTeams();

  const [signOutOpen, setSignOutOpen] = useState(false);
  const [signOutLoading, setSignOutLoading] = useState(false);
  const [syncNowLoading, setSyncNowLoading] = useState(false);

  const handleSignOut = useCallback(async () => {
    setSignOutLoading(true);
    try {
      await window.sshthing.authSignOut();
      toast.success('Signed out');
      await refresh();
      navigate('/hosts');
    } catch (err) {
      toast.error((err as Error).message ?? 'Sign-out failed');
    } finally {
      setSignOutLoading(false);
      setSignOutOpen(false);
    }
  }, [navigate, refresh]);

  const handleSyncNow = useCallback(async () => {
    setSyncNowLoading(true);
    try {
      const r = await window.sshthing.syncNow();
      if (r.success) {
        toast.success(r.message ?? 'Sync complete');
      } else {
        toast.error(r.message ?? 'Sync failed');
      }
    } catch (err) {
      toast.error((err as Error).message ?? 'Sync failed');
    } finally {
      setSyncNowLoading(false);
    }
  }, []);

  if (loading) {
    return (
      <div className="page-scroll" style={{ width: '100%' }}>
        <div style={{ maxWidth: 560, padding: '36px 32px', display: 'flex', flexDirection: 'column', gap: 20 }}>
          {/* Avatar + name row */}
          <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
            <Skeleton width={52} height={52} style={{ borderRadius: '50%', flexShrink: 0 }} />
            <div style={{ flex: 1, display: 'flex', flexDirection: 'column', gap: 8 }}>
              <Skeleton width="50%" height={16} />
              <Skeleton width="65%" height={13} />
            </div>
          </div>
          {/* Info rows */}
          <Skeleton width="100%" height={34} style={{ borderRadius: 'var(--radius)' }} />
          <Skeleton width="100%" height={34} style={{ borderRadius: 'var(--radius)' }} />
          <Skeleton width="70%" height={34} style={{ borderRadius: 'var(--radius)' }} />
        </div>
      </div>
    );
  }

  if (!session) {
    return (
      <div className="page-scroll" style={{ width: '100%' }}>
        <div
          style={{
            padding: '48px 32px',
            maxWidth: 520,
            margin: '0 auto',
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'flex-start',
            gap: 14,
          }}
        >
          <h1 style={{ fontSize: 22, fontWeight: 600, letterSpacing: '-0.01em' }}>Profile</h1>
          <p style={{ color: 'var(--muted)', fontSize: 13, margin: 0 }}>
            Sign in to enable cloud sync, share hosts with teams, and manage automation tokens.
          </p>
          <Button variant="primary" onClick={() => navigate('/sign-in')}>
            Sign in
          </Button>
        </div>
      </div>
    );
  }

  const name = session.userName?.trim() || session.userEmail.split('@')[0]!;
  const initials = initialsFor(session.userName ?? '', session.userEmail);
  const activeTeam = teams.find((t) => t.id === session.currentTeamId);

  return (
    <div className="page-scroll" style={{ width: '100%' }}>
      <div
        style={{
          padding: '32px 36px',
          maxWidth: 760,
          margin: '0 auto',
          display: 'flex',
          flexDirection: 'column',
          gap: 20,
        }}
      >
        {/* Header */}
        <h1 style={{ fontSize: 22, fontWeight: 600, letterSpacing: '-0.01em', margin: 0 }}>Profile</h1>

        {/* Profile card */}
        <section
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 16,
            padding: 18,
            background: 'var(--paper-2)',
            border: '1px solid var(--line-2)',
            borderRadius: 'var(--radius-lg)',
          }}
        >
          <div
            style={{
              width: 56,
              height: 56,
              borderRadius: 999,
              background: 'var(--accent-soft)',
              color: 'var(--accent)',
              display: 'inline-flex',
              alignItems: 'center',
              justifyContent: 'center',
              fontSize: 20,
              fontWeight: 600,
              flexShrink: 0,
            }}
          >
            {initials || '·'}
          </div>
          <div style={{ flex: 1, minWidth: 0 }}>
            <div style={{ fontSize: 16, fontWeight: 600, color: 'var(--ink)' }}>{name}</div>
            <div style={{ fontSize: 13, color: 'var(--muted)' }}>{session.userEmail}</div>
            <div style={{ fontSize: 12, color: 'var(--muted-2)', marginTop: 2 }}>
              Session expires {relativeFromNow(session.expiresAt)}
            </div>
          </div>
          <Button variant="ghost" onClick={() => setSignOutOpen(true)}>
            <SignOutIcon /> Sign out
          </Button>
        </section>

        {/* Active team */}
        <section className="settings-section">
          <div className="settings-section__title">Active team</div>
          <div className="settings-section__body">
            {activeTeam ? (
              <div className="settings-row">
                <div>
                  <div className="settings-row__label">{activeTeam.name}</div>
                  <div className="settings-row__hint">
                    Slug · {activeTeam.slug} · Role · {activeTeam.role ?? 'member'}
                  </div>
                </div>
                <Button variant="ghost" onClick={() => navigate('/teams')}>
                  Manage
                </Button>
              </div>
            ) : teamsLoading && teams.length === 0 ? (
              <p style={{ color: 'var(--muted)', fontSize: 13, margin: 0 }}>
                Loading your teams…
              </p>
            ) : teams.length === 0 ? (
              <p style={{ color: 'var(--muted)', fontSize: 13, margin: 0 }}>
                You're not part of any team yet. Create one or accept an invite from the Teams page.
              </p>
            ) : (
              <div className="settings-row">
                <div>
                  <div className="settings-row__label">Personal vault</div>
                  <div className="settings-row__hint">No team selected.</div>
                </div>
                <Button variant="ghost" onClick={() => navigate('/teams')}>
                  Pick a team
                </Button>
              </div>
            )}
          </div>
        </section>

        {/* Cloud sync */}
        <section className="settings-section">
          <div className="settings-section__title">Cloud sync</div>
          <div className="settings-section__body">
            <div className="settings-row">
              <div>
                <div className="settings-row__label" style={{ display: 'inline-flex', alignItems: 'center', gap: 8 }}>
                  {sync.state === 'ok' && (
                    <span style={{ color: 'var(--success)', display: 'inline-flex', gap: 6, alignItems: 'center' }}>
                      <CheckIcon /> Synced
                    </span>
                  )}
                  {sync.state === 'syncing' && (
                    <span style={{ display: 'inline-flex', gap: 6, alignItems: 'center' }}>
                      <span className="spinner" /> Syncing…
                    </span>
                  )}
                  {sync.state === 'error' && (
                    <span style={{ color: 'var(--danger)', display: 'inline-flex', gap: 6, alignItems: 'center' }}>
                      <CloudOffIcon /> Error
                    </span>
                  )}
                  {sync.state === 'idle' && <span>Local only</span>}
                </div>
                <div className="settings-row__hint">
                  {sync.state === 'idle'
                    ? 'Cloud sync is disabled. Configure it in Settings.'
                    : sync.state === 'error'
                    ? sync.message ?? 'Last sync ended with an error.'
                    : sync.lastSyncedAt
                    ? `Last synced ${new Date(sync.lastSyncedAt * 1000).toLocaleString()}`
                    : 'Up to date.'}
                </div>
              </div>
              <div style={{ display: 'flex', gap: 8 }}>
                <Button variant="ghost" onClick={() => navigate('/settings')}>
                  Configure
                </Button>
                <Button
                  variant="ghost"
                  onClick={handleSyncNow}
                  loading={syncNowLoading}
                  disabled={sync.state === 'idle'}
                >
                  Sync now
                </Button>
              </div>
            </div>
          </div>
        </section>
      </div>

      <Dialog
        open={signOutOpen}
        onClose={() => setSignOutOpen(false)}
        title="Sign out?"
        message="You'll need to sign in again to use cloud sync, teams, and personal vault sync."
        confirmLabel="Sign out"
        confirmVariant="danger"
        onConfirm={() => void handleSignOut()}
        loading={signOutLoading}
      />
    </div>
  );
}
