/** Open http(s) links in the OS default browser (never in-app webview). */

export function isExternalHttpUrl(url: string): boolean {
  const trimmed = (url ?? '').trim();
  if (!trimmed) return false;
  // Absolute http(s) only — relative paths must stay in-app.
  if (!/^https?:\/\//i.test(trimmed)) return false;
  try {
    const u = new URL(trimmed);
    return u.protocol === 'http:' || u.protocol === 'https:';
  } catch {
    return false;
  }
}

export function openExternalLink(url: string): void {
  const trimmed = (url ?? '').trim();
  if (!trimmed || !isExternalHttpUrl(trimmed)) return;
  try {
    if (typeof window !== 'undefined' && (window as { __TAURI__?: unknown }).__TAURI__) {
      void import('@tauri-apps/api/shell').then(({ open }) => open(trimmed));
      return;
    }
  } catch {
    /* fall through */
  }
  window.open(trimmed, '_blank', 'noopener,noreferrer');
}

/** Capture-phase click handler: keep http(s) navigation out of the Tauri webview. */
export function installExternalLinkClickInterceptor(): () => void {
  const onClick = (event: MouseEvent) => {
    if (event.defaultPrevented) return;
    if (event.button !== 0) return;
    if (event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) return;
    const target = event.target;
    if (!(target instanceof Element)) return;
    const anchor = target.closest('a[href]');
    if (!(anchor instanceof HTMLAnchorElement)) return;
    const href = anchor.getAttribute('href') ?? '';
    if (!isExternalHttpUrl(href)) return;
    // Allow pack / workbench previews that intentionally use about: or blob: (not http).
    event.preventDefault();
    event.stopPropagation();
    openExternalLink(href);
  };
  document.addEventListener('click', onClick, true);
  return () => document.removeEventListener('click', onClick, true);
}
