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
  if (src.startsWith('data:')) return src;
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
