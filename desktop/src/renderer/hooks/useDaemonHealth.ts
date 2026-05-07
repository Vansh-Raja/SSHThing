/**
 * useDaemonHealth — tracks whether the Go sidecar daemon is still alive.
 *
 * The main process broadcasts 'app:daemon-exited' when the daemon process
 * exits unexpectedly. The preload exposes window.sshthing.onDaemonExit to
 * subscribe. If that API is not present (older preload builds or not yet
 * wired by the main-process agent), the hook stays in the 'alive' state.
 *
 * NOTE: onDaemonExit is not yet in the SSHThingAPI type definition.
 * It is accessed via a type-cast on the extended API shape below so that
 * TypeScript compilation succeeds before the preload types are updated.
 */
import { useEffect, useState } from 'react';

/** Extended type that includes the optional daemon-exit callback. */
type SSHThingExtended = typeof window.sshthing & {
  onDaemonExit?: (cb: () => void) => () => void;
};

export type DaemonHealthState = 'alive' | 'disconnected';

export function useDaemonHealth(): DaemonHealthState {
  const [state, setState] = useState<DaemonHealthState>('alive');

  useEffect(() => {
    const api = window.sshthing as SSHThingExtended;
    if (typeof api.onDaemonExit !== 'function') {
      // Older preload or preload hasn't wired this yet — safe to ignore.
      return;
    }
    const unsub = api.onDaemonExit(() => {
      setState('disconnected');
    });
    return unsub;
  }, []);

  return state;
}
