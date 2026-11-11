import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { shallow } from 'zustand/shallow';
import { ChatAPI } from '../api/chatAPI';
import { getHubBaseURL } from '../config/hubUrl';
import { CAD_FILES_WRITTEN_KEY, IMPLEMENTATION_FILES_CHANGED_KEY, IMPLEMENTATION_SESSION_OUTCOME_KEY } from '../constants/promptMetadata';
import { useChatStore } from '../stores/chatStore';
import { useEditorStore } from '../stores/editorStore';
import { useFileChangeStore } from '../stores/fileChangeStore';
import { useFileExplorerStore } from '../stores/fileExplorerStore';
import { useProjectSetsStore } from '../stores/projectSetsStore';
import type { Toast } from '../stores/toastStore';
import type { Collaboration } from '../types/protocol';
import { isAwaitingWorkspaceConfirmation } from '../utils/collaborationPanelState';
import { isEditableCsvPath } from '../utils/csvTable';
import { getLanguageFromPath } from '../utils/editorLanguage';
import { refreshFileExplorerForPaths } from '../utils/refreshFileExplorer';
import { resolveWorkspaceScope, scopedRepoPaths } from '../utils/workspaceScope';

export interface UseChatWindowWorkspaceActionsArgs {
  channel: string;
  username: string;
  collaborationsByID: Record<string, Collaboration>;
  addToast: (toast: Omit<Toast, 'id' | 'count'>) => void;
  setCodeEditorOpen: (open: boolean) => void;
  setFileExplorerOpen: (open: boolean) => void;
}

/**
 * Workspace scope resolution, file-open helpers, and the collaboration workspace gate for ChatWindow.
 * Extraction loop 3 from ChatWindowRoot.
 */
