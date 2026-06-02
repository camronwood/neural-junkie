import { useCallback, useEffect, useRef, useState } from 'react';

type ResizeEdge = 'left' | 'right';

interface UseHorizontalPanelResizeOptions {
  storageKey: string;
  defaultWidth: number;
  minWidth: number;
  maxWidthRatio?: number;
  /** When set, overrides maxWidthRatio for clamp (e.g. container-based max). */
  getMaxWidth?: () => number;
  /** Handle on left edge (panel on right) or right edge (panel on left). */
  edge: ResizeEdge;
}

function resolveMaxWidth(maxWidthRatio: number, getMaxWidth?: () => number): number {
  if (getMaxWidth) {
    const w = getMaxWidth();
    if (Number.isFinite(w) && w > 0) return w;
  }
  return window.innerWidth * maxWidthRatio;
}

export function useHorizontalPanelResize({
  storageKey,
  defaultWidth,
  minWidth,
  maxWidthRatio = 0.65,
  getMaxWidth,
  edge,
}: UseHorizontalPanelResizeOptions) {
  const getMaxWidthRef = useRef(getMaxWidth);
  getMaxWidthRef.current = getMaxWidth;

  const [width, setWidth] = useState<number>(() => {
    const saved = localStorage.getItem(storageKey);
    const parsed = saved ? parseInt(saved, 10) : defaultWidth;
    if (!Number.isFinite(parsed)) return defaultWidth;
    const max = resolveMaxWidth(maxWidthRatio, getMaxWidthRef.current);
    if (parsed > max) return defaultWidth;
    return Math.max(minWidth, parsed);
  });
  const [isResizing, setIsResizing] = useState(false);
  const resizeStartX = useRef(0);
  const resizeStartWidth = useRef(width);
  const currentWidthRef = useRef(width);

  useEffect(() => {
    currentWidthRef.current = width;
  }, [width]);

  useEffect(() => {
    if (!isResizing) return;

    const handleMouseMove = (e: MouseEvent) => {
      const delta =
        edge === 'left'
          ? resizeStartX.current - e.clientX
          : e.clientX - resizeStartX.current;
      const max = resolveMaxWidth(maxWidthRatio, getMaxWidthRef.current);
      const next = Math.min(max, Math.max(minWidth, resizeStartWidth.current + delta));
      setWidth(next);
    };

    const handleMouseUp = () => {
      setIsResizing(false);
      localStorage.setItem(storageKey, String(currentWidthRef.current));
      document.body.style.cursor = '';
      document.body.style.userSelect = '';
    };

    document.addEventListener('mousemove', handleMouseMove);
    document.addEventListener('mouseup', handleMouseUp);
    document.body.style.cursor = 'col-resize';
    document.body.style.userSelect = 'none';

    return () => {
      document.removeEventListener('mousemove', handleMouseMove);
      document.removeEventListener('mouseup', handleMouseUp);
      document.body.style.cursor = '';
      document.body.style.userSelect = '';
    };
  }, [isResizing, edge, minWidth, maxWidthRatio, storageKey]);

  const onResizeStart = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    resizeStartX.current = e.clientX;
    resizeStartWidth.current = currentWidthRef.current;
    setIsResizing(true);
  }, []);

  return { width, isResizing, onResizeStart };
}
