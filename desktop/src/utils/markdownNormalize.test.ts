import { describe, it, expect } from 'vitest';
import { normalizeAgentMessageMarkdown, normalizeMarkdownFences } from './markdownNormalize';

describe('normalizeMarkdownFences', () => {
  it('fixes two-backtick bash opener and single-backtick close', () => {
    const raw = [
      'Some intro.',
      '``bash',
      'mv /Users/camronwood/development/Phoenix/IWantToSeeThisWork /Users/camronwood/development/',
      '`',
      'After.',
    ].join('\n');
    const got = normalizeMarkdownFences(raw);
    expect(got).toContain('```bash');
    expect(got).toContain('mv /Users/camronwood/development/Phoenix');
    expect(got.split('\n').some(l => l.trim() === '```')).toBe(true);
  });
});

describe('normalizeAgentMessageMarkdown', () => {
  it('combines inline opener fix', () => {
    const raw = 'Run:\n```bash mv ./a ./b\n```';
    expect(normalizeAgentMessageMarkdown(raw)).toContain('```bash\nmv ./a ./b');
  });
});
