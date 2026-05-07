/**
 * useTransfers — tracks active file transfers by subscribing to
 * transfer.progress daemon notifications.
 */
import { useCallback, useEffect, useState } from 'react';

export interface TransferEntry {
  transferId: string;
  hostId: string;
  hostLabel: string;
  direction: 'upload' | 'download';
  local: string;
  remote: string;
  status: 'started' | 'finished' | 'failed';
  error?: string;
  startedAt: number;
}

export interface UseTransfersReturn {
  transfers: TransferEntry[];
  startUpload: (hostId: string, hostLabel: string, local: string, remote: string, recursive?: boolean, preserve?: boolean) => Promise<string>;
  startDownload: (hostId: string, hostLabel: string, local: string, remote: string, recursive?: boolean, preserve?: boolean) => Promise<string>;
  cancelTransfer: (transferId: string) => Promise<void>;
  dismiss: (transferId: string) => void;
  clearFinished: () => void;
}

export function useTransfers(): UseTransfersReturn {
  const [transfers, setTransfers] = useState<TransferEntry[]>([]);

  // Subscribe to transfer.progress daemon notifications.
  useEffect(() => {
    const unsub = window.sshthing.onNotification((method: string, params: unknown) => {
      if (method !== 'transfer.progress') return;
      const p = params as TransferProgress;
      setTransfers((prev) => {
        const idx = prev.findIndex((t) => t.transferId === p.transferId);
        if (idx === -1) {
          // New entry (started notification arrives before we update state).
          return prev;
        }
        const next = [...prev];
        next[idx] = {
          ...next[idx]!,
          status: p.status,
          error: p.error,
        };
        return next;
      });
    });
    return unsub;
  }, []);

  const startUpload = useCallback(async (
    hostId: string,
    hostLabel: string,
    local: string,
    remote: string,
    recursive = false,
    preserve = false,
  ): Promise<string> => {
    const { transferId } = await window.sshthing.transferUpload({ hostId, local, remote, recursive, preserve });
    const entry: TransferEntry = {
      transferId,
      hostId,
      hostLabel,
      direction: 'upload',
      local,
      remote,
      status: 'started',
      startedAt: Date.now(),
    };
    setTransfers((prev) => [...prev, entry]);
    return transferId;
  }, []);

  const startDownload = useCallback(async (
    hostId: string,
    hostLabel: string,
    local: string,
    remote: string,
    recursive = false,
    preserve = false,
  ): Promise<string> => {
    const { transferId } = await window.sshthing.transferDownload({ hostId, local, remote, recursive, preserve });
    const entry: TransferEntry = {
      transferId,
      hostId,
      hostLabel,
      direction: 'download',
      local,
      remote,
      status: 'started',
      startedAt: Date.now(),
    };
    setTransfers((prev) => [...prev, entry]);
    return transferId;
  }, []);

  const cancelTransfer = useCallback(async (transferId: string): Promise<void> => {
    // Optimistically mark as failed in UI; the daemon will emit a failed notification too.
    setTransfers((prev) => prev.map((t) =>
      t.transferId === transferId
        ? { ...t, status: 'failed' as const, error: 'Cancelled' }
        : t,
    ));
    try {
      await window.sshthing.transferCancel(transferId);
    } catch {
      // If it already finished, the daemon returns an error — no-op.
    }
  }, []);

  const dismiss = useCallback((transferId: string) => {
    setTransfers((prev) => prev.filter((t) => t.transferId !== transferId));
  }, []);

  const clearFinished = useCallback(() => {
    setTransfers((prev) => prev.filter((t) => t.status === 'started'));
  }, []);

  return { transfers, startUpload, startDownload, cancelTransfer, dismiss, clearFinished };
}
