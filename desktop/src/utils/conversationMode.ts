import type { ChannelKind } from './inferContextScope';
import {
  hasCodeGraphSignals,
  messageReferencesOpenEditor,
  messageRequestsScanTool,
} from './inferContextScope';
import {
  hasErrorLogFollowUpSignals,
  hasImplementationContinuationSignals,
  hasImplementationRequestSignals,
} from './implementationContinuation';
import { CONVERSATION_MODE_METADATA_KEY } from '../constants/promptMetadata';

export type ConversationModeSetting = 'auto' | 'chat' | 'code';
/** Resolved mode sent on outbound messages. `clarify` = Auto could not tell chat vs code. */
export type ResolvedConversationMode = 'chat' | 'code' | 'collab' | 'clarify';

export const CONVERSATION_MODE_STORAGE_KEY = 'conversation-mode';

const CODE_VERBS_RE =
  /\b(review|refactor|debug|fix|implement|compile|lint|test|patch|edit|change|update|add|remove|rewrite|optimize|trace|diff|analyze|analyse)\b/i;

/** Verbs that almost always mean hands-on code work (not casual English). */
const STRONG_CODE_VERBS_RE =
  /\b(refactor|debug|implement|compile|lint|patch|rewrite|trace|diff)\b/i;

const FILE_PATH_RE =
  /(?:^|[\s"'`(])([./]?(?:[a-zA-Z0-9_-]+\/)+[a-zA-Z0-9_-]+\.[a-zA-Z0-9]+)/;

const GREETING_RE =
  /^(?:@\w+\s+)?(?:hi|hello|hey|yo|sup|what'?s up|howdy|good (?:morning|afternoon|evening)|thanks|thank you|ok|okay|nice|cool)[!.?\s]*$/i;

const VAGUE_HELP_RE =
  /^(?:@\w+\s+)?(?:help|look|check|can you|could you|please)\b/i;

export function loadConversationModeSetting(): ConversationModeSetting {
  try {
    if (typeof localStorage === 'undefined') return 'auto';
    const stored = localStorage.getItem(CONVERSATION_MODE_STORAGE_KEY);
    if (stored === 'auto' || stored === 'chat' || stored === 'code') {
      return stored;
    }
  } catch {
    /* ignore */
  }
  return 'auto';
}

export function saveConversationModeSetting(mode: ConversationModeSetting): void {
  try {
    if (typeof localStorage === 'undefined') return;
    localStorage.setItem(CONVERSATION_MODE_STORAGE_KEY, mode);
    if (typeof window !== 'undefined') {
      window.dispatchEvent(new CustomEvent('nj-conversation-mode-changed', { detail: mode }));
    }
  } catch {
    /* ignore */
  }
}

/** @deprecated Prefer Settings → Chat; kept for tests / power users. */
export function cycleConversationModeSetting(current: ConversationModeSetting): ConversationModeSetting {
  if (current === 'auto') return 'chat';
  if (current === 'chat') return 'code';
  return 'auto';
}

export function conversationModeSettingLabel(mode: ConversationModeSetting): string {
  switch (mode) {
    case 'auto':
      return 'Auto';
    case 'chat':
      return 'Chat';
    case 'code':
      return 'Code';
  }
}

export function hasScanOrEditorTaskSignals(message: string): boolean {
  const text = (message ?? '').trim();
  if (!text) return false;
  if (messageRequestsScanTool(text)) return true;
  if (messageReferencesOpenEditor(text)) return true;
  return false;
}

/** High-confidence code/workspace task signals (not everyday English like "update"). */
export function hasStrongCodeTaskSignals(message: string): boolean {
  const text = (message ?? '').trim();
  if (!text) return false;
  if (hasImplementationContinuationSignals(text)) return true;
  if (hasImplementationRequestSignals(text)) return true;
  if (hasErrorLogFollowUpSignals(text)) return true;
  if (hasScanOrEditorTaskSignals(text)) return true;
  if (/@codebase\b/i.test(text)) return true;
  if (hasCodeGraphSignals(text)) return true;
  if (FILE_PATH_RE.test(text)) return true;
  if (STRONG_CODE_VERBS_RE.test(text)) return true;
  if (/`[^`]+`/.test(text) && CODE_VERBS_RE.test(text)) return true;
  return false;
}

export function hasCodeTaskSignals(message: string): boolean {
  const text = (message ?? '').trim();
  if (!text) return false;
  if (hasStrongCodeTaskSignals(text)) return true;
  if (CODE_VERBS_RE.test(text)) return true;
  if (/`[^`]+`/.test(text)) return true;
  return false;
}

/**
 * True when Auto cannot confidently choose chat vs code.
 * Explicit Chat/Code settings never use this — only Auto inference.
 */
export function isConversationModeAmbiguous(
  message: string,
  options?: { ideCoding?: boolean; channelKind?: ChannelKind; hasOpenTab?: boolean }
): boolean {
  const text = (message ?? '').trim();
  if (!text) return false;
  if (options?.channelKind === 'collaboration') return false;
  if (GREETING_RE.test(text)) return false;
  // IDE coding layout already implies workspace work.
  if (options?.ideCoding) return false;
  if (hasStrongCodeTaskSignals(text)) return false;

  const weakCode = hasCodeTaskSignals(text);
  const isQuestion = text.includes('?');

  // "how do I update AWS SSO?" — weak verb + question, no path.
  if (weakCode && isQuestion) return true;
  // Short/medium weak-verb asks without a clear discuss-vs-edit cue.
  if (weakCode && text.length < 100) return true;
  // Vague help with an open tab but no code signals.
  if (
    options?.hasOpenTab &&
    !weakCode &&
    !isQuestion &&
    VAGUE_HELP_RE.test(text) &&
    text.length < 80
  ) {
    return true;
  }
  return false;
}

export function inferResolvedConversationMode(
  message: string,
  options?: { ideCoding?: boolean; channelKind?: ChannelKind; hasOpenTab?: boolean }
): ResolvedConversationMode {
  if (options?.ideCoding) {
    if (GREETING_RE.test(message.trim())) {
      return 'chat';
    }
    return 'code';
  }
  if (options?.channelKind === 'collaboration') {
    return 'collab';
  }
  if (isConversationModeAmbiguous(message, options)) {
    return 'clarify';
  }
  if (hasCodeTaskSignals(message)) {
    return 'code';
  }
  if (options?.ideCoding && options?.hasOpenTab && CODE_VERBS_RE.test(message)) {
    return 'code';
  }
  if (GREETING_RE.test(message.trim())) {
    return 'chat';
  }
  if (message.includes('?') && message.trim().length >= 20 && !hasCodeTaskSignals(message)) {
    return 'chat';
  }
  if (options?.ideCoding && options?.hasOpenTab && !GREETING_RE.test(message.trim())) {
    return 'code';
  }
  return 'chat';
}

export function resolveConversationMode(
  setting: ConversationModeSetting,
  message: string,
  options?: { ideCoding?: boolean; channelKind?: ChannelKind; hasOpenTab?: boolean }
): ResolvedConversationMode {
  if (setting === 'chat') return 'chat';
  if (setting === 'code') return 'code';
  return inferResolvedConversationMode(message, options);
}

export function formatContextIndicator(options: {
  modeSetting: ConversationModeSetting;
  resolvedMode: ResolvedConversationMode;
  scope: string;
  scopeReason?: string;
  activeTabPath?: string;
}): string {
  const modeLabel =
    options.modeSetting === 'auto'
      ? `Auto→${options.resolvedMode}`
      : options.resolvedMode;
  const scopePart =
    options.scope === 'none'
      ? 'no files'
      : options.scope === 'focus' && options.activeTabPath
        ? `focus: ${options.activeTabPath.split('/').pop()}`
        : options.scope;
  return `${modeLabel} · ${scopePart}`;
}

export { CONVERSATION_MODE_METADATA_KEY };
