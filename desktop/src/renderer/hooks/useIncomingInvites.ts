/**
 * useIncomingInvites — polls for incoming team invites.
 *
 * Loads invite lists for every team the current user belongs to and
 * returns all incoming (pending) invites that they haven't yet accepted.
 *
 * NOTE: This hook is standalone and intentionally NOT mounted anywhere yet.
 * It is wired up in InvitesBadge.tsx which will be mounted in the topbar
 * once Wave 2A makes that boundary available.
 * TODO Wave 2A: mount <InvitesBadge /> in the topbar.
 */
import { useCallback, useEffect, useState } from 'react';

export interface UseIncomingInvitesReturn {
  incomingInvites: TeamInvite[];
  count: number;
  loading: boolean;
  reload: () => void;
}

/**
 * Polls for incoming invites across all teams the user is a member of.
 *
 * @param pollIntervalMs - How often to refresh (default: 60 s). Pass 0 to
 *   disable polling.
 */
export function useIncomingInvites(pollIntervalMs = 60_000): UseIncomingInvitesReturn {
  const [incomingInvites, setIncomingInvites] = useState<TeamInvite[]>([]);
  const [loading, setLoading] = useState(false);

  const reload = useCallback(async () => {
    setLoading(true);
    try {
      const { teams } = await window.sshthing.teamsList();
      const allIncoming: TeamInvite[] = [];

      await Promise.allSettled(
        teams.map(async (team) => {
          try {
            const list = await window.sshthing.teamsInvitesList(team.id);
            allIncoming.push(...list.incoming.filter((inv) => inv.status === 'pending'));
          } catch {
            // Skip teams that fail — don't block the whole poll.
          }
        }),
      );

      setIncomingInvites(allIncoming);
    } catch {
      // If teamsList fails (e.g. not signed in), stay silent.
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void reload();

    if (pollIntervalMs > 0) {
      const interval = setInterval(() => { void reload(); }, pollIntervalMs);
      return () => clearInterval(interval);
    }
    return undefined;
  }, [reload, pollIntervalMs]);

  return {
    incomingInvites,
    count: incomingInvites.length,
    loading,
    reload,
  };
}
