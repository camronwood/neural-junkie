import { describe, expect, it } from 'vitest';
import {
  buildInlineCompletionLocalContext,
  buildNeighborSnippets,
} from './inlineCompletionContext';
import type { EditorTab } from '../stores/editorStore';

describe('buildInlineCompletionLocalContext', () => {
  it('splits prefix/suffix at offset and keeps a local window', () => {
    const text = ['line1', 'line2 PREFIX', 'line3', 'line4'].join('\n');
    const offset = text.indexOf('PREFIX') + 'PREFIX'.length;
    const { prefix, suffix, context } = buildInlineCompletionLocalContext(text, offset);
    expect(prefix.endsWith('PREFIX')).toBe(true);
    expect(suffix.startsWith('\nline3') || suffix.includes('line3')).toBe(true);
    expect(context).toContain('line2 PREFIX');
    expect(context).toContain('line3');
  });

  it('truncates very long prefix from the head', () => {
    const head = 'H'.repeat(12000);
    const text = `${head}CURSOR_TAIL`;
    const offset = text.length;
    const { prefix } = buildInlineCompletionLocalContext(text, offset);
    expect(prefix.length).toBeLessThanOrEqual(8010);
    expect(prefix.includes('CURSOR_TAIL')).toBe(true);
  });
});

describe('buildNeighborSnippets', () => {
  it('includes other open text tabs and marks recent edits', () => {
    const tabs: EditorTab[] = [
      {
        id: 'a',
        workspaceId: 'w',
        path: '/repo/a.ts',
        content: 'export const a = 1;',
        isDirty: false,
        viewMode: 'text',
      },
      {
        id: 'b',
        workspaceId: 'w',
        path: '/repo/b.ts',
        content: 'export const b = 2;\n'.repeat(5),
        isDirty: false,
        viewMode: 'text',
      },
    ];
    const snippets = buildNeighborSnippets({
      tabs,
      activeTabId: 'a',
      activePath: '/repo/a.ts',
      recentEdits: [{ path: '/repo/b.ts', editedAt: Date.now() }],
    });
    expect(snippets).toHaveLength(1);
    expect(snippets[0].path).toBe('/repo/b.ts');
    expect(snippets[0].source).toBe('recent_edit');
  });
});
