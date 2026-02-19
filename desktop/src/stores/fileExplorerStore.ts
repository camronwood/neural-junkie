import { create } from 'zustand';
import { ChatAPI } from '../api/chatAPI';

const api = new ChatAPI('localhost:8080');

export interface Workspace {
  id: string;
  name: string;
  path: string;
  created_at: string;
  last_used: string;
  is_git_repo: boolean;
  git_remote?: string;
  git_branch?: string;
}

export interface FileNode {
  name: string;
  is_dir: boolean;
  size: number;
  mod_time: string;
  children?: FileNode[];
  expanded?: boolean;
  path: string; // Full path from workspace root
}

interface FileExplorerState {
  // Workspaces
  workspaces: Workspace[];
  activeWorkspaceId: string | null;
  
  // File tree
  fileTree: Map<string, FileNode[]>; // workspaceId -> file nodes
  expandedPaths: Set<string>; // Set of expanded folder paths
  selectedPath: string | null;
  
  // Loading and error states
  loadingWorkspaces: boolean;
  loadingFiles: boolean;
  error: string | null;
  
  // Actions
  loadWorkspaces: () => Promise<void>;
  addWorkspace: (name: string, path: string) => Promise<Workspace>;
  removeWorkspace: (workspaceId: string) => Promise<void>;
  setActiveWorkspace: (workspaceId: string) => void;
  
  loadFiles: (workspaceId: string, path?: string) => Promise<void>;
  toggleExpanded: (path: string) => void;
  setSelectedPath: (path: string | null) => void;
  
  // File operations
  createFile: (workspaceId: string, path: string, content?: string) => Promise<void>;
  createFolder: (workspaceId: string, path: string) => Promise<void>;
  renameFile: (workspaceId: string, oldPath: string, newPath: string) => Promise<void>;
  deleteFile: (workspaceId: string, path: string) => Promise<void>;
  
  // Error handling
  setError: (error: string | null) => void;
  clearError: () => void;
  
  // Panel state
  setFileExplorerOpen: (open: boolean) => void;
  
  // Getters
  getActiveWorkspace: () => Workspace | null;
  getFileTree: (workspaceId: string) => FileNode[];
  isPathExpanded: (path: string) => boolean;
  getFileByPath: (workspaceId: string, path: string) => FileNode | null;
}

