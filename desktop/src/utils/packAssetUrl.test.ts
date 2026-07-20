import { describe, expect, it, vi } from 'vitest';

vi.mock('../config/hubUrl', () => ({
  getHubBaseURL: () => 'http://127.0.0.1:18765',
}));

import { normalizeToolbarChipLabel, packAssetUrl, resolvePackToolbarIconUrl } from './packAssetUrl';

describe('packAssetUrl', () => {
  it('builds hub asset URL with encoded pack id and path', () => {
    expect(packAssetUrl('model-arena', 'icons/chip.svg')).toBe(
      'http://127.0.0.1:18765/api/packs/model-arena/asset?path=icons%2Fchip.svg',
    );
  });

  it('normalizes windows separators in relative path', () => {
    expect(packAssetUrl('arena', 'icons\\chip.svg')).toContain('path=icons%2Fchip.svg');
  });
});

describe('resolvePackToolbarIconUrl', () => {
  it('passes through absolute http(s) icons', () => {
    expect(resolvePackToolbarIconUrl('p', 'https://cdn.example/i.png')).toBe('https://cdn.example/i.png');
  });

  it('maps relative icons through packAssetUrl', () => {
    expect(resolvePackToolbarIconUrl('model-arena', 'ui/icon.svg')).toContain(
      '/api/packs/model-arena/asset?',
    );
  });

  it('returns undefined for empty icon', () => {
    expect(resolvePackToolbarIconUrl('p', undefined)).toBeUndefined();
    expect(resolvePackToolbarIconUrl('p', '  ')).toBeUndefined();
  });
});

describe('normalizeToolbarChipLabel', () => {
  it('uppercases and truncates to 3 chars', () => {
    expect(normalizeToolbarChipLabel('Arena', 'model-arena')).toBe('ARE');
  });

  it('falls back to pack id when label empty', () => {
    expect(normalizeToolbarChipLabel('', 'knowledge-graph')).toBe('KNO');
  });
});
