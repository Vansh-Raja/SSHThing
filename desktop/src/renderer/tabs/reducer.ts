/**
 * Pure reducer for the tab manager. All transitions go through here so
 * (a) we get a single auditable place that defines the rules and
 * (b) we can unit-test it without React in the loop.
 *
 * Invariants enforced:
 *   - The Hosts base tab is always present at index 0 and never closes.
 *   - `activeId` always references an existing tab.
 *   - Singleton kinds (settings, profile, etc.) only have one tab; OPEN
 *     for an already-open singleton activates the existing entry.
 *   - For non-singleton kinds, OPEN dedupes by `state` identity (e.g. a
 *     `host-editor` for the same hostId activates the existing tab).
 *   - HYDRATE replaces the whole state but the post-hydrate state is
 *     re-validated: if the persisted active id is missing, fall back to
 *     the Hosts tab; if Hosts is missing, prepend it.
 */

import {
  HOSTS_TAB_ID,
  SINGLETON_KINDS,
  type Tab,
  type TabsAction,
  type TabsState,
  type TabState,
} from './types';

/** Initial state — Hosts only. */
export function initialState(now: number = Date.now()): TabsState {
  const hosts: Tab = {
    id: HOSTS_TAB_ID,
    kind: 'hosts',
    title: 'Hosts',
    closable: false,
    dirty: false,
    state: { kind: 'hosts' },
    createdAt: now,
    lastActiveAt: now,
  };
  return { tabs: [hosts], activeId: hosts.id };
}

export function tabsReducer(state: TabsState, action: TabsAction): TabsState {
  switch (action.type) {
    case 'OPEN': {
      const incoming: Tab = {
        ...action.tab,
        createdAt: action.tab.createdAt ?? Date.now(),
        lastActiveAt: action.tab.lastActiveAt ?? Date.now(),
      };

      // Dedupe: singletons by kind, non-singletons by state-equivalence.
      const existing = SINGLETON_KINDS.has(incoming.kind)
        ? state.tabs.find((t) => t.kind === incoming.kind)
        : state.tabs.find((t) => sameInstance(t, incoming));
      if (existing) {
        return { ...state, activeId: existing.id, tabs: bumpActive(state.tabs, existing.id) };
      }

      // Insert after the active tab (or at the end if active is the last
      // tab) — matches VSCode's "open beside the current" behaviour.
      const activeIdx = state.tabs.findIndex((t) => t.id === state.activeId);
      const insertAt = activeIdx >= 0 ? activeIdx + 1 : state.tabs.length;
      const next = [...state.tabs];
      next.splice(insertAt, 0, incoming);
      return { tabs: next, activeId: incoming.id };
    }

    case 'ACTIVATE': {
      const exists = state.tabs.some((t) => t.id === action.id);
      if (!exists) return state;
      return { ...state, activeId: action.id, tabs: bumpActive(state.tabs, action.id) };
    }

    case 'CLOSE': {
      const idx = state.tabs.findIndex((t) => t.id === action.id);
      if (idx < 0) return state;
      const target = state.tabs[idx]!;
      if (!target.closable) return state; // refuse to close the Hosts base tab
      const next = state.tabs.filter((t) => t.id !== action.id);
      // If we just closed the active tab, activate the neighbor; prefer
      // the one to the LEFT to mirror VSCode (right-bias feels jumpy).
      const newActive =
        state.activeId !== action.id
          ? state.activeId
          : pickNeighborId(state.tabs, idx);
      return { tabs: next, activeId: newActive };
    }

    case 'CLOSE_ALL_CLOSABLE': {
      const kept = state.tabs.filter((t) => !t.closable);
      // The Hosts tab survives; activate it.
      const fallback = kept[0]?.id ?? HOSTS_TAB_ID;
      return { tabs: kept, activeId: fallback };
    }

    case 'RENAME': {
      const next = state.tabs.map((t) =>
        t.id === action.id && action.title.trim() ? { ...t, title: action.title } : t,
      );
      return { ...state, tabs: next };
    }

    case 'SET_DIRTY': {
      const next = state.tabs.map((t) =>
        t.id === action.id && t.dirty !== action.dirty ? { ...t, dirty: action.dirty } : t,
      );
      return { ...state, tabs: next };
    }

    case 'REPLACE_STATE': {
      const next = state.tabs.map((t) =>
        t.id === action.id ? { ...t, state: action.state, lastActiveAt: Date.now() } : t,
      );
      return { ...state, tabs: next };
    }

    case 'HYDRATE': {
      // Validate: ensure Hosts is present and active id resolves.
      let { tabs, activeId } = action.state;
      const hasHosts = tabs.some((t) => t.id === HOSTS_TAB_ID && t.kind === 'hosts');
      if (!hasHosts) {
        tabs = [...initialState().tabs, ...tabs.filter((t) => t.id !== HOSTS_TAB_ID)];
      }
      if (!tabs.some((t) => t.id === activeId)) {
        activeId = HOSTS_TAB_ID;
      }
      return { tabs, activeId };
    }

    default:
      return state;
  }
}

/**
 * sameInstance — does `a` and `b` refer to the same logical tab?
 * For non-singleton kinds the comparison is on the scoped fields of the
 * state shape (hostId for editors, sessionId for terminals, etc.).
 */
function sameInstance(a: Tab, b: Tab): boolean {
  if (a.kind !== b.kind) return false;
  return sameState(a.state, b.state);
}

function sameState(a: TabState, b: TabState): boolean {
  if (a.kind !== b.kind) return false;
  switch (a.kind) {
    case 'terminal':
      return b.kind === 'terminal' && a.sessionId === b.sessionId;
    case 'host-editor':
      return b.kind === 'host-editor' && a.hostId === b.hostId;
    case 'host-editor-team':
      return b.kind === 'host-editor-team' && a.teamId === b.teamId && a.hostId === b.hostId;
    case 'exec':
      return b.kind === 'exec' && a.hostId === b.hostId;
    default:
      return true; // singleton kinds — same kind ⇒ same instance
  }
}

function bumpActive(tabs: Tab[], id: string): Tab[] {
  const now = Date.now();
  return tabs.map((t) => (t.id === id ? { ...t, lastActiveAt: now } : t));
}

function pickNeighborId(tabs: Tab[], removedIdx: number): string {
  const left = tabs[removedIdx - 1];
  if (left) return left.id;
  const right = tabs[removedIdx + 1];
  if (right) return right.id;
  return HOSTS_TAB_ID;
}
