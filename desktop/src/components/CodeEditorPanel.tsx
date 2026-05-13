import { useState, useEffect, useRef, useCallback } from 'react';
import { Editor } from '@monaco-editor/react';
import { useEditorStore } from '../stores/editorStore';
import { useToastStore } from '../stores/toastStore';
import { useEditorShortcuts } from '../hooks/useEditorShortcuts';
import type { EditorTab } from '../stores/editorStore';

interface CodeEditorPanelProps {
  onClose: () => void;
}

const MIN_WIDTH = 300;
const DEFAULT_WIDTH = 600;
const STORAGE_KEY = 'code-editor-panel-width';

export function CodeEditorPanel({ onClose }: CodeEditorPanelProps) {
  const {
    tabs,
    activeTabId,
    saving,
    error,
    setActiveTab,
    saveTab,
    saveAllTabs,
    closeTab,
    getActiveTab,
    hasUnsavedChanges,
  } = useEditorStore();

  const activeContentSyncKey = useEditorStore((s) => {
    const t = s.tabs.find((x) => x.id === s.activeTabId);
    return t?.contentSyncKey ?? 0;
  });

  const tabIdsKey = useEditorStore((s) =>
    [...s.tabs.map((t) => t.id)].sort().join(',')
  );

  const { addToast } = useToastStore();

  useEditorShortcuts();

  const [width, setWidth] = useState<number>(() => {
    const saved = localStorage.getItem(STORAGE_KEY);
    const savedWidth = saved ? parseInt(saved, 10) : DEFAULT_WIDTH;
    const maxReasonableWidth = window.innerWidth * 0.7;
    return savedWidth > maxReasonableWidth ? DEFAULT_WIDTH : savedWidth;
  });
  const [isResizing, setIsResizing] = useState(false);
  const resizeStartX = useRef<number>(0);
  const resizeStartWidth = useRef<number>(0);
  const currentWidthRef = useRef<number>(width);

  useEffect(() => {
    currentWidthRef.current = width;
  }, [width]);

  const [editor, setEditor] = useState<import('monaco-editor').editor.IStandaloneCodeEditor | null>(null);
  const monacoRef = useRef<typeof import('monaco-editor') | null>(null);
  const tabModelsRef = useRef<Map<string, import('monaco-editor').editor.ITextModel>>(new Map());
  const viewStatesRef = useRef<Map<string, import('monaco-editor').editor.ICodeEditorViewState | null>>(new Map());
  const lastAppliedRef = useRef<{ tabId: string | null; syncKey: number }>({
    tabId: null,
    syncKey: -1,
  });
  const editorListenersRef = useRef<Array<{ dispose(): void }>>([]);
  const editorRef = useRef(editor);
  editorRef.current = editor;

  const activeTab = getActiveTab();

  useEffect(() => {
    if (!activeTabId) {
      setEditor(null);
    }
  }, [activeTabId]);

  useEffect(() => {
    const state = useEditorStore.getState();
    const ids = new Set(state.tabs.map((t) => t.id));
    for (const [id, model] of [...tabModelsRef.current.entries()]) {
      if (!ids.has(id)) {
        model.dispose();
        tabModelsRef.current.delete(id);
        viewStatesRef.current.delete(id);
      }
    }
  }, [tabIdsKey]);

  useEffect(() => {
    return () => {
      for (const d of editorListenersRef.current) {
        d.dispose();
      }
      editorListenersRef.current = [];
      const ed = editorRef.current;
      if (ed) {
        try {
          ed.setModel(null);
        } catch {
          /* editor may already be disposed */
        }
      }
      for (const m of tabModelsRef.current.values()) {
        if (!m.isDisposed()) {
          m.dispose();
        }
      }
      tabModelsRef.current.clear();
      viewStatesRef.current.clear();
    };
  }, []);

  useEffect(() => {
    if (!editor || !monacoRef.current || !activeTabId) return;

    const monaco = monacoRef.current;
    const tab = useEditorStore.getState().getTabById(activeTabId);
    if (!tab) return;

    const syncKey = tab.contentSyncKey ?? 0;
    const tabSwitched = lastAppliedRef.current.tabId !== activeTabId;

    let model = tabModelsRef.current.get(activeTabId);
    if (!model || model.isDisposed()) {
      const uri = monaco.Uri.parse(
        `nj://${tab.workspaceId}/${encodeURIComponent(tab.path)}?tab=${encodeURIComponent(activeTabId)}`
      );
      model = monaco.editor.createModel(tab.content, tab.language || 'plaintext', uri);
      tabModelsRef.current.set(activeTabId, model);
    } else if (model.getValue() !== tab.content) {
      model.setValue(tab.content);
    }

    monaco.editor.setModelLanguage(model, tab.language || 'plaintext');

    if (tabSwitched) {
      const prev = lastAppliedRef.current.tabId;
      if (prev && prev !== activeTabId) {
        viewStatesRef.current.set(prev, editor.saveViewState());
      }
      const previousModel = editor.getModel();
      editor.setModel(model);
      if (
        previousModel &&
        previousModel !== model &&
        ![...tabModelsRef.current.values()].includes(previousModel)
      ) {
        previousModel.dispose();
      }
      const vs = viewStatesRef.current.get(activeTabId);
      if (vs) {
        editor.restoreViewState(vs);
      }
    }

    lastAppliedRef.current = { tabId: activeTabId, syncKey };
  }, [editor, activeTabId, activeContentSyncKey]);

  useEffect(() => {
    if (!isResizing) return;

    const handleMouseMove = (e: MouseEvent) => {
      const delta = e.clientX - resizeStartX.current;
      const newWidth = resizeStartWidth.current + delta;
      const maxWidth = Math.min(window.innerWidth * 0.6, 1200);
      const clampedWidth = Math.max(MIN_WIDTH, Math.min(maxWidth, newWidth));
      setWidth(clampedWidth);
    };

    const handleMouseUp = () => {
      setIsResizing(false);
      localStorage.setItem(STORAGE_KEY, currentWidthRef.current.toString());
    };

    document.addEventListener('mousemove', handleMouseMove);
    document.addEventListener('mouseup', handleMouseUp);
    document.body.style.cursor = 'col-resize';
    document.body.style.userSelect = 'none';

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

  const handleSave = useCallback(async () => {
    const tab = useEditorStore.getState().getActiveTab();
    if (!tab || useEditorStore.getState().saving) return;

    const success = await saveTab(tab.id);
    if (success) {
      addToast({
        type: 'success',
        title: 'File saved',
        message: `${tab.path} has been saved successfully.`,
      });
    } else {
      addToast({
        type: 'error',
        title: 'Save failed',
        message: `Failed to save ${tab.path}. Please try again.`,
        action: {
          label: 'Retry',
          onClick: () => void handleSave(),
        },
      });
    }
  }, [addToast, saveTab]);

  const handleEditorDidMount = (
    ed: import('monaco-editor').editor.IStandaloneCodeEditor,
    monaco: typeof import('monaco-editor')
  ) => {
    monacoRef.current = monaco;
    setEditor(ed);

    for (const d of editorListenersRef.current) {
      d.dispose();
    }
    editorListenersRef.current = [];

    ed.updateOptions({
      minimap: { enabled: true },
      wordWrap: 'off',
      lineNumbers: 'on',
      folding: true,
      automaticLayout: true,
      tabSize: 4,
      insertSpaces: true,
      detectIndentation: true,
      smoothScrolling: true,
      scrollBeyondLastLine: false,
      bracketPairColorization: { enabled: true },
      multiCursorModifier: 'ctrlCmd',
    });

    const subContent = ed.onDidChangeModelContent(() => {
      const { activeTabId: id, updateTabContent: upd, getTabById } = useEditorStore.getState();
      if (!id) return;
      const m = ed.getModel();
      if (!m) return;
      const next = m.getValue();
      const tabRow = getTabById(id);
      if (tabRow && tabRow.content === next) return;
      upd(id, next);
    });
    editorListenersRef.current.push(subContent);
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
  };

  const getTabDisplayName = (tab: EditorTab) => {
    const fileName = tab.path.split('/').pop() || tab.path;
    return fileName;
  };

  const getTabIcon = (tab: EditorTab) => {
    const ext = tab.path.split('.').pop()?.toLowerCase();
    const iconMap: Record<string, string> = {
      js: '📄',
      jsx: '⚛️',
      ts: '📘',
      tsx: '⚛️',
      py: '🐍',
      go: '🐹',
      rs: '🦀',
      java: '☕',
      html: '🌐',
      css: '🎨',
      json: '📋',
      md: '📝',
      txt: '📄',
      yml: '⚙️',
      yaml: '⚙️',
    };
    return iconMap[ext || ''] || '📄';
  };

  const editorOptions: import('monaco-editor').editor.IStandaloneEditorConstructionOptions = {
    theme: 'vs-dark',
    fontSize: 14,
    lineNumbers: 'on',
    minimap: { enabled: true },
    wordWrap: 'off',
    folding: true,
    automaticLayout: true,
    scrollBeyondLastLine: false,
    renderWhitespace: 'selection',
    cursorBlinking: 'blink',
    cursorSmoothCaretAnimation: 'on',
    tabSize: 4,
    insertSpaces: true,
    detectIndentation: true,
    smoothScrolling: true,
    bracketPairColorization: { enabled: true },
    multiCursorModifier: 'ctrlCmd',
  };

  return (
    <div
      className="border-r border-slack-border bg-slack-bg flex flex-col h-full relative animate-slide-in-left flex-shrink-0"
      style={{ width: `${width}px`, minWidth: `${MIN_WIDTH}px` }}
    >
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
            onClick={() => void handleSave()}
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
              {tab.isDirty && <span className="text-xs text-yellow-500">●</span>}
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

      <div className="flex-1 min-h-0">
        {activeTab ? (
          <Editor
            height="100%"
            language={activeTab.language || 'plaintext'}
            defaultValue=""
            onMount={handleEditorDidMount}
            options={editorOptions}
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

      {activeTab && (
        <div className="px-4 py-2 border-t border-slack-border bg-slack-bgHover text-xs text-slack-textMuted flex items-center justify-between">
          <div className="flex items-center gap-4">
            <span>{activeTab.path}</span>
            {activeTab.language && (
              <span className="px-2 py-1 bg-slack-bg rounded text-xs">{activeTab.language}</span>
            )}
          </div>
          <div className="flex items-center gap-2">
            {saving && <span className="text-yellow-500">Saving...</span>}
            <button
              onClick={() => void handleSave()}
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
