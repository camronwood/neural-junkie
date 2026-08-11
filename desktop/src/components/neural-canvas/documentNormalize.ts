import {
  DOCUMENT_SCHEMA_VERSION,
  emptyDocument,
  type CanvasDocument,
  type DocumentBlock,
  type DocumentTableColumn,
} from './documentTypes';

const HEADING_RE = /^(#{1,6})\s+(.+?)\s*$/;
const UNORDERED_RE = /^[-*+]\s+(.+?)\s*$/;
const ORDERED_RE = /^\d+\.\s+(.+?)\s*$/;
const IMAGE_RE = /^!\[([^\]]*)\]\(([^)\s]+)(?:\s+"[^"]*")?\)$/;
const TABLE_SEP_CELL_RE = /^:?-{3,}:?$/;
const MERMAID_RE = /```\s*mermaid\s*(?:\r?\n)?([\s\S]*?)```/gi;

function stripOuterFence(raw: string): string {
  let s = raw.trim();
  if (!s.startsWith('```')) return s;
  s = s.slice(3);
  const nl = s.indexOf('\n');
  if (nl >= 0) {
    const lang = s.slice(0, nl).trim().toLowerCase();
    if (lang === 'json' || lang === 'document' || lang === '') {
      s = s.slice(nl + 1);
    }
  }
  const end = s.lastIndexOf('```');
  if (end >= 0) s = s.slice(0, end);
  return s.trim();
}

function isDocumentShape(value: unknown): value is CanvasDocument {
  return Boolean(value && typeof value === 'object' && Array.isArray((value as CanvasDocument).blocks));
}

export function parseDocumentJSON(raw: string): CanvasDocument | null {
  const text = stripOuterFence(raw);
  if (!text.startsWith('{')) return null;
  try {
    const parsed: unknown = JSON.parse(text);
    if (!isDocumentShape(parsed)) return null;
    return {
      schema_version: DOCUMENT_SCHEMA_VERSION,
      title: typeof parsed.title === 'string' ? parsed.title : undefined,
      blocks: parsed.blocks,
    };
  } catch {
    return null;
  }
}

export function compileMarkdown(src: string): CanvasDocument {
  const text = src.replace(/\r\n/g, '\n').trim();
  if (!text) return emptyDocument();
  return { schema_version: DOCUMENT_SCHEMA_VERSION, blocks: compileSegments(text) };
}

function compileSegments(src: string): DocumentBlock[] {
  const blocks: DocumentBlock[] = [];
  const re = new RegExp(MERMAID_RE.source, 'gi');
  let cursor = 0;
  let match: RegExpExecArray | null;
  while ((match = re.exec(src)) !== null) {
    if (match.index > cursor) {
      blocks.push(...compileProse(src.slice(cursor, match.index)));
    }
    const source = match[1].trim();
    if (source) blocks.push({ type: 'mermaid', source });
    cursor = match.index + match[0].length;
  }
  if (cursor < src.length) blocks.push(...compileProse(src.slice(cursor)));
  return blocks;
}

function compileProse(src: string): DocumentBlock[] {
  const lines = src.trim().split('\n');
  if (lines.length === 1 && lines[0] === '') return [];
  const blocks: DocumentBlock[] = [];
  let prose: string[] = [];
  let items: string[] = [];
  let ordered = false;
  let inList = false;
  let inCode = false;

  const flushProse = () => {
    const text = prose.join('\n').trim();
    prose = [];
    if (text) blocks.push({ type: 'markdown', source: text });
  };
  const flushList = () => {
    if (!inList) return;
    blocks.push({ type: 'list', ordered, items: [...items] });
    items = [];
    inList = false;
  };

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    const trim = line.trim();

    if (inCode) {
      prose.push(line);
      if (trim.startsWith('```')) inCode = false;
      continue;
    }
    if (trim.startsWith('```')) {
      flushList();
      inCode = true;
      prose.push(line);
      continue;
    }
    if (!trim) {
      flushList();
      flushProse();
      continue;
    }

    const heading = trim.match(HEADING_RE);
    if (heading) {
      flushList();
      flushProse();
      blocks.push({ type: 'heading', level: Math.min(heading[1].length, 3), text: heading[2].trim() });
      continue;
    }

    const image = trim.match(IMAGE_RE);
    if (image) {
      flushList();
      flushProse();
      blocks.push({ type: 'image', alt: image[1], src: image[2] });
      continue;
    }

    if (isTableHeader(trim) && i + 1 < lines.length && isTableSeparator(lines[i + 1].trim())) {
      flushList();
      flushProse();
      const { block, consumed } = parseTable(lines.slice(i));
      blocks.push(block);
      i += consumed - 1;
      continue;
    }

    const unordered = trim.match(UNORDERED_RE);
    if (unordered) {
      flushProse();
      if (inList && ordered) flushList();
      inList = true;
      ordered = false;
      items.push(unordered[1].trim());
      continue;
    }
    const numbered = trim.match(ORDERED_RE);
    if (numbered) {
      flushProse();
      if (inList && !ordered) flushList();
      inList = true;
      ordered = true;
      items.push(numbered[1].trim());
      continue;
    }

    flushList();
    prose.push(line);
  }
  flushList();
  flushProse();
  return blocks;
}

function isTableHeader(line: string): boolean {
  return (line.match(/\|/g) ?? []).length >= 2;
}

