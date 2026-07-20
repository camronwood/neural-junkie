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
}

export interface InferContextScopeResult {
  scope: ContextScope;
  reason: string;
}

const FILE_PATH_RE =
  /(?:^|[\s"'`(])([./]?(?:[a-zA-Z0-9_-]+\/)+[a-zA-Z0-9_-]+\.[a-zA-Z0-9]+)/g;

const CODE_VERBS_RE =
  /\b(review|reivew|refactor|debug|fix|implement|compile|lint|test|patch|edit|change|update|add|remove|rewrite|optimize|trace|diff)\b/i;

/** User refers to an open editor tab without naming a repo path. */
const EDITOR_DOCUMENT_RE =
  /\b(new\s+(document|file)|document\s+open|file\s+open|open\s+(document|file|one)|in\s+(my\s+|the\s+)?editor|editor\s+open|active\s+(file|document|tab)|have\s+.{0,16}open|opened\s+.{0,16}(editor|document|file)|review\s+(this|the|it|that|one|doc)|can\s+.{0,24}review|take\s+a\s+look|look\s+at\s+(this|the|it|that|one))\b/i;

const OUTLINE_RE =
  /\b(architecture|file structure|project structure|codebase structure|repo structure|directory structure|what does this repo|how is (this|the) (repo|project) (organized|structured)|rest of the codebase)\b/i;

/** Mirrors hub knowledge-router code_graph cues (Knowledge Graph "Ask agents" prefills). */
const CODE_GRAPH_CUES_RE =
  /\b(relate to|related to|path between|who calls|who imports|what imports|depends on|dependency on|call graph|knowledge graph|connected to|imports from|used by|where is it used|rest of the codebase)\b/i;

const HOW_DOES_GRAPH_RE =
  /\bhow does\b.{0,100}\b(relate|connect|depend|import|call|used by|connected)\b/i;

const GENERAL_RE =
  /\b(aws|azure|gcp|sso|iam|cloudformation|terraform|kubernetes|explain (the )?concept|what is|who is better|who's better|how do i (use|set up)|best practices for)\b/i;

/** Phoenix scan summary / analysis MCP tool requests. */
const SCAN_TOOL_RE =
  /\b(summarize_scan_summary|summarize_scan_analysis|scan summary|scan analysis|plate (viewer|qc|assay)|imageMetadata\.json|results\.json)\b/i;

/** User asks whether the agent can see their workspace / open project. */
const WORKSPACE_VISIBILITY_RE =
  /\b(can you see|do you see|are you able to see).{0,48}(workspace|project|repo|codebase|files?\s+open|what i have open)\b/i;

const WORKSPACE_SHARING_RE =
  /\b(workspace sharing|sharing is on|shared the workspace|i('ve| have) shared|workspace is (on|shared|enabled))\b/i;

/** User confirms the agent has workspace access (not always phrased as a question). */
const WORKSPACE_ACCESS_RE =
  /\b(you have|i('ve| have) given you|given you|granted you?).{0,32}workspace (access|context|sharing)\b/i;

function hasScanToolSignals(text: string): boolean {
  return SCAN_TOOL_RE.test(text);
}

export function messageRequestsScanTool(text: string): boolean {
  return hasScanToolSignals(text);
}

export function messageAsksWorkspaceVisibility(text: string): boolean {
  const t = (text ?? '').trim();
  if (!t) return false;
  if (WORKSPACE_VISIBILITY_RE.test(t)) return true;
  if (WORKSPACE_SHARING_RE.test(t)) return true;
  if (WORKSPACE_ACCESS_RE.test(t)) return true;
  if (/\b(you have workspace|have workspace access|workspace access)\b/i.test(t)) return true;
  return /\bsee my (workspace|project|repo|codebase)\b/i.test(t);
}

export function messageReferencesOpenEditor(text: string): boolean {
  return hasEditorDocumentSignals(text);
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

function hasCodeSignals(text: string): boolean {
  if (detectFilePaths(text).length > 0) return true;
  if (CODE_VERBS_RE.test(text)) return true;
  if (/`[^`]+`/.test(text)) return true;
  return false;
}

/** True when the message asks about repo structure / graph relations (needs workspace_path). */
export function hasCodeGraphSignals(text: string): boolean {
  const t = (text ?? '').trim();
  if (!t) return false;
  if (CODE_GRAPH_CUES_RE.test(t)) return true;
  if (HOW_DOES_GRAPH_RE.test(t)) return true;
  if (/\bcodebase\b/i.test(t) && /\b(how|what|where|relate|related|structure|organized|rest of)\b/i.test(t)) {
    return true;
  }
  return OUTLINE_RE.test(t);
}

function hasOutlineSignals(text: string): boolean {
  return hasCodeGraphSignals(text);
}

function hasGeneralSignals(text: string): boolean {
  if (GENERAL_RE.test(text)) return true;
  if (/\bwho should i ask\b/i.test(text)) return true;
  if (/@\w+.*\bwho\b/i.test(text) && !hasCodeSignals(text)) return true;
  return false;
}

function hasEditorDocumentSignals(text: string): boolean {
  return EDITOR_DOCUMENT_RE.test(text);
}

function wantsActiveEditorContext(text: string, activeTabPath?: string): boolean {
  if (!activeTabPath) return false;
  if (hasEditorDocumentSignals(text)) return true;
  if (hasCodeSignals(text)) return true;
  return /\b(look at|check|read|feedback|proofread|critique)\b/i.test(text);
}

/**
 * Resolves how much workspace context to attach for an outbound human message.
 */
export function resolveContextScope(input: InferContextScopeInput): InferContextScopeResult {
  const text = (input.message ?? '').trim();
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

  if (input.ideCoding && input.activeTabPath) {
    return { scope: 'focus', reason: 'IDE layout — active file and selection' };
  }
  if (input.ideCoding) {
    return { scope: 'outline', reason: 'IDE layout — project tree' };
  }

  if (hasScanToolSignals(text) && input.activeTabPath) {
    return { scope: 'focus', reason: 'scan summary/analysis tool request with open tab' };
  }

  // auto
  if (input.channelKind === 'collaboration') {
    if (hasCodeSignals(text)) {
      return { scope: 'focus', reason: 'collab channel with code signals' };
    }
    return { scope: 'hint', reason: 'collab planning default' };
  }

  if (hasCodeSignals(text)) {
    if (input.mode === 'auto' && /\b(all files|entire (repo|project|codebase)|whole project)\b/i.test(text)) {
      return { scope: 'full', reason: 'explicit whole-repo request' };
    }
    return { scope: 'focus', reason: 'paths or code verbs in message' };
  }
  if (hasOutlineSignals(text)) {
    return { scope: 'outline', reason: 'structure or architecture question' };
  }
  if (messageAsksWorkspaceVisibility(text)) {
    if (input.activeTabPath) {
      return { scope: 'focus', reason: 'workspace visibility question with open tab' };
    }
    return { scope: 'outline', reason: 'workspace visibility question' };
  }
  if (hasGeneralSignals(text) || text.length < 12) {
    return { scope: 'none', reason: 'general or short message' };
  }
  if (hasEditorDocumentSignals(text) || wantsActiveEditorContext(text, input.activeTabPath)) {
    return { scope: 'focus', reason: 'editor document or active tab review' };
  }
  return { scope: 'hint', reason: 'ambiguous — project hint only' };
}

export function channelNameToKind(channel: string, channelType?: string): ChannelKind {
  if (channelType === 'collaboration' || channel.startsWith('collab-')) return 'collaboration';
  if (channelType === 'dm' || channel.startsWith('dm-')) return 'dm';
  if (channel === 'general') return 'general';
  return 'other';
}
