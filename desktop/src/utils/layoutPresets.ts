import type { LayoutSettings } from '../stores/settingsStore';

export type LayoutPreset = 'team' | 'ide';
export type IdeChatDock = 'right' | 'bottom';

/** Panel visibility for a layout preset (merged with user toggles when applying preset). */
export function panelsForPreset(preset: LayoutPreset): Partial<LayoutSettings> {
  if (preset === 'ide') {
    return {
      layoutPreset: 'ide',
      filesPanelVisible: true,
      editorPanelVisible: true,
      terminalPanelVisible: false,
      myAgentsPanelVisible: false,
      pendingChangesPanelVisible: false,
    };
  }
  return {
    layoutPreset: 'team',
    filesPanelVisible: false,
    editorPanelVisible: false,
  };
}

export function isIdeLayout(settings: LayoutSettings): boolean {
  return settings.layoutPreset === 'ide';
}

export function layoutPresetLabel(preset: LayoutPreset): string {
  return preset === 'ide' ? 'IDE (project-first)' : 'Team (chat-first)';
}
