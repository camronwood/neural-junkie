import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { describe, expect, it } from 'vitest';
import {
  concentrationAt,
  isScanAnalysisResultsPath,
  normalizeJSONNaN,
  parseScanAnalysisResults,
  parseWellId,
  scanAnalysisDirFromResultsPath,
  wellIdFromRowCol,
  wellAnalyteKey,
} from './scanAnalysis';

const fixturePath = join(
  __dirname,
  '..',
  '..',
  '..',
  'testdata',
  'scan-analysis',
  'reports',
  'results.json'
);

describe('scanAnalysis', () => {
  it('detects results path', () => {
    expect(isScanAnalysisResultsPath('run/reports/results.json')).toBe(true);
    expect(isScanAnalysisResultsPath('imageMetadata.json')).toBe(false);
  });

  it('normalizes NaN', () => {
    expect(normalizeJSONNaN('{"x": NaN}')).not.toContain('NaN');
  });

  it('parses fixture and indexes by well', () => {
    const raw = readFileSync(fixturePath, 'utf-8');
    const data = parseScanAnalysisResults(raw);
    expect(data.experiment.productName).toContain('Inflammatory');
    expect(data.analytes).toContain('IL-6');
    expect(concentrationAt(data, 'A1', 'IL-6')).not.toBeNull();
    expect(scanAnalysisDirFromResultsPath('run/reports/results.json')).toBe('run');
  });

  it('well helpers', () => {
    expect(wellIdFromRowCol('A', 1)).toBe('A1');
    expect(wellAnalyteKey('A1', 'IL-6')).toBe('A1|IL-6');
    expect(parseWellId('H12')).toEqual({ row: 'H', col: 12 });
  });
});
