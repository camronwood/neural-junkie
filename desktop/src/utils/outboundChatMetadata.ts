import { useEditorStore } from '../stores/editorStore';
import { useFileExplorerStore } from '../stores/fileExplorerStore';
import { useSettingsStore } from '../stores/settingsStore';
import { buildFileTreeString } from './workspaceContext';
import type { WorkspaceContext } from './workspaceContext';
import { USER_RULES_METADATA_KEY } from '../constants/promptMetadata';

/**
 * Builds metadata sent with human messages so agents receive user rules, optional workspace, and attachments.
 */
export function buildHumanOutboundMetadata(options: {
  shareWorkspace: boolean;
  /** Extra metadata from the composer (e.g. prompt_attachments, image_data). */
  composerMetadata?: Record<string, unknown>;
}): Record<string, unknown> | undefined {
  const { shareWorkspace, composerMetadata } = options;
  const meta: Record<string, unknown> = { ...(composerMetadata ?? {}) };

  const rules = (useSettingsStore.getState().settings.userRulesMarkdown ?? '').trim();
  if (rules) {
    meta[USER_RULES_METADATA_KEY] = rules;
  }

  if (shareWorkspace) {
    const editorTabs = useEditorStore.getState().tabs;
    const activeTabId = useEditorStore.getState().activeTabId;
    const { workspaces, activeWorkspaceId, fileTree } = useFileExplorerStore.getState();
    const activeWorkspace = workspaces.find(w => w.id === activeWorkspaceId) ?? workspaces[0];
    const nodes = activeWorkspace ? (fileTree[activeWorkspace.id] ?? []) : [];

    const workspaceContext: WorkspaceContext = {
      workspace_name: activeWorkspace?.name ?? '',
      workspace_path: activeWorkspace?.path ?? '',
      file_tree: buildFileTreeString(nodes, 3),
      open_files: editorTabs.map(tab => ({
        path: tab.path,
        language: tab.language ?? 'text',
        content: tab.content.substring(0, 10000),
        is_active: tab.id === activeTabId,
      })),
    };
    meta.workspace_context = workspaceContext;
  }

  if (Object.keys(meta).length === 0) {
    return undefined;
  }
  return meta;
}

export { USER_RULES_METADATA_KEY, PROMPT_ATTACHMENTS_METADATA_KEY } from '../constants/promptMetadata';
