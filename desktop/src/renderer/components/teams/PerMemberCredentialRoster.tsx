/**
 * PerMemberCredentialRoster — shows the per-member credential roster for a
 * team host in per_member mode. Admins can reveal or delete any member's
 * credential. Each member can set their own via SetMyCredentialModal.
 */
import { useCallback, useState } from 'react';
import { useCredentialRoster } from '../../hooks/useCredentialRoster';
import Dialog from '../../ui/Dialog';
import RevealTeamCredentialModal from './RevealTeamCredentialModal';
import { toast } from '../../ui/toast';
import Modal from '../../ui/Modal';
import { Skeleton } from '../Skeleton';

type PerMemberCredentialRosterProps = {
  hostId: string;
  hostLabel: string;
  canManage: boolean; // admin+
  credentialType: string; // the host's default credential type (for the set form)
};

type SetCredentialState = {
  memberId: string;
  memberDisplayName: string;
  isCurrentUser: boolean;
} | null;

type RevealState = {
  memberId: string;
  memberDisplayName: string;
} | null;

type DeleteState = {
  memberId: string;
  memberDisplayName: string;
} | null;

export default function PerMemberCredentialRoster({
  hostId,
  hostLabel,
  canManage,
  credentialType,
}: PerMemberCredentialRosterProps) {
  const { roster, loading, error, reload } = useCredentialRoster(hostId);

  const [revealState, setRevealState] = useState<RevealState>(null);
  const [revealedCred, setRevealedCred] = useState<RevealedTeamHostCredential | null>(null);
  const [revealing, setRevealing] = useState(false);

  const [deleteState, setDeleteState] = useState<DeleteState>(null);
  const [deleting, setDeleting] = useState(false);

  const [setCredState, setSetCredState] = useState<SetCredentialState>(null);
  const [credSecret, setCredSecret] = useState('');
  const [credUsername, setCredUsername] = useState('');
  const [credType, setCredType] = useState(credentialType !== 'none' ? credentialType : 'password');
  const [savingCred, setSavingCred] = useState(false);

  const handleReveal = useCallback(async (entry: TeamHostCredentialRosterEntry) => {
    setRevealState({ memberId: entry.memberId, memberDisplayName: entry.displayName });
    setRevealing(true);
    try {
      const cred = await window.sshthing.teamsHostsRevealMember(hostId, entry.memberId);
      setRevealedCred(cred);
    } catch (err: unknown) {
      toast.error((err as Error).message ?? 'Failed to reveal credential');
      setRevealState(null);
    } finally {
      setRevealing(false);
    }
  }, [hostId]);

  const handleDelete = useCallback(async () => {
    if (!deleteState) return;
    setDeleting(true);
    try {
      await window.sshthing.teamsHostsDeleteMemberCredential(hostId, deleteState.memberId);
      toast.success(`Credential for ${deleteState.memberDisplayName} deleted`);
      setDeleteState(null);
      reload();
    } catch (err: unknown) {
      toast.error((err as Error).message ?? 'Failed to delete credential');
    } finally {
      setDeleting(false);
    }
  }, [deleteState, hostId, reload]);

  const handleSaveMyCred = useCallback(async () => {
    if (!setCredState) return;
    if (!credSecret.trim()) {
      toast.error('Secret is required');
      return;
    }
    setSavingCred(true);
    try {
      await window.sshthing.teamsHostsUpsertMyCredential(hostId, {
        credentialType: credType,
        secret: credSecret.trim(),
        username: credUsername.trim() || undefined,
      });
      toast.success('Credential saved');
      setSetCredState(null);
      setCredSecret('');
      setCredUsername('');
      reload();
    } catch (err: unknown) {
      toast.error((err as Error).message ?? 'Failed to save credential');
    } finally {
      setSavingCred(false);
    }
  }, [setCredState, hostId, credType, credSecret, credUsername, reload]);

  if (loading) {
    return (
      <div style={{ display: 'flex', flexDirection: 'column', gap: 4, marginTop: 8 }}>
        {Array.from({ length: 3 }, (_, i) => (
          <div
            key={i}
            style={{
              padding: '8px 12px',
              borderRadius: 'var(--radius)',
              background: 'var(--paper-2)',
              border: '1px solid var(--line)',
              display: 'flex',
              gap: 8,
              alignItems: 'center',
            }}
          >
            <Skeleton width="40%" height={12} />
            <Skeleton width={60} height={18} style={{ marginLeft: 'auto', borderRadius: 'var(--radius)' }} />
          </div>
        ))}
      </div>
    );
  }

  if (error) {
    return (
      <div style={{ padding: 8, fontSize: 12, color: 'var(--danger)' }}>{error}</div>
    );
  }

  return (
    <>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 4, marginTop: 8 }}>
        {roster.length === 0 && (
          <div style={{ fontSize: 12, color: 'var(--muted)', padding: '8px 0' }}>
            No members with credentials yet.
          </div>
        )}
        {roster.map((entry) => (
          <div
            key={entry.memberId}
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 8,
              padding: '8px 12px',
              borderRadius: 'var(--radius)',
              background: 'var(--paper-2)',
              border: '1px solid var(--line)',
            }}
          >
            {/* Name + email */}
            <div style={{ flex: 1, minWidth: 0 }}>
              <div style={{ fontSize: 12, fontWeight: 600, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                {entry.displayName}
                {entry.isCurrentUser && (
                  <span style={{ fontSize: 10, color: 'var(--accent)', marginLeft: 6 }}>you</span>
                )}
              </div>
              <div style={{ fontSize: 11, color: 'var(--muted)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                {entry.email}
              </div>
            </div>

            {/* Status badge */}
            <span
              className="chip"
              style={{
                fontSize: 10,
                textTransform: 'uppercase',
                letterSpacing: '0.06em',
                flexShrink: 0,
                color: entry.hasCredential ? 'var(--success, #4caf50)' : 'var(--muted)',
                borderColor: entry.hasCredential ? 'var(--success, #4caf50)' : undefined,
              }}
            >
              {entry.hasCredential ? 'Configured' : 'Not configured'}
            </span>

            {/* Actions */}
            <div style={{ display: 'flex', gap: 4, flexShrink: 0 }}>
              {/* Set my credential */}
              {entry.isCurrentUser && (
                <button
                  type="button"
                  className="btn btn--ghost"
                  style={{ fontSize: 11, padding: '3px 8px' }}
                  onClick={() => {
                    setSetCredState({ memberId: entry.memberId, memberDisplayName: entry.displayName, isCurrentUser: true });
                    setCredSecret('');
                    setCredUsername('');
                    setCredType(credentialType !== 'none' ? credentialType : 'password');
                  }}
                >
                  {entry.hasCredential ? 'Update' : 'Set credential'}
                </button>
              )}

              {/* Admin actions */}
              {canManage && entry.hasCredential && (
                <>
                  <button
                    type="button"
                    className="btn btn--ghost"
                    style={{ fontSize: 11, padding: '3px 8px' }}
                    onClick={() => void handleReveal(entry)}
                    disabled={revealing && revealState?.memberId === entry.memberId}
                  >
                    {revealing && revealState?.memberId === entry.memberId ? '…' : 'Reveal'}
                  </button>
                  <button
                    type="button"
                    className="btn btn--ghost"
                    style={{ fontSize: 11, padding: '3px 8px', color: 'var(--danger)' }}
                    onClick={() => setDeleteState({ memberId: entry.memberId, memberDisplayName: entry.displayName })}
                  >
                    Delete
                  </button>
                </>
              )}
            </div>
          </div>
        ))}
      </div>

      {/* Reveal modal */}
      <RevealTeamCredentialModal
        open={!!revealedCred}
        onClose={() => { setRevealedCred(null); setRevealState(null); }}
        hostLabel={hostLabel}
        credentialScope="member"
        memberDisplayName={revealState?.memberDisplayName}
        credentialType={revealedCred?.credentialType ?? ''}
        secret={revealedCred?.secret ?? ''}
        username={revealedCred?.username}
      />

      {/* Delete confirm */}
      <Dialog
        open={!!deleteState}
        onClose={() => setDeleteState(null)}
        title="Delete member credential"
        message={`Delete the credential for ${deleteState?.memberDisplayName ?? 'this member'}? This cannot be undone.`}
        confirmLabel="Delete"
        confirmVariant="danger"
        onConfirm={() => void handleDelete()}
        loading={deleting}
      />

      {/* Set my credential modal */}
      <Modal
        open={!!setCredState}
        onClose={() => { setSetCredState(null); setCredSecret(''); }}
        title="Set my credential"
        maxWidth={440}
        footer={
          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
            <button type="button" className="btn btn--ghost" onClick={() => { setSetCredState(null); setCredSecret(''); }} disabled={savingCred}>
              Cancel
            </button>
            <button
              type="button"
              className="btn btn--primary"
              onClick={() => void handleSaveMyCred()}
              disabled={savingCred}
            >
              {savingCred ? <span className="spinner" /> : 'Save'}
            </button>
          </div>
        }
      >
        <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
          <p style={{ margin: 0, fontSize: 12, color: 'var(--muted)', lineHeight: 1.55 }}>
            Set your personal credential for <strong>{hostLabel}</strong>. This credential is
            encrypted and visible only to you (and admins with reveal permission).
          </p>

          <div className="field">
            <label className="field__label">Credential type</label>
            <select
              className="field__input"
              value={credType}
              onChange={(e) => setCredType(e.target.value)}
            >
              <option value="password">Password</option>
              <option value="private_key">Private key</option>
            </select>
          </div>

          <div className="field">
            <label className="field__label">Username (optional)</label>
            <input
              className="field__input"
              type="text"
              placeholder="override username"
              value={credUsername}
              onChange={(e) => setCredUsername(e.target.value)}
            />
          </div>

          <div className="field">
            <label className="field__label">
              {credType === 'private_key' ? 'Private key (PEM)' : 'Password'}
            </label>
            <textarea
              className="field__input"
              rows={credType === 'private_key' ? 8 : 2}
              placeholder={credType === 'private_key' ? '-----BEGIN OPENSSH PRIVATE KEY-----\n…' : 'Enter password'}
              value={credSecret}
              onChange={(e) => setCredSecret(e.target.value)}
              style={{ fontFamily: credType === 'private_key' ? 'var(--font-mono)' : undefined, fontSize: 12, resize: 'vertical' }}
              autoComplete="new-password"
            />
          </div>
        </div>
      </Modal>
    </>
  );
}
