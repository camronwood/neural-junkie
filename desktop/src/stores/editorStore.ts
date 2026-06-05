import { createWithEqualityFn as create } from 'zustand/traditional';
import { ChatAPI } from '../api/chatAPI';
import { getHubBaseURL } from '../config/hubUrl';
import type { ScanSummaryData } from '../utils/scanSummary';
import { isScanSummaryWellPath, SCAN_SUMMARY_METADATA_FILE } from '../utils/scanSummary';
import type { ScanAnalysisData } from '../utils/scanAnalysis';
import { isScanAnalysisResultsPath, SCAN_ANALYSIS_RESULTS_FILE } from '../utils/scanAnalysis';
import type { PanelQCReport } from '../utils/secondaryAnalysis';
import { getLanguageFromPath } from '../utils/editorLanguage';

const api = new ChatAPI(getHubBaseURL());

export type EditorTabViewMode = 'text' | 'csv-table' | 'image' | 'scan-summary' | 'scan-analysis' | 'comparator-analysis' | 'cad-workbench';

export interface EditorTab {
  id: string;
  workspaceId: string;
  path: string;
  content: string;
  isDirty: boolean;
  /** Bumped on disk refresh so the editor can sync without tying effects to every keystroke. */
  contentSyncKey?: number;
  cursorPosition?: { line: number; column: number };
  language?: string;
  viewMode?: EditorTabViewMode;
  /** Tauri asset URL or path for image preview tabs. */
  imageSrc?: string;
  /** Relative path to scan summary directory (parent of imageMetadata.json). */
  scanSummaryDir?: string;
  scanSummaryData?: ScanSummaryData;
  /** Well to select when the scan summary viewer opens. */
  scanSummaryInitialWell?: string;
  /** Relative path to scan analysis directory (parent of reports/results.json). */
  scanAnalysisDir?: string;
  scanAnalysisData?: ScanAnalysisData;
  scanAnalysisInitialWell?: string;
  scanAnalysisSelectedAnalyte?: string;
  /** Linked scan summary directory for image comparison. */
  linkedScanDir?: string;
  /** Linked analysis directory when viewing scan summary. */
  linkedAnalysisDir?: string;
  /** 12-Plex QC report cached on scan analysis tab. */
  panelQCReport?: PanelQCReport;
  /** Relative path to Comparator Analysis output folder. */
  comparatorAnalysisDir?: string;
  /** CAD workbench: relative .scad path in workspace. */
  cadScadPath?: string;
  cadProjectId?: string;
}

export interface OpenFileOptions {
  viewMode?: EditorTabViewMode;
  imageSrc?: string;
  scanSummaryDir?: string;
  scanSummaryData?: ScanSummaryData;
  scanAnalysisDir?: string;
  scanAnalysisData?: ScanAnalysisData;
  linkedScanDir?: string;
  linkedAnalysisDir?: string;
}

export interface EditorSelectionContext {
  startLine: number;
  endLine: number;
  text: string;
}

interface EditorState {
  // Open tabs
  tabs: EditorTab[];
  activeTabId: string | null;
  /** Non-empty selection in the active Monaco editor (for dev-pack agent context). */
  activeSelection: EditorSelectionContext | null;
  /** Jump-to-line request consumed by CodeEditorPanel. */
  revealRequest: { workspaceId: string; path: string; line: number } | null;

  // Loading and error states
  saving: boolean;
  error: string | null;
  
