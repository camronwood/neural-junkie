import { beforeEach, describe, expect, it, vi } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useChatWindowWorkspaceActions } from './useChatWindowWorkspaceActions';
import type { Collaboration } from '../types/protocol';

const openFile = vi.fn();
const revealLine = vi.fn();

vi.mock('../stores/chatStore', () => ({
  useChatStore: { getState: () => ({ channel: 'general' }) },
}));

vi.mock('../stores/fileExplorerStore', () => ({
  useFileExplorerStore: (sel: (s: Record<string, unknown>) => unknown) =>
    sel({ activeWorkspaceId: null, workspaces: [] }),
}));

vi.mock('../stores/editorStore', () => ({
  useEditorStore: Object.assign(
    (sel: (s: Record<string, unknown>) => unknown) =>
      sel({ openFile, revealLine, tabs: [], activeTabId: null }),
    { getState: () => ({}) },
  ),
}));

vi.mock('../stores/projectSetsStore', () => ({
  useProjectSetsStore: (sel: (s: Record<string, unknown>) => unknown) =>
    sel({ activeProjectSetId: null, getMemberIds: () => [] }),
}));

vi.mock('../stores/fileChangeStore', () => ({
  useFileChangeStore: { getState: () => ({ fetchPendingChanges: vi.fn() }) },
}));

function executingCollab(overrides: Partial<Collaboration> = {}): Collaboration {
  return {
    id: 'c1',
    channel: 'general',
    phase: 'executing',
    execution_mode: 'worktree',
    workspace_acknowledged: false,
    ...overrides,
  } as Collaboration;
}

describe('useChatWindowWorkspaceActions', () => {
  const addToast = vi.fn();
  const setCodeEditorOpen = vi.fn();
  const setFileExplorerOpen = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
  });

  function render(collaborationsByID: Record<string, Collaboration>) {
    return renderHook(
      ({ collabs }) =>
        useChatWindowWorkspaceActions({
          channel: 'general',
          username: 'tester',
          collaborationsByID: collabs,
          addToast,
          setCodeEditorOpen,
          setFileExplorerOpen,
        }),
      { initialProps: { collabs: collaborationsByID } },
    );
  }

  it('opens the workspace gate and toasts once for an unacknowledged executing collab', () => {
    const { result } = render({ c1: executingCollab() });
    expect(result.current.workspaceGateCollab?.id).toBe('c1');
    expect(addToast).toHaveBeenCalledTimes(1);
  });

  it('does not open the gate when the workspace is already acknowledged', () => {
    const { result } = render({ c1: executingCollab({ workspace_acknowledged: true }) });
    expect(result.current.workspaceGateCollab).toBeNull();
    expect(addToast).not.toHaveBeenCalled();
  });

  it('does not open the gate for collabs on another channel', () => {
    const { result } = render({ c1: executingCollab({ channel: 'other' }) });
    expect(result.current.workspaceGateCollab).toBeNull();
  });

  it('openWorkspaceGate is a no-op when no collab awaits confirmation', () => {
    const { result } = render({});
    act(() => result.current.openWorkspaceGate());
    expect(result.current.workspaceGateCollab).toBeNull();
    expect(result.current.channelAwaitingWorkspaceCollab).toBeNull();
  });

  it('returns no scoped repo paths with no workspaces', () => {
    const { result } = render({});
    expect(result.current.resolveScopedRepoPaths()).toEqual([]);
  });

  it('file-open helpers no-op without a workspace', async () => {
    const { result } = render({});
    await act(async () => {
      await result.current.handleQuickOpenPath('a.ts');
      await result.current.handleOpenAtLine('a.ts', 3);
      await result.current.handleCADFilesWritten({});
      await result.current.handleImplementationSessionComplete({});
    });
    expect(openFile).not.toHaveBeenCalled();
    expect(setCodeEditorOpen).not.toHaveBeenCalled();
    expect(setFileExplorerOpen).not.toHaveBeenCalled();
  });

  it('warns on implementation session verify failure', async () => {
    const { result } = render({});
    await act(async () => {
      await result.current.handleImplementationSessionComplete({
        implementation_session_outcome: { verify_failed: true, failure_type: 'tests' },
      });
    });
    expect(addToast).toHaveBeenCalled();
  });
});
