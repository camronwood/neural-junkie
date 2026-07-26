import React, { createContext, useContext, useState, useEffect } from 'react';

type Theme = {
  isDark: boolean;
  accentColor: string;
  textColor: string;
  textPrimary: string;
  textSecondary: string;
  cardBackground: string;
  navBackground: string;
  mainBackground: string;
  borderColor: string;
  version: string;
};

type ThemeContextType = {
  theme: Theme;
  toggleTheme: () => void;
};

const initialTheme: Theme = {
  isDark: true,
  accentColor: '#5865F2',
  textColor: '#FFFFFF',
  textPrimary: '#FFFFFF',
  textSecondary: '#909296',
  cardBackground: '#323437',
  navBackground: '#1e1f22',
  mainBackground: '#323437',
  borderColor: '#35373b',
  version: '1.0.0'
};

const ThemeContext = createContext<ThemeContextType | undefined>(undefined);

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const [theme, setTheme] = useState<Theme>(() => {
    // Load from localStorage if available
    const savedTheme = localStorage.getItem('theme');
    return savedTheme ? JSON.parse(savedTheme) : initialTheme;
  });

  useEffect(() => {
    localStorage.setItem('theme', JSON.stringify(theme));
  }, [theme]);

  const toggleTheme = () => {
    setTheme(prev => ({
      ...prev,
      isDark: !prev.isDark,
      accentColor: prev.isDark ? '#5865F2' : '#0079D3',
      textColor: prev.isDark ? '#FFFFFF' : '#333333',
      textPrimary: prev.isDark ? '#FFFFFF' : '#1a1b26',
      textSecondary: prev.isDark ? '#909296' : '#888888',
      cardBackground: prev.isDark ? '#323437' : '#ffffff',
      navBackground: prev.isDark ? '#1e1f22' : '#f5f6fa',
      mainBackground: prev.isDark ? '#323437' : '#ffffff',
      borderColor: prev.isDark ? '#35373b' : '#dfe1e8'
    }));
  };

  return (
    <ThemeContext.Provider value={{ theme, toggleTheme }}>
      {children}
    </ThemeContext.Provider>
  );
}

export function useTheme() {
  const context = useContext(ThemeContext);
  if (context === undefined) {
    throw new Error('useTheme must be used within a ThemeProvider');
  }
  return context;
}