import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { describe, expect, it } from 'vitest';
import {
  analyteFromSummaryCsvPath,
  mergeScanAnalysisData,
  parseScanAnalysisCsv,
} from './scanAnalysisCsv';
import {
  normalizeScanLinkInput,
  scanLinkCandidateDirs,
  scanMetadataRelativePath,
} from './scanAnalysisLink';
import { concentrationAt } from './scanAnalysis';

const fixtureCsv = join(
  __dirname,
  '..',
  '..',
  '..',
  'testdata',
  'scan-analysis',
  'reports',
  'IL-6_summary_report.csv'
);

describe('scanAnalysisLink', () => {
  it('normalizes metadata path pasted as link', () => {
    expect(normalizeScanLinkInput('scan-export/imageMetadata.json')).toBe('scan-export');
    expect(normalizeScanLinkInput('scan-export/')).toBe('scan-export');
    expect(normalizeScanLinkInput('scan-export ')).toBe('scan-export');
  });

  it('builds metadata relative path', () => {
    expect(scanMetadataRelativePath('scan-export')).toBe('scan-export/imageMetadata.json');
    expect(scanMetadataRelativePath('')).toBe('imageMetadata.json');
  });

  it('includes scan-export in candidates', () => {
    const cands = scanLinkCandidateDirs('');
    expect(cands).toContain('scan-export');
  });
});

describe('scanAnalysisCsv', () => {
  it('extracts analyte from filename', () => {
    expect(analyteFromSummaryCsvPath('reports/IL-6_summary_report.csv')).toBe('IL-6');
  });

  it('parses IL-6 fixture CSV', () => {
    const raw = readFileSync(fixtureCsv, 'utf-8');
    const data = parseScanAnalysisCsv(raw, 'IL-6');
    expect(data.analytes).toContain('IL-6');
    expect(concentrationAt(data, 'A1', 'IL-6')).not.toBeNull();
    expect(data.standardReport['IL-6']?.length).toBeGreaterThan(0);
    expect(data.unknownReport['IL-6']?.length).toBeGreaterThan(0);
    expect(data.limitsOfQuant['IL-6']?.LLOQ).toBeTruthy();
  });

  it('mergeScanAnalysisData returns single doc unchanged', () => {
    const raw = readFileSync(fixtureCsv, 'utf-8');
    const a = parseScanAnalysisCsv(raw, 'IL-6');
    expect(mergeScanAnalysisData([a])).toBe(a);
  });
});
