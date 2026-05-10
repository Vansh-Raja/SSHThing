import { lazy, Suspense, useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { HashRouter, Navigate, Route, Routes, useLocation, useNavigate } from 'react-router-dom';
import Unlock from './pages/Unlock';

// Pre-auth surfaces stay router-driven so they're standalone screens
// (the tab system is post-auth only). Post-auth content lives in the
// tab manager — see TabsProvider in AppLayout below.
const Welcome  = lazy(() => import('./pages/Welcome'));
const SignIn   = lazy(() => import('./pages/SignIn'));

import Toaster from './ui/Toaster';
import AppShell from './components/AppShell';
import AppTabBar from './components/AppTabBar';
import CommandPalette, { recordRecentHost } from './components/CommandPalette';
import HelpOverlay from './components/HelpOverlay';
import AboutModal from './components/AboutModal';
import Spotlight from './components/Spotlight';
import Dialog from './ui/Dialog';
import { openTerminalSession } from './components/TerminalTab';
import { useTheme } from './hooks/useTheme';
import { useTeams } from './hooks/useTeams';
import { useHostsCache, clearAllHostCaches } from './hooks/useHostsCache';
import { clearAuthCache } from './hooks/useAuth';
import { clearTeamsCache } from './hooks/useTeams';
import { TeamProvider, useTeamContext } from './contexts/TeamContext';
import { AppModeProvider, useAppMode } from './contexts/AppModeContext';
import { TabsProvider, useTabs, useTabActions } from './contexts/TabsContext';
import { renderTabs } from './tabs/registry';
import { type Tab, type TabKind, type TabState } from './tabs/types';
import { toast } from './ui/toast';

/**
 * Auth gate: makes sure the vault is unlocked before letting the user
 * past the unlock screen. Triggers a redirect to /unlock when locked.
 */
function AuthGuard({ children }: { children: React.ReactNode }) {
  const [checked, setChecked] = useState(false);
  const [unlocked, setUnlocked] = useState(false);
  const navigate = useNavigate();

  useEffect(() => {
    window.sshthing
      .vaultStatus()
      .then((status) => {
        setUnlocked(status.unlocked);
        setChecked(true);
        if (!status.unlocked) navigate('/unlock', { replace: true });
      })
      .catch(() => {
        setChecked(true);
        navigate('/unlock', { replace: true });
      });
  }, [navigate]);

  if (!checked) return null;
  if (!unlocked) return null;
  return <>{children}</>;
}

/**
 * AppLayout — root of the post-auth UI. Owns the tab manager + the
 * persistent shell (rail + topbar + tab strip + content area). Routes
 * still exist (HashRouter) for deep-linking and back-compat with menu
 * commands that call `navigate('/settings')` etc., but they no longer
 * mount different page components — every URL resolves to this layout
 * and the URL is used only as a signal for "open the matching tab".
 */
function AppLayout() {
  const navigate = useNavigate();
  const location = useLocation();
  const { setActiveTeamId } = useTeamContext();
  const { mode, setMode } = useAppMode();
  const { state: tabsState } = useTabs();
  const { open: openTab, close: closeTab } = useTabActions();
  const activeTab = tabsState.tabs.find((t) => t.id === tabsState.activeId);

  // URL → tab: a hash-route nav ('/settings', '/profile', …) opens or
  // focuses the matching tab. Welcome stays a route; '/' falls through
  // to the Hosts singleton (which is always already open).
  useEffect(() => {
    const kind = pathToSingletonKind(location.pathname);
    if (kind) openTab(kind, blankStateFor(kind));
  }, [location.pathname, openTab]);

  // Tab → URL: keep the URL in sync with the active tab (using replace
  // so we don't pollute the back stack). Helps menu commands that read
  // location and helps deep-linking.
  useEffect(() => {
    if (!activeTab) return;
    const target = kindToPath(activeTab.kind);
    if (target && location.pathname !== target) {
      navigate(target, { replace: true });
    }
  }, [activeTab, location.pathname, navigate]);

  // App mode follows the active tab's kind so the rail toggle reflects
  // reality regardless of how the tab was opened.
  useEffect(() => {
    if (!activeTab) return;
    if (activeTab.kind === 'teams' && mode !== 'teams') setMode('teams');
    if (activeTab.kind === 'hosts' && mode !== 'personal') setMode('personal');
  }, [activeTab, mode, setMode]);

  const [search, setSearch] = useState('');
  const [paletteOpen, setPaletteOpen] = useState(false);
  const [paletteInitialQuery, setPaletteInitialQuery] = useState('');
  const [spotlightOpen, setSpotlightOpen] = useState(false);
  const [helpOpen, setHelpOpen] = useState(false);
  const [aboutOpen, setAboutOpen] = useState(false);
  const [hosts, setHosts] = useState<HostSummary[]>([]);
  // Pending dirty-close confirmation. Set when the user clicks ✕ on a
  // tab that has `dirty=true`; clearing the dialog without confirming
  // leaves the tab open.
  const [pendingClose, setPendingClose] = useState<Tab | null>(null);
  const navigateRef = useRef(navigate);
  useEffect(() => { navigateRef.current = navigate; }, [navigate]);

  const { teams, reload: reloadTeams } = useTeams();
  const sortedTeams = useMemo(() => [...teams].sort((a, b) => a.displayOrder - b.displayOrder), [teams]);

  const { hosts: cachedHosts } = useHostsCache();
  useEffect(() => { setHosts(cachedHosts); }, [cachedHosts]);

  const onPaletteOpen = useCallback((query?: string) => {
    setPaletteInitialQuery(query ?? '');
    setPaletteOpen(true);
  }, []);
  const onSpotlightOpen = useCallback(() => setSpotlightOpen(true), []);
  const onHelpOpen = useCallback(() => setHelpOpen(true), []);

  const handleOpenKind = useCallback(
    (kind: 'hosts' | 'profile' | 'settings' | 'tokens' | 'teams') => {
      openTab(kind, blankStateFor(kind));
    },
    [openTab],
  );

  // Confirm-discard hook for the AppTabBar — returns true to allow close.
  const onConfirmClose = useCallback((tab: Tab): Promise<boolean> => {
    if (!tab.dirty) return Promise.resolve(true);
    return new Promise<boolean>((resolve) => {
      setPendingClose(tab);
      // Stash the resolver on the dialog state via a ref-y trick; simplest
      // approach is a single in-flight pending-close at a time, gated by
      // setPendingClose. The Dialog's confirm/close handlers below call
      // back into resolve through closure on this Promise.
      pendingResolverRef.current = resolve;
    });
  }, []);
  const pendingResolverRef = useRef<((ok: boolean) => void) | null>(null);

  // App-menu commands forwarded from Electron main.
  useEffect(() => {
    if (!window.sshthing.onMenuCommand) return;
    return window.sshthing.onMenuCommand((cmd) => {
      switch (cmd) {
        case 'open-settings':
          openTab('settings', { kind: 'settings' });
          break;
        case 'lock-vault':
          window.sshthing.lockVault().catch(() => {/* ignore */}).finally(() => {
            clearAllHostCaches();
            navigateRef.current('/unlock');
          });
          break;
        case 'sign-out':
          window.sshthing.authSignOut()
            .catch(() => {/* ignore */})
            .finally(() => {
              clearAuthCache();
              clearTeamsCache();
            });
          break;
        case 'open-help':
          setHelpOpen(true);
          break;
        case 'new-tab':
          window.dispatchEvent(new CustomEvent('sshthing:new-tab'));
          break;
        case 'open-account':
          openTab('profile', { kind: 'profile' });
          break;
        case 'open-about':
          setAboutOpen(true);
          break;
        case 'install-cli':
          (async () => {
            try {
              const res = await window.sshthing.installCli();
              if (res.ok) toast.success(`Installed: ${res.path}`);
              else toast.error(res.error ?? 'Install failed');
            } catch (err) {
              toast.error((err as Error).message ?? 'Install failed');
            }
          })();
          break;
        default:
          break;
      }
    });
  }, [openTab]);

  // Palette commands that need cross-component dispatch.
  useEffect(() => {
    const onAbout = () => setAboutOpen(true);
    const onTokens = () => openTab('tokens', { kind: 'tokens' });
    window.addEventListener('sshthing:cmd-about', onAbout);
    window.addEventListener('sshthing:cmd-tokens', onTokens);
    return () => {
      window.removeEventListener('sshthing:cmd-about', onAbout);
      window.removeEventListener('sshthing:cmd-tokens', onTokens);
    };
  }, [openTab]);

  // Cmd+W → close the active tab (if it's closable). Mirrors browser /
  // VSCode conventions. Falls back to no-op on the Hosts base tab.
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (!(e.metaKey || e.ctrlKey)) return;
      if (e.key !== 'w') return;
      const target = e.target;
      if (target instanceof HTMLElement) {
        // Inside a terminal pane, let the terminal handle Cmd+W (xterm
        // doesn't actually bind it, but we should never steal in inputs).
        if (target.matches('input, textarea, select, [contenteditable]')) return;
      }
      if (!activeTab || !activeTab.closable) return;
      e.preventDefault();
      if (activeTab.dirty) {
        // Route through the dirty-close confirmation modal.
        void onConfirmClose(activeTab).then((ok) => {
          if (ok) closeTab(activeTab.id);
        });
      } else {
        closeTab(activeTab.id);
      }
    };
    document.addEventListener('keydown', onKey);
    return () => document.removeEventListener('keydown', onKey);
  }, [activeTab, closeTab, onConfirmClose]);

  // Cmd+1..9 → quick-switch to indexed team.
  const sortedTeamsRef = useRef(sortedTeams);
  useEffect(() => { sortedTeamsRef.current = sortedTeams; }, [sortedTeams]);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (!(e.metaKey || e.ctrlKey)) return;
      const n = parseInt(e.key, 10);
      if (isNaN(n) || n < 1 || n > 9) return;
      const target = e.target;
      if (target instanceof HTMLElement) {
        if (target.matches('input, textarea, select, [contenteditable]')) return;
        if (target.closest('.xterm, .terminal, [data-terminal]')) return;
      }
      const team = sortedTeamsRef.current[n - 1];
      if (!team) return;
      e.preventDefault();
      setActiveTeamId(team.id);
      toast.info(`Switched to ${team.name}`);
      openTab('teams', { kind: 'teams' });
    };
    document.addEventListener('keydown', onKey);
    return () => document.removeEventListener('keydown', onKey);
  }, [setActiveTeamId, openTab]);

  // Connect-and-open: open an SSH session, dispatch a top-level
  // terminal tab. Used by the command palette + recent host actions.
  const connectAndOpen = useCallback(async (host: HostSummary) => {
    recordRecentHost(host.id);
    let sessionId: string | null = null;
    try {
      sessionId = await openTerminalSession(host, 80, 24);
    } catch {
      return;
    }
    if (!sessionId) return;
    const label = host.label.trim() || host.hostname;
    openTab('terminal', { kind: 'terminal', sessionId, hostId: host.id, hostLabel: label }, { title: label });
  }, [openTab]);

  return (
    <>
      <AppShell
        search={search}
        onSearch={setSearch}
        onPaletteOpen={onPaletteOpen}
        onSpotlightOpen={onSpotlightOpen}
        onHelpOpen={onHelpOpen}
        teams={sortedTeams}
        onTeamsChange={reloadTeams}
        activeTabKind={activeTab?.kind ?? 'hosts'}
        onOpenKind={handleOpenKind}
        tabBar={<AppTabBar onConfirmClose={onConfirmClose} />}
      >
        {renderTabs(tabsState.tabs, tabsState.activeId)}
      </AppShell>
      <CommandPalette
        open={paletteOpen}
        onClose={() => setPaletteOpen(false)}
        hosts={hosts}
        onSelectHost={(host) => {
          setPaletteOpen(false);
          void connectAndOpen(host);
        }}
        onHelpOpen={onHelpOpen}
        initialQuery={paletteInitialQuery}
      />
      <Spotlight
        open={spotlightOpen}
        onClose={() => setSpotlightOpen(false)}
        onOpenPalette={onPaletteOpen}
        onOpenHelp={onHelpOpen}
      />
      <HelpOverlay open={helpOpen} onClose={() => setHelpOpen(false)} />
      <AboutModal open={aboutOpen} onClose={() => setAboutOpen(false)} />
      <Dialog
        open={!!pendingClose}
        onClose={() => {
          pendingResolverRef.current?.(false);
          pendingResolverRef.current = null;
          setPendingClose(null);
        }}
        title="Discard changes?"
        message={pendingClose ? `${pendingClose.title} has unsaved changes that will be lost if you close it.` : ''}
        confirmLabel="Discard"
        confirmVariant="danger"
        onConfirm={() => {
          pendingResolverRef.current?.(true);
          pendingResolverRef.current = null;
          if (pendingClose) closeTab(pendingClose.id);
          setPendingClose(null);
        }}
      />
    </>
  );
}

