/**
 * TabsProvider — owns the tab manager state and exposes hook accessors.
 *
 * Two consumer surfaces:
 *   - `useTabs()` for components that need to read the list / active tab.
 *   - `useTabActions()` for components that just dispatch (Settings rail
 *     button, Hosts page when it opens an editor, etc.) — split so the
 *     dispatcher reference is stable and callers don't re-render on
 *     every tab list change.
 *
 * Persistence: hydrate from localStorage on mount, snapshot back via a
 * debounced effect on every state change. We tolerate a 1-frame mismatch
 * between memory + disk in exchange for fewer writes.
 */
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useReducer,
  useRef,
  type ReactNode,
} from 'react';
import { initialState, tabsReducer } from '../tabs/reducer';
import { loadTabs, saveTabs } from '../tabs/persistence';
import {
  DEFAULT_TITLE,
  HOSTS_TAB_ID,
  SINGLETON_KINDS,
  type Tab,
  type TabKind,
  type TabState,
  type TabsState,
} from '../tabs/types';

interface TabsContextValue {
  state: TabsState;
}

interface TabActionsValue {
  /**
   * Open or focus a tab. Singleton kinds activate the existing tab if
   * one is open; non-singletons dedupe by state-equivalence (same
   * sessionId for terminals, same hostId for editors, etc.).
   * Returns the id of the tab that ended up active.
   */
  open: (kind: TabKind, state: TabState, opts?: { title?: string }) => string;
  activate: (id: string) => void;
  close: (id: string) => void;
  closeAllClosable: () => void;
  rename: (id: string, title: string) => void;
  setDirty: (id: string, dirty: boolean) => void;
  replaceState: (id: string, state: TabState) => void;
}

const StateContext = createContext<TabsContextValue | null>(null);
const ActionsContext = createContext<TabActionsValue | null>(null);

interface TabsProviderProps {
  children: ReactNode;
  /** Hook for tests — skip localStorage hydration. */
  skipPersistence?: boolean;
}

export function TabsProvider({ children, skipPersistence }: TabsProviderProps) {
  const [state, dispatch] = useReducer(
    tabsReducer,
    undefined,
    () => {
      if (skipPersistence) return initialState();
      const persisted = loadTabs();
      if (!persisted) return initialState();
      // Run through the reducer's HYDRATE path to enforce invariants
      // (Hosts present, active id valid).
      return tabsReducer(initialState(), { type: 'HYDRATE', state: persisted });
    },
  );

  // Debounced persistence — coalesce rapid bursts (e.g. closing several
  // tabs in succession) into a single write.
  const writeTimer = useRef<number | undefined>(undefined);
  useEffect(() => {
    if (skipPersistence) return;
    window.clearTimeout(writeTimer.current);
    writeTimer.current = window.setTimeout(() => saveTabs(state), 200);
  }, [state, skipPersistence]);

  // Stable id generator. Singleton tabs always use a kind-derived id so
  // a freshly-opened "settings" tab matches a previously-persisted one.
  const idGen = useRef(0);
  const newId = useCallback(() => `tab-${++idGen.current}-${Date.now().toString(36)}`, []);
  const idForState = useCallback(
    (kind: TabKind, st: TabState): string => {
      if (SINGLETON_KINDS.has(kind)) return `tab-singleton-${kind}`;
      switch (st.kind) {
        case 'terminal': return `tab-term-${st.sessionId}`;
        case 'host-editor': return `tab-editor-${st.hostId ?? 'new'}-${newId()}`;
        case 'host-editor-team': return `tab-editor-team-${st.teamId}-${st.hostId ?? 'new'}-${newId()}`;
        case 'exec': return `tab-exec-${st.hostId}-${newId()}`;
        default: return newId();
      }
    },
    [newId],
  );

  const actions = useMemo<TabActionsValue>(() => ({
    open(kind, tabState, opts) {
      const id = idForState(kind, tabState);
      const incoming: Omit<Tab, 'createdAt' | 'lastActiveAt'> = {
        id,
        kind,
        title: opts?.title ?? DEFAULT_TITLE[kind],
        closable: kind !== 'hosts',
        dirty: false,
        state: tabState,
      };
      dispatch({ type: 'OPEN', tab: incoming });
      // The reducer dedupes; for callers that want the resolved id back,
      // we recompute from the rules — for singletons it's stable, for
      // non-singletons we just trust the freshly-generated id (which
      // wins when no dupe exists).
      return SINGLETON_KINDS.has(kind) ? `tab-singleton-${kind}` : id;
    },
    activate(id) { dispatch({ type: 'ACTIVATE', id }); },
    close(id) { dispatch({ type: 'CLOSE', id }); },
    closeAllClosable() { dispatch({ type: 'CLOSE_ALL_CLOSABLE' }); },
    rename(id, title) { dispatch({ type: 'RENAME', id, title }); },
    setDirty(id, dirty) { dispatch({ type: 'SET_DIRTY', id, dirty }); },
    replaceState(id, st) { dispatch({ type: 'REPLACE_STATE', id, state: st }); },
  }), [idForState]);

  const stateValue = useMemo<TabsContextValue>(() => ({ state }), [state]);

  return (
    <StateContext.Provider value={stateValue}>
      <ActionsContext.Provider value={actions}>
        {children}
      </ActionsContext.Provider>
    </StateContext.Provider>
  );
}

export function useTabs(): TabsContextValue {
  const ctx = useContext(StateContext);
  if (!ctx) throw new Error('useTabs must be used inside <TabsProvider>');
  return ctx;
}

export function useTabActions(): TabActionsValue {
  const ctx = useContext(ActionsContext);
  if (!ctx) throw new Error('useTabActions must be used inside <TabsProvider>');
  return ctx;
}

/** Convenience: read the active tab. Re-renders on activation change. */
export function useActiveTab(): Tab | undefined {
  const { state } = useTabs();
  return state.tabs.find((t) => t.id === state.activeId);
}

export { HOSTS_TAB_ID };
