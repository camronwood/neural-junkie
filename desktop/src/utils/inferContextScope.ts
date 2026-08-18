/**
 * Structural context scope for the prepare envelope.
 * Semantic attachment authority lives on the hub stamp (context_plan /
 * context_request). This helper only encodes explicit overrides + availability.
 */
import type { ContextScope, WorkspaceContextMode } from '../constants/promptMetadata';

export type ChannelKind = 'general' | 'dm' | 'collaboration' | 'other';

export interface InferContextScopeInput {
  message: string;
  mode: WorkspaceContextMode;
  channelKind: ChannelKind;
  activeTabPath?: string;
  /** Per-send override from composer chip */
  messageOverride?: ContextScope | null;
  /** IDE layout: prefer focus with open tab/selection when available */
  ideCoding?: boolean;
  /** When set, prefer the hub stamp tier over client heuristics. */
  stampContextTier?: ContextScope | null;
}

export interface InferContextScopeResult {
  scope: ContextScope;
  reason: string;
}

const FILE_PATH_RE =
  /(?:^|[\s"'`(])([./]?(?:[a-zA-Z0-9_-]+\/)+[a-zA-Z0-9_-]+\.[a-zA-Z0-9]+)/g;

/** Phoenix scan summary / analysis MCP tool requests (deterministic tool names). */
const SCAN_TOOL_RE =
  /\b(summarize_scan_summary|summarize_scan_analysis|scan summary|scan analysis|plate (viewer|qc|assay)|imageMetadata\.json|results\.json)\b/i;

export function messageRequestsScanTool(text: string): boolean {
  return SCAN_TOOL_RE.test(text ?? '');
}

export function messageAsksWorkspaceVisibility(text: string): boolean {
  const t = (text ?? '').trim();
  if (!t) return false;
  if (/\b(can you see|do you see|are you able to see).{0,48}(workspace|project|repo|codebase|files?\s+open|what i have open)\b/i.test(t)) {
    return true;
  }
  if (/\b(workspace sharing|sharing is on|shared the workspace|i('ve| have) shared|workspace is (on|shared|enabled))\b/i.test(t)) {
    return true;
  }
  if (/\b(you have|i('ve| have) given you|given you|granted you?).{0,32}workspace (access|context|sharing)\b/i.test(t)) {
    return true;
  }
  if (/\b(you have workspace|have workspace access|workspace access)\b/i.test(t)) return true;
  return /\bsee my (workspace|project|repo|codebase)\b/i.test(t);
}

export function messageReferencesOpenEditor(text: string): boolean {
  return /\b(open\s+(document|file|tab)|active\s+(file|document|tab)|in\s+(my\s+|the\s+)?editor)\b/i.test(
    text ?? '',
  );
}

export function hasCodeGraphSignals(text: string): boolean {
  const t = (text ?? '').trim();
  if (!t) return false;
  if (/\b(relate to|related to|path between|who calls|who imports|what imports|depends on|dependency on|call graph|knowledge graph|connected to|imports from|used by|where is it used|rest of the codebase)\b/i.test(t)) {
    return true;
  }
  if (/\bhow does\b.{0,100}\b(relate|connect|depend|import|call|used by|connected)\b/i.test(t)) {
    return true;
  }
  return /\b(architecture|file structure|project structure|codebase structure|repo structure|directory structure)\b/i.test(t);
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

/**
 * Resolves the prepare-envelope workspace tier from structural signals only.
 * Hub context_request overrides this after /api/turn/prepare.
 */
export function resolveContextScope(input: InferContextScopeInput): InferContextScopeResult {
  const text = (input.message ?? '').trim();
  if (input.stampContextTier) {
    return { scope: input.stampContextTier, reason: 'hub stamp context_plan tier' };
  }
  if (/^\s*\/collaborate\b/i.test(text)) {
    return {
      scope: 'outline',
      reason: 'collaboration command includes project tree without open-file bodies',
    };
  }
  if (input.messageOverride) {
    return { scope: input.messageOverride, reason: 'manual override' };
  }
  if (input.mode === 'off') {
    return { scope: 'none', reason: 'workspace mode off' };
  }
  if (input.mode === 'always') {
    return { scope: 'full', reason: 'workspace mode always' };
  }

  // Structural availability baseline for prepare (not NL phrase inference).
  if (input.ideCoding && input.activeTabPath) {
    return { scope: 'hint', reason: 'prepare envelope — IDE tab identity only' };
  }
  if (input.ideCoding) {
    return { scope: 'hint', reason: 'prepare envelope — IDE workspace identity' };
  }
  if (input.channelKind === 'collaboration') {
    return { scope: 'hint', reason: 'prepare envelope — collab workspace identity' };
  }
  if (detectFilePaths(text).length > 0) {
    // Explicit path tokens are deterministic parsing, not semantic phrase lists.
    return { scope: 'hint', reason: 'prepare envelope — explicit path tokens present' };
  }
  if (messageRequestsScanTool(text) && input.activeTabPath) {
    return { scope: 'hint', reason: 'prepare envelope — scan tool + open tab identity' };
  }
  return { scope: 'hint', reason: 'prepare envelope — structural workspace availability' };
}

export function channelNameToKind(channel: string, channelType?: string): ChannelKind {
  if (channelType === 'collaboration' || channel.startsWith('collab-')) return 'collaboration';
  if (channelType === 'dm' || channel.startsWith('dm-')) return 'dm';
  if (channel === 'general') return 'general';
  return 'other';
}

/** Map hub ContextRequest flags onto a ContextScope for trimWorkspaceContext. */
export function scopeFromContextRequest(req: {
  context_tier?: string;
  include_file_tree?: boolean;
  include_active_tab?: boolean;
  include_open_files?: boolean;
  include_document_bodies?: boolean;
}): ContextScope {
  const tier = (req.context_tier ?? '').trim().toLowerCase();
  if (tier === 'none' || tier === 'hint' || tier === 'outline' || tier === 'focus' || tier === 'full') {
    return tier;
  }
  if (req.include_open_files || req.include_document_bodies) return 'full';
  if (req.include_active_tab) return 'focus';
  if (req.include_file_tree) return 'outline';
  return 'hint';
}
