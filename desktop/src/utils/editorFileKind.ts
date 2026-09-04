const IMAGE_PREVIEW_EXT = /\.(png|jpe?g|gif|webp|bmp|ico|svg)$/i;
const PDF_PREVIEW_EXT = /\.pdf$/i;

/** True when the file can be previewed as an image in the code editor. */
export function isImagePreviewPath(path: string): boolean {
  if (!path) return false;
  const base = path.split(/[/\\]/).pop() ?? path;
  return IMAGE_PREVIEW_EXT.test(base);
}

/** True when the file can be previewed as a PDF in the code editor. */
export function isPdfPreviewPath(path: string): boolean {
  if (!path) return false;
  const base = path.split(/[/\\]/).pop() ?? path;
  return PDF_PREVIEW_EXT.test(base);
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

/** Convert an absolute path to a workspace-relative path, or null if outside the workspace. */
export function workspaceRelativePath(workspacePath: string, absolutePath: string): string | null {
  const root = workspacePath.replace(/[/\\]+$/, '');
  const abs = absolutePath.replace(/[/\\]+$/, '');
  const normRoot = root.replace(/\\/g, '/');
  const normAbs = abs.replace(/\\/g, '/');
  if (normAbs === normRoot) return '';
  const prefix = `${normRoot}/`;
  if (!normAbs.startsWith(prefix)) return null;
  return normAbs.slice(prefix.length);
}
