import { describe, expect, it } from 'vitest';
import { toolbarActionsFromRegistry, hasPackCapability } from './packCapabilityRegistry';
import type { ResolvedCapability } from '../api/chatAPI';

describe('toolbarActionsFromRegistry', () => {
  it('returns non-platform capabilities with toolbar ui', () => {
    const registry: ResolvedCapability[] = [
      {
        id: 'customer-pack',
        qualified_id: 'customer-pack',
        platform: true,
        ui: { toolbar: { id: 'skip', label: 'X' } },
      },
      {
        id: 'phoenix-import',
        qualified_id: 'brightest-bio-lab/phoenix-import',
        pack_id: 'brightest-bio-lab',
        ui: {
          toolbar: { id: 'phx', label: 'PHX' },
          modal: 'phoenix-import',
        },
      },
    ];
    const actions = toolbarActionsFromRegistry(registry, [
      { id: 'brightest-bio-lab', title: 'Brightest Bio Lab' },
    ]);
    expect(actions).toHaveLength(1);
    expect(actions[0]).toMatchObject({
      id: 'phx',
      label: 'PHX',
      modal: 'phoenix-import',
      packId: 'brightest-bio-lab',
      packTitle: 'Brightest Bio Lab',
    });
  });

  it('prefers icon over label text', () => {
    const registry: ResolvedCapability[] = [
      {
        id: 'pack-toolbar',
        qualified_id: 'acme-lab/pack-toolbar',
        pack_id: 'acme-lab',
        ui: {
          toolbar: { id: 'acme', label: 'ACM', icon: 'assets/icons/chip.png' },
        },
      },
    ];
    const actions = toolbarActionsFromRegistry(registry, [{ id: 'acme-lab', title: 'Acme Lab' }]);
    expect(actions[0].label).toBe('');
    expect(actions[0].iconUrl).toContain('/api/packs/acme-lab/asset');
    expect(actions[0].iconUrl).toContain('assets%2Ficons%2Fchip.png');
  });
});

describe('hasPackCapability', () => {
  it('falls back to enabled pack manifest capabilities', () => {
    const packs = [
      {
        id: 'model-arena',
        enabled: true,
        capabilities: ['model-arena', 'model-arena-workbench', 'model-arena-launcher'],
      },
    ];
    expect(hasPackCapability([], [], packs, 'model-arena-workbench')).toBe(true);
    expect(hasPackCapability([], [{ id: 'model-arena-launcher', qualified_id: 'model-arena/model-arena-launcher' }], packs, 'model-arena-launcher')).toBe(true);
  });
});
