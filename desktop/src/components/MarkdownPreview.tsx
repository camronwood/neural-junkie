import { useState, useEffect, useRef, useCallback } from 'react';
import { ChatAPI } from '../api/chatAPI';
import { extractTitle, getContentHash, parseMarkdownWithMermaid, type MermaidBlock } from '../utils/markdownRenderer';
import mermaid from 'mermaid';
import { MermaidModal } from './MermaidModal';
import { ErrorBoundary } from './ErrorBoundary';

interface MarkdownPreviewProps {
  workspaceId: string;
  filePath: string;
}

// Initialize mermaid with configuration
mermaid.initialize({
  startOnLoad: false,
  theme: 'dark',
  securityLevel: 'loose',
  fontFamily: 'ui-monospace, monospace',
});

export function MarkdownPreview({ workspaceId, filePath }: MarkdownPreviewProps) {
  const [content, setContent] = useState<string>('');
  const [renderedHtml, setRenderedHtml] = useState<string>('');
  const [mermaidBlocks, setMermaidBlocks] = useState<MermaidBlock[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);
  const [title, setTitle] = useState<string>('Markdown Preview');
  const [expandedDiagram, setExpandedDiagram] = useState<string | null>(null);
  const [isRendering] = useState<boolean>(false);
  
  const contentHashRef = useRef<string>('');
  const apiRef = useRef<ChatAPI>(new ChatAPI('localhost:8080'));
  const intervalRef = useRef<number | null>(null);
  const abortControllerRef = useRef<AbortController | null>(null);

  // Extract filename from path
  const filename = filePath.split('/').pop() || 'Unknown';

  // Fetch file content
  const fetchContent = async () => {
    // Cancel any ongoing fetch
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
    }
    
    // Create new abort controller
    abortControllerRef.current = new AbortController();
    
    try {
      setError(null);
      const fileContent = await apiRef.current.fetchFileContent(workspaceId, filePath);
      const newHash = getContentHash(fileContent);
      
      // Only update if content has changed
      if (newHash !== contentHashRef.current) {
        setContent(fileContent);
        const parseResult = parseMarkdownWithMermaid(fileContent);
        setRenderedHtml(parseResult.html);
        setMermaidBlocks(parseResult.mermaidBlocks);
        setTitle(extractTitle(fileContent));
        setLastUpdated(new Date());
        contentHashRef.current = newHash;
      }
    } catch (err) {
      // Don't set error if it was aborted
      if (err instanceof Error && err.name === 'AbortError') {
        return;
      }
      const errorMessage = err instanceof Error ? err.message : 'Failed to load file';
      setError(errorMessage);
      console.error('Failed to fetch file content:', err);
    } finally {
      setLoading(false);
    }
  };

  // Initial load
  useEffect(() => {
    fetchContent();
  }, [workspaceId, filePath]);

  // Auto-refresh every 2 seconds
  useEffect(() => {
    intervalRef.current = window.setInterval(() => {
      if (!loading && !isRendering) {
        fetchContent();
      }
    }, 2000);

    return () => {
      if (intervalRef.current) {
        window.clearInterval(intervalRef.current);
      }
    };
  }, [loading, isRendering, workspaceId, filePath]);

  // Update document title
  useEffect(() => {
    document.title = `${title} - Markdown Preview`;
  }, [title]);

  const handleRefresh = () => {
    setLoading(true);
    fetchContent();
  };

  const formatTime = (date: Date) => {
    return date.toLocaleTimeString('en-US', {
      hour12: false,
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    });
  };

  // MermaidDiagram component - Simplified to match working MermaidModal pattern
  const MermaidDiagram = ({ content, index }: { content: string; index: number }) => {
    const containerRef = useRef<HTMLDivElement>(null);
    const [isRendering, setIsRendering] = useState(false);
    const [renderError, setRenderError] = useState<string | null>(null);
    const [retryCount, setRetryCount] = useState(0);
    
    // Create stable ID based on content hash
    const contentHash = getContentHash(content);
    const idRef = useRef(`mermaid-preview-${contentHash}-${index}`);

    const renderDiagram = useCallback(async () => {
      if (!containerRef.current) return;
      
      setIsRendering(true);
      setRenderError(null);
      
      try {
        // Clear previous content
        containerRef.current.innerHTML = '';
        
        // Render the diagram
        const { svg } = await mermaid.render(idRef.current, content);
        
        // Insert SVG
        if (containerRef.current) {
          containerRef.current.innerHTML = svg;
          
          // Wait for next frame to style the SVG
          requestAnimationFrame(() => {
            if (!containerRef.current) return;
            
            const svgElement = containerRef.current.querySelector('svg');
            if (svgElement) {
              // Remove width constraints to allow horizontal scrolling
              svgElement.style.maxWidth = 'none';
              svgElement.style.width = 'auto';
              svgElement.style.display = 'block';
              
              // Remove percentage widths
              if (svgElement.hasAttribute('width')) {
                const width = svgElement.getAttribute('width');
                if (width && width.includes('%')) {
                  svgElement.removeAttribute('width');
                }
              }
            }
          });
        }
        
        setRetryCount(0);
      } catch (error) {
        console.error('Mermaid rendering error:', error);
        const errorMessage = error instanceof Error ? error.message : String(error);
        setRenderError(errorMessage);
        
        if (containerRef.current) {
          containerRef.current.innerHTML = `
            <div class="p-4 bg-red-500/10 border border-red-500/20 rounded text-red-400 text-sm">
              <strong>Mermaid Diagram Error:</strong>
              <pre class="mt-2 text-xs whitespace-pre-wrap">${errorMessage}</pre>
              <button 
                class="mt-2 px-3 py-1 bg-red-500/20 hover:bg-red-500/30 rounded text-xs transition-colors"
                onclick="this.closest('.mermaid-diagram').querySelector('.retry-button').click()"
              >
                Retry
              </button>
            </div>
          `;
        }
      } finally {
        setIsRendering(false);
      }
    }, [content]);

    const handleRetry = () => {
      setRetryCount(prev => prev + 1);
      renderDiagram();
    };

    useEffect(() => {
      // Wait for ref to be attached, then render
      if (containerRef.current) {
        renderDiagram();
      }
    }, [renderDiagram, retryCount]);

    const handleClick = () => {
      if (!renderError) {
        setExpandedDiagram(content);
      }
    };

    return (
      <div className="mermaid-diagram w-full my-6">
        <div
          ref={containerRef}
          className="w-full min-h-[200px] p-6 bg-slack-bgHover rounded border border-slack-border overflow-x-auto overflow-y-visible cursor-pointer hover:bg-slack-accent/10 transition-colors relative"
          onClick={handleClick}
          title={renderError ? "Diagram failed to render" : "Click to expand diagram"}
        >
          {isRendering && (
            <div className="absolute inset-0 flex items-center justify-center bg-slack-bgHover/80 rounded">
              <div className="flex items-center gap-2 text-slack-text">
                <div className="w-4 h-4 border-2 border-slack-accent border-t-transparent rounded-full animate-spin"></div>
                <span className="text-sm">Rendering diagram...</span>
              </div>
            </div>
          )}
          {renderError && (
            <div className="mt-2 flex justify-center">
              <button
                className="retry-button px-3 py-1 bg-slack-accent hover:bg-slack-accentHover text-white rounded text-sm transition-colors"
                onClick={handleRetry}
              >
                Retry Render
              </button>
            </div>
          )}
        </div>
      </div>
    );
  };

  // Parse HTML and replace placeholders with MermaidDiagram components
  const renderContentWithDiagrams = () => {
    if (!renderedHtml) {
      return null;
    }

    if (mermaidBlocks.length === 0) {
      return <div dangerouslySetInnerHTML={{ __html: renderedHtml }} />;
    }

    // Match div placeholders with data-mermaid-placeholder attribute
    // More flexible regex to handle:
    // - Optional whitespace around attributes
    // - Optional whitespace inside tags
    // - Different attribute ordering
    // - Self-closing tags or separate closing tags
    // - Closing tag on same line or separate line
    // - Whitespace between opening and closing tags
    const placeholderRegex = /<div\s+[^>]*data-mermaid-placeholder\s*=\s*["'](\d+)["'][^>]*\s*(?:\/>|>[\s]*<\/div>)/gi;
    const elements: React.ReactNode[] = [];
    let lastIndex = 0;
    let match;
    let foundPlaceholders = 0;
    
    // Find all placeholders and build the element array
    while ((match = placeholderRegex.exec(renderedHtml)) !== null) {
      foundPlaceholders++;
      
      // Add HTML content before this placeholder
      if (match.index > lastIndex) {
        const htmlPart = renderedHtml.substring(lastIndex, match.index);
        if (htmlPart.trim()) {
          elements.push(
            <div 
              key={`html-before-${match.index}`}
              dangerouslySetInnerHTML={{ __html: htmlPart }} 
            />
          );
        }
      }
      
      // Add the mermaid diagram component wrapped in error boundary
      const blockIndex = parseInt(match[1], 10);
      if (blockIndex >= 0 && blockIndex < mermaidBlocks.length) {
        elements.push(
          <ErrorBoundary
            key={`error-boundary-${blockIndex}`}
            fallback={
              <div className="my-6 p-4 bg-red-500/10 border border-red-500/20 rounded text-red-400 text-sm">
                <strong>Diagram Error:</strong> Failed to render Mermaid diagram. Check console for details.
              </div>
            }
          >
            <MermaidDiagram 
              key={`mermaid-${blockIndex}`} 
              content={mermaidBlocks[blockIndex].content} 
              index={blockIndex} 
            />
          </ErrorBoundary>
        );
      } else {
        // Invalid block index - log warning
        if (import.meta.env.DEV) {
          console.warn(`[MarkdownPreview] Invalid mermaid block index: ${blockIndex} (total blocks: ${mermaidBlocks.length})`);
        }
      }
      
      lastIndex = match.index + match[0].length;
    }
    
    // If primary regex didn't find placeholders, try alternative search
    // (DOMPurify or marked might have transformed the HTML structure)
    if (foundPlaceholders === 0 && mermaidBlocks.length > 0) {
      // Try finding data-mermaid-placeholder attribute in any element
      const altRegex = /data-mermaid-placeholder\s*=\s*["'](\d+)["']/gi;
      const altMatches: Array<{ index: number; blockIndex: number }> = [];
      let altMatch;
      
      while ((altMatch = altRegex.exec(renderedHtml)) !== null) {
        const blockIndex = parseInt(altMatch[1], 10);
        if (blockIndex >= 0 && blockIndex < mermaidBlocks.length) {
          altMatches.push({
            index: altMatch.index,
            blockIndex,
          });
        }
      }
      
      // If we found matches with alternative search, use those
      if (altMatches.length > 0) {
        if (import.meta.env.DEV) {
          console.warn(`[MarkdownPreview] Primary regex failed, but found ${altMatches.length} placeholders with alternative search`);
        }
        
        // Build elements from alternative matches
        const altElements: React.ReactNode[] = [];
        let altLastIndex = 0;
        
        altMatches.forEach((altMatchInfo) => {
          // Find the containing tag for this attribute
          const beforeAttr = renderedHtml.substring(0, altMatchInfo.index);
          const tagStart = beforeAttr.lastIndexOf('<');
          const tagEnd = renderedHtml.indexOf('>', altMatchInfo.index);
          
          if (tagStart >= 0 && tagEnd > tagStart) {
            // Add HTML content before this placeholder tag
            if (tagStart > altLastIndex) {
              const htmlPart = renderedHtml.substring(altLastIndex, tagStart);
              if (htmlPart.trim()) {
                altElements.push(
                  <div
                    key={`html-before-alt-${tagStart}`}
                    dangerouslySetInnerHTML={{ __html: htmlPart }}
                  />
                );
              }
            }
            
            // Add the mermaid diagram component
            altElements.push(
              <MermaidDiagram
                key={`mermaid-alt-${altMatchInfo.blockIndex}`}
                content={mermaidBlocks[altMatchInfo.blockIndex].content}
                index={altMatchInfo.blockIndex}
              />
            );
            
            altLastIndex = tagEnd + 1;
          }
        });
        
        // Add remaining HTML
        if (altLastIndex < renderedHtml.length) {
          const htmlPart = renderedHtml.substring(altLastIndex);
          if (htmlPart.trim()) {
            altElements.push(
              <div
                key={`html-after-alt-${altLastIndex}`}
                dangerouslySetInnerHTML={{ __html: htmlPart }}
              />
            );
          }
        }
        
        return <>{altElements}</>;
      }
    }
    
    // Debug logging in development
    if (import.meta.env.DEV) {
      if (mermaidBlocks.length > 0 && foundPlaceholders === 0) {
        console.warn(`[MarkdownPreview] Found ${mermaidBlocks.length} mermaid block(s) but no placeholders in HTML`);
        console.warn(`[MarkdownPreview] HTML snippet:`, renderedHtml.substring(0, 500));
      } else if (foundPlaceholders !== mermaidBlocks.length) {
        console.warn(`[MarkdownPreview] Mismatch: ${mermaidBlocks.length} blocks but ${foundPlaceholders} placeholders found`);
      }
    }
    
    // Add any remaining HTML content after the last placeholder
    if (lastIndex < renderedHtml.length) {
      const htmlPart = renderedHtml.substring(lastIndex);
      if (htmlPart.trim()) {
        elements.push(
          <div 
            key={`html-after-${lastIndex}`}
            dangerouslySetInnerHTML={{ __html: htmlPart }} 
          />
        );
      }
    }

    // If we found placeholders, return the structured elements
    // Otherwise, fall back to rendering the HTML directly (might have been sanitized incorrectly)
    if (foundPlaceholders > 0) {
      return <>{elements}</>;
    } else {
      // Fallback: render HTML directly (diagrams might not render, but at least content shows)
      return <div dangerouslySetInnerHTML={{ __html: renderedHtml }} />;
    }
  };

  if (loading && !content) {
    return (
      <div className="w-full h-screen bg-slack-bg flex items-center justify-center">
        <div className="flex items-center gap-3 text-slack-text">
          <div className="w-6 h-6 border-2 border-slack-accent border-t-transparent rounded-full animate-spin"></div>
          <span>Loading markdown preview...</span>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="w-full h-screen bg-slack-bg flex items-center justify-center">
        <div className="text-center">
          <div className="text-6xl mb-4">⚠️</div>
          <h2 className="text-xl font-bold text-slack-text mb-2">Failed to load file</h2>
          <p className="text-slack-textMuted mb-4">{error}</p>
          <button
            onClick={handleRefresh}
            className="px-4 py-2 bg-slack-accent hover:bg-slack-accentHover text-white rounded transition-colors"
          >
            Retry
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="w-full h-screen bg-slack-bg flex flex-col">
      {/* Header */}
      <div className="bg-slack-bgHover border-b border-slack-border px-4 py-3 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="text-2xl">📝</div>
          <div>
            <h1 className="font-bold text-slack-text">{filename}</h1>
            <p className="text-sm text-slack-textMuted">{filePath}</p>
          </div>
        </div>
        <div className="flex items-center gap-3">
          {lastUpdated && (
            <span className="text-sm text-slack-textMuted">
              Updated at {formatTime(lastUpdated)}
            </span>
          )}
          <button
            onClick={handleRefresh}
            className="px-3 py-1 bg-slack-bg hover:bg-slack-accent text-slack-text hover:text-white rounded transition-colors"
            title="Refresh content"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
          </button>
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-auto">
        <div className="max-w-6xl mx-auto p-6">
          <div className="markdown-content prose prose-invert max-w-none">
            {renderContentWithDiagrams()}
          </div>
        </div>
      </div>

      {/* Mermaid Modal */}
      <MermaidModal
        isOpen={expandedDiagram !== null}
        onClose={() => setExpandedDiagram(null)}
        content={expandedDiagram || ''}
      />

      {/* Loading indicator for auto-refresh */}
      {loading && content && (
        <div className="absolute top-4 right-4 bg-slack-bgHover border border-slack-border rounded px-3 py-2 flex items-center gap-2">
          <div className="w-3 h-3 border border-slack-accent border-t-transparent rounded-full animate-spin"></div>
          <span className="text-sm text-slack-text">Updating...</span>
        </div>
      )}
    </div>
  );
}
