import { useCallback, useEffect, useState } from 'react';

export interface UseTeamAuditResult {
  events: TeamAuditEvent[];
  loading: boolean;
  error: string | null;
  reload: () => void;
}

export function useTeamAudit(teamId: string | null): UseTeamAuditResult {
  const [events, setEvents] = useState<TeamAuditEvent[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const reload = useCallback(() => {
    if (!teamId) return;
    setLoading(true);
    setError(null);
    window.sshthing
      .teamsAuditList(teamId)
      .then((result) => {
        setEvents(result.events ?? []);
      })
      .catch((err: unknown) => {
        setError((err as Error).message ?? 'Failed to load audit log');
      })
      .finally(() => setLoading(false));
  }, [teamId]);

  useEffect(() => {
    if (!teamId) {
      setEvents([]);
      return;
    }
    reload();
  }, [teamId, reload]);

  return { events, loading, error, reload };
}
