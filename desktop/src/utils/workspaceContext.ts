import { FileNode } from '../stores/fileExplorerStore';

export interface WorkspaceContext {
  workspace_name: string;
  workspace_path: string;
  file_tree: string;
  open_files: OpenFileContext[];
  scan_summary?: ScanSummaryContext;
  scan_analysis?: ScanAnalysisContext;
}

export interface OpenFileContext {
  path: string;
  language: string;
  content: string;
  is_active: boolean;
  view_mode?: string;
  scan_summary_dir?: string;
  scan_analysis_dir?: string;
  selection_start_line?: number;
  selection_end_line?: number;
  selected_text?: string;
}

export interface ScanSummarySpotContext {
  analyte: string;
  row: string;
  column: string;
  x_px: number;
  y_px: number;
}

export interface ScanSummaryWellContext {
  well: string;
  time?: string;
  fov_size_x_um?: number;
  fov_size_y_um?: number;
  z_stage_position_um?: number;
  spot_count: number;
  spots: ScanSummarySpotContext[];
}

export interface ScanSummaryContext {
  summary_dir: string;
  wells_count: number;
  analytes: string[];
  active_well?: ScanSummaryWellContext;
  note: string;
}

export interface ScanAnalysisWellContext {
  well: string;
  concentration?: number | null;
  within_loq?: boolean | null;
  well_type?: string;
  well_label?: string;
}

export interface ScanAnalysisContext {
  analysis_dir: string;
  product_name?: string;
  plate_barcode?: string;
  analytes: string[];
  dilution_factor?: number;
  active_analyte?: string;
  active_well?: ScanAnalysisWellContext;
  linked_scan_dir?: string;
  note: string;
}

/**
 * Builds a human-readable indented file tree string from FileNode[].
 * Depth-limited to keep the payload manageable.
 */
export function buildFileTreeString(nodes: FileNode[], maxDepth: number = 3): string {
  const lines: string[] = [];

  function walk(node: FileNode, depth: number, prefix: string) {
    if (depth > maxDepth) return;
    const icon = node.is_dir ? '\u{1F4C1} ' : '  ';
    lines.push(`${prefix}${icon}${node.name}`);
    if (node.is_dir && node.children) {
      for (const child of node.children) {
        walk(child, depth + 1, prefix + '  ');
      }
    }
  }

  for (const node of nodes) {
    walk(node, 0, '');
  }

  return lines.join('\n');
}
