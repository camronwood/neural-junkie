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
  EDITOR_MODE_KEY,
  EDITOR_AGENT_TRUST_KEY,
  IMPLEMENTATION_SESSION_METADATA_KEY,
  type ContextScope,
  type ConversationModeSetting,
  type WorkspaceContextMode,
  USER_RULES_METADATA_KEY,
  LINKED_WORKSPACES_METADATA_KEY,
  GRANTED_DEVICE_LOCATION_KEY,
  type LinkedWorkspaceContext,
} from '../constants/promptMetadata';
import { useLocationShareStore } from '../stores/locationShareStore';
import { buildFileTreeString } from './workspaceContext';
import type { ScanSummaryContext, ScanAnalysisContext, CadContext, StructureContext, MusicContext, WorkspaceContext } from './workspaceContext';
import { concentrationAt, validationAt, isScanAnalysisResultsPath, scanAnalysisDirFromResultsPath, isScanAnalysisSummaryCSVPath } from './scanAnalysis';
import { scanAnalysisDirFromCsvPath } from './scanAnalysisCsv';
import {
  channelNameToKind,
  type ChannelKind,
} from './inferContextScope';
import type { ChannelMessageRef } from './implementationContinuation';
import type { EffectiveComposerMode } from '../constants/composerMode';
import {
  attachTurnCapabilitiesMetadata,
  resolveTurnCapabilities,
} from '../constants/turnIntent';
import { resolveWorkspaceScope, scopeSummaryLabel } from './workspaceScope';
import { useProjectSetsStore } from '../stores/projectSetsStore';

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

function buildStructureContext(tab: EditorTab | undefined): StructureContext | undefined {
  if (!tab || tab.viewMode !== 'structure-workbench') return undefined;
  const structurePath = tab.structurePath ?? tab.path;
  if (!structurePath) return undefined;
  return {
    structure_path: structurePath,
    note: 'Structure workbench tab is active. Use structure_metadata for confidence summary; discuss fold results with the user.',
  };
}

