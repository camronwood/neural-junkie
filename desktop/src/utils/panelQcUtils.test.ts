import { describe, expect, it } from 'vitest';
import type { PanelQCReport } from '../utils/secondaryAnalysis';
import {
  formatBiologyExpertQcPrompt,
  panelQcExportRelativePaths,
  panelQcToCsv,
  qcFailureWellsForAnalyte,
  qcReportRelativePath,
} from '../utils/panelQcUtils';

const sampleReport: PanelQCReport = {
  plate_label: 'test-plate',
  overall_pass: false,
  analytes: [
    {
      analyte: 'IL-6',
      pass: false,
      checks: [
        {
          name: 'ColumnDeviation',
          pass: false,
          detail: 'Column 3 deviates from plate mean by 30.00%; Column 5 deviates from plate mean by 28.00%',
        },
        {
          name: 'RowDeviation',
          pass: false,
          detail: 'Row E deviates from plate mean by 26.00%',
        },
        { name: 'LLOQ', pass: true, value: '0.04' },
      ],
    },
  ],
};

describe('panelQcUtils', () => {
  it('panelQcExportRelativePaths places files under reports', () => {
    const paths = panelQcExportRelativePaths('run-summary', 'My Plate #1');
    expect(paths.json).toBe('run-summary/reports/My_Plate_1_qc_12plex.json');
    expect(paths.csv).toBe('run-summary/reports/My_Plate_1_qc_12plex.csv');
  });

  it('qcReportRelativePath handles workspace root', () => {
    expect(qcReportRelativePath('')).toBe('reports/qc_12plex_report.json');
    expect(qcReportRelativePath('run-summary')).toBe('run-summary/reports/qc_12plex_report.json');
  });

  it('qcFailureWellsForAnalyte maps column and row failures', () => {
    const wells = qcFailureWellsForAnalyte(sampleReport, 'IL-6');
    expect(wells.has('A3')).toBe(true);
    expect(wells.has('H5')).toBe(true);
    expect(wells.has('E1')).toBe(true);
    expect(wells.has('E12')).toBe(true);
    expect(wells.has('A1')).toBe(false);
  });

  it('panelQcToCsv includes check rows', () => {
    const csv = panelQcToCsv(sampleReport);
    expect(csv).toContain('IL-6,ColumnDeviation,false');
    expect(csv.split('\n').length).toBeGreaterThan(2);
  });

  it('formatBiologyExpertQcPrompt mentions BiologyExpert and overall result', () => {
    const text = formatBiologyExpertQcPrompt(sampleReport, 'run-summary');
    expect(text).toContain('@BiologyExpert');
    expect(text).toContain('FAIL');
    expect(text).toContain('IL-6');
  });
});
