import { FC } from "react";

interface SidebarProps {
  toggleTheme: () => void;
  theme: string;
}

export default function Sidebar({ toggleTheme, theme }: SidebarProps) {
  return (
    <aside className="w-64 border border-slate-700 rounded-lg p-4 dark:border-slate-300 flex-shrink-0">
      <div className="space-y-1">
        <button
          onClick={toggleTheme}
          className="w-full px-4 py-2.5 bg-gradient-to-r from-indigo-600 to-purple-600 hover:from-indigo-700 hover:to-purple-700 text-white font-medium rounded-lg shadow-sm transition-all duration-200 flex items-center justify-between group"
        >
          <span className="flex items-center gap-2">
            <svg
              xmlns="http://www.w3.org/2000/svg"
              className="h-5 w-5 text-white/80 group-hover:text-white transition-colors"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M13 10V3L4 14h7v7l9-11h-7z"
              />
            </svg>
            Toggle Theme
          </span>
        </button>

        <div className="mt-8">
          <p className="text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider mb-3 px-2">
            Navigation
          </p>
          
          <nav className="space-y-1">
            {["Dashboard", "Projects", "Settings"].map((item, index) => (
              <button
                key={item}
                className="w-full text-left px-4 py-2.5 text-slate-300 dark:text-slate-600 hover:bg-slate-800 dark:hover:bg-slate-100 hover:text-white dark:hover:text-slate-900 rounded-lg transition-all duration-200 font-medium"
              >
                {item}
              </button>
            ))}
          </nav>
        </div>

        <div className="mt-auto pt-6 border-t border-slate-700 dark:border-slate-300">
          <p className="text-xs text-slate-500 dark:text-slate-400">
            v1.0.0 &bull; React Tailwind Minimal
          </p>
        </div>
      </div>
    </aside>
  );
}