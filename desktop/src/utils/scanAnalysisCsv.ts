import type {
  ScanAnalysisData,
  ScanAnalysisLOQ,
  ScanAnalysisStandardRow,
  ScanAnalysisUnknownRow,
  ScanAnalysisValidationRow,
} from './scanAnalysis';
import {
  parseWellId,
  wellAnalyteKey,
  wellIdFromRowCol,
} from './scanAnalysis';

export function analyteFromSummaryCsvPath(path: string): string | null {
  const base = path.split(/[/\\]/).pop() ?? path;
  const m = base.match(/^(.+)_summary_report\.csv$/i);
  return m ? m[1] : null;
}

export function scanAnalysisDirFromCsvPath(csvPath: string): string {
  const normalized = csvPath.replace(/\\/g, '/');
  const idx = normalized.indexOf('/reports/');
  if (idx >= 0) return normalized.slice(0, idx);
  if (normalized.startsWith('reports/')) return '';
  const parent = normalized.replace(/[/\\][^/\\]+$/, '');
  if (parent.endsWith('/reports') || parent === 'reports') {
    return parent.replace(/\/?reports$/, '');
  }
  return parent;
}

function parseCsvLine(line: string): string[] {
  const out: string[] = [];
  let cur = '';
  let inQuotes = false;
  for (let i = 0; i < line.length; i++) {
    const ch = line[i];
    if (ch === '"') {
      inQuotes = !inQuotes;
      continue;
    }
    if (ch === ',' && !inQuotes) {
      out.push(cur.trim());
      cur = '';
      continue;
    }
    cur += ch;
  }
  out.push(cur.trim());
  return out;
}

function parseNum(raw: string): number | null {
  const s = raw.trim().toLowerCase();
  if (!s || s === 'nan' || s === 'null') return null;
  const n = Number(s);
  return Number.isFinite(n) ? n : null;
}

function parseBool(raw: string): boolean {
  return raw.trim().toLowerCase() === 'true';
}

function wellTypeFromLabel(label: string): string {
  const l = label.toLowerCase();
  if (l.startsWith('stnd') || l.startsWith('std')) return 'standard';
  if (l === 'blank') return 'blank';
  return 'unknown';
}

type PlateGrid = Map<string, Map<number, string>>;

function normalizeWellLabel(label: string): string {
  const m = label.trim().match(/^(stnd|std|unk)(\d+)$/i);
  if (m) {
    const prefix = m[1].toLowerCase().startsWith('st') ? 'stnd' : 'unk';
    return `${prefix}${m[2].padStart(2, '0')}`;
  }
  return label.trim().toLowerCase();
}

function parsePlateGrid(lines: string[], startIdx: number): { grid: PlateGrid; nextIdx: number } {
  const grid: PlateGrid = new Map();
  let i = startIdx;
  if (i >= lines.length) return { grid, nextIdx: i };
  i += 1; // skip section title row
  while (i < lines.length) {
    const line = lines[i].trim();
    if (!line) {
      i += 1;
      continue;
    }
    if (/^[A-Za-z].*Report$/i.test(line) || line.startsWith('Plate Map') || line.startsWith('Limits of') || line.startsWith('Limit of') || line.startsWith('NOTE:')) {
      break;
    }
    const cols = parseCsvLine(line);
    const row = cols[0]?.toUpperCase();
    if (!row || row.length !== 1 || row < 'A' || row > 'H') {
      i += 1;
      continue;
    }
    const byCol = new Map<number, string>();
    for (let c = 1; c < cols.length && c <= 12; c++) {
      byCol.set(c, cols[c] ?? '');
    }
    grid.set(row, byCol);
    i += 1;
  }
  return { grid, nextIdx: i };
}

