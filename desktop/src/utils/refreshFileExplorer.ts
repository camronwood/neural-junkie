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
  // Reload root + every ancestor dir (same strategy as refreshTreeForPath in fileExplorerStore).
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
}

/** Extract workspace-relative paths from a file_change message proposal. */
export function fileChangeProposalPaths(message: {
  metadata?: Record<string, unknown>;
}): string[] {
  const raw = message.metadata?.file_change_proposal;
  if (!raw || typeof raw !== 'object') return [];
  const proposal = raw as Record<string, unknown>;
  const paths: string[] = [];
  if (typeof proposal.file_path === 'string' && proposal.file_path.trim()) {
    paths.push(proposal.file_path.trim());
  }
  if (typeof proposal.new_path === 'string' && proposal.new_path.trim()) {
    paths.push(proposal.new_path.trim());
  }
  return paths;
}
