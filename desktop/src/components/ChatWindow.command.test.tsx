import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createChatCommandActions } from '../hooks/createChatCommandActions';
import type { ChatAPI } from '../api/chatAPI';
import { createRef } from 'react';

vi.mock('../stores/fileExplorerStore', () => {
  const state = {
    workspaces: [],
    activeWorkspaceId: null,
  };
  return {
    useFileExplorerStore: Object.assign(() => state, { getState: () => state }),
  };
});

vi.mock('../stores/editorStore', () => {
  const state = {
    openKnowledgeGraphWorkbench: vi.fn(),
    openArtifact: vi.fn(),
  };
  return {
    useEditorStore: Object.assign(() => state, { getState: () => state }),
  };
});

vi.mock('../utils/repoAgentWorkspace', () => ({
  parseCreateRepoAgentCommand: () => null,
  ensureRepoAgentWorkspace: vi.fn(),
}));

vi.mock('../components/chat/chatWindowHelpers', () => ({
  withClientPaletteCommands: (defs: unknown[]) => defs,
}));

describe('createChatCommandActions', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('handleCommandExecute opens model library for /nj-open-model-library', async () => {
    const setModelLibraryOpen = vi.fn();
    const appendLocalSlashCommand = vi.fn();
    const handleSendMessage = vi.fn();
    const inputRef = createRef<HTMLTextAreaElement>();

    const { handleCommandExecute } = createChatCommandActions({
      api: {} as ChatAPI,
      channel: 'general',
      username: 'camron',
      commandDefsLength: 0,
      setCommandDefs: vi.fn(),
      setCommandPaletteFilter: vi.fn(),
      setCommandPaletteOpen: vi.fn(),
      setModelLibraryOpen,
      setCodeEditorOpen: vi.fn(),
      setFileExplorerOpen: vi.fn(),
      setAssistantTasks: vi.fn(),
      setAssistantReminders: vi.fn(),
      updateLayoutSettings: vi.fn(),
      appendLocalSlashCommand,
      handleSendMessage,
      loadCollaborations: vi.fn(),
      fetchPendingChanges: vi.fn(),
      addToast: vi.fn(),
      inputRef,
    });

    await handleCommandExecute('/nj-open-model-library');

    expect(appendLocalSlashCommand).toHaveBeenCalledWith('/nj-open-model-library');
    expect(setModelLibraryOpen).toHaveBeenCalledWith(true);
    expect(handleSendMessage).not.toHaveBeenCalled();
  });

  it('openCommandPalette loads command defs and assistant state', () => {
    const setCommandPaletteOpen = vi.fn();
    const setCommandPaletteFilter = vi.fn();
    const fetchCommands = vi.fn().mockResolvedValue([{ name: 'help' }]);
    const fetchAssistantState = vi.fn().mockResolvedValue({ tasks: [], reminders: [] });
    const setCommandDefs = vi.fn();
    const loadCollaborations = vi.fn().mockResolvedValue(undefined);
    const fetchPendingChanges = vi.fn().mockResolvedValue(undefined);

    const { openCommandPalette } = createChatCommandActions({
      api: { fetchCommands, fetchAssistantState } as unknown as ChatAPI,
      channel: 'general',
      username: 'camron',
      commandDefsLength: 0,
      setCommandDefs,
      setCommandPaletteFilter,
      setCommandPaletteOpen,
      setModelLibraryOpen: vi.fn(),
      setCodeEditorOpen: vi.fn(),
      setFileExplorerOpen: vi.fn(),
      setAssistantTasks: vi.fn(),
      setAssistantReminders: vi.fn(),
      updateLayoutSettings: vi.fn(),
      appendLocalSlashCommand: vi.fn(),
      handleSendMessage: vi.fn(),
      loadCollaborations,
      fetchPendingChanges,
      addToast: vi.fn(),
      inputRef: createRef(),
    });

    openCommandPalette('help');

    expect(setCommandPaletteFilter).toHaveBeenCalledWith('help');
    expect(setCommandPaletteOpen).toHaveBeenCalledWith(true);
    expect(fetchCommands).toHaveBeenCalledWith(true);
    expect(loadCollaborations).toHaveBeenCalledWith('general');
    expect(fetchAssistantState).toHaveBeenCalledWith('general');
    expect(fetchPendingChanges).toHaveBeenCalledWith('camron');
  });
});
