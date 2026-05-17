import { useState, useEffect, useRef } from 'react';
import { useFileExplorerStore } from '../stores/fileExplorerStore';
import { useEditorStore } from '../stores/editorStore';
import { useToastStore } from '../stores/toastStore';
import { ChatAPI } from '../api/chatAPI';
import { getHubBaseURL } from '../config/hubUrl';
import type { FileNode } from '../stores/fileExplorerStore';
import { invoke } from '@tauri-apps/api/tauri';
import { open } from '@tauri-apps/api/dialog';
import { isImagePreviewPath, workspaceAbsolutePath } from '../utils/editorFileKind';
import { resolveEditorImageSrc } from '../utils/chatImageSrc';
import { ViewportContextMenu } from './ViewportContextMenu';

interface FileExplorerPanelProps {
  onClose: () => void;
  onFileOpen?: () => void;
}

const MIN_WIDTH = 200; // Minimum usable width
const DEFAULT_WIDTH = 300;
const STORAGE_KEY = 'file-explorer-panel-width';

export function FileExplorerPanel({ onClose, onFileOpen }: FileExplorerPanelProps) {
  const {
    workspaces,
    activeWorkspaceId,
    fileTree,
    expandedPaths,
    selectedPath,
    loadingFiles,
    error,
    loadWorkspaces,
    addWorkspace,
    setActiveWorkspace,
    loadFiles,
    toggleExpanded,
    setSelectedPath,
    createFile,
    createFolder,
    renameFile,
    deleteFile,
    removeWorkspace,
    getActiveWorkspace,
    setError,
    clearError,
  } = useFileExplorerStore();

  const { openFile } = useEditorStore();
  const { addToast } = useToastStore();

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

  // State for adding new workspace
  const [showAddWorkspace, setShowAddWorkspace] = useState(false);
  const [newWorkspaceName, setNewWorkspaceName] = useState('');
  const [newWorkspacePath, setNewWorkspacePath] = useState('');

  // State for file operations
  const [contextMenu, setContextMenu] = useState<{
    x: number;
    y: number;
    path: string;
    isDir: boolean;
  } | null>(null);

  const [api] = useState(() => new ChatAPI(getHubBaseURL()));

  // Load workspaces on mount
  useEffect(() => {
    console.log('FileExplorerPanel: Loading workspaces...');
    loadWorkspaces();
  }, [loadWorkspaces]);

  // Load files when workspace changes
  useEffect(() => {
    if (activeWorkspaceId) {
      console.log('FileExplorerPanel: Loading files for workspace:', activeWorkspaceId);
      loadFiles(activeWorkspaceId);
    }
  }, [activeWorkspaceId, loadFiles]);

  // Resize handlers
  useEffect(() => {
    const handleMouseMove = (e: MouseEvent) => {
      if (!isResizing) return;
      
      const delta = e.clientX - resizeStartX.current;
      const newWidth = resizeStartWidth.current + delta;
      // Allow free resizing, but limit to reasonable maximum
      // File explorer should not take more than 40% of screen
      const maxWidth = Math.min(window.innerWidth * 0.4, 600); // Max 40% of screen or 600px
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

  const handleBrowseDirectory = async () => {
    try {
      const selected = await open({
        directory: true,
        multiple: false,
        title: 'Select Workspace Directory',
      });
      
      if (selected && typeof selected === 'string') {
        setNewWorkspacePath(selected);
        // Auto-populate name from directory if empty
        if (!newWorkspaceName) {
          const dirName = selected.split('/').pop() || '';
          setNewWorkspaceName(dirName);
        }
      }
    } catch (error) {
      console.error('Failed to open directory picker:', error);
    }
  };

  const handleAddWorkspace = async () => {
    if (!newWorkspaceName || !newWorkspacePath) return;
    
    try {
      await addWorkspace(newWorkspaceName, newWorkspacePath);
      setShowAddWorkspace(false);
      setNewWorkspaceName('');
      setNewWorkspacePath('');
    } catch (error) {
      console.error('Failed to add workspace:', error);
    }
  };

  const [pendingRemove, setPendingRemove] = useState<{ id: string; name: string } | null>(null);

  const handleRemoveWorkspace = (e: React.MouseEvent, workspaceId: string, workspaceName: string) => {
    e.stopPropagation();
    e.preventDefault();
    setPendingRemove({ id: workspaceId, name: workspaceName });
  };

  const confirmRemoveWorkspace = async () => {
    if (!pendingRemove) return;
    const { id, name } = pendingRemove;
    setPendingRemove(null);
    try {
      await removeWorkspace(id);
      addToast({ type: 'success', title: 'Workspace removed', message: `"${name}" removed from file explorer` });
    } catch (error) {
      console.error('Failed to remove workspace:', error);
      addToast({ type: 'error', title: 'Remove failed', message: error instanceof Error ? error.message : 'Failed to remove workspace' });
    }
  };

  const handleFileClick = async (file: FileNode) => {
    // Add null check for file.path to prevent crashes
    if (!file.path) {
      console.error('File path is undefined:', file);
      setError('File path is undefined');
      return;
    }
    
    if (file.is_dir) {
      console.log('Toggling directory:', file.path, 'current expanded:', !!expandedPaths[file.path]);
      const wasExpanded = !!expandedPaths[file.path];
      toggleExpanded(file.path);
      
      // If we're expanding the directory and it doesn't have children loaded, load them
      if (!wasExpanded && (!file.children || file.children.length === 0)) {
        const activeWorkspace = getActiveWorkspace();
        if (activeWorkspace) {
          try {
            console.log('Loading directory contents for:', file.path);
            await loadFiles(activeWorkspace.id, file.path);
          } catch (error) {
            console.error('Failed to load directory contents:', error);
            setError(error instanceof Error ? error.message : 'Failed to load directory contents');
          }
        }
      }
      
      setSelectedPath(file.path);
    } else {
      // Open file in editor
      const activeWorkspace = getActiveWorkspace();
      if (activeWorkspace) {
        try {
          console.log('Opening file:', file.path, 'in workspace:', activeWorkspace.id);
          if (isImagePreviewPath(file.path)) {
            const absolutePath = workspaceAbsolutePath(activeWorkspace.path, file.path);
            const imageSrc = await resolveEditorImageSrc({
              workspaceId: activeWorkspace.id,
              relativePath: file.path,
              absolutePath,
            });
            openFile(activeWorkspace.id, file.path, '', undefined, {
              viewMode: 'image',
              imageSrc,
            });
          } else {
            const content = await api.fetchFileContent(activeWorkspace.id, file.path);
            const language = getLanguageFromPath(file.path);
            console.log('File content loaded, opening in editor...');
            openFile(activeWorkspace.id, file.path, content, language);
          }
          // Auto-open the editor panel when a file is opened
          if (onFileOpen) {
            onFileOpen();
          }
        } catch (error) {
          console.error('Failed to open file:', error);
          setError(error instanceof Error ? error.message : 'Failed to open file');
        }
      }
      setSelectedPath(file.path);
    }
  };

  const handleContextMenu = (e: React.MouseEvent, file: FileNode) => {
    e.preventDefault();
    setContextMenu({
      x: e.clientX,
      y: e.clientY,
      path: file.path,
      isDir: file.is_dir,
    });
  };

  const closeContextMenu = () => {
    setContextMenu(null);
  };

  const handleCreateFile = async () => {
    if (!contextMenu || !activeWorkspaceId) return;
    
    const fileName = prompt('Enter file name:');
    if (!fileName) return;
    
    const newPath = contextMenu.isDir 
      ? `${contextMenu.path}/${fileName}`
      : `${contextMenu.path.substring(0, contextMenu.path.lastIndexOf('/'))}/${fileName}`;
    
    try {
      await createFile(activeWorkspaceId, newPath);
      closeContextMenu();
    } catch (error) {
      console.error('Failed to create file:', error);
    }
  };

  const handleCreateFolder = async () => {
    if (!contextMenu || !activeWorkspaceId) return;
    
    const folderName = prompt('Enter folder name:');
    if (!folderName) return;
    
    const newPath = contextMenu.isDir 
      ? `${contextMenu.path}/${folderName}`
      : `${contextMenu.path.substring(0, contextMenu.path.lastIndexOf('/'))}/${folderName}`;
    
    try {
      await createFolder(activeWorkspaceId, newPath);
      closeContextMenu();
    } catch (error) {
      console.error('Failed to create folder:', error);
    }
  };

  const handleRename = async () => {
    if (!contextMenu || !activeWorkspaceId) return;
    
    const newName = prompt('Enter new name:', contextMenu.path.split('/').pop() || '');
    if (!newName) return;
    
    const newPath = contextMenu.path.substring(0, contextMenu.path.lastIndexOf('/')) + '/' + newName;
    
    try {
      await renameFile(activeWorkspaceId, contextMenu.path, newPath);
      closeContextMenu();
    } catch (error) {
      console.error('Failed to rename:', error);
    }
  };

  const handleDelete = async () => {
    if (!contextMenu || !activeWorkspaceId) return;
    
    if (window.confirm(`Delete ${contextMenu.isDir ? 'folder' : 'file'}?`)) {
      try {
        await deleteFile(activeWorkspaceId, contextMenu.path);
        closeContextMenu();
      } catch (error) {
        console.error('Failed to delete:', error);
      }
    }
  };

  const handlePreviewMarkdown = async () => {
    if (!contextMenu || !activeWorkspaceId) return;
    
    try {
      await invoke('open_markdown_preview', {
        workspaceId: activeWorkspaceId,
        filePath: contextMenu.path,
      });
      closeContextMenu();
    } catch (error) {
      console.error('Failed to open markdown preview:', error);
      setError(error instanceof Error ? error.message : 'Failed to open preview');
    }
  };

  const handleCopyPath = async () => {
    if (!contextMenu) return;
    
    const activeWorkspace = getActiveWorkspace();
    if (!activeWorkspace) {
      addToast({
        type: 'error',
        title: 'No workspace',
        message: 'Please select a workspace first.',
      });
      return;
    }
    
    try {
      const absolutePath = workspaceAbsolutePath(activeWorkspace.path, contextMenu.path);

      await navigator.clipboard.writeText(absolutePath);
      addToast({
        type: 'success',
        title: 'Path copied',
        message: 'Path copied to clipboard',
      });
      closeContextMenu();
    } catch (error) {
      console.error('Failed to copy path:', error);
      addToast({
        type: 'error',
        title: 'Copy failed',
        message: 'Failed to copy path to clipboard',
      });
    }
  };

  const handleCopyRelativePath = async () => {
    if (!contextMenu) return;
    
    try {
      await navigator.clipboard.writeText(contextMenu.path);
      addToast({
        type: 'success',
        title: 'Relative path copied',
        message: 'Relative path copied to clipboard',
      });
      closeContextMenu();
    } catch (error) {
      console.error('Failed to copy relative path:', error);
      addToast({
        type: 'error',
        title: 'Copy failed',
        message: 'Failed to copy relative path to clipboard',
      });
    }
  };

  const getLanguageFromPath = (path: string): string => {
    if (!path) {
      return 'plaintext';
    }
    
    const ext = path.split('.').pop()?.toLowerCase();
    const languageMap: Record<string, string> = {
      'js': 'javascript',
      'jsx': 'javascript',
      'ts': 'typescript',
      'tsx': 'typescript',
      'py': 'python',
      'go': 'go',
      'rs': 'rust',
      'java': 'java',
      'cpp': 'cpp',
      'c': 'c',
      'cs': 'csharp',
      'php': 'php',
      'rb': 'ruby',
      'swift': 'swift',
      'kt': 'kotlin',
      'scala': 'scala',
      'html': 'html',
      'css': 'css',
      'scss': 'scss',
      'sass': 'sass',
      'less': 'less',
      'json': 'json',
      'xml': 'xml',
      'yaml': 'yaml',
      'yml': 'yaml',
      'md': 'markdown',
      'sql': 'sql',
      'sh': 'shell',
      'bash': 'shell',
      'zsh': 'shell',
      'fish': 'shell',
    };
    return languageMap[ext || ''] || 'plaintext';
  };

  const renderFileIcon = (file: FileNode) => {
    if (file.is_dir) {
      return expandedPaths[file.path] ? '📂' : '📁';
    }
    
    // Add null check for file.path to prevent crashes
    if (!file.path) {
      return '📄';
    }
    
    const ext = file.path.split('.').pop()?.toLowerCase();
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

  const renderFileTree = (files: FileNode[], level = 0) => {
    return files.map((file) => (
      <div key={file.path}>
        <div
          className={`flex items-center gap-2 py-1 px-2 cursor-pointer hover:bg-slack-bgHover rounded ${
            selectedPath === file.path ? 'bg-slack-accent text-white' : 'text-slack-text'
          }`}
          style={{ paddingLeft: `${level * 16 + 8}px` }}
          onClick={() => handleFileClick(file)}
          onContextMenu={(e) => handleContextMenu(e, file)}
        >
          <span className="text-sm">
            {file.is_dir ? (expandedPaths[file.path] ? '📂' : '📁') : renderFileIcon(file)}
          </span>
          <span className="text-sm truncate flex-1">{file.name}</span>
          {file.is_dir && (
            <span className="text-xs text-slack-textMuted">
              {expandedPaths[file.path] ? '▼' : '▶'}
            </span>
          )}
        </div>
        {file.is_dir && expandedPaths[file.path] && file.children && (
          <div>
            {renderFileTree(file.children, level + 1)}
          </div>
        )}
      </div>
    ));
  };

  const files = activeWorkspaceId ? (fileTree[activeWorkspaceId] || []) : [];

  return (
    <div 
      className="border-r border-slack-border bg-slack-bg flex flex-col h-full relative animate-slide-in-left flex-shrink-0"
      style={{ width: `${width}px`, minWidth: `${MIN_WIDTH}px` }}
    >
        {/* Resize Handle */}
        <div
          className="absolute right-0 top-0 bottom-0 cursor-col-resize z-[100] group"
          onMouseDown={handleResizeStart}
          aria-label="Resize file explorer panel"
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
        <h2 className="font-bold text-slack-text">Files</h2>
        <div className="flex items-center gap-2">
          <button
            onClick={() => setShowAddWorkspace(true)}
            className="text-slack-textMuted hover:text-slack-text transition-colors"
            title="Add workspace"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
            </svg>
          </button>
          <button
            onClick={onClose}
            className="text-slack-textMuted hover:text-slack-text transition-colors"
            title="Close file explorer"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
      </div>

      {/* Workspace Tabs */}
      <div className="px-4 py-2 border-b border-slack-border bg-slack-bgHover">
        <div className="flex gap-1 overflow-x-auto">
          {workspaces.map((workspace) => (
            <div
              key={workspace.id}
              onClick={() => setActiveWorkspace(workspace.id)}
              className={`group flex items-center gap-1 px-3 py-1 text-xs rounded transition-colors whitespace-nowrap cursor-pointer ${
                activeWorkspaceId === workspace.id
                  ? 'bg-slack-accent text-white'
                  : 'bg-slack-bgHover text-slack-textMuted hover:text-slack-text'
              }`}
              title={workspace.path}
            >
              <span>{workspace.name}</span>
              <button
                onClick={(e) => handleRemoveWorkspace(e, workspace.id, workspace.name)}
                className={`ml-1 p-0.5 rounded-sm opacity-0 group-hover:opacity-100 transition-opacity ${
                  activeWorkspaceId === workspace.id
                    ? 'hover:bg-white/20'
                    : 'hover:bg-slack-border'
                }`}
                title={`Remove ${workspace.name}`}
              >
                <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>
          ))}
        </div>
      </div>

      {/* File Tree */}
      <div className="flex-1 overflow-y-auto">
        {error ? (
          <div className="p-4 text-center">
            <div className="text-4xl mb-2">⚠️</div>
            <div className="text-sm text-red-500 mb-2">{error}</div>
            <button
              onClick={clearError}
              className="px-3 py-1 bg-red-600 hover:bg-red-700 text-white text-xs rounded transition-colors"
            >
              Dismiss
            </button>
          </div>
        ) : loadingFiles ? (
          <div className="flex items-center justify-center h-32">
            <div className="flex items-center gap-2 text-slack-textMuted">
              <div className="w-4 h-4 border border-slack-textMuted border-t-transparent rounded-full animate-spin"></div>
              Loading files...
            </div>
          </div>
        ) : files.length === 0 ? (
          <div className="p-4 text-center">
            <div className="text-4xl mb-2">📁</div>
            <div className="text-sm text-slack-textMuted">No files found</div>
          </div>
        ) : (
          <div className="py-2">
            {renderFileTree(files)}
          </div>
        )}
      </div>

      {/* Add Workspace Modal */}
      {showAddWorkspace && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50">
          <div className="bg-slack-bg border border-slack-border rounded p-6 w-96">
            <h3 className="text-lg font-bold text-slack-text mb-4">Add Workspace</h3>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-slack-text mb-1">
                  Name
                </label>
                <input
                  type="text"
                  value={newWorkspaceName}
                  onChange={(e) => setNewWorkspaceName(e.target.value)}
                  className="w-full px-3 py-2 bg-slack-bg border border-slack-border rounded text-slack-text focus:outline-none focus:border-slack-accent"
                  placeholder="Workspace name"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-slack-text mb-1">
                  Path
                </label>
                <div className="flex gap-2">
                  <input
                    type="text"
                    value={newWorkspacePath}
                    onChange={(e) => setNewWorkspacePath(e.target.value)}
                    className="flex-1 px-3 py-2 bg-slack-bg border border-slack-border rounded text-slack-text focus:outline-none focus:border-slack-accent"
                    placeholder="/path/to/workspace"
                  />
                  <button
                    onClick={handleBrowseDirectory}
                    className="px-3 py-2 bg-slack-bgHover hover:bg-slack-accent text-slack-text hover:text-white rounded transition-colors"
                    title="Browse for directory"
                  >
                    📁 Browse
                  </button>
                </div>
              </div>
            </div>
            <div className="flex gap-2 mt-6">
              <button
                onClick={handleAddWorkspace}
                disabled={!newWorkspaceName || !newWorkspacePath}
                className="px-4 py-2 bg-slack-accent hover:bg-slack-accentHover text-white text-sm rounded transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              >
                Add
              </button>
              <button
                onClick={() => setShowAddWorkspace(false)}
                className="px-4 py-2 bg-slack-bgHover text-slack-text text-sm rounded transition-colors"
              >
                Cancel
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Context menu — portaled so panel transform/overflow does not clip it */}
      {contextMenu && (
        <ViewportContextMenu
          x={contextMenu.x}
          y={contextMenu.y}
          onClose={closeContextMenu}
        >
          {/* Show Preview Markdown option for .md files */}
          {!contextMenu.isDir && contextMenu.path.toLowerCase().endsWith('.md') && (
            <button
              onClick={handlePreviewMarkdown}
              className="w-full px-4 py-2 text-left text-sm text-slack-text hover:bg-slack-bgHover"
            >
              📝 Preview Markdown
            </button>
          )}
          
          {/* Copy Path options */}
          <button
            onClick={handleCopyPath}
            className="w-full px-4 py-2 text-left text-sm text-slack-text hover:bg-slack-bgHover"
          >
            📋 Copy Path
          </button>
          <button
            onClick={handleCopyRelativePath}
            className="w-full px-4 py-2 text-left text-sm text-slack-text hover:bg-slack-bgHover"
          >
            📋 Copy Relative Path
          </button>
          
          {/* Separator before file operations */}
          <div className="border-t border-slack-border my-1" />
          
          <button
            onClick={handleCreateFile}
            className="w-full px-4 py-2 text-left text-sm text-slack-text hover:bg-slack-bgHover"
          >
            New File
          </button>
          <button
            onClick={handleCreateFolder}
            className="w-full px-4 py-2 text-left text-sm text-slack-text hover:bg-slack-bgHover"
          >
            New Folder
          </button>
          <div className="border-t border-slack-border my-1" />
          <button
            onClick={handleRename}
            className="w-full px-4 py-2 text-left text-sm text-slack-text hover:bg-slack-bgHover"
          >
            Rename
          </button>
          <button
            onClick={handleDelete}
            className="w-full px-4 py-2 text-left text-sm text-red-500 hover:bg-slack-bgHover"
          >
            Delete
          </button>
        </ViewportContextMenu>
      )}

      {/* Remove workspace confirmation */}
      {pendingRemove && (
        <>
          <div className="fixed inset-0 z-50 bg-black/50" onClick={() => setPendingRemove(null)} />
          <div className="fixed z-50 top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 bg-slack-bg border border-slack-border rounded-lg shadow-xl p-5 min-w-[300px]">
            <h3 className="text-sm font-semibold text-slack-text mb-2">Remove Workspace</h3>
            <p className="text-xs text-slack-textMuted mb-4">
              Remove <span className="font-semibold text-slack-text">"{pendingRemove.name}"</span> from
              the file explorer? No files will be deleted.
            </p>
            <div className="flex justify-end gap-2">
              <button
                onClick={() => setPendingRemove(null)}
                className="px-3 py-1.5 text-xs rounded bg-slack-bgHover text-slack-text hover:bg-slack-border transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={confirmRemoveWorkspace}
                className="px-3 py-1.5 text-xs rounded bg-red-600 text-white hover:bg-red-700 transition-colors"
              >
                Remove
              </button>
            </div>
          </div>
        </>
      )}
    </div>
  );
}
