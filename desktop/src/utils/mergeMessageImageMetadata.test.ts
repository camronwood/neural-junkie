import { describe, expect, it } from 'vitest';
import { mergeMessageImageMetadata } from './mergeMessageImageMetadata';

describe('mergeMessageImageMetadata', () => {
  it('preserves inline data when refetch redacts without path', () => {
    const existing = {
      generated_image: { mime: 'image/png', data: 'YWJj' },
    };
    const incoming = {
      generated_image: { mime: 'image/png', data_redacted: true, approx_bytes: 4 },
    };
    const merged = mergeMessageImageMetadata(existing, incoming);
    const g = merged?.generated_image as Record<string, unknown>;
    expect(g.data).toBe('YWJj');
    expect(g.data_redacted).toBeUndefined();
  });

  it('uses path from redacted refetch when present', () => {
    const existing = {
      generated_image: { mime: 'image/png', data: 'YWJj' },
    };
    const incoming = {
      generated_image: {
        mime: 'image/png',
        data_redacted: true,
        path: '/Users/me/.neural-junkie/generated-images/x.png',
      },
    };
    const merged = mergeMessageImageMetadata(existing, incoming);
    const g = merged?.generated_image as Record<string, unknown>;
    expect(g.path).toBe('/Users/me/.neural-junkie/generated-images/x.png');
    expect(g.data_redacted).toBe(true);
  });
});
