/**
 * TeamContext — globally shared active team state with localStorage persistence.
 *
 * Only stores activeTeamId + setter. The actual TeamSummary object is always
 * derived by consumers: teams.find(t => t.id === activeTeamId) ?? teams[0].
 * This avoids coupling context to the teams-list fetch.
 */
import { createContext, useCallback, useContext, useEffect, useState } from 'react';

const STORAGE_KEY = 'sshthing:activeTeamId';

interface TeamContextValue {
  activeTeamId: string | null;
  setActiveTeamId: (id: string | null) => void;
}

const TeamContext = createContext<TeamContextValue | null>(null);

export function TeamProvider({ children }: { children: React.ReactNode }) {
  const [activeTeamId, setActiveTeamIdState] = useState<string | null>(() => {
    try {
      return localStorage.getItem(STORAGE_KEY) ?? null;
    } catch {
      return null;
    }
  });

  const setActiveTeamId = useCallback((id: string | null) => {
    setActiveTeamIdState(id);
    try {
      if (id === null) {
        localStorage.removeItem(STORAGE_KEY);
      } else {
        localStorage.setItem(STORAGE_KEY, id);
      }
    } catch {
      // Ignore storage errors (e.g. in-private mode quota exceeded)
    }
  }, []);

  return (
    <TeamContext.Provider value={{ activeTeamId, setActiveTeamId }}>
      {children}
    </TeamContext.Provider>
  );
}

export function useTeamContext(): TeamContextValue {
  const ctx = useContext(TeamContext);
  if (!ctx) {
    throw new Error('useTeamContext must be used inside <TeamProvider>');
  }
  return ctx;
}
