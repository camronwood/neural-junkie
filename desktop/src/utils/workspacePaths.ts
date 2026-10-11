/** Normalize a workspace-relative path (no leading/trailing slashes). */
export function normalizeRelativePath(path: string): string {
  return path.replace(/\\/g, '/').replace(/^\/+/, '').replace(/\/+$/, '');
}

/** Normalize an absolute filesystem path for boundary comparisons. */
export function normalizeAbsolutePath(path: string): string {
  return path.replace(/\\/g, '/').replace(/\/+$/, '') || '/';
}

/**
 * True when `absolutePath` is the workspace root or a descendant.
 * Uses an exact match or `root + '/'` prefix so `/proj` does not match `/project/...`.
 */
export function pathIsUnderWorkspaceRoot(absolutePath: string, root: string): boolean {
  if (!absolutePath || !root) return false;
  const path = normalizeAbsolutePath(absolutePath);
  const base = normalizeAbsolutePath(root);
  return path === base || path.startsWith(`${base}/`);
}

/**
 * Pick the longest matching workspace root for an absolute path.
 * Prefer `/Users/me/project` over `/Users/me/proj` when both would otherwise collide.
 */
export function findLongestMatchingWorkspace<T extends { path: string }>(
  absolutePath: string,
  workspaces: readonly T[],
): T | undefined {
  let best: T | undefined;
  let bestLen = -1;
  for (const workspace of workspaces) {
    if (!pathIsUnderWorkspaceRoot(absolutePath, workspace.path)) continue;
    const len = normalizeAbsolutePath(workspace.path).length;
    if (len > bestLen) {
      best = workspace;
      bestLen = len;
    }
  }
  return best;
}

/** Exact path match, or suffix match at a `/` boundary (never bare endsWith). */
export function pathsMatchExactOrBoundary(candidate: string, target: string): boolean {
  if (!candidate || !target) return false;
  const c = candidate.replace(/\\/g, '/');
  const t = target.replace(/\\/g, '/');
  return c === t || c.endsWith(`/${t}`) || t.endsWith(`/${c}`);
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
