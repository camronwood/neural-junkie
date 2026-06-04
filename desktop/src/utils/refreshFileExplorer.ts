import { useFileExplorerStore } from '../stores/fileExplorerStore';

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

  const dirs = new Set<string>(['/']);
  for (const rel of trimmed) {
    const lastSlash = rel.lastIndexOf('/');
    dirs.add(lastSlash > -1 ? rel.slice(0, lastSlash) : '/');
  }

  const { loadFiles } = useFileExplorerStore.getState();
  await loadFiles(workspaceId, '/');
  for (const dir of dirs) {
    if (dir !== '/') {
      await loadFiles(workspaceId, dir);
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
