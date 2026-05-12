import { normalizeAgentMessageMarkdown } from './markdownNormalize';

export interface MarkdownContentPart {
  type: 'text' | 'code' | 'mermaid';
  content: string;
  language?: string;
}

/** Split normalized markdown into text / code / mermaid segments (safe for Web Workers). */
export function parseContentParts(raw: string): MarkdownContentPart[] {
  const normalized = normalizeAgentMessageMarkdown(raw);
  const parts: MarkdownContentPart[] = [];
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
