import { convertFileSrc } from '@tauri-apps/api/tauri';

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

  if (
    isAbsoluteFs &&
    typeof window !== 'undefined' &&
    // Tauri injects this at runtime in the desktop shell
    Object.prototype.hasOwnProperty.call(window, '__TAURI__')
  ) {
    return convertFileSrc(path);
  }

  if (src.startsWith('file://')) return src;
  return path;
}
