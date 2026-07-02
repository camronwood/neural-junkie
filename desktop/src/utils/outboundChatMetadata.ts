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
  EDITOR_MODE_KEY,
  EDITOR_AGENT_TRUST_KEY,
  IMPLEMENTATION_SESSION_METADATA_KEY,
  type ContextScope,
  type ConversationModeSetting,
  type WorkspaceContextMode,
  USER_RULES_METADATA_KEY,
} from '../constants/promptMetadata';
import { buildFileTreeString } from './workspaceContext';
import type { ScanSummaryContext, ScanAnalysisContext, CadContext, WorkspaceContext } from './workspaceContext';
import { concentrationAt, validationAt, isScanAnalysisResultsPath, scanAnalysisDirFromResultsPath, isScanAnalysisSummaryCSVPath } from './scanAnalysis';
import { scanAnalysisDirFromCsvPath } from './scanAnalysisCsv';
import {
  channelNameToKind,
  resolveContextScope,
  messageReferencesOpenEditor,
  messageRequestsScanTool,
  messageAsksWorkspaceVisibility,
  type ChannelKind,
} from './inferContextScope';
import { hasCodeTaskSignals, resolveConversationMode } from './conversationMode';
import {
  hasContentDeliverySignals,
  hasFileExportSignals,
  hasErrorLogFollowUpSignals,
  hasImplementationContinuationSignals,
  hasImplementationRequestSignals,
  hasImplementationStatusCheckSignals,
  hasPriorReferenceExportSignals,
  channelHasImplementationThread,
  type ChannelMessageRef,
} from './implementationContinuation';
import { hasCodeReviewSignals } from './codeReviewSignals';
import type { EffectiveComposerMode } from '../constants/composerMode';
import {
  attachTurnCapabilitiesMetadata,
  resolveTurnCapabilities,
} from '../constants/turnIntent';

