import { createWithEqualityFn as create } from 'zustand/traditional';
import type { FileChange, FileChangeDiff } from '../types/protocol';
import { ChatAPI } from '../api/chatAPI';
import { useFileExplorerStore } from './fileExplorerStore';
import { useEditorStore } from './editorStore';
import { refreshFileExplorerForPaths } from '../utils/refreshFileExplorer';

interface FileChangeState {
  // State
  pendingChanges: FileChange[];
  changesById: Record<string, FileChange>;
  busyById: Record<string, boolean>;
  errorsById: Record<string, string>;
  loading: boolean;
  error: string | null;
  selectedChangeId: string | null;
  previewData: FileChangeDiff | null;
  
  // Actions
  fetchPendingChanges: (userId?: string) => Promise<void>;
  approveChange: (changeId: string, userId?: string, newContent?: string) => Promise<void>;
  updateChangeContent: (changeId: string, newContent: string) => Promise<void>;
  rejectChange: (changeId: string, reason?: string, userId?: string) => Promise<void>;
  getFileDiff: (changeId: string) => Promise<void>;
  selectChange: (changeId: string | null) => void;
  clearError: () => void;
  refreshChanges: () => Promise<void>;
}

export const useFileChangeStore = create<FileChangeState>((set, get) => ({
  // Initial state
  pendingChanges: [],
  changesById: {},
  busyById: {},
  errorsById: {},
  loading: false,
  error: null,
  selectedChangeId: null,
  previewData: null,

  // Fetch pending file changes
  fetchPendingChanges: async (userId = 'default') => {
    set({ loading: true, error: null });
    
    try {
      const api = new ChatAPI();
      const changes = await api.listPendingFileChanges(userId);
      const pendingChanges = Array.isArray(changes) ? changes : [];
      set((state) => ({
        pendingChanges,
        changesById: pendingChanges.reduce<Record<string, FileChange>>(
          (next, change) => ({ ...next, [change.id]: change }),
          { ...state.changesById },
        ),
        loading: false,
      }));
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : 'Failed to fetch file changes';
      set({ 
        error: errorMessage, 
        loading: false,
        pendingChanges: [] // Ensure we always have an array even on error
      });
    }
  },

  // Approve a file change
  approveChange: async (changeId: string, userId = 'default', newContent?: string) => {
    set((state) => ({
      busyById: { ...state.busyById, [changeId]: true },
      errorsById: { ...state.errorsById, [changeId]: '' },
      error: null,
    }));
    
    try {
      const api = new ChatAPI();
      const approvedChange = await api.approveFileChange(changeId, userId, newContent);
      
      // Remove the approved change from the list
      const state = get();
      const existingChange = state.pendingChanges.find(change => change.id === changeId);
      const updatedChanges = state.pendingChanges.filter(change => change.id !== changeId);
      set((current) => ({
        pendingChanges: updatedChanges,
        changesById: { ...current.changesById, [changeId]: approvedChange },
        busyById: { ...current.busyById, [changeId]: false },
      }));

      // Refresh the file explorer so newly created/edited files appear immediately.
      const change = existingChange ?? approvedChange;
      const filePath = change?.file_path || change?.new_path || change?.old_path;
      if (filePath) {
        const { workspaces } = useFileExplorerStore.getState();
        const matchedWorkspace = workspaces.find(
          (workspace) =>
            filePath === workspace.path || filePath.startsWith(`${workspace.path}/`)
        );
        if (matchedWorkspace) {
          const relPath = filePath.startsWith(`${matchedWorkspace.path}/`)
            ? filePath.slice(matchedWorkspace.path.length + 1)
            : filePath;
          await refreshFileExplorerForPaths(matchedWorkspace.id, [relPath]);
          if (relPath) {
            await useEditorStore.getState().refreshTabFromDisk(matchedWorkspace.id, relPath);
          }
        }
      }
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : 'Failed to approve file change';
      set((state) => ({
        error: errorMessage,
        busyById: { ...state.busyById, [changeId]: false },
        errorsById: { ...state.errorsById, [changeId]: errorMessage },
      }));
      throw error;
    }
  },

  updateChangeContent: async (changeId: string, newContent: string) => {
    try {
      const api = new ChatAPI();
      const diffData = await api.updateFileChangeContent(changeId, newContent);
      set({ previewData: diffData, loading: false });
      set((state) => ({
        pendingChanges: state.pendingChanges.map((c) =>
          c.id === changeId ? { ...c, new_content: newContent } : c
        ),
        changesById: {
          ...state.changesById,
          [changeId]: diffData.change,
        },
      }));
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : 'Failed to update file change';
      set({ error: errorMessage });
    }
  },

  // Reject a file change
  rejectChange: async (changeId: string, reason = 'No reason provided', userId = 'default') => {
    set((state) => ({
      busyById: { ...state.busyById, [changeId]: true },
      errorsById: { ...state.errorsById, [changeId]: '' },
      error: null,
    }));
    
    try {
      const api = new ChatAPI();
      const rejectedChange = await api.rejectFileChange(changeId, reason, userId);
      
      // Remove the rejected change from the list
      const state = get();
      const updatedChanges = state.pendingChanges.filter(change => change.id !== changeId);
      set((current) => ({
        pendingChanges: updatedChanges,
        changesById: { ...current.changesById, [changeId]: rejectedChange },
        busyById: { ...current.busyById, [changeId]: false },
      }));
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : 'Failed to reject file change';
      set((state) => ({
        error: errorMessage,
        busyById: { ...state.busyById, [changeId]: false },
        errorsById: { ...state.errorsById, [changeId]: errorMessage },
      }));
      throw error;
    }
  },

  // Get file diff for preview
  getFileDiff: async (changeId: string) => {
    set((state) => ({
      busyById: { ...state.busyById, [changeId]: true },
      errorsById: { ...state.errorsById, [changeId]: '' },
      error: null,
    }));
    
    try {
      const api = new ChatAPI();
      const diffData = await api.getFileDiff(changeId);
      set((state) => ({
        previewData: diffData,
        changesById: { ...state.changesById, [changeId]: diffData.change },
        busyById: { ...state.busyById, [changeId]: false },
      }));
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : 'Failed to get file diff';
      set((state) => ({
        error: errorMessage,
        busyById: { ...state.busyById, [changeId]: false },
        errorsById: { ...state.errorsById, [changeId]: errorMessage },
      }));
      throw error;
    }
  },

  // Select a change for preview
  selectChange: (changeId: string | null) => {
    set({ selectedChangeId: changeId });
    if (changeId) {
      get().getFileDiff(changeId);
    } else {
      set({ previewData: null });
    }
  },

  // Clear error
  clearError: () => {
    set({ error: null });
  },

  // Refresh changes
  refreshChanges: async () => {
    await get().fetchPendingChanges();
  },
}));
