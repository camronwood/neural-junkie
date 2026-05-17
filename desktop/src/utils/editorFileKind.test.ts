import { describe, expect, it } from 'vitest';
import { isImagePreviewPath, isPngPath, workspaceAbsolutePath } from './editorFileKind';

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
