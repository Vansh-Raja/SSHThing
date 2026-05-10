/**
 * AppTabBar — the single tab strip shown above the main content. Renders
 * every open tab (Hosts, Settings, Terminal sessions, host editors, …)
 * in left-to-right order, with the always-present Hosts tab pinned and
 * non-closable.
 *
 * Visual style is intentionally close to the previous in-Hosts terminal
 * tab strip so the change reads as "the strip moved up + got more
 * residents" rather than a brand-new component.
 */
import { type ReactNode } from 'react';
import { useTabs, useTabActions } from '../contexts/TabsContext';
import { type Tab } from '../tabs/types';
import { HomeIcon, GearIcon, UserIcon, TokenIcon, TeamsIcon, TerminalIcon, KeyIcon, EditIcon, PlayIcon } from './icons';

interface AppTabBarProps {
  /** Optional pre-close hook; return false to cancel. Used for "discard changes?" prompt. */
  onConfirmClose?: (tab: Tab) => Promise<boolean>;
}

export default function AppTabBar({ onConfirmClose }: AppTabBarProps) {
  const { state } = useTabs();
  const { activate, close } = useTabActions();

  return (
    <div
      role="tablist"
      aria-label="Workspace tabs"
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
      {state.tabs.map((tab) => {
        const isActive = tab.id === state.activeId;
        return (
          <button
            type="button"
            role="tab"
            aria-selected={isActive}
            key={tab.id}
            className={isActive ? 'app-tab app-tab--active' : 'app-tab'}
            onClick={() => activate(tab.id)}
            onAuxClick={(e) => {
              // Middle-click closes (matches VSCode + every browser).
              if (e.button === 1 && tab.closable) {
                e.preventDefault();
                handleClose(tab);
              }
            }}
            style={{
              display: 'inline-flex',
              alignItems: 'center',
              gap: 6,
              padding: '6px 10px 6px 12px',
              border: 'none',
              borderBottom: '2px solid transparent',
              borderBottomColor: isActive ? 'var(--accent)' : 'transparent',
              background: 'transparent',
              color: isActive ? 'var(--ink)' : 'var(--muted)',
              fontFamily: 'var(--font-sans)',
              fontSize: 12,
              fontWeight: 600,
              cursor: 'pointer',
              whiteSpace: 'nowrap',
              flexShrink: 0,
              outline: 'none',
              minWidth: 0,
            }}
          >
            <span style={{ display: 'inline-flex', alignItems: 'center', width: 14, height: 14 }}>
              {iconForTab(tab)}
            </span>
            <span
              title={tab.title}
              style={{ maxWidth: 160, overflow: 'hidden', textOverflow: 'ellipsis' }}
            >
              {tab.title}
              {tab.dirty && <span style={{ marginLeft: 4, color: 'var(--accent)' }}>•</span>}
            </span>
            {tab.closable && (
              <span
                role="button"
                aria-label={`Close ${tab.title}`}
                tabIndex={-1}
                onClick={(e) => { e.stopPropagation(); handleClose(tab); }}
                style={{
                  display: 'inline-flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  width: 16,
                  height: 16,
                  marginLeft: 2,
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
          </button>
        );
      })}
    </div>
  );

  async function handleClose(tab: Tab) {
    if (!tab.closable) return;
    if (tab.dirty && onConfirmClose) {
      const ok = await onConfirmClose(tab);
      if (!ok) return;
    }
    close(tab.id);
  }
}

/**
 * iconForTab — small visual cue per kind. Hosts gets the home icon
 * (signals "this is the main view, you can't close it"); singletons get
 * their rail icon for parity; transient kinds get a verb-ish icon.
 */
function iconForTab(tab: Tab): ReactNode {
  switch (tab.kind) {
    case 'hosts': return <HomeIcon />;
    case 'settings': return <GearIcon />;
    case 'profile': return <UserIcon />;
    case 'tokens': return <TokenIcon />;
    case 'keys': return <KeyIcon />;
    case 'teams': return <TeamsIcon />;
    case 'terminal': return <TerminalIcon />;
    case 'host-editor':
    case 'host-editor-team': return <EditIcon />;
    case 'exec': return <PlayIcon />;
  }
}
