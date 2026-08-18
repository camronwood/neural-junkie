/** MIME type for dragging files from the in-app file explorer to the chat composer. */
export const WORKSPACE_FILE_DRAG_MIME = 'application/x-neural-junkie-workspace-file';
export const WORKSPACE_FILE_DROPZONE_ATTR = 'data-workspace-file-dropzone';
export const WORKSPACE_FILE_DROP_EVENT = 'neural-junkie:workspace-file-drop';
const FALLBACK_TTL_MS = 30_000;
const DRAG_END_CLEAR_DELAY_MS = 1_500;
/** Keep last hovered composer drop zone briefly so dragend can find it after WebKit dragleave. */
const STICKY_DROP_ZONE_MS = 750;

export interface WorkspaceFileDragPayload {
  workspaceId: string;
  /** Path relative to workspace root (same as file tree / editor). */
  path: string;
}

let activeWorkspaceFileDrag: { payloads: WorkspaceFileDragPayload[]; startedAt: number } | null = null;
let clearTimer: ReturnType<typeof setTimeout> | null = null;
let activeDropZone: Element | null = null;
let stickyDropZone: { element: Element; at: number } | null = null;

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

function normalizePayloadList(
  input: WorkspaceFileDragPayload | WorkspaceFileDragPayload[] | null | undefined
): WorkspaceFileDragPayload[] {
  const list = Array.isArray(input) ? input : input ? [input] : [];
  const out: WorkspaceFileDragPayload[] = [];
  const seen = new Set<string>();
  for (const item of list) {
    const normalized = validPayload(item);
    if (!normalized) continue;
    const key = `${normalized.workspaceId}::${normalized.path}`;
    if (seen.has(key)) continue;
    seen.add(key);
    out.push(normalized);
  }
  return out;
}

function parseStoredPayloads(raw: string): WorkspaceFileDragPayload[] {
  try {
    const parsed = JSON.parse(raw) as unknown;
    if (Array.isArray(parsed)) {
      return normalizePayloadList(parsed as WorkspaceFileDragPayload[]);
    }
    // Backward compatible: single object payload.
    return normalizePayloadList(parsed as WorkspaceFileDragPayload);
  } catch {
    return [];
  }
}

function resolveStickyDropZone(): Element | null {
  if (!stickyDropZone?.element?.isConnected) {
    stickyDropZone = null;
    return null;
  }
  if (Date.now() - stickyDropZone.at > STICKY_DROP_ZONE_MS) {
    return null;
  }
  return stickyDropZone.element;
}

export function setWorkspaceFileDragData(
  dataTransfer: DataTransfer,
  payload: WorkspaceFileDragPayload | WorkspaceFileDragPayload[]
): void {
  const normalized = normalizePayloadList(payload);
  if (normalized.length === 0) return;

  if (clearTimer) {
    clearTimeout(clearTimer);
    clearTimer = null;
  }
  activeWorkspaceFileDrag = { payloads: normalized, startedAt: Date.now() };
  try {
    // Always store an array so multi-file drops round-trip cleanly.
    dataTransfer.setData(WORKSPACE_FILE_DRAG_MIME, JSON.stringify(normalized));
  } catch {
    /* Some WebKit-backed runtimes reject custom drag MIME types. */
  }
  try {
    dataTransfer.setData('text/plain', normalized.map((p) => p.path).join('\n'));
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
  stickyDropZone = null;
}

export function getActiveWorkspaceFileDragPayloads(): WorkspaceFileDragPayload[] {
  if (!activeWorkspaceFileDrag) return [];
  if (Date.now() - activeWorkspaceFileDrag.startedAt > FALLBACK_TTL_MS) {
    clearWorkspaceFileDragData();
    return [];
  }
  return activeWorkspaceFileDrag.payloads;
}

export function isWorkspaceFileDragActive(): boolean {
  return getActiveWorkspaceFileDragPayloads().length > 0;
}

/** @deprecated Prefer getActiveWorkspaceFileDragPayloads — kept for single-file callers. */
export function getActiveWorkspaceFileDragPayload(): WorkspaceFileDragPayload | null {
  return getActiveWorkspaceFileDragPayloads()[0] ?? null;
}

function dispatchDropToZone(dropZone: Element, payloads: WorkspaceFileDragPayload[]): boolean {
  dropZone.dispatchEvent(
    new CustomEvent(WORKSPACE_FILE_DROP_EVENT, {
      bubbles: true,
      detail: {
        payloads,
        // Backward compatible single-file field (first path).
        payload: payloads[0],
      },
    })
  );
  return true;
}

export function dispatchWorkspaceFileDropEventAtPoint(clientX: number, clientY: number): boolean {
  const payloads = getActiveWorkspaceFileDragPayloads();
  if (payloads.length === 0 || typeof document === 'undefined') return false;

  const target =
    Number.isFinite(clientX) && Number.isFinite(clientY)
      ? document.elementFromPoint(clientX, clientY)
      : null;
  const fromPoint = target?.closest?.(`[${WORKSPACE_FILE_DROPZONE_ATTR}="true"]`) ?? null;
  // Prefer sticky/active zone before elementFromPoint: WebKit dragend often reports (0,0).
  const dropZone =
    (activeDropZone?.isConnected ? activeDropZone : null) ??
    resolveStickyDropZone() ??
    fromPoint;
  if (!dropZone) return false;

  return dispatchDropToZone(dropZone, payloads);
}

/**
 * Force-dispatch to the sticky/active composer drop zone (dragend when coords are useless).
 * Returns false if there was no recent hover over a drop zone.
 */
export function dispatchWorkspaceFileDropToStickyZone(): boolean {
  const payloads = getActiveWorkspaceFileDragPayloads();
  if (payloads.length === 0) return false;
  const dropZone =
    (activeDropZone?.isConnected ? activeDropZone : null) ?? resolveStickyDropZone();
  if (!dropZone) return false;
  return dispatchDropToZone(dropZone, payloads);
}

export function setActiveWorkspaceFileDropZone(element: Element | null): void {
  activeDropZone = element;
  if (element) {
    stickyDropZone = { element, at: Date.now() };
  } else {
    stickyDropZone = null;
  }
}

/** Clear hover tracking without wiping sticky until the workspace drag ends. */
export function clearActiveWorkspaceFileDropZoneHover(): void {
  activeDropZone = null;
}

export function scheduleWorkspaceFileDragClear(): void {
  if (clearTimer) {
    clearTimeout(clearTimer);
  }
  clearTimer = setTimeout(() => {
    activeWorkspaceFileDrag = null;
    activeDropZone = null;
    stickyDropZone = null;
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
    const fromMime = parseStoredPayloads(raw);
    if (fromMime.length > 0) {
      return fromMime;
    }
  }

  const activePayloads = getActiveWorkspaceFileDragPayloads();
  if (activePayloads.length === 0) {
    return [];
  }

  // While an in-app explorer drag is active, trust the in-memory payloads.
  // WebKit often empties custom MIME types and may put unrelated text/plain on drop.
  const plain = readDataTransferText(dataTransfer, 'text/plain').trim();
  if (!plain) {
    return activePayloads;
  }
  const plainPaths = plain
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean);
  const activePaths = new Set(activePayloads.map((p) => p.path));
  if (plainPaths.length === 0 || plainPaths.every((p) => activePaths.has(p))) {
    return activePayloads;
  }
  // text/plain mismatch: still use active payloads for in-app drags (WKWebView quirk).
  return activePayloads;
}
