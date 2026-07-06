import { useEffect } from 'react';
import { useProjectSetsStore } from '../stores/projectSetsStore';
import { useFileExplorerStore } from '../stores/fileExplorerStore';
import { useEditorStore } from '../stores/editorStore';
import { resolveWorkspaceScope, scopeSummaryLabel } from '../utils/workspaceScope';

/** Shared agent-scope state for Files panel UI. */
export function useWorkspaceAgentScope() {
  const workspaces = useFileExplorerStore((s) => s.workspaces);
  const activeWorkspaceId = useFileExplorerStore((s) => s.activeWorkspaceId);
  const editorTabs = useEditorStore((s) => s.tabs);
  const activeTabId = useEditorStore((s) => s.activeTabId);
  const projectSets = useProjectSetsStore((s) => s.projectSets);
  const activeProjectSetId = useProjectSetsStore((s) => s.activeProjectSetId);
  const loadProjectSets = useProjectSetsStore((s) => s.loadProjectSets);
  const getMemberIds = useProjectSetsStore((s) => s.getMemberIds);

  useEffect(() => {
    void loadProjectSets();
  }, [loadProjectSets]);

  const scope = resolveWorkspaceScope({
    workspaces,
    activeWorkspaceId,
    editorTabs,
    activeTabId,
    projectSetMemberIds: activeProjectSetId ? getMemberIds(activeProjectSetId) : undefined,
  });
  const scopeLabel = scopeSummaryLabel(scope);
  const activeProjectSet = projectSets.find((ps) => ps.id === activeProjectSetId);

  return { scope, scopeLabel, activeProjectSet, projectSets, activeProjectSetId };
}