/** SingletonKind — kinds that have a stable URL representation. */
type SingletonKind = 'hosts' | 'settings' | 'profile' | 'tokens' | 'keys' | 'teams';

/**
 * pathToSingletonKind / kindToPath — translate between the URL hash
 * routes that existed before the tab migration and the tab kinds.
 * Keeps menu commands and deep-links working without renaming
 * everything. Non-singleton kinds (terminal, host-editor, exec) don't
 * have a URL representation; the URL just stays at whatever singleton
 * was active before they opened.
 */
function pathToSingletonKind(path: string): SingletonKind | null {
  switch (path) {
    case '/hosts': return 'hosts';
    case '/settings': return 'settings';
    case '/profile': return 'profile';
    case '/tokens': return 'tokens';
    case '/keys': return 'keys';
    case '/teams': return 'teams';
    default: return null;
  }
}
function kindToPath(kind: TabKind): string | null {
  switch (kind) {
    case 'hosts': return '/hosts';
    case 'settings': return '/settings';
    case 'profile': return '/profile';
    case 'tokens': return '/tokens';
    case 'keys': return '/keys';
    case 'teams': return '/teams';
    // Tabs without a stable URL representation (terminals, editors,
    // exec runners) don't update the location; the URL stays at
    // whatever singleton was active before.
    default: return null;
  }
}
function blankStateFor(kind: 'hosts' | 'profile' | 'settings' | 'tokens' | 'teams' | 'keys'): TabState {
  return { kind } as TabState;
}

function ThemeInit() {
  useTheme();
  return null;
}

export default function App() {
  return (
    <HashRouter>
      <ThemeInit />
      <AppModeProvider>
        <TeamProvider>
          <Suspense fallback={null}>
            <Routes>
              {/* Pre-auth surfaces — own full-screen, no rail/topbar. */}
              <Route path="/unlock" element={<Unlock />} />
              <Route path="/sign-in" element={<SignIn />} />
              {/* /welcome is post-auth (needs vault unlocked) but renders
                  full-screen without the tab system — it's onboarding. */}
              <Route
                path="/welcome"
                element={<AuthGuard><Welcome /></AuthGuard>}
              />
              {/* Everything else is the tab-driven workspace. The tab
                  manager interprets the URL on mount + when it changes. */}
              <Route
                path="*"
                element={
                  <AuthGuard>
                    <TabsProvider>
                      <AppLayout />
                    </TabsProvider>
                  </AuthGuard>
                }
              />
            </Routes>
          </Suspense>
        </TeamProvider>
      </AppModeProvider>
      <Toaster />
    </HashRouter>
  );
}
