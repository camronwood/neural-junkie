import { describe, expect, it } from 'vitest';
import { isIdeLayout, panelsForPreset } from './layoutPresets';
import type { LayoutSettings } from '../stores/settingsStore';

describe('layoutPresets', () => {
  it('panelsForPreset ide enables editor panels', () => {
    const p = panelsForPreset('ide');
    expect(p.layoutPreset).toBe('ide');
    expect(p.filesPanelVisible).toBe(true);
    expect(p.editorPanelVisible).toBe(true);
    expect(p.pendingChangesPanelVisible).toBe(false);
  });

  it('isIdeLayout detects preset', () => {
    const s = { layoutPreset: 'ide' } as LayoutSettings;
    expect(isIdeLayout(s)).toBe(true);
  });
});
