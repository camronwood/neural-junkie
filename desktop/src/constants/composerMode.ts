/** Cursor-style composer behavior: read-only, implement, or export-to-file. */
export type ComposerMode = 'ask' | 'agent' | 'export';

export const COMPOSER_MODE_STORAGE_KEY = 'composer-mode';

export function loadComposerMode(): ComposerMode {
  try {
    if (typeof localStorage === 'undefined') return 'agent';
    const stored = localStorage.getItem(COMPOSER_MODE_STORAGE_KEY);
    if (stored === 'ask' || stored === 'agent' || stored === 'export') {
      return stored;
    }
  } catch {
    /* ignore */
  }
  return 'agent';
}

export function composerModeLabel(mode: ComposerMode): string {
  switch (mode) {
    case 'ask':
      return 'Ask';
    case 'agent':
      return 'Agent';
    case 'export':
      return 'Export';
  }
}

export function composerModePlaceholder(mode: ComposerMode): string {
  switch (mode) {
    case 'ask':
      return 'Ask a question (read-only — no file edits)…';
    case 'agent':
      return 'Describe code changes to implement…';
    case 'export':
      return 'Describe what to write to a file (uses prior replies + workspace)…';
  }
}

export function composerModeTitle(mode: ComposerMode): string {
  switch (mode) {
    case 'ask':
      return 'Read-only — workspace tools, no file proposals';
    case 'agent':
      return 'May propose file changes for your approval';
    case 'export':
      return 'Export content to a workspace file (.md, etc.)';
  }
}
