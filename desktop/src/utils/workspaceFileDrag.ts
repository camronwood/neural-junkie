/** MIME type for dragging files from the in-app file explorer to the chat composer. */
export const WORKSPACE_FILE_DRAG_MIME = 'application/x-neural-junkie-workspace-file';
export const WORKSPACE_FILE_DROPZONE_ATTR = 'data-workspace-file-dropzone';
export const WORKSPACE_FILE_DROP_EVENT = 'neural-junkie:workspace-file-drop';
const FALLBACK_TTL_MS = 30_000;
const DRAG_END_CLEAR_DELAY_MS = 1_500;

export interface WorkspaceFileDragPayload {
  workspaceId: string;
  /** Path relative to workspace root (same as file tree / editor). */
  path: string;
}

let activeWorkspaceFileDrag: { payload: WorkspaceFileDragPayload; startedAt: number } | null = null;
let clearTimer: ReturnType<typeof setTimeout> | null = null;
let activeDropZone: Element | null = null;

function validPayload(payload: WorkspaceFileDragPayload | null | undefined): WorkspaceFileDragPayload | null {
  if (
    payload &&
    typeof payload.workspaceId === 'string' &&
    typeof payload.path === 'string' &&
    payload.workspaceId.trim() &&
    payload.path.trim()
  ) {
    return { workspaceId: payload.workspaceId.trim(), path: payload.path.trim() };
  }
  return null;
}

export function setWorkspaceFileDragData(
  dataTransfer: DataTransfer,
  payload: WorkspaceFileDragPayload
): void {
  const normalized = validPayload(payload);
  if (!normalized) return;

  if (clearTimer) {
    clearTimeout(clearTimer);
    clearTimer = null;
  }
  activeWorkspaceFileDrag = { payload: normalized, startedAt: Date.now() };
  try {
    dataTransfer.setData(WORKSPACE_FILE_DRAG_MIME, JSON.stringify(normalized));
  } catch {
    /* Some WebKit-backed runtimes reject custom drag MIME types. */
  }
  try {
    dataTransfer.setData('text/plain', normalized.path);
  } catch {
    /* ignore */
  }
  dataTransfer.effectAllowed = 'copy';
}

export function clearWorkspaceFileDragData(): void {
  if (clearTimer) {
    clearTimeout(clearTimer);
    clearTimer = null;
  }
  activeWorkspaceFileDrag = null;
  activeDropZone = null;
}

export function getActiveWorkspaceFileDragPayload(): WorkspaceFileDragPayload | null {
  if (!activeWorkspaceFileDrag) return null;
  if (Date.now() - activeWorkspaceFileDrag.startedAt > FALLBACK_TTL_MS) {
    clearWorkspaceFileDragData();
    return null;
  }
  return activeWorkspaceFileDrag.payload;
}

export function dispatchWorkspaceFileDropEventAtPoint(clientX: number, clientY: number): boolean {
  const payload = getActiveWorkspaceFileDragPayload();
  if (!payload || typeof document === 'undefined') return false;

  const target = document.elementFromPoint(clientX, clientY);
  const dropZone =
    target?.closest?.(`[${WORKSPACE_FILE_DROPZONE_ATTR}="true"]`) ??
    (activeDropZone?.isConnected ? activeDropZone : null);
  if (!dropZone) return false;

  dropZone.dispatchEvent(
    new CustomEvent(WORKSPACE_FILE_DROP_EVENT, {
      bubbles: true,
      detail: { payload },
    })
  );
  return true;
}

export function setActiveWorkspaceFileDropZone(element: Element | null): void {
  activeDropZone = element;
}

export function scheduleWorkspaceFileDragClear(): void {
  if (clearTimer) {
    clearTimeout(clearTimer);
  }
  clearTimer = setTimeout(() => {
    activeWorkspaceFileDrag = null;
    clearTimer = null;
  }, DRAG_END_CLEAR_DELAY_MS);
}

function readDataTransferText(dataTransfer: DataTransfer, type: string): string {
  try {
    return dataTransfer.getData(type);
  } catch {
    return '';
  }
}

export function parseWorkspaceFileDrag(dataTransfer: DataTransfer): WorkspaceFileDragPayload[] {
  const raw = readDataTransferText(dataTransfer, WORKSPACE_FILE_DRAG_MIME);
  if (raw) {
    try {
      const parsed = validPayload(JSON.parse(raw) as WorkspaceFileDragPayload);
      if (parsed) {
        return [parsed];
      }
    } catch {
      /* ignore */
    }
  }

  const activePayload = getActiveWorkspaceFileDragPayload();
  if (activePayload) {
    const plain = readDataTransferText(dataTransfer, 'text/plain').trim();
    if (!plain || plain === activePayload.path) {
      return [activePayload];
    }
  }

  return [];
}
