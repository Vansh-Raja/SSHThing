/**
 * AppShell — the persistent three-region layout that wraps every authenticated
 * page (Hosts / Settings / Teams):
 *
 *   ┌─────┬──────────────── Topbar ─────────────┐
 *   │     ├──────────────────────────────────────┤
 *   │ rail│                                      │
 *   │     │             <Outlet />               │
 *   │     │                                      │
 *   │     ├─────────── Bottom command bar ──────┤
 *   └─────┴──────────────────────────────────────┘
 *
 * The rail is icon-only nav. The topbar holds wordmark + global search +
 * sync status + avatar. The bottom bar is a slash-command input
 * (deep-linkable via :commands like ":connect nas").
 */
import { useEffect } from 'react';
import { Link, NavLink, Outlet, useNavigate } from 'react-router-dom';
import { useAuth } from '../hooks/useAuth';
import { useSyncStatus } from '../hooks/useSyncStatus';
import { toast } from '../ui/toast';
import ErrorBoundary from './ErrorBoundary';
import DaemonHealthBanner from './DaemonHealthBanner';
import TeamSwitcherTopbar from './TeamSwitcherTopbar';
import InvitesBadge from './InvitesBadge';
import {
  TerminalIcon,
  UserIcon,
  GearIcon,
  KeyIcon,
  TeamsIcon,
  SignOutIcon,
  SearchIcon,
  CheckIcon,
  CloudOffIcon,
} from './icons';

// ──────────────────────────────────────────────────────────
// Rail
// ──────────────────────────────────────────────────────────
function RailButton({
  to,
  title,
  children,
  onClick,
}: {
  to?: string;
  title: string;
  children: React.ReactNode;
  onClick?: () => void;
}) {
  if (onClick) {
    return (
      <button type="button" className="rail__btn" title={title} onClick={onClick}>
        {children}
      </button>
    );
  }
  return (
    <NavLink
      to={to ?? '#'}
      title={title}
      className={({ isActive }) => `rail__btn${isActive ? ' rail__btn--active' : ''}`}
    >
      {children}
    </NavLink>
  );
}

function Rail() {
  const navigate = useNavigate();
  const { session } = useAuth();

  const handleSignOut = async () => {
    try {
      await window.sshthing.authSignOut?.();
      toast.success('Signed out');
    } catch (err) {
      const e = err as Error;
      toast.error(e.message ?? 'Sign-out failed');
    }
  };

  return (
    <nav className="rail" aria-label="Primary">
      <RailButton to="/hosts" title="Hosts"><TerminalIcon /></RailButton>
      <RailButton to="/account" title="Account"><UserIcon /></RailButton>
      <RailButton to="/settings" title="Settings"><GearIcon /></RailButton>
      <RailButton to="/keys" title="Keys"><KeyIcon /></RailButton>
      <RailButton to="/teams" title="Teams"><TeamsIcon /></RailButton>
      <div className="rail__spacer" />
      {session ? (
        <RailButton title="Sign out" onClick={handleSignOut}><SignOutIcon /></RailButton>
      ) : (
        <RailButton title="Sign in" onClick={() => navigate('/sign-in')}><SignOutIcon /></RailButton>
      )}
    </nav>
  );
}

