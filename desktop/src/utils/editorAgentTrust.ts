import type { LayoutSettings } from '../stores/settingsStore';

/** Default trust for IDE Agent mode sends (Settings override preserved). */
export function resolveEditorAgentTrust(layout: LayoutSettings): string {
  if (layout.editorAgentTrust) {
    return layout.editorAgentTrust;
  }
  return (layout.editorAgentMode ?? 'agent') === 'agent' ? 'auto_apply_edits' : 'interactive';
}
