/**
 * TeamHostDrawer — slide-out form for creating/editing a team host.
 * Gated by canManageHosts permission.
 */
import { useCallback, useEffect, useState } from 'react';
import Drawer from '../../ui/Drawer';
import { toast } from '../../ui/toast';

type TeamHostDrawerProps = {
  open: boolean;
  onClose: () => void;
  teamId: string;
  host: TeamHost | null;
  onSaved: () => void;
};

const CREDENTIAL_MODES = [
  { value: 'shared', label: 'Shared (one credential for all members)' },
  { value: 'per_member', label: 'Per member (each member sets their own)' },
];

const CREDENTIAL_TYPES = [
  { value: 'none', label: 'None' },
  { value: 'password', label: 'Password' },
  { value: 'private_key', label: 'Private key' },
];

const SECRET_VISIBILITY = [
  { value: 'admin_only', label: 'Admins only' },
  { value: 'members', label: 'All members' },
];

export default function TeamHostDrawer({ open, onClose, teamId, host, onSaved }: TeamHostDrawerProps) {
  const isEdit = !!host;

  const [label, setLabel] = useState('');
  const [hostname, setHostname] = useState('');
  const [username, setUsername] = useState('');
  const [port, setPort] = useState('22');
  const [group, setGroup] = useState('');
  const [notes, setNotes] = useState('');
  const [credentialMode, setCredentialMode] = useState('per_member');
  const [credentialType, setCredentialType] = useState('none');
  const [secretVisibility, setSecretVisibility] = useState('admin_only');
  const [sharedCredential, setSharedCredential] = useState('');
  const [clearSharedCredential, setClearSharedCredential] = useState(false);
  const [saving, setSaving] = useState(false);
  // True after the user has clicked "View existing" on edit mode and the
  // stored shared credential has been pulled into `sharedCredential`. Used
  // to label the action button + skip a redundant fetch.
  const [revealingShared, setRevealingShared] = useState(false);
  const [sharedRevealed, setSharedRevealed] = useState(false);

  useEffect(() => {
    if (open) {
      if (host) {
        setLabel(host.label);
        setHostname(host.hostname);
        setUsername(host.username);
        setPort(String(host.port));
        setGroup(host.group ?? '');
        setNotes(host.notes ?? '');
        setCredentialMode(host.credentialMode ?? 'per_member');
        setCredentialType(host.credentialType ?? 'none');
        setSecretVisibility(host.secretVisibility ?? 'admin_only');
        setSharedCredential('');
        setClearSharedCredential(false);
        setSharedRevealed(false);
      } else {
        setLabel('');
        setHostname('');
        setUsername('');
        setPort('22');
        setGroup('');
        setNotes('');
        setCredentialMode('per_member');
        setCredentialType('none');
        setSecretVisibility('admin_only');
        setSharedCredential('');
        setClearSharedCredential(false);
        setSharedRevealed(false);
      }
    }
  }, [open, host]);

  // Reveal the existing shared credential into the form. Audit-logged on
  // the daemon side. Only meaningful on edit + shared mode + caller has
  // canRevealSecrets (button is gated below).
  const handleRevealShared = useCallback(async () => {
    if (!host) return;
    setRevealingShared(true);
    try {
      const res = await window.sshthing.teamsHostsRevealShared(host.id);
      if (res.secret) {
        setSharedCredential(res.secret);
        setSharedRevealed(true);
        if (res.credentialType) setCredentialType(res.credentialType);
      } else {
        toast.info('No shared credential is currently stored');
      }
    } catch (err) {
      const e = err as Error;
      toast.error(e.message ?? 'Could not reveal existing shared credential');
    } finally {
      setRevealingShared(false);
    }
  }, [host]);

  const handleSubmit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      const portNum = parseInt(port, 10);
      if (!hostname.trim()) {
        toast.error('Hostname is required');
        return;
      }
      if (isNaN(portNum) || portNum < 1 || portNum > 65535) {
        toast.error('Port must be between 1 and 65535');
        return;
      }
      setSaving(true);
      try {
        if (isEdit && host) {
          const req: UpdateTeamHostRequest = {
            label: label.trim(),
            hostname: hostname.trim(),
            username: username.trim(),
            port: portNum,
            group: group.trim() || undefined,
            notes: notes.trim() || undefined,
            credentialMode,
            credentialType,
            secretVisibility,
            sharedCredential: sharedCredential.trim() || undefined,
            clearSharedCredential: clearSharedCredential || undefined,
          };
          await window.sshthing.teamsHostsUpdate(host.id, req);
          toast.success('Host updated');
        } else {
          const req: CreateTeamHostRequest = {
            label: label.trim(),
            hostname: hostname.trim(),
            username: username.trim(),
            port: portNum,
            group: group.trim() || undefined,
            notes: notes.trim() || undefined,
            credentialMode,
            credentialType,
            secretVisibility,
            sharedCredential: sharedCredential.trim() || undefined,
          };
          await window.sshthing.teamsHostsCreate(teamId, req);
          toast.success('Host created');
        }
        onSaved();
        onClose();
      } catch (err: unknown) {
        toast.error((err as Error).message ?? 'Failed to save host');
      } finally {
        setSaving(false);
      }
    },
    [isEdit, host, teamId, label, hostname, username, port, group, notes, credentialMode, credentialType, secretVisibility, sharedCredential, clearSharedCredential, onSaved, onClose],
  );

  return (
    <Drawer
      open={open}
      onClose={onClose}
      title={isEdit ? 'Edit Host' : 'Add Host'}
      footer={
        <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
          <button type="button" className="btn btn--ghost" onClick={onClose} disabled={saving}>
            Cancel
          </button>
          <button
            type="button"
            className="btn btn--primary"
            onClick={(e) => void handleSubmit(e as unknown as React.FormEvent)}
            disabled={saving}
          >
            {saving ? <span className="spinner" /> : isEdit ? 'Save' : 'Add host'}
          </button>
        </div>
      }
    >
      <form onSubmit={(e) => void handleSubmit(e)} style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
        <div className="field">
          <label className="field__label">Label</label>
          <input
            className="field__input"
            type="text"
            placeholder="My server"
            value={label}
            onChange={(e) => setLabel(e.target.value)}
          />
        </div>
        <div style={{ display: 'flex', gap: 10 }}>
          <div className="field" style={{ flex: 1 }}>
            <label className="field__label">Hostname</label>
            <input
              className="field__input"
              type="text"
              placeholder="example.com"
              value={hostname}
              onChange={(e) => setHostname(e.target.value)}
              required
            />
          </div>
          <div className="field" style={{ width: 80 }}>
            <label className="field__label">Port</label>
            <input
              className="field__input"
              type="number"
              min={1}
              max={65535}
              value={port}
              onChange={(e) => setPort(e.target.value)}
            />
          </div>
        </div>
        <div className="field">
          <label className="field__label">Username</label>
          <input
            className="field__input"
            type="text"
            placeholder="root"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
          />
        </div>
        <div className="field">
          <label className="field__label">Group (optional)</label>
          <input
            className="field__input"
            type="text"
            placeholder="production"
            value={group}
            onChange={(e) => setGroup(e.target.value)}
          />
        </div>
        <div className="field">
          <label className="field__label">Credential mode</label>
          <select
            className="field__input"
            value={credentialMode}
            onChange={(e) => setCredentialMode(e.target.value)}
          >
            {CREDENTIAL_MODES.map((m) => (
              <option key={m.value} value={m.value}>{m.label}</option>
            ))}
          </select>
        </div>
        <div className="field">
          <label className="field__label">Credential type</label>
          <select
            className="field__input"
            value={credentialType}
            onChange={(e) => setCredentialType(e.target.value)}
          >
            {CREDENTIAL_TYPES.map((t) => (
              <option key={t.value} value={t.value}>{t.label}</option>
            ))}
          </select>
        </div>
        {credentialMode === 'shared' && credentialType !== 'none' && (
          <div className="field">
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 4 }}>
              <label className="field__label" style={{ margin: 0 }}>
                Shared credential {isEdit && !sharedRevealed ? '(leave blank to keep existing)' : ''}
              </label>
              {/* PREVIOUS BUG: edit mode unconditionally blanked
                  sharedCredential and offered no path to view what was
                  stored. Admins had to clear and re-enter blind. The
                  reveal button below pulls the current secret into the
                  form so they can see it before changing. Audit-logged
                  on the daemon side via teams.hosts.credentials.revealShared. */}
              {isEdit && host?.canRevealSecrets && !sharedRevealed && (
                <button
                  type="button"
                  onClick={handleRevealShared}
                  disabled={revealingShared}
                  title="Reveal the currently-stored shared credential"
                  style={{
                    background: 'none',
                    border: 'none',
                    cursor: revealingShared ? 'progress' : 'pointer',
                    color: 'var(--muted)',
                    fontSize: 11,
                    fontFamily: 'var(--font-mono)',
                    padding: '2px 4px',
                  }}
                >
                  {revealingShared ? 'LOADING…' : 'VIEW EXISTING'}
                </button>
              )}
            </div>
            <textarea
              className="field__input"
              rows={4}
              placeholder={credentialType === 'private_key' ? 'Paste PEM private key…' : 'Password'}
              value={sharedCredential}
              onChange={(e) => setSharedCredential(e.target.value)}
              style={{ fontFamily: 'var(--font-mono)', fontSize: 11, resize: 'vertical' }}
            />
            {isEdit && (
              <label style={{ display: 'flex', alignItems: 'center', gap: 6, marginTop: 4, fontSize: 12 }}>
                <input
                  type="checkbox"
                  checked={clearSharedCredential}
                  onChange={(e) => setClearSharedCredential(e.target.checked)}
                />
                Clear existing credential
              </label>
            )}
          </div>
        )}
        <div className="field">
          <label className="field__label">Secret visibility</label>
          <select
            className="field__input"
            value={secretVisibility}
            onChange={(e) => setSecretVisibility(e.target.value)}
          >
            {SECRET_VISIBILITY.map((v) => (
              <option key={v.value} value={v.value}>{v.label}</option>
            ))}
          </select>
        </div>
        <div className="field">
          <label className="field__label">Notes (optional)</label>
          <textarea
            className="field__input"
            rows={3}
            placeholder="Internal notes visible to admins…"
            value={notes}
            onChange={(e) => setNotes(e.target.value)}
            style={{ resize: 'vertical' }}
          />
        </div>
      </form>
    </Drawer>
  );
}
