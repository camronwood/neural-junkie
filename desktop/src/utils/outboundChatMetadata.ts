import { useEditorStore } from '../stores/editorStore';
import { useFileExplorerStore } from '../stores/fileExplorerStore';
import { useSettingsStore } from '../stores/settingsStore';
import {
  CONTEXT_SCOPE_KEY,
  CONTEXT_SCOPE_REASON_KEY,
  type ContextScope,
  type WorkspaceContextMode,
  USER_RULES_METADATA_KEY,
} from '../constants/promptMetadata';
import { buildFileTreeString } from './workspaceContext';
import type { WorkspaceContext } from './workspaceContext';
import { channelNameToKind, resolveContextScope, type ChannelKind } from './inferContextScope';

const FILE_PATH_RE =
  /(?:^|[\s"'`(])([./]?(?:[a-zA-Z0-9_-]+\/)+[a-zA-Z0-9_-]+\.[a-zA-Z0-9]+)/g;

function detectFilePaths(text: string): string[] {
  const seen = new Set<string>();
  const out: string[] = [];
  let m: RegExpExecArray | null;
  const re = new RegExp(FILE_PATH_RE.source, FILE_PATH_RE.flags);
  while ((m = re.exec(text)) !== null) {
    const p = m[1];
    if (!seen.has(p)) {
      seen.add(p);
      out.push(p);
    }
  }
  return out;
}

function pathMatchesRef(tabPath: string, ref: string): boolean {
  if (!tabPath || !ref) return false;
  if (tabPath === ref || tabPath.endsWith('/' + ref) || tabPath.endsWith(ref)) return true;
  const base = ref.split('/').pop();
  return base != null && tabPath.endsWith('/' + base);
}

function trimWorkspaceContext(
  scope: ContextScope,
  full: WorkspaceContext,
  message: string,
  activeTabPath?: string
): WorkspaceContext | null {
  if (scope === 'none') return null;
  const base: WorkspaceContext = {
    workspace_name: full.workspace_name,
    workspace_path: full.workspace_path,
    file_tree: '',
    open_files: [],
  };
  if (scope === 'hint') return base;
  if (scope === 'outline' || scope === 'focus' || scope === 'full') {
    base.file_tree = full.file_tree;
  }
  if (scope === 'focus' || scope === 'full') {
    const refs = detectFilePaths(message);
    const tabs = full.open_files ?? [];
    let files = tabs;
    if (scope === 'focus') {
      files = tabs.filter(
        (tab) =>
          tab.is_active ||
          refs.some((r) => pathMatchesRef(tab.path, r))
      );
      if (files.length === 0 && activeTabPath) {
        const active = tabs.find((t) => t.path === activeTabPath);
        if (active) files = [active];
      }
    }
    base.open_files = files.map((tab) => ({
      ...tab,
      content: tab.content.substring(0, scope === 'focus' ? 10000 : 10000),
    }));
  }
  return base;
}

function loadFullWorkspaceContext(): WorkspaceContext {
  const editorTabs = useEditorStore.getState().tabs;
  const activeTabId = useEditorStore.getState().activeTabId;
  const { workspaces, activeWorkspaceId, fileTree } = useFileExplorerStore.getState();
  const activeWorkspace = workspaces.find((w) => w.id === activeWorkspaceId) ?? workspaces[0];
  const nodes = activeWorkspace ? (fileTree[activeWorkspace.id] ?? []) : [];

  return {
    workspace_name: activeWorkspace?.name ?? '',
    workspace_path: activeWorkspace?.path ?? '',
    file_tree: buildFileTreeString(nodes, 3),
    open_files: editorTabs.map((tab) => ({
      path: tab.path,
      language: tab.language ?? 'text',
      content: tab.content.substring(0, 10000),
      is_active: tab.id === activeTabId,
    })),
  };
}

/**
 * Builds metadata sent with human messages so agents receive user rules, scoped workspace, and attachments.
 */
export function buildHumanOutboundMetadata(options: {
  contextMode: WorkspaceContextMode;
  message: string;
  channel: string;
  channelKind?: ChannelKind;
  channelType?: string;
  messageOverride?: ContextScope | null;
  composerMetadata?: Record<string, unknown>;
}): Record<string, unknown> | undefined {
  const { contextMode, message, channel, composerMetadata, messageOverride, channelType } = options;
  const meta: Record<string, unknown> = { ...(composerMetadata ?? {}) };

  const rules = (useSettingsStore.getState().settings.userRulesMarkdown ?? '').trim();
  if (rules) {
    meta[USER_RULES_METADATA_KEY] = rules;
  }

  const channelKind = options.channelKind ?? channelNameToKind(channel, channelType);
  const { scope, reason } = resolveContextScope({
    message,
    mode: contextMode,
    channelKind,
    messageOverride,
    activeTabPath: useEditorStore.getState().tabs.find(
      (t) => t.id === useEditorStore.getState().activeTabId
    )?.path,
  });

  meta[CONTEXT_SCOPE_KEY] = scope;
  meta[CONTEXT_SCOPE_REASON_KEY] = reason;

  if (scope !== 'none') {
    const full = loadFullWorkspaceContext();
    const activePath = useEditorStore.getState().tabs.find(
      (t) => t.id === useEditorStore.getState().activeTabId
    )?.path;
    const trimmed = trimWorkspaceContext(scope, full, message, activePath);
    if (trimmed) {
      meta.workspace_context = trimmed;
    }
  }

  if (Object.keys(meta).length === 0) {
    return undefined;
  }
  return meta;
}

export { USER_RULES_METADATA_KEY, PROMPT_ATTACHMENTS_METADATA_KEY } from '../constants/promptMetadata';

export const WORKSPACE_CONTEXT_MODE_KEY = 'workspace-context-mode';

export function loadWorkspaceContextMode(): WorkspaceContextMode {
  try {
    if (typeof localStorage === 'undefined') {
      return 'auto';
    }
    const legacy = localStorage.getItem('share-workspace');
    const stored = localStorage.getItem(WORKSPACE_CONTEXT_MODE_KEY);
    if (stored === 'auto' || stored === 'always' || stored === 'off') {
      return stored;
    }
    if (legacy === 'true') return 'always';
    if (legacy === 'false') return 'off';
  } catch {
    /* ignore */
  }
  return 'auto';
}

export function cycleWorkspaceContextMode(current: WorkspaceContextMode): WorkspaceContextMode {
  if (current === 'auto') return 'always';
  if (current === 'always') return 'off';
  return 'auto';
}

export function workspaceContextModeLabel(mode: WorkspaceContextMode): string {
  switch (mode) {
    case 'auto':
      return 'Auto';
    case 'always':
      return 'Always';
    case 'off':
      return 'Off';
  }
}
