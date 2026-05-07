/**
 * EmptyState — a centered, visually soft placeholder for empty list/tab states.
 * All colours come from design tokens; no hard-coded values.
 */
import type { ReactNode } from 'react';

interface EmptyStateProps {
  /** Optional icon node (SVG/emoji) rendered above the title. */
  icon?: ReactNode;
  /** Short, bold headline. */
  title: string;
  /** One-liner explanation of the empty state. */
  description?: string;
  /** Optional CTA button. */
  action?: {
    label: string;
    onClick: () => void;
    /** Defaults to 'primary'. */
    variant?: 'primary' | 'ghost';
  };
}

export default function EmptyState({ icon, title, description, action }: EmptyStateProps) {
  const btnClass = action?.variant === 'ghost' ? 'btn btn--ghost' : 'btn btn--primary';
  return (
    <div className="empty-state-full">
      {icon && <div className="empty-state-full__icon">{icon}</div>}
      <div className="empty-state-full__title">{title}</div>
      {description && <div className="empty-state-full__desc">{description}</div>}
      {action && (
        <button
          type="button"
          className={btnClass}
          style={{ marginTop: 4 }}
          onClick={action.onClick}
        >
          {action.label}
        </button>
      )}
    </div>
  );
}
