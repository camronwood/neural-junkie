/** Phoenix-style scan analysis export: reports/results.json + plots/*.jpg */

import {
  PLATE_COLS,
  PLATE_ROWS,
  SCAN_SUMMARY_METADATA_FILE,
  analyteColor,
} from './scanSummary';

export {
  PLATE_COLS,
  PLATE_ROWS,
  analyteColor,
};

export const SCAN_ANALYSIS_RESULTS_FILE = 'reports/results.json';
export const SCAN_ANALYSIS_REPORTS_DIR = 'reports';
export const SCAN_ANALYSIS_PLOTS_DIR = 'plots';
export const SCAN_ANALYSIS_PROCESS_REPORT = 'reports/process_report.txt';

export type PlateGridMode = 'concentration' | 'intensity' | 'loq' | 'wellType';

export interface ScanAnalysisHeader {
  analysisDate?: string;
  analysisId?: string;
  scanResultsId?: string;
  scanDate?: string;
  productId?: string;
}

export interface ScanAnalysisExperiment {
  analysisPlateMapId?: string;
  analysisName?: string;
  analysisPlateMapName?: string;
  productName?: string;
  plateBarcode?: string;
  scanName?: string;
  signalCalculationAlgorithm?: string;
  initialConcentrations: Record<string, number>;
  dilutionFactor?: number;
}

export interface ScanAnalysisStandardRow {
  analyte: string;
  wellLabel: string;
  concentration: number;
  replicates: Record<string, number>;
  meanReplicateIntensity?: number | null;
  meanReplicateCalculatedConcentration?: number | null;
  percentBias?: number | null;
  withinLimitsOfQuantificationV2: boolean;
  upperPercentDifferenceV2?: number | null;
  lowerPercentDifferenceV2?: number | null;
}

export interface ScanAnalysisUnknownReplicate {
  replicateIndex: number;
  signal: number;
  concentration?: number | null;
}

export interface ScanAnalysisUnknownRow {
  analyte: string;
  wellLabel: string;
  replicates: ScanAnalysisUnknownReplicate[];
  meanReplicateConcentration?: number | null;
  stdevOfReplicateConcentration?: number | null;
  withinLimitsOfQuantification: boolean;
  concentrationUnit?: string;
}

export interface ScanAnalysisValidationRow {
  analyte: string;
  signal: number;
  wellRow: string;
  wellColumn: number;
  wellReplicateIndex?: number;
  wellType: string;
  wellLabel: string;
  calculatedConcentration?: number | null;
}

export interface ScanAnalysisSpotIntensity {
  wellRow: string;
  wellColumn: number;
  wellReplicateIndex?: number;
  wellType: string;
  wellLabel: string;
  analyte: string;
  row: number;
  column: number;
  signal: number;
  background: number;
  signalIntensityAlgorithm?: string;
}

export interface ScanAnalysisLOQ {
  LLOQ?: string;
  ULOQ?: string;
  LOD_label?: string;
  LOD?: string;
  concentration_units?: string;
}

export interface ScanAnalysisFitParams {
  a?: number | null;
  b?: number | null;
  c?: number | null;
  d?: number | null;
  g?: number | null;
}

export interface ScanAnalysisData {
  header: ScanAnalysisHeader;
  experiment: ScanAnalysisExperiment;
  standardReport: Record<string, ScanAnalysisStandardRow[]>;
  unknownReport: Record<string, ScanAnalysisUnknownRow[]>;
  validation: ScanAnalysisValidationRow[];
  spotIntensities: ScanAnalysisSpotIntensity[];
  limitsOfQuant: Record<string, ScanAnalysisLOQ>;
  fitParameters: Record<string, ScanAnalysisFitParams>;
  analytes: string[];
  byWellAnalyte: Map<string, ScanAnalysisValidationRow>;
  byWell: Map<string, ScanAnalysisValidationRow[]>;
  spotsByWellAnalyte: Map<string, ScanAnalysisSpotIntensity[]>;
}

export function normalizeJSONNaN(raw: string): string {
  return raw.replace(/\bNaN\b/g, 'null');
}

export function wellIdFromRowCol(row: string, col: number): string {
  return `${row}${col}`;
}

export function wellAnalyteKey(wellId: string, analyte: string): string {
  return `${wellId}|${analyte}`;
}

export function parseWellId(wellId: string): { row: string; col: number } | null {
  if (wellId.length < 2) return null;
  const row = wellId.slice(0, 1);
  const col = parseInt(wellId.slice(1), 10);
  if (!PLATE_ROWS.includes(row as (typeof PLATE_ROWS)[number]) || col < 1 || col > 12) {
    return null;
  }
  return { row, col };
}

