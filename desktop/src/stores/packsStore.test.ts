import { describe, expect, it } from 'vitest';
import { usePacksStore } from './packsStore';

describe('packsStore', () => {
  it('hasCapability unions hub capabilities and registry', () => {
    usePacksStore.setState({
      capabilities: ['git-rest', 'scan-summary-viewer', 'customer-lab-pack/phoenix-import'],
      capabilityRegistry: [
        {
          id: 'scan-summary-viewer',
          qualified_id: 'customer-lab-pack/scan-summary-viewer',
          kind: 'file-viewer',
          viewer: 'nj.scan-summary',
        },
      ],
      shortIdCollisions: [],
      packs: [],
      catalog: [],
      layoutOwner: 'software-development',
      layoutProfile: 'ide',
    });
    expect(usePacksStore.getState().hasCapability('git-rest')).toBe(true);
    expect(usePacksStore.getState().hasCapability('scan-summary-viewer')).toBe(true);
    expect(usePacksStore.getState().hasCapability('ide-v2')).toBe(false);
  });

  it('hasCapability falls back to enabled pack manifest capabilities', () => {
    usePacksStore.setState({
      capabilities: ['model-arena-launcher'],
      capabilityRegistry: [
        {
          id: 'model-arena-launcher',
          qualified_id: 'model-arena/model-arena-launcher',
          pack_id: 'model-arena',
          kind: 'toolbar-chip',
          ui: { toolbar: { id: 'arena', label: 'Arena' }, modal: 'model-arena' },
        },
      ],
      shortIdCollisions: [],
      packs: [
        {
          id: 'model-arena',
          title: 'Model Arena',
          description: '',
          installed: true,
          enabled: true,
          capabilities: ['model-arena', 'model-arena-workbench', 'model-arena-launcher'],
        },
      ],
      catalog: [],
      layoutOwner: '',
      layoutProfile: 'team',
    });
    expect(usePacksStore.getState().hasCapability('model-arena-workbench')).toBe(true);
    expect(usePacksStore.getState().hasCapability('model-arena-launcher')).toBe(true);
  });

  it('ideEnabled falls back to ide pack row when capability missing', () => {
    usePacksStore.setState({
      capabilities: [],
      capabilityRegistry: [],
      shortIdCollisions: [],
      packs: [
        {
          id: 'ide',
          title: 'IDE',
          description: '',
          installed: true,
          enabled: true,
        },
      ],
      catalog: [],
      layoutOwner: 'ide',
      layoutProfile: 'ide',
    });
    expect(usePacksStore.getState().idePackActive()).toBe(true);
    expect(usePacksStore.getState().ideEnabled()).toBe(true);
    expect(usePacksStore.getState().hasCapability('ide-v2')).toBe(false);
  });

  it('layoutProfile reflects store state', () => {
    usePacksStore.setState({ layoutProfile: 'ide' });
    expect(usePacksStore.getState().layoutProfile).toBe('ide');
  });

  it('exposes dev_linked on pack status', () => {
    usePacksStore.setState({
      packs: [
        {
          id: 'acme-lab',
          title: 'Acme Lab',
          description: '',
          installed: true,
          enabled: false,
          custom: true,
          dev_linked: true,
          dev_source_path: '/tmp/acme-lab',
        },
      ],
      catalog: [],
      capabilities: [],
      capabilityRegistry: [],
      shortIdCollisions: [],
      layoutOwner: '',
      layoutProfile: 'team',
    });
    const pack = usePacksStore.getState().packs[0];
    expect(pack.dev_linked).toBe(true);
    expect(pack.dev_source_path).toBe('/tmp/acme-lab');
  });

  it('mutation-shaped payload with toolbar capability surfaces toolbar actions', () => {
    usePacksStore.getState().applyPacksResponse({
      packs: [
        {
          id: 'model-arena',
          title: 'Model Arena',
          description: '',
          installed: true,
          enabled: true,
        },
      ],
      pack_id: 'model-arena',
      layout_owner: '',
      layout_profile: 'team',
      capabilities: ['model-arena-launcher'],
      capability_registry: [
        {
          id: 'model-arena-launcher',
          qualified_id: 'model-arena/model-arena-launcher',
          pack_id: 'model-arena',
          kind: 'toolbar-chip',
          ui: { toolbar: { id: 'arena', label: 'Arena' }, modal: 'model-arena' },
        },
      ],
      short_id_collisions: [],
    });

    const actions = usePacksStore.getState().getToolbarActions();
    expect(actions).toHaveLength(1);
    expect(actions[0]).toMatchObject({
      id: 'arena',
      label: 'ARE',
      modal: 'model-arena',
      packId: 'model-arena',
      packTitle: 'Model Arena',
    });
    expect(usePacksStore.getState().shortIdCollisions).toEqual([]);
  });
});
