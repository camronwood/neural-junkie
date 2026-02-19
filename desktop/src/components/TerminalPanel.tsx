import { useState, useRef, useEffect } from 'react';
import { useTerminalStore, TerminalCommand } from '../stores/terminalStore';
import { CommandSuggestionCard } from './CommandSuggestionCard';
import { terminalAPI } from '../api/terminalAPI';

interface TerminalPanelProps {
  height: number;
}

export function TerminalPanel({ height }: TerminalPanelProps) {
  const {
    suggestedCommands,
    commandHistory,
    executingCommand,
    clearSuggestedCommands,
    clearHistory,
    consoleOutput,
    currentWorkingDir,
    addConsoleCommand,
    addConsoleOutput,
    setCurrentWorkingDir,
    navigateHistory,
    panelHeight,
    setPanelHeight,
  } = useTerminalStore();

  const [activeTab, setActiveTab] = useState<'suggestions' | 'history' | 'console'>('suggestions');
  const [consoleInput, setConsoleInput] = useState('');
  const [isResizing, setIsResizing] = useState(false);
  const consoleOutputRef = useRef<HTMLDivElement>(null);
  const consoleInputRef = useRef<HTMLInputElement>(null);

  // Initialize shell session on mount
  useEffect(() => {
    const initShell = async () => {
      try {
        await terminalAPI.startShellSession();
        const cwd = await terminalAPI.getCurrentWorkingDir();
        setCurrentWorkingDir(cwd);
      } catch (error) {
        console.error('Failed to initialize shell session:', error);
      }
    };
    initShell();
  }, [setCurrentWorkingDir]);

  // Set up shell output listeners
  useEffect(() => {
    const setupListeners = async () => {
      try {
        await terminalAPI.onShellOutput((output) => {
          addConsoleOutput('stdout', output);
        });
        await terminalAPI.onShellError((error) => {
          addConsoleOutput('stderr', error);
        });
      } catch (error) {
        console.error('Failed to setup shell listeners:', error);
      }
    };
    setupListeners();
  }, [addConsoleOutput]);

  // Auto-scroll console output
  useEffect(() => {
    if (consoleOutputRef.current) {
      consoleOutputRef.current.scrollTop = consoleOutputRef.current.scrollHeight;
    }
  }, [consoleOutput]);

  // Handle console command execution
  const handleConsoleCommand = async (command: string) => {
    if (!command.trim()) return;

    addConsoleCommand(command);
    addConsoleOutput('command', `$ ${command}`);
    setConsoleInput('');

    try {
      await terminalAPI.executeInSession(command);
      // Update working directory after command
      const cwd = await terminalAPI.getCurrentWorkingDir();
      setCurrentWorkingDir(cwd);
    } catch (error) {
      addConsoleOutput('stderr', `Error: ${error}`);
    }
  };

  // Handle console input key events
  const handleConsoleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      handleConsoleCommand(consoleInput);
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      const historyCommand = navigateHistory('up');
      if (historyCommand) {
        setConsoleInput(historyCommand);
      }
    } else if (e.key === 'ArrowDown') {
      e.preventDefault();
      const historyCommand = navigateHistory('down');
      if (historyCommand) {
        setConsoleInput(historyCommand);
      } else {
        setConsoleInput('');
      }
    }
  };

  // Handle resize functionality
  const handleMouseDown = (e: React.MouseEvent) => {
    setIsResizing(true);
    const startY = e.clientY;
    const startHeight = panelHeight;

    const handleMouseMove = (moveEvent: MouseEvent) => {
      const delta = startY - moveEvent.clientY;
      const newHeight = Math.min(Math.max(startHeight + delta, 150), window.innerHeight * 0.8);
      setPanelHeight(newHeight);
    };

    const handleMouseUp = () => {
      setIsResizing(false);
      document.removeEventListener('mousemove', handleMouseMove);
      document.removeEventListener('mouseup', handleMouseUp);
    };

    document.addEventListener('mousemove', handleMouseMove);
    document.addEventListener('mouseup', handleMouseUp);
  };

  // Format duration for display
  const formatDuration = (ms: number): string => {
    if (ms < 1000) {
      return `${ms}ms`;
    }
    return `${(ms / 1000).toFixed(2)}s`;
  };

  // Format timestamp for display
  const formatTimestamp = (timestamp: string): string => {
    const date = new Date(timestamp);
    return date.toLocaleTimeString('en-US', {
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    });
  };

  // Command history item component
  const CommandHistoryItem = ({ command }: { command: TerminalCommand }) => {
    const [isExpanded, setIsExpanded] = useState(false);
    const hasOutput = command.stdout.length > 0 || command.stderr.length > 0;

    return (
      <div className="bg-gray-800 border border-gray-700 rounded-lg p-3 mb-2">
        {/* Header */}
        <div
          className="flex items-center justify-between cursor-pointer"
          onClick={() => setIsExpanded(!isExpanded)}
        >
          <div className="flex items-center gap-3">
            <span className={`text-lg ${command.status === 'completed' ? 'text-green-400' : 'text-red-400'}`}>
              {command.status === 'completed' ? '✅' : 
               command.status === 'failed' ? '❌' : 
               command.status === 'executing' ? '⏳' : '⏸️'}
            </span>
            <div>
              <div className="font-mono text-sm text-white">
                $ {command.command}
              </div>
              <div className="text-xs text-gray-400">
                {formatTimestamp(command.started_at)} • {formatDuration(command.duration_ms)}
                {command.exit_code !== 0 && ` • Exit code: ${command.exit_code}`}
              </div>
            </div>
          </div>
          {hasOutput && (
            <button
              className="text-gray-400 hover:text-white transition-colors"
              onClick={(e) => {
                e.stopPropagation();
                setIsExpanded(!isExpanded);
              }}
            >
              <svg
                className={`w-4 h-4 transition-transform ${isExpanded ? 'rotate-180' : ''}`}
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M19 9l-7 7-7-7"
                />
              </svg>
            </button>
          )}
        </div>

        {/* Output */}
        {isExpanded && hasOutput && (
          <div className="mt-3 border-t border-gray-700 pt-3">
            {command.stdout && (
              <div className="mb-2">
                <div className="text-xs font-semibold text-gray-400 uppercase mb-1">
                  Output
                </div>
                <pre className="bg-black rounded p-2 text-xs font-mono text-green-400 whitespace-pre-wrap overflow-x-auto">
                  {command.stdout}
                </pre>
              </div>
            )}
            {command.stderr && (
              <div>
                <div className="text-xs font-semibold text-red-400 uppercase mb-1">
                  Errors
                </div>
                <pre className="bg-black rounded p-2 text-xs font-mono text-red-400 whitespace-pre-wrap overflow-x-auto">
                  {command.stderr}
                </pre>
              </div>
            )}
          </div>
        )}
      </div>
    );
  };

  return (
    <div 
      className="bg-gray-900 border-t border-gray-700 flex flex-col transition-all duration-300 ease-in-out overflow-hidden"
      style={{ height: `${height}px` }}
    >
      {/* Resize Handle */}
      <div
        className={`h-2 bg-gray-700 hover:bg-gray-600 cursor-ns-resize flex items-center justify-center transition-colors ${
          isResizing ? 'bg-blue-500' : ''
        }`}
        onMouseDown={handleMouseDown}
      >
        <div className="w-8 h-0.5 bg-gray-400 rounded"></div>
      </div>

      {/* Header */}
      <div className="flex items-center justify-between p-3 border-b border-gray-700">
        <div className="flex items-center gap-4">
          <h3 className="text-white font-semibold">Terminal</h3>
          <div className="flex gap-1">
            <button
              onClick={() => setActiveTab('suggestions')}
              className={`px-3 py-1 text-xs rounded transition-colors ${
                activeTab === 'suggestions'
                  ? 'bg-blue-600 text-white'
                  : 'bg-gray-700 text-gray-300 hover:bg-gray-600'
              }`}
            >
              Suggestions ({suggestedCommands.length})
            </button>
            <button
              onClick={() => setActiveTab('history')}
              className={`px-3 py-1 text-xs rounded transition-colors ${
                activeTab === 'history'
                  ? 'bg-blue-600 text-white'
                  : 'bg-gray-700 text-gray-300 hover:bg-gray-600'
              }`}
            >
              History ({commandHistory.length})
            </button>
            <button
              onClick={() => setActiveTab('console')}
              className={`px-3 py-1 text-xs rounded transition-colors ${
                activeTab === 'console'
                  ? 'bg-blue-600 text-white'
                  : 'bg-gray-700 text-gray-300 hover:bg-gray-600'
              }`}
            >
              Console
            </button>
          </div>
        </div>
        
        <div className="flex items-center gap-2">
          <button
            onClick={() => {
              if (activeTab === 'suggestions') {
                clearSuggestedCommands();
              } else {
                clearHistory();
              }
            }}
            className="px-2 py-1 text-xs bg-gray-700 hover:bg-gray-600 text-gray-300 rounded transition-colors"
          >
            Clear
          </button>
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto p-3">
        {activeTab === 'suggestions' ? (
          <div>
            {suggestedCommands.length === 0 ? (
              <div className="text-center text-gray-400 py-8">
                <div className="text-4xl mb-2">⚡</div>
                <div>No command suggestions yet</div>
                <div className="text-sm mt-1">
                  Agents will suggest commands here when they detect them in their responses
                </div>
              </div>
            ) : (
              suggestedCommands.map((suggestion: any) => (
                <CommandSuggestionCard
                  key={suggestion.id}
                  suggestion={suggestion}
                />
              ))
            )}
          </div>
        ) : activeTab === 'history' ? (
          <div>
            {commandHistory.length === 0 ? (
              <div className="text-center text-gray-400 py-8">
                <div className="text-4xl mb-2">📜</div>
                <div>No command history yet</div>
                <div className="text-sm mt-1">
                  Executed commands will appear here
                </div>
              </div>
            ) : (
              commandHistory.map((command: any) => (
                <CommandHistoryItem key={command.id} command={command} />
              ))
            )}
          </div>
        ) : (
          <div className="flex flex-col h-full">
            {/* Console Output */}
            <div 
              ref={consoleOutputRef}
              className="flex-1 overflow-y-auto font-mono text-sm bg-black rounded p-3 mb-3"
            >
              {consoleOutput.length === 0 ? (
                <div className="text-gray-500">
                  Interactive terminal ready. Type commands below.
                </div>
              ) : (
                consoleOutput.map((output: any, index: number) => (
                  <div key={index} className="mb-1">
                    {output.type === 'command' && (
                      <div className="text-green-400">{output.content}</div>
                    )}
                    {output.type === 'stdout' && (
                      <div className="text-white whitespace-pre-wrap">{output.content}</div>
                    )}
                    {output.type === 'stderr' && (
                      <div className="text-red-400 whitespace-pre-wrap">{output.content}</div>
                    )}
                    {output.type === 'system' && (
                      <div className="text-yellow-400">{output.content}</div>
                    )}
                  </div>
                ))
              )}
            </div>
            
            {/* Console Input */}
            <div className="border-t border-gray-700 p-2 flex items-center gap-2">
              <span className="text-green-400 font-mono text-sm">
                {currentWorkingDir} $
              </span>
              <input
                ref={consoleInputRef}
                type="text"
                value={consoleInput}
                onChange={(e) => setConsoleInput(e.target.value)}
                onKeyDown={handleConsoleKeyDown}
                className="flex-1 bg-transparent text-white font-mono text-sm outline-none"
                placeholder="Type a command..."
                autoFocus
              />
            </div>
          </div>
        )}
      </div>

      {/* Status bar */}
      <div className="px-3 py-2 bg-gray-800 border-t border-gray-700 text-xs text-gray-400">
        {executingCommand ? (
          <div className="flex items-center gap-2">
            <span className="text-yellow-400">⏳</span>
            <span>Executing: {executingCommand.command}</span>
          </div>
        ) : (
          <div className="flex items-center justify-between">
            <span>Ready</span>
            <span>{suggestedCommands.length} pending • {commandHistory.length} executed</span>
          </div>
        )}
      </div>
    </div>
  );
}