export function isScanAnalysisResultsPath(path: string): boolean {
  if (!path) return false;
  const normalized = path.replace(/\\/g, '/');
  return normalized.endsWith('/reports/results.json') || normalized === 'reports/results.json';
}

export function isScanAnalysisProcessReportPath(path: string): boolean {
  if (!path) return false;
  const normalized = path.replace(/\\/g, '/');
  return normalized.endsWith('/reports/process_report.txt') || normalized === 'reports/process_report.txt';
}

export function isScanAnalysisSummaryCSVPath(path: string): boolean {
  if (!path) return false;
  const base = path.split(/[/\\]/).pop() ?? path;
  return /_summary_report\.csv$/i.test(base);
}

/** True when a directory listing contains a reports/ subfolder (results loaded on open). */
export function isScanAnalysisDirListing(files: { name: string; is_dir?: boolean }[]): boolean {
  return files.some((f) => f.is_dir && f.name === SCAN_ANALYSIS_REPORTS_DIR);
}

/** True when listing includes reports/results.json directly. */
export function isScanAnalysisReportsListing(files: { name: string; is_dir?: boolean }[]): boolean {
  return files.some((f) => !f.is_dir && f.name === 'results.json');
}

export function isScanAnalysisRootListing(
  files: { name: string; is_dir?: boolean; children?: { name: string; is_dir?: boolean }[] }[]
): boolean {
  const reportsDir = files.find((f) => f.is_dir && f.name === SCAN_ANALYSIS_REPORTS_DIR);
  if (reportsDir?.children?.some((c) => !c.is_dir && (c.name === 'results.json' || /_summary_report\.csv$/i.test(c.name)))) {
    return true;
  }
  return false;
}

export function scanAnalysisDirFromResultsPath(resultsPath: string): string {
  const normalized = resultsPath.replace(/\\/g, '/');
  if (normalized.endsWith('/reports/results.json')) {
    return normalized.slice(0, -'/reports/results.json'.length);
  }
  if (normalized === 'reports/results.json') {
    return '';
  }
  const idx = Math.max(resultsPath.lastIndexOf('/'), resultsPath.lastIndexOf('\\'));
  if (idx <= 0) return '';
  const parent = resultsPath.slice(0, idx);
  const parentBase = parent.split(/[/\\]/).pop() ?? '';
  if (parentBase === SCAN_ANALYSIS_REPORTS_DIR) {
    const grandIdx = Math.max(parent.lastIndexOf('/'), parent.lastIndexOf('\\'));
    return grandIdx <= 0 ? '' : parent.slice(0, grandIdx);
  }
  return parent;
}

export function isCombinedRunDirListing(files: { name: string; is_dir?: boolean }[]): boolean {
  const hasMetadata = files.some((f) => !f.is_dir && f.name === SCAN_SUMMARY_METADATA_FILE);
  const hasReports = files.some((f) => f.is_dir && f.name === SCAN_ANALYSIS_REPORTS_DIR);
  return hasMetadata && hasReports;
}

export function resolveLinkedScanDir(analysisDir: string): string {
  const dir = analysisDir.replace(/[/\\]+$/, '');
  if (!dir) return '';
  // Combined: imageMetadata.json at same level as reports/
  return dir;
}

export function plotRelativePath(
  analysisDir: string,
  analyte: string,
  kind: 'calibration_curve' | 'heat_map'
): string {
  const base = analysisDir.replace(/[/\\]+$/, '');
  const suffix = kind === 'calibration_curve' ? 'calibration_curve.jpg' : 'heat_map.jpg';
  const rel = `${SCAN_ANALYSIS_PLOTS_DIR}/${analyte}_${suffix}`;
  return base ? `${base}/${rel}` : rel;
}

export function processReportRelativePath(analysisDir: string): string {
  const base = analysisDir.replace(/[/\\]+$/, '');
  return base ? `${base}/${SCAN_ANALYSIS_PROCESS_REPORT}` : SCAN_ANALYSIS_PROCESS_REPORT;
}

export function resultsRelativePath(analysisDir: string): string {
  const base = analysisDir.replace(/[/\\]+$/, '');
  return base ? `${base}/${SCAN_ANALYSIS_RESULTS_FILE}` : SCAN_ANALYSIS_RESULTS_FILE;
}

function mapHeader(raw: Record<string, unknown>): ScanAnalysisHeader {
  return {
    analysisDate: String(raw.analysis_date ?? ''),
    analysisId: String(raw.analysis_id ?? ''),
    scanResultsId: String(raw.scan_results_id ?? ''),
    scanDate: String(raw.scan_date ?? ''),
    productId: String(raw.product_id ?? ''),
  };
}

