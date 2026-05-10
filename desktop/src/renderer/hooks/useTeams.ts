/**
 * useTeams — subscribes to the user's team list, with a localStorage
 * cache so the renderer paints the team list synchronously before the
 * daemon's teamsList() RPC (which goes over the network) completes.
 *
 * - On mount: read cached teams synchronously → render → fire teamsList()
 *   in the background → reconcile.
 * - Module-scope pub/sub so all consumers (AppLayout, Account page, team
 *   switcher, Teams page) share one in-flight fetch.
 */
import { useCallback, useEffect, useState } from 'react';

const CODE_NOT_SIGNED_IN = -32010;
const CACHE_KEY = 'sshthing.teams.v1';

function isNotSignedInError(err: unknown): boolean {
  if (!err || typeof err !== 'object') return false;
  const e = err as { code?: number };
  return e.code === CODE_NOT_SIGNED_IN;
}

interface CacheEntry {
  teams: TeamSummary[];
  notSignedIn: boolean;
  at: number;
}

function readCached(): CacheEntry | null {
  try {
    const raw = localStorage.getItem(CACHE_KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as CacheEntry;
    if (!Array.isArray(parsed.teams)) return null;
    return parsed;
  } catch { return null; }
}

function writeCached(entry: CacheEntry): void {
  try { localStorage.setItem(CACHE_KEY, JSON.stringify(entry)); } catch { /* ignore */ }
}

export function clearTeamsCache(): void {
  try { localStorage.removeItem(CACHE_KEY); } catch { /* ignore */ }
}

type State = { teams: TeamSummary[]; notSignedIn: boolean; error: string | null };
type Listener = (s: State) => void;
const listeners = new Set<Listener>();
let lastState: State | undefined;
let inFlight: Promise<void> | null = null;

function notify(s: State): void {
  lastState = s;
  for (const l of listeners) l(s);
}

async function fetchOnce(): Promise<void> {
  if (inFlight) return inFlight;
  inFlight = (async () => {
    try {
      const result = await window.sshthing.teamsList();
      const next: State = { teams: result.teams ?? [], notSignedIn: false, error: null };
      writeCached({ teams: next.teams, notSignedIn: false, at: Date.now() });
      notify(next);
    } catch (err) {
      if (isNotSignedInError(err)) {
        const next: State = { teams: [], notSignedIn: true, error: null };
        writeCached({ teams: [], notSignedIn: true, at: Date.now() });
        notify(next);
      } else {
        // Keep the cached teams visible on transient network failures.
        const cached = readCached();
        notify({
          teams: cached?.teams ?? [],
          notSignedIn: cached?.notSignedIn ?? false,
          error: (err as Error).message ?? 'Failed to load teams',
        });
      }
    } finally {
      inFlight = null;
    }
  })();
  return inFlight;
}

export interface UseTeamsResult {
  teams: TeamSummary[];
  loading: boolean;
  notSignedIn: boolean;
  error: string | null;
  reload: () => void;
}

export function useTeams(): UseTeamsResult {
  const initial: State = lastState ?? (() => {
    const cached = readCached();
    return cached
      ? { teams: cached.teams, notSignedIn: cached.notSignedIn, error: null }
      : { teams: [], notSignedIn: false, error: null };
  })();

  const [state, setState] = useState<State>(initial);
  // Loading only when there's no cached state and a fetch is pending.
  const [loading, setLoading] = useState<boolean>(lastState === undefined && readCached() === null);

  useEffect(() => {
    const l: Listener = (s) => setState(s);
    listeners.add(l);
    return () => { listeners.delete(l); };
  }, []);

  const reload = useCallback(() => {
    setLoading(true);
    void fetchOnce().finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    reload();
  }, [reload]);

  return {
    teams: state.teams,
    loading,
    notSignedIn: state.notSignedIn,
    error: state.error,
    reload,
  };
}
