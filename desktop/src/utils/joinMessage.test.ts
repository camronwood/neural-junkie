import { describe, expect, it, beforeEach } from 'vitest';
import {
  isHumanJoinAnnouncement,
  joinMessageStorageKey,
  resetJoinMessageDebounceForTests,
  shouldSendChannelJoinMessage,
} from './joinMessage';

describe('joinMessage', () => {
  beforeEach(() => {
    sessionStorage.clear();
    resetJoinMessageDebounceForTests();
  });

  it('skips DM channels', () => {
    expect(shouldSendChannelJoinMessage('dm-camron-assistant', 'Camron')).toBe(false);
    expect(sessionStorage.getItem(joinMessageStorageKey('dm-camron-assistant', 'Camron'))).toBeNull();
  });

  it('allows once per session for public channels', () => {
    expect(shouldSendChannelJoinMessage('general', 'Camron')).toBe(true);
    expect(shouldSendChannelJoinMessage('general', 'Camron')).toBe(false);
  });

  it('prefers stable user id for storage key', () => {
    expect(joinMessageStorageKey('general', 'Camron', 'user-42')).toBe(
      'nj-join-sent:general:user-42'
    );
  });

  it('debounces rapid join attempts within 60s', () => {
    expect(shouldSendChannelJoinMessage('general', 'Camron')).toBe(true);
    sessionStorage.removeItem(joinMessageStorageKey('general', 'Camron'));
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
