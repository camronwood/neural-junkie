const IMAGE_PREVIEW_EXT = /\.(png|jpe?g|gif|webp|bmp|ico|svg)$/i;

/** True when the file can be previewed as an image in the code editor. */
export function isImagePreviewPath(path: string): boolean {
  if (!path) return false;
  const base = path.split(/[/\\]/).pop() ?? path;
  return IMAGE_PREVIEW_EXT.test(base);
}

/** @deprecated Use isImagePreviewPath */
export function isPngPath(path: string): boolean {
  return isImagePreviewPath(path);
}

/** Join workspace root with a relative file path (no trailing slash on root). */
export function workspaceAbsolutePath(workspacePath: string, relativePath: string): string {
  const root = workspacePath.endsWith('/') || workspacePath.endsWith('\\')
    ? workspacePath.slice(0, -1)
    : workspacePath;
  const rel = relativePath.replace(/^[/\\]+/, '');
  return `${root}/${rel}`;
}
