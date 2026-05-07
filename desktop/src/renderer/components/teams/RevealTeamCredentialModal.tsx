/**
 * RevealTeamCredentialModal — displays a revealed shared or per-member credential
 * in a one-time-view modal with a copy button. Shows an audit-log warning.
 *
 * Role-gate: only rendered when the host's canRevealSecrets flag is true.
 */
import { useCallback, useState } from 'react';
import Modal from '../../ui/Modal';
import { toast } from '../../ui/toast';

type RevealTeamCredentialModalProps = {
  open: boolean;
  onClose: () => void;
  hostLabel: string;
  /** 'shared' | 'member' */
  credentialScope: 'shared' | 'member';
  memberDisplayName?: string;
  credentialType: string;
  secret: string;
  username?: string;
};

export default function RevealTeamCredentialModal({
  open,
  onClose,
  hostLabel,
  credentialScope,
  memberDisplayName,
  credentialType,
  secret,
  username,
}: RevealTeamCredentialModalProps) {
  const [copied, setCopied] = useState(false);

  const handleCopy = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(secret);
      setCopied(true);
      toast.success('Copied to clipboard');
      setTimeout(() => setCopied(false), 2000);
    } catch {
      toast.error('Failed to copy — select and copy manually');
    }
  }, [secret]);

  const title =
    credentialScope === 'shared'
      ? `Shared credential — ${hostLabel}`
      : `Credential for ${memberDisplayName ?? 'member'} — ${hostLabel}`;

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={title}
      maxWidth={480}
      footer={
        <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
          <button type="button" className="btn btn--ghost" onClick={onClose}>
            Close
          </button>
          <button
            type="button"
            className="btn btn--primary"
            onClick={() => void handleCopy()}
          >
            {copied ? 'Copied!' : 'Copy'}
          </button>
        </div>
      }
    >
      <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
        {/* Audit warning */}
        <div
          style={{
            padding: '8px 12px',
            borderRadius: 'var(--radius)',
            background: 'color-mix(in srgb, var(--warning, #ff9800) 12%, transparent)',
            border: '1px solid color-mix(in srgb, var(--warning, #ff9800) 30%, transparent)',
            fontSize: 12,
            color: 'var(--ink)',
            lineHeight: 1.55,
          }}
        >
          This reveal has been audit-logged. Handle with care — do not share outside
          of this team.
        </div>

        <dl
          style={{
            display: 'grid',
            gridTemplateColumns: 'max-content 1fr',
            gap: '6px 12px',
            margin: 0,
          }}
        >
          <dt style={{ fontSize: 11, color: 'var(--muted)', alignSelf: 'center' }}>Type</dt>
          <dd style={{ fontSize: 12, margin: 0, fontFamily: 'var(--font-mono)' }}>
            {credentialType}
          </dd>
          {username && (
            <>
              <dt style={{ fontSize: 11, color: 'var(--muted)', alignSelf: 'center' }}>Username</dt>
              <dd style={{ fontSize: 12, margin: 0, fontFamily: 'var(--font-mono)' }}>{username}</dd>
            </>
          )}
          <dt style={{ fontSize: 11, color: 'var(--muted)', alignSelf: 'start', paddingTop: 2 }}>
            {credentialType === 'private_key' ? 'Private key' : 'Password'}
          </dt>
          <dd style={{ margin: 0 }}>
            <textarea
              readOnly
              value={secret}
              rows={credentialType === 'private_key' ? 8 : 2}
              style={{
                width: '100%',
                fontFamily: 'var(--font-mono)',
                fontSize: 11,
                resize: 'vertical',
                background: 'var(--paper-2)',
                border: '1px solid var(--line)',
                borderRadius: 'var(--radius)',
                color: 'var(--ink)',
                padding: '6px 8px',
              }}
              onFocus={(e) => e.currentTarget.select()}
            />
          </dd>
        </dl>
      </div>
    </Modal>
  );
}
