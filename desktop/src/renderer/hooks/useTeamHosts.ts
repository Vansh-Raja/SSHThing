import { useCallback, useEffect, useState } from 'react';

export interface UseTeamHostsResult {
  hosts: TeamHost[];
  loading: boolean;
  error: string | null;
  reload: () => void;
}

export function useTeamHosts(teamId: string | null): UseTeamHostsResult {
  const [hosts, setHosts] = useState<TeamHost[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const reload = useCallback(() => {
    if (!teamId) return;
    setLoading(true);
    setError(null);
    window.sshthing
      .teamsHostsList(teamId)
      .then((result) => {
        setHosts(result.hosts ?? []);
      })
      .catch((err: unknown) => {
        setError((err as Error).message ?? 'Failed to load team hosts');
      })
      .finally(() => setLoading(false));
  }, [teamId]);

  useEffect(() => {
    if (!teamId) {
      setHosts([]);
      return;
    }
    reload();
  }, [teamId, reload]);

  return { hosts, loading, error, reload };
}
