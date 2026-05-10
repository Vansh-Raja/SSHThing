/**
 * useHealthScheduler — background health-probe scheduler.
 *
 * - Toggle on/off stored in localStorage under key 'health:scheduler:enabled'.
 * - Polls every `intervalMs` (default 5 minutes).
 * - Caps concurrent in-flight probes at MAX_CONCURRENT (5).
 * - Skips a host that already has an in-flight probe.
 */
import { useCallback, useEffect, useRef, useState } from 'react';
import type { UseHealthReturn } from './useHealth';

const STORAGE_KEY = 'health:scheduler:enabled';
const DEFAULT_INTERVAL_MS = 5 * 60 * 1000; // 5 minutes
const MAX_CONCURRENT = 5;

function readEnabled(): boolean {
  try {
    return localStorage.getItem(STORAGE_KEY) === 'true';
  } catch {
    return false;
  }
}

function writeEnabled(v: boolean): void {
  try {
    localStorage.setItem(STORAGE_KEY, v ? 'true' : 'false');
  } catch {
    // localStorage may be unavailable in some Electron contexts
  }
}

interface UseHealthSchedulerOptions {
  hosts: HostSummary[];
  health: UseHealthReturn;
  intervalMs?: number;
  /**
   * Stable identifier for the host set being scheduled. Used to dedupe the
   * "auto-probe on mount" so we don't re-probe every time the user navigates
   * back to this page. Pass "personal" for the personal host list and
   * `team:<teamId>` for each team's host list. The auto-probe runs once per
   * (key, app-launch); the periodic background scheduler still runs every
   * mount when enabled in Settings.
   */
  scopeKey?: string;
}

export interface UseHealthSchedulerReturn {
  enabled: boolean;
  setEnabled: (v: boolean) => void;
  intervalMs: number;
}

// Module-scope memory of which scope keys have already had their initial
// auto-probe this app launch. A page remount within the same app session
// won't trigger another full sweep — matches the user's preference of
// "probe once per app launch (per scope)".
const autoProbedScopes = new Set<string>();

/** Reset the auto-probe memory. Useful for a manual "Refresh all" action. */
export function resetAutoProbedScopes(): void { autoProbedScopes.clear(); }

export function useHealthScheduler({
  hosts,
  health,
  intervalMs = DEFAULT_INTERVAL_MS,
  scopeKey,
}: UseHealthSchedulerOptions): UseHealthSchedulerReturn {
  const [enabled, setEnabledState] = useState<boolean>(readEnabled);
  // Stable ref to avoid recreating the interval callback on every render.
  const hostsRef = useRef<HostSummary[]>(hosts);
  const probingRef = useRef<Set<string>>(health.probing);
  const probeRef = useRef<UseHealthReturn['probe']>(health.probe);

  useEffect(() => { hostsRef.current = hosts; }, [hosts]);
  useEffect(() => { probingRef.current = health.probing; }, [health.probing]);
  useEffect(() => { probeRef.current = health.probe; }, [health.probe]);

  const setEnabled = useCallback((v: boolean) => {
    writeEnabled(v);
    setEnabledState(v);
  }, []);

  useEffect(() => {
    const runCycle = () => {
      const currentHosts = hostsRef.current;
      const currentProbing = probingRef.current;
      // Determine candidates: not already probing.
      const candidates = currentHosts.filter((h) => !currentProbing.has(h.id));
      // Cap to MAX_CONCURRENT minus current in-flight.
      const slots = MAX_CONCURRENT - currentProbing.size;
      if (slots <= 0) return;
      const batch = candidates.slice(0, slots);
      for (const host of batch) {
        void probeRef.current(host.id);
      }
    };

    // Auto-probe ONCE per (scope, app-launch). If the caller supplied a
    // scopeKey we check the module-level set first; without a key we keep
    // the legacy probe-on-every-mount behaviour (callers opting in to the
    // dedupe should always pass a key).
    let immediateTimer: ReturnType<typeof setTimeout> | undefined;
    if (!scopeKey || !autoProbedScopes.has(scopeKey)) {
      // Small delay so the UI renders cached health first, then probes update live.
      immediateTimer = setTimeout(() => {
        runCycle();
        if (scopeKey) autoProbedScopes.add(scopeKey);
      }, 800);
    }

    // Background interval only when scheduler is explicitly enabled in Settings.
    let intervalId: ReturnType<typeof setInterval> | undefined;
    if (enabled) {
      intervalId = setInterval(runCycle, intervalMs);
    }

    return () => {
      if (immediateTimer) clearTimeout(immediateTimer);
      if (intervalId) clearInterval(intervalId);
    };
  }, [enabled, intervalMs, scopeKey]);

  return { enabled, setEnabled, intervalMs };
}
