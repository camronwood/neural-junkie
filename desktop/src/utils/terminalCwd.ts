import { useFileExplorerStore } from '../stores/fileExplorerStore';
import type { Collaboration } from '../types/protocol';
import { isCollabSandboxPath } from './outboundChatMetadata';

/**
 * Resolves the working directory for terminal tabs and agent command suggestions.
 */
export function resolveTerminalCwd(options?: {
  collaboration?: Collaboration | null;
  explicit?: string;
}): string {
  const explicit = options?.explicit?.trim();
  if (explicit && explicit !== '~') {
    return explicit;
  }

  const collab = options?.collaboration;
  if (collab) {
    const source = collab.source_repo_path?.trim();
    if (source && !isCollabSandboxPath(source)) {
      return source;
    }
    const work = collab.working_directory?.trim();
    if (work) {
      return work;
    }
  }

  const { workspaces, activeWorkspaceId } = useFileExplorerStore.getState();
  const active = workspaces.find((w) => w.id === activeWorkspaceId) ?? workspaces[0];
  const path = active?.path?.trim();
  if (path && !isCollabSandboxPath(path)) {
    return path;
  }

  return '~';
}
