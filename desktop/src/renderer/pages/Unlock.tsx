import { useState, type FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import PasswordField from '../ui/PasswordField';
import Button from '../ui/Button';

type PageMode = 'unlock' | 'create';

export default function Unlock() {
  const [mode, setMode] = useState<PageMode>('unlock');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const navigate = useNavigate();

  const handleUnlock = async (e: FormEvent) => {
    e.preventDefault();
    if (!password) return;
    setLoading(true);
    setError('');
    try {
      await window.sshthing.unlock(password);
      navigate('/hosts');
    } catch (err: unknown) {
      const e2 = err as Error & { code?: number };
      if (e2.code === -32010) {
        setError('Invalid password. Please try again.');
      } else if (e2.code === -32011) {
        // Vault doesn't exist — switch to create mode.
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
      // After creation, unlock so the app transitions into the unlocked state.
      await window.sshthing.unlock(password);
      navigate('/hosts');
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

  if (mode === 'create') {
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
          <h1
            style={{
              fontSize: 28,
              fontWeight: 800,
              letterSpacing: '-0.03em',
              color: 'var(--ink)',
              marginBottom: 6,
            }}
          >
            SSHThing
          </h1>
          <p style={{ color: 'var(--muted)', fontSize: 13 }}>
            Create a vault password to get started
          </p>
        </div>

        <form
          onSubmit={handleCreate}
          style={{
            display: 'flex',
            flexDirection: 'column',
            gap: 12,
            width: '100%',
            maxWidth: 320,
          }}
        >
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

          {error && (
            <p style={{ color: 'var(--danger)', fontSize: 12, margin: 0 }}>{error}</p>
          )}

          <Button type="submit" variant="primary" loading={loading}>
            Create Vault
          </Button>

          <button
            type="button"
            style={{
              background: 'none',
              border: 'none',
              color: 'var(--muted)',
              fontSize: 12,
              cursor: 'pointer',
              textAlign: 'center',
              padding: '4px 0',
            }}
            onClick={() => { setMode('unlock'); setPassword(''); setConfirmPassword(''); setError(''); }}
          >
            Already have a vault? Unlock instead
          </button>
        </form>
      </div>
    );
  }

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
        <h1
          style={{
            fontSize: 28,
            fontWeight: 800,
            letterSpacing: '-0.03em',
            color: 'var(--ink)',
            marginBottom: 6,
          }}
        >
          SSHThing
        </h1>
        <p style={{ color: 'var(--muted)', fontSize: 13 }}>
          Enter your vault password to unlock
        </p>
      </div>

      <form
        onSubmit={handleUnlock}
        style={{
          display: 'flex',
          flexDirection: 'column',
          gap: 12,
          width: '100%',
          maxWidth: 320,
        }}
      >
        <PasswordField
          label="Vault password"
          placeholder="Password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          autoFocus
          autoComplete="current-password"
        />

        {error && (
          <p style={{ color: 'var(--danger)', fontSize: 12, margin: 0 }}>{error}</p>
        )}

        <Button type="submit" variant="primary" loading={loading}>
          Unlock
        </Button>

        <button
          type="button"
          style={{
            background: 'none',
            border: 'none',
            color: 'var(--muted)',
            fontSize: 12,
            cursor: 'pointer',
            textAlign: 'center',
            padding: '4px 0',
          }}
          onClick={() => { setMode('create'); setPassword(''); setError(''); }}
        >
          No vault yet? Create one
        </button>
      </form>
    </div>
  );
}
