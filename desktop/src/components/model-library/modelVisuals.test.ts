import { describe, expect, it } from 'vitest';
import { defaultStatusLabel, modelVisualForItem } from './modelVisuals';

describe('modelVisualForItem', () => {
  it('uses icon_key gradient when present', () => {
    const v = modelVisualForItem({ title: 'Qwen 2.5', iconKey: 'qwen', tags: ['general'] });
    expect(v.categoryLabel).toBe('Qwen');
    expect(v.monogram).toBe('Q2');
  });

  it('falls back to tag-based gradient', () => {
    const v = modelVisualForItem({ title: 'Code Model', tags: ['code'] });
    expect(v.categoryLabel).toBe('Code');
    expect(v.gradientFrom).toBe('#059669');
  });

  it('uses default gradient when no keys match', () => {
    const v = modelVisualForItem({ title: 'Unknown', tags: ['foo'] });
    expect(v.categoryLabel).toBe('Model');
  });
});

describe('defaultStatusLabel', () => {
  it('maps known statuses', () => {
    expect(defaultStatusLabel('installed')).toBe('Installed');
    expect(defaultStatusLabel('on_disk')).toBe('On disk');
    expect(defaultStatusLabel('cloud')).toBe('Cloud');
  });
});
