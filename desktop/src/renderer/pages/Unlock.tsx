import { useEffect, useRef, useState, type FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import PasswordField from '../ui/PasswordField';
import Button from '../ui/Button';
import { setVaultSalt } from '../hooks/useHostsCache';

type PageMode = 'unlock' | 'create' | 'biometric' | 'biometric-failed' | 'offer-touchid';

interface BiometricStatus {
  available: boolean;
  enabled: boolean;
  expiresAt: number;
  expired: boolean;
}

export default function Unlock() {
  const [mode, setMode] = useState<PageMode>('unlock');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [bio, setBio] = useState<BiometricStatus | null>(null);
  // Cache the password briefly *only* to power the post-unlock "Enable Touch ID?"
  // prompt. Cleared as soon as the user enables (or skips) the prompt.
  const pendingPasswordRef = useRef<string>('');
  const navigate = useNavigate();

  const goPostUnlock = async () => {
    try {
      const { hosts } = await window.sshthing.listHosts();
      const hasSeen = localStorage.getItem('sshthing-welcome-shown') === 'true';
      if (hosts.length === 0 && !hasSeen) {
        navigate('/welcome');
      } else {
        navigate('/hosts');
      }
    } catch {
      navigate('/hosts');
    }
  };

  // ── On mount: check biometric status and auto-trigger the prompt if eligible.
  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const status = await window.sshthing.biometricStatus();
        if (cancelled) return;
        setBio(status);
        if (status.available && status.enabled && !status.expired) {
          // Auto-fire the Touch ID prompt. Show a placeholder UI while the
          // macOS dialog is up.
          setMode('biometric');
          try {
            const r = await window.sshthing.unlockWithBiometric();
            if (cancelled) return;
            if (r?.salt) setVaultSalt(r.salt);
            await goPostUnlock();
          } catch (err: unknown) {
            if (cancelled) return;
            const e = err as Error & { code?: number; data?: { kind?: string } };
            const kind = e.data?.kind ?? '';
            // Cancellation falls back silently; everything else shows password.
            if (kind === 'biometric_cancelled' || kind === 'biometric_auth_failed') {
              setMode('biometric-failed');
            } else {
              // unavailable / not_found / stale / etc — drop into password mode.
              setMode('unlock');
            }
          }
        }
      } catch {
        // Daemon may not yet support biometric RPCs in older builds.
        // Just ignore and present the password screen as today.
      }
    })();
    return () => { cancelled = true; };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const handleUnlock = async (e: FormEvent) => {
    e.preventDefault();
    if (!password) return;
    setLoading(true);
    setError('');
    try {
      const r = await window.sshthing.unlock(password);
      if (r?.salt) setVaultSalt(r.salt);
      // After a successful password unlock, offer to enable Touch ID
      // (only when the hardware is available and the feature isn't already on).
      if (bio?.available && !bio.enabled) {
        pendingPasswordRef.current = password;
        setMode('offer-touchid');
        setLoading(false);
        return;
      }
      await goPostUnlock();
    } catch (err: unknown) {
      const e2 = err as Error & { code?: number };
      if (e2.code === -32010) {
        setError('Invalid password. Please try again.');
      } else if (e2.code === -32011) {
        setMode('create');
        setPassword('');
        setError('');
      } else {
        setError(e2.message ?? String(err));
      }
    } finally {
      setLoading(false);
    }
  };

  const handleCreate = async (e: FormEvent) => {
    e.preventDefault();
    if (!password) return;
    if (password.length < 8) {
      setError('Password must be at least 8 characters.');
      return;
    }
    if (password !== confirmPassword) {
      setError('Passwords do not match.');
      return;
    }
    setLoading(true);
    setError('');
    try {
      await window.sshthing.createVault(password);
      const r = await window.sshthing.unlock(password);
      if (r?.salt) setVaultSalt(r.salt);
      if (bio?.available && !bio.enabled) {
        pendingPasswordRef.current = password;
        setMode('offer-touchid');
        setLoading(false);
        return;
      }
      await goPostUnlock();
    } catch (err: unknown) {
      const e2 = err as Error & { code?: number };
      if (e2.code === -32601) {
        setError('Vault creation requires a newer daemon version.');
      } else {
        setError(e2.message ?? String(err));
      }
    } finally {
      setLoading(false);
    }
  };

  const handleEnableTouchID = async () => {
    const pw = pendingPasswordRef.current;
    pendingPasswordRef.current = '';
    setLoading(true);
    try {
      await window.sshthing.enableBiometric(pw);
    } catch {
      // Don't block unlock on biometric setup failure.
    } finally {
      setLoading(false);
    }
    await goPostUnlock();
  };

  const handleSkipTouchID = async () => {
    pendingPasswordRef.current = '';
    await goPostUnlock();
  };

  const handleRetryBiometric = async () => {
    setMode('biometric');
    setError('');
    try {
      const r = await window.sshthing.unlockWithBiometric();
      if (r?.salt) setVaultSalt(r.salt);
      await goPostUnlock();
    } catch {
      setMode('biometric-failed');
    }
  };

  // ── Mode renderers ────────────────────────────────────────────────────

  if (mode === 'biometric') {
    return (
      <CenteredCard
        title="SSHThing"
        subtitle="Touch ID required"
        body={
          <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 14 }}>
            <FingerprintIcon />
            <p style={{ color: 'var(--muted)', fontSize: 13, margin: 0, textAlign: 'center' }}>
              Authenticate to unlock your vault.
            </p>
          </div>
        }
      />
    );
  }

  if (mode === 'biometric-failed') {
    return (
      <CenteredCard
        title="SSHThing"
        subtitle="Touch ID failed"
        body={
          <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 12, width: '100%', maxWidth: 320 }}>
            <p style={{ color: 'var(--muted)', fontSize: 13, margin: 0, textAlign: 'center' }}>
              Try again with Touch ID, or use your password.
            </p>
            <Button variant="primary" onClick={handleRetryBiometric} style={{ width: '100%' }}>
              Try Touch ID again
            </Button>
            <button
              type="button"
              onClick={() => setMode('unlock')}
              style={{ background: 'none', border: 'none', color: 'var(--muted)', fontSize: 12, cursor: 'pointer' }}
            >
              Use password instead
            </button>
          </div>
        }
      />
    );
  }

  if (mode === 'offer-touchid') {
    return (
      <CenteredCard
        title="SSHThing"
        subtitle="Use Touch ID for the next 7 days?"
        body={
          <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 12, width: '100%', maxWidth: 360 }}>
            <FingerprintIcon />
            <p style={{ color: 'var(--muted)', fontSize: 13, margin: 0, textAlign: 'center' }}>
              You won't need to type your password to open SSHThing for the next 7 days.
              You can turn this off anytime in Settings.
            </p>
            <Button variant="primary" onClick={handleEnableTouchID} loading={loading} style={{ width: '100%' }}>
              Enable Touch ID
            </Button>
            <button
              type="button"
              onClick={handleSkipTouchID}
              style={{ background: 'none', border: 'none', color: 'var(--muted)', fontSize: 12, cursor: 'pointer' }}
            >
              Not now
            </button>
          </div>
        }
      />
    );
  }

  if (mode === 'create') {
    return (
      <CenteredCard
        title="SSHThing"
        subtitle="Create a vault password to get started"
        body={
          <form onSubmit={handleCreate} style={{ display: 'flex', flexDirection: 'column', gap: 12, width: '100%', maxWidth: 320 }}>
            <PasswordField
              label="New vault password"
              placeholder="Min 8 characters"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              autoFocus
              autoComplete="new-password"
            />
            <PasswordField
              label="Confirm password"
              placeholder="Repeat password"
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              autoComplete="new-password"
            />
            {error && <p style={{ color: 'var(--danger)', fontSize: 12, margin: 0 }}>{error}</p>}
            <Button type="submit" variant="primary" loading={loading}>Create Vault</Button>
            <button
              type="button"
              onClick={() => { setMode('unlock'); setPassword(''); setConfirmPassword(''); setError(''); }}
              style={{ background: 'none', border: 'none', color: 'var(--muted)', fontSize: 12, cursor: 'pointer', padding: '4px 0' }}
            >
              Already have a vault? Unlock instead
            </button>
          </form>
        }
      />
    );
  }

  // mode === 'unlock'
  return (
    <CenteredCard
      title="SSHThing"
      subtitle="Enter your vault password to unlock"
      body={
        <form onSubmit={handleUnlock} style={{ display: 'flex', flexDirection: 'column', gap: 12, width: '100%', maxWidth: 320 }}>
          <PasswordField
            label="Vault password"
            placeholder="Password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            autoFocus
            autoComplete="current-password"
          />
          {error && <p style={{ color: 'var(--danger)', fontSize: 12, margin: 0 }}>{error}</p>}
          <Button type="submit" variant="primary" loading={loading}>Unlock</Button>
          {bio?.available && bio.enabled && !bio.expired && (
            <button
              type="button"
              onClick={handleRetryBiometric}
              style={{ background: 'none', border: 'none', color: 'var(--accent)', fontSize: 12, cursor: 'pointer', padding: '4px 0' }}
            >
              Use Touch ID
            </button>
          )}
          <button
            type="button"
            onClick={() => { setMode('create'); setPassword(''); setError(''); }}
            style={{ background: 'none', border: 'none', color: 'var(--muted)', fontSize: 12, cursor: 'pointer', padding: '4px 0' }}
          >
            No vault yet? Create one
          </button>
        </form>
      }
    />
  );
}

