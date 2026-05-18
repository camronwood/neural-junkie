import { marked } from 'marked';
import DOMPurify from 'dompurify';
import { resolveChatImageSrc } from './chatImageSrc';

export interface MermaidBlock {
  type: 'mermaid';
  content: string;
}

export interface MarkdownParseResult {
  html: string;
  mermaidBlocks: MermaidBlock[];
}

// Configure marked for GitHub Flavored Markdown
marked.setOptions({
  breaks: true,
  gfm: true,
});

// Configure DOMPurify for XSS protection
const purify = DOMPurify;

export interface MarkdownRenderOptions {
  sanitize?: boolean;
  breaks?: boolean;
  gfm?: boolean;
}

export function renderMarkdown(
  content: string,
  options: MarkdownRenderOptions = {}
): string {
  const {
    sanitize = true,
    breaks = true,
    gfm = true,
  } = options;

  // Configure marked options
  const markedOptions = {
    breaks,
    gfm,
  };

  // Render markdown to HTML
  let html = marked.parse(content, markedOptions) as string;

  // Sanitize HTML to prevent XSS attacks
  if (sanitize) {
    html = purify.sanitize(html, {
      ALLOWED_TAGS: [
        'h1', 'h2', 'h3', 'h4', 'h5', 'h6',
        'p', 'br', 'hr',
        'strong', 'em', 'u', 's', 'del', 'ins',
        'code', 'pre',
        'blockquote',
        'ul', 'ol', 'li',
        'table', 'thead', 'tbody', 'tr', 'th', 'td',
        'a', 'img',
        'div', 'span',
      ],
      ALLOWED_ATTR: [
        'href', 'title', 'alt', 'src', 'width', 'height',
        'class', 'id', 'style',
        'target', 'rel',
        'data-mermaid-placeholder',
      ],
      ALLOW_DATA_ATTR: false,
    });
  }

  return html;
}

/** Rewrite <img src="…"> for Tauri asset paths and other chat-local URLs. */
export function resolveChatImagesInMarkdownHtml(html: string): string {
  return html.replace(
    /<img\b([^>]*?)\ssrc=(["'])(.*?)\2/gi,
    (_match, attrs: string, quote: string, src: string) => {
      const resolved = resolveChatImageSrc(src);
      return `<img${attrs} src=${quote}${resolved}${quote}`;
    }
  );
}

/** GFM HTML for chat message bodies (same marked pipeline as preview/plan + chat images). */
export function renderChatMarkdown(content: string): string {
  return resolveChatImagesInMarkdownHtml(renderMarkdown(content));
}

export function parseMarkdownWithMermaid(content: string): MarkdownParseResult {
  const mermaidBlocks: MermaidBlock[] = [];
  
  // Find all mermaid code blocks - improved regex to handle various formats:
  // - Case insensitive 'mermaid' keyword
  // - Optional whitespace after 'mermaid'
  // - Different line ending formats (CRLF, LF, CR)
  // - Optional leading/trailing whitespace
  // Optional newline after ```mermaid — agents sometimes use ```mermaid\n or ```mermaid flowchart
  const mermaidRegex = /```\s*mermaid\s*(?:\r?\n)?([\s\S]*?)```/gi;
  const matches: Array<{ fullMatch: string; content: string; index: number }> = [];
  
  // First, collect all matches with their positions
  let match;
  while ((match = mermaidRegex.exec(content)) !== null) {
    const mermaidContent = match[1].trim();
    // Only add non-empty blocks
    if (mermaidContent.length > 0) {
      matches.push({
        fullMatch: match[0],
        content: mermaidContent,
        index: match.index,
      });
    }
  }
  
  // Build processed content by replacing from end to start (to preserve indices)
  let processedContent = content;
  
  // Sort matches by index in descending order for safe replacement
  matches.sort((a, b) => b.index - a.index);
  
  matches.forEach((matchInfo, arrayIndex) => {
    // Store the mermaid block
    const blockIndex = matches.length - 1 - arrayIndex;
    mermaidBlocks.unshift({
      type: 'mermaid',
      content: matchInfo.content,
    });
    
    // Create placeholder
    const placeholder = `<div data-mermaid-placeholder="${blockIndex}"></div>`;
    
    // Replace using indices (working backwards preserves positions)
    const before = processedContent.substring(0, matchInfo.index);
    const after = processedContent.substring(matchInfo.index + matchInfo.fullMatch.length);
    processedContent = before + placeholder + after;
  });
  
  // Render the processed markdown (without mermaid blocks)
  const html = renderMarkdown(processedContent);
  
  // Debug logging in development
  if (import.meta.env.DEV && mermaidBlocks.length > 0) {
    console.log(`[MarkdownRenderer] Found ${mermaidBlocks.length} mermaid block(s)`);
    console.log(`[MarkdownRenderer] HTML contains placeholders: ${html.includes('data-mermaid-placeholder')}`);
  }
  
  return {
    html,
    mermaidBlocks,
  };
}

export type MarkdownSegment =
  | { type: 'markdown'; content: string }
  | { type: 'mermaid'; content: string };

/**
 * Split raw markdown into alternating markdown / mermaid segments
 * **before** any marked/DOMPurify processing. This avoids the fragile
 * placeholder-in-HTML approach entirely.
 */
export function splitMarkdownAndMermaid(raw: string): MarkdownSegment[] {
  // Optional newline after ```mermaid — agents sometimes use ```mermaid\n or ```mermaid flowchart
  const mermaidRegex = /```\s*mermaid\s*(?:\r?\n)?([\s\S]*?)```/gi;
  const segments: MarkdownSegment[] = [];
  let cursor = 0;
  let match: RegExpExecArray | null;

  while ((match = mermaidRegex.exec(raw)) !== null) {
    if (match.index > cursor) {
      segments.push({ type: 'markdown', content: raw.slice(cursor, match.index) });
    }
    const mermaidContent = match[1].trim();
    if (mermaidContent.length > 0) {
      segments.push({ type: 'mermaid', content: mermaidContent });
    }
    cursor = match.index + match[0].length;
  }

  if (cursor < raw.length) {
    segments.push({ type: 'markdown', content: raw.slice(cursor) });
  }

  return segments;
}

export function extractTitle(content: string): string {
  // Extract first heading (h1) from markdown content
  const lines = content.split('\n');
  for (const line of lines) {
    const trimmed = line.trim();
    if (trimmed.startsWith('# ')) {
      return trimmed.substring(2).trim();
    }
  }
  
  // Fallback to first non-empty line
  for (const line of lines) {
    const trimmed = line.trim();
    if (trimmed && !trimmed.startsWith('#')) {
      return trimmed.length > 50 ? trimmed.substring(0, 50) + '...' : trimmed;
    }
  }
  
  return 'Untitled';
}

export function getContentHash(content: string): string {
  // Simple hash function for content comparison
  let hash = 0;
  for (let i = 0; i < content.length; i++) {
    const char = content.charCodeAt(i);
    hash = ((hash << 5) - hash) + char;
    hash = hash & hash; // Convert to 32-bit integer
  }
  return hash.toString(36);
}
