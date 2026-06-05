import { useState, useEffect, useRef, useCallback } from 'react';
import { Editor } from '@monaco-editor/react';
import { shallow } from 'zustand/shallow';
import { useEditorStore } from '../stores/editorStore';
import { useToastStore } from '../stores/toastStore';
import { useEditorShortcuts } from '../hooks/useEditorShortcuts';
import { useInlinePendingHunks, useMonacoDiagnostics } from '../hooks/useInlinePendingHunks';
import { useInlineCompletion } from '../hooks/useInlineCompletion';
import { EditorReviewBar } from './EditorReviewBar';
import { useFileExplorerStore } from '../stores/fileExplorerStore';
import { usePacksStore } from '../stores/packsStore';
import { useSettingsStore } from '../stores/settingsStore';
import { useDiagnosticsStore } from '../stores/diagnosticsStore';
import { ChatAPI } from '../api/chatAPI';
import { getHubBaseURL } from '../config/hubUrl';
import type { EditorTab } from '../stores/editorStore';
import { EditorImagePreview } from './EditorImagePreview';
import { ScanSummaryViewer } from './ScanSummaryViewer';
import { CadWorkbench } from './CadWorkbench';
import { ScanAnalysisViewer } from './ScanAnalysisViewer';
import { ComparatorAnalysisViewer } from './ComparatorAnalysisViewer';
import { ErrorBoundary } from './ErrorBoundary';
import { shrinkablePanelStyle } from '../utils/panelLayout';
import { getMonacoThemeId, registerMonacoThemes } from '../utils/editorThemes';
import { CsvTableViewer } from './CsvTableViewer';
import { isEditableCsvPath } from '../utils/csvTable';

function tabLabel(tab: EditorTab): string {
  const path = tab.path ?? '';
  if (!path) return 'Untitled';
  const parts = path.split('/');
  return parts[parts.length - 1] || path;
}

interface CodeEditorPanelProps {
  onClose: () => void;
  /** embedded = flex center column in IDE layout; overlay = resizable slide-in panel */
  variant?: 'overlay' | 'embedded';
}

const MIN_WIDTH = 300;
const COMPACT_MIN_WIDTH = 220;
const DEFAULT_WIDTH = 600;
const STORAGE_KEY = 'code-editor-panel-width';

