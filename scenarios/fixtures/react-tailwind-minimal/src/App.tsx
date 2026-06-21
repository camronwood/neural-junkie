import "./index.css";
import { useState } from "react";

function renderSidebarFooter() {
  return (
    <footer className="text-xs text-slate-600 border-t border-slate-800 pt-3">
      <p>Neural Junkie fixture</p>
      <p>Selection extract target block</p>
      <p>Keep this helper scoped to sidebar only</p>
      <p>Do not move theme toggle logic here</p>
    </footer>
  );
}

export default function App() {
  const [theme, setTheme] = useState("dark");

  const toggleTheme = () => {
    setTheme((prevTheme) => (prevTheme === "dark" ? "light" : "dark"));
  };

  return (
    <div className="min-h-screen bg-slate-900 text-slate-100 p-4">
      <aside className="w-48 border border-slate-700 rounded p-3">
        <p className="text-sm text-slate-400">Sidebar</p>
        <button className="mt-4 px-3 py-1 bg-slate-800 rounded" onClick={toggleTheme}>
          Toggle Theme
        </button>
        {renderSidebarFooter()}
      </aside>
    </div>
  );
}
