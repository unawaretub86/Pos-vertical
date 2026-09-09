'use client';

import { createContext, useContext, useEffect, useState, ReactNode } from 'react';

type Theme = 'dark' | 'light';

interface ThemeContextValue {
  theme: Theme;
  toggle: () => void;
}

const ThemeContext = createContext<ThemeContextValue>({ theme: 'dark', toggle: () => {} });

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [theme, setTheme] = useState<Theme>('dark');

  useEffect(() => {
    const saved = localStorage.getItem('pos_theme') as Theme | null;
    const initial = saved ?? 'dark';
    setTheme(initial);
    document.documentElement.setAttribute('data-theme', initial);
  }, []);

  function toggle() {
    setTheme(prev => {
      const next: Theme = prev === 'dark' ? 'light' : 'dark';
      localStorage.setItem('pos_theme', next);
      document.documentElement.setAttribute('data-theme', next);
      return next;
    });
  }

  return (
    <ThemeContext.Provider value={{ theme, toggle }}>
      {children}
    </ThemeContext.Provider>
  );
}

export function useTheme() {
  return useContext(ThemeContext);
}
