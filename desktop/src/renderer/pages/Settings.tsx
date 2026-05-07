/**
 * Settings — Phase 3D.
 * Sections: Vault, Appearance, SSH Defaults, Sync Provider.
 */
import { useCallback, useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import Button from '../ui/Button';
// Note: useNavigate is retained for the lock vault redirect.
import PasswordField from '../ui/PasswordField';
import Select from '../ui/Select';
import Dialog from '../ui/Dialog';
import { toast } from '../ui/toast';
import { useTheme } from '../hooks/useTheme';
import { useAuth } from '../hooks/useAuth';

const HEALTH_SCHEDULER_KEY = 'health:scheduler:enabled';
const HEALTH_INTERVAL_KEY = 'health:scheduler:intervalMinutes';

function readHealthScheduler(): { enabled: boolean; intervalMinutes: number } {
  try {
    const enabled = localStorage.getItem(HEALTH_SCHEDULER_KEY) === 'true';
    const raw = parseInt(localStorage.getItem(HEALTH_INTERVAL_KEY) ?? '5', 10);
    const intervalMinutes = isNaN(raw) || raw < 1 ? 5 : raw;
    return { enabled, intervalMinutes };
  } catch {
    return { enabled: false, intervalMinutes: 5 };
  }
}

const DEFAULT_SETTINGS: AppSettings = {
  theme: 'dark',
  fontSize: 13,
  termType: 'xterm-256color',
  keepAliveSeconds: 30,
  hostKeyPolicy: 'strict',
  passwordBackend: 'keychain',
  syncProvider: 'off',
};

export default function Settings() {
  const navigate = useNavigate();
  const { theme, setTheme } = useTheme();
  const { session: authSession, refresh: refreshAuth } = useAuth();

  const [settings, setSettings] = useState<AppSettings>(DEFAULT_SETTINGS);
  const [loadError, setLoadError] = useState('');

  // Health scheduler
  const [healthScheduler, setHealthScheduler] = useState(readHealthScheduler);
  const patchHealthScheduler = (patch: Partial<{ enabled: boolean; intervalMinutes: number }>) => {
    const next = { ...healthScheduler, ...patch };
    setHealthScheduler(next);
    try {
      localStorage.setItem(HEALTH_SCHEDULER_KEY, next.enabled ? 'true' : 'false');
      localStorage.setItem(HEALTH_INTERVAL_KEY, String(next.intervalMinutes));
    } catch {
      // ignore
    }
  };

  // Vault change password
  const [oldPw, setOldPw] = useState('');
  const [newPw, setNewPw] = useState('');
  const [confirmPw, setConfirmPw] = useState('');
  const [pwLoading, setPwLoading] = useState(false);
  const [pwError, setPwError] = useState('');

  // Lock vault confirm
  const [lockOpen, setLockOpen] = useState(false);
  const [lockLoading, setLockLoading] = useState(false);

  // Vacuum confirm
  const [vacuumOpen, setVacuumOpen] = useState(false);
  const [vacuumLoading, setVacuumLoading] = useState(false);

  // Sync
  const [syncNowLoading, setSyncNowLoading] = useState(false);
  const [signOutLoading, setSignOutLoading] = useState(false);

  // Updates
  const [daemonVersion, setDaemonVersion] = useState('');
  const [checkUpdateLoading, setCheckUpdateLoading] = useState(false);

  const loadSettings = useCallback(async () => {
    try {
      const s = await window.sshthing.getSettings();
      setSettings(s);
    } catch {
      setLoadError('Settings RPC not yet available — showing defaults.');
    }
  }, []);

  useEffect(() => { void loadSettings(); }, [loadSettings]);

  useEffect(() => {
    window.sshthing.daemonVersion()
      .then((v) => setDaemonVersion(v.version))
      .catch(() => setDaemonVersion(''));
  }, []);

  const patchSettings = async (patch: Partial<AppSettings>) => {
    const next = { ...settings, ...patch };
    setSettings(next);
    try {
      await window.sshthing.setSettings(patch);
    } catch {
      // Fail silently; settings saved locally
    }
  };

  const handleThemeChange = async (t: 'light' | 'dark' | 'system') => {
    setTheme(t);
    await patchSettings({ theme: t });
  };

  const handleChangePassword = async () => {
    if (!oldPw || !newPw) { setPwError('Fill in all fields.'); return; }
    if (newPw !== confirmPw) { setPwError('New passwords do not match.'); return; }
    if (newPw.length < 8) { setPwError('Password must be at least 8 characters.'); return; }
    setPwLoading(true);
    setPwError('');
    try {
      await window.sshthing.changeVaultPassword(oldPw, newPw);
      toast.success('Vault password changed');
      setOldPw(''); setNewPw(''); setConfirmPw('');
    } catch (err: unknown) {
      const e = err as Error & { code?: number };
      if (e.code === -32601) {
        setPwError('Change password requires a newer daemon version.');
      } else {
        setPwError(e.message ?? 'Failed to change password');
      }
    } finally {
      setPwLoading(false);
    }
  };

  const handleLock = async () => {
    setLockLoading(true);
    try {
      await window.sshthing.lockVault();
      navigate('/unlock');
    } catch (err: unknown) {
      const e = err as Error;
      toast.error(e.message ?? 'Failed to lock vault');
    } finally {
      setLockLoading(false);
      setLockOpen(false);
    }
  };

  const handleVacuum = async () => {
    setVacuumLoading(true);
    try {
      await window.sshthing.vacuumVault();
      toast.success('Vault vacuumed');
    } catch (err: unknown) {
      const e = err as Error & { code?: number };
      if (e.code === -32601) {
        toast.error('Vacuum requires a newer daemon version.');
      } else {
        toast.error(e.message ?? 'Vacuum failed');
      }
    } finally {
      setVacuumLoading(false);
      setVacuumOpen(false);
    }
  };

  return (
    <div className="page-scroll" style={{ width: '100%' }}>
      <div style={{ maxWidth: 720, margin: '0 auto', width: '100%' }}>
        <div className="settings-page__header">
          <h1 style={{ fontSize: 22, fontWeight: 600, letterSpacing: '-0.01em', margin: 0 }}>Settings</h1>
        </div>
        <div
          style={{
            padding: '8px 32px 32px',
            display: 'flex',
            flexDirection: 'column',
            gap: 24,
            width: '100%',
          }}
        >

      {loadError && (
        <p style={{ color: 'var(--muted)', fontSize: 12, fontStyle: 'italic' }}>{loadError}</p>
      )}

      {/* ---- Vault ---- */}
      <section className="settings-section">
        <div className="settings-section__title">Vault</div>
        <div className="settings-section__body">
          <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
            <PasswordField
              label="Current password"
              value={oldPw}
              onChange={(e) => setOldPw(e.target.value)}
            />
            <PasswordField
              label="New password"
              value={newPw}
              onChange={(e) => setNewPw(e.target.value)}
            />
            <PasswordField
              label="Confirm new password"
              value={confirmPw}
              onChange={(e) => setConfirmPw(e.target.value)}
            />
            {pwError && <p style={{ color: 'var(--danger)', fontSize: 12 }}>{pwError}</p>}
            <Button variant="primary" onClick={handleChangePassword} loading={pwLoading}>
              Change password
            </Button>
          </div>

          <hr style={{ border: 0, borderTop: '1px solid var(--line)', margin: '4px 0' }} />

          <div className="settings-row">
            <div>
              <div className="settings-row__label">Lock vault</div>
              <div className="settings-row__hint">Requires re-entering password on next open.</div>
            </div>
            <Button variant="ghost" onClick={() => setLockOpen(true)}>Lock</Button>
          </div>

          <div className="settings-row">
            <div>
              <div className="settings-row__label">Vacuum</div>
              <div className="settings-row__hint">Reclaim unused space from the encrypted database.</div>
            </div>
            <Button variant="ghost" onClick={() => setVacuumOpen(true)}>Vacuum</Button>
          </div>
        </div>
      </section>

      {/* ---- Appearance ---- */}
      <section className="settings-section">
        <div className="settings-section__title">Appearance</div>
        <div className="settings-section__body">
          <div className="settings-row">
            <div className="settings-row__label">Theme</div>
            <div className="segmented">
              {(['light', 'dark', 'system'] as const).map((t) => (
                <button
                  key={t}
                  type="button"
                  className="segmented__item"
                  aria-selected={theme === t}
                  onClick={() => handleThemeChange(t)}
                  style={{ textTransform: 'capitalize' }}
                >
                  {t}
                </button>
              ))}
            </div>
          </div>

          <div className="settings-row">
            <div>
              <div className="settings-row__label">Font size</div>
              <div className="settings-row__hint">Terminal font size in px.</div>
            </div>
            <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              <button
                type="button"
                className="btn btn--ghost"
                style={{ minHeight: 28, padding: '0 8px' }}
                onClick={() => void patchSettings({ fontSize: Math.max(8, settings.fontSize - 1) })}
              >
                −
              </button>
              <span style={{ fontSize: 13, minWidth: 24, textAlign: 'center' }}>
                {settings.fontSize}
              </span>
              <button
                type="button"
                className="btn btn--ghost"
                style={{ minHeight: 28, padding: '0 8px' }}
                onClick={() => void patchSettings({ fontSize: Math.min(32, settings.fontSize + 1) })}
              >
                +
              </button>
            </div>
          </div>
        </div>
      </section>

      {/* ---- SSH defaults ---- */}
      <section className="settings-section">
        <div className="settings-section__title">SSH Defaults</div>
        <div className="settings-section__body">
          <Select
            label="Terminal type"
            options={[
              { value: 'xterm-256color', label: 'xterm-256color' },
              { value: 'xterm', label: 'xterm' },
              { value: 'vt100', label: 'vt100' },
            ]}
            value={settings.termType}
            onChange={(e) => void patchSettings({ termType: e.target.value })}
          />

          <div className="settings-row">
            <div>
              <div className="settings-row__label">Keep-alive (seconds)</div>
              <div className="settings-row__hint">Send SSH keep-alive every N seconds. 0 = disabled.</div>
            </div>
            <div className="field" style={{ margin: 0 }}>
              <input
                type="number"
                min="0"
                max="3600"
                className="field__input"
                value={settings.keepAliveSeconds}
                onChange={(e) => void patchSettings({ keepAliveSeconds: parseInt(e.target.value, 10) || 0 })}
                style={{ width: 80 }}
              />
            </div>
          </div>

          <Select
            label="Host key policy"
            options={[
              { value: 'strict', label: 'Strict (verify known_hosts)' },
              { value: 'auto_add', label: 'Auto-add (trust on first use)' },
              { value: 'insecure', label: 'Insecure (skip verification)' },
            ]}
            value={settings.hostKeyPolicy}
            onChange={(e) => void patchSettings({ hostKeyPolicy: e.target.value })}
          />

          {navigator.platform.includes('Mac') && (
            <Select
              label="Password backend (macOS)"
              options={[
                { value: 'keychain', label: 'macOS Keychain' },
                { value: 'vault', label: 'SSHThing vault' },
              ]}
              value={settings.passwordBackend}
              onChange={(e) => void patchSettings({ passwordBackend: e.target.value })}
            />
          )}
        </div>
      </section>

      {/* ---- Sync Provider ---- */}
      <section className="settings-section">
        <div className="settings-section__title">Sync Provider</div>
        <div className="settings-section__body">
          <p style={{ color: 'var(--muted)', fontSize: 12, margin: 0 }}>
            Cloud sync and git sync will be available in a future release.
            Sign in via the sign-in button (coming in Phase 4) to enable cloud sync.
          </p>
          <div className="settings-row">
            <div className="settings-row__label">Provider</div>
            <div className="segmented">
              {(['off', 'git', 'cloud'] as const).map((p) => (
                <button
                  key={p}
                  type="button"
                  className="segmented__item"
                  aria-selected={settings.syncProvider === p}
                  onClick={() => void patchSettings({ syncProvider: p })}
                  style={{ textTransform: 'capitalize' }}
                  disabled={p !== 'off'}
                >
                  {p}
                </button>
              ))}
            </div>
          </div>
        </div>
      </section>

      {/* ---- Updates ---- */}
      <section className="settings-section">
        <div className="settings-section__title">Updates</div>
        <div className="settings-section__body">
          <div className="settings-row">
            <div className="settings-row__label">Release channel</div>
            <div className="segmented">
              {(['stable', 'beta'] as const).map((ch) => (
                <button
                  key={ch}
                  type="button"
                  className="segmented__item"
                  aria-selected={settings.releaseChannel === ch}
                  onClick={() => void patchSettings({ releaseChannel: ch })}
                  style={{ textTransform: 'capitalize' }}
                >
                  {ch}
                </button>
              ))}
            </div>
          </div>

          <div className="settings-row">
            <div>
              <div className="settings-row__label">Auto-apply updates</div>
              <div className="settings-row__hint">Restart and install automatically when an update is downloaded.</div>
            </div>
            <button
              type="button"
              role="switch"
              aria-checked={settings.autoApplyUpdates ?? false}
              onClick={() => void patchSettings({ autoApplyUpdates: !(settings.autoApplyUpdates ?? false) })}
              style={{
                width: 40,
                height: 22,
                borderRadius: 11,
                background: (settings.autoApplyUpdates ?? false) ? 'var(--accent)' : 'var(--line)',
                border: 'none',
                cursor: 'pointer',
                position: 'relative',
                flexShrink: 0,
                transition: 'background 0.2s',
              }}
            >
              <span
                style={{
                  position: 'absolute',
                  top: 2,
                  left: (settings.autoApplyUpdates ?? false) ? 20 : 2,
                  width: 18,
                  height: 18,
                  borderRadius: '50%',
                  background: 'white',
                  transition: 'left 0.2s',
                }}
              />
            </button>
          </div>

          <div className="settings-row">
            <div>
              <div className="settings-row__label">Current version</div>
              <div className="settings-row__hint">{daemonVersion || '—'}</div>
            </div>
            <Button
              variant="ghost"
              onClick={() => {
                setCheckUpdateLoading(true);
                window.sshthing.checkForUpdates()
                  .then(() => toast.success('Checking for updates…'))
                  .catch((err: unknown) => {
                    const e = err as Error;
                    toast.error(e.message ?? 'Check failed');
                  })
                  .finally(() => setCheckUpdateLoading(false));
              }}
              loading={checkUpdateLoading}
            >
              Check now
            </Button>
          </div>
        </div>
      </section>

      {/* ---- Health Scheduler ---- */}
      <section className="settings-section">
        <div className="settings-section__title">Health Monitoring</div>
        <div className="settings-section__body">
          <div className="settings-row">
            <div>
              <div className="settings-row__label">Background health probes</div>
              <div className="settings-row__hint">
                Periodically probe all hosts in the background. Off by default.
              </div>
            </div>
            <button
              type="button"
              role="switch"
              aria-checked={healthScheduler.enabled}
              onClick={() => patchHealthScheduler({ enabled: !healthScheduler.enabled })}
              style={{
                width: 40,
                height: 22,
                borderRadius: 11,
                background: healthScheduler.enabled ? 'var(--accent)' : 'var(--line)',
                border: 'none',
                cursor: 'pointer',
                position: 'relative',
                flexShrink: 0,
                transition: 'background 0.2s',
              }}
            >
              <span
                style={{
                  position: 'absolute',
                  top: 2,
                  left: healthScheduler.enabled ? 20 : 2,
                  width: 18,
                  height: 18,
                  borderRadius: '50%',
                  background: 'white',
                  transition: 'left 0.2s',
                }}
              />
            </button>
          </div>

          {healthScheduler.enabled && (
            <div className="settings-row">
              <div>
                <div className="settings-row__label">Probe interval (minutes)</div>
                <div className="settings-row__hint">Minimum 1 minute.</div>
              </div>
              <div className="field" style={{ margin: 0 }}>
                <input
                  type="number"
                  min="1"
                  max="60"
                  className="field__input"
                  value={healthScheduler.intervalMinutes}
                  onChange={(e) =>
                    patchHealthScheduler({ intervalMinutes: Math.max(1, parseInt(e.target.value, 10) || 5) })
                  }
                  style={{ width: 64 }}
                />
              </div>
            </div>
          )}
        </div>
      </section>

      {/* Confirm dialogs */}
      <Dialog
        open={lockOpen}
        onClose={() => setLockOpen(false)}
        title="Lock vault"
        message="Lock the vault now? You will need to enter your password to use SSHThing again."
        confirmLabel="Lock"
        confirmVariant="danger"
        onConfirm={() => void handleLock()}
        loading={lockLoading}
      />

      <Dialog
        open={vacuumOpen}
        onClose={() => setVacuumOpen(false)}
        title="Vacuum vault"
        message="Reclaim unused space from the encrypted database. The vault must be unlocked. This may take a few seconds."
        confirmLabel="Vacuum"
        onConfirm={() => void handleVacuum()}
        loading={vacuumLoading}
      />
      </div>
      </div>
    </div>
  );
}
