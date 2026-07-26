import React from 'react';

const MainContent = ({ theme }) => {
  // Minimal content area - ready for future expansion
  return (
    <div className="flex h-full flex-col space-y-6">
      {/* Centered minimal layout */}
      <div className="flex h-full flex-col items-center justify-center rounded-xl bg-neutral-100 dark:bg-neutral-900/50 p-8 transition-colors duration-300">
        <div className="text-center space-y-4">
          <h1 className="text-2xl font-light text-neutral-500 dark:text-neutral-400">
            Minimal Dashboard
          </h1>
          <p className="max-w-md text-sm text-neutral-400 dark:text-neutral-500">
            Settings will appear here in future iterations. Current implementation focuses on clean, minimal layout with theme support.
          </p>
        </div>

        {/* Feature placeholders */}
        <div className="mt-8 grid w-full max-w-lg grid-cols-3 gap-4">
          {/* Stats Card - Placeholder */}
          <div className="flex h-24 flex-col items-center justify-center rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900/50 shadow-sm transition-all hover:scale-105">
            <span className="text-3xl">📊</span>
            <span className="mt-2 text-xs font-medium text-neutral-400">Stats</span>
          </div>

          {/* Actions Card - Placeholder */}
          <div className="flex h-24 flex-col items-center justify-center rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900/50 shadow-sm transition-all hover:scale-105">
            <span className="text-3xl">⚡</span>
            <span className="mt-2 text-xs font-medium text-neutral-400">Actions</span>
          </div>

          {/* Settings Card - Placeholder */}
          <div className="flex h-24 flex-col items-center justify-center rounded-xl border border-neutral-200 dark:border-neutral-800 bg-white dark:bg-neutral-900/50 shadow-sm transition-all hover:scale-105">
            <span className="text-3xl">⚙️</span>
            <span className="mt-2 text-xs font-medium text-neutral-400">Settings</span>
          </div>
        </div>

        {/* Instructions for developers */}
        <div className="mt-8 w-full max-w-lg rounded-xl bg-gradient-to-br from-neutral-100 to-neutral-50 dark:from-neutral-900/30 dark:to-neutral-950/30 p-4 text-sm opacity-75">
          <p className="font-medium">Developer Notes:</p>
          <ul className="mt-2 space-y-1">
            <li>• Add your widgets/cards here</li>
            <li>• Connect to backend API endpoints</li>
            <li>• Implement real data visualization</li>
            <li>• Add charts, graphs, tables as needed</li>
          </ul>
        </div>
      </div>
    </div>
  );
};

export default MainContent;