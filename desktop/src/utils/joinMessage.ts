/** One join announcement per channel per browser session (avoids reconnect/HMR spam). */

export function joinMessageStorageKey(channel: string, username: string): string {
  return `nj-join-sent:${channel}:${username}`;
}

/** Returns true if we should send a join line for this channel (DMs never announce). */
export function shouldSendChannelJoinMessage(channel: string, username: string): boolean {
  const ch = channel.trim();
  if (!ch || ch.startsWith('dm-')) {
    return false;
  }
  try {
    const key = joinMessageStorageKey(ch, username);
    if (sessionStorage.getItem(key)) {
      return false;
    }
    sessionStorage.setItem(key, '1');
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
