/**
 * Tabs — terminal tab strip (Phase 3B).
 * Uses @radix-ui/react-tabs under the hood for accessibility.
 */
import * as RadixTabs from '@radix-ui/react-tabs';
import clsx from 'clsx';
import { type ReactNode } from 'react';

export type TabItem = {
  id: string;
  label: ReactNode;
  onClose?: () => void;
};

type TabsProps = {
  tabs: TabItem[];
  active: string;
  onTabChange: (id: string) => void;
  onNewTab?: () => void;
  children: ReactNode;
};

export default function Tabs({ tabs, active, onTabChange, onNewTab, children }: TabsProps) {
  return (
    <RadixTabs.Root
      value={active}
      onValueChange={onTabChange}
      style={{ display: 'flex', flexDirection: 'column', height: '100%' }}
    >
      <RadixTabs.List
        aria-label="Terminal tabs"
        style={{
          display: 'flex',
          alignItems: 'center',
          background: 'var(--paper-2)',
          borderBottom: '1.5px solid var(--line)',
          flexShrink: 0,
          overflowX: 'auto',
          scrollbarWidth: 'none',
        }}
      >
        {tabs.map((tab) => (
          <RadixTabs.Trigger
            key={tab.id}
            value={tab.id}
            className={clsx('term-tab', active === tab.id && 'term-tab--active')}
            style={{
              display: 'inline-flex',
              alignItems: 'center',
              gap: 6,
              padding: '6px 12px',
              border: 'none',
              borderBottom: '2px solid transparent',
              background: 'transparent',
              color: active === tab.id ? 'var(--ink)' : 'var(--muted)',
              fontFamily: 'var(--font-mono)',
              fontSize: 12,
              fontWeight: 700,
              letterSpacing: '0.04em',
              cursor: 'pointer',
              whiteSpace: 'nowrap',
              borderBottomColor: active === tab.id ? 'var(--accent)' : 'transparent',
              flexShrink: 0,
              outline: 'none',
            }}
          >
            <span style={{ maxWidth: 140, overflow: 'hidden', textOverflow: 'ellipsis' }}>
              {tab.label}
            </span>
            {tab.onClose && (
              <span
                role="button"
                aria-label="Close tab"
                tabIndex={-1}
                onClick={(e) => { e.stopPropagation(); tab.onClose!(); }}
                style={{
                  display: 'inline-flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  width: 16,
                  height: 16,
                  borderRadius: 2,
                  fontSize: 14,
                  lineHeight: 1,
                  color: 'var(--muted)',
                  cursor: 'pointer',
                }}
              >
                ×
              </span>
            )}
          </RadixTabs.Trigger>
        ))}
        {onNewTab && (
          <button
            type="button"
            onClick={onNewTab}
            title="New tab (Cmd+T)"
            style={{
              display: 'inline-flex',
              alignItems: 'center',
              justifyContent: 'center',
              width: 28,
              height: 28,
              background: 'transparent',
              border: 'none',
              color: 'var(--muted)',
              fontSize: 18,
              cursor: 'pointer',
              marginLeft: 4,
              flexShrink: 0,
            }}
          >
            +
          </button>
        )}
      </RadixTabs.List>
      <div style={{ flex: 1, overflow: 'hidden', position: 'relative' }}>
        {children}
      </div>
    </RadixTabs.Root>
  );
}
