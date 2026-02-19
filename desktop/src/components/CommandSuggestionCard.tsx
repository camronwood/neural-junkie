import { useState } from 'react';
import { CommandSuggestion } from '../stores/terminalStore';
import { terminalAPI } from '../api/terminalAPI';
import { useTerminalStore } from '../stores/terminalStore';

interface CommandSuggestionCardProps {
  suggestion: CommandSuggestion;
}

export function CommandSuggestionCard({ suggestion }: CommandSuggestionCardProps) {
  const [isExecuting, setIsExecuting] = useState(false);
  const { removeSuggestedCommand, addCommand, setExecutingCommand } = useTerminalStore();

  const handleApprove = async () => {
    setIsExecuting(true);
    
    // Create a terminal command record
    const terminalCommand = {
      id: suggestion.id,
      command: suggestion.command,
      status: 'executing' as const,
      exit_code: 0,
      stdout: '',
      stderr: '',
      duration_ms: 0,
      started_at: new Date().toISOString(),
    };

    // Add to history and set as executing
    addCommand(terminalCommand);
    setExecutingCommand(terminalCommand);

    try {
      // Execute the command
      const result = await terminalAPI.executeCommand(suggestion.command);
      
      // Update the command with results
      const updatedCommand = {
        ...terminalCommand,
        status: result.success ? 'completed' as const : 'failed' as const,
        exit_code: result.exit_code,
        stdout: result.stdout,
        stderr: result.stderr,
        duration_ms: result.duration_ms,
        ended_at: new Date().toISOString(),
      };

      // Update in store
      useTerminalStore.getState().updateCommand(suggestion.id, updatedCommand);
      setExecutingCommand(null);
      
    } catch (error) {
      // Handle execution error
      const errorCommand = {
        ...terminalCommand,
        status: 'failed' as const,
        exit_code: -1,
        stderr: error instanceof Error ? error.message : 'Unknown error',
        ended_at: new Date().toISOString(),
      };

      useTerminalStore.getState().updateCommand(suggestion.id, errorCommand);
      setExecutingCommand(null);
    } finally {
      setIsExecuting(false);
      // Remove from suggestions
      removeSuggestedCommand(suggestion.id);
    }
  };

  const handleReject = () => {
    removeSuggestedCommand(suggestion.id);
  };

  return (
    <div className="bg-gray-800 border border-gray-700 rounded-lg p-4 mb-3">
      {/* Header */}
      <div className="flex items-center justify-between mb-2">
        <div className="flex items-center gap-2">
          <span className="text-green-400 text-sm font-mono">
            {suggestion.agent_name}
          </span>
          <span className="text-gray-400 text-xs">
            {suggestion.is_safe ? '🔒 Safe' : '⚠️ Requires approval'}
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
            {isExecuting ? 'Executing...' : 'Approve'}
          </button>
        </div>
      </div>

      {/* Description */}
      {suggestion.description && (
        <div className="text-gray-300 text-sm mb-2">
          {suggestion.description}
        </div>
      )}

      {/* Command */}
      <div className="bg-black rounded p-2 font-mono text-sm">
        <div className="text-green-400 mb-1">
          $ {suggestion.command}
        </div>
        {suggestion.plugin !== 'shell' && (
          <div className="text-blue-400 text-xs">
            Plugin: {suggestion.plugin}
          </div>
        )}
      </div>
    </div>
  );
}
