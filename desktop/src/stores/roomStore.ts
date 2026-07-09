import { createWithEqualityFn as create } from 'zustand/traditional';
import { ChatAPI } from '../api/chatAPI';
import {
  clearHubConnectionOverride,
  getHubSessionToken,
  setHubConnectionOverride,
  setHubSessionToken,
} from '../config/hubUrl';
import { useChatStore } from './chatStore';

function notifyRoomChannelsUpdated() {
  if (typeof window !== 'undefined') {
    window.dispatchEvent(new Event('nj-room-channels-updated'));
  }
}

export type RoomMode = 'host' | 'guest' | null;

export interface RoomSessionSnapshot {
  mode: RoomMode;
  roomId: string;
  joinCode: string;
  hubUrl: string;
  hubToken: string;
  roomChannel: string;
}

interface RoomState {
  mode: RoomMode;
  room: any | null;
  roomChannel: string | null;
  joinCode: string;
  hostHubUrlInput: string;
  joining: boolean;
  creating: boolean;
  error: string | null;
  presenceMembers: any[];
  presenceLoading: boolean;
  previous: {
    serverAddr: string;
    channel: string;
    sessionToken: string | null;
  } | null;

  setHostHubUrlInput: (v: string) => void;
  clearError: () => void;

  createRoom: (params?: { name?: string; ttlHours?: number; maxMembers?: number }) => Promise<void>;
  joinRoom: (params: { hostHubUrl: string; joinCode: string; username: string }) => Promise<void>;
  refreshPresence: () => Promise<void>;
  leaveRoom: () => Promise<void>;
  endRoom: () => Promise<void>;
}

export const useRoomStore = create<RoomState>((set, get) => ({
  mode: null,
  room: null,
  roomChannel: null,
  joinCode: '',
  hostHubUrlInput: '',
  joining: false,
  creating: false,
  error: null,
  presenceMembers: [],
  presenceLoading: false,
  previous: null,

  setHostHubUrlInput: (v) => set({ hostHubUrlInput: v }),
  clearError: () => set({ error: null }),

  createRoom: async (params) => {
    set({ creating: true, error: null });
    try {
      const api = new ChatAPI();
      const resp = await api.createRoom({
        name: params?.name ?? '',
        ttl_hours: params?.ttlHours ?? 0,
        max_members: params?.maxMembers ?? 0,
      });
      const room = resp.room;
      const code = room?.join_code ?? '';
      const channel = resp.channel?.name ?? null;
      set({
        mode: 'host',
        room,
        joinCode: code,
        roomChannel: channel,
        creating: false,
      });
      if (channel) {
        useChatStore.getState().setChannel(channel);
      }
      notifyRoomChannelsUpdated();
    } catch (e: any) {
      set({ error: e?.message ?? String(e), creating: false });
    }
  },

  joinRoom: async ({ hostHubUrl, joinCode, username }) => {
    set({ joining: true, error: null });
    try {
      const prev = useChatStore.getState();
      set({
        previous: {
          serverAddr: prev.serverAddr,
          channel: prev.channel,
          sessionToken: getHubSessionToken(),
        },
      });

      const api = new ChatAPI();
      const resp = await api.joinRoom(hostHubUrl, joinCode, username);

      const hubUrl = resp.hub_url;
      const hubToken = resp.hub_token ?? '';
      const roomChannel = resp.room_channel;
      const room = resp.room;

      setHubConnectionOverride(hubUrl, hubToken);
      setHubSessionToken(resp.session?.token ?? null);

      useChatStore.getState().setServerAddr(hubUrl);
      useChatStore.getState().setChannel(roomChannel);
      if (username?.trim()) {
        useChatStore.getState().setUsername(username.trim());
      }

      try {
        const hostApi = new ChatAPI(hubUrl);
        const channelList = await hostApi.fetchChannels();
        useChatStore.getState().setChannels(channelList);
      } catch {
        // Sidebar refresh listener will retry.
      }

      set({
        mode: 'guest',
        room,
        roomChannel,
        joinCode: room?.join_code ?? joinCode,
        presenceMembers: room?.members ?? [],
        joining: false,
      });
      notifyRoomChannelsUpdated();
    } catch (e: any) {
      set({ error: e?.message ?? String(e), joining: false });
    }
  },

  refreshPresence: async () => {
    const st = get();
    const roomId = st.room?.id;
    if (!roomId) return;
    set({ presenceLoading: true });
    try {
      const api = new ChatAPI();
      const resp = await api.getRoomPresence(roomId);
      set({ presenceMembers: resp.members ?? [], presenceLoading: false });
    } catch (e: any) {
      set({ error: e?.message ?? String(e), presenceLoading: false });
    }
  },

  leaveRoom: async () => {
    const st = get();
    if (!st.room?.id) return;
    set({ error: null });
    try {
      const api = new ChatAPI();
      await api.leaveRoom(st.room.id);
    } catch (e: any) {
      // Still attempt restore on leave errors (host might have ended room).
      set({ error: e?.message ?? String(e) });
    } finally {
      const prev = get().previous;
      clearHubConnectionOverride();
      setHubSessionToken(prev?.sessionToken ?? null);
      if (prev) {
        useChatStore.getState().setServerAddr(prev.serverAddr);
        useChatStore.getState().setChannel(prev.channel);
      }
      set({
        mode: null,
        room: null,
        roomChannel: null,
        joinCode: '',
        presenceMembers: [],
        previous: null,
      });
      notifyRoomChannelsUpdated();
    }
  },

  endRoom: async () => {
    const st = get();
    if (!st.room?.id) return;
    set({ error: null });
    try {
      const api = new ChatAPI();
      await api.endRoom(st.room.id);
      set({ mode: null, room: null, roomChannel: null, joinCode: '', presenceMembers: [] });
      notifyRoomChannelsUpdated();
    } catch (e: any) {
      set({ error: e?.message ?? String(e) });
    }
  },
}));

