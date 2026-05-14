import { describe, expect, it } from 'vitest';
import { promoteStandaloneImageFilePaths } from './markdownNormalize';

describe('promoteStandaloneImageFilePaths', () => {
  it('rewrites a lone absolute macOS image path to markdown image', () => {
    const input = 'Saved to:\n\n/Users/me/project/out.png\n\nDone.';
    expect(promoteStandaloneImageFilePaths(input)).toBe(
      'Saved to:\n\n![](/Users/me/project/out.png)\n\nDone.'
    );
  });

  it('rewrites Windows-style absolute paths', () => {
    const input = 'C:\\Users\\me\\x.JPEG';
    expect(promoteStandaloneImageFilePaths(input)).toBe('![](C:\\Users\\me\\x.JPEG)');
  });

  it('does not rewrite paths with spaces (likely prose)', () => {
    const input = '/Users/me/my photos/out.png';
    expect(promoteStandaloneImageFilePaths(input)).toBe(input);
  });

  it('does not rewrite lines containing backticks', () => {
    const input = '`/Users/me/out.png`';
    expect(promoteStandaloneImageFilePaths(input)).toBe(input);
  });

  it('does not rewrite existing markdown images', () => {
    const input = '![](/Users/me/out.png)';
    expect(promoteStandaloneImageFilePaths(input)).toBe(input);
  });
});
