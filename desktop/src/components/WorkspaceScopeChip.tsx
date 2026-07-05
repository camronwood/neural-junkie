import { useEffect } from 'react';
import { useProjectSetsStore } from '../stores/projectSetsStore';
import { useFileExplorerStore } from '../stores/fileExplorerStore';
import { scopeSummaryLabel, resolveWorkspaceScope } from '../utils/workspaceScope';
import { useEditorStore } from '../stores/editorStore';

/** Shows active multi-repo scope and project set selector. */
export function WorkspaceScopeChip() {
  const workspaces = useFileExplorerStore((s) => s.workspaces);
  const activeWorkspaceId = useFileExplorerStore((s) => s.activeWorkspaceId);
  const editorTabs = useEditorStore((s) => s.tabs);
  const activeTabId = useEditorStore((s) => s.activeTabId);
  const projectSets = useProjectSetsStore((s) => s.projectSets);
  const activeProjectSetId = useProjectSetsStore((s) => s.activeProjectSetId);
  const setActiveProjectSet = useProjectSetsStore((s) => s.setActiveProjectSet);
  const loadProjectSets = useProjectSetsStore((s) => s.loadProjectSets);
  const getMemberIds = useProjectSetsStore((s) => s.getMemberIds);
  const createProjectSet = useProjectSetsStore((s) => s.createProjectSet);

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
  const label = scopeSummaryLabel(scope);

  const handleCreateProjectSet = async () => {
    const name = window.prompt('Project set name (e.g. Neural Junkie Platform)');
    if (!name?.trim()) return;
    const primary = workspaces.find((w) => w.id === activeWorkspaceId) ?? workspaces[0];
    if (!primary) return;
    const members = workspaces.filter((w) => w.id !== primary.id).map((w) => w.id);
    const ps = await createProjectSet({
      name: name.trim(),
      primaryWorkspaceId: primary.id,
      memberWorkspaceIds: members,
    });
    if (ps) {
      setActiveProjectSet(ps.id);
    }
  };

  if (!label && projectSets.length === 0 && workspaces.length < 2) {
    return (
      <button
        type="button"
        className="ml-2 px-2 py-0.5 text-[10px] uppercase tracking-wide rounded bg-slack-bgHover text-slack-textMuted"
        onClick={() => void handleCreateProjectSet()}
        title="Save linked repos as a project set"
      >
        Link repos
      </button>
    );
  }

  return (
    <span className="inline-flex items-center gap-2 ml-2">
      {label ? (
        <span
          className="px-2 py-0.5 text-[10px] uppercase tracking-wide rounded bg-slack-bgHover text-slack-textMuted whitespace-nowrap"
          title="Repositories included in agent scope for this message"
        >
          Scope: {label}
        </span>
      ) : null}
      <select
        className="text-[10px] uppercase tracking-wide rounded bg-slack-bgHover text-slack-textMuted border-none py-0.5 px-1 max-w-[140px]"
        value={activeProjectSetId ?? ''}
        onChange={(e) => setActiveProjectSet(e.target.value || null)}
        title="Project set — linked repos stay in agent scope"
      >
        <option value="">No project set</option>
        {projectSets.map((ps) => (
          <option key={ps.id} value={ps.id}>
            {ps.name}
          </option>
        ))}
      </select>
      <button
        type="button"
        className="px-1.5 py-0.5 text-[10px] rounded bg-slack-bgHover text-slack-textMuted"
        onClick={() => void handleCreateProjectSet()}
        title="Create project set from registered workspaces"
      >
        +
      </button>
    </span>
  );
}
