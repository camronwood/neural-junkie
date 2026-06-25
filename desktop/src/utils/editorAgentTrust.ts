import type { LayoutSettings } from '../stores/settingsStore';
import type { ComposerMode } from '../constants/composerMode';

/** Default trust for IDE Agent mode sends (Settings override preserved). */
export function resolveEditorAgentTrust(
  layout: LayoutSettings,
  composerMode?: ComposerMode
): string {
  const mode = composerMode ?? layout.editorAgentMode ?? 'agent';
  if (mode === 'ask' || mode === 'plan') {
    return 'interactive';
  }
  if (mode === 'agent' || mode === 'export') {
    return 'auto_apply_edits';
  }
  if (layout.editorAgentTrust) {
    return layout.editorAgentTrust;
  }
  return 'auto_apply_edits';
}
