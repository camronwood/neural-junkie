import type { Message } from '../types/protocol';

/** Max messages kept per channel in the desktop UI (active list + per-channel cache). */
export const MAX_UI_CHANNEL_MESSAGES = 500;

/** Max thread replies kept in memory per thread. */
export const MAX_UI_THREAD_MESSAGES = 500;

/** Safety cap for a single in-flight stream body (characters) to bound WebView memory. */
export const MAX_STREAM_CONTENT_CHARS = 5_000_000;

const STREAM_TRUNCATION_NOTE = '\n\n*[Stream truncated for memory cap]*';

export function trimMessagesToMax(messages: Message[], max: number): Message[] {
  if (messages.length <= max) {
    return messages;
  }
  return messages.slice(-max);
}

/** Truncate streaming markdown/text in place for memory safety. */
export function capStreamContent(content: string): string {
  if (content.length <= MAX_STREAM_CONTENT_CHARS) {
    return content;
  }
  const budget = MAX_STREAM_CONTENT_CHARS - STREAM_TRUNCATION_NOTE.length;
  if (budget < 1) {
    return STREAM_TRUNCATION_NOTE.trim();
  }
  return content.slice(0, budget) + STREAM_TRUNCATION_NOTE;
}
