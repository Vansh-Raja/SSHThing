import { lazy, Suspense, useCallback, useEffect, useRef, useState } from 'react';
import { HashRouter, Navigate, Route, Routes, useLocation, useNavigate } from 'react-router-dom';
import Unlock from './pages/Unlock';
import Hosts from './pages/Hosts';

// ── Code-split the heavier, less-frequently-visited routes. The main
// bundle drops by ~300 KB so cold-start parses faster; users pay the
// per-route fetch only when they actually navigate to that page.
const Welcome  = lazy(() => import('./pages/Welcome'));
const Settings = lazy(() => import('./pages/Settings'));
const Teams    = lazy(() => import('./pages/Teams'));
const SignIn   = lazy(() => import('./pages/SignIn'));
const Profile  = lazy(() => import('./pages/Profile'));
const Keys     = lazy(() => import('./pages/Keys'));
const Tokens   = lazy(() => import('./pages/Tokens'));
import Toaster from './ui/Toaster';
import AppShell from './components/AppShell';
import CommandPalette, { recordRecentHost } from './components/CommandPalette';
import HelpOverlay from './components/HelpOverlay';
import AboutModal from './components/AboutModal';
import Spotlight from './components/Spotlight';
import { openTerminalSession } from './components/TerminalTab';
import { useTheme } from './hooks/useTheme';
import { useTeams } from './hooks/useTeams';
import { useHostsCache, clearAllHostCaches } from './hooks/useHostsCache';
import { clearAuthCache } from './hooks/useAuth';
import { clearTeamsCache } from './hooks/useTeams';
import { TeamProvider, useTeamContext } from './contexts/TeamContext';
import { AppModeProvider, useAppMode } from './contexts/AppModeContext';
import { toast } from './ui/toast';

/**
 * Redirect root route based on current app mode.
 */
function ModeRedirect() {
  const { mode } = useAppMode();
  return <Navigate to={mode === 'teams' ? '/teams' : '/hosts'} replace />;
}

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
 * AppLayout — wraps the routed pages in the persistent AppShell
 * (icon rail + topbar + bottombar). Owns the global search + command
 * palette + command-bar plumbing.
 */
