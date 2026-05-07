/**
 * useHealth — provides health probe and list operations.
 * Maintains a Map<hostId, HealthResult> of the most recent result per host.
 */
import { useCallback, useState } from 'react';

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

  const setResult = useCallback((result: HealthResult) => {
    setHealthMap((prev) => {
      const next = new Map(prev);
      next.set(result.hostId, result);
      return next;
    });
  }, []);

  const probe = useCallback(async (hostId: string) => {
    // Skip if already in-flight for this specific host.
    setProbing((prev) => {
      if (prev.has(hostId)) return prev;
      const next = new Set(prev);
      next.add(hostId);
      return next;
    });
    try {
      const result = await window.sshthing.healthProbe(hostId);
      setResult(result);
    } finally {
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
