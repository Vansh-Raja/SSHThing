/**
 * localStorage-backed restore for the tab manager.
 *
 * What persists:
 *   - The list of open tabs minus their kind-specific state.
 *   - For singleton kinds (settings, profile, tokens, keys, teams) we
 *     restore both the entry AND any persisted `dirty` flag.
 *   - For `host-editor` and `host-editor-team` we restore the
 *     hostId reference; the page itself re-fetches the host on mount.
 *   - For `exec` we restore the host scope; in-flight runs are NOT
 *     resumed (the daemon doesn't support reattach for one-shots).
 *   - For `terminal` we deliberately do NOT persist — daemon sessions
 *     don't survive an app restart, so restoring a terminal tab would
 *     surface a stale session id and look broken.
 */

import {
  HOSTS_TAB_ID,
  type Tab,
  type TabKind,
  type TabsState,
} from './types';

const STORAGE_KEY = 'sshthing.tabs.v1';
const RESTORABLE: ReadonlySet<TabKind> = new Set([
  'hosts',
  'settings',
  'profile',
  'tokens',
  'keys',
  'teams',
  'host-editor',
  'host-editor-team',
  'exec',
]);

interface PersistedShape {
  v: 1;
  tabs: Tab[];
  activeId: string;
}

export function saveTabs(state: TabsState): void {
  try {
    const persistable = state.tabs.filter((t) => RESTORABLE.has(t.kind));
    const activeStillThere = persistable.some((t) => t.id === state.activeId);
    const payload: PersistedShape = {
      v: 1,
      tabs: persistable,
      activeId: activeStillThere ? state.activeId : HOSTS_TAB_ID,
    };
    localStorage.setItem(STORAGE_KEY, JSON.stringify(payload));
  } catch {
    // Quota exhausted or sandboxed — restore-on-launch is a polish
    // feature, not a correctness one. Silently drop.
  }
}

export function loadTabs(): TabsState | null {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as PersistedShape;
    if (parsed.v !== 1 || !Array.isArray(parsed.tabs)) return null;
    // Guard against drift: drop entries whose kind isn't in our set or
    // whose state shape's discriminant doesn't match the kind. This
    // protects users who downgrade from a future version that added a
    // new kind.
    const filtered = parsed.tabs.filter(
      (t) => RESTORABLE.has(t.kind) && t.state?.kind === t.kind,
    );
    return { tabs: filtered, activeId: parsed.activeId };
  } catch {
    return null;
  }
}

export function clearTabs(): void {
  try { localStorage.removeItem(STORAGE_KEY); } catch { /* ignore */ }
}
