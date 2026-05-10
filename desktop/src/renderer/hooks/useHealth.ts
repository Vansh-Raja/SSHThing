/**
 * useHealth — provides health probe and list operations.
 * Maintains a Map<hostId, HealthResult> of the most recent result per host.
 */
import { useCallback, useRef, useState } from 'react';

export type HealthMap = Map<string, HealthResult>;

export interface UseHealthReturn {
  healthMap: HealthMap;
  probing: Set<string>;
  probe: (hostId: string) => Promise<void>;
  loadAll: () => Promise<void>;
  setResult: (result: HealthResult) => void;
}

export function useHealth(): UseHealthReturn {
  const [healthMap, setHealthMap] = useState<HealthMap>(new Map());
  const [probing, setProbing] = useState<Set<string>>(new Set());
  // Synchronous mirror of `probing` so the dedupe check below can read the
  // current in-flight set without waiting for React to commit the setState.
  // Without this ref, two probe(hostId) calls in the same tick both pass
  // the "not in set" check and fire duplicate healthProbe RPCs.
  const inFlightRef = useRef<Set<string>>(new Set());

  const setResult = useCallback((result: HealthResult) => {
    setHealthMap((prev) => {
      const next = new Map(prev);
      next.set(result.hostId, result);
      return next;
    });
  }, []);

  const probe = useCallback(async (hostId: string) => {
    // Dedupe against the synchronous ref, not React state. The previous
    // implementation only short-circuited the setProbing callback, so the
    // function body kept running and a duplicate RPC fired.
    if (inFlightRef.current.has(hostId)) return;
    inFlightRef.current.add(hostId);
    setProbing((prev) => {
      const next = new Set(prev);
      next.add(hostId);
      return next;
    });
    try {
      const result = await window.sshthing.healthProbe(hostId);
      setResult(result);
    } finally {
      inFlightRef.current.delete(hostId);
      setProbing((prev) => {
        const next = new Set(prev);
        next.delete(hostId);
        return next;
      });
    }
  }, [setResult]);

  const loadAll = useCallback(async () => {
    try {
      const { results } = await window.sshthing.healthList();
      setHealthMap((prev) => {
        const next = new Map(prev);
        for (const r of results) {
          next.set(r.hostId, r);
        }
        return next;
      });
    } catch {
      // Health list RPC unavailable — fail silently.
    }
  }, []);

  return { healthMap, probing, probe, loadAll, setResult };
}
