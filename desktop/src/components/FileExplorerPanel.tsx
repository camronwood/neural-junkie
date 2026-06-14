import { useState, useEffect, useMemo, useRef } from 'react';
import { useFileExplorerStore } from '../stores/fileExplorerStore';
import { useEditorStore } from '../stores/editorStore';
import { usePacksStore } from '../stores/packsStore';
import { useToastStore } from '../stores/toastStore';
import { ChatAPI } from '../api/chatAPI';
import { getHubBaseURL } from '../config/hubUrl';
import type { FileNode } from '../stores/fileExplorerStore';
import { invoke } from '@tauri-apps/api/tauri';
import { open } from '@tauri-apps/api/dialog';
import { isImagePreviewPath, workspaceAbsolutePath } from '../utils/editorFileKind';
import {
  dispatchWorkspaceFileDropEventAtPoint,
  scheduleWorkspaceFileDragClear,
  setWorkspaceFileDragData,
} from '../utils/workspaceFileDrag';
import { resolveEditorImageSrc } from '../utils/chatImageSrc';
import {
  isScanSummaryDirListing,
  isScanSummaryMetadataPath,
  isScanSummaryWellPath,
  isScanSummaryWorkspaceRoot,
  parseScanSummaryMetadata,
  scanSummaryDirForFilePath,
  scanSummaryDirFromMetadataPath,
  SCAN_SUMMARY_METADATA_FILE,
} from '../utils/scanSummary';
import {
  isCombinedRunDirListing,
  isScanAnalysisDirListing,
  isScanAnalysisResultsPath,
  isScanAnalysisRootListing,
  isScanAnalysisSummaryCSVPath,
  SCAN_ANALYSIS_REPORTS_DIR,
} from '../utils/scanAnalysis';
import { analyteFromSummaryCsvPath } from '../utils/scanAnalysisCsv';
import { loadScanAnalysisData, analysisDirFromFilePath } from '../utils/scanAnalysisLoad';
import { isComparatorAnalysisPath } from '../utils/secondaryAnalysis';
import { isEditableCsvPath } from '../utils/csvTable';
import { useSecondaryAnalysisStore } from '../stores/secondaryAnalysisStore';
import { PACK_CAP } from '../stores/packCapabilities';
import { shrinkablePanelStyle } from '../utils/panelLayout';
import { workspacesForTabBar } from '../utils/workspaceOrder';
import { ViewportContextMenu } from './ViewportContextMenu';
import { WorkspaceSwitcherModal } from './WorkspaceSwitcherModal';
import { useShortcutOverlay } from '../shortcuts/useShortcutOverlay';
import { WorkspaceTabBar } from './WorkspaceTabBar';
import { devLog } from '../utils/devLog';
import { qcReportRelativePath } from '../utils/panelQcUtils';

interface FileExplorerPanelProps {
  onClose: () => void;
  onFileOpen?: () => void;
  variant?: 'overlay' | 'embedded';
}

const MIN_WIDTH = 200; // Minimum usable width
const COMPACT_MIN_WIDTH = 160;
const DEFAULT_WIDTH = 300;
const STORAGE_KEY = 'file-explorer-panel-width';

