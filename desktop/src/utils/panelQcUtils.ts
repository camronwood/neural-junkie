import type { PanelQCReport } from './secondaryAnalysis';
import { PLATE_COLS, PLATE_ROWS } from './scanAnalysis';

const QC_REPORT_FILE = 'reports/qc_12plex_report.json';

export function qcReportRelativePath(analysisDir: string): string {
  const base = analysisDir.replace(/[/\\]+$/, '');
  return base ? `${base}/${QC_REPORT_FILE}` : QC_REPORT_FILE;
}

/** Wells to highlight when viewing QC failures for one analyte. */
export function qcFailureWellsForAnalyte(
  report: PanelQCReport,
  analyte: string
): Set<string> {
  const wells = new Set<string>();
  const row = report.analytes.find((a) => a.analyte === analyte);
  if (!row) return wells;

  for (const check of row.checks) {
    if (check.pass) continue;
    switch (check.name) {
      case 'IntraplateCV':
        for (const r of PLATE_ROWS) {
          if (r === 'A' || r === 'B') continue;
          for (const c of PLATE_COLS) {
            if ((r === 'C' || r === 'D') && ['1', '2', '3', '4'].includes(String(c))) continue;
            wells.add(`${r}${c}`);
          }
        }
        break;
      case 'ColumnDeviation':
        for (const m of check.detail?.matchAll(/Column (\d+)/g) ?? []) {
          const col = m[1];
          for (const r of PLATE_ROWS) wells.add(`${r}${col}`);
        }
        break;
      case 'RowDeviation':
        for (const m of check.detail?.matchAll(/Row ([A-H])/g) ?? []) {
          const rowLetter = m[1];
          for (const c of PLATE_COLS) wells.add(`${rowLetter}${c}`);
        }
        break;
      case 'SpikeRecovery':
        for (const r of ['C', 'D']) {
          for (const c of ['1', '2', '3', '4']) wells.add(`${r}${c}`);
        }
        break;
      default:
        break;
    }
  }
  return wells;
}

/** Workspace-relative paths for QC export files under reports/. */
export function panelQcExportRelativePaths(
  analysisDir: string,
  runLabel: string
): { json: string; csv: string } {
  const slug = (runLabel.replace(/[^\w.-]+/g, '_') || 'plate').slice(0, 80);
  const dir = analysisDir.replace(/[/\\]+$/, '');
  const reports = dir ? `${dir}/reports` : 'reports';
  return {
    json: `${reports}/${slug}_qc_12plex.json`,
    csv: `${reports}/${slug}_qc_12plex.csv`,
  };
}

export function panelQcJsonContent(report: PanelQCReport): string {
  return JSON.stringify(report, null, 2);
}

/** Browser fallback; anchor downloads are blocked in the Tauri webview — prefer hub saveFileContent. */
export function downloadPanelQcJson(report: PanelQCReport, filename: string): void {
  downloadTextFile(panelQcJsonContent(report), filename, 'application/json');
}

export function panelQcToCsv(report: PanelQCReport): string {
  const lines = ['analyte,check,pass,value,detail'];
  for (const a of report.analytes) {
    for (const c of a.checks) {
      lines.push(
        [
          a.analyte,
          c.name,
          c.pass ? 'true' : 'false',
          csvEscape(c.value ?? ''),
          csvEscape(c.detail ?? ''),
        ].join(',')
      );
    }
  }
  return lines.join('\n');
}

function csvEscape(value: string): string {
  if (/[",\n]/.test(value)) return `"${value.replace(/"/g, '""')}"`;
  return value;
}

export function downloadPanelQcCsv(report: PanelQCReport, filename: string): void {
  downloadTextFile(panelQcToCsv(report), filename, 'text/csv');
}

function downloadTextFile(content: string, filename: string, mime: string): void {
  const blob = new Blob([content], { type: mime });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  a.style.display = 'none';
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  window.setTimeout(() => URL.revokeObjectURL(url), 0);
}

export function formatBiologyExpertQcPrompt(report: PanelQCReport, analysisDir: string): string {
  const failed = report.analytes.filter((a) => !a.pass);
  const summary = [
    `@BiologyExpert Please interpret this 12-Plex SOP QC report for plate "${report.plate_label}".`,
    `Analysis folder: ${analysisDir || '(workspace root)'}`,
    `Overall: ${report.overall_pass ? 'PASS' : 'FAIL'}`,
    failed.length
      ? `Failed analytes (${failed.length}): ${failed.map((a) => a.analyte).join(', ')}`
      : 'All analytes passed.',
    '',
    'QC was already run in the viewer. Interpret the summary above.',
    'If you need to re-check files, call the hub MCP tool run_12plex_qc (not a shell command).',
  ];
  if (report.messages?.length) {
    summary.push('', 'Messages:', ...report.messages.slice(0, 8).map((m) => `- ${m}`));
  }
  return summary.join('\n');
}
