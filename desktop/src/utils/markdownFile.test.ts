import { describe, expect, it } from 'vitest';
import { isMarkdownPath } from './markdownFile';

describe('isMarkdownPath', () => {
  it('matches .md and .markdown', () => {
    expect(isMarkdownPath('docs/README.md')).toBe(true);
    expect(isMarkdownPath('notes/changelog.markdown')).toBe(true);
  });

  it('rejects non-markdown paths', () => {
    expect(isMarkdownPath('src/main.go')).toBe(false);
    expect(isMarkdownPath('README')).toBe(false);
    expect(isMarkdownPath('')).toBe(false);
  });
});
