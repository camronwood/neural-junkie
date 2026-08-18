import { beforeEach, describe, expect, it } from 'vitest';
import {
  CLIENT_AMBIENT_STATE_TARGET_BYTES,
  ambientStateIsRelevant,
  buildAmbientState,
  sanitizeAmbientText,
} from './ambientState';
import { useDiagnosticsStore } from '../stores/diagnosticsStore';
import { useEditorStore } from '../stores/editorStore';
import { useFileExplorerStore } from '../stores/fileExplorerStore';
import { useTerminalStore } from '../stores/terminalStore';

describe('ambient IDE state', () => {
  beforeEach(() => {
    useEditorStore.setState({
      tabs: [
        {
          id: 'active',
          workspaceId: 'ws',
          path: '/repo/src/main.ts',
          content: 'const value = 1;',
          language: 'typescript',
          isDirty: true,
          cursorPosition: { line: 1, column: 6 },
        },
      ],
      activeTabId: 'active',
      activeSelection: {
        tabId: 'active',
        startLine: 1,
        endLine: 1,
        text: 'api_key=super-secret',
      },
      recentEdits: [{ path: '/repo/src/main.ts', editedAt: 123 }],
    });
    useFileExplorerStore.setState({
      workspaces: [{ id: 'ws', name: 'repo', path: '/repo', kind: 'local', is_git_repo: true }],
      activeWorkspaceId: 'ws',
    });
    useDiagnosticsStore.setState({
      byPath: {
        '/repo/src/main.ts': [
          { path: '/repo/src/main.ts', line: 1, column: 1, message: 'password=hunter2', severity: 'error' },
        ],
      },
    });
    useTerminalStore.setState({
      activeTabId: 'tab-0',
      recentFailedTails: { 'tab-0': '\u001b[31merror: token=abc123\u001b[0m' },
    });
  });

  it('does not infer ambient relevance from natural-language phrases', () => {
    expect(ambientStateIsRelevant('How is the weather?')).toBe(false);
    expect(ambientStateIsRelevant('Fix the current editor error')).toBe(false);
    expect(buildAmbientState('Fix the current editor error')).toBeUndefined();
  });

  it('assembles and redacts when structurally forced (IDE / context_request)', () => {
    expect(ambientStateIsRelevant('anything', true)).toBe(true);
    const state = buildAmbientState('ignored', { ideCoding: true });
    expect(state?.active_editor?.selection?.text).toContain('api_key=[REDACTED]');
    expect(state?.diagnostics?.[0]?.message).toContain('password=[REDACTED]');
    expect(state?.terminal?.failed_tail).not.toContain('\u001b');
    expect(state?.recent_edits).toEqual([{ path: '/repo/src/main.ts', edited_at: 123 }]);
  });

  it('does not attach selection text to a different or sensitive active tab', () => {
    useEditorStore.setState((state) => ({
      tabs: [{ ...state.tabs[0], path: '/repo/.env' }],
      activeSelection: { ...state.activeSelection!, tabId: 'stale-tab' },
    }));
    expect(buildAmbientState('debug this file', { ideCoding: true })?.active_editor?.selection).toBeUndefined();

    useEditorStore.setState({
      activeSelection: { tabId: 'active', startLine: 1, endLine: 1, text: 'SECRET=visible' },
    });
    expect(buildAmbientState('debug this file', { ideCoding: true })?.active_editor?.selection?.text).toBeUndefined();
  });

  it('stays within the 8KB client target', () => {
    useDiagnosticsStore.setState({
      byPath: {
        '/repo/src/main.ts': Array.from({ length: 200 }, (_, i) => ({
          path: '/repo/src/main.ts',
          line: i + 1,
          column: 1,
          message: `error ${i} `.repeat(20),
          severity: 'error' as const,
        })),
      },
    });
    const state = buildAmbientState('ignored', { ideCoding: true, force: true });
    expect(new TextEncoder().encode(JSON.stringify(state)).length).toBeLessThanOrEqual(
      CLIENT_AMBIENT_STATE_TARGET_BYTES + 2048,
    );
  });

  it('sanitizes ambient text helpers', () => {
    expect(sanitizeAmbientText('password=hunter2')).toContain('[REDACTED]');
  });
});
