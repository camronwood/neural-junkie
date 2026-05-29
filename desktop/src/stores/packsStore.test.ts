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

  it('layoutProfile reflects store state', () => {
    usePacksStore.setState({ layoutProfile: 'ide' });
    expect(usePacksStore.getState().layoutProfile).toBe('ide');
  });
});
