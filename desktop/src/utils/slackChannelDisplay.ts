import type { Channel } from '../types/protocol';

/** Parse "Slack: #channel-name" from hub channel description. */
export function slackLabelFromDescription(description?: string): string | null {
  if (!description) return null;
  const m = description.trim().match(/^Slack:\s*(#\S+)/i);
  return m ? m[1] : null;
}

/** Human label for a hub channel row (Slack mirrors show #name, not slack:C…). */
export function slackChannelDisplayName(ch: Channel): string {
  const dn = ch.display_name?.trim();
  if (dn) return dn;
  const fromDesc = slackLabelFromDescription(ch.description);
  if (fromDesc) return fromDesc;
  if (ch.name.startsWith('slack:')) {
    return ch.name;
  }
  return ch.name;
}

export function isSlackMirrorChannelName(name: string): boolean {
  return name.startsWith('slack:');
}

/** Sidebar / header label for any channel type. */
export function channelSidebarLabel(ch: Channel, collaborationLabel?: (c: Channel) => string): string {
  if (isSlackMirrorChannelName(ch.name)) {
    return slackChannelDisplayName(ch);
  }
  if (ch.type === 'collaboration' && collaborationLabel) {
    return collaborationLabel(ch);
  }
  return ch.name;
}
