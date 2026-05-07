import { useEffect, useState } from 'react';

type Theme = 'light' | 'dark' | 'system';

function resolvedTheme(theme: Theme): 'light' | 'dark' {
  if (theme === 'system') {
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
  }
  return theme;
}

function applyTheme(theme: Theme): void {
  const resolved = resolvedTheme(theme);
  // Dark is the default (set on :root in tokens.css). Adding the .light class
  // on <html> swaps the palette to the light overrides.
  document.documentElement.classList.toggle('light', resolved === 'light');
  document.documentElement.classList.toggle('dark', resolved === 'dark');
}

export function useTheme() {
  const [theme, setThemeState] = useState<Theme>(() => {
    return (localStorage.getItem('sshthing-theme') as Theme | null) ?? 'dark';
  });

  useEffect(() => {
    applyTheme(theme);
  }, [theme]);

  // Listen for system preference changes when theme is 'system'.
  useEffect(() => {
    if (theme !== 'system') return;
    const mq = window.matchMedia('(prefers-color-scheme: dark)');
    const handler = () => applyTheme('system');
    mq.addEventListener('change', handler);
    return () => mq.removeEventListener('change', handler);
  }, [theme]);

  const setTheme = (t: Theme) => {
    localStorage.setItem('sshthing-theme', t);
    setThemeState(t);
  };

  return { theme, setTheme };
}
