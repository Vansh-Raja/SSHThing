import { useEffect } from 'react';

type NotificationCallback = (method: string, params: unknown) => void;

/**
 * Subscribe to daemon push notifications for the lifetime of the component.
 */
export function useNotifications(cb: NotificationCallback): void {
  useEffect(() => {
    if (!window.sshthing) return;
    const unsub = window.sshthing.onNotification(cb);
    return unsub;
  }, [cb]);
}
