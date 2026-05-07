/**
 * One-time reveal modal — shows the credential for a host and lets the user copy it.
 * The credential is never stored in state longer than the modal is open.
 */
import { useEffect, useState } from 'react';
import Modal from '../ui/Modal';
import Button from '../ui/Button';
import Spinner from '../ui/Spinner';
import { toast } from '../ui/toast';

type RevealCredentialModalProps = {
  open: boolean;
  hostId: string | null;
  hostLabel: string;
  onClose: () => void;
};

export default function RevealCredentialModal({
  open,
  hostId,
  hostLabel,
  onClose,
}: RevealCredentialModalProps) {
  const [loading, setLoading] = useState(false);
  const [credential, setCredential] = useState<string | null>(null);
  const [authMode, setAuthMode] = useState('');
  const [error, setError] = useState('');

  useEffect(() => {
    if (!open || !hostId) return;
    setLoading(true);
    setCredential(null);
    setError('');
    window.sshthing
      .revealCredential(hostId)
      .then((result) => {
        setCredential(result.credential);
        setAuthMode(result.authMode);
      })
      .catch((err: unknown) => {
        const e = err as Error & { code?: number };
        if (e.code === -32601) {
          setError('Reveal credential requires a newer daemon version.');
        } else {
          setError(e.message ?? 'Failed to reveal credential');
        }
      })
      .finally(() => setLoading(false));
  }, [open, hostId]);

  const handleClose = () => {
    setCredential(null);
    onClose();
  };

  const copy = () => {
    if (!credential) return;
    navigator.clipboard.writeText(credential).then(
      () => toast.success('Copied to clipboard'),
      () => toast.error('Failed to copy'),
    );
  };

  return (
    <Modal
      open={open}
      onClose={handleClose}
      title={`Reveal Credential — ${hostLabel}`}
      footer={
        <div className="modal__actions">
          {credential && (
            <Button variant="secondary" onClick={copy}>
              Copy
            </Button>
          )}
          <Button variant="primary" onClick={handleClose}>
            Done
          </Button>
        </div>
      }
    >
      {loading && (
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, color: 'var(--muted)' }}>
          <Spinner />
          <span>Retrieving credential…</span>
        </div>
      )}
      {error && (
        <p style={{ color: 'var(--danger)', fontSize: 12 }}>{error}</p>
      )}
      {credential && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
          <span style={{ fontSize: 11, color: 'var(--muted)', textTransform: 'uppercase', letterSpacing: '0.08em', fontWeight: 700 }}>
            {authMode === 'key' ? 'Private key' : 'Password'}
          </span>
          <pre
            style={{
              fontFamily: 'var(--font-mono)',
              fontSize: 11,
              background: 'var(--paper-2)',
              border: '1.5px solid var(--line)',
              borderRadius: 2,
              padding: 10,
              margin: 0,
              overflowX: 'auto',
              whiteSpace: 'pre-wrap',
              wordBreak: 'break-all',
              maxHeight: 240,
              overflowY: 'auto',
              userSelect: 'text',
            }}
          >
            {credential}
          </pre>
          <p style={{ color: 'var(--danger)', fontSize: 11, margin: 0, fontWeight: 600 }}>
            This credential is visible once. Close this dialog when done.
          </p>
        </div>
      )}
    </Modal>
  );
}
