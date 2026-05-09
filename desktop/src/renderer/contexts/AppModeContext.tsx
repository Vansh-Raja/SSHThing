/**
 * AppModeContext — globally shared Personal / Teams mode with localStorage persistence.
 *
 * Mirrors the TUI's Shift+T toggle: the entire app switches between
 * Personal mode (local hosts) and Teams mode (cloud team hosts).
 */
import { createContext, useCallback, useContext, useEffect, useState } from 'react';

export type AppMode = 'personal' | 'teams';

const STORAGE_KEY = 'sshthing:appMode';

interface AppModeContextValue {
  mode: AppMode;
  toggleMode: () => void;
  setMode: (mode: AppMode) => void;
}

const AppModeContext = createContext<AppModeContextValue | null>(null);

function readStoredMode(): AppMode {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (raw === 'teams') return 'teams';
  } catch {
    // ignore storage errors
  }
  return 'personal';
}

export function AppModeProvider({ children }: { children: React.ReactNode }) {
  const [mode, setModeState] = useState<AppMode>(readStoredMode);

  useEffect(() => {
    try {
      localStorage.setItem(STORAGE_KEY, mode);
    } catch {
      // ignore storage errors
    }
  }, [mode]);

  const toggleMode = useCallback(() => {
    setModeState((prev) => (prev === 'personal' ? 'teams' : 'personal'));
  }, []);

  const setMode = useCallback((next: AppMode) => {
    setModeState(next);
  }, []);

  return (
    <AppModeContext.Provider value={{ mode, toggleMode, setMode }}>
      {children}
    </AppModeContext.Provider>
  );
}

export function useAppMode(): AppModeContextValue {
  const ctx = useContext(AppModeContext);
  if (!ctx) {
    throw new Error('useAppMode must be used inside <AppModeProvider>');
  }
  return ctx;
}
