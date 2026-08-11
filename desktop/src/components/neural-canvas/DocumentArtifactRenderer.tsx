import { useEffect, useState } from 'react';
import { ChatAPI } from '../../api/chatAPI';
import { getHubBaseURL } from '../../config/hubUrl';
import { parseArtifactAssetSrc } from '../../utils/artifactAssetImages';
import { MermaidCanvas } from '../MermaidCanvas';
import { RichMarkdownView } from '../RichMarkdownView';
import { ArtifactTable, EmptyArtifact, normalizeTable, safeImageSource } from './artifactViews';
import { unwrapDocument } from './documentNormalize';
import type { CanvasDocument, DocumentBlock } from './documentTypes';
import type { ArtifactRendererProps, NeuralCanvasArtifact } from './types';

const CALLOUT_TONE: Record<string, string> = {
  info: 'border-sky-400 bg-sky-500/10 text-sky-100',
  warn: 'border-amber-400 bg-amber-500/10 text-amber-100',
  note: 'border-violet-400 bg-violet-500/10 text-violet-100',
};

function headingTag(level: number): 'h1' | 'h2' | 'h3' {
  if (level <= 1) return 'h1';
  if (level === 2) return 'h2';
  return 'h3';
}

function headingClass(level: number, compact?: boolean): string {
  if (level <= 1) return compact ? 'text-xl font-semibold text-slate-100' : 'text-3xl font-semibold text-slate-100';
  if (level === 2) return compact ? 'text-lg font-semibold text-slate-100' : 'text-2xl font-semibold text-slate-100';
  return compact ? 'text-base font-semibold text-slate-200' : 'text-xl font-semibold text-slate-200';
}

function columnChildAllowed(type: string, nested: boolean): boolean {
  if (nested && type === 'columns') return false;
  if (type === 'mermaid') return false;
  return true;
}

export function DocumentArtifactRenderer({ artifact, compact }: ArtifactRendererProps) {
  const doc = unwrapDocument(artifact.data);
  if (!doc.blocks.length) {
    return (
      <div className={compact ? 'p-2 text-sm text-slack-textMuted' : 'h-full overflow-auto p-4 text-sm text-slack-textMuted'}>
        Empty canvas — keep chatting to fill it in
      </div>
    );
  }
  return (
    <div className={compact ? 'space-y-3' : 'h-full space-y-5 overflow-auto p-4'}>
      {doc.blocks.map((block, index) => (
        <DocumentBlockView
          key={`${block.type}-${index}`}
          block={block}
          artifact={artifact}
          compact={compact}
        />
      ))}
    </div>
  );
}