function buildMusicContext(tab: EditorTab | undefined): MusicContext | undefined {
  if (!tab || tab.viewMode !== 'music-workbench') return undefined;
  return {
    music_path: tab.musicPath,
    project_path: tab.musicProjectPath ?? (tab.path.endsWith('.nj-music.json') ? tab.path : undefined),
    note: 'Music workbench tab is active. Prefer updating project.nj-music.json sections; use generate_music or extract_stems for audio.',
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
    const ideOn = usePacksStore.getState().ideEnabled();
    const sel = ideOn ? useEditorStore.getState().activeSelection : null;
    const activeTabId = useEditorStore.getState().activeTabId;
    const activePath = activeTabPath ?? files.find((t) => t.is_active)?.path;
    base.open_files = files.map((tab) => {
      const row = {
        ...tab,
        content: tab.content.substring(0, scope === 'focus' ? 10000 : 10000),
      };
      if (
        ideOn &&
        sel &&
        sel.tabId === activeTabId &&
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

function loadScopedWorkspaceContext(): {
  primary: WorkspaceContext;
  linked: LinkedWorkspaceContext[];
  scopeLabel: string | null;
} {
  const editorTabs = useEditorStore.getState().tabs;
  const activeTabId = useEditorStore.getState().activeTabId;
  const { workspaces, activeWorkspaceId, fileTree } = useFileExplorerStore.getState();
  const activeProjectSetId = useProjectSetsStore.getState().activeProjectSetId;
  const projectSetMemberIds = activeProjectSetId
    ? useProjectSetsStore.getState().getMemberIds(activeProjectSetId)
    : undefined;

  const scope = resolveWorkspaceScope({
    workspaces,
    activeWorkspaceId,
    editorTabs,
    activeTabId,
    projectSetMemberIds,
  });

  const primaryWs = scope.primary;
  const primaryNodes = primaryWs ? (fileTree[primaryWs.id] ?? []) : [];
  const primaryTabs = primaryWs
    ? editorTabs.filter((tab) => tab.workspaceId === primaryWs.id)
    : editorTabs;
  const activeTab = editorTabs.find((tab) => tab.id === activeTabId);

  const primary: WorkspaceContext = {
    workspace_id: primaryWs?.id ?? '',
    workspace_kind: primaryWs?.kind ?? 'local',
    sidecar_url: primaryWs?.sidecar_url ?? '',
    workspace_name: primaryWs?.name ?? '',
    workspace_path: primaryWs?.path ?? '',
    file_tree: buildFileTreeString(primaryNodes, 3),
    open_files: primaryTabs.map((tab) => ({
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
    structure: buildStructureContext(activeTab),
    music: buildMusicContext(activeTab),
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

  const linked: LinkedWorkspaceContext[] = scope.linked.map((lw) => {
    const nodes = fileTree[lw.workspace_id] ?? [];
    return {
      ...lw,
      file_tree: buildFileTreeString(nodes, 2),
    };
  });

  return { primary, linked, scopeLabel: scopeSummaryLabel(scope) };
}

/** @deprecated use loadScopedWorkspaceContext */
function loadFullWorkspaceContext(): WorkspaceContext {
  return loadScopedWorkspaceContext().primary;
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
  } = options;
  const meta: Record<string, unknown> = { ...(composerMetadata ?? {}) };

  const rules = (useSettingsStore.getState().settings.userRulesMarkdown ?? '').trim();
  if (rules) {
    meta[USER_RULES_METADATA_KEY] = rules;
  }

  const locationShare = useLocationShareStore.getState();
  if (locationShare.sharing && locationShare.snapshot && locationShare.granted !== false) {
    const snap = locationShare.snapshot;
    const captured = Date.parse(snap.captured_at);
    const ageS = Number.isFinite(captured) ? Math.max(0, (Date.now() - captured) / 1000) : snap.age_s ?? 0;
    meta[GRANTED_DEVICE_LOCATION_KEY] = {
      lat: snap.lat,
      lon: snap.lon,
      accuracy_m: snap.accuracy_m,
      display_name: snap.display_name,
      captured_at: snap.captured_at,
      age_s: ageS,
      source: snap.source ?? 'session',
      shared: true,
    };
  }

  const activeTab = useEditorStore.getState().tabs.find(
    (t) => t.id === useEditorStore.getState().activeTabId
  );
  const activeTabPath = activeTab?.path;
  if (
    activeTab?.viewMode === 'neural-canvas' &&
    activeTab.artifactId &&
    activeTab.artifactId !== '__library__'
  ) {
    meta.open_artifact = {
      id: activeTab.artifactId,
      title: activeTab.path || 'Neural Canvas',
      // Always stamp renderer_id — hub open-canvas promote needs it. Default
      // markdown when the tab has not yet cached a renderer from a fetch.
      renderer_id: activeTab.artifactRendererId || 'nj.markdown',
    };
  }
  const channelKind = options.channelKind ?? channelNameToKind(channel, channelType);
  const composerModeRaw = meta[EDITOR_MODE_KEY];
  const composerMode: EffectiveComposerMode =
    composerModeRaw === 'ask' ||
    composerModeRaw === 'plan' ||
    composerModeRaw === 'agent' ||
    composerModeRaw === 'export'
      ? composerModeRaw
      : 'agent';

  let scope: ContextScope;
  let reason: string;
  if (messageOverride) {
    scope = messageOverride;
    reason = 'manual override';
  } else if (contextMode === 'off') {
    scope = 'none';
    reason = 'workspace mode off';
  } else if (contextMode === 'always') {
    scope = 'full';
    reason = 'workspace mode always';
  } else if (ideCoding || composerMode === 'export') {
    scope = activeTabPath ? 'focus' : 'outline';
    reason = activeTabPath ? 'active editor context' : 'workspace outline';
  } else if (channelKind === 'collaboration') {
    scope = 'hint';
    reason = 'collaboration workspace hint';
  } else {
    scope = 'hint';
    reason = 'workspace mode auto';
  }

  meta[EDITOR_MODE_KEY] = composerMode;
  if (composerMode === 'ask' || composerMode === 'plan') {
    meta[EDITOR_AGENT_TRUST_KEY] = 'interactive';
    delete meta[IMPLEMENTATION_SESSION_METADATA_KEY];
  } else if (composerMode === 'export') {
    meta[IMPLEMENTATION_SESSION_METADATA_KEY] = true;
  } else {
    delete meta[IMPLEMENTATION_SESSION_METADATA_KEY];
  }
  delete meta.conversation_mode;

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
    const { primary: full, linked, scopeLabel } = loadScopedWorkspaceContext();
    const activePath = useEditorStore.getState().tabs.find(
      (t) => t.id === useEditorStore.getState().activeTabId
    )?.path;
    const trimmed = trimWorkspaceContext(scope, full, message, activePath);
    if (trimmed) {
      const skipCollabSandbox =
        isCollaborateCommand(message) && isCollabSandboxPath(trimmed.workspace_path);
      if (!skipCollabSandbox) {
        meta.workspace_context = trimmed;
        if (linked.length > 0) {
          meta[LINKED_WORKSPACES_METADATA_KEY] = linked;
        }
        if (scopeLabel) {
          meta.scope_summary = scopeLabel;
        }
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

  return attachTurnCapabilitiesMetadata(
    meta,
    resolveTurnCapabilities({
      composerMode,
      contextScope: scope,
      implementationSession: meta[IMPLEMENTATION_SESSION_METADATA_KEY] === true,
    }),
    {
      trustPreference:
        typeof meta[EDITOR_AGENT_TRUST_KEY] === 'string'
          ? (meta[EDITOR_AGENT_TRUST_KEY] as string)
          : undefined,
    }
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
