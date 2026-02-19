import { useState, useEffect, useRef } from 'react';
import { Editor } from '@monaco-editor/react';
import { useEditorStore } from '../stores/editorStore';
import { useToastStore } from '../stores/toastStore';
import { useEditorShortcuts } from '../hooks/useEditorShortcuts';
import type { EditorTab } from '../stores/editorStore';

interface CodeEditorPanelProps {
  onClose: () => void;
}

const MIN_WIDTH = 300; // Minimum usable width
const DEFAULT_WIDTH = 600;
const STORAGE_KEY = 'code-editor-panel-width';

export function CodeEditorPanel({ onClose }: CodeEditorPanelProps) {
  const {
    tabs,
    activeTabId,
    saving,
    error,
    setActiveTab,
    updateTabContent,
    updateTabCursor,
    saveTab,
    saveAllTabs,
    closeTab,
    getActiveTab,
    hasUnsavedChanges,
  } = useEditorStore();
  
  const { addToast } = useToastStore();
  
  // Initialize keyboard shortcuts
  useEditorShortcuts();

  // Resize state
  const [width, setWidth] = useState<number>(() => {
    const saved = localStorage.getItem(STORAGE_KEY);
    const savedWidth = saved ? parseInt(saved, 10) : DEFAULT_WIDTH;
    // Sanity check: ensure saved width is reasonable (not larger than screen)
    const maxReasonableWidth = window.innerWidth * 0.7; // Max 70% of screen
    return savedWidth > maxReasonableWidth ? DEFAULT_WIDTH : savedWidth;
  });
  const [isResizing, setIsResizing] = useState(false);
  const resizeStartX = useRef<number>(0);
  const resizeStartWidth = useRef<number>(0);
  const currentWidthRef = useRef<number>(width);
  
  // Keep ref in sync with state
  useEffect(() => {
    currentWidthRef.current = width;
  }, [width]);

  // Editor state
  const [editor, setEditor] = useState<any>(null);

  // const [api] = useState(() => new ChatAPI('localhost:8080'));

  const activeTab = getActiveTab();

  // Resize handlers
  useEffect(() => {
    const handleMouseMove = (e: MouseEvent) => {
      if (!isResizing) return;
      
      const delta = e.clientX - resizeStartX.current;
      const newWidth = resizeStartWidth.current + delta;
      // Allow free resizing, but limit to reasonable maximum
      // Code editor should not take more than 60% of screen to leave room for chat and sidebar
      const maxWidth = Math.min(window.innerWidth * 0.6, 1200); // Max 60% of screen or 1200px, whichever is smaller
      const clampedWidth = Math.max(MIN_WIDTH, Math.min(maxWidth, newWidth));
      
      setWidth(clampedWidth);
    };

    const handleMouseUp = () => {
      if (isResizing) {
        setIsResizing(false);
        localStorage.setItem(STORAGE_KEY, currentWidthRef.current.toString());
      }
    };

    if (isResizing) {
      document.addEventListener('mousemove', handleMouseMove);
      document.addEventListener('mouseup', handleMouseUp);
      document.body.style.cursor = 'col-resize';
      document.body.style.userSelect = 'none';
    }

    return () => {
      document.removeEventListener('mousemove', handleMouseMove);
      document.removeEventListener('mouseup', handleMouseUp);
      document.body.style.cursor = '';
      document.body.style.userSelect = '';
    };
  }, [isResizing]);

  const handleResizeStart = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setIsResizing(true);
    resizeStartX.current = e.clientX;
    resizeStartWidth.current = currentWidthRef.current;
  };

  const handleEditorDidMount = (editor: any, monaco: any) => {
    setEditor(editor);
    
    // Configure editor options
    editor.updateOptions({
      minimap: { enabled: true },
      wordWrap: 'on',
      lineNumbers: 'on',
      folding: true,
      automaticLayout: true,
    });

    // Handle content changes
    editor.onDidChangeModelContent(() => {
      if (activeTab) {
        const content = editor.getValue();
        updateTabContent(activeTab.id, content);
      }
    });

    // Handle cursor position changes
    editor.onDidChangeCursorPosition((e: any) => {
      if (activeTab) {
        updateTabCursor(activeTab.id, {
          line: e.position.lineNumber,
          column: e.position.column,
        });
      }
    });

    // Handle keyboard shortcuts
    editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyS, () => {
      if (activeTab) {
        handleSave();
      }
    });
  };

  const handleSave = async () => {
    if (!activeTab || saving) return;
    
    const success = await saveTab(activeTab.id);
    if (success) {
      addToast({
        type: 'success',
        title: 'File saved',
        message: `${activeTab.path} has been saved successfully.`,
      });
    } else {
      addToast({
        type: 'error',
        title: 'Save failed',
        message: `Failed to save ${activeTab.path}. Please try again.`,
        action: {
          label: 'Retry',
          onClick: () => handleSave(),
        },
      });
    }
  };
  
  const handleSaveAll = async () => {
    const success = await saveAllTabs();
    if (success) {
      addToast({
        type: 'success',
        title: 'All files saved',
        message: 'All modified files have been saved successfully.',
      });
    } else {
      addToast({
        type: 'error',
        title: 'Save failed',
        message: 'Failed to save some files. Please check the errors and try again.',
      });
    }
  };

  const handleTabClick = (tabId: string) => {
    setActiveTab(tabId);
  };

  const handleTabClose = (e: React.MouseEvent, tabId: string) => {
    e.stopPropagation();
    closeTab(tabId);
  };

  const handleTabContextMenu = (e: React.MouseEvent) => {
    e.preventDefault();
    // TODO: Implement tab context menu
  };

  const getTabDisplayName = (tab: EditorTab) => {
    const fileName = tab.path.split('/').pop() || tab.path;
    return fileName;
  };

  const getTabIcon = (tab: EditorTab) => {
    const ext = tab.path.split('.').pop()?.toLowerCase();
    const iconMap: Record<string, string> = {
      'js': '📄',
      'jsx': '⚛️',
      'ts': '📘',
      'tsx': '⚛️',
      'py': '🐍',
      'go': '🐹',
      'rs': '🦀',
      'java': '☕',
      'html': '🌐',
      'css': '🎨',
      'json': '📋',
      'md': '📝',
      'txt': '📄',
      'yml': '⚙️',
      'yaml': '⚙️',
    };
    return iconMap[ext || ''] || '📄';
  };

  // Update editor content when active tab changes
  useEffect(() => {
    if (editor && activeTab) {
      const currentContent = editor.getValue();
      if (currentContent !== activeTab.content) {
        editor.setValue(activeTab.content);
      }
      
      // Restore cursor position
      if (activeTab.cursorPosition) {
        editor.setPosition({
          lineNumber: activeTab.cursorPosition.line,
          column: activeTab.cursorPosition.column,
        });
      }
    }
  }, [editor, activeTab]);

  return (
    <div 
      className="border-r border-slack-border bg-slack-bg flex flex-col h-full relative animate-slide-in-left flex-shrink-0"
      style={{ width: `${width}px`, minWidth: `${MIN_WIDTH}px` }}
    >
        {/* Resize Handle */}
        <div
          className="absolute right-0 top-0 bottom-0 cursor-col-resize z-[100] group"
          onMouseDown={handleResizeStart}
          aria-label="Resize code editor panel"
          style={{ 
            width: '6px', 
            marginRight: '-3px',
            pointerEvents: 'auto',
          }}
        >
          <div className="absolute inset-0 bg-transparent group-hover:bg-blue-500/30 transition-colors" />
          <div className="absolute right-1/2 top-1/2 -translate-y-1/2 translate-x-1/2 w-1 h-8 bg-gray-400 group-hover:bg-blue-500 rounded-full opacity-0 group-hover:opacity-100 transition-opacity" />
        </div>
      
      {/* Header */}
      <div className="px-4 py-3 border-b border-slack-border flex items-center justify-between bg-slack-bgHover">
        <h2 className="font-bold text-slack-text">Code Editor</h2>
        <div className="flex items-center gap-2">
          {hasUnsavedChanges() && (
            <span className="text-xs text-yellow-500">Unsaved changes</span>
          )}
          {saving && (
            <span className="text-xs text-blue-500 flex items-center gap-1">
              <div className="w-3 h-3 border border-blue-500 border-t-transparent rounded-full animate-spin"></div>
              Saving...
            </span>
          )}
          {error && (
            <span className="text-xs text-red-500" title={error}>
              Save failed
            </span>
          )}
          <button
            onClick={handleSave}
            disabled={saving || !activeTab}
            className="px-2 py-1 text-xs bg-green-600 hover:bg-green-700 disabled:bg-gray-600 disabled:cursor-not-allowed text-white rounded transition-colors"
            title="Save current file (Cmd+S)"
          >
            Save
          </button>
          <button
            onClick={handleSaveAll}
            disabled={saving || !hasUnsavedChanges()}
            className="px-2 py-1 text-xs bg-blue-600 hover:bg-blue-700 disabled:bg-gray-600 disabled:cursor-not-allowed text-white rounded transition-colors"
            title="Save all files (Cmd+Shift+S)"
          >
            Save All
          </button>
          <button
            onClick={onClose}
            className="text-slack-textMuted hover:text-slack-text transition-colors"
            title="Close code editor"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
      </div>

      {/* Tab Bar */}
      {tabs.length > 0 && (
        <div className="flex border-b border-slack-border bg-slack-bgHover overflow-x-auto">
          {tabs.map((tab) => (
            <div
              key={tab.id}
              className={`flex items-center gap-2 px-3 py-2 cursor-pointer border-r border-slack-border min-w-0 ${
                activeTabId === tab.id
                  ? 'bg-slack-bg text-slack-text'
                  : 'text-slack-textMuted hover:text-slack-text hover:bg-slack-bg'
              }`}
              onClick={() => handleTabClick(tab.id)}
              onContextMenu={handleTabContextMenu}
            >
              <span className="text-sm">{getTabIcon(tab)}</span>
              <span className="text-sm truncate max-w-32">{getTabDisplayName(tab)}</span>
              {tab.isDirty && (
                <span className="text-xs text-yellow-500">●</span>
              )}
              <button
                onClick={(e) => handleTabClose(e, tab.id)}
                className="text-slack-textMuted hover:text-slack-text transition-colors ml-1"
                title="Close tab"
              >
                <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>
          ))}
        </div>
      )}

      {/* Editor */}
      <div className="flex-1 min-h-0">
        {activeTab ? (
          <Editor
            height="100%"
            language={activeTab.language || 'plaintext'}
            value={activeTab.content}
            onMount={handleEditorDidMount}
            options={{
              theme: 'vs-dark',
              fontSize: 14,
              lineNumbers: 'on',
              minimap: { enabled: true },
              wordWrap: 'on',
              folding: true,
              automaticLayout: true,
              scrollBeyondLastLine: false,
              renderWhitespace: 'selection',
              cursorBlinking: 'blink',
              cursorSmoothCaretAnimation: 'on',
            }}
          />
        ) : (
          <div className="flex items-center justify-center h-full text-slack-textMuted">
            <div className="text-center">
              <div className="text-4xl mb-3">📝</div>
              <div className="text-lg font-medium mb-2">No file open</div>
              <div className="text-sm">Open a file from the file explorer to start editing</div>
            </div>
          </div>
        )}
      </div>

      {/* Status Bar */}
      {activeTab && (
        <div className="px-4 py-2 border-t border-slack-border bg-slack-bgHover text-xs text-slack-textMuted flex items-center justify-between">
          <div className="flex items-center gap-4">
            <span>{activeTab.path}</span>
            {activeTab.language && (
              <span className="px-2 py-1 bg-slack-bg rounded text-xs">
                {activeTab.language}
              </span>
            )}
          </div>
          <div className="flex items-center gap-2">
            {saving && (
              <span className="text-yellow-500">Saving...</span>
            )}
            <button
              onClick={handleSave}
              disabled={!activeTab.isDirty || saving}
              className="px-2 py-1 bg-slack-accent hover:bg-slack-accentHover text-white text-xs rounded transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              Save (⌘S)
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
