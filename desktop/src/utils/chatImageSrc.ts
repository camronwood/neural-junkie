import { convertFileSrc } from '@tauri-apps/api/tauri';
import { ChatAPI } from '../api/chatAPI';
import { getHubBaseURL } from '../config/hubUrl';

function isTauriShell(): boolean {
  return (
    typeof window !== 'undefined' &&
    Object.prototype.hasOwnProperty.call(window, '__TAURI__')
  );
}

/**
 * Image src for the code editor (workspace files).
 * Tauri: asset protocol. Browser dev: hub base64 data URL.
 */
export async function resolveEditorImageSrc(options: {
  workspaceId: string;
  relativePath: string;
  absolutePath: string;
}): Promise<string> {
  const { workspaceId, relativePath, absolutePath } = options;
  if (isTauriShell()) {
    return convertFileSrc(absolutePath);
  }
  const api = new ChatAPI(getHubBaseURL());
  return api.fetchWorkspaceImageDataUrl(workspaceId, relativePath);
}

/**
 * Turn chat/markdown image URLs into something the WebView can load.
 * - data: and http(s) URLs pass through unchanged.
 * - Absolute filesystem paths use Tauri's asset protocol when running inside Tauri.
 */
export function resolveChatImageSrc(raw: string): string {
  const src = raw.trim().replace(/^<|>$/g, '');
  if (!src) return src;
  const lower = src.toLowerCase();
  if (src.startsWith('data:')) {
    if (!lower.startsWith('data:image/')) return '';
    return src;
  }
  if (lower.startsWith('javascript:')) return '';
  if (/^https?:\/\//i.test(src)) return src;

  let path = src;
  if (src.startsWith('file://')) {
    try {
      const u = new URL(src);
      path = decodeURIComponent(u.pathname);
      // file:///C:/Users/... → /C:/Users/... (Windows)
      if (/^\/[A-Za-z]:\//.test(path)) {
        path = path.slice(1);
      }
    } catch {
      return src;
    }
  }

  const isAbsoluteFs =
    path.startsWith('/') ||
    /^[A-Za-z]:[\\/]/.test(path);

  if (isAbsoluteFs && isTauriShell()) {
    return convertFileSrc(path);
  }

  if (src.startsWith('file://')) return src;
  return path;
}

/** Src for hub `generated_image` metadata (inline base64, Tauri path, or local hub file API). */
export function generatedImageSrc(meta: Record<string, unknown> | undefined): string | null {
  if (!meta) return null;
  const g = meta.generated_image as Record<string, unknown> | undefined;
  if (!g) return null;
  const mime = String(g.mime || 'image/png');
  const data = String(g.data || '');
  if (data && g.data_redacted !== true) {
    return `data:${mime};base64,${data}`;
  }
  const path = typeof g.path === 'string' ? g.path.trim() : '';
  if (!path) return null;
  if (isTauriShell()) {
    return resolveChatImageSrc(path);
  }
  return `${getHubBaseURL()}/api/local-image?path=${encodeURIComponent(path)}`;
}
