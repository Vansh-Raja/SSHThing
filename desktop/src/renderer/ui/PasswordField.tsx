import { type InputHTMLAttributes, type ReactNode, useState } from 'react';
import clsx from 'clsx';

type PasswordFieldProps = Omit<InputHTMLAttributes<HTMLInputElement>, 'type'> & {
  label?: string;
  hint?: ReactNode;
  error?: string;
};

export default function PasswordField({
  label,
  hint,
  error,
  className,
  id,
  ...rest
}: PasswordFieldProps) {
  const [show, setShow] = useState(false);
  const inputId = id ?? (label ? `pf-${label.toLowerCase().replace(/\s+/g, '-')}` : undefined);

  return (
    <div className="field">
      {label && (
        <label htmlFor={inputId} className="field__label">
          {label}
        </label>
      )}
      <div style={{ position: 'relative', display: 'flex' }}>
        <input
          id={inputId}
          type={show ? 'text' : 'password'}
          className={clsx('field__input', error && 'field__input--error', className)}
          style={{ paddingRight: 40 }}
          {...rest}
        />
        <button
          type="button"
          onClick={() => setShow((s) => !s)}
          aria-label={show ? 'Hide password' : 'Show password'}
          style={{
            position: 'absolute',
            right: 8,
            top: '50%',
            transform: 'translateY(-50%)',
            background: 'none',
            border: 'none',
            cursor: 'pointer',
            color: 'var(--muted)',
            fontSize: 12,
            fontFamily: 'var(--font-mono)',
            padding: '2px 4px',
          }}
        >
          {show ? 'HIDE' : 'SHOW'}
        </button>
      </div>
      {error && <span style={{ color: 'var(--danger)', fontSize: 11 }}>{error}</span>}
      {hint && !error && <span style={{ color: 'var(--muted)', fontSize: 11 }}>{hint}</span>}
    </div>
  );
}
