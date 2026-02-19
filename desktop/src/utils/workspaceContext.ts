import { FileNode } from '../stores/fileExplorerStore';

export interface WorkspaceContext {
  workspace_name: string;
  workspace_path: string;
  file_tree: string;
  open_files: OpenFileContext[];
}

export interface OpenFileContext {
  path: string;
  language: string;
  content: string;
  is_active: boolean;
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
