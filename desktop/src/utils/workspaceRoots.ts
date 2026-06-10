import { useFileExplorerStore } from '../stores/fileExplorerStore';

/** Absolute workspace root paths for Tauri IPC path containment. */
export function getWorkspaceRoots(): string[] {
  return useFileExplorerStore
    .getState()
    .workspaces.map((w) => w.path.trim())
    .filter(Boolean);
}
