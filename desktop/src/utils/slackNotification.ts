import type { Channel, Message } from '../types/protocol';
import { channelSidebarLabel, isSlackHubChannelName } from './slackChannelDisplay';

const INBOUND_SOURCES = new Set(['slack', 'slack_inbox', 'slack_human_dm']);

/** True for hub messages ingested from Slack (not NJ user/agent outbound). */
export function isSlackInboundMessage(message: Message): boolean {
  if (!message.channel?.startsWith('slack:')) return false;
  if (message.from?.id?.startsWith('slack:')) return true;
  const src = message.metadata?.source;
  return typeof src === 'string' && INBOUND_SOURCES.has(src);
}

function stripSlackInboundHeader(content: string): string {
  return content
    .replace(/^\[DM from[^\]]+\]\s*\n?/i, '')
    .replace(/^\[Forwarded from[^\]]+\]\s*\n?/i, '')
    .trim();
}

export function slackInboundPreview(message: Message, maxLen = 120): string {
  const text = stripSlackInboundHeader(message.content || '');
  if (!text) return 'New message';
  if (text.length <= maxLen) return text;
  return `${text.slice(0, maxLen - 1)}…`;
}

export function slackInboundSenderLabel(message: Message): string {
  const meta = message.metadata?.slack_user_display_name;
  if (typeof meta === 'string' && meta.trim()) {
    return meta.trim();
  }
  const name = message.from?.name?.trim();
  if (name) {
    return name.replace(/\s*\(@[^)]+\)\s*$/, '').trim() || name;
  }
  return 'Slack';
}

export function slackChannelLabel(channels: Channel[], channelName: string): string {
  const ch = channels.find((c) => c.name === channelName);
  if (ch) return channelSidebarLabel(ch);
  if (isSlackHubChannelName(channelName)) return 'Slack Inbox';
  if (channelName.startsWith('slack:')) return 'Slack';
  return channelName;
}

export function shouldNotifySlackInbound(message: Message): boolean {
  if (!isSlackInboundMessage(message)) return false;
  if (message.type === 'agent_status' || message.type === 'agent_join' || message.type === 'agent_leave') {
    return false;
  }
  if (message.type !== 'chat' && message.type !== 'question' && message.type !== 'answer') {
    return false;
  }
  return stripSlackInboundHeader(message.content || '').length > 0 || !!message.metadata?.slack_human_dm;
}
