import "./index.css";
import { useState, useEffect } from "react";

// Use CSS variable for theme colors instead of dark: modifiers
const THEME_STORAGE_KEY = "neural-junkie-theme";

export default function App() {
  const [theme, setTheme] = useState(() => {
    // Initialize from localStorage or default to dark
    return (localStorage.getItem(THEME_STORAGE_KEY) as string) || "dark";
  });

  useEffect(() => {
    const isDark = theme === "dark";
    if (isDark) {
      document.documentElement.classList.add("dark");
    } else {
      document.documentElement.classList.remove("dark");
    }
    localStorage.setItem(THEME_STORAGE_KEY, theme);
  }, [theme]);

  const toggleTheme = () => {
    setTheme((prevTheme) => (prevTheme === "dark" ? "light" : "dark"));
  };

  return (
    <div className="min-h-screen dark:bg-slate-900 dark:text-slate-100 bg-white text-slate-900 p-4">
      <aside className="w-48 border border-slate-300 rounded p-3 dark:border-slate-700">
        <p className="text-sm text-slate-500 dark:text-slate-400">Sidebar</p>
        <button
          onClick={toggleTheme}
          className="mt-4 px-3 py-1 bg-indigo-600 text-white rounded hover:bg-indigo-700 transition-colors focus:outline-none focus:ring-2 focus:ring-indigo-500"
        >
          Toggle Theme
        </button>
      </aside>
    </div>
  );
}