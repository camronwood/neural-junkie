import { getWorkspaceRoots } from './workspaceRoots';

/** Workspace roots argument for Tauri IPC path containment. */
export function ipcWorkspaceRoots(): { allowedRoots: string[] } {
  return { allowedRoots: getWorkspaceRoots() };
}

/** Register a native picker path for pack dev / custom install IPC. */
export async function registerPackPickerPath(absolutePath: string): Promise<void> {
  const { invoke } = await import('@tauri-apps/api/core');
  await invoke('register_pack_path', { path: absolutePath });
}
