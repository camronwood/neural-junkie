import type { Channel } from '../types/protocol';

/** Parse "Slack: #channel-name" from hub channel description. */
export function slackLabelFromDescription(description?: string): string | null {
  if (!description) return null;
  const m = description.trim().match(/^Slack:\s*(#\S+)/i);
  return m ? m[1] : null;
}

/** Hub channel created by the Slack bridge (mirror or personal inbox). */
export function isSlackHubChannelName(name: string): boolean {
  return isSlackMirrorChannelName(name) || name.startsWith('slack:inbox:');
}

/** Human label for a hub channel row (Slack mirrors show #name, not slack:C…). */
export function slackChannelDisplayName(ch: Channel): string {
  const dn = ch.display_name?.trim();
  if (dn) return dn;
  const fromDesc = slackLabelFromDescription(ch.description);
  if (fromDesc) return fromDesc;
  if (ch.name.startsWith('slack:inbox:')) {
    const parts = ch.name.split(':');
    // slack:inbox:U_OWNER:U_PEER — one DM per Slack user
    if (parts.length >= 4) {
      if (dn) return dn;
      const fromDesc = (ch.description || '').match(/Slack DM —\s*(.+)/i);
      if (fromDesc?.[1]?.trim()) {
        return fromDesc[1].trim();
      }
      return 'Slack DM';
    }
    const inboxMatch = (ch.description || '').match(/Slack inbox —\s*(.+)/i);
    if (inboxMatch?.[1]?.trim()) {
      return `Inbox · ${inboxMatch[1].trim()}`;
    }
    return 'Slack Inbox';
  }
  if (ch.name.startsWith('slack:')) {
    return ch.name;
  }
  return ch.name;
}

/** Whether the chat header should show the hub channel id under the Slack label. */
export function showSlackHubChannelIdInHeader(name: string): boolean {
  return isSlackMirrorChannelName(name) && !name.startsWith('slack:inbox:');
}

export function isSlackMirrorChannelName(name: string): boolean {
  return name.startsWith('slack:');
}

/** Sidebar / header label for any channel type. */
export function channelSidebarLabel(ch: Channel, collaborationLabel?: (c: Channel) => string): string {
  if (isSlackHubChannelName(ch.name)) {
    return slackChannelDisplayName(ch);
  }
  if (ch.type === 'collaboration' && collaborationLabel) {
    return collaborationLabel(ch);
  }
  return ch.name;
}
