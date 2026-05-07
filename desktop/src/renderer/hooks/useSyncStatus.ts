/**
 * useSyncStatus — light wrapper around the daemon's sync.status RPC for
 * the top-bar indicator. Polls every 30 s and listens for sync.progress
 * notifications. Returns 'idle' when no provider is configured.
 */
import { useEffect, useState } from 'react';

export type SyncState = 'idle' | 'syncing' | 'ok' | 'error';

interface SyncStatus {
  state: SyncState;
  message?: string;
  lastSyncedAt?: number | null;
}

export function useSyncStatus(): SyncStatus {
  const [status, setStatus] = useState<SyncStatus>({ state: 'idle' });

  useEffect(() => {
    let cancelled = false;

    const refresh = async () => {
      try {
        const fn = window.sshthing.syncStatus;
        if (!fn) return;
        const res = await fn();
        if (cancelled) return;
        if (!res || res.provider === 'off') {
          setStatus({ state: 'idle' });
          return;
        }
        if (res.lastResultStatus === 'error') {
          setStatus({ state: 'error', message: res.lastMessage });
          return;
        }
        setStatus({
          state: 'ok',
          lastSyncedAt: res.lastResultAt ?? null,
        });
      } catch {
        // Sync RPC may not exist on this build; stay idle.
      }
    };

    void refresh();
    const id = setInterval(refresh, 30_000);

    const onNote = (_method: string, _params: unknown) => {
      // sync.progress notifications would update state here once wired.
    };
    const off = window.sshthing.onNotification?.(onNote);

    return () => {
      cancelled = true;
      clearInterval(id);
      off?.();
    };
  }, []);

  return status;
}
