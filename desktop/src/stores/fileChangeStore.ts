import { create } from 'zustand';
import type { FileChange, FileChangeDiff } from '../types/protocol';
import { ChatAPI } from '../api/chatAPI';
import { useFileExplorerStore } from './fileExplorerStore';
import { useEditorStore } from './editorStore';

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
      const approvedChange = await api.approveFileChange(changeId, userId);
      
      // Remove the approved change from the list
      const state = get();
      const existingChange = state.pendingChanges.find(change => change.id === changeId);
      const updatedChanges = state.pendingChanges.filter(change => change.id !== changeId);
      set({ pendingChanges: updatedChanges, loading: false });

      // Refresh the file explorer so newly created/edited files appear immediately.
      const change = existingChange ?? approvedChange;
      const filePath = change?.file_path || change?.new_path || change?.old_path;
      if (filePath) {
        const { workspaces, loadFiles } = useFileExplorerStore.getState();
        const matchedWorkspace = workspaces.find(workspace =>
          filePath === workspace.path || filePath.startsWith(`${workspace.path}/`)
        );
        if (matchedWorkspace) {
          const relPath = filePath.startsWith(`${matchedWorkspace.path}/`)
            ? filePath.slice(matchedWorkspace.path.length + 1)
            : '';
          const lastSlash = relPath.lastIndexOf('/');
          const parentPath = lastSlash > -1 ? relPath.slice(0, lastSlash) : '/';

          // Refresh root and parent directory so nested file trees update quickly.
          await loadFiles(matchedWorkspace.id, '/');
          if (parentPath && parentPath !== '/') {
            await loadFiles(matchedWorkspace.id, parentPath);
          }

          // Reload the open editor buffer for the affected file (if open and not dirty).
          if (relPath) {
            await useEditorStore.getState().refreshTabFromDisk(matchedWorkspace.id, relPath);
          }
        }
      }
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
