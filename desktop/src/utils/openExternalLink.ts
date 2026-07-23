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

/**
 * Opens an http(s) URL in the OS browser.
 * Returns true when a handoff was attempted successfully; false when the URL
 * was invalid or every opener path failed (caller should show a copyable link).
 */
export async function openExternalLinkAsync(url: string): Promise<boolean> {
  const trimmed = (url ?? '').trim();
  if (!trimmed || !isExternalHttpUrl(trimmed)) return false;

  try {
    const { isTauri } = await import('@tauri-apps/api/core');
    if (isTauri()) {
      try {
        const { openUrl } = await import('@tauri-apps/plugin-opener');
        await openUrl(trimmed);
        return true;
      } catch {
        // Missing URL scope / older builds: fall through to Rust command.
      }
      try {
        const { invoke } = await import('@tauri-apps/api/core');
        await invoke('open_browser_window', { url: trimmed });
        return true;
      } catch {
        // Fall through to window.open last resort.
      }
    }
  } catch {
    /* not Tauri or core import failed */
  }

  const opened = window.open(trimmed, '_blank', 'noopener,noreferrer');
  return opened != null;
}

export function openExternalLink(url: string): void {
  void openExternalLinkAsync(url);
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
