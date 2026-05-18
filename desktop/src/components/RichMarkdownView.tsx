import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  renderMarkdown,
  splitMarkdownAndMermaid,
} from '../utils/markdownRenderer';
import { renderMermaidSvg } from '../utils/mermaidConfig';
import { MermaidModal } from './MermaidModal';
import { ErrorBoundary } from './ErrorBoundary';

interface MermaidDiagramProps {
  content: string;
  onExpand: (content: string) => void;
  compact?: boolean;
}

function MermaidDiagram({ content, onExpand, compact }: MermaidDiagramProps) {
  const svgTargetRef = useRef<HTMLDivElement>(null);
  const mountedRef = useRef(true);
  const [isRendering, setIsRendering] = useState(false);
  const [renderError, setRenderError] = useState<string | null>(null);
  const [retryCount, setRetryCount] = useState(0);

  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
    };
  }, []);

  const renderDiagram = useCallback(async () => {
    if (!svgTargetRef.current) return;

    setIsRendering(true);
    setRenderError(null);

    try {
      svgTargetRef.current.innerHTML = '';
      const svg = await renderMermaidSvg(content);

      if (!mountedRef.current || !svgTargetRef.current) return;
      svgTargetRef.current.innerHTML = svg;

      requestAnimationFrame(() => {
        if (!mountedRef.current || !svgTargetRef.current) return;
        const svgEl = svgTargetRef.current.querySelector('svg');
        if (svgEl) {
          svgEl.style.maxWidth = 'none';
          svgEl.style.width = 'auto';
          svgEl.style.display = 'block';
          if (svgEl.getAttribute('width')?.includes('%')) {
            svgEl.removeAttribute('width');
          }
        }
      });

      setRetryCount(0);
    } catch (err) {
      if (!mountedRef.current) return;
      console.error('Mermaid rendering error:', err);
      setRenderError(err instanceof Error ? err.message : String(err));
    } finally {
      if (mountedRef.current) setIsRendering(false);
    }
  }, [content]);

  const handleRetry = () => {
    setRetryCount((prev) => prev + 1);
    renderDiagram();
  };

  useEffect(() => {
    if (svgTargetRef.current) {
      renderDiagram();
    }
  }, [renderDiagram, retryCount]);

  const padding = compact ? 'p-3' : 'p-6';
  const margin = compact ? 'my-3' : 'my-6';
  const minHeight = compact ? 'min-h-[120px]' : 'min-h-[200px]';

  return (
    <div className={`mermaid-diagram w-full ${margin}`}>
      {renderError ? (
        <div className="p-4 bg-red-500/10 border border-red-500/20 rounded text-red-400 text-sm">
          <strong>Mermaid Diagram Error:</strong>
          <pre className="mt-2 text-xs whitespace-pre-wrap">{renderError}</pre>
          <button
            type="button"
            className="mt-2 px-3 py-1 bg-red-500/20 hover:bg-red-500/30 rounded text-xs transition-colors"
            onClick={handleRetry}
          >
            Retry
          </button>
        </div>
      ) : (
        <div
          className={`w-full ${minHeight} ${padding} bg-slack-bgHover rounded border border-slack-border overflow-x-auto overflow-y-visible cursor-pointer hover:bg-slack-accent/10 transition-colors relative`}
          onClick={() => onExpand(content)}
          role="button"
          tabIndex={0}
          title="Click to expand diagram"
        >
          <div ref={svgTargetRef} />
          {isRendering && (
            <div className="absolute inset-0 flex items-center justify-center bg-slack-bgHover/80 rounded">
              <div className="flex items-center gap-2 text-slack-text">
                <div className="w-4 h-4 border-2 border-slack-accent border-t-transparent rounded-full animate-spin" />
                <span className="text-sm">Rendering diagram...</span>
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

export interface RichMarkdownViewProps {
  content: string;
  /** Tighter spacing for side panels (collaboration plan, etc.). */
  compact?: boolean;
  className?: string;
}

export function RichMarkdownView({ content, compact = false, className = '' }: RichMarkdownViewProps) {
  const [expandedDiagram, setExpandedDiagram] = useState<string | null>(null);

  const segments = useMemo(() => splitMarkdownAndMermaid(content), [content]);

  if (!content.trim()) {
    return null;
  }

  const rootClass = [
    'markdown-content prose prose-invert max-w-none',
    compact ? 'markdown-content--compact' : '',
    className,
  ]
    .filter(Boolean)
    .join(' ');

  return (
    <>
      <div className={rootClass}>
        {segments.map((seg, i) => {
          if (seg.type === 'mermaid') {
            return (
              <ErrorBoundary
                key={`mermaid-${i}`}
                fallback={
                  <div className="my-3 p-4 bg-red-500/10 border border-red-500/20 rounded text-red-400 text-sm">
                    <strong>Diagram Error:</strong> Failed to render Mermaid diagram.
                  </div>
                }
              >
                <MermaidDiagram
                  content={seg.content}
                  onExpand={setExpandedDiagram}
                  compact={compact}
                />
              </ErrorBoundary>
            );
          }
          const html = renderMarkdown(seg.content);
          return <div key={`md-${i}`} dangerouslySetInnerHTML={{ __html: html }} />;
        })}
      </div>

      <MermaidModal
        isOpen={expandedDiagram !== null}
        onClose={() => setExpandedDiagram(null)}
        content={expandedDiagram || ''}
      />
    </>
  );
}