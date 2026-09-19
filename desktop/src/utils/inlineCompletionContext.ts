import type { EditorTab, RecentEditorEdit } from '../stores/editorStore';

export type NeighborSnippet = {
  path: string;
  content: string;
  source?: string;
};

const MAX_PREFIX_CHARS = 8000;
const MAX_SUFFIX_CHARS = 4000;
const CONTEXT_LINES_BEFORE = 80;
const CONTEXT_LINES_AFTER = 80;
const MAX_NEIGHBOR_TABS = 4;
const MAX_NEIGHBOR_SNIPPET_CHARS = 1200;

function takeTail(s: string, max: number): string {
  if (s.length <= max) return s;
  return `…\n${s.slice(s.length - max)}`;
}

function takeHead(s: string, max: number): string {
  if (s.length <= max) return s;
  return `${s.slice(0, max)}\n…`;
}

/** Build FIM prefix/suffix + local window context from Monaco model text. */
export function buildInlineCompletionLocalContext(
  fullText: string,
  offset: number
): { prefix: string; suffix: string; context: string } {
  const safeOffset = Math.max(0, Math.min(offset, fullText.length));
  const prefix = takeTail(fullText.slice(0, safeOffset), MAX_PREFIX_CHARS);
  const suffix = takeHead(fullText.slice(safeOffset), MAX_SUFFIX_CHARS);

  const lines = fullText.split('\n');
  let lineStart = 0;
  let lineIndex = 0;
  for (let i = 0; i < lines.length; i++) {
    const next = lineStart + lines[i].length + (i < lines.length - 1 ? 1 : 0);
    if (safeOffset <= next || i === lines.length - 1) {
      lineIndex = i;
      break;
    }
    lineStart = next;
  }
  const ctxStart = Math.max(0, lineIndex - CONTEXT_LINES_BEFORE);
  const ctxEnd = Math.min(lines.length, lineIndex + CONTEXT_LINES_AFTER + 1);
  const context = lines.slice(ctxStart, ctxEnd).join('\n');

  return { prefix, suffix, context };
}

/** Neighbor snippets from other open tabs + recent-edit locality. */
export function buildNeighborSnippets(params: {
  tabs: EditorTab[];
  activeTabId: string | null;
  activePath?: string;
  recentEdits: RecentEditorEdit[];
}): NeighborSnippet[] {
  const { tabs, activeTabId, activePath, recentEdits } = params;
  const out: NeighborSnippet[] = [];
  const seen = new Set<string>();
  if (activePath) seen.add(activePath);

  const recentPaths = new Set(recentEdits.slice(0, 8).map((e) => e.path));

  for (const tab of tabs) {
    if (out.length >= MAX_NEIGHBOR_TABS) break;
    if (!tab.path || tab.id === activeTabId || seen.has(tab.path)) continue;
    if (tab.viewMode && tab.viewMode !== 'text') continue;
    const content = (tab.content ?? '').trim();
    if (content.length < 8) continue;
    seen.add(tab.path);
    const source = recentPaths.has(tab.path) ? 'recent_edit' : 'open_tab';
    out.push({
      path: tab.path,
      content: takeHead(content, MAX_NEIGHBOR_SNIPPET_CHARS),
      source,
    });
  }

  // Prefer recent-edit tabs that may not be open (path-only locality marker).
  for (const edit of recentEdits) {
    if (out.length >= MAX_NEIGHBOR_TABS) break;
    if (!edit.path || seen.has(edit.path)) continue;
    const open = tabs.find((t) => t.path === edit.path && (!t.viewMode || t.viewMode === 'text'));
    if (!open?.content) continue;
    seen.add(edit.path);
    out.push({
      path: edit.path,
      content: takeHead(open.content, MAX_NEIGHBOR_SNIPPET_CHARS),
      source: 'recent_edit',
    });
  }

  return out;
}
