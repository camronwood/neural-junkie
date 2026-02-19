import React, { useEffect, useRef, useState } from 'react';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { vscDarkPlus } from 'react-syntax-highlighter/dist/esm/styles/prism';
import mermaid from 'mermaid';
import { MermaidModal } from './MermaidModal';
import { getContentHash } from '../utils/markdownRenderer';

interface MessageContentProps {
  content: string;
}

// Initialize mermaid with configuration
mermaid.initialize({
  startOnLoad: false,
  theme: 'dark',
  securityLevel: 'loose',
  fontFamily: 'ui-monospace, monospace',
});

interface ContentPart {
  type: 'text' | 'code' | 'mermaid';
  content: string;
  language?: string;
}

// Parse markdown syntax and convert to React elements
function parseMarkdownToElements(text: string): React.ReactNode[] {
  const elements: React.ReactNode[] = [];
  let currentIndex = 0;

  // Combined regex to find all markdown patterns
  const combinedRegex = /(\*\*(.+?)\*\*|\*(.+?)\*|`([^`]+)`|\[([^\]]+)\]\(([^)]+)\))/g;
  
  let match;
  const matches: Array<{ index: number; length: number; type: string; content: string; url?: string }> = [];
  
  // Find all matches
  while ((match = combinedRegex.exec(text)) !== null) {
    if (match[2]) {
      // Bold **text**
      matches.push({
        index: match.index,
        length: match[0].length,
        type: 'strong',
        content: match[2],
      });
    } else if (match[3]) {
      // Italic *text*
      matches.push({
        index: match.index,
        length: match[0].length,
        type: 'em',
        content: match[3],
      });
    } else if (match[4]) {
      // Inline code `text`
      matches.push({
        index: match.index,
        length: match[0].length,
        type: 'code',
        content: match[4],
      });
    } else if (match[5] && match[6]) {
      // Link [text](url)
      matches.push({
        index: match.index,
        length: match[0].length,
        type: 'link',
        content: match[5],
        url: match[6],
      });
    }
  }

  // Build elements array
  matches.forEach((m, idx) => {
    // Add text before this match
    if (m.index > currentIndex) {
      elements.push(text.substring(currentIndex, m.index));
    }

    // Add the formatted element
    switch (m.type) {
      case 'strong':
        elements.push(
          <strong key={`strong-${idx}`} className="font-bold text-slack-text">
            {m.content}
          </strong>
        );
        break;
      case 'em':
        elements.push(
          <em key={`em-${idx}`} className="italic text-slack-text">
            {m.content}
          </em>
        );
        break;
      case 'code':
        elements.push(
          <code
            key={`code-${idx}`}
            className="px-1.5 py-0.5 bg-slack-bgHover text-slack-accent rounded text-sm font-mono border border-slack-border"
          >
            {m.content}
          </code>
        );
        break;
      case 'link':
        elements.push(
          <a
            key={`link-${idx}`}
            href={m.url}
            target="_blank"
            rel="noopener noreferrer"
            className="text-slack-accent hover:underline"
          >
            {m.content}
          </a>
        );
        break;
    }

    currentIndex = m.index + m.length;
  });

  // Add any remaining text
  if (currentIndex < text.length) {
    elements.push(text.substring(currentIndex));
  }

  // If no matches found, return the original text
  return elements.length > 0 ? elements : [text];
}

function parseContent(content: string): ContentPart[] {
  const parts: ContentPart[] = [];
  const codeBlockRegex = /```(\w+)?\n([\s\S]*?)```/g;
  let lastIndex = 0;
  let match;

  while ((match = codeBlockRegex.exec(content)) !== null) {
    // Add text before code block
    if (match.index > lastIndex) {
      const textContent = content.substring(lastIndex, match.index);
      if (textContent.trim()) {
        parts.push({ type: 'text', content: textContent });
      }
    }

    const language = match[1] || 'text';
    const code = match[2].trim();

    // Check if it's a mermaid diagram
    if (language.toLowerCase() === 'mermaid') {
      parts.push({ type: 'mermaid', content: code });
    } else {
      parts.push({ type: 'code', content: code, language });
    }

    lastIndex = match.index + match[0].length;
  }

  // Add remaining text
  if (lastIndex < content.length) {
    const textContent = content.substring(lastIndex);
    if (textContent.trim()) {
      parts.push({ type: 'text', content: textContent });
    }
  }

  // If no code blocks found, return the entire content as text
  if (parts.length === 0) {
    parts.push({ type: 'text', content });
  }

  return parts;
}

function MermaidDiagram({ content, onClick }: { content: string; onClick: () => void }) {
  const containerRef = useRef<HTMLDivElement>(null);
  const [isRendering, setIsRendering] = useState(false);
  const [renderError, setRenderError] = useState<string | null>(null);
  const [retryCount, setRetryCount] = useState(0);
  
  // Create stable ID based on content hash
  const contentHash = getContentHash(content);
  const idRef = useRef(`mermaid-${contentHash}-${Math.random().toString(36).substr(2, 9)}`);

  const renderDiagram = async () => {
    if (!containerRef.current) return;
    
    setIsRendering(true);
    setRenderError(null);
    
    try {
      // Clear previous content
      containerRef.current.innerHTML = '';
      
      // Render the diagram
      const { svg } = await mermaid.render(idRef.current, content);
      containerRef.current.innerHTML = svg;
      setRetryCount(0); // Reset retry count on success
    } catch (error) {
      console.error('Mermaid rendering error:', error);
      const errorMessage = error instanceof Error ? error.message : String(error);
      setRenderError(errorMessage);
      containerRef.current.innerHTML = `
        <div class="p-4 bg-red-500/10 border border-red-500/20 rounded text-red-400 text-sm">
          <strong>Mermaid Diagram Error:</strong>
          <pre class="mt-2 text-xs">${errorMessage}</pre>
          <button 
            class="mt-2 px-3 py-1 bg-red-500/20 hover:bg-red-500/30 rounded text-xs transition-colors"
            onclick="this.closest('.mermaid-diagram').querySelector('.retry-button').click()"
          >
            Retry
          </button>
        </div>
      `;
    } finally {
      setIsRendering(false);
    }
  };

  const handleRetry = () => {
    setRetryCount(prev => prev + 1);
    renderDiagram();
  };

  useEffect(() => {
    renderDiagram();
  }, [content, retryCount]);

  const handleClick = () => {
    if (!renderError) {
      onClick();
    }
  };

  return (
    <div className="mermaid-diagram">
      <div
        ref={containerRef}
        className="my-3 p-4 bg-slack-bgHover rounded border border-slack-border overflow-x-auto cursor-pointer hover:bg-slack-accent/10 transition-colors"
        onClick={handleClick}
        title={renderError ? "Diagram failed to render" : "Click to expand diagram"}
      />
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
  );
}

function CodeBlock({ content, language }: { content: string; language: string }) {
  return (
    <div className="my-3 rounded overflow-hidden border border-slack-border">
      {/* Language label */}
      <div className="px-3 py-1 bg-slack-bgHover border-b border-slack-border text-xs text-slack-textMuted font-mono">
        {language}
      </div>
      
      {/* Code content */}
      <SyntaxHighlighter
        language={language}
        style={vscDarkPlus}
        customStyle={{
          margin: 0,
          padding: '1rem',
          background: '#1e1e1e',
          fontSize: '0.875rem',
        }}
        showLineNumbers={true}
        wrapLongLines={false}
      >
        {content}
      </SyntaxHighlighter>
    </div>
  );
}

export function MessageContent({ content }: MessageContentProps) {
  const [expandedDiagram, setExpandedDiagram] = useState<string | null>(null);
  const parts = parseContent(content);

  const handleDiagramClick = (diagramContent: string) => {
    setExpandedDiagram(diagramContent);
  };

  return (
    <>
      {parts.map((part, index) => {
        if (part.type === 'mermaid') {
          return (
            <MermaidDiagram 
              key={`mermaid-${getContentHash(part.content)}-${index}`} 
              content={part.content} 
              onClick={() => handleDiagramClick(part.content)}
            />
          );
        } else if (part.type === 'code') {
          return (
            <CodeBlock
              key={`code-${getContentHash(part.content)}-${index}`}
              content={part.content}
              language={part.language || 'text'}
            />
          );
        } else {
          // Parse markdown in text content
          const markdownElements = parseMarkdownToElements(part.content);
          return (
            <div
              key={`text-${getContentHash(part.content)}-${index}`}
              className="message-content leading-relaxed whitespace-pre-wrap"
            >
              {markdownElements}
            </div>
          );
        }
      })}
      
      {/* Mermaid Modal */}
      <MermaidModal
        isOpen={expandedDiagram !== null}
        onClose={() => setExpandedDiagram(null)}
        content={expandedDiagram || ''}
      />
    </>
  );
}

