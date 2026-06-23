import { describe, expect, it } from 'vitest';
import {
  basenameRelativePath,
  duplicateRelativePath,
  joinRelativePath,
  newItemParentPath,
  normalizeRelativePath,
  parentRelativePath,
  replaceBasename,
} from './workspacePaths';

describe('workspacePaths', () => {
  it('normalizes leading slashes', () => {
    expect(normalizeRelativePath('/foo/bar/')).toBe('foo/bar');
    expect(normalizeRelativePath('README.md')).toBe('README.md');
  });

  it('parent and basename for root files', () => {
    expect(parentRelativePath('README.md')).toBe('');
    expect(basenameRelativePath('README.md')).toBe('README.md');
    expect(replaceBasename('README.md', 'NOTES.md')).toBe('NOTES.md');
  });

  it('joins nested paths', () => {
    expect(joinRelativePath('src', 'lib', 'foo.go')).toBe('src/lib/foo.go');
    expect(newItemParentPath('src/lib/foo.go', false)).toBe('src/lib');
    expect(newItemParentPath('src/lib', true)).toBe('src/lib');
  });

  it('duplicate naming', () => {
    expect(duplicateRelativePath('main.go')).toBe('main copy.go');
    expect(duplicateRelativePath('src/pkg')).toBe('src/pkg copy');
  });
});
