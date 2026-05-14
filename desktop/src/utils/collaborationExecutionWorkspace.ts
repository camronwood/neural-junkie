import type { Collaboration } from '../types/protocol';
import { useFileExplorerStore } from '../stores/fileExplorerStore';

const linkedCollabExecutionWorkspaces = new Set<string>();

/**
 * Registers the hub-created collaboration sandbox as a desktop workspace (once per session per collab id).
 */
export async function ensureCollaborationExecutionWorkspace(collab: Collaboration): Promise<void> {
  if (collab.phase !== 'executing' || !collab.working_directory?.trim()) {
    return;
  }

  if (linkedCollabExecutionWorkspaces.has(collab.id)) {
    return;
  }

  try {
    const name = `Collab: ${(collab.title || 'session').slice(0, 48)}`;
    await useFileExplorerStore.getState().addWorkspace(name, collab.working_directory);
    await useFileExplorerStore.getState().loadWorkspaces();
    const wid = useFileExplorerStore.getState().activeWorkspaceId;
    if (wid) {
      await useFileExplorerStore.getState().loadFiles(wid, '/');
    }
    linkedCollabExecutionWorkspaces.add(collab.id);
  } catch (e) {
    console.error('[ensureCollaborationExecutionWorkspace]', e);
  }
}