function buildIndexes(data: Omit<ScanAnalysisData, 'byWellAnalyte' | 'byWell' | 'spotsByWellAnalyte' | 'analytes'> & { analytes: string[] }): ScanAnalysisData {
  const byWellAnalyte = new Map<string, ScanAnalysisValidationRow>();
  const byWell = new Map<string, ScanAnalysisValidationRow[]>();
  for (const row of data.validation) {
    const wellId = wellIdFromRowCol(row.wellRow, row.wellColumn);
    const key = wellAnalyteKey(wellId, row.analyte);
    byWellAnalyte.set(key, row);
    const list = byWell.get(wellId) ?? [];
    list.push(row);
    byWell.set(wellId, list);
  }
  return {
    ...data,
    byWellAnalyte,
    byWell,
    spotsByWellAnalyte: new Map(),
  };
}

/** Parse one Phoenix *_summary_report.csv into ScanAnalysisData (single analyte). */
export function parseScanAnalysisCsv(raw: string, analyte: string): ScanAnalysisData {
  const lines = raw.replace(/\r\n/g, '\n').split('\n');
  const standardReport: Record<string, ScanAnalysisStandardRow[]> = { [analyte]: [] };
  const unknownReport: Record<string, ScanAnalysisUnknownRow[]> = { [analyte]: [] };
  const limitsOfQuant: Record<string, ScanAnalysisLOQ> = {};
  const validation: ScanAnalysisValidationRow[] = [];

  let plateConcentrations: PlateGrid = new Map();
  let plateLabels: PlateGrid = new Map();
  let plateIntensities: PlateGrid = new Map();

  let i = 0;
  while (i < lines.length) {
    const line = lines[i].trim();
    if (line === 'Standard Report') {
      i += 2;
      while (i < lines.length) {
        const row = lines[i].trim();
        if (!row || row.startsWith('NOTE:') || row === 'Unknown Report') break;
        const cols = parseCsvLine(row);
        if (cols.length < 2) break;
        standardReport[analyte].push({
          analyte,
          wellLabel: cols[0],
          concentration: parseNum(cols[1]) ?? 0,
          replicates: {},
          meanReplicateIntensity: parseNum(cols[4] ?? ''),
          meanReplicateCalculatedConcentration: parseNum(cols[5] ?? ''),
          percentBias: parseNum(cols[6] ?? ''),
          withinLimitsOfQuantificationV2: parseBool(cols[7] ?? 'false'),
          upperPercentDifferenceV2: parseNum(cols[8] ?? ''),
          lowerPercentDifferenceV2: parseNum(cols[9] ?? ''),
        });
        i += 1;
      }
      continue;
    }
    if (line === 'Unknown Report') {
      i += 2;
      while (i < lines.length) {
        const row = lines[i].trim();
        if (!row || row.startsWith('Plate Map')) break;
        const cols = parseCsvLine(row);
        if (cols.length < 2) break;
        unknownReport[analyte].push({
          analyte,
          wellLabel: cols[0],
          replicates: [{ replicateIndex: 0, signal: parseNum(cols[1]) ?? 0, concentration: parseNum(cols[2] ?? '') }],
          meanReplicateConcentration: parseNum(cols[3] ?? ''),
          stdevOfReplicateConcentration: parseNum(cols[4] ?? ''),
          withinLimitsOfQuantification: parseBool(cols[5] ?? 'false'),
          concentrationUnit: 'pg/ml',
        });
        i += 1;
      }
      continue;
    }
    if (line === 'Plate Map Concentrations') {
      const parsed = parsePlateGrid(lines, i);
      plateConcentrations = parsed.grid;
      i = parsed.nextIdx;
      continue;
    }
    if (line === 'Plate Map Labels') {
      const parsed = parsePlateGrid(lines, i);
      plateLabels = parsed.grid;
      i = parsed.nextIdx;
      continue;
    }
    if (line === 'Plate Map Intensities') {
      const parsed = parsePlateGrid(lines, i);
      plateIntensities = parsed.grid;
      i = parsed.nextIdx;
      continue;
    }
    if (line.startsWith('Limits of Quantification')) {
      i += 2;
      if (i < lines.length) {
        const cols = parseCsvLine(lines[i]);
        limitsOfQuant[analyte] = {
          LLOQ: cols[0],
          ULOQ: cols[1],
          concentration_units: 'pg/ml',
        };
      }
      i += 1;
      continue;
    }
    if (line.startsWith('Limit of Detection')) {
      i += 2;
      if (i < lines.length && limitsOfQuant[analyte]) {
        limitsOfQuant[analyte].LOD = parseCsvLine(lines[i])[0];
        limitsOfQuant[analyte].LOD_label = 'Calculated LOD';
      }
      i += 1;
      continue;
    }
    i += 1;
  }

  for (const [row, cols] of plateLabels) {
    for (const [col, label] of cols) {
      if (!label) continue;
      const wellId = wellIdFromRowCol(row, col);
      if (!parseWellId(wellId)) continue;
      const conc = parseNum(plateConcentrations.get(row)?.get(col) ?? '');
      const intensity = parseNum(plateIntensities.get(row)?.get(col) ?? '');
      const unk = unknownReport[analyte].find(
        (u) => normalizeWellLabel(u.wellLabel) === normalizeWellLabel(label)
      );
      validation.push({
        analyte,
        signal: intensity ?? 0,
        wellRow: row,
        wellColumn: col,
        wellType: wellTypeFromLabel(label),
        wellLabel: label,
        calculatedConcentration: conc ?? unk?.meanReplicateConcentration ?? null,
      });
    }
  }

  return buildIndexes({
    header: {},
    experiment: {
      productName: '',
      initialConcentrations: { [analyte]: 0 },
    },
    standardReport,
    unknownReport,
    validation,
    spotIntensities: [],
    limitsOfQuant,
    fitParameters: {},
    analytes: [analyte],
  });
}