export function FileExplorerPanel({ onClose, onFileOpen, variant = 'overlay' }: FileExplorerPanelProps) {
  const embedded = variant === 'embedded';
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
    refreshTreeForPath,
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

  const { openFile, openScanSummary, openScanAnalysis, openCadWorkbench, openComparatorAnalysis, setPanelQCReport } =
    useEditorStore();
  const hasScanSummary = usePacksStore((s) => s.hasCapability('scan-summary-viewer'));
  const hasScanAnalysis = usePacksStore((s) => s.hasCapability('scan-analysis-viewer'));
  const hasSecondaryAnalysis = usePacksStore((s) => s.hasCapability(PACK_CAP.SECONDARY_ANALYSIS_VIEWER));
  const hasCadWorkbench = usePacksStore((s) => s.hasCapability('cad-workbench'));
  const addToBasket = useSecondaryAnalysisStore((s) => s.addToBasket);
  const setPanelOpen = useSecondaryAnalysisStore((s) => s.setPanelOpen);
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
  const [workspaceAddMode, setWorkspaceAddMode] = useState<'create' | 'link'>('create');
  const [showWorkspaceSwitcher, setShowWorkspaceSwitcher] = useState(false);
  const workspaceSwitcherRequestNonce = useFileExplorerStore((s) => s.workspaceSwitcherRequestNonce);

  useEffect(() => {
    if (workspaceSwitcherRequestNonce > 0) {
      setShowWorkspaceSwitcher(true);
    }
  }, [workspaceSwitcherRequestNonce]);

  useShortcutOverlay('workspaceSwitcher', showWorkspaceSwitcher, () => setShowWorkspaceSwitcher(false));
  const [newWorkspaceName, setNewWorkspaceName] = useState('');
  const [newWorkspacePath, setNewWorkspacePath] = useState('');
  const [newWorkspaceParentPath, setNewWorkspaceParentPath] = useState('');

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
    devLog('FileExplorerPanel: Loading workspaces...');
    loadWorkspaces();
    void usePacksStore.getState().fetchPacks();
  }, [loadWorkspaces]);

  // Load files when workspace changes
  useEffect(() => {
    if (activeWorkspaceId) {
      devLog('FileExplorerPanel: Loading files for workspace:', activeWorkspaceId);
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

  const handleBrowseDirectory = async (target: 'link' | 'parent' = 'link') => {
    try {
      const selected = await open({
        directory: true,
        multiple: false,
        title: target === 'parent' ? 'Select parent folder for new workspace' : 'Select Workspace Directory',
      });

      if (selected && typeof selected === 'string') {
        if (target === 'parent') {
          setNewWorkspaceParentPath(selected);
        } else {
          setNewWorkspacePath(selected);
          if (!newWorkspaceName) {
            const dirName = selected.split('/').pop() || '';
            setNewWorkspaceName(dirName);
          }
        }
      }
    } catch (error) {
      console.error('Failed to open directory picker:', error);
    }
  };

  const resetAddWorkspaceForm = () => {
    setShowAddWorkspace(false);
    setWorkspaceAddMode('create');
    setNewWorkspaceName('');
    setNewWorkspacePath('');
    setNewWorkspaceParentPath('');
  };

  const handleAddWorkspace = async () => {
    if (!newWorkspaceName.trim()) return;
    if (workspaceAddMode === 'link' && !newWorkspacePath.trim()) return;

    try {
      if (workspaceAddMode === 'create') {
        await addWorkspace(newWorkspaceName.trim(), '', {
          create: true,
          parentPath: newWorkspaceParentPath.trim() || undefined,
        });
      } else {
        await addWorkspace(newWorkspaceName.trim(), newWorkspacePath.trim());
      }
      addToast({
        type: 'success',
        title: 'Workspace added',
        message:
          workspaceAddMode === 'create'
            ? `Created "${newWorkspaceName.trim()}"`
            : `Linked "${newWorkspaceName.trim()}"`,
      });
      resetAddWorkspaceForm();
    } catch (error) {
      console.error('Failed to add workspace:', error);
      addToast({
        type: 'error',
        title: 'Workspace failed',
        message: error instanceof Error ? error.message : 'Failed to add workspace',
      });
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

  const openScanAnalysisAtPath = async (
    workspaceId: string,
    analysisDir: string,
    options?: { initialWell?: string; selectedAnalyte?: string; linkedScanDir?: string; csvPath?: string }
  ) => {
    if (!hasScanAnalysis) {
      addToast({
        type: 'info',
        title: 'Life sciences pack',
        message: 'Install and enable Life sciences in Settings → Domain packs to open scan analysis.',
      });
      return;
    }
    try {
      const { data, linkedScanDir, source } = await loadScanAnalysisData(api, workspaceId, analysisDir, {
        csvPath: options?.csvPath,
        linkedScanDir: options?.linkedScanDir,
      });
      const selectedAnalyte =
        options?.selectedAnalyte ??
        (options?.csvPath ? analyteFromSummaryCsvPath(options.csvPath) ?? undefined : undefined);
      openScanAnalysis(workspaceId, analysisDir, data, {
        initialWell: options?.initialWell,
        selectedAnalyte,
        linkedScanDir,
      });
      if (onFileOpen) {
        onFileOpen();
      }
      const linkNote = linkedScanDir ? ` Scan linked: ${linkedScanDir || '(workspace root)'}.` : '';
      addToast({
        type: 'success',
        title: 'Scan analysis',
        message:
          (options?.initialWell ? `Opened well ${options.initialWell}` : `Opened analysis viewer (${source})`) +
          linkNote,
      });
    } catch (error) {
      console.error('Failed to open scan analysis:', error);
      const message = error instanceof Error ? error.message : 'Failed to open scan analysis';
      setError(message);
      addToast({ type: 'error', title: 'Scan analysis', message });
    }
  };

  const tryOpenScanAnalysisFile = async (
    workspaceId: string,
    filePath: string
  ): Promise<boolean> => {
    if (!hasScanAnalysis) return false;
    if (!isScanAnalysisResultsPath(filePath) && !isScanAnalysisSummaryCSVPath(filePath)) {
      return false;
    }
    const analysisDir = analysisDirFromFilePath(filePath);
    const selectedAnalyte = analyteFromSummaryCsvPath(filePath) ?? undefined;
    await openScanAnalysisAtPath(workspaceId, analysisDir, {
      csvPath: isScanAnalysisSummaryCSVPath(filePath) ? filePath : undefined,
      selectedAnalyte,
    });
    return true;
  };

  const openScanSummaryAtPath = async (
    workspaceId: string,
    summaryDir: string,
    initialWell?: string
  ) => {
    if (!hasScanSummary) {
      addToast({
        type: 'info',
        title: 'Life sciences pack',
        message: 'Enable Life sciences in Settings → Domain packs to open scan summaries.',
      });
      return;
    }
    const metaPath = summaryDir
      ? `${summaryDir.replace(/[/\\]+$/, '')}/${SCAN_SUMMARY_METADATA_FILE}`
      : SCAN_SUMMARY_METADATA_FILE;
    try {
      const raw = await api.fetchFileContent(workspaceId, metaPath);
      if (!raw || typeof raw !== 'string') {
        throw new Error('Empty metadata response from hub');
      }
      const data = parseScanSummaryMetadata(raw);
      openScanSummary(workspaceId, summaryDir, data, initialWell);
      if (onFileOpen) {
        onFileOpen();
      }
      addToast({
        type: 'success',
        title: 'Scan summary',
        message: initialWell ? `Opened well ${initialWell}` : 'Opened plate viewer',
      });
    } catch (error) {
      console.error('Failed to open scan summary:', error);
      const message = error instanceof Error ? error.message : 'Failed to open scan summary';
      setError(message);
      addToast({ type: 'error', title: 'Scan summary', message });
    }
  };

  const tryOpenScanSummaryFile = async (
    workspaceId: string,
    filePath: string
  ): Promise<boolean> => {
    if (!hasScanSummary) return false;
    const summaryDir = scanSummaryDirForFilePath(filePath);
    const isMetadata = isScanSummaryMetadataPath(filePath);
    const isWell = isScanSummaryWellPath(filePath);
    if (!isMetadata && !isWell) return false;
    const initialWell = isWell ? (filePath.split(/[/\\]/).pop() ?? 'A1') : undefined;
    await openScanSummaryAtPath(workspaceId, summaryDir, initialWell);
    return true;
  };

  const handleFileClick = async (file: FileNode) => {
    // Add null check for file.path to prevent crashes
    if (!file.path) {
      console.error('File path is undefined:', file);
      setError('File path is undefined');
      return;
    }
    
    if (file.is_dir) {
      devLog('Toggling directory:', file.path, 'current expanded:', !!expandedPaths[file.path]);
      const wasExpanded = !!expandedPaths[file.path];
      toggleExpanded(file.path);
      
      // If we're expanding the directory and it doesn't have children loaded, load them
      if (!wasExpanded && (!file.children || file.children.length === 0)) {
        const activeWorkspace = getActiveWorkspace();
        if (activeWorkspace) {
          try {
            devLog('Loading directory contents for:', file.path);
            await loadFiles(activeWorkspace.id, file.path);
          } catch (error) {
            console.error('Failed to load directory contents:', error);
            setError(error instanceof Error ? error.message : 'Failed to load directory contents');
          }
        }
      }
      
      setSelectedPath(file.path);
    } else {
      const activeWorkspace = getActiveWorkspace();
      if (activeWorkspace) {
        const openedAnalysis = await tryOpenScanAnalysisFile(activeWorkspace.id, file.path);
        if (openedAnalysis) {
          setSelectedPath(file.path);
          return;
        }
        const opened = await tryOpenScanSummaryFile(activeWorkspace.id, file.path);
        if (opened) {
          setSelectedPath(file.path);
          return;
        }
        if (hasCadWorkbench && !file.is_dir && file.path.toLowerCase().endsWith('.scad')) {
          const content = await api.fetchFileContent(activeWorkspace.id, file.path);
          openCadWorkbench(activeWorkspace.id, file.path, content);
          if (onFileOpen) onFileOpen();
          setSelectedPath(file.path);
          return;
        }
      }
      // Open file in editor
      if (activeWorkspace) {
        try {
          devLog('Opening file:', file.path, 'in workspace:', activeWorkspace.id);
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
          } else if (isEditableCsvPath(file.path)) {
            const content = await api.fetchFileContent(activeWorkspace.id, file.path);
            devLog('CSV file loaded, opening table view...');
            openFile(activeWorkspace.id, file.path, content, 'plaintext', { viewMode: 'csv-table' });
          } else {
            const content = await api.fetchFileContent(activeWorkspace.id, file.path);
            const language = getLanguageFromPath(file.path);
            devLog('File content loaded, opening in editor...');
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

  const handleFileDragStart = (e: React.DragEvent, file: FileNode) => {
    if (file.is_dir || !activeWorkspaceId || !file.path) return;
    e.stopPropagation();
    setWorkspaceFileDragData(e.dataTransfer, {
      workspaceId: activeWorkspaceId,
      path: file.path,
    });
  };

  const handleFileDragEnd = (e: React.DragEvent) => {
    dispatchWorkspaceFileDropEventAtPoint(e.clientX, e.clientY);
    scheduleWorkspaceFileDragClear();
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

  const findFileNode = (nodes: FileNode[], path: string): FileNode | undefined => {
    for (const node of nodes) {
      if (node.path === path) return node;
      if (node.children?.length) {
        const found = findFileNode(node.children, path);
        if (found) return found;
      }
    }
    return undefined;
  };

  const isScanAnalysisFolder = (file: FileNode): boolean => {
    if (!file.is_dir) return false;
    if (file.children?.length && isScanAnalysisDirListing(file.children)) {
      return true;
    }
    return /-summary$/i.test(file.name) || /-summary$/i.test(file.path);
  };

  const isScanSummaryFolder = (file: FileNode): boolean => {
    if (!file.is_dir) return false;
    if (file.children?.length && isScanSummaryDirListing(file.children)) {
      return true;
    }
    return /-summary$/i.test(file.name) || /-summary$/i.test(file.path);
  };

  const contextMenuIsScanAnalysis = (): boolean => {
    if (!contextMenu || !activeWorkspaceId) return false;
    if (isScanAnalysisResultsPath(contextMenu.path) || isScanAnalysisSummaryCSVPath(contextMenu.path)) {
      return true;
    }
    if (!contextMenu.isDir) return false;
    const tree = fileTree[activeWorkspaceId] ?? [];
    if (!contextMenu.path || contextMenu.path === '/' || contextMenu.path === '.') {
      return tree.some((f) => f.is_dir && f.name === SCAN_ANALYSIS_REPORTS_DIR);
    }
    const node = findFileNode(tree, contextMenu.path);
    if (node) return isScanAnalysisFolder(node);
    return contextMenu.path.endsWith(`/${SCAN_ANALYSIS_REPORTS_DIR}`);
  };

  const handleOpenScanAnalysisFromMenu = async () => {
    if (!contextMenu || !activeWorkspaceId) return;
    let analysisDir = '';
    if (isScanAnalysisResultsPath(contextMenu.path)) {
      analysisDir = analysisDirFromFilePath(contextMenu.path);
    } else if (isScanAnalysisSummaryCSVPath(contextMenu.path)) {
      analysisDir = analysisDirFromFilePath(contextMenu.path);
    } else if (contextMenu.isDir) {
      analysisDir =
        !contextMenu.path || contextMenu.path === '/' || contextMenu.path === '.'
          ? ''
          : contextMenu.path;
      if (analysisDir.endsWith(`/${SCAN_ANALYSIS_REPORTS_DIR}`)) {
        analysisDir = analysisDir.replace(/[/\\]reports$/, '');
      }
    }
    await openScanAnalysisAtPath(activeWorkspaceId, analysisDir, {
      csvPath: isScanAnalysisSummaryCSVPath(contextMenu.path) ? contextMenu.path : undefined,
      selectedAnalyte: analyteFromSummaryCsvPath(contextMenu.path) ?? undefined,
    });
    closeContextMenu();
  };

  const contextMenuIsComparator = (): boolean => {
    if (!contextMenu?.isDir || !contextMenu.path) return false;
    const base = contextMenu.path.split('/').pop() ?? contextMenu.path;
    return isComparatorAnalysisPath(base) || isComparatorAnalysisPath(contextMenu.path);
  };

  const contextMenuSummaryDir = (): string => {
    if (!contextMenu) return '';
    if (isScanAnalysisResultsPath(contextMenu.path) || isScanAnalysisSummaryCSVPath(contextMenu.path)) {
      return analysisDirFromFilePath(contextMenu.path);
    }
    if (contextMenu.isDir) {
      let d = contextMenu.path === '/' || contextMenu.path === '.' ? '' : contextMenu.path;
      if (d.endsWith(`/${SCAN_ANALYSIS_REPORTS_DIR}`)) {
        d = d.replace(/[/\\]reports$/, '');
      }
      return d;
    }
    return '';
  };

  const handleRun12PlexQCFromMenu = async () => {
    if (!contextMenu || !activeWorkspaceId || !hasSecondaryAnalysis) return;
    const analysisDir = contextMenuSummaryDir();
    closeContextMenu();
    try {
      const report = await api.run12PlexQC({
        workspace_id: activeWorkspaceId,
        analysis_dir: analysisDir,
        write_report: true,
      });
      await refreshTreeForPath(activeWorkspaceId, qcReportRelativePath(analysisDir));
      addToast({
        type: report.overall_pass ? 'success' : 'info',
        title: '12-Plex QC',
        message: report.overall_pass
          ? `${analysisDir || 'Plate'} passed SOP QC`
          : `${analysisDir || 'Plate'} failed one or more QC checks`,
      });
      await openScanAnalysisAtPath(activeWorkspaceId, analysisDir);
      const tab = useEditorStore
        .getState()
        .tabs.find(
          (t) =>
            t.workspaceId === activeWorkspaceId &&
            t.viewMode === 'scan-analysis' &&
            t.scanAnalysisDir === analysisDir
        );
      if (tab) setPanelQCReport(tab.id, report);
    } catch (e) {
      addToast({
        type: 'error',
        title: '12-Plex QC',
        message: e instanceof Error ? e.message : 'QC failed',
      });
    }
  };

  const handleAddToAnalysisBasket = () => {
    const dir = contextMenuSummaryDir();
    if (!dir) return;
    addToBasket(dir);
    setPanelOpen(true);
    addToast({ type: 'success', title: 'Analysis basket', message: `Added ${dir}` });
    closeContextMenu();
  };

  const handleOpenComparatorFromMenu = () => {
    if (!contextMenu || !activeWorkspaceId || !contextMenu.isDir) return;
    const dir =
      contextMenu.path === '/' || contextMenu.path === '.' ? '' : contextMenu.path.replace(/\/$/, '');
    openComparatorAnalysis(activeWorkspaceId, dir);
    if (onFileOpen) onFileOpen();
    closeContextMenu();
  };

  const handleOpenSecondaryPanel = () => {
    setPanelOpen(true);
    closeContextMenu();
  };

  const handleOpenCadFromMenu = async () => {
    if (!contextMenu || !activeWorkspaceId || contextMenu.isDir) return;
    if (!contextMenu.path.toLowerCase().endsWith('.scad')) return;
    try {
      const content = await api.fetchFileContent(activeWorkspaceId, contextMenu.path);
      openCadWorkbench(activeWorkspaceId, contextMenu.path, content);
      closeContextMenu();
      if (onFileOpen) onFileOpen();
    } catch (e) {
      addToast({ type: 'error', title: 'CAD workbench', message: e instanceof Error ? e.message : 'Failed to open' });
    }
  };

  const contextMenuIsScanSummary = (): boolean => {
    if (!contextMenu || !activeWorkspaceId) return false;
    if (isScanSummaryMetadataPath(contextMenu.path) || isScanSummaryWellPath(contextMenu.path)) {
      return true;
    }
    if (!contextMenu.isDir) return false;
    const tree = fileTree[activeWorkspaceId] ?? [];
    if (!contextMenu.path || contextMenu.path === '/' || contextMenu.path === '.') {
      return isScanSummaryWorkspaceRoot(tree);
    }
    const node = findFileNode(tree, contextMenu.path);
    if (node) return isScanSummaryFolder(node);
    return /-summary$/i.test(contextMenu.path);
  };

  const handleOpenScanSummaryFromMenu = async () => {
    if (!contextMenu || !activeWorkspaceId) return;
    let summaryDir = '';
    let initialWell: string | undefined;
    if (isScanSummaryWellPath(contextMenu.path)) {
      summaryDir = scanSummaryDirForFilePath(contextMenu.path);
      initialWell = contextMenu.path.split(/[/\\]/).pop();
    } else if (isScanSummaryMetadataPath(contextMenu.path)) {
      summaryDir = scanSummaryDirFromMetadataPath(contextMenu.path);
    } else if (contextMenu.isDir) {
      summaryDir =
        !contextMenu.path || contextMenu.path === '/' || contextMenu.path === '.'
          ? ''
          : contextMenu.path;
    }
    await openScanSummaryAtPath(activeWorkspaceId, summaryDir, initialWell);
    closeContextMenu();
  };

  const renderFileIcon = (file: FileNode) => {
    if (file.is_dir) {
      if (isScanAnalysisFolder(file)) {
        return expandedPaths[file.path] ? '📊' : '📊';
      }
      if (isScanSummaryFolder(file)) {
        return expandedPaths[file.path] ? '🔬' : '🔬';
      }
      return expandedPaths[file.path] ? '📂' : '📁';
    }
    if (isScanAnalysisResultsPath(file.path) || isScanAnalysisSummaryCSVPath(file.path)) {
      return '📊';
    }
    if (isScanSummaryMetadataPath(file.path) || isScanSummaryWellPath(file.path)) {
      return '🔬';
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
          } ${!file.is_dir ? 'cursor-grab active:cursor-grabbing' : ''}`}
          style={{ paddingLeft: `${level * 16 + 8}px` }}
          draggable={!file.is_dir}
          onDragStart={(e) => handleFileDragStart(e, file)}
          onDragEnd={handleFileDragEnd}
          onClick={() => handleFileClick(file)}
          onContextMenu={(e) => handleContextMenu(e, file)}
          title={file.is_dir ? undefined : 'Drag to chat to attach as context'}
        >
          <span className="text-sm">{renderFileIcon(file)}</span>
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
  const activeIsScanSummaryRoot =
    hasScanSummary && activeWorkspaceId != null && isScanSummaryWorkspaceRoot(files);
  const activeIsScanAnalysisRoot =
    hasScanSummary &&
    activeWorkspaceId != null &&
    (files.some((f) => f.is_dir && f.name === SCAN_ANALYSIS_REPORTS_DIR) ||
      isScanAnalysisRootListing(files));
  const activeIsCombinedRun =
    hasScanSummary && activeWorkspaceId != null && isCombinedRunDirListing(files);
  const workspaceSwitcherOverflow = useMemo(
    () => workspacesForTabBar(workspaces, activeWorkspaceId).overflowCount,
    [workspaces, activeWorkspaceId]
  );
  const canSwitchWorkspaces = workspaces.length > 1;

  return (
    <div 
      className={
        embedded
          ? 'border-r border-slack-border bg-slack-bg flex flex-col h-full relative'
          : 'border-r border-slack-border bg-slack-bg flex flex-col h-full relative animate-slide-in-left'
      }
      style={shrinkablePanelStyle(width, embedded ? MIN_WIDTH : COMPACT_MIN_WIDTH)}
    >
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
          {canSwitchWorkspaces && (
            <button
              type="button"
              onClick={() => setShowWorkspaceSwitcher(true)}
              className="text-slack-textMuted hover:text-slack-text transition-colors flex-shrink-0 px-1 py-0.5 text-xs font-medium"
              title="All workspaces"
              aria-label={
                workspaceSwitcherOverflow > 0
                  ? `All workspaces, ${workspaceSwitcherOverflow} not shown in tabs`
                  : 'All workspaces'
              }
            >
              {workspaceSwitcherOverflow > 0 ? `... +${workspaceSwitcherOverflow}` : '...'}
            </button>
          )}
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

      {(activeIsScanSummaryRoot || activeIsCombinedRun) && activeWorkspaceId && (
        <div className="px-4 py-2 border-b border-slack-border bg-slack-bg">
          <button
            type="button"
            onClick={() => void openScanSummaryAtPath(activeWorkspaceId, '')}
            className="w-full px-3 py-1.5 text-xs font-medium rounded bg-slack-accent/20 text-slack-accent hover:bg-slack-accent hover:text-white transition-colors"
          >
            Open scan summary
          </button>
        </div>
      )}

      {(activeIsScanAnalysisRoot || activeIsCombinedRun) && activeWorkspaceId && (
        <div className="px-4 py-2 border-b border-slack-border bg-slack-bg">
          <button
            type="button"
            onClick={() => void openScanAnalysisAtPath(activeWorkspaceId, '')}
            className="w-full px-3 py-1.5 text-xs font-medium rounded bg-purple-600/20 text-purple-300 hover:bg-purple-600 hover:text-white transition-colors"
          >
            {activeIsCombinedRun ? 'Open combined run (analysis)' : 'Open scan analysis'}
          </button>
        </div>
      )}

      {/* Workspace Tabs */}
      <div className="px-4 py-2 border-b border-slack-border bg-slack-bgHover">
        <WorkspaceTabBar
          workspaces={workspaces}
          activeWorkspaceId={activeWorkspaceId}
          onSelect={setActiveWorkspace}
          onRemove={handleRemoveWorkspace}
        />
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
        ) : loadingFiles && files.length === 0 ? (
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

      <WorkspaceSwitcherModal
        isOpen={showWorkspaceSwitcher}
        onClose={() => setShowWorkspaceSwitcher(false)}
        workspaces={workspaces}
        activeWorkspaceId={activeWorkspaceId}
        onSelect={setActiveWorkspace}
        onRemoveRequest={(id, name) => {
          setShowWorkspaceSwitcher(false);
          setPendingRemove({ id, name });
        }}
      />

      {/* Add Workspace Modal */}
      {showAddWorkspace && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50">
          <div className="bg-slack-bg border border-slack-border rounded p-6 w-[28rem] max-w-[95vw]">
            <h3 className="text-lg font-bold text-slack-text mb-4">Add Workspace</h3>
            <div className="flex rounded-md border border-slack-border overflow-hidden text-xs mb-4" role="tablist">
              <button
                type="button"
                role="tab"
                aria-selected={workspaceAddMode === 'create'}
                onClick={() => setWorkspaceAddMode('create')}
                className={`flex-1 px-3 py-1.5 font-medium ${
                  workspaceAddMode === 'create'
                    ? 'bg-slack-accent text-white'
                    : 'bg-slack-bgHover text-slack-textMuted hover:text-slack-text'
                }`}
              >
                Create new
              </button>
              <button
                type="button"
                role="tab"
                aria-selected={workspaceAddMode === 'link'}
                onClick={() => setWorkspaceAddMode('link')}
                className={`flex-1 px-3 py-1.5 font-medium ${
                  workspaceAddMode === 'link'
                    ? 'bg-slack-accent text-white'
                    : 'bg-slack-bgHover text-slack-textMuted hover:text-slack-text'
                }`}
              >
                Link existing
              </button>
            </div>
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
                  placeholder="Phoenix run 2026-06-04"
                />
              </div>
              {workspaceAddMode === 'create' ? (
                <div>
                  <label className="block text-sm font-medium text-slack-text mb-1">
                    Location (optional)
                  </label>
                  <div className="flex gap-2">
                    <input
                      type="text"
                      value={newWorkspaceParentPath}
                      onChange={(e) => setNewWorkspaceParentPath(e.target.value)}
                      className="flex-1 px-3 py-2 bg-slack-bg border border-slack-border rounded text-slack-text focus:outline-none focus:border-slack-accent font-mono text-xs"
                      placeholder="~/.neural-junkie/workspaces (default)"
                    />
                    <button
                      type="button"
                      onClick={() => void handleBrowseDirectory('parent')}
                      className="px-3 py-2 bg-slack-bgHover hover:bg-slack-accent text-slack-text hover:text-white rounded transition-colors"
                      title="Browse for parent folder"
                    >
                      📁
                    </button>
                  </div>
                  <p className="mt-1 text-[11px] text-slack-textMuted">
                    Creates a new folder from the name. Default root:{' '}
                    <code className="font-mono">~/.neural-junkie/workspaces/</code>
                  </p>
                </div>
              ) : (
                <div>
                  <label className="block text-sm font-medium text-slack-text mb-1">
                    Path
                  </label>
                  <div className="flex gap-2">
                    <input
                      type="text"
                      value={newWorkspacePath}
                      onChange={(e) => setNewWorkspacePath(e.target.value)}
                      className="flex-1 px-3 py-2 bg-slack-bg border border-slack-border rounded text-slack-text focus:outline-none focus:border-slack-accent font-mono text-xs"
                      placeholder="/path/to/existing/folder"
                    />
                    <button
                      type="button"
                      onClick={() => void handleBrowseDirectory('link')}
                      className="px-3 py-2 bg-slack-bgHover hover:bg-slack-accent text-slack-text hover:text-white rounded transition-colors"
                      title="Browse for directory"
                    >
                      📁
                    </button>
                  </div>
                </div>
              )}
            </div>
            <div className="flex gap-2 mt-6">
              <button
                type="button"
                onClick={() => void handleAddWorkspace()}
                disabled={
                  !newWorkspaceName.trim() ||
                  (workspaceAddMode === 'link' && !newWorkspacePath.trim())
                }
                className="px-4 py-2 bg-slack-accent hover:bg-slack-accentHover text-white text-sm rounded transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {workspaceAddMode === 'create' ? 'Create' : 'Add'}
              </button>
              <button
                type="button"
                onClick={resetAddWorkspaceForm}
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
          {hasScanAnalysis && contextMenuIsScanAnalysis() && (
            <button
              onClick={() => void handleOpenScanAnalysisFromMenu()}
              className="w-full px-4 py-2 text-left text-sm text-slack-text hover:bg-slack-bgHover"
            >
              📊 Open scan analysis
            </button>
          )}

          {hasSecondaryAnalysis && contextMenuIsScanAnalysis() && contextMenu?.isDir && (
            <>
              <button
                onClick={() => void handleRun12PlexQCFromMenu()}
                className="w-full px-4 py-2 text-left text-sm text-slack-text hover:bg-slack-bgHover"
              >
                ✅ Run 12-Plex QC
              </button>
              <button
                onClick={handleAddToAnalysisBasket}
                className="w-full px-4 py-2 text-left text-sm text-slack-text hover:bg-slack-bgHover"
              >
                🧪 Add to analysis basket
              </button>
              <button
                onClick={handleOpenSecondaryPanel}
                className="w-full px-4 py-2 text-left text-sm text-slack-text hover:bg-slack-bgHover"
              >
                📋 Secondary analysis panel…
              </button>
            </>
          )}

          {hasSecondaryAnalysis && contextMenuIsComparator() && (
            <>
              <button
                onClick={handleOpenComparatorFromMenu}
                className="w-full px-4 py-2 text-left text-sm text-slack-text hover:bg-slack-bgHover"
              >
                📈 Open comparator analysis
              </button>
              <button
                onClick={handleOpenSecondaryPanel}
                className="w-full px-4 py-2 text-left text-sm text-slack-text hover:bg-slack-bgHover"
              >
                🧬 Run endogenous analysis…
              </button>
            </>
          )}

          {hasCadWorkbench && !contextMenu.isDir && contextMenu.path.toLowerCase().endsWith('.scad') && (
            <button
              onClick={() => void handleOpenCadFromMenu()}
              className="w-full px-4 py-2 text-left text-sm text-slack-text hover:bg-slack-bgHover"
            >
              📐 Open CAD workbench
            </button>
          )}

          {hasScanSummary && contextMenuIsScanSummary() && (
              <button
                onClick={handleOpenScanSummaryFromMenu}
                className="w-full px-4 py-2 text-left text-sm text-slack-text hover:bg-slack-bgHover"
              >
                🔬 Open scan summary
              </button>
            )}

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
