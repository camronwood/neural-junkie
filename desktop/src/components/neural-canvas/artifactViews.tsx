import type { ArtifactRendererProps } from './types';

export const textFrom = (value: unknown): string =>
  typeof value === 'string' ? value : JSON.stringify(value, null, 2);

export function EmptyArtifact({ message }: { message: string }) {
  return (
    <div className="flex min-h-24 items-center justify-center rounded border border-dashed border-slate-700 p-4 text-sm text-slate-400">
      {message}
    </div>
  );
}

export interface TableColumn {
  key: string;
  label: string;
}

export function normalizeTable(data: unknown): { columns: TableColumn[]; rows: unknown[] } {
  if (!data || typeof data !== 'object') return { columns: [], rows: [] };
  const source = data as { columns?: unknown; rows?: unknown };
  const rows = Array.isArray(source.rows) ? source.rows : [];
  const supplied = Array.isArray(source.columns) ? source.columns : [];
  const columns = supplied.flatMap((column): TableColumn[] => {
    if (typeof column === 'string') return [{ key: column, label: column }];
    if (!column || typeof column !== 'object') return [];
    const candidate = column as { key?: unknown; label?: unknown };
    if (typeof candidate.key !== 'string') return [];
    return [{
      key: candidate.key,
      label: typeof candidate.label === 'string' ? candidate.label : candidate.key,
    }];
  });
  if (columns.length || !rows.length || !rows[0] || typeof rows[0] !== 'object') {
    return { columns, rows };
  }
  return {
    columns: Object.keys(rows[0] as Record<string, unknown>).map((key) => ({ key, label: key })),
    rows,
  };
}

function tableCell(row: unknown, column: TableColumn, index: number): string {
  if (Array.isArray(row)) return textFrom(row[index] ?? '');
  if (row && typeof row === 'object') return textFrom((row as Record<string, unknown>)[column.key] ?? '');
  return textFrom(row);
}

export function ArtifactTable({
  columns,
  rows,
  compact,
  framed = true,
}: {
  columns: TableColumn[];
  rows: unknown[];
  compact?: boolean;
  framed?: boolean;
}) {
  const visibleRows = compact ? rows.slice(0, 4) : rows;
  return (
    <div className={`overflow-auto ${framed ? 'rounded border border-slate-700' : ''} ${framed && !compact ? '' : ''}`}>
      <table className="w-full border-collapse text-left text-sm">
        <thead className="sticky top-0 bg-slate-900 text-slate-300">
          <tr>{columns.map((column) => (
            <th key={column.key} className="border-b border-slate-700 px-3 py-2 font-medium">
              {column.label}
            </th>
          ))}</tr>
        </thead>
        <tbody>
          {visibleRows.map((row, rowIndex) => (
            <tr key={rowIndex} className="odd:bg-slate-900/30">
              {columns.map((column, columnIndex) => (
                <td key={column.key} className="border-b border-slate-800 px-3 py-2 text-slate-200">
                  {tableCell(row, column, columnIndex)}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

export function TableArtifactRenderer({ artifact, compact }: ArtifactRendererProps) {
  const { columns, rows } = normalizeTable(artifact.data);
  if (!columns.length) return <EmptyArtifact message="No table columns" />;
  return (
    <div className={compact ? '' : 'm-4 h-[calc(100%-2rem)]'}>
      <ArtifactTable columns={columns} rows={rows} compact={compact} />
    </div>
  );
}

export function safeImageSource(value: unknown): string | null {
  if (typeof value !== 'string') return null;
  if (/^(https?:|blob:|data:image\/(?:png|jpeg|gif|webp|svg\+xml);)/i.test(value)) return value;
  if (/(?:^|\/)api\/artifacts\/[^/]+\/assets\//i.test(value)) return value;
  return null;
}

export function ImageArtifactRenderer({ artifact }: ArtifactRendererProps) {
  const data = typeof artifact.data === 'string'
    ? { src: artifact.data, alt: artifact.title, caption: undefined }
    : artifact.data as { src?: unknown; alt?: unknown; caption?: unknown };
  const src = safeImageSource(data?.src);
  if (!src) return <EmptyArtifact message="Image source is not allowed" />;

  return (
    <figure className="flex h-full flex-col items-center justify-center">
      <img
        src={src}
        alt={typeof data.alt === 'string' ? data.alt : artifact.title}
        className="max-h-full max-w-full rounded object-contain"
      />
      {typeof data.caption === 'string' && <figcaption className="mt-2 text-sm text-slate-400">{data.caption}</figcaption>}
    </figure>
  );
}
