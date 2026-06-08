import { describe, expect, it } from 'vitest';
import {
  looksLikeBlockMarkdown,
  normalizeProseMarkdownBlocks,
  promoteStandaloneImageFilePaths,
} from './markdownNormalize';
import { renderChatMarkdown } from './markdownRenderer';

describe('normalizeProseMarkdownBlocks', () => {
  it('breaks inline horizontal rules and headings', () => {
    const input =
      'professional and engaging manner. --- ### Introducing Neural Junkie: Revolutionizing Collaboration';
    const out = normalizeProseMarkdownBlocks(input);
    expect(out).toContain('\n---\n');
    expect(out).toContain('\n### Introducing Neural Junkie');
  });

  it('breaks numbered lists and sub-bullets', () => {
    const input = '#### Key Features 1. Real-Time Collaboration: - Integrated Workspaces: Neural Junkie';
    const out = normalizeProseMarkdownBlocks(input);
    expect(out).toContain('#### Key Features');
    expect(out).toContain('\n1. Real-Time Collaboration');
    expect(out).toContain(':\n\n- Integrated Workspaces');
  });

  it('renders article-style markdown to headings and lists', () => {
    const normalized = normalizeProseMarkdownBlocks(
      'Intro text. --- ### Title Here #### Section 1. First item: - Detail one 2. Second item: - Detail two'
    );
    const html = renderChatMarkdown(normalized);
    expect(html).toContain('<h3');
    expect(html).toContain('<h4');
    expect(html).toContain('<ol');
    expect(html).toContain('<ul');
    expect(html).toContain('<hr');
  });

  it('does not alter fenced code blocks', () => {
    const input = '```go\nfmt.Println(" --- ### not a header ")\n```';
    expect(normalizeProseMarkdownBlocks(input)).toBe(input);
  });
});

describe('looksLikeBlockMarkdown', () => {
  it('detects headings and lists', () => {
    expect(looksLikeBlockMarkdown('### Hello')).toBe(true);
    expect(looksLikeBlockMarkdown('plain chat reply')).toBe(false);
  });
});

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
