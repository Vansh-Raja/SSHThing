/**
 * Type definitions for the global tab manager.
 *
 * The renderer has ONE tab strip (extracted from the previous in-Hosts
 * terminal-tab UI). Every workspace view — Hosts, Settings, Profile,
 * Tokens, Keys, Teams, the host editor, the exec runner, and individual
 * SSH terminal sessions — is rendered inside this strip as a tab.
 *
 * The Hosts tab is special: always present, leftmost, non-closable. It's
 * the "home" surface and shows the existing sidebar+detail layout when
 * no other tab is active.
 *
 * Singleton kinds (`settings`, `profile`, `tokens`, `keys`, `teams`)
 * focus the existing tab if one is already open instead of creating a
 * duplicate. Non-singleton kinds (`terminal`, `host-editor`, `exec`)
 * support multiple instances scoped by `state` (e.g. one host-editor
 * tab per host being edited).
 */

export type TabKind =
  | 'hosts' // base tab, always present, non-closable
  | 'settings'
  | 'profile'
  | 'tokens'
  | 'keys'
  | 'teams'
  | 'terminal' // SSH session, one per session id
  | 'host-editor' // create/edit a personal host (or "new")
  | 'host-editor-team' // create/edit a team host
  | 'exec'; // long-running remote command runner

/** Per-kind state shape. Strongly typed so callers can't smuggle nonsense. */
export type TabState =
  | { kind: 'hosts' }
  | { kind: 'settings' }
  | { kind: 'profile' }
  | { kind: 'tokens'; mode?: 'personal' | 'teams' }
  | { kind: 'keys' }
  | { kind: 'teams' }
  | { kind: 'terminal'; sessionId: string; hostId: string; hostLabel: string }
  | { kind: 'host-editor'; hostId: string | null /* null = new */ }
  | { kind: 'host-editor-team'; teamId: string; hostId: string | null }
  | { kind: 'exec'; hostId: string; hostLabel: string };

export interface Tab {
  /** Stable per-tab id; persists across reloads for restorable kinds. */
  id: string;
  kind: TabKind;
  /** What's shown in the tab strip. Tabs can rename themselves (e.g. terminal honoring OSC 1/2). */
  title: string;
  /** Hosts is `false`. All other kinds default to `true`. */
  closable: boolean;
  /**
   * True when the tab has unsaved state (only meaningful for kinds
   * that surface a "Discard changes?" prompt on close — host-editor
   * primarily). Persisted to disk so a forced reload re-prompts.
   */
  dirty: boolean;
  /** Kind-specific scope/data. */
  state: TabState;
  /** Wall-clock ms — used to break ties in MRU activation. */
  createdAt: number;
  /** Updated whenever the tab becomes active or its state mutates. */
  lastActiveAt: number;
}

export interface TabsState {
  /** Ordered left-to-right. Hosts is always at index 0. */
  tabs: Tab[];
  /** id of the currently visible tab. Always references a real entry. */
  activeId: string;
}

/**
 * Reducer actions. Kept narrow on purpose — there's no "update arbitrary
 * tab field" escape hatch; each callsite uses the action that captures
 * its intent so debugging stays sane.
 */
export type TabsAction =
  | { type: 'OPEN'; tab: Omit<Tab, 'createdAt' | 'lastActiveAt'> & Partial<Pick<Tab, 'createdAt' | 'lastActiveAt'>> }
  | { type: 'ACTIVATE'; id: string }
  | { type: 'CLOSE'; id: string }
  | { type: 'CLOSE_ALL_CLOSABLE' }
  | { type: 'RENAME'; id: string; title: string }
  | { type: 'SET_DIRTY'; id: string; dirty: boolean }
  | { type: 'REPLACE_STATE'; id: string; state: TabState }
  | { type: 'HYDRATE'; state: TabsState };

/** The fixed `id` of the always-present Hosts tab. Use this constant rather than a magic string. */
export const HOSTS_TAB_ID = 'tab-hosts-base';

/** Kinds that are singletons — opening another time focuses the existing tab. */
export const SINGLETON_KINDS: ReadonlySet<TabKind> = new Set([
  'hosts',
  'settings',
  'profile',
  'tokens',
  'keys',
  'teams',
]);

/**
 * Default human-readable title for a kind. Used when callers open a
 * singleton tab without supplying a title.
 */
export const DEFAULT_TITLE: Record<TabKind, string> = {
  'hosts': 'Hosts',
  'settings': 'Settings',
  'profile': 'Profile',
  'tokens': 'Tokens',
  'keys': 'Keys',
  'teams': 'Teams',
  'terminal': 'Terminal',
  'host-editor': 'New host',
  'host-editor-team': 'New team host',
  'exec': 'Run command',
};