// ──────────────────────────────────────────────────────────
// Top bar
// ──────────────────────────────────────────────────────────
function Topbar({
  search,
  onSearch,
  onPaletteOpen,
  teams,
  onTeamsChange,
}: {
  search: string;
  onSearch: (q: string) => void;
  onPaletteOpen: (query?: string) => void;
  teams: TeamSummary[];
  onTeamsChange: () => void;
}) {
  const sync = useSyncStatus();
  const { session } = useAuth();

  const initials = session
    ? (session.userEmail ?? '')
        .split('@')[0]
        .split(/[._-]/)
        .filter(Boolean)
        .slice(0, 2)
        .map((s: string) => s[0]!.toUpperCase())
        .join('')
    : '';

  return (
    <header className="topbar" aria-label="App top bar">
      <Link to="/hosts" style={{ textDecoration: 'none', color: 'inherit' }}>
        <span className="topbar__brand">sshthing</span>
      </Link>

      <div className="topbar__search">
        <span className="topbar__search-icon"><SearchIcon /></span>
        <input
          className="topbar__search-input"
          type="search"
          placeholder="Search hosts… (⌘K)"
          value={search}
          onChange={(e) => {
            const v = e.target.value;
            onSearch(v);
            // Opening the palette pre-filled on the first character lets the
            // topbar input act as a palette trigger without breaking its own
            // controlled-input semantics (the palette takes over from here).
            if (v.length > 0) {
              onPaletteOpen(v);
              onSearch('');
            }
          }}
          onFocus={() => {
            // Clicking / tabbing into the bar opens the palette immediately.
            onPaletteOpen(search || undefined);
            onSearch('');
          }}
          onKeyDown={(e) => {
            if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
              e.preventDefault();
              onPaletteOpen();
            }
          }}
          spellCheck={false}
        />
        <span className="topbar__search-kbd">⌘K</span>
      </div>

      {session && (
        <div className="topbar__teams">
          <TeamSwitcherTopbar teams={teams} onTeamsChange={onTeamsChange} />
        </div>
      )}

      <div className="topbar__right">
        <InvitesBadge />

        {sync.state === 'ok' && (
          <span className="topbar__sync topbar__sync--ok" title="Last synced just now">
            <CheckIcon /> Synced
          </span>
        )}
        {sync.state === 'syncing' && (
          <span className="topbar__sync" title="Sync in progress">
            <span className="spinner" /> Syncing
          </span>
        )}
        {sync.state === 'error' && (
          <span className="topbar__sync topbar__sync--err" title={sync.message ?? 'Sync error'}>
            <CloudOffIcon /> Offline
          </span>
        )}
        {sync.state === 'idle' && session && (
          <span className="topbar__sync topbar__sync--idle">Local only</span>
        )}

        {session ? (
          <Link to="/account" className="topbar__avatar" title={session.userEmail ?? 'Account'}>
            {initials || '·'}
          </Link>
        ) : (
          <Link to="/sign-in" className="topbar__avatar" title="Sign in">·</Link>
        )}
      </div>
    </header>
  );
}

// Bottom command bar removed for now — will return as a VS Code-style
// command palette when the slash-command surface is properly designed.

// ──────────────────────────────────────────────────────────
// Shell
// ──────────────────────────────────────────────────────────
export interface AppShellContext {
  search: string;
  onPaletteOpen: (query?: string) => void;
}

interface AppShellProps {
  search: string;
  onSearch: (q: string) => void;
  onPaletteOpen: (query?: string) => void;
  onHelpOpen: () => void;
  teams: TeamSummary[];
  onTeamsChange: () => void;
}

export default function AppShell({ search, onSearch, onPaletteOpen, onHelpOpen, teams, onTeamsChange }: AppShellProps) {
  // Cmd+K anywhere → open palette
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault();
        onPaletteOpen();
        return;
      }
      // `?` outside an input → open help overlay
      if (
        e.key === '?' &&
        !e.metaKey &&
        !e.ctrlKey &&
        !(e.target instanceof HTMLElement &&
          e.target.matches('input, textarea, [contenteditable]'))
      ) {
        e.preventDefault();
        onHelpOpen();
      }
    };
    document.addEventListener('keydown', onKey);
    return () => document.removeEventListener('keydown', onKey);
  }, [onPaletteOpen, onHelpOpen]);

  return (
    <div className="app-shell">
      <Rail />
      <Topbar search={search} onSearch={onSearch} onPaletteOpen={onPaletteOpen} teams={teams} onTeamsChange={onTeamsChange} />
      <main className="main">
        <ErrorBoundary>
          <Outlet />
        </ErrorBoundary>
      </main>
      <DaemonHealthBanner />
    </div>
  );
}
