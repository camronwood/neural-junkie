/** Phoenix-style scan summary export: imageMetadata.json + extensionless well TIFFs. */

export const SCAN_SUMMARY_METADATA_FILE = 'imageMetadata.json';

export const WELL_ID_PATTERN = /^[A-H](?:[1-9]|1[0-2])$/;

export interface ScanSummarySpot {
  analyte: string;
  row: string;
  column: string;
  x_px: number;
  y_px: number;
}

export interface ScanSummaryWellMeta {
  imageName: string;
  time?: string;
  fovSizeXUm?: number;
  fovSizeYUm?: number;
  zStagePositionUm?: number;
  xStagePositionUm?: number;
  yStagePositionUm?: number;
  spots: ScanSummarySpot[];
}

export interface ScanSummaryData {
  metadata: ScanSummaryWellMeta[];
  byWell: Map<string, ScanSummaryWellMeta>;
}

export function isScanSummaryMetadataPath(path: string): boolean {
  if (!path) return false;
  const base = path.split(/[/\\]/).pop() ?? path;
  return base === SCAN_SUMMARY_METADATA_FILE;
}

/** Summary directory if listing includes imageMetadata.json. */
export function isScanSummaryDirListing(files: { name: string; is_dir?: boolean }[]): boolean {
  return files.some((f) => !f.is_dir && f.name === SCAN_SUMMARY_METADATA_FILE);
}

export function scanSummaryDirFromMetadataPath(metadataPath: string): string {
  const idx = Math.max(metadataPath.lastIndexOf('/'), metadataPath.lastIndexOf('\\'));
  return idx <= 0 ? '' : metadataPath.slice(0, idx);
}

/** True for extensionless well files A1–H12. */
export function isScanSummaryWellPath(path: string): boolean {
  if (!path) return false;
  const base = path.split(/[/\\]/).pop() ?? path;
  return WELL_ID_PATTERN.test(base);
}

/** Parent directory of a file path within the workspace (empty if at workspace root). */
export function parentDirFromFilePath(filePath: string): string {
  const idx = Math.max(filePath.lastIndexOf('/'), filePath.lastIndexOf('\\'));
  return idx <= 0 ? '' : filePath.slice(0, idx);
}

/** Resolve scan-summary folder for a well or metadata path. */
export function scanSummaryDirForFilePath(filePath: string): string {
  if (isScanSummaryMetadataPath(filePath)) {
    return scanSummaryDirFromMetadataPath(filePath);
  }
  if (isScanSummaryWellPath(filePath)) {
    return parentDirFromFilePath(filePath);
  }
  return '';
}

/** Workspace root is a scan summary when imageMetadata.json is at the top level. */
export function isScanSummaryWorkspaceRoot(files: { name: string; is_dir?: boolean }[]): boolean {
  return isScanSummaryDirListing(files);
}

export function wellImageRelativePath(summaryDir: string, wellId: string): string {
  const dir = summaryDir.replace(/[/\\]+$/, '');
  return dir ? `${dir}/${wellId}` : wellId;
}

export function parseScanSummaryMetadata(raw: string): ScanSummaryData {
  const parsed = JSON.parse(raw) as { metadata?: ScanSummaryWellMeta[] };
  if (!parsed || !Array.isArray(parsed.metadata)) {
    throw new Error('Invalid scan summary: expected { metadata: [...] }');
  }
  const metadata = parsed.metadata.map((m) => ({
    ...m,
    imageName: String(m.imageName ?? ''),
    spots: Array.isArray(m.spots) ? m.spots : [],
  }));
  const byWell = new Map<string, ScanSummaryWellMeta>();
  for (const well of metadata) {
    if (well.imageName) {
      byWell.set(well.imageName, well);
    }
  }
  return { metadata, byWell };
}

export const PLATE_ROWS = ['A', 'B', 'C', 'D', 'E', 'F', 'G', 'H'] as const;
export const PLATE_COLS = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12] as const;

export function allPlateWellIds(): string[] {
  const ids: string[] = [];
  for (const row of PLATE_ROWS) {
    for (const col of PLATE_COLS) {
      ids.push(`${row}${col}`);
    }
  }
  return ids;
}

/** Fixed palette for 12-plex + controls (stable across wells). */
export const ANALYTE_COLORS: Record<string, string> = {
  BLANK: '#6b7280',
  POS: '#22c55e',
  'IFN-gamma': '#ef4444',
  'IL-10': '#f97316',
  'IL-13': '#eab308',
  'IL-1alpha': '#84cc16',
  'IL-1beta': '#14b8a6',
  'IL-2': '#06b6d4',
  'IL-4': '#3b82f6',
  'IL-5': '#6366f1',
  'IL-6': '#8b5cf6',
  'IL-8': '#a855f7',
  'IL-12p70': '#ec4899',
  'TNF-alpha': '#f43f5e',
};

export function analyteColor(analyte: string): string {
  return ANALYTE_COLORS[analyte] ?? '#94a3b8';
}

export function runLabelFromDirPath(dirPath: string): string {
  const base = dirPath.split(/[/\\]/).pop() ?? dirPath;
  return base.replace(/-summary$/i, '').trim() || base;
}
