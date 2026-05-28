import { describe, expect, it } from 'vitest';
import { isImagePreviewPath, isPngPath, workspaceAbsolutePath, workspaceRelativePath } from './editorFileKind';

describe('isImagePreviewPath', () => {
  it('matches common image extensions', () => {
    expect(isImagePreviewPath('foo.png')).toBe(true);
    expect(isImagePreviewPath('foo.PNG')).toBe(true);
    expect(isImagePreviewPath('photo.jpeg')).toBe(true);
    expect(isImagePreviewPath('icon.svg')).toBe(true);
  });

  it('rejects non-image extensions', () => {
    expect(isImagePreviewPath('foo.rs')).toBe(false);
    expect(isImagePreviewPath('foo.png.bak')).toBe(false);
    expect(isImagePreviewPath('')).toBe(false);
  });
});

describe('isPngPath', () => {
  it('aliases isImagePreviewPath for png', () => {
    expect(isPngPath('foo.png')).toBe(true);
    expect(isPngPath('foo.jpg')).toBe(true);
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

describe('workspaceRelativePath', () => {
  it('returns empty string for workspace root', () => {
    expect(workspaceRelativePath('/Users/me/summary-test', '/Users/me/summary-test')).toBe('');
  });

  it('returns relative path for nested folder', () => {
    expect(
      workspaceRelativePath('/Users/me/summary-test', '/Users/me/summary-test/scan-export')
    ).toBe('scan-export');
  });

  it('returns null for paths outside workspace', () => {
    expect(workspaceRelativePath('/Users/me/summary-test', '/Users/me/other/scan-export')).toBeNull();
  });
});
