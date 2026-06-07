import { describe, expect, it } from 'vitest';
import type { Channel } from '../types/protocol';
import { isSlackHubChannelName, slackChannelDisplayName } from './slackChannelDisplay';

describe('slackChannelDisplay', () => {
  it('labels mirror channels from display_name', () => {
    const ch: Channel = {
      id: '1',
      name: 'slack:C01234567',
      display_name: '#engineering',
      description: 'Slack: #engineering',
      type: 'custom',
      created_at: '',
      updated_at: '',
    };
    expect(slackChannelDisplayName(ch)).toBe('#engineering');
  });

  it('labels inbox channels from description', () => {
    const ch: Channel = {
      id: '2',
      name: 'slack:inbox:U123',
      description: 'Slack inbox — Camron',
      type: 'custom',
      created_at: '',
      updated_at: '',
    };
    expect(slackChannelDisplayName(ch)).toBe('Inbox · Camron');
  });

  it('labels peer inbox channels with peer display name', () => {
    const ch: Channel = {
      id: '3',
      name: 'slack:inbox:U123:U456',
      display_name: 'Demo User',
      description: 'Slack DM — Demo User',
      type: 'custom',
      created_at: '',
      updated_at: '',
    };
    expect(slackChannelDisplayName(ch)).toBe('Demo User');
  });

  it('detects slack hub channel names', () => {
    expect(isSlackHubChannelName('slack:C1')).toBe(true);
    expect(isSlackHubChannelName('slack:inbox:U1')).toBe(true);
    expect(isSlackHubChannelName('general')).toBe(false);
  });
});