export function useChatWindowWorkspaceActions({
  channel,
  username,
  collaborationsByID,
  addToast,
  setCodeEditorOpen,
  setFileExplorerOpen,
}: UseChatWindowWorkspaceActionsArgs) {
  const { activeWorkspaceId, workspaces: explorerWorkspaces } = useFileExplorerStore(
    (s) => ({ activeWorkspaceId: s.activeWorkspaceId, workspaces: s.workspaces }),
    shallow
  );
  const openFileInEditor = useEditorStore((s) => s.openFile);
  const revealLineInEditor = useEditorStore((s) => s.revealLine);
  const editorTabs = useEditorStore((s) => s.tabs);
  const activeTabId = useEditorStore((s) => s.activeTabId);
  const activeProjectSetId = useProjectSetsStore((s) => s.activeProjectSetId);
  const getProjectSetMemberIds = useProjectSetsStore((s) => s.getMemberIds);

  const resolveScopedRepoPaths = useCallback((): string[] => {
    const scope = resolveWorkspaceScope({
      workspaces: explorerWorkspaces,
      activeWorkspaceId,
      editorTabs,
      activeTabId,
      projectSetMemberIds: activeProjectSetId
        ? getProjectSetMemberIds(activeProjectSetId)
        : undefined,
    });
    return scopedRepoPaths(scope);
  }, [
    explorerWorkspaces,
    activeWorkspaceId,
    editorTabs,
    activeTabId,
    activeProjectSetId,
    getProjectSetMemberIds,
  ]);

  const handleQuickOpenPath = useCallback(
    async (path: string) => {
      const ws =
        explorerWorkspaces.find((w) => w.id === activeWorkspaceId) ??
        explorerWorkspaces[0];
      if (!ws) return;
      try {
        const api = new ChatAPI(getHubBaseURL());
        const content = await api.fetchFileContent(ws.id, path);
        if (isEditableCsvPath(path)) {
          openFileInEditor(ws.id, path, content, 'plaintext', { viewMode: 'csv-table' });
        } else {
          openFileInEditor(ws.id, path, content, getLanguageFromPath(path));
        }
        setCodeEditorOpen(true);
        setFileExplorerOpen(true);
      } catch (e) {
        addToast({
          type: 'error',
          title: 'Open file',
          message: e instanceof Error ? e.message : String(e),
        });
      }
    },
    [activeWorkspaceId, explorerWorkspaces, openFileInEditor, addToast]
  );

  const handleImplementationSessionComplete = useCallback(
    async (metadata?: Record<string, unknown>) => {
      const outcomeRaw = metadata?.[IMPLEMENTATION_SESSION_OUTCOME_KEY];
      if (outcomeRaw && typeof outcomeRaw === 'object') {
        const outcome = outcomeRaw as Record<string, unknown>;
        if (outcome.verify_failed === true || outcome.circuit_breaker_triggered === true) {
          const reason =
            (typeof outcome.failure_type === 'string' && outcome.failure_type) ||
            (outcome.circuit_breaker_triggered ? 'circuit breaker' : 'verify failed');
          addToast({
            type: 'warning',
            title: 'Implementation session issue',
            message: `Session finished with ${reason}. Expand the outcome card in chat for details.`,
          });
        } else if (outcome.outcome === 'wrong_route' && typeof outcome.suggested_agent === 'string') {
          addToast({
            type: 'info',
            title: 'Wrong specialist for this task',
            message: `Try ${outcome.suggested_agent} in Agent mode for boot-fix work.`,
          });
        }
      }
      const raw = metadata?.[IMPLEMENTATION_FILES_CHANGED_KEY];
      const paths = Array.isArray(raw) ? raw.filter((p): p is string => typeof p === 'string' && p.trim() !== '') : [];
      if (paths.length === 0) return;
      const ws =
        explorerWorkspaces.find((w) => w.id === activeWorkspaceId) ??
        explorerWorkspaces[0];
      if (!ws) return;
      const chatApi = new ChatAPI(getHubBaseURL());
      for (const relPath of paths) {
        try {
          const existing = useEditorStore.getState().getTabByPath(ws.id, relPath);
          if (existing) {
            await useEditorStore.getState().refreshTabFromDisk(ws.id, relPath);
            revealLineInEditor(ws.id, relPath, 1);
          } else {
            const content = await chatApi.fetchFileContent(ws.id, relPath);
            if (isEditableCsvPath(relPath)) {
              openFileInEditor(ws.id, relPath, content, 'plaintext', { viewMode: 'csv-table' });
            } else {
              openFileInEditor(ws.id, relPath, content, getLanguageFromPath(relPath));
            }
          }
        } catch (e) {
          console.error('[impl-session] open changed file:', relPath, e);
        }
      }
      await refreshFileExplorerForPaths(ws.id, paths);
      setCodeEditorOpen(true);
      setFileExplorerOpen(true);
      void useFileChangeStore.getState().fetchPendingChanges(username || 'default');
    },
    [
      activeWorkspaceId,
      explorerWorkspaces,
      openFileInEditor,
      revealLineInEditor,
      username,
      addToast,
    ]
  );

  const handleCADFilesWritten = useCallback(
    async (metadata?: Record<string, unknown>) => {
      const raw = metadata?.[CAD_FILES_WRITTEN_KEY];
      const paths = Array.isArray(raw)
        ? raw.filter((p): p is string => typeof p === 'string' && p.trim() !== '')
        : [];
      if (paths.length === 0) return;
      const ws =
        explorerWorkspaces.find((w) => w.id === activeWorkspaceId) ??
        explorerWorkspaces[0];
      if (!ws) return;
      await refreshFileExplorerForPaths(ws.id, paths);
      setFileExplorerOpen(true);
    },
    [activeWorkspaceId, explorerWorkspaces]
  );

  const handleOpenAtLine = useCallback(
    async (path: string, line: number) => {
      const ws =
        explorerWorkspaces.find((w) => w.id === activeWorkspaceId) ??
        explorerWorkspaces[0];
      if (!ws) return;
      try {
        const api = new ChatAPI(getHubBaseURL());
        const content = await api.fetchFileContent(ws.id, path);
        if (isEditableCsvPath(path)) {
          openFileInEditor(ws.id, path, content, 'plaintext', { viewMode: 'csv-table' });
        } else {
          openFileInEditor(ws.id, path, content, getLanguageFromPath(path));
        }
        revealLineInEditor(ws.id, path, line);
        setCodeEditorOpen(true);
        setFileExplorerOpen(true);
      } catch (e) {
        addToast({
          type: 'error',
          title: 'Open symbol',
          message: e instanceof Error ? e.message : String(e),
        });
      }
    },
    [
      activeWorkspaceId,
      explorerWorkspaces,
      openFileInEditor,
      revealLineInEditor,
      addToast,
    ]
  );

  const [workspaceGateCollab, setWorkspaceGateCollab] = useState<Collaboration | null>(null);
  const [workspaceGateBusy, setWorkspaceGateBusy] = useState(false);
  const dismissedWorkspaceGateIdRef = useRef<string | null>(null);
  const workspaceGateToastIdRef = useRef<string | null>(null);

  useEffect(() => {
    const activeCh = useChatStore.getState().channel;
    let next: Collaboration | null = null;
    for (const c of Object.values(collaborationsByID)) {
      if (c.phase !== 'executing' || c.workspace_acknowledged) continue;
      const isWorktree = c.execution_mode === 'worktree';
      if (!isWorktree && !c.working_directory?.trim()) continue;
      if (c.channel !== activeCh) continue;
      if (dismissedWorkspaceGateIdRef.current === c.id) continue;
      next = c;
      break;
    }
    setWorkspaceGateCollab(next);
    if (next && workspaceGateToastIdRef.current !== next.id) {
      workspaceGateToastIdRef.current = next.id;
      addToast({
        type: 'info',
        title: 'Confirm execution workspace',
        message: 'Agents are paused until you click Continue in the workspace dialog.',
      });
    }
    if (!next) {
      workspaceGateToastIdRef.current = null;
    }
  }, [collaborationsByID, channel, addToast]);

  const channelAwaitingWorkspaceCollab = useMemo(() => {
    for (const c of Object.values(collaborationsByID)) {
      if (c.channel !== channel) continue;
      if (isAwaitingWorkspaceConfirmation(c)) return c;
    }
    return null;
  }, [collaborationsByID, channel]);

  const openWorkspaceGate = useCallback(() => {
    if (!channelAwaitingWorkspaceCollab) return;
    dismissedWorkspaceGateIdRef.current = null;
    setWorkspaceGateCollab(channelAwaitingWorkspaceCollab);
  }, [channelAwaitingWorkspaceCollab]);

  const openWorkspaceGateForCollab = useCallback((collab: Collaboration) => {
    if (!isAwaitingWorkspaceConfirmation(collab)) return;
    dismissedWorkspaceGateIdRef.current = null;
    setWorkspaceGateCollab(collab);
  }, []);

  return {
    activeWorkspaceId,
    explorerWorkspaces,
    resolveScopedRepoPaths,
    handleQuickOpenPath,
    handleImplementationSessionComplete,
    handleCADFilesWritten,
    handleOpenAtLine,
    workspaceGateCollab,
    setWorkspaceGateCollab,
    workspaceGateBusy,
    setWorkspaceGateBusy,
    dismissedWorkspaceGateIdRef,
    workspaceGateToastIdRef,
    channelAwaitingWorkspaceCollab,
    openWorkspaceGate,
    openWorkspaceGateForCollab,
  };
}
