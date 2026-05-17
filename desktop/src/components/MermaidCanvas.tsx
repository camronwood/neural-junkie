import { useState, useEffect, useRef, useCallback } from 'react';
import { renderMermaidSvg } from '../utils/mermaidConfig';

export interface MermaidCanvasProps {
  content: string;
  active?: boolean;
  className?: string;
  showZoomControls?: boolean;
}

const WHEEL_ZOOM_INTENSITY = 0.002;
const BUTTON_ZOOM_FACTOR = 1.2;
const MIN_SCALE = 1e-4;
function formatZoomLabel(scale: number): string {
  const pct = scale * 100;
  if (pct >= 10000 || pct < 0.1) return `${scale.toFixed(2)}×`;
  if (pct >= 1000) return `${Math.round(pct)}%`;
  return `${Math.round(pct)}%`;
}

function readSvgNativeSize(svg: SVGSVGElement): { width: number; height: number } {
  if (svg.viewBox?.baseVal && svg.viewBox.baseVal.width > 0) {
    return {
      width: svg.viewBox.baseVal.width,
      height: svg.viewBox.baseVal.height,
    };
  }
  const widthAttr = svg.getAttribute('width');
  const heightAttr = svg.getAttribute('height');
  if (widthAttr && heightAttr && !widthAttr.includes('%')) {
    return {
      width: parseFloat(widthAttr) || 800,
      height: parseFloat(heightAttr) || 600,
    };
  }
  const rect = svg.getBoundingClientRect();
  if (rect.width > 0 && rect.height > 0) {
    return { width: rect.width, height: rect.height };
  }
  return { width: 800, height: 600 };
}

/** Size the SVG element directly so initial fit stays sharp (avoid CSS scale for fit). */
function applySvgDisplaySize(
  svg: SVGSVGElement,
  nativeWidth: number,
  nativeHeight: number,
  displayScale: number
) {
  const w = nativeWidth * displayScale;
  const h = nativeHeight * displayScale;
  svg.setAttribute('width', String(w));
  svg.setAttribute('height', String(h));
  svg.style.width = `${w}px`;
  svg.style.height = `${h}px`;
  svg.style.maxWidth = 'none';
  svg.style.display = 'block';
}

