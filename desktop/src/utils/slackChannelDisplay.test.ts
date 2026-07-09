import { describe, expect, it } from 'vitest';
import type { Channel } from '../types/protocol';
import {
  channelSidebarLabel,
  isSlackHubChannelName,
  roomChannelSidebarLabel,
  showSlackHubChannelIdInHeader,
  slackChannelDisplayName,
} from './slackChannelDisplay';

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

  it('hides inbox hub id in chat header', () => {
    expect(showSlackHubChannelIdInHeader('slack:inbox:U1:U2')).toBe(false);
    expect(showSlackHubChannelIdInHeader('slack:C01234567')).toBe(true);
  });

  it('labels room channels from display_name', () => {
    const ch: Channel = {
      id: 'room-1',
      name: 'room-abc12345-general',
      display_name: 'Design review',
      description: 'Design review',
      type: 'room',
      room_id: 'abc12345-uuid',
      created: '',
      agents: [],
    };
    expect(roomChannelSidebarLabel(ch)).toBe('Design review');
    expect(channelSidebarLabel(ch)).toBe('Design review');
  });

  it('labels room channels from room_id when unnamed', () => {
    const ch: Channel = {
      id: 'room-1',
      name: 'room-abc12345-general',
      description: 'Room chat',
      type: 'room',
      room_id: 'abc12345-uuid',
      created: '',
      agents: [],
    };
    expect(roomChannelSidebarLabel(ch)).toBe('Room · abc12345');
    expect(channelSidebarLabel(ch)).toBe('Room · abc12345');
  });
});
