import { useEditorStore } from '../stores/editorStore';
import { useFileChangeStore } from '../stores/fileChangeStore';
import { useFileExplorerStore } from '../stores/fileExplorerStore';
import { useSettingsStore } from '../stores/settingsStore';
import type { FileChange } from '../types/protocol';
import { getLanguageFromPath } from './editorLanguage';

/** Match a pending change to an open editor tab by absolute or relative path. */
export function findPendingChangeForTabPath(
  pendingChanges: ReadonlyArray<FileChange>,
  tabPath: string,
): FileChange | null {
  const normalized = tabPath.replace(/\\/g, '/');
  return (
    pendingChanges.find((c) => {
      const p = (c.file_path || c.new_path || '').replace(/\\/g, '/');
      return p === normalized || p.endsWith(`/${normalized}`) || normalized.endsWith(`/${p}`);
    }) ?? null
  );
}

/**
 * Speculatively sync a pending file proposal into the editor: select it, load diff,
 * and open/focus the file when a matching workspace is available.
 */
export async function syncPendingChangeToEditor(change: FileChange): Promise<void> {
  const absolutePath = change.file_path || change.old_path || change.new_path || '';
  if (!absolutePath) return;

  useFileChangeStore.getState().selectChange(change.id);
  try {
    await useFileChangeStore.getState().getFileDiff(change.id);
  } catch {
    // Diff may fail for deletes; still try to open the path.
  }

  const explorer = useFileExplorerStore.getState();
  const workspace =
    explorer.workspaces.find(
      (item) => absolutePath === item.path || absolutePath.startsWith(`${item.path}/`),
    ) ??
    explorer.workspaces.find((item) => item.id === explorer.activeWorkspaceId) ??
    explorer.workspaces[0];
  if (!workspace) return;

  const relativePath = absolutePath.startsWith(`${workspace.path}/`)
    ? absolutePath.slice(workspace.path.length + 1)
    : absolutePath;

  const existing = useEditorStore
    .getState()
    .tabs.find((tab) => tab.workspaceId === workspace.id && tab.path === relativePath);
  if (existing) {
    useEditorStore.getState().setActiveTab(existing.id);
  } else {
    const content =
      change.operation === 'create'
        ? change.new_content || ''
        : change.old_content || change.new_content || '';
    useEditorStore
      .getState()
      .openFile(workspace.id, relativePath, content, getLanguageFromPath(relativePath));
  }

  await useSettingsStore.getState().updateLayoutSettings({
    editorPanelVisible: true,
    filesPanelVisible: true,
  });
}

/** After pending list refresh, sync the newest matching proposal for open tabs. */
export async function syncOpenTabsWithPendingChanges(
  pendingChanges: ReadonlyArray<FileChange>,
): Promise<void> {
  const pending = Array.isArray(pendingChanges) ? pendingChanges : [];
  if (pending.length === 0) return;

  const tabs = useEditorStore.getState().tabs ?? [];
  const activeId = useEditorStore.getState().activeTabId;
  const ordered = [
    ...tabs.filter((t) => t.id === activeId),
    ...tabs.filter((t) => t.id !== activeId),
  ];
  for (const tab of ordered) {
    const change = findPendingChangeForTabPath(pending, tab.path);
    if (!change) continue;
    if (change.operation !== 'edit' && change.operation !== 'create') continue;
    await syncPendingChangeToEditor(change);
    return;
  }
  // No open-tab match: still open the newest pending edit so hunks are reviewable.
  const newest = [...pending]
    .filter((c) => c.operation === 'edit' || c.operation === 'create')
    .sort((a, b) => Date.parse(b.requested_at || '') - Date.parse(a.requested_at || ''))[0];
  if (newest) {
    await syncPendingChangeToEditor(newest);
  }
}
