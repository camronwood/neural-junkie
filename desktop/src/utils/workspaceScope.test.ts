import { describe, expect, it } from 'vitest';
import { resolveWorkspaceScope, scopedRepoPaths, scopeSummaryLabel } from './workspaceScope';
import type { Workspace } from '../stores/fileExplorerStore';
import type { EditorTab } from '../stores/editorStore';

const ws = (id: string, path: string, name?: string): Workspace => ({
  id,
  name: name ?? id,
  path,
  created_at: '',
  last_used: '',
  is_git_repo: true,
});

const tab = (id: string, workspaceId: string, path: string): EditorTab =>
  ({
    id,
    workspaceId,
    path,
    content: '',
    language: 'go',
    isDirty: false,
  }) as EditorTab;

describe('resolveWorkspaceScope', () => {
  it('includes open-tab workspaces as linked', () => {
    const workspaces = [ws('a', '/tmp/primary', 'primary'), ws('b', '/tmp/linked', 'linked')];
    const scope = resolveWorkspaceScope({
      workspaces,
      activeWorkspaceId: 'a',
      editorTabs: [tab('t1', 'a', 'main.go'), tab('t2', 'b', 'pkg/foo.go')],
      activeTabId: 't1',
    });
    expect(scope.primary?.id).toBe('a');
    expect(scope.linked).toHaveLength(1);
    expect(scope.linked[0].workspace_id).toBe('b');
    expect(scope.linked[0].source).toBe('open_tab');
    expect(scopeSummaryLabel(scope)).toBe('primary + 1 repo');
    expect(scopedRepoPaths(scope)).toEqual(['/tmp/primary', '/tmp/linked']);
  });

  it('dedupes primary from linked', () => {
    const workspaces = [ws('a', '/tmp/primary')];
    const scope = resolveWorkspaceScope({
      workspaces,
      activeWorkspaceId: 'a',
      editorTabs: [tab('t1', 'a', 'main.go')],
      activeTabId: 't1',
    });
    expect(scope.linked).toHaveLength(0);
    expect(scopeSummaryLabel(scope)).toBeNull();
  });
});
