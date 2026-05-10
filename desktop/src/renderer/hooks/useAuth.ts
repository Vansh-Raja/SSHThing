/**
 * useAuth — subscribes to the current auth session, with a localStorage
 * cache so the renderer can paint the signed-in state synchronously
 * before the daemon's authSession() RPC completes.
 *
 * - On mount: read cached session synchronously → render → fire authSession()
 *   in the background → reconcile.
 * - Re-fetch on auth.signedIn / auth.signedOut notifications from the daemon.
 * - Module-scope pub/sub so multiple consumers share one in-flight fetch.
 */
import { useCallback, useEffect, useState } from 'react';
import { useNotifications } from './useNotifications';

const CACHE_KEY = 'sshthing.auth.session.v1';

function readCached(): AuthSessionInfo | null {
  try {
    const raw = localStorage.getItem(CACHE_KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as { session: AuthSessionInfo | null; at: number };
    if (!parsed?.session) return null;
    // If the cached session has clearly expired, drop it.
    if (parsed.session.expiresAt && parsed.session.expiresAt * 1000 < Date.now()) return null;
    return parsed.session;
  } catch { return null; }
}

function writeCached(s: AuthSessionInfo | null): void {
  try {
    if (s) localStorage.setItem(CACHE_KEY, JSON.stringify({ session: s, at: Date.now() }));
    else localStorage.removeItem(CACHE_KEY);
  } catch { /* ignore */ }
}

/** Clear the auth cache (e.g. on explicit sign-out). */
export function clearAuthCache(): void { writeCached(null); }

// Module-scope shared state so multiple useAuth() callers share one fetch.
type Listener = (s: AuthSessionInfo | null) => void;
const listeners = new Set<Listener>();
let lastResult: AuthSessionInfo | null | undefined = undefined;
let inFlight: Promise<AuthSessionInfo | null> | null = null;

function notify(s: AuthSessionInfo | null): void {
  lastResult = s;
  for (const l of listeners) l(s);
}

async function fetchOnce(): Promise<AuthSessionInfo | null> {
  if (inFlight) return inFlight;
  inFlight = (async () => {
    try {
      const res = await window.sshthing.authSession();
      const s = res.session ?? null;
      writeCached(s);
      notify(s);
      return s;
    } catch {
      return null;
    } finally {
      inFlight = null;
    }
  })();
  return inFlight;
}

export interface UseAuthResult {
  session: AuthSessionInfo | null;
  /** True only when there's no cached session AND a fetch is in flight. */
  loading: boolean;
  refresh: () => void;
}

export function useAuth(): UseAuthResult {
  const initial = lastResult !== undefined ? lastResult : readCached();
  const [session, setSession] = useState<AuthSessionInfo | null>(initial);
  // We're "loading" only on the very first fetch with no cached value.
  const [loading, setLoading] = useState<boolean>(initial === null && lastResult === undefined);

  // Subscribe to cross-component updates.
  useEffect(() => {
    const l: Listener = (s) => setSession(s);
    listeners.add(l);
    return () => { listeners.delete(l); };
  }, []);

  const refresh = useCallback(() => {
    setLoading(true);
    void fetchOnce().finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    refresh();
  }, [refresh]);

  // Re-fetch when the daemon emits sign-in or sign-out notifications.
  const handleNotification = useCallback(
    (method: string) => {
      if (method === 'auth.signedIn') {
        refresh();
      } else if (method === 'auth.signedOut') {
        writeCached(null);
        notify(null);
      }
    },
    [refresh],
  );
  useNotifications(handleNotification);

  return { session, loading, refresh };
}
