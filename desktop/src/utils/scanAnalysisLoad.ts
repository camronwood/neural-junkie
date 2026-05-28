import type { ChatAPI } from '../api/chatAPI';
import type { ScanAnalysisData } from './scanAnalysis';
import {
  analyteFromSummaryCsvPath,
  mergeScanAnalysisData,
  parseScanAnalysisCsv,
  scanAnalysisDirFromCsvPath,
} from './scanAnalysisCsv';
import { probeLinkedScanDir } from './scanAnalysisLink';
import {
  isScanAnalysisSummaryCSVPath,
  parseScanAnalysisResults,
  resultsRelativePath,
  SCAN_ANALYSIS_REPORTS_DIR,
} from './scanAnalysis';

export interface LoadScanAnalysisOptions {
  csvPath?: string;
  selectedAnalyte?: string;
  linkedScanDir?: string;
}

function reportsDirRelative(analysisDir: string): string {
  const base = analysisDir.replace(/[/\\]+$/, '');
  return base ? `${base}/${SCAN_ANALYSIS_REPORTS_DIR}` : SCAN_ANALYSIS_REPORTS_DIR;
}

async function loadCsvReports(
  api: ChatAPI,
  workspaceId: string,
  analysisDir: string,
  csvPath?: string
): Promise<ScanAnalysisData> {
  if (csvPath && isScanAnalysisSummaryCSVPath(csvPath)) {
    const analyte = analyteFromSummaryCsvPath(csvPath);
    if (!analyte) throw new Error('Could not determine analyte from CSV filename');
    const raw = await api.fetchFileContent(workspaceId, csvPath);
    return parseScanAnalysisCsv(raw, analyte);
  }

  const listing = await api.fetchFiles(workspaceId, reportsDirRelative(analysisDir));
  const csvFiles = listing
    .filter((f: { is_dir?: boolean; name?: string }) => !f.is_dir && /_summary_report\.csv$/i.test(f.name ?? ''))
    .map((f: { path?: string; name?: string }) => f.path ?? `${reportsDirRelative(analysisDir)}/${f.name}`);

  if (csvFiles.length === 0) {
    throw new Error('No reports/results.json or *_summary_report.csv files found');
  }

  const docs: ScanAnalysisData[] = [];
  for (const path of csvFiles) {
    const analyte = analyteFromSummaryCsvPath(path);
    if (!analyte) continue;
    const raw = await api.fetchFileContent(workspaceId, path);
    docs.push(parseScanAnalysisCsv(raw, analyte));
  }
  if (docs.length === 0) throw new Error('Failed to parse any summary report CSV files');
  return mergeScanAnalysisData(docs);
}

export async function loadScanAnalysisData(
  api: ChatAPI,
  workspaceId: string,
  analysisDir: string,
  options?: LoadScanAnalysisOptions
): Promise<{ data: ScanAnalysisData; linkedScanDir: string; source: 'json' | 'csv' }> {
  let data: ScanAnalysisData | null = null;
  let source: 'json' | 'csv' = 'json';

  try {
    const raw = await api.fetchFileContent(workspaceId, resultsRelativePath(analysisDir));
    if (raw && typeof raw === 'string') {
      data = parseScanAnalysisResults(raw);
    }
  } catch {
    data = null;
  }

  if (!data) {
    data = await loadCsvReports(api, workspaceId, analysisDir, options?.csvPath);
    source = 'csv';
  } else if (options?.csvPath && isScanAnalysisSummaryCSVPath(options.csvPath)) {
    const analyte = analyteFromSummaryCsvPath(options.csvPath);
    if (analyte && !data.analytes.includes(analyte)) {
      try {
        const raw = await api.fetchFileContent(workspaceId, options.csvPath);
        const csvDoc = parseScanAnalysisCsv(raw, analyte);
        data = mergeScanAnalysisData([data, csvDoc]);
      } catch {
        /* optional enrich */
      }
    }
  }

  let linkedScanDir = options?.linkedScanDir ?? '';
  if (!linkedScanDir) {
    linkedScanDir = await probeLinkedScanDir(workspaceId, analysisDir, (ws, path) =>
      api.fetchFileContent(ws, path)
    );
  }

  return { data, linkedScanDir, source };
}

export function analysisDirFromFilePath(filePath: string): string {
  if (isScanAnalysisSummaryCSVPath(filePath)) {
    return scanAnalysisDirFromCsvPath(filePath);
  }
  const normalized = filePath.replace(/\\/g, '/');
  if (normalized.endsWith('/reports/results.json')) {
    return normalized.slice(0, -'/reports/results.json'.length);
  }
  return scanAnalysisDirFromCsvPath(filePath);
}
