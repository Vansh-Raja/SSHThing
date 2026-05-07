import Modal from '../ui/Modal';

interface WelcomeModalProps {
  open: boolean;
  onClose: () => void;
}

export default function WelcomeModal({ open, onClose }: WelcomeModalProps) {
  return (
    <Modal open={open} onClose={onClose} title="Welcome to SSHThing" maxWidth={420} dismissible={false}>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 16, color: 'var(--muted)', fontSize: 13 }}>
        <p style={{ margin: 0, lineHeight: 1.55 }}>
          Your vault is ready. Here are a few things to get you started:
        </p>
        <ul style={{ margin: 0, paddingLeft: 18, lineHeight: 1.7 }}>
          <li>Your vault is ready</li>
          <li>Add hosts from the sidebar</li>
          <li>
            Use{' '}
            <kbd
              style={{
                background: 'var(--surface-2)',
                border: '1px solid var(--border)',
                borderRadius: 'var(--radius-sm)',
                padding: '1px 4px',
                fontFamily: 'inherit',
                fontSize: 12,
              }}
            >
              Cmd+K
            </kbd>{' '}
            for quick actions
          </li>
          <li>Sign in for Teams and Cloud sync</li>
        </ul>
        <div className="modal__actions" style={{ marginTop: 8 }}>
          <button type="button" className="btn btn--primary" onClick={onClose}>
            Get started
          </button>
        </div>
      </div>
    </Modal>
  );
}