  // Actions
  openFile: (
    workspaceId: string,
    path: string,
    content: string,
    language?: string,
    options?: OpenFileOptions
  ) => void;
  openScanSummary: (
    workspaceId: string,
    summaryDir: string,
    data: ScanSummaryData,
    initialWell?: string,
    options?: { linkedAnalysisDir?: string }
  ) => void;
  openScanAnalysis: (
    workspaceId: string,
    analysisDir: string,
    data: ScanAnalysisData,
    options?: {
      initialWell?: string;
      selectedAnalyte?: string;
      linkedScanDir?: string;
    }
  ) => void;
  openComparatorAnalysis: (workspaceId: string, analysisDir: string) => void;
  setPanelQCReport: (tabId: string, report: PanelQCReport | undefined) => void;
  openCadWorkbench: (
    workspaceId: string,
    scadPath: string,
    content: string,
    options?: { projectId?: string }
  ) => void;
  linkScanToAnalysisTab: (tabId: string, scanDir: string) => void;
  linkAnalysisToScanTab: (tabId: string, analysisDir: string) => void;
  findLinkedAnalysisTab: (workspaceId: string, scanDir: string) => EditorTab | undefined;
  findLinkedScanTab: (workspaceId: string, analysisDir: string) => EditorTab | undefined;
  activateAnalysisWell: (tabId: string, wellId: string, analyte?: string) => void;
  activateScanWell: (tabId: string, wellId: string) => void;
  closeTab: (tabId: string) => void;
  setActiveTab: (tabId: string) => void;
  setTabViewMode: (tabId: string, viewMode: EditorTabViewMode) => void;
  updateTabContent: (tabId: string, content: string) => void;
  updateTabCursor: (tabId: string, position: { line: number; column: number }) => void;
  setActiveSelection: (selection: EditorSelectionContext | null) => void;
  revealLine: (workspaceId: string, path: string, line: number) => void;
  clearRevealRequest: () => void;
  markTabDirty: (tabId: string, isDirty: boolean) => void;
  saveTab: (tabId: string) => Promise<boolean>;
  saveAllTabs: () => Promise<boolean>;
  refreshTabFromDisk: (workspaceId: string, path: string) => Promise<void>;
  closeAllTabs: () => void;
  closeOtherTabs: (keepTabId: string) => void;
  closeTabsToRight: (tabId: string) => void;
  closeTabsToLeft: (tabId: string) => void;
  setError: (error: string | null) => void;
  
  // Getters
  getActiveTab: () => EditorTab | null;
  getTabById: (tabId: string) => EditorTab | null;
  getTabByPath: (workspaceId: string, path: string) => EditorTab | null;
  hasUnsavedChanges: () => boolean;
}

