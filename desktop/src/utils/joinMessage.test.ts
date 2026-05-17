import { describe, expect, it, beforeEach } from 'vitest';
import {
  isHumanJoinAnnouncement,
  joinMessageStorageKey,
  shouldSendChannelJoinMessage,
} from './joinMessage';

describe('joinMessage', () => {
  beforeEach(() => {
    sessionStorage.clear();
  });

  it('skips DM channels', () => {
    expect(shouldSendChannelJoinMessage('dm-camron-assistant', 'Camron')).toBe(false);
    expect(sessionStorage.getItem(joinMessageStorageKey('dm-camron-assistant', 'Camron'))).toBeNull();
  });

  it('allows once per session for public channels', () => {
    expect(shouldSendChannelJoinMessage('general', 'Camron')).toBe(true);
    expect(shouldSendChannelJoinMessage('general', 'Camron')).toBe(false);
  });

  it('detects join announcements', () => {
    expect(
      isHumanJoinAnnouncement({
        type: 'system_info',
        content: 'Camron has joined the chat',
      })
    ).toBe(true);
  });
});
