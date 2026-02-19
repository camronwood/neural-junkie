import { create } from 'zustand';
import type { FileChange, FileChangeDiff } from '../types/protocol';
import { ChatAPI } from '../api/chatAPI';

interface FileChangeState {
  // State
  pendingChanges: FileChange[];
  loading: boolean;
  error: string | null;
  selectedChangeId: string | null;
  previewData: FileChangeDiff | null;
  
  // Actions
  fetchPendingChanges: (userId?: string) => Promise<void>;
  approveChange: (changeId: string, userId?: string) => Promise<void>;
  rejectChange: (changeId: string, reason?: string, userId?: string) => Promise<void>;
  getFileDiff: (changeId: string) => Promise<void>;
  selectChange: (changeId: string | null) => void;
  clearError: () => void;
  refreshChanges: () => Promise<void>;
}

export const useFileChangeStore = create<FileChangeState>((set, get) => ({
  // Initial state
  pendingChanges: [],
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
      // Ensure changes is always an array, never null
      set({ pendingChanges: Array.isArray(changes) ? changes : [], loading: false });
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
  approveChange: async (changeId: string, userId = 'default') => {
    set({ loading: true, error: null });
    
    try {
      const api = new ChatAPI();
      await api.approveFileChange(changeId, userId);
      
      // Remove the approved change from the list
      const state = get();
      const updatedChanges = state.pendingChanges.filter(change => change.id !== changeId);
      set({ pendingChanges: updatedChanges, loading: false });
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : 'Failed to approve file change';
      set({ error: errorMessage, loading: false });
    }
  },

  // Reject a file change
  rejectChange: async (changeId: string, reason = 'No reason provided', userId = 'default') => {
    set({ loading: true, error: null });
    
    try {
      const api = new ChatAPI();
      await api.rejectFileChange(changeId, reason, userId);
      
      // Remove the rejected change from the list
      const state = get();
      const updatedChanges = state.pendingChanges.filter(change => change.id !== changeId);
      set({ pendingChanges: updatedChanges, loading: false });
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : 'Failed to reject file change';
      set({ error: errorMessage, loading: false });
    }
  },

  // Get file diff for preview
  getFileDiff: async (changeId: string) => {
    set({ loading: true, error: null });
    
    try {
      const api = new ChatAPI();
      const diffData = await api.getFileDiff(changeId);
      set({ previewData: diffData, loading: false });
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : 'Failed to get file diff';
      set({ error: errorMessage, loading: false });
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
