import "./index.css";
import { useState, useEffect } from "react";

export default function App() {
  const [theme, setTheme] = useState("dark");

  useEffect(() => {
    document.documentElement.classList.toggle("dark", theme === "dark");
  }, [theme]);

  const toggleTheme = () => {
    setTheme(theme === "dark" ? "light" : "dark");
  };

  return (
    <div className="min-h-screen bg-slate-900 text-slate-100 p-4 dark:bg-slate-100 dark:text-slate-900">
      <aside className="w-48 border border-slate-700 rounded p-3 dark:border-slate-300">
        <p className="text-sm text-slate-400 dark:text-slate-600">Sidebar</p>
        <button
          onClick={toggleTheme}
          className="mt-2 px-2 py-1 bg-slate-700 text-slate-100 rounded dark:bg-slate-300 dark:text-slate-900"
        >
          Toggle Theme
        </button>
      </aside>
    </div>
  );
}