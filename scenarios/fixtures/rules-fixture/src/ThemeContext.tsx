import React, { createContext, useContext, useState, useEffect } from 'react';
import type { ThemeContextValue } from '../theme';
import { darkTheme, lightTheme } from '../theme';

export const ThemeContext = createContext<ThemeContextValue>({ 
  theme: 'dark', 
  toggleTheme: () => {} 
});

interface ThemeProviderProps {
  children: React.ReactNode;
}

export const ThemeProvider: React.FC<ThemeProviderProps> = ({ children }) => {
  const [theme, setTheme] = useState<'light' | 'dark'>('dark');

  useEffect(() => {
    // Sync with HTML class on mount and theme change
    const updateClass = (t: 'light' | 'dark') => {
      document.documentElement.classList.toggle('dark', t === 'dark');
      document.documentElement.classList.toggle('light', t === 'light');
    };

    updateClass(theme);

    // Listen for system preference changes
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    const handleChange = () => {
      setTheme(mediaQuery.matches ? 'dark' : 'light');
    };

    mediaQuery.addEventListener('change', handleChange);
    return () => mediaQuery.removeEventListener('change', handleChange);
  }, []);

  const toggleTheme = () => {
    setTheme((prev) => (prev === 'light' ? 'dark' : 'light'));
  };

  // Apply background/text colors to HTML for CSS var usage
  useEffect(() => {
    document.documentElement.style.setProperty('--bg-color', theme === 'dark' ? darkTheme.background : lightTheme.background);
    document.documentElement.style.setProperty('--text-color', theme === 'dark' ? darkTheme.text : lightTheme.text);
  }, [theme]);

  return (
    <ThemeContext.Provider value={{ theme, toggleTheme }}>
      {children}
    </ThemeContext.Provider>
  );
};

export const useTheme = () => useContext(ThemeContext);

// Re-export colors for convenience
export { darkTheme, lightTheme };