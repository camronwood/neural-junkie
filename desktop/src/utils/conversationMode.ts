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
export type ResolvedConversationMode = 'chat' | 'code' | 'collab';

export const CONVERSATION_MODE_STORAGE_KEY = 'conversation-mode';

const CODE_VERBS_RE =
  /\b(review|refactor|debug|fix|implement|compile|lint|test|patch|edit|change|update|add|remove|rewrite|optimize|trace|diff|analyze|analyse)\b/i;

const FILE_PATH_RE =
  /(?:^|[\s"'`(])([./]?(?:[a-zA-Z0-9_-]+\/)+[a-zA-Z0-9_-]+\.[a-zA-Z0-9]+)/;

const GREETING_RE =
  /^(?:@\w+\s+)?(?:hi|hello|hey|yo|sup|what'?s up|howdy|good (?:morning|afternoon|evening)|thanks|thank you|ok|okay|nice|cool)[!.?\s]*$/i;

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

export function hasCodeTaskSignals(message: string): boolean {
  const text = (message ?? '').trim();
  if (!text) return false;
  if (hasImplementationContinuationSignals(text)) return true;
  if (hasImplementationRequestSignals(text)) return true;
  if (hasErrorLogFollowUpSignals(text)) return true;
  if (hasScanOrEditorTaskSignals(text)) return true;
  if (/@codebase\b/i.test(text)) return true;
  if (hasCodeGraphSignals(text)) return true;
  if (CODE_VERBS_RE.test(text)) return true;
  if (FILE_PATH_RE.test(text)) return true;
  if (/`[^`]+`/.test(text)) return true;
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
