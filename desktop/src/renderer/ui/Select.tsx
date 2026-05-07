import { type SelectHTMLAttributes, type ReactNode } from 'react';
import clsx from 'clsx';

type SelectOption = { value: string; label: string };

type SelectProps = SelectHTMLAttributes<HTMLSelectElement> & {
  label?: string;
  options: SelectOption[];
  hint?: ReactNode;
  error?: string;
};

export default function Select({
  label,
  options,
  hint,
  error,
  className,
  id,
  ...rest
}: SelectProps) {
  const selectId = id ?? (label ? `sel-${label.toLowerCase().replace(/\s+/g, '-')}` : undefined);
  return (
    <div className="field">
      {label && (
        <label htmlFor={selectId} className="field__label">
          {label}
        </label>
      )}
      <select
        id={selectId}
        className={clsx('field__input', className)}
        style={{ cursor: 'pointer' }}
        {...rest}
      >
        {options.map((opt) => (
          <option key={opt.value} value={opt.value}>
            {opt.label}
          </option>
        ))}
      </select>
      {error && <span style={{ color: 'var(--danger)', fontSize: 11 }}>{error}</span>}
      {hint && !error && <span style={{ color: 'var(--muted)', fontSize: 11 }}>{hint}</span>}
    </div>
  );
}
