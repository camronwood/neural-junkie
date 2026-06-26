import { describe, expect, it } from 'vitest';
import { ancestorPrefixesForPath } from './refreshFileExplorer';

describe('ancestorPrefixesForPath', () => {
  it('returns all ancestor dirs for nested paths', () => {
    expect(ancestorPrefixesForPath('src/components/New.tsx')).toEqual([
      'src',
      'src/components',
    ]);
  });

  it('returns empty for root-level files', () => {
    expect(ancestorPrefixesForPath('README.md')).toEqual([]);
  });

  it('normalizes leading slashes and backslashes', () => {
    expect(ancestorPrefixesForPath('\\src\\foo\\bar.csv')).toEqual(['src', 'src/foo']);
  });
});
