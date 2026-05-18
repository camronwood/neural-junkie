import type { ContextScope, WorkspaceContextMode } from '../constants/promptMetadata';

export type ChannelKind = 'general' | 'dm' | 'collaboration' | 'other';

export interface InferContextScopeInput {
  message: string;
  mode: WorkspaceContextMode;
  channelKind: ChannelKind;
  activeTabPath?: string;
  /** Per-send override from composer chip */
  messageOverride?: ContextScope | null;
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
  /\b(architecture|file structure|project structure|codebase structure|repo structure|directory structure|what does this repo|how is (this|the) (repo|project) (organized|structured))\b/i;

const GENERAL_RE =
  /\b(aws|azure|gcp|sso|iam|cloudformation|terraform|kubernetes|explain (the )?concept|what is|who is better|who's better|how do i (use|set up)|best practices for)\b/i;

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

function hasOutlineSignals(text: string): boolean {
  return OUTLINE_RE.test(text);
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
  if (input.messageOverride) {
    return { scope: input.messageOverride, reason: 'manual override' };
  }
  if (input.mode === 'off') {
    return { scope: 'none', reason: 'workspace mode off' };
  }
  if (input.mode === 'always') {
    return { scope: 'full', reason: 'workspace mode always' };
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
  if (channel === 'general' || channel.startsWith('project-')) return 'general';
  return 'other';
}
