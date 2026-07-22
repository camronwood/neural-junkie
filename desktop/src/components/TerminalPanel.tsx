import { useState, useCallback, useEffect } from 'react';
import { useTerminalStore, createNewTab } from '../stores/terminalStore';
import { XTerminal } from './XTerminal';
import { SuggestionBanner } from './SuggestionBanner';
import { terminalAPI } from '../api/terminalAPI';
import { resolveTerminalCwd } from '../utils/terminalCwd';
import { shellQuote } from '../utils/runTerminalCommand';
import type { Collaboration } from '../types/protocol';
import { registerRestartBlocker } from '../utils/restartSafety';

interface TerminalPanelProps {
  channel: string;
  api: import('../api/chatAPI').ChatAPI;
  collaboration?: Collaboration | null;
}

export function TerminalPanel({ channel, api, collaboration }: TerminalPanelProps) {
  const {
    tabs,
    activeTabId,
    addTab,
    removeTab,
    setActiveTab,
    panelHeight,
    setPanelHeight,
    suggestedCommands,
  } = useTerminalStore();

  const [isResizing, setIsResizing] = useState(false);

  useEffect(
    () =>
      registerRestartBlocker('terminal-foreground-work', () => {
        const active = useTerminalStore.getState().foregroundSessionIds;
        if (active.size === 0) return null;
        return {
          id: 'terminal-foreground-work',
          message: `${active.size} terminal session${active.size === 1 ? ' has' : 's have'} foreground work running.`,
        };
      }),
    []
  );

  const handleAddTab = useCallback(() => {
    const cwd = resolveTerminalCwd({ collaboration: collaboration ?? null });
    const tab = createNewTab('user', undefined, cwd);
    addTab(tab);
  }, [addTab, collaboration]);

  useEffect(() => {
    const cwd = resolveTerminalCwd({ collaboration: collaboration ?? null });
    const store = useTerminalStore.getState();
    store.alignActiveTabCwd(cwd);
    if (cwd && cwd !== '~') {
      void terminalAPI.writePtySession(store.activeTabId, `cd ${shellQuote(cwd)}\n`);
    }
  }, [collaboration?.id, collaboration?.source_repo_path, collaboration?.working_directory, collaboration?.phase]);

  const handleCloseTab = useCallback(
    (e: React.MouseEvent, tabId: string) => {
      e.stopPropagation();
      removeTab(tabId);
    },
    [removeTab]
  );

  const handleMouseDown = (e: React.MouseEvent) => {
    setIsResizing(true);
    const startY = e.clientY;
    const startHeight = panelHeight;

    const handleMouseMove = (moveEvent: MouseEvent) => {
      const delta = startY - moveEvent.clientY;
      // Leave room for the top toolbar so the command-helper footer stays visible.
      const maxHeight = Math.max(150, window.innerHeight - 48);
      const newHeight = Math.min(Math.max(startHeight + delta, 150), Math.min(window.innerHeight * 0.8, maxHeight));
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

  return (
    <div className="h-full min-h-0 bg-slack-bg border-t border-slack-border flex flex-col overflow-hidden">
      {/* Resize handle */}
      <div
        className={`h-1.5 cursor-ns-resize flex items-center justify-center transition-colors ${
          isResizing ? 'bg-blue-500' : 'bg-gray-700 hover:bg-gray-600'
        }`}
        onMouseDown={handleMouseDown}
      >
        <div className="w-8 h-0.5 bg-gray-500 rounded" />
      </div>

      {/* Tab bar */}
      <div className="flex items-center bg-slack-bgHover border-b border-slack-border px-1 min-h-[32px]">
        <div className="flex items-center gap-0.5 overflow-x-auto flex-1 scrollbar-none">
          {tabs.map((tab) => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={`group flex items-center gap-1.5 px-3 py-1.5 text-xs whitespace-nowrap rounded-t transition-colors ${
                activeTabId === tab.id
                  ? 'bg-slack-bg text-slack-text border-t border-x border-slack-border'
                  : 'text-slack-textMuted hover:text-slack-text hover:bg-slack-bgHover'
              }`}
            >
              {tab.type === 'agent' ? (
                <svg className="w-3 h-3 text-purple-400 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
                </svg>
              ) : (
                <svg className="w-3 h-3 text-green-400 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
                </svg>
              )}
              <span>{tab.label}</span>
              <span
                onClick={(e) => handleCloseTab(e, tab.id)}
                className="ml-1 opacity-0 group-hover:opacity-100 hover:bg-gray-600 rounded p-0.5 transition-opacity"
              >
                <svg className="w-2.5 h-2.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </span>
            </button>
          ))}
        </div>

        {/* Add tab button */}
        <button
          onClick={handleAddTab}
          className="flex items-center justify-center w-6 h-6 text-gray-400 hover:text-white hover:bg-gray-700 rounded transition-colors ml-1 flex-shrink-0"
          title="New Terminal"
        >
          <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
          </svg>
        </button>
      </div>

      {/* Suggestion banner */}
      {suggestedCommands.length > 0 && (
        <SuggestionBanner
          suggestions={suggestedCommands}
          activeTabId={activeTabId}
          channel={channel}
          api={api}
          collaboration={collaboration}
        />
      )}

      {/* Terminal instances - each tab gets its own xterm, hidden when not active */}
      <div className="flex-1 min-h-0 relative overflow-hidden">
        {tabs.map((tab) => (
          <div
            key={tab.id}
            className="absolute inset-0"
            style={{ visibility: activeTabId === tab.id ? 'visible' : 'hidden' }}
          >
            <XTerminal
              sessionId={tab.id}
              cwd={tab.cwd}
              isActive={activeTabId === tab.id}
            />
          </div>
        ))}
      </div>
      <div className="shrink-0 px-2 py-1 border-t border-slack-border/50 text-[10px] leading-tight text-slack-textMuted">
        Ctrl+C interrupt · Ctrl+L / {navigator.platform.toUpperCase().includes('MAC') ? 'Cmd' : 'Mod'}+K
        clear · links open in browser
      </div>
    </div>
  );
}
