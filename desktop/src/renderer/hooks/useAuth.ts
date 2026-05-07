/**
 * useAuth — subscribes to the current auth session.
 *
 * Polls auth.session() on mount and re-fetches on auth.signedIn / auth.signedOut
 * notifications from the daemon.
 */
import { useCallback, useEffect, useState } from 'react';
import { useNotifications } from './useNotifications';

export interface UseAuthResult {
  session: AuthSessionInfo | null;
  loading: boolean;
  refresh: () => void;
}

export function useAuth(): UseAuthResult {
  const [session, setSession] = useState<AuthSessionInfo | null>(null);
  const [loading, setLoading] = useState(true);

  const refresh = useCallback(() => {
    setLoading(true);
    window.sshthing
      .authSession()
      .then((res) => {
        setSession(res.session);
      })
      .catch(() => {
        setSession(null);
      })
      .finally(() => {
        setLoading(false);
      });
  }, []);

  useEffect(() => {
    refresh();
  }, [refresh]);

  // Re-fetch when the daemon emits sign-in or sign-out notifications.
  const handleNotification = useCallback(
    (method: string) => {
      if (method === 'auth.signedIn' || method === 'auth.signedOut') {
        refresh();
      }
    },
    [refresh]
  );
  useNotifications(handleNotification);

  return { session, loading, refresh };
}
