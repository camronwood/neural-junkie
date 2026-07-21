import { describe, expect, it } from 'vitest';
import {
  globMatch,
  hasPackCapability,
  matchArtifactRenderer,
  toolbarActionsFromRegistry,
} from './packCapabilityRegistry';
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

  it('keeps label as fallback when icon URL is set', () => {
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
    expect(actions[0].label).toBe('ACM');
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

describe('artifact renderer matching', () => {
  it('supports recursive brace globs', () => {
    expect(globMatch('**/*.{pdb,cif,mmcif}', 'data/structure/model.cif')).toBe(true);
    expect(globMatch('**/*.{pdb,cif,mmcif}', 'data/structure/model.csv')).toBe(false);
  });

  it('requires compatible media and schema versions', () => {
    const registry: ResolvedCapability[] = [{
      id: 'assay-report',
      qualified_id: 'lab/assay-report',
      kind: 'artifact-renderer',
      renderer: 'nj.chart',
      media_types: ['application/vnd.neural-junkie.chart+json'],
      schema_version_min: 1,
      schema_version_max: 2,
    }];
    expect(matchArtifactRenderer(registry, 'application/vnd.neural-junkie.chart+json', 2)?.renderer).toBe('nj.chart');
    expect(matchArtifactRenderer(registry, 'application/vnd.neural-junkie.chart+json', 3)).toBeUndefined();
  });
});
