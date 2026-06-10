/** One join announcement per channel per browser session (avoids reconnect/HMR spam). */

const JOIN_DEBOUNCE_MS = 60_000;
const lastJoinSentAt = new Map<string, number>();

/** Test-only: clears in-memory debounce state between vitest cases. */
export function resetJoinMessageDebounceForTests(): void {
  lastJoinSentAt.clear();
}

export function joinMessageStorageKey(
  channel: string,
  username: string,
  userId?: string
): string {
  const actor = (userId?.trim() || username).trim();
  return `nj-join-sent:${channel}:${actor}`;
}

/** Returns true if we should send a join line for this channel (DMs never announce). */
export function shouldSendChannelJoinMessage(
  channel: string,
  username: string,
  userId?: string
): boolean {
  const ch = channel.trim();
  if (!ch || ch.startsWith('dm-')) {
    return false;
  }
  try {
    const now = Date.now();
    const last = lastJoinSentAt.get(ch) ?? 0;
    if (now - last < JOIN_DEBOUNCE_MS) {
      return false;
    }

    const key = joinMessageStorageKey(ch, username, userId);
    if (sessionStorage.getItem(key)) {
      return false;
    }
    sessionStorage.setItem(key, '1');
    lastJoinSentAt.set(ch, now);
    return true;
  } catch {
    return false;
  }
}

export function isHumanJoinAnnouncement(message: {
  type?: string;
  content?: string;
}): boolean {
  return (
    message.type === 'system_info' &&
    typeof message.content === 'string' &&
    message.content.includes('has joined the chat')
  );
}
