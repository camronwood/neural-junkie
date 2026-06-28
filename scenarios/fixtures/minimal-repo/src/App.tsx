import React, { useState } from 'react';
import './theme.css';

// Component for styled button
const AccentButton: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  return (
    <button 
      className="btn-primary" 
      style={{ padding: '10px 20px', cursor: 'pointer', transition: 'all 0.3s ease' }}
    >
      {children}
    </button>
  );
};

// Theme card component
const ThemeCard: React.FC<{ title: string; content: React.ReactNode }> = ({ title, children }) => (
  <div 
    className="component" 
    style={{ padding: '20px', borderRadius: '8px', transition: 'all 0.3s ease' }}
  >
    <h3>{title}</h3>
    {children}
  </div>
);

const App: React.FC = () => {
  const [isDark, setIsDark] = useState(false);

  const toggleTheme = () => {
    setIsDark(!isDark);
    document.documentElement.setAttribute('data-theme', isDark ? '' : 'dark');
  };

  return (
    <div style={{ minHeight: '100vh' }}>
      <nav>
        <button onClick={toggleTheme}>
          Toggle Theme ({isDark ? 'Dark Mode' : 'Light Mode'})
        </button>
      </nav>

      <main>
        <h1>Theme Demo</h1>
        <p style={{ color: 'var(--text-primary)', backgroundColor: 'var(--bg-secondary)' }}>
          This text will change background based on the theme.
        </p>

        <ThemeCard title="Component using CSS variables">
          <p>Background: var(--bg-primary)</p>
          <p>Text: var(--text-primary)</p>
          <p>Border: var(--border-color)</p>
          <AccentButton>Accent Button - var(--accent-color)</AccentButton>
        </ThemeCard>

        <ThemeCard title="Additional Test Area">
          <p data-theme="light">Light mode test text</p>
          <p data-theme="dark">Dark mode test text</p>
        </ThemeCard>
      </main>

      <footer style={{ 
        padding: '16px', 
        backgroundColor: 'var(--bg-secondary)',
        marginTop: '32px' 
      }}>
        <p style={{ color: 'var(--text-secondary)' }}>© 2025 Theme Demo</p>
      </footer>
    </div>
  );
};

export default App;