/** Merge multiple CSV-derived documents (one per analyte) into one viewer document. */
export function mergeScanAnalysisData(docs: ScanAnalysisData[]): ScanAnalysisData {
  if (docs.length === 0) {
    throw new Error('no analysis documents to merge');
  }
  if (docs.length === 1) return docs[0];

  const merged: ScanAnalysisData = {
    header: docs[0].header,
    experiment: { ...docs[0].experiment, initialConcentrations: {} },
    standardReport: {},
    unknownReport: {},
    validation: [],
    spotIntensities: [],
    limitsOfQuant: {},
    fitParameters: {},
    analytes: [],
    byWellAnalyte: new Map(),
    byWell: new Map(),
    spotsByWellAnalyte: new Map(),
  };

  for (const doc of docs) {
    for (const [k, v] of Object.entries(doc.experiment.initialConcentrations)) {
      merged.experiment.initialConcentrations[k] = v;
    }
    Object.assign(merged.standardReport, doc.standardReport);
    Object.assign(merged.unknownReport, doc.unknownReport);
    Object.assign(merged.limitsOfQuant, doc.limitsOfQuant);
    Object.assign(merged.fitParameters, doc.fitParameters);
    merged.validation.push(...doc.validation);
    merged.spotIntensities.push(...doc.spotIntensities);
    if (doc.experiment.productName && !merged.experiment.productName) {
      merged.experiment.productName = doc.experiment.productName;
    }
    if (doc.experiment.dilutionFactor && !merged.experiment.dilutionFactor) {
      merged.experiment.dilutionFactor = doc.experiment.dilutionFactor;
    }
  }

  merged.analytes = Array.from(
    new Set([
      ...Object.keys(merged.experiment.initialConcentrations),
      ...merged.validation.map((v) => v.analyte),
    ])
  ).sort();

  for (const row of merged.validation) {
    const wellId = wellIdFromRowCol(row.wellRow, row.wellColumn);
    const key = wellAnalyteKey(wellId, row.analyte);
    merged.byWellAnalyte.set(key, row);
    const list = merged.byWell.get(wellId) ?? [];
    list.push(row);
    merged.byWell.set(wellId, list);
  }

  return merged;
}
