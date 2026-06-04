/** 12-Plex QC types returned by POST /api/secondary-analysis/12plex-qc */

export interface QCCheckResult {
  name: string;
  pass: boolean;
  value?: string;
  detail?: string;
}

export interface AnalyteQCRow {
  analyte: string;
  pass: boolean;
  checks: QCCheckResult[];
}

export interface PanelQCReport {
  plate_label: string;
  product_name?: string;
  overall_pass: boolean;
  messages?: string[];
  analytes: AnalyteQCRow[];
}

export type SecondaryAnalysisWorkflow =
  | 'comparator'
  | 'endogenous'
  | 'std_curves'
  | 'print_order'
  | '12plex_qc_excel'
  | 'spc_charts';

export type SecondaryAnalysisJobStatus =
  | 'queued'
  | 'running'
  | 'done'
  | 'failed'
  | 'cancelled';

export interface SecondaryAnalysisJob {
  id: string;
  workflow: SecondaryAnalysisWorkflow;
  status: SecondaryAnalysisJobStatus;
  workspace_id?: string;
  output_dir?: string;
  log_tail?: string[];
  error?: string;
  config?: Record<string, unknown>;
  created_at?: string;
  updated_at?: string;
}

export interface ComparatorSummary {
  analysis_dir: string;
  conditions?: string[];
  source_plates?: string[];
  lloq_uloq_path?: string;
  lloq_uloq_rows?: string[][];
  plate_stats?: Record<string, string[][]>;
  interplate_stats?: Record<string, string[][]>;
  artifacts?: ComparatorArtifactGroup[];
}

export interface ComparatorArtifactGroup {
  condition: string;
  plate: string;
  files: { relative_path: string; name: string; kind: string }[];
}

export function isComparatorAnalysisDirName(name: string): boolean {
  return name.startsWith('Comparator Analysis');
}

export function isComparatorAnalysisPath(relPath: string): boolean {
  const norm = relPath.replace(/\\/g, '/').replace(/\/+$/, '');
  const base = norm.split('/').pop() ?? norm;
  return isComparatorAnalysisDirName(base);
}
