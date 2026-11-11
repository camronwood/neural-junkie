import { useEffect, useMemo } from 'react';
import type { Channel } from '../types/protocol';
import { useRoomStore } from '../stores/roomStore';

export type ActiveRoomSidebarMeta = {
  id: string;
  joinCode?: string;
  name?: string;
} | null;

/**
 * Room-mode channel filtering, presence polling, and sidebar meta for ChatWindow.
 * Extracted from ChatWindowRoot (extraction loop 1).
 */
export function useChatWindowRoomChrome(channels: Channel[], channel: string) {
  const roomMode = useRoomStore((s) => s.mode);
  const activeRoomId = useRoomStore((s) => s.room?.id ?? null);
  const activeRoomJoinCode = useRoomStore((s) => s.joinCode);
  const activeRoomName = useRoomStore((s) => s.room?.name ?? null);
  const roomPresenceMembers = useRoomStore((s) => s.presenceMembers);
  const refreshRoomPresence = useRoomStore((s) => s.refreshPresence);

  useEffect(() => {
    if (roomMode !== 'guest' || !activeRoomId) return;
    void refreshRoomPresence();
    const id = window.setInterval(() => {
      void refreshRoomPresence();
    }, 3000);
    return () => window.clearInterval(id);
  }, [roomMode, activeRoomId, refreshRoomPresence]);

  const visibleChannels = useMemo(() => {
    if (roomMode === 'guest' && activeRoomId) {
      return channels.filter((c) => c.type === 'room' && c.room_id === activeRoomId);
    }
    return channels;
  }, [channels, roomMode, activeRoomId]);

  const activeRoomSidebarMeta: ActiveRoomSidebarMeta = useMemo(() => {
    if (!activeRoomId) return null;
    return {
      id: activeRoomId,
      joinCode: activeRoomJoinCode || undefined,
      name: activeRoomName || undefined,
    };
  }, [activeRoomId, activeRoomJoinCode, activeRoomName]);

  const currentChannelType = useMemo(() => {
    return channels.find((c) => c.name === channel)?.type ?? null;
  }, [channels, channel]);

  const connectedRoomMembers = useMemo(() => {
    if (currentChannelType !== 'room') return [];
    return (roomPresenceMembers ?? []).filter((m: { connected?: boolean }) => m?.connected === true);
  }, [currentChannelType, roomPresenceMembers]);

  return {
    roomMode,
    activeRoomId,
    activeRoomJoinCode,
    activeRoomName,
    roomPresenceMembers,
    refreshRoomPresence,
    visibleChannels,
    activeRoomSidebarMeta,
    currentChannelType,
    connectedRoomMembers,
  };
}
