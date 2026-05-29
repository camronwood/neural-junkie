import { useEditorStore } from '../stores/editorStore';
import { useFileExplorerStore } from '../stores/fileExplorerStore';
import { usePacksStore } from '../stores/packsStore';
import { useSettingsStore } from '../stores/settingsStore';
import type { EditorTab } from '../stores/editorStore';
import {
  COLLAB_SOURCE_MODE_KEY,
  COLLAB_SOURCE_PATH_KEY,
} from '../constants/collabWorkspace';
import {
  CONTEXT_SCOPE_KEY,
  CONTEXT_SCOPE_REASON_KEY,
  CONVERSATION_MODE_METADATA_KEY,
  type ContextScope,
  type ConversationModeSetting,
  type WorkspaceContextMode,
  USER_RULES_METADATA_KEY,
} from '../constants/promptMetadata';
import { buildFileTreeString } from './workspaceContext';
import type { ScanSummaryContext, ScanAnalysisContext, WorkspaceContext } from './workspaceContext';
import { concentrationAt, validationAt } from './scanAnalysis';
import { channelNameToKind, resolveContextScope, type ChannelKind } from './inferContextScope';
import { resolveConversationMode } from './conversationMode';

const FILE_PATH_RE =
  /(?:^|[\s"'`(])([./]?(?:[a-zA-Z0-9_-]+\/)+[a-zA-Z0-9_-]+\.[a-zA-Z0-9]+)/g;

/** True when the path is a Neural Junkie collaboration sandbox or review folder. */
export function isCollabSandboxPath(workspacePath: string): boolean {
  const normalized = (workspacePath ?? '').replace(/\\/g, '/').trim();
  if (!normalized) return false;
  return /\/\.neural-junkie\/collaborations(\/|$)/.test(normalized);
}

/** True for /collaborate slash commands (with optional flags before @mentions). */
export function isCollaborateCommand(message: string): boolean {
  return /^\s*\/collaborate\b/i.test((message ?? '').trim());
}

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

function buildScanSummaryContext(tab: EditorTab | undefined): ScanSummaryContext | undefined {
  if (!tab || tab.viewMode !== 'scan-summary' || !tab.scanSummaryData) return undefined;
  const activeWellId =
    tab.scanSummaryInitialWell && tab.scanSummaryData.byWell.has(tab.scanSummaryInitialWell)
      ? tab.scanSummaryInitialWell
      : tab.scanSummaryData.metadata[0]?.imageName;
  const activeWell = activeWellId ? tab.scanSummaryData.byWell.get(activeWellId) : undefined;
  const analytes = Array.from(
    new Set(
      tab.scanSummaryData.metadata.flatMap((well) =>
        well.spots.map((spot) => spot.analyte).filter(Boolean)
      )
    )
  ).sort();

  return {
    summary_dir: tab.scanSummaryDir ?? '',
    wells_count: tab.scanSummaryData.metadata.length,
    analytes,
    active_well: activeWell
      ? {
          well: activeWell.imageName,
          time: activeWell.time,
          fov_size_x_um: activeWell.fovSizeXUm,
          fov_size_y_um: activeWell.fovSizeYUm,
          z_stage_position_um: activeWell.zStagePositionUm,
          spot_count: activeWell.spots.length,
          spots: activeWell.spots.slice(0, 64).map((spot) => ({
            analyte: spot.analyte,
            row: spot.row,
            column: spot.column,
            x_px: spot.x_px,
            y_px: spot.y_px,
          })),
        }
      : undefined,
    note:
      'Phoenix scan summary metadata was shared. Well image pixels are not included unless an image is explicitly attached.',
  };
}

function buildScanAnalysisContext(tab: EditorTab | undefined): ScanAnalysisContext | undefined {
  if (!tab || tab.viewMode !== 'scan-analysis' || !tab.scanAnalysisData) return undefined;
  const activeWellId = tab.scanAnalysisInitialWell ?? 'A1';
  const activeAnalyte = tab.scanAnalysisSelectedAnalyte ?? tab.scanAnalysisData.analytes[0] ?? '';
  const validation = validationAt(tab.scanAnalysisData, activeWellId, activeAnalyte);
  let withinLoq: boolean | null = null;
  if (validation) {
    const unknownRows = tab.scanAnalysisData.unknownReport[activeAnalyte] ?? [];
    const unk = unknownRows.find((u) => u.wellLabel === validation.wellLabel);
    if (unk) withinLoq = unk.withinLimitsOfQuantification;
    else {
      const stdRows = tab.scanAnalysisData.standardReport[activeAnalyte] ?? [];
      const std = stdRows.find((s) => s.wellLabel === validation.wellLabel);
      if (std) withinLoq = std.withinLimitsOfQuantificationV2;
    }
  }

  return {
    analysis_dir: tab.scanAnalysisDir ?? '',
    product_name: tab.scanAnalysisData.experiment.productName,
    plate_barcode: tab.scanAnalysisData.experiment.plateBarcode,
    analytes: tab.scanAnalysisData.analytes,
    dilution_factor: tab.scanAnalysisData.experiment.dilutionFactor,
    active_analyte: activeAnalyte,
    linked_scan_dir: tab.linkedScanDir,
    active_well: validation
      ? {
          well: activeWellId,
          concentration: concentrationAt(tab.scanAnalysisData, activeWellId, activeAnalyte),
          within_loq: withinLoq,
          well_type: validation.wellType,
          well_label: validation.wellLabel,
        }
      : undefined,
    note:
      'Phoenix scan analysis results were shared. Concentrations may require dilution factor adjustment.',
  };
}

export function trimWorkspaceContext(
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
    scan_summary: full.scan_summary,
    scan_analysis: full.scan_analysis,
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
    const devPack = usePacksStore.getState().softwareDevelopmentEnabled();
    const sel = devPack ? useEditorStore.getState().activeSelection : null;
    const activePath = activeTabPath ?? files.find((t) => t.is_active)?.path;
    base.open_files = files.map((tab) => {
      const row = {
        ...tab,
        content: tab.content.substring(0, scope === 'focus' ? 10000 : 10000),
      };
      if (
        devPack &&
        sel &&
        activePath &&
        tab.path === activePath &&
        tab.is_active
      ) {
        row.selection_start_line = sel.startLine;
        row.selection_end_line = sel.endLine;
        row.selected_text = sel.text;
      }
      return row;
    });
  }
  return base;
}

function loadFullWorkspaceContext(): WorkspaceContext {
  const editorTabs = useEditorStore.getState().tabs;
  const activeTabId = useEditorStore.getState().activeTabId;
  const { workspaces, activeWorkspaceId, fileTree } = useFileExplorerStore.getState();
  const activeWorkspace = workspaces.find((w) => w.id === activeWorkspaceId) ?? workspaces[0];
  const nodes = activeWorkspace ? (fileTree[activeWorkspace.id] ?? []) : [];
  const activeTab = editorTabs.find((tab) => tab.id === activeTabId);

  return {
    workspace_name: activeWorkspace?.name ?? '',
    workspace_path: activeWorkspace?.path ?? '',
    file_tree: buildFileTreeString(nodes, 3),
    open_files: editorTabs.map((tab) => ({
      path: tab.path,
      language: tab.language ?? 'text',
      content: tab.content.substring(0, 10000),
      is_active: tab.id === activeTabId,
      view_mode: tab.viewMode,
      scan_summary_dir: tab.scanSummaryDir,
      scan_analysis_dir: tab.scanAnalysisDir,
    })),
    scan_summary: buildScanSummaryContext(activeTab),
    scan_analysis: buildScanAnalysisContext(activeTab),
  };
}

/**
 * Builds metadata sent with human messages so agents receive user rules, scoped workspace, and attachments.
 */
export function buildHumanOutboundMetadata(options: {
  contextMode: WorkspaceContextMode;
  conversationMode?: ConversationModeSetting;
  message: string;
  channel: string;
  channelKind?: ChannelKind;
  channelType?: string;
  messageOverride?: ContextScope | null;
  composerMetadata?: Record<string, unknown>;
  /** When true, attach IDE-focused workspace context (active tab + selection). */
  ideCoding?: boolean;
}): Record<string, unknown> | undefined {
  const { contextMode, message, channel, composerMetadata, messageOverride, channelType, ideCoding } =
    options;
  const meta: Record<string, unknown> = { ...(composerMetadata ?? {}) };

  const rules = (useSettingsStore.getState().settings.userRulesMarkdown ?? '').trim();
  if (rules) {
    meta[USER_RULES_METADATA_KEY] = rules;
  }

  const channelKind = options.channelKind ?? channelNameToKind(channel, channelType);
  const activeTabPath = useEditorStore.getState().tabs.find(
    (t) => t.id === useEditorStore.getState().activeTabId
  )?.path;
  const hasOpenTab = Boolean(activeTabPath);

  const conversationModeSetting = options.conversationMode ?? 'auto';
  const resolvedConversationMode = resolveConversationMode(conversationModeSetting, message, {
    ideCoding,
    channelKind,
    hasOpenTab,
  });
  meta[CONVERSATION_MODE_METADATA_KEY] = resolvedConversationMode;

  let { scope, reason } = resolveContextScope({
    message,
    mode: contextMode,
    channelKind,
    messageOverride,
    activeTabPath,
    ideCoding,
  });

  if (resolvedConversationMode === 'chat') {
    scope = 'none';
    reason = 'conversation mode: chat';
  } else if (resolvedConversationMode === 'collab') {
    // scope follows collab / inferContextScope rules
  }

  meta[CONTEXT_SCOPE_KEY] = scope;
  meta[CONTEXT_SCOPE_REASON_KEY] = reason;

  const collabMode = composerMetadata?.[COLLAB_SOURCE_MODE_KEY];
  const collabPath = composerMetadata?.[COLLAB_SOURCE_PATH_KEY];
  if (collabMode === 'none') {
    delete meta.workspace_context;
  } else if (
    collabMode === 'path' &&
    typeof collabPath === 'string' &&
    collabPath.trim()
  ) {
    const p = collabPath.trim();
    meta.workspace_context = {
      workspace_name: p.split('/').filter(Boolean).pop() ?? p,
      workspace_path: p,
      file_tree: '',
      open_files: [],
    };
  }

  if (scope !== 'none' && collabMode !== 'none') {
    const full = loadFullWorkspaceContext();
    const activePath = useEditorStore.getState().tabs.find(
      (t) => t.id === useEditorStore.getState().activeTabId
    )?.path;
    const trimmed = trimWorkspaceContext(scope, full, message, activePath);
    if (trimmed) {
      const skipCollabSandbox =
        isCollaborateCommand(message) && isCollabSandboxPath(trimmed.workspace_path);
      if (!skipCollabSandbox) {
        meta.workspace_context = trimmed;
      }
    }
  }

  // Plain /collaborate from chat should bind the active explorer workspace like the collab form.
  if (isCollaborateCommand(message) && !/\s--no-workspace\b/i.test(message)) {
    if (meta[COLLAB_SOURCE_MODE_KEY] !== 'none') {
      if (!meta[COLLAB_SOURCE_MODE_KEY]) {
        meta[COLLAB_SOURCE_MODE_KEY] = 'active';
      }
      const full = loadFullWorkspaceContext();
      const wsPath = full.workspace_path?.trim() ?? '';
      if (wsPath && !isCollabSandboxPath(wsPath) && !meta.workspace_context) {
        meta.workspace_context = {
          workspace_name: full.workspace_name,
          workspace_path: wsPath,
          file_tree: full.file_tree,
          open_files: [],
        };
        meta[COLLAB_SOURCE_PATH_KEY] = wsPath;
      } else if (wsPath && !isCollabSandboxPath(wsPath)) {
        meta[COLLAB_SOURCE_PATH_KEY] = wsPath;
      }
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
