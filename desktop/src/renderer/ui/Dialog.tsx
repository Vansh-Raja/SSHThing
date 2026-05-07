/**
 * Dialog — confirm/alert modal. Wraps Modal with standardized footer actions.
 */
import { type ReactNode } from 'react';
import Modal from './Modal';

type DialogProps = {
  open: boolean;
  onClose: () => void;
  title: string;
  message: ReactNode;
  /** Label for the confirm button. Default: "Confirm" */
  confirmLabel?: string;
  /** Variant of confirm button. Default: "primary" */
  confirmVariant?: 'primary' | 'danger';
  onConfirm: () => void;
  loading?: boolean;
};

export default function Dialog({
  open,
  onClose,
  title,
  message,
  confirmLabel = 'Confirm',
  confirmVariant = 'primary',
  onConfirm,
  loading = false,
}: DialogProps) {
  return (
    <Modal
      open={open}
      onClose={onClose}
      title={title}
      footer={
        <div className="modal__actions">
          <button type="button" className="btn btn--ghost" onClick={onClose} disabled={loading}>
            Cancel
          </button>
          <button
            type="button"
            className={`btn btn--${confirmVariant}`}
            onClick={onConfirm}
            disabled={loading}
          >
            {loading ? <span className="spinner" /> : confirmLabel}
          </button>
        </div>
      }
    >
      <p style={{ color: 'var(--muted)', fontSize: 13, lineHeight: 1.55 }}>{message}</p>
    </Modal>
  );
}
