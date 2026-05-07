import { type ButtonHTMLAttributes, type ReactNode } from 'react';
import clsx from 'clsx';

type IconButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  label: string;
  children: ReactNode;
};

export default function IconButton({ label, children, className, ...rest }: IconButtonProps) {
  return (
    <button
      type="button"
      aria-label={label}
      className={clsx('btn btn--ghost', className)}
      style={{ minHeight: 28, padding: '0 6px' }}
      {...rest}
    >
      {children}
    </button>
  );
}
