import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  applyEditStreamMessage,
  cancelEditApplyStream,
  resetEditApplyStreamStateForTests,
  wasEditApplyStreamed,
} from './applyEditStreamDelta';
import type { Message } from '../types/protocol';
import {
  EDIT_APPLY_CHANGE_ID_KEY,
  EDIT_APPLY_DONE_KEY,
  EDIT_APPLY_OFFSET_KEY,
  EDIT_APPLY_PATH_KEY,
  STREAM_KIND_EDIT_APPLY,
  STREAM_KIND_METADATA_KEY,
} from '../types/protocol';

const openFile = vi.fn();
const setActiveTab = vi.fn();
const applyExternalTabContent = vi.fn();
const tabs: Array<{ id: string; workspaceId: string; path: string; content: string }> = [];

vi.mock('../stores/editorStore', () => ({
  useEditorStore: {
    getState: () => ({
      tabs,
      openFile: (...args: unknown[]) => {
        openFile(...args);
        tabs.push({
          id: 'tab-1',
          workspaceId: args[0] as string,
          path: args[1] as string,
          content: args[2] as string,
        });
      },
      setActiveTab,
      applyExternalTabContent,
    }),
  },
}));

vi.mock('../stores/fileExplorerStore', () => ({
  useFileExplorerStore: {
    getState: () => ({
      workspaces: [{ id: 'ws1', path: '/ws' }],
      activeWorkspaceId: 'ws1',
    }),
  },
}));

vi.mock('../stores/settingsStore', () => ({
  useSettingsStore: {
    getState: () => ({
      updateLayoutSettings: vi.fn().mockResolvedValue(undefined),
    }),
  },
}));

const syncPendingChangeToEditor = vi.fn().mockResolvedValue(undefined);
vi.mock('./syncPendingChangeToEditor', () => ({
  syncPendingChangeToEditor: (...args: unknown[]) => syncPendingChangeToEditor(...args),
}));

vi.mock('../stores/fileChangeStore', () => ({
  useFileChangeStore: {
    getState: () => ({
      pendingChanges: [
        {
          id: 'chg-1',
          operation: 'edit',
          file_path: '/ws/src/a.ts',
          old_content: 'old',
          new_content: 'hello world',
          status: 'pending',
        },
      ],
      fetchPendingChanges: vi.fn().mockResolvedValue(undefined),
    }),
  },
}));

function editDelta(partial: {
  content?: string;
  offset?: number;
  done?: boolean;
  type?: 'stream_delta' | 'stream_end';
}): Message {
  return {
    id: 'stream-1',
    type: partial.type ?? 'stream_delta',
    content: partial.content ?? '',
    from: { id: 'a1', name: 'Agent', type: 'agent' },
    timestamp: '2026-01-01T00:00:00Z',
    channel: 'general',
    metadata: {
      [STREAM_KIND_METADATA_KEY]: STREAM_KIND_EDIT_APPLY,
      [EDIT_APPLY_CHANGE_ID_KEY]: 'chg-1',
      [EDIT_APPLY_PATH_KEY]: '/ws/src/a.ts',
      [EDIT_APPLY_OFFSET_KEY]: partial.offset ?? 0,
      ...(partial.done ? { [EDIT_APPLY_DONE_KEY]: true } : {}),
    },
  } as Message;
}

describe('applyEditStreamDelta', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    tabs.length = 0;
    resetEditApplyStreamStateForTests();
  });

  it('auto-opens tab on first delta and applies progressive content', () => {
    expect(applyEditStreamMessage(editDelta({ content: 'hello', offset: 0 }))).toBe(true);
    expect(openFile).toHaveBeenCalledWith('ws1', 'src/a.ts', 'hello', expect.any(String));
    expect(applyExternalTabContent).toHaveBeenCalledWith('tab-1', 'hello');
    expect(wasEditApplyStreamed('chg-1')).toBe(true);

    applyEditStreamMessage(editDelta({ content: ' world', offset: 5 }));
    expect(applyExternalTabContent).toHaveBeenLastCalledWith('tab-1', 'hello world');
  });

  it('hands off to pending hunk sync on stream_end', () => {
    applyEditStreamMessage(editDelta({ content: 'hello', offset: 0 }));
    applyEditStreamMessage(editDelta({ type: 'stream_end', done: true, offset: 5 }));
    expect(syncPendingChangeToEditor).toHaveBeenCalledWith(
      expect.objectContaining({ id: 'chg-1' }),
    );
  });

  it('ignores non-edit-apply streams', () => {
    const msg = {
      ...editDelta({ content: 'x' }),
      metadata: { tool_step: 'start' },
    } as Message;
    expect(applyEditStreamMessage(msg)).toBe(false);
    expect(openFile).not.toHaveBeenCalled();
  });

  it('stops applying after cancel (reject)', () => {
    applyEditStreamMessage(editDelta({ content: 'hel', offset: 0 }));
    cancelEditApplyStream('chg-1');
    applyExternalTabContent.mockClear();
    applyEditStreamMessage(editDelta({ content: 'lo', offset: 3 }));
    expect(applyExternalTabContent).not.toHaveBeenCalled();
  });
});