// ── Tiny presentational helpers ────────────────────────────────────────

function CenteredCard({ title, subtitle, body }: { title: string; subtitle: string; body: React.ReactNode }) {
  return (
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        height: '100%',
        gap: 20,
        padding: 32,
      }}
    >
      <div style={{ textAlign: 'center', marginBottom: 8 }}>
        <h1 style={{ fontSize: 28, fontWeight: 800, letterSpacing: '-0.03em', color: 'var(--ink)', marginBottom: 6 }}>
          {title}
        </h1>
        <p style={{ color: 'var(--muted)', fontSize: 13 }}>{subtitle}</p>
      </div>
      {body}
    </div>
  );
}

function FingerprintIcon() {
  return (
    <svg width="56" height="56" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" style={{ color: 'var(--accent)' }}>
      <path d="M12 11c0 4.5-1 7.5-2.5 9" />
      <path d="M8 13c0-2.2 1.8-4 4-4s4 1.8 4 4" />
      <path d="M12 11v3" />
      <path d="M16 12c0 4-1 6.5-2 8" />
      <path d="M5 9c1-2 3-3.5 5-4" />
      <path d="M19 9c-.7-1.5-1.8-2.7-3-3.5" />
      <path d="M3 13c0-1.5.4-2.9 1-4.2" />
    </svg>
  );
}
