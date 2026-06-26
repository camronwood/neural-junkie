/** Whether the native WebView context menu (Reload, etc.) should be allowed. */
export function shouldAllowNativeContextMenu(target: EventTarget | null): boolean {
  const el = target as HTMLElement | null;
  if (!el) return false;
  if (el.closest('.monaco-editor')) return true;
  if (el.closest('.xterm')) return true;
  if (el.tagName === 'INPUT' || el.tagName === 'TEXTAREA' || el.tagName === 'SELECT') return true;
  if (el.isContentEditable) return true;
  return false;
}

/**
 * Block the native WebView context menu (Reload wipes WS/PTY sessions).
 * Custom React context menus still work; Monaco/xterm/inputs keep native copy/paste.
 */
export function initContextMenuPolicy(): () => void {
  const onContextMenu = (event: MouseEvent) => {
    if (!shouldAllowNativeContextMenu(event.target)) {
      event.preventDefault();
    }
  };
  document.addEventListener('contextmenu', onContextMenu, true);
  return () => document.removeEventListener('contextmenu', onContextMenu, true);
}
