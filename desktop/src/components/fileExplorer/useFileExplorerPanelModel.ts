import { useState, useEffect, useMemo, useRef } from 'react';
import { useFileExplorerStore } from '../../stores/fileExplorerStore';
import { useEditorStore } from '../../stores/editorStore';
import { usePacksStore } from '../../stores/packsStore';
import { openPackViewer } from '../../stores/packViewerHost';
import { useToastStore } from '../../stores/toastStore';
import { ChatAPI } from '../../api/chatAPI';
import { getHubBaseURL } from '../../config/hubUrl';
import type { FileNode } from '../../stores/fileExplorerStore';
import { invoke } from '@tauri-apps/api/core';
import { open } from '@tauri-apps/plugin-dialog';
import { isImagePreviewPath, isPdfPreviewPath, workspaceAbsolutePath } from '../../utils/editorFileKind';
import {
  dispatchWorkspaceFileDropEventAtPoint,
  dispatchWorkspaceFileDropToStickyZone,
  scheduleWorkspaceFileDragClear,
  setWorkspaceFileDragData,
} from '../../utils/workspaceFileDrag';
import { resolveEditorImageSrc, resolveEditorPdfSrc } from '../../utils/chatImageSrc';
import {
  isScanSummaryMetadataPath,
  isScanSummaryWellPath,
  isScanSummaryWorkspaceRoot,
  parseScanSummaryMetadata,
  scanSummaryDirForFilePath,
  scanSummaryDirFromMetadataPath,
  SCAN_SUMMARY_METADATA_FILE,
} from '../../utils/scanSummary';
import {
  isCombinedRunDirListing,
  isScanAnalysisResultsPath,
  isScanAnalysisRootListing,
  isScanAnalysisSummaryCSVPath,
  SCAN_ANALYSIS_REPORTS_DIR,
} from '../../utils/scanAnalysis';
import { analyteFromSummaryCsvPath } from '../../utils/scanAnalysisCsv';
import { loadScanAnalysisData, analysisDirFromFilePath } from '../../utils/scanAnalysisLoad';
import { isComparatorAnalysisPath } from '../../utils/secondaryAnalysis';
import { isEditableCsvPath } from '../../utils/csvTable';
import { useSecondaryAnalysisStore } from '../../stores/secondaryAnalysisStore';
import { PACK_CAP } from '../../stores/packCapabilities';
import { workspacesForTabBar } from '../../utils/workspaceOrder';
import { useShortcutOverlay } from '../../shortcuts/useShortcutOverlay';
import { devLog } from '../../utils/devLog';
import { qcReportRelativePath } from '../../utils/panelQcUtils';
import {
  basenameRelativePath,
  duplicateRelativePath,
  joinRelativePath,
  newItemParentPath,
  replaceBasename,
} from '../../utils/workspacePaths';
import { useTerminalStore } from '../../stores/terminalStore';

import {
  getLanguageFromPath,
  findFileNode,
  isScanAnalysisFolder,
  isScanSummaryFolder,
} from './fileExplorerUtils';
import type { FileExplorerPanelProps } from './types';

const MIN_WIDTH = 200;
const COMPACT_MIN_WIDTH = 160;
const DEFAULT_WIDTH = 300;
const STORAGE_KEY = 'file-explorer-panel-width';

