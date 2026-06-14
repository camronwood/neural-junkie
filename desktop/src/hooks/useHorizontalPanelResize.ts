import { useCallback, useEffect, useRef, useState, type RefObject } from 'react';

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
  /** Observe container size changes and reclamp saved width (e.g. flex row). */
  containerRef?: RefObject<HTMLElement | null>;
  /** When this value changes, reclamp width (e.g. panel visibility toggles). */
  reclampKey?: string;
}

function resolveMaxWidth(maxWidthRatio: number, getMaxWidth?: () => number): number {
  if (getMaxWidth) {
    const w = getMaxWidth();
    if (Number.isFinite(w) && w > 0) return w;
  }
  return window.innerWidth * maxWidthRatio;
}

function clampWidth(width: number, minWidth: number, max: number): number {
  return Math.min(max, Math.max(minWidth, width));
}

export function useHorizontalPanelResize({
  storageKey,
  defaultWidth,
  minWidth,
  maxWidthRatio = 0.65,
  getMaxWidth,
  edge,
  containerRef,
  reclampKey,
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

  const reclamp = useCallback(() => {
    if (isResizing) return;
    const max = resolveMaxWidth(maxWidthRatio, getMaxWidthRef.current);
    const next = clampWidth(currentWidthRef.current, minWidth, max);
    if (next === currentWidthRef.current) return;
    currentWidthRef.current = next;
    setWidth(next);
    localStorage.setItem(storageKey, String(next));
  }, [isResizing, minWidth, maxWidthRatio, storageKey]);

  useEffect(() => {
    const onWindowResize = () => reclamp();
    window.addEventListener('resize', onWindowResize);
    return () => window.removeEventListener('resize', onWindowResize);
  }, [reclamp]);

  useEffect(() => {
    const el = containerRef?.current;
    if (!el || typeof ResizeObserver === 'undefined') return;
    const observer = new ResizeObserver(() => reclamp());
    observer.observe(el);
    return () => observer.disconnect();
  }, [containerRef, reclamp]);

  useEffect(() => {
    reclamp();
  }, [reclampKey, reclamp]);

  useEffect(() => {
    if (!isResizing) return;

    const handleMouseMove = (e: MouseEvent) => {
      const delta =
        edge === 'left'
          ? resizeStartX.current - e.clientX
          : e.clientX - resizeStartX.current;
      const max = resolveMaxWidth(maxWidthRatio, getMaxWidthRef.current);
      const next = clampWidth(resizeStartWidth.current + delta, minWidth, max);
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
