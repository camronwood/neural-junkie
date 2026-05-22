import { createWithEqualityFn as create } from 'zustand/traditional';
import { ChatAPI } from '../api/chatAPI';
import { getHubBaseURL } from '../config/hubUrl';
import type { ScanSummaryData } from '../utils/scanSummary';
import { isScanSummaryWellPath, SCAN_SUMMARY_METADATA_FILE } from '../utils/scanSummary';

const api = new ChatAPI(getHubBaseURL());

export type EditorTabViewMode = 'text' | 'image' | 'scan-summary';

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
}

export interface OpenFileOptions {
  viewMode?: EditorTabViewMode;
  imageSrc?: string;
  scanSummaryDir?: string;
  scanSummaryData?: ScanSummaryData;
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
    initialWell?: string
  ) => void;
  closeTab: (tabId: string) => void;
  setActiveTab: (tabId: string) => void;
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
      language: viewMode === 'image' || viewMode === 'scan-summary' ? undefined : language,
      viewMode,
      imageSrc,
      scanSummaryDir: options?.scanSummaryDir,
      scanSummaryData: options?.scanSummaryData,
    };
    
    set({
      tabs: [...state.tabs, newTab],
      activeTabId: newTab.id,
    });
  },

  openScanSummary: (workspaceId, summaryDir, data, initialWell) => {
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
            ? { ...t, scanSummaryData: data, scanSummaryInitialWell: initialWell ?? t.scanSummaryInitialWell }
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
    };
    set({
      tabs: staleTextTab
        ? state.tabs.map((t) => (t.id === staleTextTab.id ? newTab : t))
        : [...state.tabs, newTab],
      activeTabId: tabId,
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
  
  updateTabContent: (tabId, content) => {
    set((state) => ({
      tabs: state.tabs.map((tab) => {
        if (tab.id !== tabId) return tab;
        if (tab.viewMode === 'image' || tab.viewMode === 'scan-summary') return tab;
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
    if (tab.viewMode === 'image' || tab.viewMode === 'scan-summary') return true;

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
      (tab) => tab.isDirty && tab.viewMode !== 'image' && tab.viewMode !== 'scan-summary'
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
    const matches = state.tabs.filter(
      (tab) => tab.workspaceId === workspaceId && tab.path === path
    );
    if (matches.length === 0) return null;
    return matches.find((t) => t.viewMode === 'scan-summary') ?? matches[0];
  },
  
  hasUnsavedChanges: () => {
    const state = get();
    return state.tabs.some(tab => tab.isDirty);
  },
}));
