import { describe, expect, it } from 'vitest';
import { usePacksStore } from './packsStore';

describe('packsStore', () => {
  it('hasCapability unions hub capabilities', () => {
    usePacksStore.setState({
      capabilities: ['git-rest', 'scan-summary-viewer'],
      packs: [],
      catalog: [],
      layoutOwner: 'software-development',
      layoutProfile: 'ide',
    });
    expect(usePacksStore.getState().hasCapability('git-rest')).toBe(true);
    expect(usePacksStore.getState().hasCapability('scan-summary-viewer')).toBe(true);
    expect(usePacksStore.getState().hasCapability('ide-v2')).toBe(false);
  });

  it('softwareDevelopmentEnabled falls back to pack row when capability missing', () => {
    usePacksStore.setState({
      capabilities: [],
      packs: [
        {
          id: 'software-development',
          title: 'Software development',
          description: '',
          installed: true,
          enabled: true,
        },
      ],
      catalog: [],
      layoutOwner: 'software-development',
      layoutProfile: 'ide',
    });
    expect(usePacksStore.getState().softwareDevelopmentPackActive()).toBe(true);
    expect(usePacksStore.getState().softwareDevelopmentEnabled()).toBe(true);
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
      layoutOwner: '',
      layoutProfile: 'team',
    });
    const pack = usePacksStore.getState().packs[0];
    expect(pack.dev_linked).toBe(true);
    expect(pack.dev_source_path).toBe('/tmp/acme-lab');
  });
});
