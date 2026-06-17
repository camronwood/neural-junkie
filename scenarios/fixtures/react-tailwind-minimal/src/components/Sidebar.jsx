// Import necessary hooks, context and components
import React from 'react';
import { useTheme } from '../contexts/ThemeContext';

// Define the Sidebar component
const Sidebar = () => {
  // Use theme context instead of local state
  const { theme, toggleTheme } = useTheme();

  return (
    <div className={`sidebar ${theme === 'dark' ? 'dark' : ''}`}
      style={{ backgroundColor: theme === 'dark' ? '#121212' : '#ffffff', color: theme === 'dark' ? '#ffffff' : '#000000' }}
    >
      <button onClick={toggleTheme}>
        {theme === 'dark' ? 'Switch to Light Theme' : 'Switch to Dark Theme'}
      </button>
      {/* Sidebar content here */}
      <div className="mt-4">
        <p>Current theme: <strong>{theme === 'dark' ? 'Dark Mode' : 'Light Mode'}</strong></p>
        <div className="mt-2 text-sm opacity-75">
          Tip: Add more settings items here in the future!
        </div>
      </div>
    </div>
  );
};

export default Sidebar;

// Dark theme classes (handled by Tailwind via dark mode)
// .dark {
//   background-color: #121212 !important;
//   color: #ffffff !important;
// }