function isTableSeparator(line: string): boolean {
  if ((line.match(/\|/g) ?? []).length < 2) return false;
  return splitTableRow(line).every((cell) => !cell || TABLE_SEP_CELL_RE.test(cell));
}

function parseTable(lines: string[]): { block: DocumentBlock; consumed: number } {
  const headers = splitTableRow(lines[0]);
  const used = new Map<string, number>();
  const columns: DocumentTableColumn[] = headers.map((header) => {
    let key = slugKey(header) || 'col';
    const n = used.get(key) ?? 0;
    if (n > 0) key = `${key}_${n + 1}`;
    used.set(slugKey(header) || 'col', n + 1);
    return { key, label: header || key };
  });
  const rows: Record<string, string>[] = [];
  let consumed = 2;
  for (let i = 2; i < lines.length; i++) {
    const trim = lines[i].trim();
    if (!trim || !trim.includes('|')) break;
    if (isTableSeparator(trim)) {
      consumed++;
      continue;
    }
    const cells = splitTableRow(trim);
    const row: Record<string, string> = {};
    columns.forEach((col, index) => {
      row[col.key] = cells[index] ?? '';
    });
    rows.push(row);
    consumed++;
  }
  return { block: { type: 'table', columns, rows }, consumed };
}

function splitTableRow(line: string): string[] {
  let s = line.trim();
  if (s.startsWith('|')) s = s.slice(1);
  if (s.endsWith('|')) s = s.slice(0, -1);
  return s.split('|').map((part) => part.trim());
}

function slugKey(label: string): string {
  return label
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '_')
    .replace(/^_|_$/g, '');
}

export function unwrapDocument(data: unknown): CanvasDocument {
  if (data == null || data === '') return emptyDocument();
  if (isDocumentShape(data)) {
    return {
      schema_version: DOCUMENT_SCHEMA_VERSION,
      title: typeof data.title === 'string' ? data.title : undefined,
      blocks: data.blocks,
    };
  }
  if (typeof data === 'string') {
    return parseDocumentJSON(data) ?? compileMarkdown(data);
  }
  if (typeof data === 'object') {
    const obj = data as Record<string, unknown>;
    for (const key of ['markdown', 'content', 'text', 'body'] as const) {
      if (typeof obj[key] === 'string') return compileMarkdown(obj[key] as string);
    }
  }
  return compileMarkdown(JSON.stringify(data, null, 2));
}

export function documentFromModelOutput(raw: string): CanvasDocument {
  const text = raw.trim();
  if (!text) return emptyDocument();
  return parseDocumentJSON(text) ?? compileMarkdown(text);
}

export function documentToMarkdown(doc: CanvasDocument): string {
  const chunks = doc.blocks.map(blockToMarkdown).filter(Boolean);
  return chunks.length ? `${chunks.join('\n\n')}\n` : '';
}

function blockToMarkdown(block: DocumentBlock): string {
  switch (block.type) {
    case 'heading': {
      const level = Math.min(3, Math.max(1, block.level ?? 1));
      return `${'#'.repeat(level)} ${(block.text ?? '').trim()}`;
    }
    case 'markdown':
      return (block.source ?? '').trim();
    case 'list':
      return (block.items ?? [])
        .map((item, index) => (block.ordered ? `${index + 1}. ${item}` : `- ${item}`))
        .join('\n');
    case 'table':
      return tableToMarkdown(block);
    case 'callout': {
      const title = (block.title ?? '').trim();
      const body = (block.body ?? '').trim();
      if (title && body) return `> **${title}**\n>\n> ${body.replace(/\n/g, '\n> ')}`;
      if (title) return `> **${title}**`;
      if (body) return `> ${body.replace(/\n/g, '\n> ')}`;
      return '';
    }
    case 'mermaid':
      return block.source?.trim() ? `\`\`\`mermaid\n${block.source.trim()}\n\`\`\`` : '';
    case 'image':
      return block.src ? `![${block.alt || block.caption || ''}](${block.src})` : '';
    case 'columns':
      return (block.cols ?? [])
        .map((col) => col.map(blockToMarkdown).filter(Boolean).join('\n\n'))
        .filter(Boolean)
        .join('\n\n');
    default:
      return '';
  }
}

function tableToMarkdown(block: DocumentBlock): string {
  const columns = (block.columns ?? []).flatMap((column): DocumentTableColumn[] => {
    if (typeof column === 'string') return [{ key: column, label: column }];
    if (column && typeof column === 'object' && typeof column.key === 'string') {
      return [{ key: column.key, label: column.label || column.key }];
    }
    return [];
  });
  if (!columns.length) return '';
  const header = `| ${columns.map((col) => col.label || col.key).join(' | ')} |`;
  const sep = `| ${columns.map(() => '---').join(' | ')} |`;
  const rows = (block.rows ?? []).map((row) => {
    if (Array.isArray(row)) {
      return `| ${columns.map((_, index) => String(row[index] ?? '')).join(' | ')} |`;
    }
    if (row && typeof row === 'object') {
      const rec = row as Record<string, unknown>;
      return `| ${columns.map((col) => String(rec[col.key] ?? '')).join(' | ')} |`;
    }
    return `| ${columns.map(() => '').join(' | ')} |`;
  });
  return [header, sep, ...rows].join('\n');
}