export const useEditorStore = create<EditorState>((set, get) => ({
  tabs: [],
  activeTabId: null,
  activeSelection: null,
  revealRequest: null,
  saving: false,
  error: null,
  
  openFile: (workspaceId, path, content, language, options) => {
    const state = get();
    const viewMode = options?.viewMode ?? 'text';
    const imageSrc = options?.imageSrc;

    // Check if file is already open
    const existingTab = state.getTabByPath(workspaceId, path);
    if (existingTab) {
      set({ activeTabId: existingTab.id });
      return;
    }

    // Create new tab
    const newTab: EditorTab = {
      id: `tab_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`,
      workspaceId,
      path,
      content,
      isDirty: false,
      contentSyncKey: 0,
      language: viewMode === 'image' || viewMode === 'csv-table' || viewMode === 'scan-summary' || viewMode === 'scan-analysis' || viewMode === 'comparator-analysis' || viewMode === 'cad-workbench' ? undefined : language,
      viewMode,
      imageSrc,
      scanSummaryDir: options?.scanSummaryDir,
      scanSummaryData: options?.scanSummaryData,
      scanAnalysisDir: options?.scanAnalysisDir,
      scanAnalysisData: options?.scanAnalysisData,
      linkedScanDir: options?.linkedScanDir,
      linkedAnalysisDir: options?.linkedAnalysisDir,
    };
    
    set({
      tabs: [...state.tabs, newTab],
      activeTabId: newTab.id,
    });
  },

  openScanSummary: (workspaceId, summaryDir, data, initialWell, options) => {
    const state = get();
    const path = summaryDir
      ? `${summaryDir.replace(/\/$/, '')}/${SCAN_SUMMARY_METADATA_FILE}`
      : SCAN_SUMMARY_METADATA_FILE;
    const existingTab = state.tabs.find(
      (t) => t.workspaceId === workspaceId && t.viewMode === 'scan-summary' && t.scanSummaryDir === summaryDir
    );
    if (existingTab) {
      set({
        activeTabId: existingTab.id,
        tabs: state.tabs.map((t) =>
          t.id === existingTab.id
            ? {
                ...t,
                scanSummaryData: data,
                scanSummaryInitialWell: initialWell ?? t.scanSummaryInitialWell,
                linkedAnalysisDir: options?.linkedAnalysisDir ?? t.linkedAnalysisDir,
              }
            : t
        ),
      });
      return;
    }
    // Replace a text tab for the same metadata file if the user opened JSON by mistake.
    const staleTextTab = state.tabs.find(
      (t) =>
        t.workspaceId === workspaceId &&
        t.viewMode === 'text' &&
        (t.path === path ||
          t.path.endsWith(`/${SCAN_SUMMARY_METADATA_FILE}`) ||
          (initialWell != null && isScanSummaryWellPath(t.path)))
    );
    const tabId =
      staleTextTab?.id ?? `tab_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    const newTab: EditorTab = {
      id: tabId,
      workspaceId,
      path,
      content: '',
      isDirty: false,
      viewMode: 'scan-summary',
      scanSummaryDir: summaryDir,
      scanSummaryData: data,
      scanSummaryInitialWell: initialWell,
      linkedAnalysisDir: options?.linkedAnalysisDir,
    };
    set({
      tabs: staleTextTab
        ? state.tabs.map((t) => (t.id === staleTextTab.id ? newTab : t))
        : [...state.tabs, newTab],
      activeTabId: tabId,
    });
  },

  openScanAnalysis: (workspaceId, analysisDir, data, options) => {
    const state = get();
    const path = analysisDir
      ? `${analysisDir.replace(/\/$/, '')}/${SCAN_ANALYSIS_RESULTS_FILE}`
      : SCAN_ANALYSIS_RESULTS_FILE;
    const selectedAnalyte =
      options?.selectedAnalyte && data.analytes.includes(options.selectedAnalyte)
        ? options.selectedAnalyte
        : data.analytes[0];
    const existingTab = state.tabs.find(
      (t) =>
        t.workspaceId === workspaceId &&
        t.viewMode === 'scan-analysis' &&
        t.scanAnalysisDir === analysisDir
    );
    if (existingTab) {
      set({
        activeTabId: existingTab.id,
        tabs: state.tabs.map((t) =>
          t.id === existingTab.id
            ? {
                ...t,
                scanAnalysisData: data,
                scanAnalysisInitialWell: options?.initialWell ?? t.scanAnalysisInitialWell,
                scanAnalysisSelectedAnalyte: selectedAnalyte ?? t.scanAnalysisSelectedAnalyte,
                linkedScanDir: options?.linkedScanDir ?? t.linkedScanDir,
              }
            : t
        ),
      });
      return;
    }
    const staleTextTab = state.tabs.find(
      (t) =>
        t.workspaceId === workspaceId &&
        t.viewMode === 'text' &&
        (t.path === path ||
          t.path.endsWith(`/${SCAN_ANALYSIS_RESULTS_FILE}`) ||
          isScanAnalysisResultsPath(t.path))
    );
    const tabId =
      staleTextTab?.id ?? `tab_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    const newTab: EditorTab = {
      id: tabId,
      workspaceId,
      path,
      content: '',
      isDirty: false,
      viewMode: 'scan-analysis',
      scanAnalysisDir: analysisDir,
      scanAnalysisData: data,
      scanAnalysisInitialWell: options?.initialWell ?? 'A1',
      scanAnalysisSelectedAnalyte: selectedAnalyte,
      linkedScanDir: options?.linkedScanDir,
    };
    set({
      tabs: staleTextTab
        ? state.tabs.map((t) => (t.id === staleTextTab.id ? newTab : t))
        : [...state.tabs, newTab],
      activeTabId: tabId,
    });
  },

  openComparatorAnalysis: (workspaceId, analysisDir) => {
    const state = get();
    const path = `${analysisDir.replace(/\/$/, '')}/Summary Statistics/LLOQs_and_ULOQs.csv`;
    const existingTab = state.tabs.find(
      (t) =>
        t.workspaceId === workspaceId &&
        t.viewMode === 'comparator-analysis' &&
        t.comparatorAnalysisDir === analysisDir
    );
    if (existingTab) {
      set({ activeTabId: existingTab.id });
      return;
    }
    const tabId = `tab_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    const newTab: EditorTab = {
      id: tabId,
      workspaceId,
      path,
      content: '',
      isDirty: false,
      viewMode: 'comparator-analysis',
      comparatorAnalysisDir: analysisDir,
    };
    set({
      tabs: [...state.tabs, newTab],
      activeTabId: tabId,
    });
  },

  setPanelQCReport: (tabId, report) => {
    set({
      tabs: get().tabs.map((t) => (t.id === tabId ? { ...t, panelQCReport: report } : t)),
    });
  },

  openCadWorkbench: (workspaceId, scadPath, content, options) => {
    const state = get();
    const normalized = scadPath.replace(/\\/g, '/').replace(/^\/+/, '');

    const existingTab = state.tabs.find(
      (t) =>
        t.workspaceId === workspaceId &&
        t.viewMode === 'cad-workbench' &&
        (t.cadScadPath ?? t.path).replace(/\\/g, '/').replace(/^\/+/, '') === normalized
    );
    if (existingTab) {
      set({
        activeTabId: existingTab.id,
        tabs: state.tabs.map((t) =>
          t.id === existingTab.id
            ? { ...t, path: normalized, cadScadPath: normalized, content, isDirty: false }
            : t
        ),
      });
      return;
    }

    const activeTab = state.activeTabId
      ? state.tabs.find((t) => t.id === state.activeTabId)
      : null;
    if (
      activeTab?.viewMode === 'cad-workbench' &&
      activeTab.workspaceId === workspaceId &&
      (activeTab.cadScadPath ?? activeTab.path).replace(/\\/g, '/').replace(/^\/+/, '') !== normalized
    ) {
      set({
        activeTabId: activeTab.id,
        tabs: state.tabs.map((t) =>
          t.id === activeTab.id
            ? {
                ...t,
                path: normalized,
                cadScadPath: normalized,
                content,
                isDirty: false,
                contentSyncKey: (t.contentSyncKey ?? 0) + 1,
                cadProjectId: options?.projectId ?? t.cadProjectId,
              }
            : t
        ),
      });
      return;
    }

    const staleTextTab = state.tabs.find(
      (t) => t.workspaceId === workspaceId && t.viewMode === 'text' && t.path.replace(/\\/g, '/').replace(/^\/+/, '') === normalized
    );
    const tabId =
      staleTextTab?.id ?? `tab_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    const newTab: EditorTab = {
      id: tabId,
      workspaceId,
      path: normalized,
      content,
      isDirty: false,
      viewMode: 'cad-workbench',
      cadScadPath: normalized,
      cadProjectId: options?.projectId,
    };
    set({
      tabs: staleTextTab
        ? state.tabs.map((t) => (t.id === staleTextTab.id ? newTab : t))
        : [...state.tabs, newTab],
      activeTabId: tabId,
    });
  },

  linkScanToAnalysisTab: (tabId, scanDir) => {
    set({
      tabs: get().tabs.map((t) =>
        t.id === tabId && t.viewMode === 'scan-analysis' ? { ...t, linkedScanDir: scanDir } : t
      ),
    });
  },

  linkAnalysisToScanTab: (tabId, analysisDir) => {
    set({
      tabs: get().tabs.map((t) =>
        t.id === tabId && t.viewMode === 'scan-summary'
          ? { ...t, linkedAnalysisDir: analysisDir }
          : t
      ),
    });
  },

  findLinkedAnalysisTab: (workspaceId, scanDir) => {
    const normalized = scanDir.replace(/[/\\]+$/, '');
    return get().tabs.find(
      (t) =>
        t.workspaceId === workspaceId &&
        t.viewMode === 'scan-analysis' &&
        (t.linkedScanDir === normalized || t.scanAnalysisDir === normalized)
    );
  },

  findLinkedScanTab: (workspaceId, analysisDir) => {
    const normalized = analysisDir.replace(/[/\\]+$/, '');
    return get().tabs.find(
      (t) =>
        t.workspaceId === workspaceId &&
        t.viewMode === 'scan-summary' &&
        (t.scanSummaryDir === normalized || t.linkedAnalysisDir === normalized)
    );
  },

  activateAnalysisWell: (tabId, wellId, analyte) => {
    set({
      activeTabId: tabId,
      tabs: get().tabs.map((t) =>
        t.id === tabId
          ? {
              ...t,
              scanAnalysisInitialWell: wellId,
              scanAnalysisSelectedAnalyte: analyte ?? t.scanAnalysisSelectedAnalyte,
            }
          : t
      ),
    });
  },

  activateScanWell: (tabId, wellId) => {
    set({
      activeTabId: tabId,
      tabs: get().tabs.map((t) =>
        t.id === tabId ? { ...t, scanSummaryInitialWell: wellId } : t
      ),
    });
  },
  
  closeTab: (tabId) => {
    const state = get();
    const tabIndex = state.tabs.findIndex(tab => tab.id === tabId);
    if (tabIndex === -1) return;
    
    const newTabs = state.tabs.filter(tab => tab.id !== tabId);
    let newActiveTabId = state.activeTabId;
    
    // If we're closing the active tab, set a new active tab
    if (state.activeTabId === tabId) {
      if (newTabs.length === 0) {
        newActiveTabId = null;
      } else if (tabIndex < newTabs.length) {
        // Activate the tab at the same position
        newActiveTabId = newTabs[tabIndex].id;
      } else {
        // Activate the last tab
        newActiveTabId = newTabs[newTabs.length - 1].id;
      }
    }
    
    set({
      tabs: newTabs,
      activeTabId: newActiveTabId,
    });
  },
  
  setActiveTab: (tabId) => {
    set({ activeTabId: tabId });
  },

  setTabViewMode: (tabId, viewMode) => {
    set((state) => ({
      tabs: state.tabs.map((tab) => {
        if (tab.id !== tabId) return tab;
        if (tab.viewMode === viewMode) return tab;
        const language =
          viewMode === 'text' ? tab.language ?? getLanguageFromPath(tab.path) : undefined;
        return { ...tab, viewMode, language };
      }),
    }));
  },
  
  updateTabContent: (tabId, content) => {
    set((state) => ({
      tabs: state.tabs.map((tab) => {
        if (tab.id !== tabId) return tab;
        if (tab.viewMode === 'cad-workbench') {
          return tab.content === content ? tab : { ...tab, content, isDirty: false };
        }
        if (tab.viewMode === 'image' || tab.viewMode === 'csv-table' || tab.viewMode === 'scan-summary' || tab.viewMode === 'scan-analysis' || tab.viewMode === 'comparator-analysis') return tab;
        if (tab.content === content) return tab;
        return { ...tab, content, isDirty: true };
      }),
    }));
  },
  
  updateTabCursor: (tabId, position) => {
    set(state => ({
      tabs: state.tabs.map(tab =>
        tab.id === tabId
          ? { ...tab, cursorPosition: position }
          : tab
      ),
    }));
  },

  setActiveSelection: (selection) => {
    set({ activeSelection: selection });
  },

  revealLine: (workspaceId, path, line) => {
    set({ revealRequest: { workspaceId, path, line: Math.max(1, line) } });
  },

  clearRevealRequest: () => {
    set({ revealRequest: null });
  },
  
  markTabDirty: (tabId, isDirty) => {
    set(state => ({
      tabs: state.tabs.map(tab =>
        tab.id === tabId
          ? { ...tab, isDirty }
          : tab
      ),
    }));
  },
  
  saveTab: async (tabId) => {
    const state = get();
    const tab = state.getTabById(tabId);
    if (!tab) return false;
    if (tab.viewMode === 'image' || tab.viewMode === 'scan-summary' || tab.viewMode === 'scan-analysis' || tab.viewMode === 'comparator-analysis' || tab.viewMode === 'cad-workbench') return true;

    set({ saving: true, error: null });
    
    try {
      await api.saveFileContent(tab.workspaceId, tab.path, tab.content);
      
      // Mark as not dirty after successful save
      set(state => ({
        tabs: state.tabs.map(t =>
          t.id === tabId
            ? { ...t, isDirty: false }
            : t
        ),
        saving: false,
        error: null,
      }));
      
      return true;
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : 'Failed to save file';
      set({ saving: false, error: errorMessage });
      return false;
    }
  },
  
  saveAllTabs: async () => {
    const state = get();
    const dirtyTabs = state.tabs.filter(
      (tab) => tab.isDirty && tab.viewMode !== 'image' && tab.viewMode !== 'scan-summary' && tab.viewMode !== 'scan-analysis' && tab.viewMode !== 'comparator-analysis' && tab.viewMode !== 'cad-workbench'
    );
    
    if (dirtyTabs.length === 0) return true;
    
    set({ saving: true, error: null });
    
    try {
      const savePromises = dirtyTabs.map(tab => 
        api.saveFileContent(tab.workspaceId, tab.path, tab.content)
      );
      
      await Promise.all(savePromises);
      
      // Mark all tabs as not dirty
      set(state => ({
        tabs: state.tabs.map(tab => ({ ...tab, isDirty: false })),
        saving: false,
        error: null,
      }));
      
      return true;
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : 'Failed to save files';
      set({ saving: false, error: errorMessage });
      return false;
    }
  },

  refreshTabFromDisk: async (workspaceId, path) => {
    const state = get();
    const tab = state.getTabByPath(workspaceId, path);
    if (!tab || tab.isDirty) {
      // Never overwrite unsaved edits in the editor.
      return;
    }

    if (tab.viewMode === 'image' || tab.viewMode === 'scan-summary') {
      set(current => ({
        tabs: current.tabs.map(t =>
          t.id === tab.id
            ? {
                ...t,
                contentSyncKey: (t.contentSyncKey ?? 0) + 1,
              }
            : t
        ),
      }));
      return;
    }

    if (tab.viewMode === 'cad-workbench') {
      try {
        const latestContent = await api.fetchFileContent(workspaceId, path);
        set(current => ({
          tabs: current.tabs.map(t =>
            t.id === tab.id
              ? {
                  ...t,
                  content: latestContent,
                  isDirty: false,
                  contentSyncKey: (t.contentSyncKey ?? 0) + 1,
                }
              : t
          ),
        }));
      } catch (error) {
        const errorMessage = error instanceof Error ? error.message : 'Failed to refresh file from disk';
        set({ error: errorMessage });
      }
      return;
    }

    try {
      const latestContent = await api.fetchFileContent(workspaceId, path);
      set(current => ({
        tabs: current.tabs.map(t =>
          t.id === tab.id
            ? {
                ...t,
                content: latestContent,
                isDirty: false,
                contentSyncKey: (t.contentSyncKey ?? 0) + 1,
              }
            : t
        ),
      }));
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : 'Failed to refresh file from disk';
      set({ error: errorMessage });
    }
  },
  
  closeAllTabs: () => {
    set({ tabs: [], activeTabId: null });
  },
  
  closeOtherTabs: (keepTabId) => {
    set(state => ({
      tabs: state.tabs.filter(tab => tab.id === keepTabId),
      activeTabId: keepTabId,
    }));
  },
  
  closeTabsToRight: (tabId) => {
    const state = get();
    const tabIndex = state.tabs.findIndex(tab => tab.id === tabId);
    if (tabIndex === -1) return;
    
    const newTabs = state.tabs.slice(0, tabIndex + 1);
    set({ tabs: newTabs });
  },
  
  closeTabsToLeft: (tabId) => {
    const state = get();
    const tabIndex = state.tabs.findIndex(tab => tab.id === tabId);
    if (tabIndex === -1) return;
    
    const newTabs = state.tabs.slice(tabIndex);
    set({ tabs: newTabs });
  },
  
  setError: (error) => {
    set({ error });
  },
  
  // Getters
  getActiveTab: () => {
    const state = get();
    return state.tabs.find(tab => tab.id === state.activeTabId) || null;
  },
  
  getTabById: (tabId) => {
    const state = get();
    return state.tabs.find(tab => tab.id === tabId) || null;
  },
  
  getTabByPath: (workspaceId, path) => {
    const state = get();
    const normalized = path.replace(/\\/g, '/');
    const matches = state.tabs.filter(
      (tab) =>
        tab.workspaceId === workspaceId &&
        (tab.path === normalized || tab.cadScadPath === normalized)
    );
    if (matches.length === 0) return null;
    return matches.find((t) => t.viewMode === 'scan-summary') ?? matches[0];
  },
  
  hasUnsavedChanges: () => {
    const state = get();
    return state.tabs.some(tab => tab.isDirty);
  },
}));
