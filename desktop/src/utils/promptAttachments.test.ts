import { describe, expect, it } from 'vitest';
import {
  attachmentsFromFileList,
  inferLanguageFromPath,
  isBinaryPath,
  mergePromptAttachments,
} from './promptAttachments';

describe('promptAttachments', () => {
  it('infers language from path', () => {
    expect(inferLanguageFromPath('/foo/bar/main.go')).toBe('go');
    expect(inferLanguageFromPath('README.md')).toBe('markdown');
  });

  it('skips binary extensions', () => {
    expect(isBinaryPath('photo.png')).toBe(true);
    expect(isBinaryPath('src/main.rs')).toBe(false);
  });

  it('merges with caps', () => {
    const merged = mergePromptAttachments([], [
      { path: 'a.txt', language: 'text', content: 'hello' },
      { path: 'a.txt', language: 'text', content: 'dup' },
      { path: 'b.txt', language: 'text', content: 'world' },
    ]);
    expect(merged).toHaveLength(2);
    expect(merged[0].path).toBe('a.txt');
  });

  it('reads text from FileList', async () => {
    const file = new File(['package main\n'], 'main.go', { type: 'text/plain' });
    const out = await attachmentsFromFileList([file], []);
    expect(out).toHaveLength(1);
    expect(out[0].path).toBe('main.go');
    expect(out[0].language).toBe('go');
    expect(out[0].content).toContain('package main');
  });
});
