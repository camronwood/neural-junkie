import React, { useEffect, useMemo, useRef, useState, memo } from 'react';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { vscDarkPlus } from 'react-syntax-highlighter/dist/esm/styles/prism';
import { renderMermaidSvg } from '../utils/mermaidConfig';
import { MermaidModal } from './MermaidModal';
import { mapHighlighterLanguage, normalizeAgentMessageMarkdown } from '../utils/markdownNormalize';
import {
  type ContentPart,
  getCachedContentParts,
  getCachedMarkdownElements,
} from '../utils/messageContentCache';
import { perfMarkEnd, perfMarkStart } from '../utils/perfMarks';

export type { ContentPart } from '../utils/messageContentCache';

const CODE_BLOCK_CUSTOM_STYLE: React.CSSProperties = {
  margin: 0,
  padding: '0.75rem 1rem',
  background: '#1e1e1e',
  fontSize: '0.8125rem',
  lineHeight: 1.55,
};

const CODE_TAG_PROPS = {
  style: {
    whiteSpace: 'pre-wrap' as const,
    wordBreak: 'break-word' as const,
  },
};

interface MessageContentProps {
  content: string;
  /** When true, skip Prism/Mermaid/markdown parsing (plain text only). */
  isStreaming?: boolean;
}

// Parse markdown syntax and convert to React elements
function parseMarkdownToElements(text: string): React.ReactNode[] {
  const elements: React.ReactNode[] = [];
  let currentIndex = 0;

  const combinedRegex = /(\*\*(.+?)\*\*|\*(.+?)\*|`([^`]+)`|\[([^\]]+)\]\(([^)]+)\))/g;

  let match;
  const matches: Array<{ index: number; length: number; type: string; content: string; url?: string }> = [];

  while ((match = combinedRegex.exec(text)) !== null) {
    if (match[2]) {
      matches.push({
        index: match.index,
        length: match[0].length,
        type: 'strong',
        content: match[2],
      });
    } else if (match[3]) {
      matches.push({
        index: match.index,
        length: match[0].length,
        type: 'em',
        content: match[3],
      });
    } else if (match[4]) {
      matches.push({
        index: match.index,
        length: match[0].length,
        type: 'code',
        content: match[4],
      });
    } else if (match[5] && match[6]) {
      matches.push({
        index: match.index,
        length: match[0].length,
        type: 'link',
        content: match[5],
        url: match[6],
      });
    }
  }

  matches.forEach((m, idx) => {
    if (m.index > currentIndex) {
      elements.push(text.substring(currentIndex, m.index));
    }

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

  if (currentIndex < text.length) {
    elements.push(text.substring(currentIndex));
  }

  return elements.length > 0 ? elements : [text];
}

function parseContent(content: string): ContentPart[] {
  const normalized = normalizeAgentMessageMarkdown(content);
  const parts: ContentPart[] = [];
  const codeBlockRegex = /```([\w-]*)?\s*\r?\n([\s\S]*?)```/g;
  let lastIndex = 0;
  let match: RegExpExecArray | null;

  while ((match = codeBlockRegex.exec(normalized)) !== null) {
    if (match.index > lastIndex) {
      const textContent = normalized.substring(lastIndex, match.index);
      if (textContent.trim()) {
        parts.push({ type: 'text', content: textContent });
      }
    }

    const language = match[1] || 'text';
    const code = match[2].trim();

    if (language.toLowerCase() === 'mermaid') {
      parts.push({ type: 'mermaid', content: code });
    } else {
      parts.push({ type: 'code', content: code, language });
    }

    lastIndex = match.index + match[0].length;
  }

  if (lastIndex < normalized.length) {
    const textContent = normalized.substring(lastIndex);
    if (textContent.trim()) {
      parts.push({ type: 'text', content: textContent });
    }
  }

  if (parts.length === 0) {
    parts.push({ type: 'text', content: normalized });
  }

  return parts;
}

function MermaidDiagram({ content, onClick }: { content: string; onClick: () => void }) {
  const containerRef = useRef<HTMLDivElement>(null);
  const [isRendering, setIsRendering] = useState(false);
  const [renderError, setRenderError] = useState<string | null>(null);
  const [retryCount, setRetryCount] = useState(0);

  const renderDiagram = async () => {
    if (!containerRef.current) return;

    setIsRendering(true);
    setRenderError(null);

    try {
      containerRef.current.innerHTML = '';
      const svg = await renderMermaidSvg(content);
      containerRef.current.innerHTML = svg;
      setRetryCount(0);
    } catch (error) {
      console.error('Mermaid rendering error:', error);
      const errorMessage = error instanceof Error ? error.message : String(error);
      setRenderError(errorMessage);
    } finally {
      setIsRendering(false);
    }
  };

  const handleRetry = () => {
    setRetryCount((prev) => prev + 1);
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
    <div className="mermaid-diagram relative">
      {renderError ? (
        <div className="my-3 p-4 bg-red-500/10 border border-red-500/20 rounded text-red-400 text-sm">
          <strong>Mermaid Diagram Error:</strong>
          <pre className="mt-2 text-xs whitespace-pre-wrap">{renderError}</pre>
          <button
            className="mt-2 px-3 py-1 bg-red-500/20 hover:bg-red-500/30 rounded text-xs transition-colors"
            onClick={handleRetry}
          >
            Retry
          </button>
        </div>
      ) : (
        <div
          ref={containerRef}
          className="my-3 p-4 bg-slack-bgHover rounded border border-slack-border overflow-x-auto cursor-pointer hover:bg-slack-accent/10 transition-colors"
          onClick={handleClick}
          title="Click to expand diagram"
        />
      )}
      {isRendering && (
        <div className="absolute inset-0 flex items-center justify-center bg-slack-bgHover/80 rounded">
          <div className="flex items-center gap-2 text-slack-text">
            <div className="w-4 h-4 border-2 border-slack-accent border-t-transparent rounded-full animate-spin" />
            <span className="text-sm">Rendering diagram...</span>
          </div>
        </div>
      )}
    </div>
  );
}

const CodeBlock = memo(function CodeBlockImpl({ content, language }: { content: string; language: string }) {
  const hl = mapHighlighterLanguage(language || 'text');
  const showLineNumbers = useMemo(() => content.split('\n').length <= 40, [content]);
  return (
    <div className="my-3 overflow-hidden rounded-md border border-slack-border shadow-sm">
      <div className="border-b border-slack-border bg-slack-bgHover px-3 py-1.5 text-xs font-mono text-slack-textMuted">
        {language || 'text'}
      </div>
      <SyntaxHighlighter
        language={hl}
        style={vscDarkPlus}
        customStyle={CODE_BLOCK_CUSTOM_STYLE}
        codeTagProps={CODE_TAG_PROPS}
        showLineNumbers={showLineNumbers}
        wrapLongLines
      >
        {content}
      </SyntaxHighlighter>
    </div>
  );
});

export function MessageContent({ content, isStreaming }: MessageContentProps) {
  const [expandedDiagram, setExpandedDiagram] = useState<string | null>(null);

  /**
   * Hooks must run in a stable order — always call useMemo. When streaming,
   * we skip the parse work by returning an empty array, then the render path
   * below renders a plain <pre> instead of the parsed pipeline.
   */
  const parts = useMemo(() => {
    if (isStreaming) return [];
    perfMarkStart('messageContent.parse');
    const p = getCachedContentParts(content, parseContent);
    perfMarkEnd('messageContent.parse');
    return p;
  }, [content, isStreaming]);

  const handleDiagramClick = (diagramContent: string) => {
    setExpandedDiagram(diagramContent);
  };

  if (isStreaming) {
    return (
      <>
        <pre className="message-content leading-relaxed whitespace-pre-wrap font-sans text-slack-text text-sm m-0">
          {content}
        </pre>
        <MermaidModal isOpen={expandedDiagram !== null} onClose={() => setExpandedDiagram(null)} content={expandedDiagram || ''} />
      </>
    );
  }

  return (
    <>
      {/* Plain markdown strings inherit color here; Message.tsx also sets text-slack-text. */}
      <div className="text-slack-text">
        {parts.map((part, index) => {
          if (part.type === 'mermaid') {
            return (
              <MermaidDiagram
                key={`mermaid-${index}`}
                content={part.content}
                onClick={() => handleDiagramClick(part.content)}
              />
            );
          }
          if (part.type === 'code') {
            return (
              <CodeBlock key={`code-${index}`} content={part.content} language={part.language || 'text'} />
            );
          }
          const markdownElements = getCachedMarkdownElements(part.content, parseMarkdownToElements);
          return (
            <div key={`text-${index}`} className="message-content leading-relaxed whitespace-pre-wrap">
              {markdownElements}
            </div>
          );
        })}
      </div>

      <MermaidModal isOpen={expandedDiagram !== null} onClose={() => setExpandedDiagram(null)} content={expandedDiagram || ''} />
    </>
  );
}