function DocumentBlockView({
  block,
  artifact,
  compact,
  nested = false,
}: {
  block: DocumentBlock;
  artifact: NeuralCanvasArtifact;
  compact?: boolean;
  nested?: boolean;
}) {
  switch (block.type) {
    case 'heading': {
      const Tag = headingTag(block.level ?? 1);
      return <Tag className={headingClass(block.level ?? 1, compact)}>{block.text || 'Untitled'}</Tag>;
    }
    case 'markdown':
      if (!block.source?.trim()) return null;
      return (
        <RichMarkdownView
          content={block.source}
          compact={compact}
          artifactId={typeof artifact.id === 'string' ? artifact.id : undefined}
        />
      );
    case 'list': {
      const items = block.items ?? [];
      if (!items.length) return null;
      const ListTag = block.ordered ? 'ol' : 'ul';
      return (
        <ListTag className={`${block.ordered ? 'list-decimal' : 'list-disc'} space-y-1 pl-6 text-slate-200`}>
          {items.map((item, index) => (
            <li key={index}>{item}</li>
          ))}
        </ListTag>
      );
    }
    case 'table': {
      const { columns, rows } = normalizeTable({ columns: block.columns ?? [], rows: block.rows ?? [] });
      if (!columns.length) return <EmptyArtifact message="No table columns" />;
      return <ArtifactTable columns={columns} rows={rows} compact={compact} />;
    }
    case 'callout': {
      const tone = CALLOUT_TONE[block.tone ?? 'note'] ?? CALLOUT_TONE.note;
      return (
        <aside className={`rounded-md border-l-4 px-3 py-2 ${tone}`}>
          {block.title ? <div className="text-sm font-semibold">{block.title}</div> : null}
          {block.body ? <p className="mt-1 text-sm opacity-90">{block.body}</p> : null}
        </aside>
      );
    }
    case 'mermaid':
      if (!block.source?.trim()) return null;
      return (
        <div className={compact ? 'h-40' : 'min-h-[200px]'}>
          <MermaidCanvas
            content={block.source}
            active
            showZoomControls={!compact}
            className="h-full w-full"
          />
        </div>
      );
    case 'image':
      return (
        <DocumentImage
          src={block.src}
          alt={block.alt}
          caption={block.caption}
          artifactId={typeof artifact.id === 'string' ? artifact.id : undefined}
          fallbackTitle={artifact.title}
        />
      );
    case 'columns': {
      if (nested) return <UnknownBlock block={block} />;
      const cols = (block.cols ?? []).filter((col) => col.length);
      if (!cols.length) return null;
      return (
        <div
          className="grid gap-4"
          style={{ gridTemplateColumns: `repeat(${Math.min(3, Math.max(1, cols.length))}, minmax(0, 1fr))` }}
        >
          {cols.map((col, colIndex) => (
            <div key={colIndex} className="min-w-0 space-y-3">
              {col.map((child, childIndex) => (
                columnChildAllowed(child.type, true) ? (
                  <DocumentBlockView
                    key={`${child.type}-${childIndex}`}
                    block={child}
                    artifact={artifact}
                    compact={compact}
                    nested
                  />
                ) : (
                  <UnknownBlock key={`${child.type}-${childIndex}`} block={child} />
                )
              ))}
            </div>
          ))}
        </div>
      );
    }
    default:
      return <UnknownBlock block={block} />;
  }
}

function DocumentImage({
  src,
  alt,
  caption,
  artifactId,
  fallbackTitle,
}: {
  src?: string;
  alt?: string;
  caption?: string;
  artifactId?: string;
  fallbackTitle: string;
}) {
  const allowed = safeImageSource(src);
  const [resolved, setResolved] = useState<string | null>(allowed);

  useEffect(() => {
    if (!src) {
      setResolved(null);
      return;
    }
    const parsed = parseArtifactAssetSrc(src, artifactId);
    if (!parsed) {
      setResolved(safeImageSource(src));
      return;
    }
    let cancelled = false;
    const api = new ChatAPI(getHubBaseURL());
    void api.fetchArtifactAssetDataUrl(parsed.artifactId, parsed.name).then((dataUrl) => {
      if (!cancelled && dataUrl) setResolved(dataUrl);
    }).catch(() => {
      if (!cancelled) setResolved(safeImageSource(src));
    });
    return () => {
      cancelled = true;
    };
  }, [src, artifactId]);

  if (!resolved) return <EmptyArtifact message="Image source is not allowed" />;
  return (
    <figure className="flex flex-col items-center justify-center">
      <img src={resolved} alt={alt || fallbackTitle} className="max-h-full max-w-full rounded object-contain" />
      {caption ? <figcaption className="mt-2 text-sm text-slate-400">{caption}</figcaption> : null}
    </figure>
  );
}

function UnknownBlock({ block }: { block: DocumentBlock }) {
  return (
    <div className="rounded border border-dashed border-slate-600 p-3 text-xs text-slate-400">
      <div className="font-medium text-slate-300">Unknown block: {block.type || 'untitled'}</div>
      <pre className="mt-2 max-h-32 overflow-auto whitespace-pre-wrap">{JSON.stringify(block, null, 2)}</pre>
    </div>
  );
}

export function documentFromArtifact(artifact: ArtifactRendererProps['artifact']): CanvasDocument {
  return unwrapDocument(artifact.data);
}
