/**
 * Tab content registry — given a Tab, render the right React subtree.
 *
 * The registry holds React.lazy wrappers for the heavier kinds so the
 * initial JS bundle doesn't pull in Settings, Teams, Tokens etc. unless
 * the user actually opens those tabs. Hosts is eager-loaded because it's
 * the always-mounted base tab.
 *
 * Tab content is mounted on first activation and STAYS mounted for the
 * lifetime of the tab (just hidden via display:none when inactive).
 * That preserves form state, scroll position, in-flight operations,
 * etc., across tab switches — same trick the previous in-Hosts terminal
 * tabs already used.
 */
import { lazy, Suspense, type ReactNode } from 'react';
import { type Tab } from './types';
import HostsTab from './kinds/HostsTab';

const SettingsTab = lazy(() => import('./kinds/SettingsTab'));
const ProfileTab = lazy(() => import('./kinds/ProfileTab'));
const TokensTab = lazy(() => import('./kinds/TokensTab'));
const KeysTab = lazy(() => import('./kinds/KeysTab'));
const TeamsTab = lazy(() => import('./kinds/TeamsTab'));
const TerminalTabHost = lazy(() => import('./kinds/TerminalTabHost'));
const HostEditorTab = lazy(() => import('./kinds/HostEditorTab'));
const HostEditorTeamTab = lazy(() => import('./kinds/HostEditorTeamTab'));
const ExecTab = lazy(() => import('./kinds/ExecTab'));

export interface TabContentProps {
  tab: Tab;
  isActive: boolean;
}

/**
 * Render every open tab; toggle visibility via display:none rather than
 * unmounting on switch. This is what makes "Settings keeps its scroll
 * position when I bounce to a Terminal and back" work.
 */
export function renderTabs(tabs: Tab[], activeId: string): ReactNode {
  return tabs.map((tab) => {
    const isActive = tab.id === activeId;
    return (
      <div
        key={tab.id}
        role="tabpanel"
        aria-hidden={!isActive}
        style={{
          display: isActive ? 'flex' : 'none',
          flexDirection: 'column',
          flex: 1,
          minHeight: 0,
          overflow: 'hidden',
        }}
      >
        <Suspense fallback={null}>
          {renderForKind(tab, isActive)}
        </Suspense>
      </div>
    );
  });
}

function renderForKind(tab: Tab, isActive: boolean): ReactNode {
  switch (tab.kind) {
    case 'hosts':
      return <HostsTab tab={tab} isActive={isActive} />;
    case 'settings':
      return <SettingsTab tab={tab} isActive={isActive} />;
    case 'profile':
      return <ProfileTab tab={tab} isActive={isActive} />;
    case 'tokens':
      return <TokensTab tab={tab} isActive={isActive} />;
    case 'keys':
      return <KeysTab tab={tab} isActive={isActive} />;
    case 'teams':
      return <TeamsTab tab={tab} isActive={isActive} />;
    case 'terminal':
      return <TerminalTabHost tab={tab} isActive={isActive} />;
    case 'host-editor':
      return <HostEditorTab tab={tab} isActive={isActive} />;
    case 'host-editor-team':
      return <HostEditorTeamTab tab={tab} isActive={isActive} />;
    case 'exec':
      return <ExecTab tab={tab} isActive={isActive} />;
  }
}
