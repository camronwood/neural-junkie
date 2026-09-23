import { describe, expect, it } from 'vitest';
import { SETTINGS_ESSENTIALS_GROUP, ALL_SETTINGS_TABS } from './settingsNav';

describe('settingsNav connectors', () => {
  it('exposes Connectors in Essentials so it is visible without expanding Advanced', () => {
    const item = SETTINGS_ESSENTIALS_GROUP.items.find((i) => i.id === 'connectors');
    expect(item).toEqual({ id: 'connectors', label: 'Connectors' });
  });

  it('includes connectors in ALL_SETTINGS_TABS', () => {
    expect(ALL_SETTINGS_TABS).toContain('connectors');
  });
});