function AppLayout() {
  const navigate = useNavigate();
  const location = useLocation();
  const { activeTeamId, setActiveTeamId } = useTeamContext();
  const { mode, setMode } = useAppMode();

  // Keep mode in sync with the current route so the rail + toggle always match
  // regardless of how navigation happened (mode toggle, palette, back button, etc.)
  useEffect(() => {
    const path = location.pathname;
    if (path === '/teams' && mode !== 'teams') {
      setMode('teams');
    } else if (path === '/hosts' && mode !== 'personal') {
      setMode('personal');
    }
  }, [location.pathname, mode, setMode]);

  const [search, setSearch] = useState('');
  const [paletteOpen, setPaletteOpen] = useState(false);
  const [paletteInitialQuery, setPaletteInitialQuery] = useState('');
  const [spotlightOpen, setSpotlightOpen] = useState(false);
  const [helpOpen, setHelpOpen] = useState(false);
  const [aboutOpen, setAboutOpen] = useState(false);
  const [hosts, setHosts] = useState<HostSummary[]>([]);
  // Stable ref so menu-command handler doesn't re-register on every render
  const navigateRef = useRef(navigate);
  useEffect(() => { navigateRef.current = navigate; }, [navigate]);

  // Load teams for topbar switcher and Cmd+1..9 quick-switch
  const { teams, reload: reloadTeams } = useTeams();
  const sortedTeams = [...teams].sort((a, b) => a.displayOrder - b.displayOrder);

  // Single source of truth for hosts, shared with Hosts.tsx via the
  // pub/sub inside useHostsCache. This eliminates the second daemon RPC
  // that used to fire on every route change.
  const { hosts: cachedHosts } = useHostsCache();
  useEffect(() => { setHosts(cachedHosts); }, [cachedHosts]);

  const onPaletteOpen = useCallback((query?: string) => {
    setPaletteInitialQuery(query ?? '');
    setPaletteOpen(true);
  }, []);

  const onSpotlightOpen = useCallback(() => setSpotlightOpen(true), []);

  const onHelpOpen = useCallback(() => setHelpOpen(true), []);

  // Listen for app-menu commands sent from the Electron main process.
  useEffect(() => {
    if (!window.sshthing.onMenuCommand) return;
    return window.sshthing.onMenuCommand((cmd) => {
      switch (cmd) {
        case 'open-settings':
          navigateRef.current('/settings');
          break;
        case 'lock-vault':
          window.sshthing.lockVault().catch(() => {/* ignore */}).finally(() => {
            clearAllHostCaches();
            // Don't clear auth/teams here — locking the vault doesn't sign
            // the user out of the cloud, and we want their team list to be
            // ready immediately after re-unlock.
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
          // Forward to the Hosts page via a custom event so it can open a new tab.
          window.dispatchEvent(new CustomEvent('sshthing:new-tab'));
          break;
        case 'open-account':
          navigateRef.current('/profile');
          break;
        case 'open-about':
          setAboutOpen(true);
          break;
        case 'install-cli':
          (async () => {
            try {
              const res = await window.sshthing.installCli();
              if (res.ok) {
                toast.success(`Installed: ${res.path}`);
              } else {
                toast.error(res.error ?? 'Install failed');
              }
            } catch (err) {
              toast.error((err as Error).message ?? 'Install failed');
            }
          })();
          break;
        default:
          break;
      }
    });
  }, []);

  // Listen for palette commands that need navigation
  useEffect(() => {
    const onAbout = () => setAboutOpen(true);
    const onTokens = () => navigateRef.current('/tokens');
    window.addEventListener('sshthing:cmd-about', onAbout);
    window.addEventListener('sshthing:cmd-tokens', onTokens);
    return () => {
      window.removeEventListener('sshthing:cmd-about', onAbout);
      window.removeEventListener('sshthing:cmd-tokens', onTokens);
    };
  }, []);

  // Cmd+1..9 → quick-switch to indexed team (1-based, sorted by displayOrder).
  // Bail out if the focus is inside a terminal pane or any input/textarea to
  // avoid colliding with terminal tab switching or text editing.
  const sortedTeamsRef = useRef(sortedTeams);
  useEffect(() => { sortedTeamsRef.current = sortedTeams; }, [sortedTeams]);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (!(e.metaKey || e.ctrlKey)) return;
      const n = parseInt(e.key, 10);
      if (isNaN(n) || n < 1 || n > 9) return;

      // Don't intercept when focus is in a terminal, input, or contenteditable.
      const target = e.target;
      if (target instanceof HTMLElement) {
        if (target.matches('input, textarea, select, [contenteditable]')) return;
        if (target.closest('.xterm, .terminal, [data-terminal]')) return;
      }

      const team = sortedTeamsRef.current[n - 1];
      if (!team) return; // out of range — no-op

      e.preventDefault();
      setActiveTeamId(team.id);
      toast.info(`Switched to ${team.name}`);
      navigateRef.current('/teams');
    };

    document.addEventListener('keydown', onKey);
    return () => document.removeEventListener('keydown', onKey);
  }, [setActiveTeamId]);

  const connectAndOpen = useCallback(async (host: HostSummary) => {
    recordRecentHost(host.id);
    let sessionId: string | null = null;
    try {
      sessionId = await openTerminalSession(host, 80, 24);
    } catch {
      return; // toasted inside
    }
    if (!sessionId) return;
    // Surface to Hosts page via custom event so it can adopt the session as a tab.
    window.dispatchEvent(
      new CustomEvent('sshthing:adopt-session', {
        detail: { hostId: host.id, sessionId, label: host.label.trim() || host.hostname },
      }),
    );
    if (location.pathname !== '/hosts') navigate('/hosts');
  }, [location.pathname, navigate]);

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
      />
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
    </>
  );
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
            <Route path="/unlock" element={<Unlock />} />
            <Route path="/sign-in" element={<SignIn />} />
            <Route
              element={
                <AuthGuard>
                  <AppLayout />
                </AuthGuard>
              }
            >
              <Route path="/welcome" element={<Welcome />} />
              <Route path="/hosts" element={<Hosts />} />
              <Route path="/settings" element={<Settings />} />
              <Route path="/teams" element={<Teams />} />
              <Route path="/profile" element={<Profile />} />
              <Route path="/keys" element={<Keys />} />
              <Route path="/tokens" element={<Tokens />} />
            </Route>
            <Route path="/" element={<ModeRedirect />} />
            <Route path="*" element={<Navigate to="/hosts" replace />} />
          </Routes>
          </Suspense>
        </TeamProvider>
      </AppModeProvider>
      <Toaster />
    </HashRouter>
  );
}
