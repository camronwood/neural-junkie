/** True when path ends with .png (case-insensitive). */
export function isPngPath(path: string): boolean {
  if (!path) return false;
  const base = path.split(/[/\\]/).pop() ?? path;
  return /\.png$/i.test(base);
}

/** Join workspace root with a relative file path (no trailing slash on root). */
export function workspaceAbsolutePath(workspacePath: string, relativePath: string): string {
  const root = workspacePath.endsWith('/') || workspacePath.endsWith('\\')
    ? workspacePath.slice(0, -1)
    : workspacePath;
  const rel = relativePath.replace(/^[/\\]+/, '');
  return `${root}/${rel}`;
}
