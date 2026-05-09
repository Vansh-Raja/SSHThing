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
}

export interface UseHealthSchedulerReturn {
  enabled: boolean;
  setEnabled: (v: boolean) => void;
  intervalMs: number;
}

export function useHealthScheduler({
  hosts,
  health,
  intervalMs = DEFAULT_INTERVAL_MS,
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

    // Always auto-probe on mount / hosts change (TUI parity: auto-refresh on page enter).
    // Small delay so the UI renders cached health first, then probes update live.
    const immediateTimer = setTimeout(runCycle, 800);

    // Background interval only when scheduler is explicitly enabled.
    let intervalId: ReturnType<typeof setInterval> | undefined;
    if (enabled) {
      intervalId = setInterval(runCycle, intervalMs);
    }

    return () => {
      clearTimeout(immediateTimer);
      if (intervalId) clearInterval(intervalId);
    };
  }, [enabled, intervalMs]);

  return { enabled, setEnabled, intervalMs };
}
