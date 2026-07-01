export interface LoraTrainExtraRow {
  row_id?: string;
  instruction: string;
  input?: string;
  output: string;
  source_kind?: string;
  source_ref?: string;
}

export interface LoraTrainDatasetRow extends LoraTrainExtraRow {
  included?: boolean;
  message_at?: string;
}

const MAX_IMPORT_ROWS = 500;

export function parsePastedTrainingRows(text: string): LoraTrainExtraRow[] {
  const trimmed = text.trim();
  if (!trimmed) {
    throw new Error('Paste is empty');
  }
  if (trimmed.startsWith('{') || trimmed.includes('\n{')) {
    return parseJSONLText(trimmed);
  }
  const rows: LoraTrainExtraRow[] = [];
  for (const line of trimmed.split('\n')) {
    const row = line.trim();
    if (!row) continue;
    const parts = row.split('\t');
    if (parts.length < 2) {
      throw new Error('Use JSON lines or tab-separated instruction<TAB>output');
    }
    const instruction = parts[0]?.trim() ?? '';
    const output = parts.slice(1).join('\t').trim();
    validateImportRow(instruction, output);
    rows.push({ instruction, output, source_kind: 'import' });
    if (rows.length >= MAX_IMPORT_ROWS) break;
  }
  if (rows.length === 0) {
    throw new Error('No valid training rows found');
  }
  return rows;
}

export function parseJSONLText(text: string): LoraTrainExtraRow[] {
  const rows: LoraTrainExtraRow[] = [];
  let lineNo = 0;
  for (const line of text.split('\n')) {
    lineNo++;
    const trimmed = line.trim();
    if (!trimmed) continue;
    let parsed: unknown;
    try {
      parsed = JSON.parse(trimmed);
    } catch {
      throw new Error(`Line ${lineNo}: invalid JSON`);
    }
    const row = parsed as LoraTrainExtraRow;
    validateImportRow(row.instruction ?? '', row.output ?? '');
    rows.push({
      instruction: row.instruction.trim(),
      input: row.input?.trim() || undefined,
      output: row.output.trim(),
      source_kind: 'import',
    });
    if (rows.length >= MAX_IMPORT_ROWS) break;
  }
  if (rows.length === 0) {
    throw new Error('No valid training rows found');
  }
  return rows;
}

export async function parseJSONLFile(file: File): Promise<LoraTrainExtraRow[]> {
  const text = await file.text();
  return parseJSONLText(text);
}

function validateImportRow(instruction: string, output: string) {
  if (!instruction.trim()) {
    throw new Error('Each row needs an instruction');
  }
  if (!output.trim()) {
    throw new Error('Each row needs an output');
  }
}

export function truncateTrainingText(text: string, max = 72): string {
  const oneLine = text.replace(/\s+/g, ' ').trim();
  if (!oneLine) return '';
  return oneLine.length <= max ? oneLine : `${oneLine.slice(0, max - 1)}…`;
}

export function sourceKindLabel(kind?: string): string {
  switch (kind) {
    case 'channel':
      return 'chat';
    case 'collaboration':
      return 'collab';
    case 'learning':
      return 'learning';
    case 'import':
      return 'import';
    case 'index':
      return 'index';
    case 'repo':
      return 'repo';
    default:
      return kind?.trim() || 'chat';
  }
}
