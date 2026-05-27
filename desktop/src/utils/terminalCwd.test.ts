import { describe, expect, it, beforeEach } from 'vitest';
import { useFileExplorerStore } from '../stores/fileExplorerStore';
import { resolveTerminalCwd } from './terminalCwd';
import type { Collaboration } from '../types/protocol';

describe('resolveTerminalCwd', () => {
  beforeEach(() => {
    useFileExplorerStore.setState({
      workspaces: [{ id: 'w1', name: 'sandbox', path: '/Users/me/development/sandbox' }],
      activeWorkspaceId: 'w1',
      fileTree: {},
    });
  });

  it('prefers collaboration source_repo_path over sandbox', () => {
    const collab: Collaboration = {
      id: 'c1',
      title: 't',
      description: 'd',
      phase: 'executing',
      agents: [],
      channel: 'collab-c1',
      created_by: 'u',
      created_at: '',
      updated_at: '',
      source_repo_path: '/Users/me/development/sandbox',
      working_directory: '/Users/me/.neural-junkie/collaborations/c1',
    };
    expect(resolveTerminalCwd({ collaboration: collab })).toBe('/Users/me/development/sandbox');
  });

  it('ignores collab sandbox as active workspace fallback', () => {
    useFileExplorerStore.setState({
      workspaces: [
        {
          id: 'w-collab',
          name: 'Collab',
          path: '/Users/me/.neural-junkie/collaborations/old-id',
        },
      ],
      activeWorkspaceId: 'w-collab',
      fileTree: {},
    });
    expect(resolveTerminalCwd()).toBe('~');
  });
});
