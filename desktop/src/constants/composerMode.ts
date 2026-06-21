import {
  hasFileExportSignals,
  hasPriorReferenceExportSignals,
  hasCombinedContentDeliveryExport,
} from '../utils/implementationContinuation';

/** Cursor-style composer behavior: read-only, plan-only, implement, or export-to-file. */
export type ComposerMode = 'ask' | 'plan' | 'agent' | 'export';

export const COMPOSER_MODE_STORAGE_KEY = 'composer-mode';

export function loadComposerMode(): ComposerMode {
  try {
    if (typeof localStorage === 'undefined') return 'agent';
    const stored = localStorage.getItem(COMPOSER_MODE_STORAGE_KEY);
    if (stored === 'ask' || stored === 'plan' || stored === 'agent' || stored === 'export') {
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
    case 'plan':
      return 'Plan';
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
    case 'plan':
      return 'Outline an approach and steps (no file edits yet)…';
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
    case 'plan':
      return 'Structured plan only — no file edits or implementation session';
    case 'agent':
      return 'May propose file changes for your approval';
    case 'export':
      return 'Export content to a workspace file (.md, etc.)';
  }
}

/**
 * Resolves the metadata composer mode sent to the hub. Export is applied
 * automatically when the message looks like a file export — the UI chip can
 * stay on Agent/Ask.
 */
export function resolveEffectiveComposerMode(
  message: string,
  composerMode: ComposerMode
): ComposerMode {
  if (composerMode === 'ask') return 'ask';
  if (composerMode === 'plan') return 'plan';
  if (hasCombinedContentDeliveryExport(message)) {
    return composerMode === 'export' ? 'agent' : composerMode;
  }
  if (hasFileExportSignals(message) || hasPriorReferenceExportSignals(message)) {
    return 'export';
  }
  return composerMode;
}
