/**
 * useExecHistory — persists the last 20 exec results in localStorage.
 * Each entry holds hostId, command, exit code, duration, timestamp, and
 * stdout/stderr truncated to 4 KB.
 */
import { useCallback, useEffect, useState } from 'react';

const STORAGE_KEY = 'sshthing.exec_history';
const MAX_ENTRIES = 20;
const MAX_OUTPUT_BYTES = 4096;

export interface ExecHistoryEntry {
  id: string;
  hostId: string;
  hostLabel: string;
  command: string;
  stdout: string;
  stderr: string;
  exitCode: number;
  durationMs: number;
  executedAt: number;
}

function truncate(s: string, maxBytes: number): string {
  if (s.length <= maxBytes) return s;
  return s.slice(0, maxBytes) + '\n… (truncated)';
}

function loadHistory(): ExecHistoryEntry[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return [];
    const parsed: unknown = JSON.parse(raw);
    if (!Array.isArray(parsed)) return [];
    return parsed as ExecHistoryEntry[];
  } catch {
    return [];
  }
}

function saveHistory(entries: ExecHistoryEntry[]): void {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(entries));
  } catch {
    // localStorage may be full — silently ignore
  }
}

export interface UseExecHistoryReturn {
  history: ExecHistoryEntry[];
  addEntry: (
    hostId: string,
    hostLabel: string,
    command: string,
    result: { stdout: string; stderr: string; exitCode: number; durationMs: number },
  ) => void;
  clearHistory: () => void;
}

export function useExecHistory(): UseExecHistoryReturn {
  const [history, setHistory] = useState<ExecHistoryEntry[]>(() => loadHistory());

  // Keep state and storage in sync when another tab writes.
  useEffect(() => {
    const handleStorage = (e: StorageEvent) => {
      if (e.key === STORAGE_KEY) {
        setHistory(loadHistory());
      }
    };
    window.addEventListener('storage', handleStorage);
    return () => window.removeEventListener('storage', handleStorage);
  }, []);

  const addEntry = useCallback((
    hostId: string,
    hostLabel: string,
    command: string,
    result: { stdout: string; stderr: string; exitCode: number; durationMs: number },
  ) => {
    const entry: ExecHistoryEntry = {
      id: `exec_${Date.now()}_${Math.random().toString(36).slice(2, 7)}`,
      hostId,
      hostLabel,
      command,
      stdout: truncate(result.stdout, MAX_OUTPUT_BYTES),
      stderr: truncate(result.stderr, MAX_OUTPUT_BYTES),
      exitCode: result.exitCode,
      durationMs: result.durationMs,
      executedAt: Date.now(),
    };
    setHistory((prev) => {
      const next = [entry, ...prev].slice(0, MAX_ENTRIES);
      saveHistory(next);
      return next;
    });
  }, []);

  const clearHistory = useCallback(() => {
    setHistory([]);
    saveHistory([]);
  }, []);

  return { history, addEntry, clearHistory };
}
