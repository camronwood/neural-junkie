/**
 * Dev eval for chat response formatting (no live hub required).
 * Mirrors the manual DM checklist: bullets, numbered, table, control prose, streaming detector.
 */
import { describe, expect, it } from 'vitest';
import {
  looksLikeBlockMarkdown,
  normalizeProseMarkdownBlocks,
} from './markdownNormalize';
import { renderChatMarkdown } from './markdownRenderer';

describe('chat formatting DM eval', () => {
  it('renders agent bullet lists', () => {
    const input =
      'Here are five tips:\n\n- Tip one\n- Tip two\n- Tip three\n- Tip four\n- Tip five';
    expect(looksLikeBlockMarkdown(input)).toBe(true);
    const html = renderChatMarkdown(input);
    expect(html).toContain('<ul');
    expect(html).toContain('<li');
  });

  it('normalizes glued asterisk bullets', () => {
    const normalized = normalizeProseMarkdownBlocks(
      'Tips: * First tip * Second tip * Third tip'
    );
    const html = renderChatMarkdown(normalized);
    expect(html).toContain('<ul');
    expect(html).toContain('<li');
  });

  it('renders numbered steps', () => {
    const input = 'Do this:\n\n1. First\n2. Second\n3. Third';
    expect(looksLikeBlockMarkdown(input)).toBe(true);
    const html = renderChatMarkdown(input);
    expect(html).toContain('<ol');
    expect(html).toContain('<li');
  });

  it('renders markdown tables', () => {
    const input =
      '| Name | Status |\n| --- | --- |\n| alpha | ok |\n| beta | pending |';
    expect(looksLikeBlockMarkdown(input)).toBe(true);
    const html = renderChatMarkdown(input);
    expect(html).toContain('<table');
    expect(html).toContain('<th');
  });

  it('detects blockquotes for streaming GFM', () => {
    const input = '> Important note about the deploy';
    expect(looksLikeBlockMarkdown(input)).toBe(true);
    expect(renderChatMarkdown(input)).toContain('<blockquote');
  });

  it('keeps control prose as plain paragraphs', () => {
    const input =
      'TLS encrypts traffic between client and server using certificates.';
    expect(looksLikeBlockMarkdown(input)).toBe(false);
    const html = renderChatMarkdown(input);
    expect(html).not.toContain('<ul');
    expect(html).not.toContain('<ol');
    expect(html).not.toContain('<table');
  });

  it('breaks lowercase dash lists after an intro', () => {
    const normalized = normalizeProseMarkdownBlocks(
      'Steps: - first do this - second do that - third finish'
    );
    const html = renderChatMarkdown(normalized);
    expect(html).toContain('<ul');
    expect(html).toContain('<li');
  });
});
