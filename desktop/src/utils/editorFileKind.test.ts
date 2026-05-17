import { describe, expect, it } from 'vitest';
import { isPngPath, workspaceAbsolutePath } from './editorFileKind';

describe('isPngPath', () => {
  it('matches .png case-insensitively', () => {
    expect(isPngPath('foo.png')).toBe(true);
    expect(isPngPath('foo.PNG')).toBe(true);
    expect(isPngPath('assets/out/foo.Png')).toBe(true);
  });

  it('rejects non-png extensions', () => {
    expect(isPngPath('foo.jpg')).toBe(false);
    expect(isPngPath('foo.png.bak')).toBe(false);
    expect(isPngPath('')).toBe(false);
  });
});

describe('workspaceAbsolutePath', () => {
  it('joins without duplicate slashes when root has trailing slash', () => {
    expect(workspaceAbsolutePath('/Users/me/proj/', 'assets/a.png')).toBe(
      '/Users/me/proj/assets/a.png'
    );
  });

  it('joins when root has no trailing slash', () => {
    expect(workspaceAbsolutePath('/Users/me/proj', 'assets/a.png')).toBe(
      '/Users/me/proj/assets/a.png'
    );
  });

  it('strips leading slash from relative path', () => {
    expect(workspaceAbsolutePath('/ws', '/nested/file.png')).toBe('/ws/nested/file.png');
  });
});
