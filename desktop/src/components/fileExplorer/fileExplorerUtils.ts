import type { FileNode } from '../../stores/fileExplorerStore';
import {
  isScanAnalysisDirListing,
  isScanAnalysisResultsPath,
  isScanAnalysisSummaryCSVPath,
} from '../../utils/scanAnalysis';
import {
  isScanSummaryDirListing,
  isScanSummaryMetadataPath,
  isScanSummaryWellPath,
} from '../../utils/scanSummary';

export function getLanguageFromPath(path: string): string {
  if (!path) {
    return 'plaintext';
  }

  const ext = path.split('.').pop()?.toLowerCase();
  const languageMap: Record<string, string> = {
    js: 'javascript',
    jsx: 'javascript',
    ts: 'typescript',
    tsx: 'typescript',
    py: 'python',
    go: 'go',
    rs: 'rust',
    java: 'java',
    cpp: 'cpp',
    c: 'c',
    cs: 'csharp',
    php: 'php',
    rb: 'ruby',
    swift: 'swift',
    kt: 'kotlin',
    scala: 'scala',
    html: 'html',
    css: 'css',
    scss: 'scss',
    sass: 'sass',
    less: 'less',
    json: 'json',
    xml: 'xml',
    yaml: 'yaml',
    yml: 'yaml',
    md: 'markdown',
    sql: 'sql',
    sh: 'shell',
    bash: 'shell',
    zsh: 'shell',
    fish: 'shell',
  };
  return languageMap[ext || ''] || 'plaintext';
}

export function findFileNode(nodes: FileNode[], path: string): FileNode | undefined {
  for (const node of nodes) {
    if (node.path === path) return node;
    if (node.children?.length) {
      const found = findFileNode(node.children, path);
      if (found) return found;
    }
  }
  return undefined;
}

export function isScanAnalysisFolder(file: FileNode): boolean {
  if (!file.is_dir) return false;
  if (file.children?.length && isScanAnalysisDirListing(file.children)) {
    return true;
  }
  return /-summary$/i.test(file.name) || /-summary$/i.test(file.path);
}

export function isScanSummaryFolder(file: FileNode): boolean {
  if (!file.is_dir) return false;
  if (file.children?.length && isScanSummaryDirListing(file.children)) {
    return true;
  }
  return /-summary$/i.test(file.name) || /-summary$/i.test(file.path);
}

export function fileExplorerIcon(
  file: FileNode,
  expandedPaths: Record<string, boolean>
): string {
  if (file.is_dir) {
    if (isScanAnalysisFolder(file)) {
      return expandedPaths[file.path] ? '📊' : '📊';
    }
    if (isScanSummaryFolder(file)) {
      return expandedPaths[file.path] ? '🔬' : '🔬';
    }
    return expandedPaths[file.path] ? '📂' : '📁';
  }
  if (isScanAnalysisResultsPath(file.path) || isScanAnalysisSummaryCSVPath(file.path)) {
    return '📊';
  }
  if (isScanSummaryMetadataPath(file.path) || isScanSummaryWellPath(file.path)) {
    return '🔬';
  }

  if (!file.path) {
    return '📄';
  }

  const ext = file.path.split('.').pop()?.toLowerCase();
  const iconMap: Record<string, string> = {
    js: '📄',
    jsx: '⚛️',
    ts: '📘',
    tsx: '⚛️',
    py: '🐍',
    go: '🐹',
    rs: '🦀',
    java: '☕',
    html: '🌐',
    css: '🎨',
    json: '📋',
    md: '📝',
    txt: '📄',
    yml: '⚙️',
    yaml: '⚙️',
  };
  return iconMap[ext || ''] || '📄';
}
