export const DOCUMENT_RENDERER_ID = 'nj.document';
export const DOCUMENT_MEDIA_TYPE = 'application/vnd.neural-junkie.document+json';
export const DOCUMENT_SCHEMA_VERSION = 1;

export type DocumentBlockType =
  | 'heading'
  | 'markdown'
  | 'list'
  | 'table'
  | 'callout'
  | 'mermaid'
  | 'image'
  | 'columns';

export interface DocumentTableColumn {
  key: string;
  label?: string;
}

export interface DocumentBlock {
  type: string;
  level?: number;
  text?: string;
  source?: string;
  ordered?: boolean;
  items?: string[];
  columns?: Array<string | DocumentTableColumn>;
  rows?: unknown[];
  tone?: 'info' | 'warn' | 'note' | string;
  title?: string;
  body?: string;
  src?: string;
  alt?: string;
  caption?: string;
  cols?: DocumentBlock[][];
}

export interface CanvasDocument {
  schema_version: number;
  title?: string;
  blocks: DocumentBlock[];
}

export function emptyDocument(): CanvasDocument {
  return { schema_version: DOCUMENT_SCHEMA_VERSION, blocks: [] };
}

export function blankDocument(title = 'Canvas'): CanvasDocument {
  const text = title.trim() || 'Canvas';
  return {
    schema_version: DOCUMENT_SCHEMA_VERSION,
    title: text,
    blocks: [{ type: 'heading', level: 1, text }],
  };
}
