import { useEffect, useState } from 'react';
import Modal from '../ui/Modal';

interface AboutModalProps {
  open: boolean;
  onClose: () => void;
}

export default function AboutModal({ open, onClose }: AboutModalProps) {
  const [daemonVersion, setDaemonVersion] = useState<string | null>(null);

  useEffect(() => {
    if (!open) return;
    window.sshthing
      .daemonVersion()
      .then((v) => setDaemonVersion(v.version))
      .catch(() => setDaemonVersion('unknown'));
  }, [open]);

  return (
    <Modal open={open} onClose={onClose} title="About SSHThing" maxWidth={380}>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 14, color: 'var(--muted)', fontSize: 13 }}>
        <div style={{ textAlign: 'center' }}>
          <div style={{ fontSize: 20, fontWeight: 800, color: 'var(--ink)', marginBottom: 4 }}>SSHThing</div>
          <div>Desktop v1.0.0</div>
        </div>

        <div
          style={{
            background: 'var(--surface-2)',
            borderRadius: 'var(--radius-md)',
            padding: '10px 12px',
            display: 'flex',
            flexDirection: 'column',
            gap: 4,
          }}
        >
          <div style={{ display: 'flex', justifyContent: 'space-between' }}>
            <span>Daemon version</span>
            <span style={{ color: 'var(--ink)', fontFamily: 'var(--font-mono)', fontSize: 12 }}>
              {daemonVersion ?? '…'}
            </span>
          </div>
        </div>

        <p style={{ margin: 0, lineHeight: 1.55 }}>
          Licensed under the SSHThing License. Built with Electron and Go.
        </p>

        <div className="modal__actions" style={{ marginTop: 4 }}>
          <button type="button" className="btn btn--primary" onClick={onClose}>
            Close
          </button>
        </div>
      </div>
    </Modal>
  );
}
