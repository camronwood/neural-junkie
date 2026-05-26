import type { CSSProperties } from 'react';

/** Keep a saved panel width as the preferred size, but allow flex rows to compress on small screens. */
export function shrinkablePanelStyle(preferredWidth: number, compactMinWidth: number): CSSProperties {
  return {
    width: preferredWidth,
    flex: `0 1 ${preferredWidth}px`,
    minWidth: compactMinWidth,
  };
}
