import { useEffect, useState } from 'react';
import { createPortal } from 'react-dom';
import { dismissToast, subscribeToasts, type ToastRecord, type ToastVariant } from './toast';

/** Inline icon per toast variant — small, stroke-based SVG using currentColor. */
function ToastIcon({ variant }: { variant: ToastVariant }) {
  switch (variant) {
    case 'success':
      return (
        <svg width={16} height={16} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round" aria-hidden="true" style={{ flexShrink: 0 }}>
          <circle cx="12" cy="12" r="9" />
          <path d="M8 12l3 3 5-6" />
        </svg>
      );
    case 'error':
      return (
        <svg width={16} height={16} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round" aria-hidden="true" style={{ flexShrink: 0 }}>
          <circle cx="12" cy="12" r="9" />
          <path d="M15 9l-6 6M9 9l6 6" />
        </svg>
      );
    case 'warning':
      return (
        <svg width={16} height={16} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round" aria-hidden="true" style={{ flexShrink: 0 }}>
          <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" />
          <path d="M12 9v4M12 17h.01" />
        </svg>
      );
    case 'info':
      return (
        <svg width={16} height={16} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round" aria-hidden="true" style={{ flexShrink: 0 }}>
          <circle cx="12" cy="12" r="9" />
          <path d="M12 8h.01M12 12v4" />
        </svg>
      );
    default:
      return null;
  }
}

export default function Toaster() {
  const [toasts, setToasts] = useState<ToastRecord[]>([]);

  useEffect(() => {
    return subscribeToasts(setToasts);
  }, []);

  return createPortal(
    <div className="toaster" aria-live="polite" aria-atomic="false">
      {toasts.map((t) => (
        <button
          key={t.id}
          className={`toast toast--${t.variant}`}
          role={t.variant === 'error' ? 'alert' : 'status'}
          type="button"
          onClick={() => dismissToast(t.id)}
        >
          <ToastIcon variant={t.variant} />
          {t.message}
        </button>
      ))}
    </div>,
    document.body,
  );
}
