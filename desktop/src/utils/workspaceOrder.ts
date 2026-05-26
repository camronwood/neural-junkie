import type { Workspace } from '../stores/fileExplorerStore';

export const TAB_BAR_MAX = 5;

function lastUsedMs(ws: Workspace): number {
  const t = Date.parse(ws.last_used);
  return Number.isFinite(t) ? t : 0;
}

export function sortWorkspacesByLastUsed(workspaces: Workspace[]): Workspace[] {
  return [...workspaces].sort((a, b) => lastUsedMs(b) - lastUsedMs(a));
}

export function workspacesForTabBar(
  workspaces: Workspace[],
  activeId: string | null,
  maxVisible: number = TAB_BAR_MAX
): { visible: Workspace[]; overflowCount: number } {
  if (workspaces.length === 0) {
    return { visible: [], overflowCount: 0 };
  }

  const byId = new Map(workspaces.map((w) => [w.id, w]));
  const sorted = sortWorkspacesByLastUsed(workspaces);
  const visible: Workspace[] = [];
  const seen = new Set<string>();

  if (activeId && byId.has(activeId)) {
    visible.push(byId.get(activeId)!);
    seen.add(activeId);
  }

  for (const ws of sorted) {
    if (seen.has(ws.id)) continue;
    visible.push(ws);
    seen.add(ws.id);
    if (visible.length >= maxVisible) break;
  }

  const overflowCount = Math.max(0, workspaces.length - visible.length);
  return { visible, overflowCount };
}

export function filterWorkspacesForSwitcher(
  workspaces: Workspace[],
  query: string
): Workspace[] {
  const q = query.trim().toLowerCase();
  if (!q) return sortWorkspacesByLastUsed(workspaces);

  return sortWorkspacesByLastUsed(workspaces).filter((ws) => {
    const hay = [
      ws.name,
      ws.path,
      ws.git_branch ?? '',
      ws.git_remote ?? '',
    ]
      .join('\n')
      .toLowerCase();
    return hay.includes(q);
  });
}
