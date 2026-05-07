import { type ButtonHTMLAttributes, type ReactNode } from 'react';
import clsx from 'clsx';

type ButtonVariant = 'primary' | 'secondary' | 'danger' | 'ghost';

type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: ButtonVariant;
  loading?: boolean;
  children: ReactNode;
};

export default function Button({
  variant = 'secondary',
  loading = false,
  disabled,
  children,
  className,
  ...rest
}: ButtonProps) {
  return (
    <button
      className={clsx(
        'btn',
        variant === 'primary' && 'btn--primary',
        variant === 'danger' && 'btn--danger',
        variant === 'ghost' && 'btn--ghost',
        className,
      )}
      disabled={disabled ?? loading}
      {...rest}
    >
      {loading ? <span className="spinner" /> : children}
    </button>
  );
}
