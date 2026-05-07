import { useCallback, useEffect, useState } from 'react';

export interface UseCredentialRosterResult {
  roster: TeamHostCredentialRosterEntry[];
  loading: boolean;
  error: string | null;
  reload: () => void;
}

export function useCredentialRoster(hostId: string | null): UseCredentialRosterResult {
  const [roster, setRoster] = useState<TeamHostCredentialRosterEntry[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const reload = useCallback(() => {
    if (!hostId) return;
    setLoading(true);
    setError(null);
    window.sshthing
      .teamsHostsRosterList(hostId)
      .then((result) => {
        setRoster(result.roster ?? []);
      })
      .catch((err: unknown) => {
        setError((err as Error).message ?? 'Failed to load credential roster');
      })
      .finally(() => setLoading(false));
  }, [hostId]);

  useEffect(() => {
    if (!hostId) {
      setRoster([]);
      return;
    }
    reload();
  }, [hostId, reload]);

  return { roster, loading, error, reload };
}
