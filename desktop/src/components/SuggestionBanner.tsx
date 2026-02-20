import { useState } from 'react';
import { CommandSuggestion, useTerminalStore } from '../stores/terminalStore';
import { terminalAPI } from '../api/terminalAPI';

interface SuggestionBannerProps {
  suggestions: CommandSuggestion[];
  activeTabId: string;
}

export function SuggestionBanner({ suggestions, activeTabId }: SuggestionBannerProps) {
  const [currentIndex, setCurrentIndex] = useState(0);
  const { removeSuggestedCommand } = useTerminalStore();

  const suggestion = suggestions[Math.min(currentIndex, suggestions.length - 1)];
  if (!suggestion) return null;

  const handleRun = async () => {
    const cmd = suggestion.command;
    removeSuggestedCommand(suggestion.id);
    // Write the command text + Enter into the active PTY so the user sees it execute normally
    await terminalAPI.writePtySession(activeTabId, cmd + '\n');
  };

  const handleDismiss = () => {
    removeSuggestedCommand(suggestion.id);
    if (currentIndex >= suggestions.length - 1) {
      setCurrentIndex(Math.max(0, currentIndex - 1));
    }
  };

  const handleDismissAll = () => {
    useTerminalStore.getState().clearSuggestedCommands();
    setCurrentIndex(0);
  };

  return (
    <div className="bg-[#1e2030] border-b border-gray-700/50 px-3 py-1.5 flex items-center gap-3 text-xs">
      <div className="flex items-center gap-1.5 text-purple-400 flex-shrink-0">
        <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
        </svg>
        <span className="font-medium">{suggestion.agent_name}</span>
      </div>

      <code className="bg-black/40 px-2 py-0.5 rounded text-green-400 font-mono truncate flex-1 min-w-0">
        {suggestion.command}
      </code>

      {suggestion.description && (
        <span className="text-gray-400 truncate max-w-[200px] hidden lg:inline">
          {suggestion.description}
        </span>
      )}

      <div className="flex items-center gap-1 flex-shrink-0">
        {suggestions.length > 1 && (
          <span className="text-gray-500 mr-1">
            {Math.min(currentIndex, suggestions.length - 1) + 1}/{suggestions.length}
          </span>
        )}

        {suggestions.length > 1 && (
          <>
            <button
              onClick={() => setCurrentIndex(Math.max(0, currentIndex - 1))}
              disabled={currentIndex <= 0}
              className="px-1 py-0.5 text-gray-400 hover:text-white disabled:opacity-30 transition-colors"
            >
              <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
              </svg>
            </button>
            <button
              onClick={() => setCurrentIndex(Math.min(suggestions.length - 1, currentIndex + 1))}
              disabled={currentIndex >= suggestions.length - 1}
              className="px-1 py-0.5 text-gray-400 hover:text-white disabled:opacity-30 transition-colors"
            >
              <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
              </svg>
            </button>
          </>
        )}

        <button
          onClick={handleRun}
          className="px-2 py-0.5 bg-green-600 hover:bg-green-500 text-white rounded transition-colors font-medium"
        >
          Run
        </button>
        <button
          onClick={handleDismiss}
          className="px-2 py-0.5 bg-gray-700 hover:bg-gray-600 text-gray-300 rounded transition-colors"
        >
          Dismiss
        </button>
        {suggestions.length > 1 && (
          <button
            onClick={handleDismissAll}
            className="px-2 py-0.5 text-gray-500 hover:text-gray-300 transition-colors"
            title="Dismiss all"
          >
            <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        )}
      </div>
    </div>
  );
}
