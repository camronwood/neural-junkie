import { describe, expect, it } from 'vitest';
import {
  basenameRelativePath,
  duplicateRelativePath,
  findLongestMatchingWorkspace,
  joinRelativePath,
  newItemParentPath,
  normalizeRelativePath,
  parentRelativePath,
  pathIsUnderWorkspaceRoot,
  pathsMatchExactOrBoundary,
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

  it('pathIsUnderWorkspaceRoot rejects prefix collisions', () => {
    expect(pathIsUnderWorkspaceRoot('/Users/me/project/src/a.ts', '/Users/me/proj')).toBe(false);
    expect(pathIsUnderWorkspaceRoot('/Users/me/project/src/a.ts', '/Users/me/project')).toBe(true);
    expect(pathIsUnderWorkspaceRoot('/Users/me/project', '/Users/me/project')).toBe(true);
  });

  it('findLongestMatchingWorkspace prefers the longest root', () => {
    const workspaces = [{ path: '/Users/me/proj' }, { path: '/Users/me/project' }];
    expect(findLongestMatchingWorkspace('/Users/me/project/src/a.ts', workspaces)?.path).toBe(
      '/Users/me/project',
    );
    expect(findLongestMatchingWorkspace('/Users/me/proj/file.ts', workspaces)?.path).toBe(
      '/Users/me/proj',
    );
    expect(findLongestMatchingWorkspace('/Users/me/other/file.ts', workspaces)).toBeUndefined();
  });

  it('pathsMatchExactOrBoundary requires a path segment boundary', () => {
    expect(pathsMatchExactOrBoundary('/workspace/src/app.ts', '/workspace/src/app.ts')).toBe(true);
    expect(pathsMatchExactOrBoundary('/workspace/src/app.ts', 'src/app.ts')).toBe(true);
    expect(pathsMatchExactOrBoundary('/workspace/foo/bar.ts', 'bar.ts')).toBe(true);
    expect(pathsMatchExactOrBoundary('/workspace/foobar.ts', 'bar.ts')).toBe(false);
    expect(pathsMatchExactOrBoundary('/workspace/src/app.ts', 'src/app')).toBe(false);
  });
});
