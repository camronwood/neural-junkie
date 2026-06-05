import { isScanAnalysisSummaryCSVPath } from './scanAnalysis';

/** True for workspace CSV files that should open in the table editor (not Phoenix summary reports). */
export function isEditableCsvPath(path: string): boolean {
  if (!path) return false;
  const lower = path.toLowerCase();
  if (!lower.endsWith('.csv')) return false;
  return !isScanAnalysisSummaryCSVPath(path);
}

/** Parse one CSV line respecting quoted fields. */
export function parseCsvLine(line: string): string[] {
  const out: string[] = [];
  let cur = '';
  let inQuotes = false;
  for (let i = 0; i < line.length; i++) {
    const ch = line[i];
    if (inQuotes) {
      if (ch === '"') {
        if (line[i + 1] === '"') {
          cur += '"';
          i += 1;
        } else {
          inQuotes = false;
        }
      } else {
        cur += ch;
      }
      continue;
    }
    if (ch === '"') {
      inQuotes = true;
      continue;
    }
    if (ch === ',') {
      out.push(cur);
      cur = '';
      continue;
    }
    cur += ch;
  }
  out.push(cur);
  return out;
}

export function escapeCsvCell(value: string): string {
  if (/[",\n\r]/.test(value)) {
    return `"${value.replace(/"/g, '""')}"`;
  }
  return value;
}

/** Parse CSV text into a rectangular row matrix (pads short rows). */
export function parseCsvTable(raw: string): string[][] {
  const normalized = raw.replace(/\r\n/g, '\n').replace(/\r/g, '\n');
  const lines = normalized.split('\n');
  const rows: string[][] = [];
  let maxCols = 0;
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    const isTrailingBlank = line === '' && i === lines.length - 1;
    if (isTrailingBlank) continue;
    if (line === '' && rows.length === 0) continue;
    const cols = parseCsvLine(line);
    rows.push(cols);
    maxCols = Math.max(maxCols, cols.length);
  }
  if (rows.length === 0) {
    return [['']];
  }
  return rows.map((row) => {
    if (row.length >= maxCols) return row;
    return [...row, ...Array(maxCols - row.length).fill('')];
  });
}

/** Serialize rows back to CSV text (trailing newline omitted). */
export function serializeCsvTable(rows: string[][]): string {
  if (rows.length === 0) return '';
  return rows.map((row) => row.map((cell) => escapeCsvCell(cell ?? '')).join(',')).join('\n');
}