export function useFileExplorerPanelModel({
  onClose,
  onFileOpen,
  variant = 'overlay',
}: FileExplorerPanelProps) {
  const embedded = variant === 'embedded';
  const {
    workspaces,
    activeWorkspaceId,
    fileTree,
    expandedPaths,
    loadingFiles,
    error,
    loadWorkspaces,
    addWorkspace,
    connectRemoteWorkspace,
    setActiveWorkspace,
    loadFiles,
    refreshTreeForPath,
    toggleExpanded,
    setSelectedPath,
    toggleSelectedPath,
    isPathSelected,
    createFile,
    createFolder,
    renameFile,
    deleteFile,
    removeWorkspace,
    getActiveWorkspace,
    setError,
    clearError,
  } = useFileExplorerStore();

  const { openFile, openScanSummary, openScanAnalysis, openCadWorkbench, openHtmlBrowser, openMusicWorkbench, openArenaWorkbench, openComparatorAnalysis, setPanelQCReport } =
    useEditorStore();
  const hasScanSummary = usePacksStore((s) => s.hasCapability('scan-summary-viewer'));
  const hasScanAnalysis = usePacksStore((s) => s.hasCapability('scan-analysis-viewer'));
  const hasSecondaryAnalysis = usePacksStore((s) => s.hasCapability(PACK_CAP.SECONDARY_ANALYSIS_VIEWER));
  const hasCadWorkbench = usePacksStore((s) => s.hasCapability('cad-workbench'));
  const hasHtmlBrowserWorkbench = usePacksStore((s) => s.hasCapability(PACK_CAP.WEB_BROWSER_WORKBENCH));
  const hasMusicWorkbench = usePacksStore((s) => s.hasCapability(PACK_CAP.MUSIC_WORKBENCH));
  const hasArenaWorkbench = usePacksStore((s) => s.hasCapability(PACK_CAP.MODEL_ARENA_WORKBENCH));
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
  const [workspaceAddMode, setWorkspaceAddMode] = useState<'create' | 'link' | 'remote' | 'devcontainer'>('create');
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
  const [remoteHost, setRemoteHost] = useState('');
  const [remoteUser, setRemoteUser] = useState('');
  const [remotePath, setRemotePath] = useState('');
  const [sidecarUrl, setSidecarUrl] = useState('http://127.0.0.1:19876');
  const [sidecarToken, setSidecarToken] = useState('');
  const [devcontainerRepoPath, setDevcontainerRepoPath] = useState('');
  const [devcontainerPlan, setDevcontainerPlan] = useState<{
    container_name?: string;
    workspace_folder?: string;
    image?: string;
    sidecar_port?: number;
  } | null>(null);
  const [devcontainerPlanLoading, setDevcontainerPlanLoading] = useState(false);

  // State for file operations
  const [contextMenu, setContextMenu] = useState<{
    x: number;
    y: number;
    path: string;
    isDir: boolean;
    /** True when opened on empty panel area (workspace root). */
    isBackground?: boolean;
  } | null>(null);
  const [nameDialog, setNameDialog] = useState<{
    mode: 'rename' | 'create-file' | 'create-folder';
    initial: string;
    /** Parent dir for create-file / create-folder ('' = workspace root). */
    createParentPath?: string;
  } | null>(null);

  const setTerminalPanelOpen = useTerminalStore((s) => s.setPanelOpen);
  const alignTerminalCwd = useTerminalStore((s) => s.alignActiveTabCwd);

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

  const handleBrowseDirectory = async (target: 'link' | 'parent' | 'devcontainer' = 'link') => {
    try {
      const selected = await open({
        directory: true,
        multiple: false,
        title: target === 'parent' ? 'Select parent folder for new workspace' : 'Select Workspace Directory',
      });

      if (selected && typeof selected === 'string') {
        if (target === 'parent') {
          setNewWorkspaceParentPath(selected);
        } else if (target === 'devcontainer') {
          setDevcontainerRepoPath(selected);
          setDevcontainerPlan(null);
          if (!newWorkspaceName) {
            const dirName = selected.split('/').pop() || '';
            setNewWorkspaceName(dirName);
          }
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
    setRemoteHost('');
    setRemoteUser('');
    setRemotePath('');
    setSidecarUrl('http://127.0.0.1:19876');
    setSidecarToken('');
    setDevcontainerRepoPath('');
    setDevcontainerPlan(null);
    setDevcontainerPlanLoading(false);
  };

  const handleLoadDevcontainerPlan = async () => {
    const repo = devcontainerRepoPath.trim();
    if (!repo) return;
    setDevcontainerPlanLoading(true);
    try {
      const plan = await api.fetchDevcontainerPlanByPath(repo);
      setDevcontainerPlan(plan);
      const pingUrl =
        plan?.sidecar_port != null
          ? `http://127.0.0.1:${plan.sidecar_port}`
          : sidecarUrl.trim();
      if (plan?.sidecar_port != null && !sidecarUrl.includes(String(plan.sidecar_port))) {
        setSidecarUrl(pingUrl);
      }
      const pingOk = await api.pingSidecar(pingUrl, sidecarToken.trim() || undefined);
      if (!pingOk) {
        addToast({
          type: 'info',
          title: 'Sidecar not reachable',
          message: 'Run devcontainer up, start nj-remote in the container, then connect.',
        });
      }
    } catch (error) {
      setDevcontainerPlan(null);
      addToast({
        type: 'error',
        title: 'Dev container plan failed',
        message: error instanceof Error ? error.message : 'Could not load devcontainer.json',
      });
    } finally {
      setDevcontainerPlanLoading(false);
    }
  };

  const handleAddWorkspace = async () => {
    if (!newWorkspaceName.trim()) return;
    if (workspaceAddMode === 'link' && !newWorkspacePath.trim()) return;
    if (workspaceAddMode === 'remote' && (!remoteHost.trim() || !remotePath.trim() || !sidecarUrl.trim())) return;
    if (workspaceAddMode === 'devcontainer' && (!devcontainerRepoPath.trim() || !sidecarUrl.trim())) return;

    try {
      if (workspaceAddMode === 'remote') {
        await connectRemoteWorkspace({
          name: newWorkspaceName.trim(),
          remoteHost: remoteHost.trim(),
          remoteUser: remoteUser.trim() || 'ubuntu',
          remotePath: remotePath.trim(),
          sidecarUrl: sidecarUrl.trim(),
          token: sidecarToken.trim(),
          kind: 'ssh',
        });
        addToast({
          type: 'success',
          title: 'Remote workspace connected',
          message: `Connected to ${remoteHost.trim()}`,
        });
        resetAddWorkspaceForm();
        return;
      } else if (workspaceAddMode === 'devcontainer') {
        await connectRemoteWorkspace({
          name: newWorkspaceName.trim(),
          remoteHost: remoteHost.trim() || 'devcontainer',
          remoteUser: remoteUser.trim() || 'devcontainer',
          remotePath: devcontainerPlan?.workspace_folder?.trim() || devcontainerRepoPath.trim(),
          sidecarUrl: sidecarUrl.trim(),
          token: sidecarToken.trim(),
          kind: 'devcontainer',
        });
        addToast({
          type: 'success',
          title: 'Dev container workspace connected',
          message: devcontainerPlan?.container_name
            ? `Attached to ${devcontainerPlan.container_name}`
            : 'Connected via nj-remote sidecar',
        });
        resetAddWorkspaceForm();
        return;
      } else if (workspaceAddMode === 'create') {
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

  const handleFileClick = async (file: FileNode, e?: React.MouseEvent) => {
    const multi = !!(e?.metaKey || e?.ctrlKey);
    // Add null check for file.path to prevent crashes
    if (!file.path) {
      console.error('File path is undefined:', file);
      setError('File path is undefined');
      return;
    }

    if (multi && !file.is_dir) {
      e?.preventDefault();
      // Cmd/Ctrl+click toggles multi-select without opening. Do this before any
      // await so an in-flight single-click open cannot wipe the selection.
      toggleSelectedPath(file.path, true);
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
      // Select immediately (before awaits) so Cmd/Ctrl+click can extend while
      // content loads. Never call setSelectedPath after awaits — that collapses
      // multi-select back to one file.
      setSelectedPath(file.path);
      const activeWorkspace = getActiveWorkspace();
      if (activeWorkspace) {
        const registry = usePacksStore.getState().capabilityRegistry;
        const packOpen = await openPackViewer(
          registry,
          activeWorkspace.id,
          file.path,
          (wsId, p) => api.fetchFileContent(wsId, p),
        );
        if (packOpen === 'opened') {
          if (onFileOpen) onFileOpen();
          return;
        }
        const openedAnalysis = await tryOpenScanAnalysisFile(activeWorkspace.id, file.path);
        if (openedAnalysis) {
          return;
        }
        const opened = await tryOpenScanSummaryFile(activeWorkspace.id, file.path);
        if (opened) {
          return;
        }
        // Extension fallbacks for packs that have not yet declared file-viewer globs.
        if (hasCadWorkbench && !file.is_dir && file.path.toLowerCase().endsWith('.scad')) {
          const content = await api.fetchFileContent(activeWorkspace.id, file.path);
          openCadWorkbench(activeWorkspace.id, file.path, content);
          if (onFileOpen) onFileOpen();
          return;
        }
        if (hasMusicWorkbench && !file.is_dir && (/\.(wav|mp3|flac|aiff?)$/i.test(file.path) || /\.nj-music\.json$/i.test(file.path) || file.path.endsWith('project.nj-music.json'))) {
          let content = '';
          if (/\.nj-music\.json$/i.test(file.path) || file.path.endsWith('project.nj-music.json')) {
            content = await api.fetchFileContent(activeWorkspace.id, file.path);
          }
          openMusicWorkbench(activeWorkspace.id, file.path, content);
          if (onFileOpen) onFileOpen();
          return;
        }
        if (hasArenaWorkbench && !file.is_dir && /\.nj-arena\.json$/i.test(file.path)) {
          openArenaWorkbench(activeWorkspace.id, file.path);
          if (onFileOpen) onFileOpen();
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
              preview: e?.detail !== 2,
            });
          } else if (isPdfPreviewPath(file.path)) {
            const absolutePath = workspaceAbsolutePath(activeWorkspace.path, file.path);
            const imageSrc = await resolveEditorPdfSrc({
              workspaceId: activeWorkspace.id,
              relativePath: file.path,
              absolutePath,
            });
            openFile(activeWorkspace.id, file.path, '', undefined, {
              viewMode: 'pdf',
              imageSrc,
              preview: e?.detail !== 2,
            });
          } else if (isEditableCsvPath(file.path)) {
            const content = await api.fetchFileContent(activeWorkspace.id, file.path);
            devLog('CSV file loaded, opening table view...');
            openFile(activeWorkspace.id, file.path, content, 'plaintext', {
              viewMode: 'csv-table',
              preview: e?.detail !== 2,
            });
          } else {
            const content = await api.fetchFileContent(activeWorkspace.id, file.path);
            const language = getLanguageFromPath(file.path);
            devLog('File content loaded, opening in editor...');
            openFile(activeWorkspace.id, file.path, content, language, {
              preview: e?.detail !== 2,
            });
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
    }
  };

  const handleFileDragStart = (e: React.DragEvent, file: FileNode) => {
    if (file.is_dir || !activeWorkspaceId || !file.path) return;
    e.stopPropagation();
    const selected = useFileExplorerStore.getState().selectedPaths;
    const paths =
      selected.includes(file.path) && selected.length > 1 ? selected : [file.path];
    setWorkspaceFileDragData(
      e.dataTransfer,
      paths.map((path) => ({ workspaceId: activeWorkspaceId, path }))
    );
  };

  const handleFileDragEnd = (e: React.DragEvent) => {
    // WKWebView often skips HTML5 drop for in-app drags and reports (0,0) on dragend.
    // Try point first, then sticky composer zone from the last hover.
    const delivered =
      dispatchWorkspaceFileDropEventAtPoint(e.clientX, e.clientY) ||
      dispatchWorkspaceFileDropToStickyZone();
    if (!delivered) {
      scheduleWorkspaceFileDragClear();
    } else {
      // Drop handler clears drag data; still schedule a safety clear.
      scheduleWorkspaceFileDragClear();
    }
  };

  const handleContextMenu = (e: React.MouseEvent, file: FileNode) => {
    e.preventDefault();
    e.stopPropagation();
    setContextMenu({
      x: e.clientX,
      y: e.clientY,
      path: file.path,
      isDir: file.is_dir,
    });
  };

  const handleBackgroundContextMenu = (e: React.MouseEvent) => {
    e.preventDefault();
    if (!activeWorkspaceId) return;
    setContextMenu({
      x: e.clientX,
      y: e.clientY,
      path: '',
      isDir: true,
      isBackground: true,
    });
  };

  const closeContextMenu = () => {
    setContextMenu(null);
  };

  const openCreateAtRoot = (mode: 'create-file' | 'create-folder') => {
    if (!activeWorkspaceId) return;
    setNameDialog({ mode, initial: '', createParentPath: '' });
  };

  const contextFileNode = (): FileNode | null => {
    if (!contextMenu || !activeWorkspaceId) return null;
    const tree = fileTree[activeWorkspaceId] ?? [];
    return findFileNode(tree, contextMenu.path) ?? {
      path: contextMenu.path,
      name: basenameRelativePath(contextMenu.path),
      is_dir: contextMenu.isDir,
      size: 0,
      mod_time: '',
    };
  };

  const handleOpenFromMenu = async () => {
    const node = contextFileNode();
    if (!node || node.is_dir) return;
    closeContextMenu();
    await handleFileClick(node);
  };

  const handleCreateFile = () => {
    if (!contextMenu) return;
    setNameDialog({
      mode: 'create-file',
      initial: '',
      createParentPath: newItemParentPath(contextMenu.path, contextMenu.isDir),
    });
  };

  const handleCreateFolder = () => {
    if (!contextMenu) return;
    setNameDialog({
      mode: 'create-folder',
      initial: '',
      createParentPath: newItemParentPath(contextMenu.path, contextMenu.isDir),
    });
  };

  const handleRename = () => {
    if (!contextMenu || contextMenu.isBackground) return;
    setNameDialog({
      mode: 'rename',
      initial: basenameRelativePath(contextMenu.path),
    });
  };

  const handleNameDialogConfirm = async (name: string) => {
    if (!nameDialog || !activeWorkspaceId) {
      setNameDialog(null);
      return;
    }
    try {
      if (nameDialog.mode === 'rename') {
        if (!contextMenu || contextMenu.isBackground) {
          setNameDialog(null);
          return;
        }
        const newPath = replaceBasename(contextMenu.path, name);
        if (newPath === contextMenu.path) {
          setNameDialog(null);
          closeContextMenu();
          return;
        }
        await renameFile(activeWorkspaceId, contextMenu.path, newPath);
        addToast({ type: 'success', title: 'Renamed', message: newPath });
      } else {
        const parent =
          nameDialog.createParentPath ??
          (contextMenu ? newItemParentPath(contextMenu.path, contextMenu.isDir) : '');
        const newPath = joinRelativePath(parent, name);
        if (nameDialog.mode === 'create-folder') {
          await createFolder(activeWorkspaceId, newPath);
          addToast({ type: 'success', title: 'Folder created', message: newPath });
        } else {
          await createFile(activeWorkspaceId, newPath);
          addToast({ type: 'success', title: 'File created', message: newPath });
        }
      }
      setNameDialog(null);
      closeContextMenu();
    } catch (error) {
      console.error('File operation failed:', error);
      addToast({
        type: 'error',
        title: 'File operation failed',
        message: error instanceof Error ? error.message : String(error),
      });
    }
  };

  const handleDuplicate = async () => {
    if (!contextMenu || !activeWorkspaceId || contextMenu.isDir) return;
    try {
      const newPath = duplicateRelativePath(contextMenu.path);
      const content = await api.fetchFileContent(activeWorkspaceId, contextMenu.path);
      await createFile(activeWorkspaceId, newPath, content);
      addToast({ type: 'success', title: 'Duplicated', message: newPath });
      closeContextMenu();
    } catch (error) {
      addToast({
        type: 'error',
        title: 'Duplicate failed',
        message: error instanceof Error ? error.message : String(error),
      });
    }
  };

  const handleCopyName = async () => {
    if (!contextMenu) return;
    try {
      await navigator.clipboard.writeText(basenameRelativePath(contextMenu.path));
      addToast({ type: 'success', title: 'Name copied', message: basenameRelativePath(contextMenu.path) });
      closeContextMenu();
    } catch {
      addToast({ type: 'error', title: 'Copy failed', message: 'Could not copy file name' });
    }
  };

  const handleRevealInFinder = async () => {
    if (!contextMenu) return;
    const activeWorkspace = getActiveWorkspace();
    if (!activeWorkspace) return;
    try {
      const absolutePath =
        contextMenu.isBackground || !contextMenu.path
          ? activeWorkspace.path
          : workspaceAbsolutePath(activeWorkspace.path, contextMenu.path);
      const { openPath } = await import('@tauri-apps/plugin-opener');
      await openPath(absolutePath);
      closeContextMenu();
    } catch (error) {
      addToast({
        type: 'error',
        title: 'Reveal failed',
        message: error instanceof Error ? error.message : String(error),
      });
    }
  };

  const handleOpenInTerminal = () => {
    if (!contextMenu) return;
    const activeWorkspace = getActiveWorkspace();
    if (!activeWorkspace) return;
    const absolutePath =
      contextMenu.isBackground || !contextMenu.path
        ? activeWorkspace.path
        : contextMenu.isDir
          ? workspaceAbsolutePath(activeWorkspace.path, contextMenu.path)
          : workspaceAbsolutePath(activeWorkspace.path, newItemParentPath(contextMenu.path, false) || '.');
    alignTerminalCwd(absolutePath);
    setTerminalPanelOpen(true);
    closeContextMenu();
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

  const handleOpenHtmlFromMenu = async () => {
    if (!contextMenu || !activeWorkspaceId || contextMenu.isDir) return;
    if (!/\.html?$/i.test(contextMenu.path)) return;
    try {
      const content = await api.fetchFileContent(activeWorkspaceId, contextMenu.path);
      openHtmlBrowser(activeWorkspaceId, contextMenu.path, content);
      closeContextMenu();
      if (onFileOpen) onFileOpen();
    } catch (e) {
      addToast({ type: 'error', title: 'HTML browser', message: e instanceof Error ? e.message : 'Failed to open' });
    }
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


  return {
    COMPACT_MIN_WIDTH,
    MIN_WIDTH,
    activeIsCombinedRun,
    activeIsScanAnalysisRoot,
    activeIsScanSummaryRoot,
    activeWorkspaceId,
    canSwitchWorkspaces,
    clearError,
    closeContextMenu,
    confirmRemoveWorkspace,
    contextMenu,
    contextMenuIsComparator,
    contextMenuIsScanAnalysis,
    contextMenuIsScanSummary,
    devcontainerPlan,
    devcontainerPlanLoading,
    devcontainerRepoPath,
    embedded,
    error,
    expandedPaths,
    files,
    getActiveWorkspace,
    handleAddToAnalysisBasket,
    handleAddWorkspace,
    handleBackgroundContextMenu,
    handleBrowseDirectory,
    handleContextMenu,
    handleCopyName,
    handleCopyPath,
    handleCopyRelativePath,
    handleCreateFile,
    handleCreateFolder,
    handleDelete,
    handleDuplicate,
    handleFileClick,
    handleFileDragEnd,
    handleFileDragStart,
    handleLoadDevcontainerPlan,
    handleNameDialogConfirm,
    handleOpenCadFromMenu,
    handleOpenComparatorFromMenu,
    handleOpenFromMenu,
    handleOpenHtmlFromMenu,
    handleOpenInTerminal,
    handleOpenScanAnalysisFromMenu,
    handleOpenScanSummaryFromMenu,
    handleOpenSecondaryPanel,
    handlePreviewMarkdown,
    handleRemoveWorkspace,
    handleRename,
    handleResizeStart,
    handleRevealInFinder,
    handleRun12PlexQCFromMenu,
    hasCadWorkbench,
    hasHtmlBrowserWorkbench,
    hasScanAnalysis,
    hasScanSummary,
    hasSecondaryAnalysis,
    isPathSelected,
    loadingFiles,
    nameDialog,
    newWorkspaceName,
    newWorkspaceParentPath,
    newWorkspacePath,
    onClose,
    openCreateAtRoot,
    openScanAnalysisAtPath,
    openScanSummaryAtPath,
    pendingRemove,
    remoteHost,
    remotePath,
    remoteUser,
    resetAddWorkspaceForm,
    setActiveWorkspace,
    setDevcontainerPlan,
    setDevcontainerRepoPath,
    setNameDialog,
    setNewWorkspaceName,
    setNewWorkspaceParentPath,
    setNewWorkspacePath,
    setPendingRemove,
    setRemoteHost,
    setRemotePath,
    setRemoteUser,
    setShowAddWorkspace,
    setShowWorkspaceSwitcher,
    setSidecarToken,
    setSidecarUrl,
    setWorkspaceAddMode,
    showAddWorkspace,
    showWorkspaceSwitcher,
    sidecarToken,
    sidecarUrl,
    width,
    workspaceAddMode,
    workspaceSwitcherOverflow,
    workspaces,
  };
}
