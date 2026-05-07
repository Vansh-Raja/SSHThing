import { useCallback, useEffect, useState } from 'react';

/**
 * RPC error code returned when the user is not signed in to the cloud.
 * Matches the CodeNotSignedIn constant in the daemon RPC layer.
 */
const CODE_NOT_SIGNED_IN = -32010;

function isNotSignedInError(err: unknown): boolean {
  if (!err || typeof err !== 'object') return false;
  const e = err as { code?: number };
  return e.code === CODE_NOT_SIGNED_IN;
}

export interface UseTeamsResult {
  teams: TeamSummary[];
  loading: boolean;
  notSignedIn: boolean;
  error: string | null;
  reload: () => void;
}

export function useTeams(): UseTeamsResult {
  const [teams, setTeams] = useState<TeamSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [notSignedIn, setNotSignedIn] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const reload = useCallback(() => {
    setLoading(true);
    setError(null);
    setNotSignedIn(false);
    window.sshthing
      .teamsList()
      .then((result) => {
        setTeams(result.teams ?? []);
      })
      .catch((err: unknown) => {
        if (isNotSignedInError(err)) {
          setNotSignedIn(true);
        } else {
          setError((err as Error).message ?? 'Failed to load teams');
        }
      })
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    reload();
  }, [reload]);

  return { teams, loading, notSignedIn, error, reload };
}
