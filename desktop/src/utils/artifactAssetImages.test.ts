import { describe, expect, it } from 'vitest';
import { parseArtifactAssetSrc } from './artifactAssetImages';

describe('parseArtifactAssetSrc', () => {
  it('parses absolute api asset paths', () => {
    expect(parseArtifactAssetSrc('/api/artifacts/abc123/assets/embed-1.png')).toEqual({
      artifactId: 'abc123',
      name: 'embed-1.png',
    });
  });

  it('parses relative api asset paths', () => {
    expect(parseArtifactAssetSrc('api/artifacts/abc123/assets/photo.jpg')).toEqual({
      artifactId: 'abc123',
      name: 'photo.jpg',
    });
  });

  it('uses fallback artifact id for bare asset names', () => {
    expect(parseArtifactAssetSrc('embed-1.png', 'art-9')).toEqual({
      artifactId: 'art-9',
      name: 'embed-1.png',
    });
  });

  it('rejects unrelated urls', () => {
    expect(parseArtifactAssetSrc('https://example.com/x.png')).toBeNull();
  });
});
