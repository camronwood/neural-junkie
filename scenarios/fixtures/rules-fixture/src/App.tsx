import "./index.css";
import { useState, useEffect } from "react";

export default function App() {
  const [theme, setTheme] = useState("dark");

  useEffect(() => {
    document.documentElement.classList.toggle("dark", theme === "dark");
  }, [theme]);

  const toggleTheme = () => {
    setTheme((prevTheme) => (prevTheme === "dark" ? "light" : "dark"));
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-900 to-slate-800 text-slate-100 p-4 dark:from-slate-950 dark:to-slate-900 transition-colors duration-300">
      {/* Sidebar */}
      <aside className="w-64 bg-gradient-to-br from-slate-950 to-slate-900 border border-slate-800 rounded-lg shadow-xl p-4 flex flex-col gap-4 transition-all duration-300 dark:from-slate-200 dark:to-slate-300 dark:border-slate-600">
        {/* Logo & Title */}
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 rounded-xl bg-gradient-to-br from-indigo-500 to-purple-600 flex items-center justify-center text-white font-bold text-lg shadow-inner">
            NJ
          </div>
          <p className="text-lg font-semibold bg-gradient-to-r from-indigo-400 to-purple-400 bg-clip-text text-transparent transition-colors duration-300 dark:from-indigo-600 dark:to-purple-600">
            Neural Junkie
          </p>
        </div>

        {/* Theme Toggle */}
        <button
          onClick={toggleTheme}
          aria-label={`Switch to ${theme === "dark" ? "light" : "dark"} mode`}
          className="flex items-center justify-center w-full px-4 py-3 bg-gradient-to-r from-slate-800 to-slate-900 hover:from-indigo-600 hover:to-purple-600 text-slate-100 rounded-xl transition-all duration-200 gap-3 shadow-lg border border-slate-700/50 hover:border-indigo-400 dark:from-slate-300 dark:to-slate-400 dark:text-slate-800 dark:hover:from-purple-600 dark:hover:to-pink-600 dark:hover:border-indigo-400/50"
        >
          {theme === "dark" ? (
            <>
              <span className="text-xl">☀</span>
              <span className="font-semibold text-sm tracking-wide">Light Mode</span>
            </>
          ) : (
            <>
              <span className="text-xl">🌙</span>
              <span className="font-semibold text-sm tracking-wide">Dark Mode</span>
            </>
          )}
        </button>

        {/* Quick Actions */}
        <div className="flex flex-col gap-2 flex-1 overflow-hidden">
          <h3 className="text-xs font-bold uppercase tracking-widest text-slate-500 pl-2 transition-colors duration-300 dark:text-slate-400/60">Quick Actions</h3>

          {/* Action Buttons */}
          <div className="flex flex-col gap-2 overflow-y-auto flex-1 custom-scrollbar">
            {["New Project", "Load Model", "Clear Cache", "About"].map((label) => (
              <button
                key={label}
                onClick={() => alert(label)}
                className="group px-4 py-3 bg-gradient-to-r from-slate-800/50 to-slate-900/50 hover:from-indigo-600/20 hover:to-purple-600/20 text-left rounded-lg transition-all duration-200 border border-transparent hover:border-indigo-400/30 text-sm font-medium tracking-tight text-slate-300 group-hover:text-white dark:bg-slate-300/30 dark:text-slate-700 dark:group-hover:text-slate-900 dark:hover:border-indigo-500/30"
              >
                <span className="flex items-center gap-2">
                  <span className="inline-block w-1.5 h-1.5 rounded-full bg-slate-600 group-hover:bg-indigo-400 transition-colors duration-200 dark:bg-slate-600/50"></span>
                  {label}
                </span>
              </button>
            ))}
          </div>
        </div>

        {/* Footer */}
        <footer className="text-xs text-center py-3 transition-colors duration-300 border-t border-slate-800 dark:border-slate-700">
          <span className="text-slate-500 dark:text-slate-400/60">v1.0.0</span>
          <span className="mx-2 text-slate-600 dark:text-slate-300">•</span>
          <span className="text-slate-400 dark:text-indigo-500/70 font-medium">Neural Junkie</span>
        </footer>
      </aside>
    </div>
  );
}