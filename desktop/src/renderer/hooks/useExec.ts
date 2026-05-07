/**
 * useExec — runs a one-shot non-interactive command on a remote host.
 */
import { useCallback, useState } from 'react';

export interface UseExecReturn {
  loading: boolean;
  result: ExecResult | null;
  error: string;
  run: (hostId: string, cmd: string, timeoutMs?: number) => Promise<void>;
  reset: () => void;
}

export function useExec(): UseExecReturn {
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<ExecResult | null>(null);
  const [error, setError] = useState('');

  const run = useCallback(async (hostId: string, cmd: string, timeoutMs?: number) => {
    setLoading(true);
    setResult(null);
    setError('');
    try {
      const res = await window.sshthing.sessionExec(hostId, cmd, timeoutMs);
      setResult(res);
    } catch (err: unknown) {
      const e = err as Error;
      setError(e.message ?? 'Command failed');
    } finally {
      setLoading(false);
    }
  }, []);

  const reset = useCallback(() => {
    setLoading(false);
    setResult(null);
    setError('');
  }, []);

  return { loading, result, error, run, reset };
}
