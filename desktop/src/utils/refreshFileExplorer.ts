import { useFileExplorerStore } from '../stores/fileExplorerStore';

/** All directory prefixes along a workspace-relative file path (e.g. src/components/Foo.tsx → src, src/components). */
export function ancestorPrefixesForPath(relativePath: string): string[] {
  const normalized = relativePath.replace(/\\/g, '/').replace(/^\/+/, '').trim();
  const dirParts = normalized.includes('/')
    ? normalized.split('/').slice(0, -1).filter(Boolean)
    : [];
  const prefixes: string[] = [];
  let acc = '';
  for (const part of dirParts) {
    acc = acc ? `${acc}/${part}` : part;
    prefixes.push(acc);
  }
  return prefixes;
}

/** Refresh file explorer directories that contain the given workspace-relative paths. */
export async function refreshFileExplorerForPaths(
  workspaceId: string,
  relativePaths: string[]
): Promise<void> {
  const trimmed = relativePaths
    .map((p) => p.replace(/\\/g, '/').replace(/^\/+/, '').trim())
    .filter(Boolean);
  if (trimmed.length === 0) {
    await useFileExplorerStore.getState().loadFiles(workspaceId, '/');
    return;
  }

  const prefixes = new Set<string>();
  for (const rel of trimmed) {
    for (const p of ancestorPrefixesForPath(rel)) {
      prefixes.add(p);
    }
  }

  const { loadFiles, expandedPaths } = useFileExplorerStore.getState();
  // Reload root + every ancestor dir.
  await loadFiles(workspaceId, '/');
  const sorted = [...prefixes].sort((a, b) => a.localeCompare(b));
  for (const p of sorted) {
    await loadFiles(workspaceId, p);
  }

  // Re-hydrate expanded folders along the changed paths that were not in the prefix set.
  for (const [path, expanded] of Object.entries(expandedPaths)) {
    if (!expanded || path === '/' || prefixes.has(path)) continue;
    const touchesChange = trimmed.some((rel) => rel === path || rel.startsWith(`${path}/`));
    if (touchesChange) {
      await loadFiles(workspaceId, path);
    }
  }

  // Refresh all other expanded folders (cap to avoid hub call storms).
  const expandedList = Object.entries(expandedPaths)
    .filter(([, open]) => open)
    .map(([path]) => path)
    .filter((path) => path && path !== '/' && !prefixes.has(path))
    .sort((a, b) => a.localeCompare(b));
  const maxExpanded = 24;
  for (let i = 0; i < expandedList.length && i < maxExpanded; i++) {
    await loadFiles(workspaceId, expandedList[i]);
  }
}

/** Extract workspace-relative paths from a file_change message proposal. */
export function fileChangeProposalPaths(message: {
  metadata?: Record<string, unknown>;
}): string[] {
  const paths: string[] = [];
  const push = (value: unknown) => {
    if (typeof value === 'string' && value.trim()) {
      paths.push(value.trim());
    }
  };

  const card = message.metadata?.change_proposal;
  if (card && typeof card === 'object') {
    const c = card as Record<string, unknown>;
    push(c.file_path);
    push(c.new_path);
    if (Array.isArray(c.paths)) {
      for (const p of c.paths) push(p);
    }
  }

  const raw = message.metadata?.file_change_proposal;
  if (raw && typeof raw === 'object') {
    const proposal = raw as Record<string, unknown>;
    push(proposal.file_path);
    push(proposal.new_path);
  }

  const batch = message.metadata?.file_change_batch_proposal;
  if (batch && typeof batch === 'object') {
    const b = batch as Record<string, unknown>;
    if (Array.isArray(b.paths)) {
      for (const p of b.paths) push(p);
    }
    if (Array.isArray(b.proposals)) {
      for (const item of b.proposals) {
        if (item && typeof item === 'object') {
          push((item as Record<string, unknown>).file_path);
        }
      }
    }
  }

  return [...new Set(paths)];
}
