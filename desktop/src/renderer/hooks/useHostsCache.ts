/**
 * useHostsCache — single source of truth for the host list, with a
 * stale-while-revalidate cache backed by localStorage.
 *
 * On mount: reads cached hosts synchronously (so the first paint can
 * include the host list, before the daemon even responds), then fires a
 * background `listHosts()` and swaps in fresh data.
 *
 * Cache is keyed by the vault salt so a different vault doesn't see another
 * vault's hosts. The salt is captured on the most recent vault.unlock /
 * vault.status response and persisted in localStorage too.
 *
 * Subscribers (App.tsx palette + Hosts.tsx list) share a single in-memory
 * source via a tiny pub/sub so we don't re-fetch on every page nav.
 */
import { useCallback, useEffect, useState } from 'react';

const SALT_KEY = 'sshthing.vault.salt.v1';
const HOSTS_KEY_PREFIX = 'sshthing.hosts.v1.';

function readCachedSalt(): string | null {
  try { return localStorage.getItem(SALT_KEY); } catch { return null; }
}

function readCachedHosts(salt: string): HostSummary[] {
  if (!salt) return [];
  try {
    const raw = localStorage.getItem(HOSTS_KEY_PREFIX + salt);
    if (!raw) return [];
    const parsed: unknown = JSON.parse(raw);
    if (!Array.isArray(parsed)) return [];
    return parsed as HostSummary[];
  } catch {
    return [];
  }
}

function writeCachedHosts(salt: string, hosts: HostSummary[]): void {
  if (!salt) return;
  try {
    localStorage.setItem(HOSTS_KEY_PREFIX + salt, JSON.stringify(hosts));
  } catch {
    // quota / serialisation failures are non-fatal
  }
}

/**
 * setVaultSalt should be called by the unlock/status flows so the cache key
 * is up-to-date. Side-effect: also clears any cache scoped to a different salt.
 */
export function setVaultSalt(salt: string): void {
  if (!salt) return;
  try {
    const prev = localStorage.getItem(SALT_KEY);
    if (prev && prev !== salt) {
      // Different vault — drop the old cache to avoid leaking.
      localStorage.removeItem(HOSTS_KEY_PREFIX + prev);
    }
    localStorage.setItem(SALT_KEY, salt);
  } catch {
    // ignore
  }
}

/**
 * Clear all host caches. Call on sign-out / vault.locked when the renderer
 * shouldn't keep a viewable copy around (privacy on shared machines).
 */
export function clearAllHostCaches(): void {
  try {
    const keys: string[] = [];
    for (let i = 0; i < localStorage.length; i++) {
      const k = localStorage.key(i);
      if (k && (k.startsWith(HOSTS_KEY_PREFIX) || k === SALT_KEY)) keys.push(k);
    }
    for (const k of keys) localStorage.removeItem(k);
  } catch {
    // ignore
  }
}

// ── Pub/sub so multiple consumers share one fetch ────────────────────────

type Listener = (hosts: HostSummary[]) => void;
const listeners = new Set<Listener>();
let inFlight: Promise<HostSummary[]> | null = null;
let lastResult: HostSummary[] | null = null;

function notify(hosts: HostSummary[]): void {
  lastResult = hosts;
  for (const l of listeners) l(hosts);
}

/**
 * fetchHosts runs at most one listHosts request at a time across all
 * subscribers. Subsequent callers during the in-flight request piggy-back
 * on the same promise.
 */
async function fetchHosts(): Promise<HostSummary[]> {
  if (inFlight) return inFlight;
  inFlight = (async () => {
    try {
      const res = await window.sshthing.listHosts();
      const list = res.hosts ?? [];
      notify(list);
      const salt = readCachedSalt();
      if (salt) writeCachedHosts(salt, list);
      return list;
    } finally {
      inFlight = null;
    }
  })();
  return inFlight;
}

export interface UseHostsCacheResult {
  hosts: HostSummary[];
  /** True only on the very first fetch when there's no cached snapshot. */
  loading: boolean;
  /** Force a re-fetch (used after host CRUD ops). */
  refresh: () => Promise<void>;
}

export function useHostsCache(): UseHostsCacheResult {
  // Synchronous initial state from cache — paints before daemon replies.
  const initial = (() => {
    const salt = readCachedSalt();
    if (lastResult) return lastResult;
    if (!salt) return [];
    return readCachedHosts(salt);
  })();
  const [hosts, setHosts] = useState<HostSummary[]>(initial);
  const [loading, setLoading] = useState<boolean>(initial.length === 0);

  // Subscribe to shared updates.
  useEffect(() => {
    const l: Listener = (h) => setHosts(h);
    listeners.add(l);
    return () => { listeners.delete(l); };
  }, []);

  // Kick off a revalidation on mount if we don't have a recent result.
  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        await fetchHosts();
      } catch {
        // ignore; UI will show whatever cached state existed
      }
      if (!cancelled) setLoading(false);
    })();
    return () => { cancelled = true; };
  }, []);

  const refresh = useCallback(async () => {
    await fetchHosts();
  }, []);

  return { hosts, loading, refresh };
}
