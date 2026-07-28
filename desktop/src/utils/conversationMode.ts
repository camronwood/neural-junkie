import type { ChannelKind } from './inferContextScope';
import { CONVERSATION_MODE_METADATA_KEY } from '../constants/promptMetadata';

export type ConversationModeSetting = 'auto' | 'chat' | 'code';
/** Resolved mode sent on outbound messages. `clarify` = Auto could not tell chat vs code. */
export type ResolvedConversationMode = 'chat' | 'code' | 'collab' | 'clarify';

export const CONVERSATION_MODE_STORAGE_KEY = 'conversation-mode';

/** @here / @channel / @everyone — channel-wide fan-out (structural mention tokens). */
const HERE_MENTION_RE = /(?:^|[\s])@(?:here|channel|everyone)\b/i;

export function stripLeadingMentions(message: string): string {
  return (message ?? '')
    .replace(/^(?:\s*@(?:here|channel|everyone|\w+)\b)+\s*/gi, '')
    .trim();
}

export function hasHereOrChannelMention(message: string): boolean {
  return HERE_MENTION_RE.test(message ?? '');
}

/** Casual room ping — discuss, don't dive into tools/repo. Mentions + short body; no NL verb banks. */
export function isSocialOrStatusPing(message: string): boolean {
  const text = (message ?? '').trim();
  if (!text) return false;
  if (hasStrongCodeTaskSignals(text)) return false;
  const stripped = stripLeadingMentions(text);
  if (!stripped) {
    return hasHereOrChannelMention(text);
  }
  // Short @here/@channel body without structural code signals stays chat.
  if (hasHereOrChannelMention(text) && stripped.length <= 80) {
    return true;
  }
  return false;
}

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

/** Strong code signals — structural only (@codebase / path-like tokens). NL verbs removed. */
export function hasStrongCodeTaskSignals(message: string): boolean {
  const text = (message ?? '').trim();
  if (!text) return false;
  if (/@codebase\b/i.test(text)) return true;
  if (/(?:^|[\s"'`(])([./]?(?:[a-zA-Z0-9_-]+\/)+[a-zA-Z0-9_-]+\.[a-zA-Z0-9]+)/.test(text)) {
    return true;
  }
  return false;
}

export function hasCodeTaskSignals(message: string): boolean {
  return hasStrongCodeTaskSignals(message);
}

export function hasScanOrEditorTaskSignals(_message: string): boolean {
  return false;
}

export function isConversationModeAmbiguous(
  _message: string,
  options?: { ideCoding?: boolean; channelKind?: ChannelKind; hasOpenTab?: boolean }
): boolean {
  if (options?.channelKind === 'collaboration') return false;
  if (options?.ideCoding) return false;
  return false;
}

/**
 * Auto no longer invents chat vs code from NL phrases. Explicit setting wins;
 * Auto defaults to chat unless IDE coding layout or collab channel.
 */
export function inferResolvedConversationMode(
  message: string,
  options?: { ideCoding?: boolean; channelKind?: ChannelKind; hasOpenTab?: boolean }
): ResolvedConversationMode {
  if (isSocialOrStatusPing(message)) {
    return 'chat';
  }
  if (options?.ideCoding) {
    return 'code';
  }
  if (options?.channelKind === 'collaboration') {
    return 'collab';
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
