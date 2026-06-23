/** Normalize a workspace-relative path (no leading/trailing slashes). */
export function normalizeRelativePath(path: string): string {
  return path.replace(/\\/g, '/').replace(/^\/+/, '').replace(/\/+$/, '');
}

/** Parent directory of a relative path; empty string for root-level items. */
export function parentRelativePath(path: string): string {
  const norm = normalizeRelativePath(path);
  const idx = norm.lastIndexOf('/');
  if (idx <= 0) return '';
  return norm.slice(0, idx);
}

/** Final path segment (file or folder name). */
export function basenameRelativePath(path: string): string {
  const norm = normalizeRelativePath(path);
  const idx = norm.lastIndexOf('/');
  return idx >= 0 ? norm.slice(idx + 1) : norm;
}

/** Join workspace-relative path segments. */
export function joinRelativePath(...parts: string[]): string {
  const joined = parts
    .flatMap((p) => normalizeRelativePath(p).split('/').filter(Boolean))
    .join('/');
  return normalizeRelativePath(joined);
}

/** Replace the final segment of a relative path. */
export function replaceBasename(path: string, newName: string): string {
  const parent = parentRelativePath(path);
  const name = newName.trim().replace(/[/\\]+/g, '');
  if (!name) return normalizeRelativePath(path);
  return parent ? joinRelativePath(parent, name) : name;
}

/** Default sibling path for duplicate: `file copy.ext` or `folder copy`. */
export function duplicateRelativePath(path: string): string {
  const base = basenameRelativePath(path);
  const parent = parentRelativePath(path);
  const dot = base.lastIndexOf('.');
  const copyName =
    dot > 0 ? `${base.slice(0, dot)} copy${base.slice(dot)}` : `${base} copy`;
  return parent ? joinRelativePath(parent, copyName) : copyName;
}

/** Target directory for new items from context on a file or folder. */
export function newItemParentPath(contextPath: string, isDir: boolean): string {
  const norm = normalizeRelativePath(contextPath);
  if (!norm) return '';
  return isDir ? norm : parentRelativePath(norm);
}
