/** Cursor-style composer behavior: read-only, plan-only, or implement. */
export type ComposerMode = 'ask' | 'plan' | 'agent';

/** Hub metadata mode — export is an explicit structural mode (not phrase-detected). */
export type EffectiveComposerMode = ComposerMode | 'export';

export const COMPOSER_MODE_STORAGE_KEY = 'composer-mode';

export function loadComposerMode(): ComposerMode {
  try {
    if (typeof localStorage === 'undefined') return 'agent';
    const stored = localStorage.getItem(COMPOSER_MODE_STORAGE_KEY);
    if (stored === 'ask' || stored === 'plan' || stored === 'agent') {
      return stored;
    }
    if (stored === 'export') {
      return 'agent';
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
    case 'plan':
      return 'Plan';
    case 'agent':
      return 'Agent';
  }
}

export function composerModePlaceholder(mode: ComposerMode): string {
  switch (mode) {
    case 'ask':
      return 'Ask a question (read-only — no file edits)…';
    case 'plan':
      return 'Plan how to change the code (research first, no file edits until Build)…';
    case 'agent':
      return 'Describe code changes to implement…';
  }
}

export function composerModeTitle(mode: ComposerMode): string {
  switch (mode) {
    case 'ask':
      return 'Read-only — workspace tools, no file proposals';
    case 'plan':
      return 'Research then a structured plan — no file edits until Build';
    case 'agent':
      return 'May propose file changes for your approval';
  }
}

/**
 * Resolves the metadata composer mode sent to the hub.
 * Export is never inferred from natural-language phrases — only the chip / explicit mode.
 */
export function resolveEffectiveComposerMode(
  _message: string,
  composerMode: ComposerMode
): EffectiveComposerMode {
  return composerMode;
}
