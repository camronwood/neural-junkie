import { describe, expect, it } from 'vitest';
import type { Message } from '../types/protocol';
import {
  isSlackInboundMessage,
  shouldNotifySlackInbound,
  slackInboundPreview,
  slackInboundSenderLabel,
} from './slackNotification';

function slackMsg(overrides: Partial<Message> = {}): Message {
  return {
    id: 'm1',
    type: 'chat',
    channel: 'slack:inbox:U1:U2',
    content: '[DM from Demo User]\nhello there',
    timestamp: new Date().toISOString(),
    from: { id: 'slack:U2', name: 'Demo User', type: 'general' },
    metadata: { source: 'slack_human_dm', slack_user_display_name: 'Demo User' },
    ...overrides,
  };
}

describe('slackNotification', () => {
  it('detects slack inbound messages', () => {
    expect(isSlackInboundMessage(slackMsg())).toBe(true);
    expect(
      isSlackInboundMessage(
        slackMsg({ from: { id: 'user-1', name: 'Camron', type: 'human' }, metadata: {} })
      )
    ).toBe(false);
  });

  it('builds preview and sender labels', () => {
    expect(slackInboundPreview(slackMsg())).toBe('hello there');
    expect(slackInboundSenderLabel(slackMsg())).toBe('Demo User');
  });

  it('skips non-chat slack noise', () => {
    expect(shouldNotifySlackInbound(slackMsg({ type: 'agent_status' }))).toBe(false);
    expect(shouldNotifySlackInbound(slackMsg())).toBe(true);
  });
});