export function CodeEditorPanel({ onClose, variant = 'overlay' }: CodeEditorPanelProps) {
  const embedded = variant === 'embedded';
  const {
    tabs,
    activeTabId,
    saving,
    error,
    hasUnsavedChanges,
    setActiveTab,
    saveTab,
    saveAllTabs,
    closeTab,
    setTabViewMode,
    updateTabContent,
  } = useEditorStore(
    (s) => ({
      tabs: s.tabs,
      activeTabId: s.activeTabId,
      saving: s.saving,
      error: s.error,
      hasUnsavedChanges: s.tabs.some((t) => t.isDirty),
      setActiveTab: s.setActiveTab,
      saveTab: s.saveTab,
      saveAllTabs: s.saveAllTabs,
      closeTab: s.closeTab,
      setTabViewMode: s.setTabViewMode,
      updateTabContent: s.updateTabContent,
    }),
    shallow
  );

  const activeTab = useEditorStore((s) =>
    s.tabs.find((tab) => tab.id === s.activeTabId) ?? null
  );

  const activeContentSyncKey = activeTab?.contentSyncKey ?? 0;

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
  const revealRequest = useEditorStore((s) => s.revealRequest);
  const clearRevealRequest = useEditorStore((s) => s.clearRevealRequest);

  const inlineCompletionOn = useSettingsStore(
    (s) => s.layoutSettings.inlineCompletionEnabled ?? false
  );
  const colorTheme = useSettingsStore((s) => s.settings.colorTheme ?? 'slack');
  const monacoThemeId = getMonacoThemeId(colorTheme);
  const devPack = usePacksStore((s) => s.softwareDevelopmentEnabled());
  const workspaceRoot = useFileExplorerStore((s) => {
    const tab = activeTab;
    if (!tab) return undefined;
    return s.workspaces.find((w) => w.id === tab.workspaceId)?.path;
  });

  useMonacoDiagnostics(editor, monacoRef.current, activeTab?.path, activeTab?.language);
  useInlinePendingHunks(editor, monacoRef.current, activeTabId);
  useInlineCompletion(
    editor,
    monacoRef.current,
    devPack && inlineCompletionOn,
    activeTab?.language,
    activeTab?.path
  );

  useEffect(() => {
    if (!editor || !revealRequest || !activeTab) return;
    if (
      revealRequest.workspaceId !== activeTab.workspaceId ||
      revealRequest.path !== activeTab.path
    ) {
      return;
    }
    editor.revealLineInCenter(revealRequest.line);
    editor.setPosition({ lineNumber: revealRequest.line, column: 1 });
    clearRevealRequest();
  }, [editor, revealRequest, activeTab, clearRevealRequest]);

  useEffect(() => {
    if (!activeTab?.workspaceId || !activeTab.path) return;
    const lang = activeTab.language;
    const ext = activeTab.path.split('.').pop()?.toLowerCase();
    let lspLang: 'go' | 'rust' | 'python' | null = null;
    if (lang === 'go' || ext === 'go') lspLang = 'go';
    else if (lang === 'rust' || ext === 'rs') lspLang = 'rust';
    else if (lang === 'python' || ext === 'py') lspLang = 'python';
    if (!lspLang) return;
    let cancelled = false;
    const api = new ChatAPI(getHubBaseURL());
    const fetch =
      lspLang === 'go'
        ? api.getGoLSPDiagnostics(activeTab.workspaceId)
        : api.getLSPDiagnostics(lspLang, activeTab.workspaceId);
    void fetch.then((items) => {
      if (cancelled) return;
      for (const d of items) {
        if (d.path === activeTab.path || d.path.endsWith('/' + activeTab.path)) {
          useDiagnosticsStore.getState().setForPath(activeTab.path, [
            ...(useDiagnosticsStore.getState().byPath[activeTab.path] ?? []).filter(
              (x) => x.message !== d.message
            ),
            {
              path: activeTab.path,
              line: d.line,
              column: d.column,
              message: d.message,
              severity: d.severity === 'warning' ? 'warning' : 'error',
            },
          ]);
        }
      }
    });
    return () => {
      cancelled = true;
    };
  }, [activeTab?.workspaceId, activeTab?.path, activeTab?.language]);

  const isImageTab = activeTab?.viewMode === 'image';
  const isScanSummaryTab = activeTab?.viewMode === 'scan-summary';
  const isScanAnalysisTab = activeTab?.viewMode === 'scan-analysis';
  const isCadWorkbenchTab = activeTab?.viewMode === 'cad-workbench';
  const isCsvTableTab = activeTab?.viewMode === 'csv-table';
  const isCsvFileTab = activeTab ? isEditableCsvPath(activeTab.path) : false;
  const isPreviewTab = isImageTab || isScanSummaryTab || isScanAnalysisTab || isCadWorkbenchTab;

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
    if (!tab || tab.viewMode === 'image' || tab.viewMode === 'csv-table' || tab.viewMode === 'scan-summary' || tab.viewMode === 'scan-analysis' || tab.viewMode === 'cad-workbench') return;

    const syncKey = tab.contentSyncKey ?? 0;
    const tabSwitched = lastAppliedRef.current.tabId !== activeTabId;

    try {
      let model = tabModelsRef.current.get(activeTabId);
      if (!model || model.isDisposed()) {
        const safePath = tab.path || 'untitled';
        const uri = monaco.Uri.parse(
          `nj://${tab.workspaceId || 'ws'}/${encodeURIComponent(safePath)}?tab=${encodeURIComponent(activeTabId)}`
        );
        model = monaco.editor.createModel(tab.content ?? '', tab.language || 'plaintext', uri);
        tabModelsRef.current.set(activeTabId, model);
      } else if (model.getValue() !== tab.content) {
        model.setValue(tab.content ?? '');
      }

      monaco.editor.setModelLanguage(model, tab.language || 'plaintext');

      if (tabSwitched) {
        const prev = lastAppliedRef.current.tabId;
        if (prev && prev !== activeTabId) {
          viewStatesRef.current.set(prev, editor.saveViewState());
        }
        editor.setModel(model);
        const vs = viewStatesRef.current.get(activeTabId);
        if (vs) {
          editor.restoreViewState(vs);
        }
      }

      lastAppliedRef.current = { tabId: activeTabId, syncKey };
    } catch (err) {
      console.error('[CodeEditorPanel] Monaco model sync failed:', err);
    }
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
    if (!tab || tab.viewMode === 'image' || tab.viewMode === 'csv-table' || tab.viewMode === 'scan-summary' || tab.viewMode === 'scan-analysis' || tab.viewMode === 'cad-workbench' || useEditorStore.getState().saving) return;

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

  useEffect(() => {
    const monaco = monacoRef.current;
    if (!monaco) return;
    monaco.editor.setTheme(monacoThemeId);
  }, [monacoThemeId]);

  const handleEditorBeforeMount = (monaco: typeof import('monaco-editor')) => {
    registerMonacoThemes(monaco);
  };

  const handleEditorDidMount = (
    ed: import('monaco-editor').editor.IStandaloneCodeEditor,
    monaco: typeof import('monaco-editor')
  ) => {
    monacoRef.current = monaco;
    registerMonacoThemes(monaco);
    monaco.editor.setTheme(monacoThemeId);
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

    const subSelection = ed.onDidChangeCursorSelection(() => {
      const sel = ed.getSelection();
      const model = ed.getModel();
      if (!sel || !model || sel.isEmpty()) {
        useEditorStore.getState().setActiveSelection(null);
        return;
      }
      const text = model.getValueInRange(sel);
      if (!text.trim()) {
        useEditorStore.getState().setActiveSelection(null);
        return;
      }
      useEditorStore.getState().setActiveSelection({
        startLine: sel.startLineNumber,
        endLine: sel.endLineNumber,
        text: text.length > 2048 ? text.slice(0, 2048) : text,
      });
    });
    editorListenersRef.current.push(subSelection);
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

  const getTabIcon = (tab: EditorTab) => {
    if (tab.viewMode === 'image') return '🖼️';
    if (tab.viewMode === 'cad-workbench') return '📐';
    if (tab.viewMode === 'scan-summary') return '🔬';
    if (tab.viewMode === 'scan-analysis') return '📊';
    if (tab.viewMode === 'comparator-analysis') return '📈';
    if (tab.viewMode === 'csv-table') return '🧮';
    const ext = (tab.path ?? '').split('.').pop()?.toLowerCase();
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
      csv: '🧮',
      yml: '⚙️',
      yaml: '⚙️',
      png: '🖼️',
    };
    return iconMap[ext || ''] || '📄';
  };

  const editorOptions: import('monaco-editor').editor.IStandaloneEditorConstructionOptions = {
    theme: monacoThemeId,
    fontSize: 14,
    lineNumbers: 'on',
    minimap: { enabled: true },
    wordWrap: 'off',
    folding: true,
    automaticLayout: true,
    scrollBeyondLastLine: false,
    renderWhitespace: 'selection',
    cursorBlinking: 'blink',
    tabSize: 4,
    insertSpaces: true,
    detectIndentation: true,
    smoothScrolling: true,
    bracketPairColorization: { enabled: true },
    multiCursorModifier: 'ctrlCmd',
  };

  return (
    <div
      className={
        embedded
          ? 'border-r border-slack-border bg-slack-bg flex flex-col h-full min-w-0 flex-1 relative'
          : 'border-r border-slack-border bg-slack-bg flex flex-col h-full relative animate-slide-in-left'
      }
      style={embedded ? undefined : shrinkablePanelStyle(width, COMPACT_MIN_WIDTH)}
    >
      {!embedded && (
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
      )}

      <EditorReviewBar workspaceRoot={workspaceRoot} />

      <div className="px-4 py-3 border-b border-slack-border flex items-center justify-between bg-slack-bgHover">
        <h2 className="font-bold text-slack-text">Code Editor</h2>
        <div className="flex items-center gap-2">
          {hasUnsavedChanges && (
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
            disabled={saving || !activeTab || isPreviewTab}
            className="px-2 py-1 text-xs bg-green-600 hover:bg-green-700 disabled:bg-gray-600 disabled:cursor-not-allowed text-white rounded transition-colors"
            title={isPreviewTab ? 'Preview only' : 'Save current file (Cmd+S)'}
          >
            Save
          </button>
          <button
            onClick={handleSaveAll}
            disabled={saving || !hasUnsavedChanges}
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
              <span className="text-sm truncate max-w-32">{tabLabel(tab)}</span>
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
          activeTab.viewMode === 'scan-analysis' && activeTab.scanAnalysisData != null ? (
            <ErrorBoundary
              fallback={
                <div className="p-4 text-sm text-red-300">
                  Scan analysis viewer crashed. Close this tab and reopen the analysis folder.
                </div>
              }
            >
              <ScanAnalysisViewer
                workspaceId={activeTab.workspaceId}
                analysisDir={activeTab.scanAnalysisDir ?? ''}
                data={activeTab.scanAnalysisData}
                initialWell={activeTab.scanAnalysisInitialWell ?? 'A1'}
                initialAnalyte={activeTab.scanAnalysisSelectedAnalyte}
                linkedScanDir={activeTab.linkedScanDir}
                tabId={activeTab.id}
              />
            </ErrorBoundary>
          ) : activeTab.viewMode === 'comparator-analysis' ? (
            <ErrorBoundary
              fallback={
                <div className="p-4 text-sm text-red-300">
                  Comparator viewer crashed. Close this tab and reopen the analysis output folder.
                </div>
              }
            >
              <ComparatorAnalysisViewer
                workspaceId={activeTab.workspaceId}
                analysisDir={activeTab.comparatorAnalysisDir ?? ''}
              />
            </ErrorBoundary>
          ) : activeTab.viewMode === 'scan-summary' && activeTab.scanSummaryData != null ? (
            <ScanSummaryViewer
              workspaceId={activeTab.workspaceId}
              summaryDir={activeTab.scanSummaryDir ?? ''}
              data={activeTab.scanSummaryData}
              initialWell={activeTab.scanSummaryInitialWell ?? 'A1'}
              linkedAnalysisDir={activeTab.linkedAnalysisDir}
            />
          ) : activeTab.viewMode === 'cad-workbench' && activeTab.cadScadPath ? (
            <CadWorkbench
              key={`${activeTab.id}:${activeTab.cadScadPath}`}
              workspaceId={activeTab.workspaceId}
              scadPath={activeTab.cadScadPath}
              initialContent={activeTab.content}
              projectId={activeTab.cadProjectId}
              tabId={activeTab.id}
            />
          ) : activeTab.viewMode === 'csv-table' ? (
            <CsvTableViewer
              content={activeTab.content}
              onContentChange={(csv) => updateTabContent(activeTab.id, csv)}
            />
          ) : activeTab.viewMode === 'image' ? (
            activeTab.imageSrc ? (
              <EditorImagePreview
                src={activeTab.imageSrc}
                alt={tabLabel(activeTab)}
                reloadKey={activeTab.contentSyncKey ?? 0}
              />
            ) : (
              <div className="flex items-center justify-center h-full text-slack-textMuted p-6">
                <div className="text-center text-sm">Image preview unavailable</div>
              </div>
            )
          ) : (
            <Editor
              key={activeTabId ?? 'none'}
              height="100%"
              language={activeTab.language || 'plaintext'}
              defaultValue=""
              theme={monacoThemeId}
              beforeMount={handleEditorBeforeMount}
              onMount={handleEditorDidMount}
              options={editorOptions}
            />
          )
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
            {isPreviewTab ? (
              <span className="px-2 py-1 bg-slack-bg rounded text-xs">
                {isCadWorkbenchTab ? 'CAD workbench' : isScanAnalysisTab ? 'Scan analysis' : isScanSummaryTab ? 'Scan summary' : 'Preview only'}
              </span>
            ) : isCsvTableTab ? (
              <span className="px-2 py-1 bg-slack-bg rounded text-xs">CSV table</span>
            ) : (
              activeTab.language && (
                <span className="px-2 py-1 bg-slack-bg rounded text-xs">{activeTab.language}</span>
              )
            )}
            {isCsvFileTab && (
              <div className="flex rounded border border-slack-border overflow-hidden text-xs">
                <button
                  type="button"
                  onClick={() => setTabViewMode(activeTab.id, 'csv-table')}
                  className={`px-2 py-0.5 ${isCsvTableTab ? 'bg-slack-accent text-white' : 'text-slack-textMuted hover:bg-slack-bgHover'}`}
                >
                  Table
                </button>
                <button
                  type="button"
                  onClick={() =>
                    setTabViewMode(activeTab.id, 'text')
                  }
                  className={`px-2 py-0.5 border-l border-slack-border ${!isCsvTableTab ? 'bg-slack-accent text-white' : 'text-slack-textMuted hover:bg-slack-bgHover'}`}
                >
                  Text
                </button>
              </div>
            )}
          </div>
          {!isPreviewTab && (
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
          )}
        </div>
      )}
    </div>
  );
}
