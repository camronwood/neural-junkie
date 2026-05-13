import type { ReactNode } from 'react';

export interface ContentPart {
  type: 'text' | 'code' | 'mermaid';
  content: string;
  language?: string;
}

const MAX_PARSE_ENTRIES = 100;

/** LRU-ish: delete oldest when over capacity (Map iteration order = insertion order). */
function trimMap<K, V>(map: Map<K, V>, max: number) {
  while (map.size > max) {
    const first = map.keys().next().value as K;
    map.delete(first);
  }
}

const parseContentPartsCache = new Map<string, ContentPart[]>();

export function getCachedContentParts(
  content: string,
  parseContent: (raw: string) => ContentPart[]
): ContentPart[] {
  const hit = parseContentPartsCache.get(content);
  if (hit) return hit;
  const parts = parseContent(content);
  parseContentPartsCache.set(content, parts);
  trimMap(parseContentPartsCache, MAX_PARSE_ENTRIES);
  return parts;
}

const markdownElementsCache = new Map<string, ReactNode[]>();

export function getCachedMarkdownElements(
  text: string,
  parseMarkdownToElements: (t: string) => ReactNode[]
): ReactNode[] {
  const hit = markdownElementsCache.get(text);
  if (hit) return hit;
  const nodes = parseMarkdownToElements(text);
  markdownElementsCache.set(text, nodes);
  trimMap(markdownElementsCache, MAX_PARSE_ENTRIES);
  return nodes;
}
