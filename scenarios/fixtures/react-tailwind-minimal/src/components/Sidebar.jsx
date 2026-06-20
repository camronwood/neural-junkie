import React from 'react';
import { useTheme } from '../contexts/ThemeContext';

const Sidebar = () => {
  const { theme, toggleTheme } = useTheme();

  return (
    <div className={`sidebar h-full flex flex-col p-4 transition-colors duration-300 ${theme === 'dark' ? 'dark' : ''}`}>
      {/* Theme Toggle */}
      <button
        onClick={toggleTheme}
        className="flex items-center justify-center gap-2 py-3 px-4 rounded-lg bg-neutral-100 dark:bg-neutral-800 hover:bg-neutral-200 dark:hover:bg-neutral-700 text-sm font-medium transition-all"
      >
        {theme === 'dark' ? (
          <>☀️ Switch to Light Mode</>
        ) : (
          <>🌙 Switch to Dark Mode</>
        )}
      </button>

      {/* Theme Status Indicator */}
      <div className="mt-4 p-3 rounded-lg bg-neutral-100 dark:bg-neutral-800 text-sm">
        <p className={theme === 'dark' ? 'text-white' : 'text-neutral-800'}>
          Current theme: <strong>{theme === 'dark' ? 'Dark Mode' : 'Light Mode'}</strong>
        </p>
      </div>

      {/* Future Settings Items */}
      <div className="mt-2 text-xs opacity-75">
        Tip: Add more settings items here in the future!
      </div>
    </div>
  );
};

export default Sidebar;