function mapExperiment(raw: Record<string, unknown>): ScanAnalysisExperiment {
  const initial = (raw.initial_concentrations as Record<string, number>) ?? {};
  return {
    analysisPlateMapId: String(raw.analysis_plate_map_id ?? ''),
    analysisName: String(raw.analysis_name ?? ''),
    analysisPlateMapName: String(raw.analysis_plate_map_name ?? ''),
    productName: String(raw.product_name ?? ''),
    plateBarcode: String(raw.plate_barcode ?? ''),
    scanName: String(raw.scan_name ?? ''),
    signalCalculationAlgorithm: String(raw.signal_calculation_algorithm ?? ''),
    initialConcentrations: initial,
    dilutionFactor: typeof raw.dilution_factor === 'number' ? raw.dilution_factor : undefined,
  };
}

function mapStandardRows(rows: unknown[]): ScanAnalysisStandardRow[] {
  return rows.map((r) => {
    const row = r as Record<string, unknown>;
    return {
      analyte: String(row.analyte ?? ''),
      wellLabel: String(row.well_label ?? ''),
      concentration: Number(row.concentration ?? 0),
      replicates: (row.replicates as Record<string, number>) ?? {},
      meanReplicateIntensity: row.mean_replicate_intensity as number | null | undefined,
      meanReplicateCalculatedConcentration: row.mean_replicate_calculated_concentration as number | null | undefined,
      percentBias: row.percent_bias as number | null | undefined,
      withinLimitsOfQuantificationV2: Boolean(row.within_limits_of_quantification_v2),
      upperPercentDifferenceV2: row.upper_percent_difference_v2 as number | null | undefined,
      lowerPercentDifferenceV2: row.lower_percent_difference_v2 as number | null | undefined,
    };
  });
}

function mapUnknownRows(rows: unknown[]): ScanAnalysisUnknownRow[] {
  return rows.map((r) => {
    const row = r as Record<string, unknown>;
    const reps = ((row.replicates as unknown[]) ?? []).map((rep) => {
      const r2 = rep as Record<string, unknown>;
      return {
        replicateIndex: Number(r2.replicate_index ?? 0),
        signal: Number(r2.signal ?? 0),
        concentration: r2.concentration as number | null | undefined,
      };
    });
    return {
      analyte: String(row.analyte ?? ''),
      wellLabel: String(row.well_label ?? ''),
      replicates: reps,
      meanReplicateConcentration: row.mean_replicate_concentration as number | null | undefined,
      stdevOfReplicateConcentration: row.stdev_of_replicate_concentration as number | null | undefined,
      withinLimitsOfQuantification: Boolean(row.within_limits_of_quantification),
      concentrationUnit: String(row.concentration_unit ?? ''),
    };
  });
}

function mapValidationRows(rows: unknown[]): ScanAnalysisValidationRow[] {
  return rows.map((r) => {
    const row = r as Record<string, unknown>;
    return {
      analyte: String(row.analyte ?? ''),
      signal: Number(row.signal ?? 0),
      wellRow: String(row.well_row ?? ''),
      wellColumn: Number(row.well_column ?? 0),
      wellReplicateIndex: row.well_replicate_index as number | undefined,
      wellType: String(row.well_type ?? ''),
      wellLabel: String(row.well_label ?? ''),
      calculatedConcentration: row.calculated_concentration as number | null | undefined,
    };
  });
}

function mapSpotIntensities(rows: unknown[]): ScanAnalysisSpotIntensity[] {
  return rows.map((r) => {
    const row = r as Record<string, unknown>;
    return {
      wellRow: String(row.well_row ?? ''),
      wellColumn: Number(row.well_column ?? 0),
      wellReplicateIndex: row.well_replicate_index as number | undefined,
      wellType: String(row.well_type ?? ''),
      wellLabel: String(row.well_label ?? ''),
      analyte: String(row.analyte ?? ''),
      row: Number(row.row ?? 0),
      column: Number(row.column ?? 0),
      signal: Number(row.signal ?? 0),
      background: Number(row.background ?? 0),
      signalIntensityAlgorithm: String(row.signal_intensity_algorithm ?? ''),
    };
  });
}

