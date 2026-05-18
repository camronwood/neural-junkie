/** MIME type for dragging files from the in-app file explorer to the chat composer. */
export const WORKSPACE_FILE_DRAG_MIME = 'application/x-neural-junkie-workspace-file';

export interface WorkspaceFileDragPayload {
  workspaceId: string;
  /** Path relative to workspace root (same as file tree / editor). */
  path: string;
}

export function setWorkspaceFileDragData(
  dataTransfer: DataTransfer,
  payload: WorkspaceFileDragPayload
): void {
  dataTransfer.setData(WORKSPACE_FILE_DRAG_MIME, JSON.stringify(payload));
  dataTransfer.setData('text/plain', payload.path);
  dataTransfer.effectAllowed = 'copy';
}

export function parseWorkspaceFileDrag(dataTransfer: DataTransfer): WorkspaceFileDragPayload[] {
  const raw = dataTransfer.getData(WORKSPACE_FILE_DRAG_MIME);
  if (!raw) return [];
  try {
    const parsed = JSON.parse(raw) as WorkspaceFileDragPayload;
    if (
      parsed &&
      typeof parsed.workspaceId === 'string' &&
      typeof parsed.path === 'string' &&
      parsed.workspaceId.trim() &&
      parsed.path.trim()
    ) {
      return [{ workspaceId: parsed.workspaceId.trim(), path: parsed.path.trim() }];
    }
  } catch {
    /* ignore */
  }
  return [];
}