export const useFileExplorerStore = create<FileExplorerState>((set, get) => ({
  workspaces: [],
  activeWorkspaceId: null,
  fileTree: new Map(),
  expandedPaths: new Set(),
  selectedPath: null,
  loadingWorkspaces: false,
  loadingFiles: false,
  error: null,
  
  loadWorkspaces: async () => {
    console.log('FileExplorerStore: Loading workspaces...');
    set({ loadingWorkspaces: true, error: null });
    try {
      const workspaces = await api.fetchWorkspaces();
      console.log('FileExplorerStore: Loaded workspaces:', workspaces);
      set({ 
        workspaces,
        activeWorkspaceId: workspaces[0]?.id || null,
        loadingWorkspaces: false,
      });
    } catch (error) {
      console.error('Failed to load workspaces:', error);
      set({ 
        loadingWorkspaces: false,
        error: error instanceof Error ? error.message : 'Failed to load workspaces'
      });
    }
  },
  
  addWorkspace: async (name, path) => {
    try {
      const workspace = await api.addWorkspace(name, path);
      set(state => ({
        workspaces: [...state.workspaces, workspace],
        activeWorkspaceId: workspace.id,
      }));
      return workspace;
    } catch (error) {
      console.error('Failed to add workspace:', error);
      set({ error: error instanceof Error ? error.message : 'Failed to add workspace' });
      throw error;
    }
  },
  
  removeWorkspace: async (workspaceId) => {
    try {
      // TODO: Implement actual API call
      console.log('Removing workspace:', workspaceId);
      
      set(state => {
        const newWorkspaces = state.workspaces.filter(w => w.id !== workspaceId);
        const newActiveWorkspaceId = state.activeWorkspaceId === workspaceId 
          ? (newWorkspaces[0]?.id || null)
          : state.activeWorkspaceId;
        
        const newFileTree = new Map(state.fileTree);
        newFileTree.delete(workspaceId);
        
        return {
          workspaces: newWorkspaces,
          activeWorkspaceId: newActiveWorkspaceId,
          fileTree: newFileTree,
        };
      });
    } catch (error) {
      console.error('Failed to remove workspace:', error);
      throw error;
    }
  },
  
  setActiveWorkspace: (workspaceId) => {
    set({ activeWorkspaceId: workspaceId });
  },
  
  loadFiles: async (workspaceId, path = '/') => {
    console.log('FileExplorerStore: Loading files for workspace:', workspaceId, 'path:', path);
    set({ loadingFiles: true, error: null });
    try {
      const files = await api.fetchFiles(workspaceId, path);
      console.log('FileExplorerStore: Loaded files:', files);
      set(state => {
        const newFileTree = new Map(state.fileTree);
        
        if (path === '/') {
          // For root path, replace the entire tree
          newFileTree.set(workspaceId, files);
        } else {
          // For subdirectories, we need to build a hierarchical structure
          // For now, let's store files by path and build the tree dynamically
          const currentFiles = newFileTree.get(workspaceId) || [];
          
          // Find the parent directory and add children
          const updateFileTree = (fileList: FileNode[], targetPath: string, newFiles: FileNode[]): FileNode[] => {
            return fileList.map(file => {
              if (file.path === targetPath && file.is_dir) {
                return { ...file, children: newFiles };
              } else if (file.children) {
                return { ...file, children: updateFileTree(file.children, targetPath, newFiles) };
              }
              return file;
            });
          };
          
          const updatedFiles = updateFileTree(currentFiles, path, files);
          newFileTree.set(workspaceId, updatedFiles);
        }
        
        return {
          fileTree: newFileTree,
          loadingFiles: false,
        };
      });
    } catch (error) {
      console.error('Failed to load files:', error);
      set({ 
        loadingFiles: false,
        error: error instanceof Error ? error.message : 'Failed to load files'
      });
    }
  },
  
  toggleExpanded: (path) => {
    set(state => {
      const newExpandedPaths = new Set(state.expandedPaths);
      if (newExpandedPaths.has(path)) {
        newExpandedPaths.delete(path);
      } else {
        newExpandedPaths.add(path);
      }
      return { expandedPaths: newExpandedPaths };
    });
  },
  
  setSelectedPath: (path) => {
    set({ selectedPath: path });
  },
  
  createFile: async (workspaceId, path, content = '') => {
    try {
      await api.createFile(workspaceId, path, content);
      // Refresh file tree after creation
      await get().loadFiles(workspaceId);
    } catch (error) {
      console.error('Failed to create file:', error);
      set({ error: error instanceof Error ? error.message : 'Failed to create file' });
      throw error;
    }
  },
  
  createFolder: async (workspaceId, path) => {
    try {
      await api.createFile(workspaceId, path, ''); // Create empty file as folder
      // Refresh file tree after creation
      await get().loadFiles(workspaceId);
    } catch (error) {
      console.error('Failed to create folder:', error);
      set({ error: error instanceof Error ? error.message : 'Failed to create folder' });
      throw error;
    }
  },
  
  renameFile: async (workspaceId, oldPath, newPath) => {
    try {
      await api.renameFile(workspaceId, oldPath, newPath);
      // Refresh file tree after rename
      await get().loadFiles(workspaceId);
    } catch (error) {
      console.error('Failed to rename file:', error);
      set({ error: error instanceof Error ? error.message : 'Failed to rename file' });
      throw error;
    }
  },
  
  deleteFile: async (workspaceId, path) => {
    try {
      await api.deleteFile(workspaceId, path);
      // Refresh file tree after deletion
      await get().loadFiles(workspaceId);
    } catch (error) {
      console.error('Failed to delete file:', error);
      set({ error: error instanceof Error ? error.message : 'Failed to delete file' });
      throw error;
    }
  },
  
  // Getters
  getActiveWorkspace: () => {
    const state = get();
    return state.workspaces.find(w => w.id === state.activeWorkspaceId) || null;
  },
  
  getFileTree: (workspaceId) => {
    const state = get();
    return state.fileTree.get(workspaceId) || [];
  },
  
  isPathExpanded: (path) => {
    const state = get();
    return state.expandedPaths.has(path);
  },
  
  getFileByPath: (workspaceId, path) => {
    const state = get();
    const files = state.fileTree.get(workspaceId) || [];
    
    // Simple recursive search
    const findFile = (nodes: FileNode[], targetPath: string): FileNode | null => {
      for (const node of nodes) {
        if (node.path === targetPath) {
          return node;
        }
        if (node.children) {
          const found = findFile(node.children, targetPath);
          if (found) return found;
        }
      }
      return null;
    };
    
    return findFile(files, path);
  },
  
  setError: (error) => {
    set({ error });
  },
  
  clearError: () => {
    set({ error: null });
  },
  
  setFileExplorerOpen: (open) => {
    // This is handled by the parent component, but we need it for the interface
    console.log('File explorer open:', open);
  },
}));
