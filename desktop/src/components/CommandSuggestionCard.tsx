import { useState } from 'react';
import { CommandSuggestion } from '../stores/terminalStore';
import { terminalAPI } from '../api/terminalAPI';
import { useTerminalStore } from '../stores/terminalStore';

interface CommandSuggestionCardProps {
  suggestion: CommandSuggestion;
}

export function CommandSuggestionCard({ suggestion }: CommandSuggestionCardProps) {
  const [isExecuting, setIsExecuting] = useState(false);
  const { removeSuggestedCommand } = useTerminalStore();
  const activeTabId = useTerminalStore((s) => s.activeTabId);

  const handleApprove = async () => {
    setIsExecuting(true);
    try {
      await terminalAPI.writePtySession(activeTabId, suggestion.command + '\n');
    } catch {
      await terminalAPI.executeCommand(suggestion.command, suggestion.cwd);
    } finally {
      setIsExecuting(false);
      removeSuggestedCommand(suggestion.id);
    }
  };

  const handleReject = () => {
    removeSuggestedCommand(suggestion.id);
  };

  return (
    <div className="bg-gray-800 border border-gray-700 rounded-lg p-4 mb-3">
      <div className="flex items-center justify-between mb-2">
        <div className="flex items-center gap-2">
          <span className="text-green-400 text-sm font-mono">
            {suggestion.agent_name}
          </span>
          <span className="text-gray-400 text-xs">
            {suggestion.is_safe ? 'Safe' : 'Requires approval'}
          </span>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={handleReject}
            disabled={isExecuting}
            className="px-3 py-1 text-xs bg-red-600 hover:bg-red-700 disabled:bg-gray-600 text-white rounded transition-colors"
          >
            Reject
          </button>
          <button
            onClick={handleApprove}
            disabled={isExecuting}
            className="px-3 py-1 text-xs bg-green-600 hover:bg-green-700 disabled:bg-gray-600 text-white rounded transition-colors"
          >
            {isExecuting ? 'Running...' : 'Run'}
          </button>
        </div>
      </div>

      {suggestion.description && (
        <div className="text-gray-300 text-sm mb-2">
          {suggestion.description}
        </div>
      )}

      <div className="bg-black rounded p-2 font-mono text-sm">
        <div className="text-green-400">
          $ {suggestion.command}
        </div>
      </div>
    </div>
  );
}
