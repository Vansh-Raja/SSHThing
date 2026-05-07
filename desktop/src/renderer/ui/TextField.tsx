import { type InputHTMLAttributes, type ReactNode } from 'react';
import clsx from 'clsx';

type TextFieldProps = InputHTMLAttributes<HTMLInputElement> & {
  label?: string;
  hint?: ReactNode;
  error?: string;
};

export default function TextField({
  label,
  hint,
  error,
  className,
  id,
  ...rest
}: TextFieldProps) {
  const inputId = id ?? (label ? `tf-${label.toLowerCase().replace(/\s+/g, '-')}` : undefined);
  return (
    <div className="field">
      {label && (
        <label htmlFor={inputId} className="field__label">
          {label}
        </label>
      )}
      <input
        id={inputId}
        className={clsx('field__input', error && 'field__input--error', className)}
        {...rest}
      />
      {error && <span style={{ color: 'var(--danger)', fontSize: 11 }}>{error}</span>}
      {hint && !error && <span style={{ color: 'var(--muted)', fontSize: 11 }}>{hint}</span>}
    </div>
  );
}