const FILE_PATH_RE =
  /(?:^|[\s"'`(])([./]?(?:[a-zA-Z0-9_-]+\/)+[a-zA-Z0-9_-]+\.[a-zA-Z0-9]+)/g;

/** True when the path is a Neural Junkie collaboration sandbox, review folder, or project deliverables dir. */
export function isCollabSandboxPath(workspacePath: string): boolean {
  const normalized = (workspacePath ?? '').replace(/\\/g, '/').trim();
  if (!normalized) return false;
  if (/\/\.neural-junkie\/collaborations(\/|$)/.test(normalized)) {
    return true;
  }
  // Reject project collabs/<uuid>/ deliverable folders (not the repo root).
  return /\/collabs\/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}(\/|$)/i.test(
    normalized
  );
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

function buildCadContext(tab: EditorTab | undefined): CadContext | undefined {
  if (!tab || tab.viewMode !== 'cad-workbench') return undefined;
  const scadPath = tab.cadScadPath ?? tab.path;
  if (!scadPath) return undefined;
  return {
    scad_path: scadPath,
    project_id: tab.cadProjectId,
    note: 'CAD workbench tab is active. Use this scad_path with write_openscad, render_openscad, and list_openscad_params.',
  };
}

function buildScanSummaryContext(tab: EditorTab | undefined): ScanSummaryContext | undefined {
  if (!tab || tab.viewMode !== 'scan-summary') return undefined;
  const summaryDir = tab.scanSummaryDir ?? '';
  if (!tab.scanSummaryData) {
    if (!summaryDir) return undefined;
    return {
      summary_dir: summaryDir,
      wells_count: 0,
      analytes: [],
      note:
        'Phoenix scan summary viewer tab is active. Use summary_dir with summarize_scan_summary for QC.',
    };
  }
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
  if (!tab) return undefined;

  if (tab.viewMode !== 'scan-analysis') {
    const path = tab.path ?? '';
    if (isScanAnalysisResultsPath(path)) {
      return {
        analysis_dir: scanAnalysisDirFromResultsPath(path),
        analytes: [],
        note:
          'Phoenix scan analysis results.json is open in the editor. Use analysis_dir with summarize_scan_analysis for QC.',
      };
    }
    if (isScanAnalysisSummaryCSVPath(path)) {
      return {
        analysis_dir: scanAnalysisDirFromCsvPath(path),
        analytes: [],
        note:
          'Phoenix scan analysis summary CSV is open in the editor. Use analysis_dir with summarize_scan_analysis for QC.',
      };
    }
    return undefined;
  }

  const analysisDir = tab.scanAnalysisDir ?? '';
  if (!tab.scanAnalysisData) {
    if (!analysisDir) return undefined;
    return {
      analysis_dir: analysisDir,
      analytes: [],
      note:
        'Phoenix scan analysis viewer tab is active. Use analysis_dir with summarize_scan_analysis or run_12plex_qc for QC.',
      panel_qc_overall_pass: tab.panelQCReport?.overall_pass,
      panel_qc_analyte_count: tab.panelQCReport?.analytes?.length,
    };
  }
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
      'Phoenix scan analysis results were shared. Concentrations may require dilution factor adjustment. Use run_12plex_qc for SOP QC.',
    panel_qc_overall_pass: tab.panelQCReport?.overall_pass,
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
    workspace_id: full.workspace_id,
    workspace_kind: full.workspace_kind,
    sidecar_url: full.sidecar_url,
    workspace_name: full.workspace_name,
    workspace_path: full.workspace_path,
    file_tree: '',
    open_files: [],
    scan_summary: full.scan_summary,
    scan_analysis: full.scan_analysis,
    active_editor: full.active_editor,
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
    workspace_id: activeWorkspace?.id ?? '',
    workspace_kind: activeWorkspace?.kind ?? 'local',
    sidecar_url: activeWorkspace?.sidecar_url ?? '',
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
    cad: buildCadContext(activeTab),
    active_editor: activeTab
      ? {
          path: activeTab.path,
          view_mode: activeTab.viewMode,
          scan_summary_dir: activeTab.scanSummaryDir,
          scan_analysis_dir: activeTab.scanAnalysisDir,
          is_active: true,
        }
      : undefined,
  };
}

function messageRequestsCADWorkspace(message: string): boolean {
  return (
    /\b(create|write|save|make|generate|add|update|edit|render|export)\b/i.test(message) &&
    /\b(\.scad|openscad|cad|stl|3d|model|mesh|ball|cube|sphere|part)\b/i.test(message)
  );
}

/** True for dm-{user}-assistant channels (personal assistant only). */
export function isPersonalAssistantDmChannel(channel: string): boolean {
  const name = (channel ?? '').trim().toLowerCase();
  return name.endsWith('-assistant') && name.startsWith('dm-');
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
  /** Recent messages in the active channel (for implementation-thread carry-forward). */
  recentChannelMessages?: ChannelMessageRef[];
}): Record<string, unknown> | undefined {
  const {
    contextMode,
    message,
    channel,
    composerMetadata,
    messageOverride,
    channelType,
    ideCoding,
    recentChannelMessages,
  } = options;
  const meta: Record<string, unknown> = { ...(composerMetadata ?? {}) };

  const rules = (useSettingsStore.getState().settings.userRulesMarkdown ?? '').trim();
  if (rules) {
    meta[USER_RULES_METADATA_KEY] = rules;
  }

  const channelKind = options.channelKind ?? channelNameToKind(channel, channelType);
  const personalAssistantDm = channelKind === 'dm' && isPersonalAssistantDmChannel(channel);
  const specialistDm = channelKind === 'dm' && !personalAssistantDm;
  const implementationThreadActive =
    specialistDm && channelHasImplementationThread(recentChannelMessages);
  const implementationThreadFollowUp =
    implementationThreadActive &&
    (hasErrorLogFollowUpSignals(message) ||
      hasImplementationStatusCheckSignals(message) ||
      hasImplementationContinuationSignals(message));
  const activeTab = useEditorStore.getState().tabs.find(
    (t) => t.id === useEditorStore.getState().activeTabId
  );
  const activeTabPath = activeTab?.path;
  const hasOpenTab = Boolean(activeTabPath);
  const hasScanViewerTab =
    activeTab?.viewMode === 'scan-analysis' || activeTab?.viewMode === 'scan-summary';

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

  let explicitEditorMode =
    typeof composerMetadata?.[EDITOR_MODE_KEY] === 'string'
      ? String(composerMetadata[EDITOR_MODE_KEY]).trim()
      : '';

  // Personal-assistant DMs should stay read-only unless the message is clearly a code/export task.
  if (
    personalAssistantDm &&
    explicitEditorMode !== 'ask' &&
    !hasFileExportSignals(message) &&
    !hasPriorReferenceExportSignals(message) &&
    !hasImplementationRequestSignals(message) &&
    !hasImplementationContinuationSignals(message) &&
    !hasCodeTaskSignals(message)
  ) {
    meta[EDITOR_MODE_KEY] = 'ask';
    delete meta[IMPLEMENTATION_SESSION_METADATA_KEY];
    explicitEditorMode = 'ask';
  }

  if (explicitEditorMode === 'export' && contextMode !== 'off') {
    scope = activeTabPath ? 'focus' : 'outline';
    reason = 'composer mode: export';
    meta[CONVERSATION_MODE_METADATA_KEY] = 'code';
  } else if (
    explicitEditorMode === 'agent' &&
    contextMode !== 'off' &&
    composerMetadata?.[IMPLEMENTATION_SESSION_METADATA_KEY] === true
  ) {
    if (scope === 'none' || scope === 'hint') {
      scope = activeTabPath ? 'focus' : 'outline';
      reason = 'composer mode: agent';
    }
    meta[CONVERSATION_MODE_METADATA_KEY] = 'code';
  }

  const asksAboutOpenFile =
    messageReferencesOpenEditor(message) ||
    (/\bwhat\b/i.test(message) && /\bsee\b/i.test(message) && /\b(open|file)\b/i.test(message));

  const needsOpenEditorContext =
    contextMode !== 'off' &&
    (messageRequestsScanTool(message) || asksAboutOpenFile);

  if (needsOpenEditorContext) {
    scope = activeTabPath || hasScanViewerTab ? 'focus' : 'hint';
    reason = 'open editor or scan tool request';
    meta[CONVERSATION_MODE_METADATA_KEY] = 'code';
  } else if (implementationThreadFollowUp && contextMode !== 'off') {
    if (scope === 'none' || scope === 'hint') {
      scope = activeTabPath ? 'focus' : 'outline';
      reason = 'implementation thread continuation';
    }
    meta[CONVERSATION_MODE_METADATA_KEY] = 'code';
    if (meta[EDITOR_MODE_KEY] === 'ask' || !meta[EDITOR_MODE_KEY]) {
      meta[EDITOR_MODE_KEY] = 'agent';
      meta[IMPLEMENTATION_SESSION_METADATA_KEY] = true;
    }
    meta[EDITOR_AGENT_TRUST_KEY] = 'auto_apply_edits';
  } else if (
    specialistDm &&
    hasErrorLogFollowUpSignals(message) &&
    contextMode !== 'off'
  ) {
    if (scope === 'none' || scope === 'hint') {
      scope = activeTabPath ? 'focus' : 'outline';
      reason = 'error log follow-up';
    }
    meta[CONVERSATION_MODE_METADATA_KEY] = 'code';
  } else if (hasImplementationContinuationSignals(message) && contextMode !== 'off') {
    if (scope === 'none' || scope === 'hint') {
      scope = activeTabPath ? 'focus' : 'outline';
      reason = 'implementation continuation';
    }
    meta[CONVERSATION_MODE_METADATA_KEY] = 'code';
  } else if (
    channelKind === 'dm' &&
    contextMode !== 'off' &&
    hasImplementationRequestSignals(message)
  ) {
    if (scope === 'full' || scope === 'outline' || scope === 'none' || scope === 'hint') {
      scope = activeTabPath ? 'focus' : 'outline';
      reason = 'DM implementation request';
    }
    meta[CONVERSATION_MODE_METADATA_KEY] = 'code';
  } else if (messageAsksWorkspaceVisibility(message) && contextMode !== 'off') {
    if (scope === 'none' || scope === 'hint') {
      scope = activeTabPath ? 'focus' : 'outline';
      reason = 'workspace visibility question';
    }
  } else if (hasCodeReviewSignals(message) && contextMode !== 'off') {
    if (scope === 'none' || scope === 'hint') {
      scope = activeTabPath ? 'focus' : 'outline';
      reason = 'project code review';
    }
  } else if (hasContentDeliverySignals(message) && contextMode !== 'off') {
    if (scope === 'none' || scope === 'hint') {
      const sharedChannel = channelKind === 'general' || channelType === 'public';
      if (sharedChannel && !activeTabPath) {
        scope = 'hint';
        reason = 'content delivery on shared channel (path only)';
      } else {
        scope = activeTabPath ? 'focus' : 'outline';
        reason = 'content delivery needs project context';
      }
    }
  } else if (hasFileExportSignals(message) && contextMode !== 'off') {
    scope = activeTabPath ? 'focus' : 'outline';
    reason = 'file export to workspace';
    meta[CONVERSATION_MODE_METADATA_KEY] = 'code';
  } else if (messageRequestsCADWorkspace(message) && contextMode !== 'off') {
    if (scope === 'none' || scope === 'hint') {
      scope = 'hint';
      reason = 'CAD file operation needs workspace path';
    }
  } else if (resolvedConversationMode === 'chat' && contextMode !== 'always') {
    scope = 'none';
    reason = 'conversation mode: chat';
  } else if (resolvedConversationMode === 'collab') {
    // scope follows collab / inferContextScope rules
  } else if (
    contextMode !== 'off' &&
    hasScanViewerTab &&
    (messageRequestsScanTool(message) || messageReferencesOpenEditor(message))
  ) {
    scope = 'focus';
    reason = 'active scan viewer tab in editor';
  }

  meta[CONTEXT_SCOPE_KEY] = scope;
  meta[CONTEXT_SCOPE_REASON_KEY] = reason;

  // Personal Assistant DMs: no project workspace unless the user asked for code/export work.
  if (
    personalAssistantDm &&
    meta[EDITOR_MODE_KEY] === 'ask' &&
    !hasFileExportSignals(message) &&
    !hasPriorReferenceExportSignals(message) &&
    !hasImplementationRequestSignals(message) &&
    !hasImplementationContinuationSignals(message) &&
    !hasCodeTaskSignals(message) &&
    !hasContentDeliverySignals(message) &&
    !messageAsksWorkspaceVisibility(message)
  ) {
    scope = 'none';
    reason = 'personal assistant DM';
    meta[CONTEXT_SCOPE_KEY] = scope;
    meta[CONTEXT_SCOPE_REASON_KEY] = reason;
    meta[CONVERSATION_MODE_METADATA_KEY] = 'chat';
    delete meta.workspace_context;
  }

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

  const composerModeRaw = meta[EDITOR_MODE_KEY];
  const composerMode: EffectiveComposerMode =
    composerModeRaw === 'ask' || composerModeRaw === 'plan' || composerModeRaw === 'agent' || composerModeRaw === 'export'
      ? composerModeRaw
      : 'agent';
  if (composerMode === 'ask' || composerMode === 'plan') {
    meta[EDITOR_AGENT_TRUST_KEY] = 'interactive';
  } else {
    meta[EDITOR_AGENT_TRUST_KEY] = 'auto_apply_edits';
  }
  return attachTurnCapabilitiesMetadata(
    meta,
    resolveTurnCapabilities({
      composerMode,
      contextScope: scope,
      implementationSession: meta[IMPLEMENTATION_SESSION_METADATA_KEY] === true,
    })
  );
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