export function parseScanAnalysisResults(raw: string): ScanAnalysisData {
  const normalized = normalizeJSONNaN(raw);
  const parsed = JSON.parse(normalized) as Record<string, unknown>;

  const experiment = mapExperiment((parsed.experiment_data as Record<string, unknown>) ?? {});
  const standardRaw = (parsed.standard_report_data as Record<string, unknown[]>) ?? {};
  const unknownRaw = (parsed.unknown_report_data as Record<string, unknown[]>) ?? {};

  const standardReport: Record<string, ScanAnalysisStandardRow[]> = {};
  for (const [k, v] of Object.entries(standardRaw)) {
    standardReport[k] = mapStandardRows(v ?? []);
  }
  const unknownReport: Record<string, ScanAnalysisUnknownRow[]> = {};
  for (const [k, v] of Object.entries(unknownRaw)) {
    unknownReport[k] = mapUnknownRows(v ?? []);
  }

  const validation = mapValidationRows((parsed.validation_data as unknown[]) ?? []);
  const spotIntensities = mapSpotIntensities((parsed.spot_intensities as unknown[]) ?? []);
  const limitsOfQuant = (parsed.limits_of_quantification as Record<string, ScanAnalysisLOQ>) ?? {};
  const fitParameters = (parsed.fit_parameters as Record<string, ScanAnalysisFitParams>) ?? {};

  const analyteSet = new Set<string>(Object.keys(experiment.initialConcentrations));
  for (const row of validation) {
    if (row.analyte) analyteSet.add(row.analyte);
  }
  const analytes = Array.from(analyteSet).sort();

  const byWellAnalyte = new Map<string, ScanAnalysisValidationRow>();
  const byWell = new Map<string, ScanAnalysisValidationRow[]>();
  for (const row of validation) {
    const wellId = wellIdFromRowCol(row.wellRow, row.wellColumn);
    const key = wellAnalyteKey(wellId, row.analyte);
    byWellAnalyte.set(key, row);
    const list = byWell.get(wellId) ?? [];
    list.push(row);
    byWell.set(wellId, list);
  }

  const spotsByWellAnalyte = new Map<string, ScanAnalysisSpotIntensity[]>();
  for (const spot of spotIntensities) {
    const wellId = wellIdFromRowCol(spot.wellRow, spot.wellColumn);
    const key = wellAnalyteKey(wellId, spot.analyte);
    const list = spotsByWellAnalyte.get(key) ?? [];
    list.push(spot);
    spotsByWellAnalyte.set(key, list);
  }

  return {
    header: mapHeader((parsed.header_data as Record<string, unknown>) ?? {}),
    experiment,
    standardReport,
    unknownReport,
    validation,
    spotIntensities,
    limitsOfQuant,
    fitParameters,
    analytes,
    byWellAnalyte,
    byWell,
    spotsByWellAnalyte,
  };
}

export function validationAt(
  data: ScanAnalysisData,
  wellId: string,
  analyte: string
): ScanAnalysisValidationRow | undefined {
  return data.byWellAnalyte.get(wellAnalyteKey(wellId, analyte));
}

export function concentrationAt(
  data: ScanAnalysisData,
  wellId: string,
  analyte: string
): number | null {
  const row = validationAt(data, wellId, analyte);
  if (!row || row.calculatedConcentration == null || Number.isNaN(row.calculatedConcentration)) {
    return null;
  }
  return row.calculatedConcentration;
}

export function runLabelFromAnalysisDir(dirPath: string): string {
  const base = dirPath.split(/[/\\]/).pop() ?? dirPath;
  return base.replace(/-summary$/i, '').trim() || base;
}

/** Plate cell value for grid coloring based on mode. */
export function plateCellValue(
  data: ScanAnalysisData,
  wellId: string,
  analyte: string,
  mode: PlateGridMode
): string | number | boolean | null {
  const row = validationAt(data, wellId, analyte);
  if (!row) return null;
  switch (mode) {
    case 'concentration':
      return row.calculatedConcentration ?? null;
    case 'intensity':
      return row.signal;
    case 'loq': {
      const unknownRows = data.unknownReport[analyte] ?? [];
      const match = unknownRows.find((u) => u.wellLabel === row.wellLabel);
      if (match) return match.withinLimitsOfQuantification;
      const stdRows = data.standardReport[analyte] ?? [];
      const stdMatch = stdRows.find((s) => s.wellLabel === row.wellLabel);
      if (stdMatch) return stdMatch.withinLimitsOfQuantificationV2;
      return null;
    }
    case 'wellType':
      return row.wellType;
    default:
      return null;
  }
}

export function allWellAnalyteConcentrations(
  data: ScanAnalysisData,
  wellId: string
): { analyte: string; concentration: number | null; withinLoq: boolean | null }[] {
  const rows = data.byWell.get(wellId) ?? [];
  return rows.map((row) => {
    const unknownRows = data.unknownReport[row.analyte] ?? [];
    const unk = unknownRows.find((u) => u.wellLabel === row.wellLabel);
    let withinLoq: boolean | null = null;
    if (unk) {
      withinLoq = unk.withinLimitsOfQuantification;
    } else {
      const stdRows = data.standardReport[row.analyte] ?? [];
      const std = stdRows.find((s) => s.wellLabel === row.wellLabel);
      if (std) withinLoq = std.withinLimitsOfQuantificationV2;
    }
    return {
      analyte: row.analyte,
      concentration: row.calculatedConcentration ?? null,
      withinLoq,
    };
  });
}
