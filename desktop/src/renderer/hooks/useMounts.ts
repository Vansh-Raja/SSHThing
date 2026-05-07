/**
 * useMounts — tracks active SSHFS mounts.
 * Provides start, stop, and refresh operations.
 */
import { useCallback, useState } from 'react';

export interface UseMountsReturn {
  mounts: MountSummary[];
  mounting: Set<string>;
  unmounting: Set<string>;
  mountStart: (hostId: string, remotePath: string) => Promise<MountSummary>;
  mountStop: (hostId: string) => Promise<void>;
  loadMounts: () => Promise<void>;
}

export function useMounts(): UseMountsReturn {
  const [mounts, setMounts] = useState<MountSummary[]>([]);
  const [mounting, setMounting] = useState<Set<string>>(new Set());
  const [unmounting, setUnmounting] = useState<Set<string>>(new Set());

  const loadMounts = useCallback(async () => {
    try {
      const { mounts: list } = await window.sshthing.mountList();
      setMounts(list);
    } catch {
      // Mount list RPC unavailable — fail silently.
    }
  }, []);

  const mountStart = useCallback(async (hostId: string, remotePath: string): Promise<MountSummary> => {
    setMounting((prev) => {
      const next = new Set(prev);
      next.add(hostId);
      return next;
    });
    try {
      const summary = await window.sshthing.mountStart(hostId, remotePath);
      setMounts((prev) => {
        // Replace or append.
        const without = prev.filter((m) => m.hostId !== hostId);
        return [...without, summary];
      });
      return summary;
    } finally {
      setMounting((prev) => {
        const next = new Set(prev);
        next.delete(hostId);
        return next;
      });
    }
  }, []);

  const mountStop = useCallback(async (hostId: string): Promise<void> => {
    setUnmounting((prev) => {
      const next = new Set(prev);
      next.add(hostId);
      return next;
    });
    try {
      await window.sshthing.mountStop(hostId);
      setMounts((prev) => prev.filter((m) => m.hostId !== hostId));
    } finally {
      setUnmounting((prev) => {
        const next = new Set(prev);
        next.delete(hostId);
        return next;
      });
    }
  }, []);

  return { mounts, mounting, unmounting, mountStart, mountStop, loadMounts };
}
