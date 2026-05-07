import { useCallback, useEffect, useState } from 'react';

export interface UseTeamMembersResult {
  members: TeamMember[];
  loading: boolean;
  error: string | null;
  reload: () => void;
}

export function useTeamMembers(teamId: string | null): UseTeamMembersResult {
  const [members, setMembers] = useState<TeamMember[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const reload = useCallback(() => {
    if (!teamId) return;
    setLoading(true);
    setError(null);
    window.sshthing
      .teamsMembersList(teamId)
      .then((result) => {
        setMembers(result.members ?? []);
      })
      .catch((err: unknown) => {
        setError((err as Error).message ?? 'Failed to load members');
      })
      .finally(() => setLoading(false));
  }, [teamId]);

  useEffect(() => {
    if (!teamId) {
      setMembers([]);
      return;
    }
    reload();
  }, [teamId, reload]);

  return { members, loading, error, reload };
}
