import React, { createContext, useContext, useState } from 'react';

export const ThemeContext = createContext<{ theme: string; toggleTheme: () => void }>({ theme: 'light', toggleTheme: () => {} });

export const ThemeProvider: React.FC = ({ children }) => {
  const [theme, setTheme] = useState('light');

  const toggleTheme = () => {
    setTheme(theme === 'light' ? 'dark' : 'light');
  };

  return (
    <ThemeContext.Provider value={{ theme, toggleTheme }}>
      {children}
    </ThemeContext.Provider>
  );
};

export const useTheme = () => useContext(ThemeContext);