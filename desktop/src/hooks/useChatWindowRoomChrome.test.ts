import { describe, expect, it, vi, beforeEach } from 'vitest';
import { renderHook } from '@testing-library/react';
import { useChatWindowRoomChrome } from './useChatWindowRoomChrome';
import type { Channel } from '../types/protocol';

const refreshPresence = vi.fn();

vi.mock('../stores/roomStore', () => ({
  useRoomStore: (sel: (s: Record<string, unknown>) => unknown) =>
    sel({
      mode: 'host',
      room: null,
      joinCode: null,
      presenceMembers: [],
      refreshPresence,
    }),
}));

describe('useChatWindowRoomChrome', () => {
  beforeEach(() => {
    refreshPresence.mockClear();
  });

  it('returns all channels when not a guest in a room', () => {
    const channels: Channel[] = [
      { name: 'general', type: 'public' } as Channel,
      { name: 'room-1', type: 'room', room_id: 'r1' } as Channel,
    ];
    const { result } = renderHook(() => useChatWindowRoomChrome(channels, 'general'));
    expect(result.current.visibleChannels).toHaveLength(2);
    expect(result.current.activeRoomSidebarMeta).toBeNull();
    expect(result.current.connectedRoomMembers).toEqual([]);
  });
});
