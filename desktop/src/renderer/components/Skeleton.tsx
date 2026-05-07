/**
 * Skeleton — shimmer loading placeholder.
 * Uses CSS animation defined in globals.css (.skeleton-shimmer).
 *
 * Usage:
 *   <Skeleton width="100%" height={14} />
 *   <Skeleton width={120} height={12} style={{ borderRadius: '50%' }} />
 */
import type { CSSProperties } from 'react';

interface SkeletonProps {
  width?: number | string;
  height?: number | string;
  style?: CSSProperties;
  className?: string;
}

export function Skeleton({ width = '100%', height = 14, style, className }: SkeletonProps) {
  return (
    <div
      className={`skeleton${className ? ` ${className}` : ''}`}
      style={{ width, height, ...style }}
      aria-hidden="true"
    />
  );
}

/**
 * SkeletonRows — renders N skeleton rows for list loading states.
 */
interface SkeletonRowsProps {
  count?: number;
  /** Height of each row container. Default 48. */
  rowHeight?: number;
}

export function SkeletonRows({ count = 4, rowHeight = 48 }: SkeletonRowsProps) {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
      {Array.from({ length: count }, (_, i) => (
        <div
          key={i}
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 12,
            padding: '8px 12px',
            height: rowHeight,
            borderRadius: 'var(--radius)',
            background: 'var(--paper-2)',
            border: '1px solid var(--line)',
          }}
          aria-hidden="true"
        >
          <Skeleton width={32} height={32} style={{ borderRadius: '50%', flexShrink: 0 }} />
          <div style={{ flex: 1, display: 'flex', flexDirection: 'column', gap: 6 }}>
            <Skeleton width="55%" height={13} />
            <Skeleton width="35%" height={11} />
          </div>
          <Skeleton width={48} height={20} style={{ borderRadius: 'var(--radius)' }} />
        </div>
      ))}
    </div>
  );
}

export default Skeleton;
