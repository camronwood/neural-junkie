import type { Dispatch, MutableRefObject, RefObject, SetStateAction } from 'react';
import { ChatAPI } from '../api/chatAPI';
import { useEditorStore } from '../stores/editorStore';
import { useFileExplorerStore } from '../stores/fileExplorerStore';
import type { LayoutSettings } from '../stores/settingsStore';
import type { Toast } from '../stores/toastStore';
import type {
  AssistantReminder,
  AssistantTask,
  CommandDefinition,
} from '../types/protocol';
import {
  ensureRepoAgentWorkspace,
  parseCreateRepoAgentCommand,
} from '../utils/repoAgentWorkspace';
import { withClientPaletteCommands } from '../components/chat/chatWindowHelpers';

export type ChatCommandActionsDeps = {
  api: ChatAPI;
  channel: string;
  username: string;
  commandDefsLength: number;
  setCommandDefs: Dispatch<SetStateAction<CommandDefinition[]>>;
  setCommandPaletteFilter: Dispatch<SetStateAction<string>>;
  setCommandPaletteOpen: Dispatch<SetStateAction<boolean>>;
  setModelLibraryOpen: Dispatch<SetStateAction<boolean>>;
  setCodeEditorOpen: Dispatch<SetStateAction<boolean>>;
  setFileExplorerOpen: Dispatch<SetStateAction<boolean>>;
  setAssistantTasks: Dispatch<SetStateAction<AssistantTask[]>>;
  setAssistantReminders: Dispatch<SetStateAction<AssistantReminder[]>>;
  updateLayoutSettings: (patch: Partial<LayoutSettings>) => Promise<void> | void;
  appendLocalSlashCommand: (commandText: string) => void;
  handleSendMessage: (content: string, metadata?: Record<string, unknown>) => Promise<boolean>;
  loadCollaborations: (targetChannel: string) => Promise<unknown>;
  fetchPendingChanges: (userId: string) => Promise<unknown>;
  addToast: (toast: Omit<Toast, 'id' | 'count'>) => void;
  inputRef: RefObject<HTMLTextAreaElement | null>;
};

function clearComposerInput(inputRef: RefObject<HTMLTextAreaElement | null>) {
  (inputRef.current as (HTMLTextAreaElement & { clearInput?: () => void }) | null)?.clearInput?.();
}

/** Command palette / local slash handlers extracted from ChatWindow. */
export function createChatCommandActions(deps: ChatCommandActionsDeps) {
  const ensureCommandDefs = async (forceRefresh: boolean = false) => {
    if (!forceRefresh && deps.commandDefsLength > 0) return;
    try {
      const defs = await deps.api.fetchCommands(forceRefresh);
      deps.setCommandDefs(withClientPaletteCommands(defs));
    } catch (err) {
      console.error('Failed to load command definitions:', err);
      deps.setCommandDefs(withClientPaletteCommands([]));
    }
  };

  const handleCommandExecute = async (
    commandString: string,
    metadata?: Record<string, unknown>
  ) => {
    const trimmed = commandString.trim();
    if (trimmed === '/nj-open-model-library') {
      deps.appendLocalSlashCommand(trimmed);
      deps.setModelLibraryOpen(true);
      clearComposerInput(deps.inputRef);
      return;
    }
    if (trimmed === '/nj-open-knowledge-graph') {
      deps.appendLocalSlashCommand(trimmed);
      const fe = useFileExplorerStore.getState();
      const ws = fe.workspaces.find((w) => w.id === fe.activeWorkspaceId);
      if (ws?.path && fe.activeWorkspaceId) {
        useEditorStore.getState().openKnowledgeGraphWorkbench(fe.activeWorkspaceId, ws.path);
        deps.setCodeEditorOpen(true);
        void deps.updateLayoutSettings({ editorPanelVisible: true });
      } else {
        deps.addToast({
          type: 'warning',
          title: 'No workspace',
          message: 'Open a workspace in the Files panel first.',
        });
      }
      clearComposerInput(deps.inputRef);
      return;
    }
    if (trimmed === '/nj-open-neural-canvas') {
      deps.appendLocalSlashCommand(trimmed);
      const fe = useFileExplorerStore.getState();
      useEditorStore.getState().openArtifact(
        fe.activeWorkspaceId ?? '',
        '__library__',
        'Neural Canvas',
      );
      deps.setCodeEditorOpen(true);
      void deps.updateLayoutSettings({ editorPanelVisible: true });
      clearComposerInput(deps.inputRef);
      return;
    }
    const repoAgentCmd = parseCreateRepoAgentCommand(trimmed);
    const sent = await deps.handleSendMessage(commandString, metadata);
    if (sent !== false) {
      clearComposerInput(deps.inputRef);
    }
    if (repoAgentCmd && sent !== false) {
      window.setTimeout(() => {
        void ensureRepoAgentWorkspace(repoAgentCmd.repoPath, {
          preferredName: repoAgentCmd.agentName,
        }).then((workspaceId) => {
          if (workspaceId) {
            deps.setFileExplorerOpen(true);
          }
        });
      }, 400);
    }
  };

  const openCommandPalette = (filter = '') => {
    deps.setCommandPaletteFilter(filter);
    deps.setCommandPaletteOpen(true);
    void ensureCommandDefs(true);
    void deps.loadCollaborations(deps.channel);
    void deps.api
      .fetchAssistantState(deps.channel)
      .then((state) => {
        deps.setAssistantTasks(state.tasks || []);
        deps.setAssistantReminders(state.reminders || []);
      })
      .catch((error) => console.error('Failed to load assistant state:', error));
    void deps.fetchPendingChanges(deps.username || 'default').catch((error) =>
      console.error('Failed to load pending file changes:', error)
    );
  };

  return {
    ensureCommandDefs,
    handleCommandExecute,
    openCommandPalette,
  };
}

/** Stable deps bag for tests that need to mutate commandDefsLength. */
export type ChatCommandActionsDepsRef = MutableRefObject<ChatCommandActionsDeps>;
