// Import necessary hooks and components
import React, { useState } from 'react';

// Define the Sidebar component
const Sidebar = () => {
  const [isDarkMode, setIsDarkMode] = useState(false);

  const toggleTheme = () => {
    setIsDarkMode(!isDarkMode);
    document.body.classList.toggle('dark-theme');
  };

  return (
    <div className={`sidebar ${isDarkMode ? 'dark' : ''}`}
      style={{ backgroundColor: isDarkMode ? '#121212' : '#ffffff', color: isDarkMode ? '#ffffff' : '#000000' }}
    >
      <button onClick={toggleTheme}>Toggle Theme</button>
      {/* Sidebar content here */}
    </div>
  );
};

export default Sidebar;

// Add dark theme styles in a global stylesheet or here if preferred
/*
.dark-theme {
  background-color: #121212 !important;
  color: #ffffff !important;
}
*/