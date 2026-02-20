import { useState, useCallback } from 'react';
import { useTerminalStore, createNewTab } from '../stores/terminalStore';
import { XTerminal } from './XTerminal';
import { SuggestionBanner } from './SuggestionBanner';

interface TerminalPanelProps {
  height: number;
}

export function TerminalPanel({ height }: TerminalPanelProps) {
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

  const handleAddTab = useCallback(() => {
    const tab = createNewTab();
    addTab(tab);
  }, [addTab]);

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

  return (
    <div
      className="bg-[#1a1b26] border-t border-gray-700 flex flex-col overflow-hidden"
      style={{ height: `${height}px` }}
    >
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
      <div className="flex items-center bg-[#13141f] border-b border-gray-800 px-1 min-h-[32px]">
        <div className="flex items-center gap-0.5 overflow-x-auto flex-1 scrollbar-none">
          {tabs.map((tab) => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={`group flex items-center gap-1.5 px-3 py-1.5 text-xs whitespace-nowrap rounded-t transition-colors ${
                activeTabId === tab.id
                  ? 'bg-[#1a1b26] text-white border-t border-x border-gray-700'
                  : 'text-gray-400 hover:text-gray-200 hover:bg-gray-800/50'
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
        />
      )}

      {/* Terminal instances - each tab gets its own xterm, hidden when not active */}
      <div className="flex-1 relative overflow-hidden">
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
    </div>
  );
}