export function MermaidCanvas({
  content,
  active = true,
  className = '',
  showZoomControls = true,
}: MermaidCanvasProps) {
  const [userScale, setUserScale] = useState(1);
  const [position, setPosition] = useState({ x: 0, y: 0 });
  const [isDragging, setIsDragging] = useState(false);
  const [dragStart, setDragStart] = useState({ x: 0, y: 0 });
  const [lastPosition, setLastPosition] = useState({ x: 0, y: 0 });
  const [renderError, setRenderError] = useState<string | null>(null);
  const [isRendering, setIsRendering] = useState(false);
  const [retryCount, setRetryCount] = useState(0);

  const containerRef = useRef<HTMLDivElement>(null);
  const diagramRef = useRef<HTMLDivElement>(null);
  const fitScaleRef = useRef(1);
  const nativeSizeRef = useRef({ width: 800, height: 600 });
  const displayScale = () => fitScaleRef.current * userScale;

  const applyZoomAtPoint = useCallback(
    (factor: number, centerX: number, centerY: number) => {
      if (factor <= 0 || !Number.isFinite(factor)) return;
      setUserScale((prevScale) => {
        const nextScale = Math.max(MIN_SCALE, prevScale * factor);
        const appliedFactor = nextScale / prevScale;
        setPosition((prevPos) => ({
          x: prevPos.x - centerX * (appliedFactor - 1),
          y: prevPos.y - centerY * (appliedFactor - 1),
        }));
        return nextScale;
      });
    },
    []
  );

  useEffect(() => {
    if (!active) return;
    setPosition({ x: 0, y: 0 });
    setLastPosition({ x: 0, y: 0 });
    setUserScale(1);
  }, [active, content]);

  useEffect(() => {
    if (!active || !diagramRef.current || !containerRef.current) return;

    const renderDiagram = async () => {
      setIsRendering(true);
      setRenderError(null);

      try {
        diagramRef.current!.innerHTML = '';
        const svg = await renderMermaidSvg(content);
        diagramRef.current!.innerHTML = svg;

        requestAnimationFrame(() => {
          if (!diagramRef.current || !containerRef.current) return;

          const svgElement = diagramRef.current.querySelector('svg');
          if (!svgElement) return;

          const native = readSvgNativeSize(svgElement);
          nativeSizeRef.current = native;

          const containerRect = containerRef.current.getBoundingClientRect();
          const padding = 32;
          const availableWidth = containerRect.width - padding;
          const availableHeight = containerRect.height - padding;

          const scaleX = availableWidth / native.width;
          const scaleY = availableHeight / native.height;
          const fitScale = Math.max(MIN_SCALE, Math.min(scaleX, scaleY));
          fitScaleRef.current = fitScale;

          applySvgDisplaySize(svgElement, native.width, native.height, fitScale);
          setUserScale(1);
          setPosition({ x: 0, y: 0 });
          setLastPosition({ x: 0, y: 0 });
        });
      } catch (error) {
        const message = error instanceof Error ? error.message : String(error);
        console.error('Mermaid rendering error:', error);
        setRenderError(message);
        fitScaleRef.current = 1;
        setUserScale(1);
      } finally {
        setIsRendering(false);
      }
    };

    renderDiagram();
  }, [active, content, retryCount]);

  useEffect(() => {
    const container = containerRef.current;
    if (!container || !active) return;

    const onWheel = (e: WheelEvent) => {
      e.preventDefault();

      const rect = container.getBoundingClientRect();
      const centerX = e.clientX - rect.left - rect.width / 2;
      const centerY = e.clientY - rect.top - rect.height / 2;

      const factor = Math.exp(-e.deltaY * WHEEL_ZOOM_INTENSITY);
      applyZoomAtPoint(factor, centerX, centerY);
    };

    container.addEventListener('wheel', onWheel, { passive: false });
    return () => container.removeEventListener('wheel', onWheel);
  }, [active, applyZoomAtPoint]);

  const handleZoomIn = () => applyZoomAtPoint(BUTTON_ZOOM_FACTOR, 0, 0);
  const handleZoomOut = () => applyZoomAtPoint(1 / BUTTON_ZOOM_FACTOR, 0, 0);

  const handleReset = () => {
    setUserScale(1);
    setPosition({ x: 0, y: 0 });
    setLastPosition({ x: 0, y: 0 });
  };

  const stopDrag = () => setIsDragging(false);

  const handleMouseDown = (e: React.MouseEvent) => {
    if (renderError) return;
    if (e.target === diagramRef.current || diagramRef.current?.contains(e.target as Node)) {
      setIsDragging(true);
      setDragStart({ x: e.clientX, y: e.clientY });
      setLastPosition(position);
    }
  };

  const handleMouseMove = (e: React.MouseEvent) => {
    if (isDragging) {
      setPosition({
        x: lastPosition.x + (e.clientX - dragStart.x),
        y: lastPosition.y + (e.clientY - dragStart.y),
      });
    }
  };

  // Pan + user zoom only on transform; initial fit uses SVG width/height (sharp).
  // No CSS transition on transform — avoids compositor smear on macOS when zooming.
  const diagramTransform = `translate(${position.x}px, ${position.y}px) scale(${userScale})`;

  return (
    <div className={`relative flex flex-col flex-1 min-h-0 ${className}`}>
      {showZoomControls && (
        <div className='absolute top-4 left-4 z-10 flex flex-col gap-2'>
          <button
            type='button'
            onClick={handleZoomIn}
            className='p-2 bg-slack-bgHover hover:bg-slack-accent text-slack-text hover:text-white rounded transition-colors'
            title='Zoom In'
          >
            <svg className='w-5 h-5' fill='none' stroke='currentColor' viewBox='0 0 24 24'>
              <path strokeLinecap='round' strokeLinejoin='round' strokeWidth={2} d='M12 6v6m0 0v6m0-6h6m-6 0H6' />
            </svg>
          </button>
          <button
            type='button'
            onClick={handleZoomOut}
            className='p-2 bg-slack-bgHover hover:bg-slack-accent text-slack-text hover:text-white rounded transition-colors'
            title='Zoom Out'
          >
            <svg className='w-5 h-5' fill='none' stroke='currentColor' viewBox='0 0 24 24'>
              <path strokeLinecap='round' strokeLinejoin='round' strokeWidth={2} d='M18 12H6' />
            </svg>
          </button>
          <button
            type='button'
            onClick={handleReset}
            className='p-2 bg-slack-bgHover hover:bg-slack-accent text-slack-text hover:text-white rounded transition-colors'
            title='Reset View'
          >
            <svg className='w-5 h-5' fill='none' stroke='currentColor' viewBox='0 0 24 24'>
              <path
                strokeLinecap='round'
                strokeLinejoin='round'
                strokeWidth={2}
                d='M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15'
              />
            </svg>
          </button>
        </div>
      )}

      {renderError ? (
        <div className='flex flex-1 items-center justify-center p-8'>
          <div className='max-w-lg p-4 bg-red-500/10 border border-red-500/20 rounded text-red-400 text-sm'>
            <strong>Mermaid Diagram Error:</strong>
            <pre className='mt-2 text-xs whitespace-pre-wrap'>{renderError}</pre>
            <button
              type='button'
              className='mt-3 px-3 py-1 bg-red-500/20 hover:bg-red-500/30 rounded text-xs transition-colors'
              onClick={() => setRetryCount((prev) => prev + 1)}
            >
              Retry
            </button>
          </div>
        </div>
      ) : (
        <div
          ref={containerRef}
          className='relative flex-1 min-h-0 overflow-hidden cursor-grab active:cursor-grabbing flex items-center justify-center'
          onMouseDown={handleMouseDown}
          onMouseMove={handleMouseMove}
          onMouseUp={stopDrag}
          onMouseLeave={stopDrag}
        >
          {isRendering && (
            <div className='absolute inset-0 flex items-center justify-center bg-black/40 z-10'>
              <div className='flex items-center gap-2 text-slack-text'>
                <div className='w-5 h-5 border-2 border-slack-accent border-t-transparent rounded-full animate-spin' />
                <span className='text-sm'>Rendering diagram...</span>
              </div>
            </div>
          )}
          <div
            ref={diagramRef}
            className='p-4 bg-white rounded border border-slack-border flex items-center justify-center'
            style={{
              transform: diagramTransform,
              transformOrigin: 'center center',
            }}
          />
        </div>
      )}

      {showZoomControls && !renderError && (
        <div className='absolute bottom-4 left-4 z-10 px-3 py-1 bg-slack-bgHover text-slack-text rounded text-sm'>
          {formatZoomLabel(displayScale())}
        </div>
      )}
    </div>
  );
}