import type { Workspace } from '../stores/fileExplorerStore';
import type { EditorTab } from '../stores/editorStore';
import type { LinkedWorkspaceContext } from '../constants/promptMetadata';
import { normalizeWorkspacePath } from './repoAgentWorkspace';

export const MAX_SCOPED_WORKSPACES = 4;
export const MAX_LINKED_WORKSPACES = 3;

export interface WorkspaceScope {
  primary: Workspace | null;
  linked: LinkedWorkspaceContext[];
}

/**
 * Resolves multi-repo scope: active workspace is primary; open editor tabs add linked workspaces.
 */
export function resolveWorkspaceScope(input: {
  workspaces: Workspace[];
  activeWorkspaceId: string | null;
  editorTabs: EditorTab[];
  activeTabId: string | null;
  /** Optional project-set member workspace IDs (phase 2). */
  projectSetMemberIds?: string[];
}): WorkspaceScope {
  const { workspaces, activeWorkspaceId, editorTabs, activeTabId, projectSetMemberIds } = input;
  const primary =
    workspaces.find((w) => w.id === activeWorkspaceId) ?? workspaces[0] ?? null;
  const primaryNorm = primary ? normalizeWorkspacePath(primary.path) : '';

  const seen = new Set<string>();
  if (primaryNorm) seen.add(primaryNorm);

  const linked: LinkedWorkspaceContext[] = [];

  const addLinked = (ws: Workspace, source: LinkedWorkspaceContext['source'], tabs: EditorTab[]) => {
    const norm = normalizeWorkspacePath(ws.path);
    if (!norm || seen.has(norm) || linked.length >= MAX_LINKED_WORKSPACES) return;
    seen.add(norm);
    linked.push({
      workspace_id: ws.id,
      workspace_path: ws.path,
      workspace_name: ws.name,
      source,
      open_files: tabs
        .filter((tab) => tab.workspaceId === ws.id)
        .map((tab) => ({
          path: tab.path,
          language: tab.language ?? 'text',
          content: tab.content.substring(0, 10000),
          is_active: tab.id === activeTabId,
          view_mode: tab.viewMode,
          scan_summary_dir: tab.scanSummaryDir,
          scan_analysis_dir: tab.scanAnalysisDir,
        })),
    });
  };

  const tabWorkspaceIds = [...new Set(editorTabs.map((t) => t.workspaceId).filter(Boolean))];
  for (const wsId of tabWorkspaceIds) {
    if (primary && wsId === primary.id) continue;
    const ws = workspaces.find((w) => w.id === wsId);
    if (!ws) continue;
    const tabs = editorTabs.filter((t) => t.workspaceId === wsId);
    addLinked(ws, 'open_tab', tabs);
  }

  if (projectSetMemberIds?.length) {
    for (const wsId of projectSetMemberIds) {
      if (primary && wsId === primary.id) continue;
      const ws = workspaces.find((w) => w.id === wsId);
      if (!ws) continue;
      const tabs = editorTabs.filter((t) => t.workspaceId === wsId);
      addLinked(ws, 'project_set', tabs);
    }
  }

  return { primary, linked };
}

export function scopeSummaryLabel(scope: WorkspaceScope): string | null {
  const count = (scope.primary ? 1 : 0) + scope.linked.length;
  if (count <= 1) return null;
  const primaryName = scope.primary?.name ?? 'workspace';
  const extra = count - 1;
  return `${primaryName} + ${extra} repo${extra === 1 ? '' : 's'}`;
}

export function scopedRepoPaths(scope: WorkspaceScope): string[] {
  const out: string[] = [];
  if (scope.primary?.path?.trim()) out.push(scope.primary.path.trim());
  for (const lw of scope.linked) {
    if (lw.workspace_path?.trim()) out.push(lw.workspace_path.trim());
  }
  return out.slice(0, MAX_SCOPED_WORKSPACES);
}
