/**
 * Settings — Phase 3D + Sync depth features.
 * Sections: Vault, Appearance, SSH Defaults, Sync Provider, Updates, Health Monitoring.
 */
import { useCallback, useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import Button from '../ui/Button';
// Note: useNavigate is retained for the lock vault redirect.
import PasswordField from '../ui/PasswordField';
import Select from '../ui/Select';
import TextField from '../ui/TextField';
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
  autoSyncAfterCRUD: false,
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

  // Sync devices
  const [devices, setDevices] = useState<Record<string, unknown>[]>([]);
  const [devicesLoading, setDevicesLoading] = useState(false);
  const [forgetDeviceId, setForgetDeviceId] = useState<string | null>(null);
  const [forgetLoading, setForgetLoading] = useState(false);

  // Sync events
  const [events, setEvents] = useState<Array<{ source: string; action: string; itemType?: string; itemCount?: number; createdAt: number }>>([]);
  const [eventsLoading, setEventsLoading] = useState(false);

  // Git wizard
  const [gitRepoUrl, setGitRepoUrl] = useState('');
  const [gitBranch, setGitBranch] = useState('main');
  const [gitSshKeyPath, setGitSshKeyPath] = useState('');
  const [gitTestLoading, setGitTestLoading] = useState(false);
  const [gitTestResult, setGitTestResult] = useState<{ ok: boolean; message?: string } | null>(null);
  const [gitSaveLoading, setGitSaveLoading] = useState(false);

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

  // Load sync devices and events on mount
  useEffect(() => {
    const loadDevices = async () => {
      setDevicesLoading(true);
      try {
        const res = await window.sshthing.syncDevices();
        setDevices(res.devices ?? []);
      } catch {
        // ignore; stubbed on some builds
      } finally {
        setDevicesLoading(false);
      }
    };
    const loadEvents = async () => {
      setEventsLoading(true);
      try {
        const res = await window.sshthing.syncEvents();
        setEvents(res.events ?? []);
      } catch {
        // ignore; stubbed on some builds
      } finally {
        setEventsLoading(false);
      }
    };
    void loadDevices();
    void loadEvents();
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

  const handleForgetDevice = async () => {
    if (!forgetDeviceId) return;
    setForgetLoading(true);
    try {
      await window.sshthing.syncForgetDevice(forgetDeviceId);
      setDevices((prev) => prev.filter((d) => (d.id ?? d.deviceId) !== forgetDeviceId));
      toast.success('Device removed');
    } catch (err: unknown) {
      const e = err as Error;
      toast.error(e.message ?? 'Failed to remove device');
    } finally {
      setForgetLoading(false);
      setForgetDeviceId(null);
    }
  };

  const handleTestGit = async () => {
    if (!gitRepoUrl.trim()) {
      setGitTestResult({ ok: false, message: 'Repository URL is required.' });
      return;
    }
    setGitTestLoading(true);
    setGitTestResult(null);
    try {
      const res = await window.sshthing.syncTestGit(gitRepoUrl.trim(), gitSshKeyPath.trim());
      setGitTestResult(res);
    } catch (err: unknown) {
      const e = err as Error;
      setGitTestResult({ ok: false, message: e.message ?? 'Connection test failed' });
    } finally {
      setGitTestLoading(false);
    }
  };

  const handleSaveGit = async () => {
    setGitSaveLoading(true);
    try {
      await window.sshthing.syncConfigure({
        provider: 'git',
        repoUrl: gitRepoUrl.trim(),
        branch: gitBranch.trim() || 'main',
        sshKeyPath: gitSshKeyPath.trim(),
      });
      toast.success('Git sync configuration saved');
      await loadSettings();
    } catch (err: unknown) {
      const e = err as Error;
      toast.error(e.message ?? 'Failed to save configuration');
    } finally {
      setGitSaveLoading(false);
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

      {/* ---- Sync ---- */}
      <section className="settings-section">
        <div className="settings-section__title">Sync Provider</div>
        <div className="settings-section__body">
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
                  disabled={p === 'cloud'}
                >
                  {p}
                </button>
              ))}
            </div>
          </div>

          {/* Auto-sync toggle */}
          <div className="settings-row">
            <div>
              <div className="settings-row__label">Auto-sync after changes</div>
              <div className="settings-row__hint">Automatically sync after adding, updating, or removing hosts.</div>
            </div>
            <button
              type="button"
              role="switch"
              aria-checked={settings.autoSyncAfterCRUD ?? false}
              onClick={() => void patchSettings({ autoSyncAfterCRUD: !(settings.autoSyncAfterCRUD ?? false) })}
              style={{
                width: 40,
                height: 22,
                borderRadius: 11,
                background: (settings.autoSyncAfterCRUD ?? false) ? 'var(--accent)' : 'var(--line)',
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
                  left: (settings.autoSyncAfterCRUD ?? false) ? 20 : 2,
                  width: 18,
                  height: 18,
                  borderRadius: '50%',
                  background: 'white',
                  transition: 'left 0.2s',
                }}
              />
            </button>
          </div>

          {/* Git sync wizard */}
          {settings.syncProvider === 'git' && (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 12, borderTop: '1px solid var(--line)', paddingTop: 12 }}>
              <TextField
                label="Repository URL"
                placeholder="git@github.com:user/repo.git"
                value={gitRepoUrl}
                onChange={(e) => setGitRepoUrl(e.target.value)}
              />
              <TextField
                label="Branch"
                placeholder="main"
                value={gitBranch}
                onChange={(e) => setGitBranch(e.target.value)}
              />
              <TextField
                label="SSH key path"
                placeholder="~/.ssh/id_rsa"
                value={gitSshKeyPath}
                onChange={(e) => setGitSshKeyPath(e.target.value)}
                hint={
                  <span>
                    {/* TODO: add a native file picker for SSH key files instead of manual path entry */}
                    Path to the private SSH key used to access the repository.
                  </span>
                }
              />
              <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
                <Button variant="secondary" onClick={handleTestGit} loading={gitTestLoading}>
                  Test connection
                </Button>
                <Button variant="primary" onClick={handleSaveGit} loading={gitSaveLoading}>
                  Save
                </Button>
              </div>
              {gitTestResult && (
                <p
                  style={{
                    color: gitTestResult.ok ? 'var(--success)' : 'var(--danger)',
                    fontSize: 12,
                    margin: 0,
                  }}
                >
                  {gitTestResult.ok ? 'OK' : gitTestResult.message ?? 'Error'}
                </p>
              )}
            </div>
          )}

          {/* Sync Devices */}
          <div style={{ borderTop: '1px solid var(--line)', paddingTop: 12 }}>
            <div style={{ fontSize: 13, fontWeight: 500, marginBottom: 8 }}>Sync Devices</div>
            {devicesLoading ? (
              <p style={{ color: 'var(--muted)', fontSize: 12, margin: 0 }}>Loading…</p>
            ) : devices.length === 0 ? (
              <p style={{ color: 'var(--muted)', fontSize: 12, margin: 0 }}>No devices registered yet.</p>
            ) : (
              <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
                {devices.map((d, idx) => {
                  const deviceId = (d.id ?? d.deviceId ?? `device-${idx}`) as string;
                  const lastSync = d.lastSyncAt ?? d.lastSyncedAt ?? d.lastSyncTime ?? null;
                  return (
                    <div
                      key={deviceId}
                      style={{
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'space-between',
                        gap: 8,
                        padding: '8px 10px',
                        borderRadius: 6,
                        background: 'var(--paper-3)',
                      }}
                    >
                      <div style={{ minWidth: 0 }}>
                        <div style={{ fontSize: 13, fontWeight: 500, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                          {deviceId}
                        </div>
                        {lastSync && (
                          <div style={{ fontSize: 11, color: 'var(--muted)' }}>
                            Last sync: {new Date(lastSync as string | number).toLocaleString()}
                          </div>
                        )}
                      </div>
                      <Button
                        variant="ghost"
                        onClick={() => setForgetDeviceId(deviceId)}
                        style={{ minHeight: 28, padding: '0 8px', fontSize: 12 }}
                      >
                        Remove
                      </Button>
                    </div>
                  );
                })}
              </div>
            )}
          </div>

          {/* Sync History */}
          <div style={{ borderTop: '1px solid var(--line)', paddingTop: 12 }}>
            <div style={{ fontSize: 13, fontWeight: 500, marginBottom: 8 }}>Sync History</div>
            {eventsLoading ? (
              <p style={{ color: 'var(--muted)', fontSize: 12, margin: 0 }}>Loading…</p>
            ) : events.length === 0 ? (
              <p style={{ color: 'var(--muted)', fontSize: 12, margin: 0 }}>No sync events yet.</p>
            ) : (
              <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
                {events.map((e, idx) => (
                  <div
                    key={idx}
                    style={{
                      display: 'flex',
                      flexDirection: 'column',
                      gap: 2,
                      padding: '8px 10px',
                      borderRadius: 6,
                      background: 'var(--paper-3)',
                    }}
                  >
                    <div style={{ fontSize: 11, color: 'var(--muted)' }}>
                      {new Date(e.createdAt).toLocaleString()}
                    </div>
                    <div style={{ fontSize: 13 }}>
                      {e.source} — {e.action}
                      {e.itemType ? ` · ${e.itemType}` : ''}
                      {typeof e.itemCount === 'number' ? ` (${e.itemCount})` : ''}
                    </div>
                  </div>
                ))}
              </div>
            )}
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

      <Dialog
        open={forgetDeviceId !== null}
        onClose={() => setForgetDeviceId(null)}
        title="Remove device"
        message="Remove this device from the sync registry? It can re-register on its next sync."
        confirmLabel="Remove"
        confirmVariant="danger"
        onConfirm={() => void handleForgetDevice()}
        loading={forgetLoading}
      />
      </div>
      </div>
    </div>
  );
}
