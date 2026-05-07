import clsx from 'clsx';
import { type ReactNode } from 'react';

type TagProps = {
  children: ReactNode;
  variant?: 'default' | 'accent' | 'muted';
  onRemove?: () => void;
};

export default function Tag({ children, variant = 'default', onRemove }: TagProps) {
  return (
    <span
      className={clsx(
        'chip',
        variant === 'accent' && 'chip--accent',
        variant === 'muted' && 'chip--muted',
      )}
    >
      {children}
      {onRemove && (
        <button
          type="button"
          onClick={onRemove}
          aria-label="Remove"
          style={{ background: 'none', border: 'none', cursor: 'pointer', padding: 0, lineHeight: 1, color: 'inherit' }}
        >
          ×
        </button>
      )}
    </span>
  );
}
