import { create } from 'zustand';
import type { Message, AgentInfo, ThinkingAgent, AgentType, ThreadMetadata, CachedAgentInfo, Channel } from '../types/protocol';
import {
  channelTimelineAllowsEmptyContent,
  getReasoningText,
  isReasoningStreamDelta,
  REASONING_APPEND_METADATA_KEY,
  REASONING_TEXT_METADATA_KEY,
} from '../types/protocol';
import type { ConnectionStatus } from '../hooks/useWebSocket';
import { ChatAPI } from '../api/chatAPI';
import { getHubBaseURL, normalizeLegacyHubServerAddr } from '../config/hubUrl';
import {
  capStreamContent,
  MAX_UI_CHANNEL_MESSAGES,
  MAX_UI_THREAD_MESSAGES,
  trimMessagesToMax,
} from '../config/messageLimits';

export interface ChatState {
  // Connection
  connectionStatus: ConnectionStatus;
  serverAddr: string;
  channel: string;
  username: string;
  
  // Messages
  messages: Message[];
  
  // Agents
  agents: AgentInfo[];

  // Channels
  channels: Channel[];
  channelMessages: Map<string, Message[]>;
  unreadChannels: Set<string>;
  
  // Threads
  openThreadId: string | null;
  threadMessages: Map<string, Message[]>; // Thread ID -> messages
  threadMetadata: Map<string, ThreadMetadata>; // Thread ID -> metadata
  
  // UI State
  isTyping: boolean;
  errorMessage: string | null;
  channelThinkingAgents: Map<string, Map<string, ThinkingAgent>>; // Channel -> (Agent ID -> Agent info)
  
  // My Agents Panel
  myAgentsPanelOpen: boolean;
  myAgents: CachedAgentInfo[];
  
  // Removed Agents Panel
  removedAgentsPanelOpen: boolean;
  removedAgents: AgentInfo[];
  
  // Loading Agents
  loadingAgents: Set<string>; // Set of agent names currently loading
  
  // Streaming messages (in-flight token-by-token responses)
  streamingMessages: Record<string, Message>;
  
  // Actions
  setConnectionStatus: (status: ConnectionStatus) => void;
  setServerAddr: (addr: string) => void;
  setChannel: (channel: string) => void;
  setUsername: (username: string) => void;
  addMessage: (message: Message) => void;
  setMessages: (messages: Message[]) => void;
  setAgents: (agents: AgentInfo[]) => void;
  setIsTyping: (isTyping: boolean) => void;
  setErrorMessage: (message: string | null) => void;
  addThinkingAgent: (channelName: string, agentId: string, agentName: string, agentType: AgentType) => void;
  removeThinkingAgent: (channelName: string, agentId: string) => void;
  clearThinkingAgents: (channelName?: string) => void;
  cleanupStaleThinking: (channelName: string, messages: Message[]) => void;
  updateAgentStatus: (agentId: string, updates: Partial<AgentInfo>) => void;
  
  // Channel actions
  setChannels: (channels: Channel[]) => void;
  switchChannel: (channelName: string) => void;
  markChannelUnread: (channelName: string) => void;
  clearChannelUnread: (channelName: string) => void;
  addMessageToCache: (channelName: string, message: Message) => void;
  /** Replace cached messages for a channel (e.g. after server-side history prune). */
  replaceChannelMessagesCache: (channelName: string, messages: Message[]) => void;

  // Thread actions
  openThread: (threadId: string) => void;
  closeThread: () => void;
  addThreadMessage: (message: Message) => void;
  setThreadMessages: (threadId: string, messages: Message[]) => void;
  updateThreadMetadata: (threadId: string, metadata: ThreadMetadata) => void;
  
  // My Agents Panel actions
  setMyAgentsPanelOpen: (open: boolean) => void;
  setMyAgents: (agents: CachedAgentInfo[]) => void;
  loadMyAgent: (agent: CachedAgentInfo) => void;
  
  // Loading Agents actions
  addLoadingAgent: (agentName: string) => void;
  removeLoadingAgent: (agentName: string) => void;
  clearLoadingAgents: () => void;
  
  // Removed Agents Panel actions
  setRemovedAgentsPanelOpen: (open: boolean) => void;
  setRemovedAgents: (agents: AgentInfo[]) => void;
  removeAgentFromConversation: (agentId: string) => void;
  recallAgent: (agentId: string) => void;
  
  // Streaming actions
  appendStreamDelta: (msg: Message) => void;
  finalizeStream: (streamId: string) => void;
  
  // Provider switching actions
  switchAgentProvider: (agentId: string, provider: string, model: string) => Promise<void>;
  switchAllAgentProviders: (provider: string, model: string) => Promise<void>;
  
  // Logout action
  logout: () => void;
  
  reset: () => void;
}

/** Cleared on logout/reset — see create() for stream coalescing state */
function clearStreamCoalesceState(
  streamPending: Map<string, Message>,
  streamFlushRaf: { current: number }
) {
  if (streamFlushRaf.current !== 0) {
    if (typeof cancelAnimationFrame !== 'undefined') {
      cancelAnimationFrame(streamFlushRaf.current);
    } else {
      clearTimeout(streamFlushRaf.current);
    }
    streamFlushRaf.current = 0;
  }
  streamPending.clear();
}

const initialState = {
  connectionStatus: 'disconnected' as ConnectionStatus,
  serverAddr: getHubBaseURL(),
  channel: localStorage.getItem('last-channel') || 'general',
  username: '',
  messages: [],
  agents: [],
  channels: [] as Channel[],
  channelMessages: new Map<string, Message[]>(),
  unreadChannels: new Set<string>(),
  openThreadId: null,
  threadMessages: new Map<string, Message[]>(),
  threadMetadata: new Map<string, ThreadMetadata>(),
  isTyping: false,
  errorMessage: null,
  channelThinkingAgents: new Map<string, Map<string, ThinkingAgent>>(),
  myAgentsPanelOpen: false,
  myAgents: [],
  removedAgentsPanelOpen: false,
  removedAgents: [],
  loadingAgents: new Set<string>(),
  streamingMessages: {} as Record<string, Message>,
};

export const useChatStore = create<ChatState>((set, get) => {
  /** Deltas merged here and flushed once per animation frame */
  const streamPending = new Map<string, Message>();
  const streamFlushRaf = { current: 0 };

  const flushPendingStreamDeltas = () => {
    streamFlushRaf.current = 0;
    if (streamPending.size === 0) return;
    const batch = new Map(streamPending);
    streamPending.clear();
    set((state) => {
      const next = { ...state.streamingMessages };
      for (const [id, msg] of batch) {
        next[id] = { ...msg, content: capStreamContent(msg.content ?? '') };
      }
      return { streamingMessages: next };
    });
  };

  const scheduleStreamFlush = () => {
    if (streamFlushRaf.current !== 0) return;
    const schedule =
      typeof requestAnimationFrame !== 'undefined'
        ? requestAnimationFrame
        : (cb: FrameRequestCallback) =>
            window.setTimeout(() => cb(performance.now()), 16) as unknown as typeof requestAnimationFrame;
    streamFlushRaf.current = schedule(() => {
      flushPendingStreamDeltas();
    }) as unknown as number;
  };

  return {
  ...initialState,
  
  setConnectionStatus: (status) => set({ connectionStatus: status }),
  
  setServerAddr: (addr) => set({ serverAddr: normalizeLegacyHubServerAddr(addr) }),
  
  setChannel: (channel) => set({ channel }),
  
  setUsername: (username) => set({ username }),
  
  addMessage: (message) =>
    set((state) => {
      // Skip empty messages (some CLI agents send blank status messages)
      if (!message.content?.trim() && !channelTimelineAllowsEmptyContent(message.type)) {
        return state;
      }

      const existingIdx = state.messages.findIndex(m => m.id === message.id);
      if (existingIdx !== -1) {
        // A message with this ID already exists (e.g. promoted from streaming).
        // Replace it with the authoritative final version which carries full metadata.
        const updated = [...state.messages];
        updated[existingIdx] = message;
        return {
          messages: trimMessagesToMax(updated, MAX_UI_CHANNEL_MESSAGES),
          isTyping: false,
        };
      }

      return {
        messages: trimMessagesToMax([...state.messages, message], MAX_UI_CHANNEL_MESSAGES),
        isTyping: false,
      };
    }),
  
  setMessages: (messages) =>
    set({
      messages: trimMessagesToMax(
        messages.filter(
          (m) => !!m.content?.trim() || channelTimelineAllowsEmptyContent(m.type)
        ),
        MAX_UI_CHANNEL_MESSAGES
      ),
    }),
  
  setAgents: (agents) => set({ agents }),
  
  setIsTyping: (isTyping) => set({ isTyping }),
  
  setErrorMessage: (message) => set({ errorMessage: message }),
  
  addThinkingAgent: (channelName, agentId, agentName, agentType) =>
    set((state) => {
      const innerExisting = state.channelThinkingAgents.get(channelName);
      const prev = innerExisting?.get(agentId);
      if (prev && prev.name === agentName && prev.type === agentType) {
        return state;
      }
      const outer = new Map(state.channelThinkingAgents);
      const inner = new Map(outer.get(channelName) || []);
      inner.set(agentId, { id: agentId, name: agentName, type: agentType });
      outer.set(channelName, inner);
      return { channelThinkingAgents: outer };
    }),
  
  removeThinkingAgent: (channelName, agentId) =>
    set((state) => {
      const inner = state.channelThinkingAgents.get(channelName);
      if (!inner || !inner.has(agentId)) return state;
      const outer = new Map(state.channelThinkingAgents);
      const newInner = new Map(inner);
      newInner.delete(agentId);
      if (newInner.size === 0) {
        outer.delete(channelName);
      } else {
        outer.set(channelName, newInner);
      }
      return { channelThinkingAgents: outer };
    }),
  
  clearThinkingAgents: (channelName) =>
    set((state) => {
      if (channelName) {
        if (!state.channelThinkingAgents.has(channelName)) return state;
        const outer = new Map(state.channelThinkingAgents);
        outer.delete(channelName);
        return { channelThinkingAgents: outer };
      }
      return { channelThinkingAgents: new Map<string, Map<string, ThinkingAgent>>() };
    }),

  cleanupStaleThinking: (channelName, messages) =>
    set((state) => {
      const inner = state.channelThinkingAgents.get(channelName);
      if (!inner || inner.size === 0) return state;
      const respondedAgentIds = new Set(
        messages
          .filter(m => m.type === 'chat' || m.type === 'answer')
          .map(m => m.from?.id)
          .filter(Boolean)
      );
      let changed = false;
      const newInner = new Map(inner);
      for (const agentId of newInner.keys()) {
        if (respondedAgentIds.has(agentId)) {
          newInner.delete(agentId);
          changed = true;
        }
      }
      if (!changed) return state;
      const outer = new Map(state.channelThinkingAgents);
      if (newInner.size === 0) {
        outer.delete(channelName);
      } else {
        outer.set(channelName, newInner);
      }
      return { channelThinkingAgents: outer };
    }),
  
  updateAgentStatus: (agentId, updates) =>
    set((state) => {
      const agent = state.agents.find((a) => a.id === agentId);
      if (!agent) return state;
      const changed = (Object.keys(updates) as (keyof AgentInfo)[]).some(
        (k) => agent[k] !== updates[k]
      );
      if (!changed) return state;
      const updatedAgents = state.agents.map((a) =>
        a.id === agentId ? { ...a, ...updates } : a
      );
      return { agents: updatedAgents };
    }),
  
  // Channel actions
  setChannels: (channels) => set({ channels }),

  switchChannel: (channelName) => {
    if (streamFlushRaf.current !== 0) {
      if (typeof cancelAnimationFrame !== 'undefined') {
        cancelAnimationFrame(streamFlushRaf.current);
      } else {
        clearTimeout(streamFlushRaf.current);
      }
      streamFlushRaf.current = 0;
    }
    streamPending.clear();
    set((state) => {
      // Cache current channel's messages before switching
      const newCache = new Map(state.channelMessages);
      newCache.set(state.channel, trimMessagesToMax(state.messages, MAX_UI_CHANNEL_MESSAGES));

      // Restore cached messages for the target channel (or empty)
      const cachedMessages = trimMessagesToMax(
        newCache.get(channelName) || [],
        MAX_UI_CHANNEL_MESSAGES
      );

      // Clear unread for the channel we're switching to
      const newUnread = new Set(state.unreadChannels);
      newUnread.delete(channelName);

      return {
        channel: channelName,
        messages: cachedMessages,
        channelMessages: newCache,
        unreadChannels: newUnread,
        openThreadId: null,
        streamingMessages: {},
      };
    });
  },

  markChannelUnread: (channelName) =>
    set((state) => {
      if (channelName === state.channel) return state;
      const newUnread = new Set(state.unreadChannels);
      newUnread.add(channelName);
      return { unreadChannels: newUnread };
    }),

  clearChannelUnread: (channelName) =>
    set((state) => {
      const newUnread = new Set(state.unreadChannels);
      newUnread.delete(channelName);
      return { unreadChannels: newUnread };
    }),

  addMessageToCache: (channelName, message) =>
    set((state) => {
      const newCache = new Map(state.channelMessages);
      const cached = newCache.get(channelName) || [];
      if (cached.some(m => m.id === message.id)) return state;
      newCache.set(
        channelName,
        trimMessagesToMax([...cached, message], MAX_UI_CHANNEL_MESSAGES)
      );
      return { channelMessages: newCache };
    }),

  replaceChannelMessagesCache: (channelName, messages) =>
    set((state) => {
      const newCache = new Map(state.channelMessages);
      newCache.set(channelName, trimMessagesToMax(messages, MAX_UI_CHANNEL_MESSAGES));
      return { channelMessages: newCache };
    }),

  // Thread actions
  openThread: (threadId) => set({ openThreadId: threadId }),
  
  closeThread: () => set({ openThreadId: null }),
  
  addThreadMessage: (message) =>
    set((state) => {
      const threadId = message.thread_id || '';
      const currentMessages = state.threadMessages.get(threadId) || [];
      
      // Prevent duplicate messages
      const isDuplicate = currentMessages.some(m => m.id === message.id);
      if (isDuplicate) {
        console.log('[ChatStore] Skipping duplicate thread message:', message.id);
        return state; // Return unchanged state
      }
      
      const newThreadMessages = new Map(state.threadMessages);
      newThreadMessages.set(
        threadId,
        trimMessagesToMax([...currentMessages, message], MAX_UI_THREAD_MESSAGES)
      );
      return { threadMessages: newThreadMessages };
    }),
  
  setThreadMessages: (threadId, messages) =>
    set((state) => {
      const newThreadMessages = new Map(state.threadMessages);
      newThreadMessages.set(threadId, trimMessagesToMax(messages, MAX_UI_THREAD_MESSAGES));
      return { threadMessages: newThreadMessages };
    }),
  
  updateThreadMetadata: (threadId, metadata) =>
    set((state) => {
      const newThreadMetadata = new Map(state.threadMetadata);
      newThreadMetadata.set(threadId, metadata);
      return { threadMetadata: newThreadMetadata };
    }),
  
  // My Agents Panel actions
  setMyAgentsPanelOpen: (open) => set({ myAgentsPanelOpen: open }),
  
  setMyAgents: (agents) => set({ myAgents: agents }),
  
  loadMyAgent: (agent) => {
    // This will be implemented to send the appropriate command
    // For now, we'll just log it - the actual implementation will be in the component
    console.log('Loading my agent:', agent);
  },
  
  // Loading Agents actions
  addLoadingAgent: (agentName) => {
    set((state) => {
      if (state.loadingAgents.has(agentName)) return state;
      return {
        loadingAgents: new Set([...state.loadingAgents, agentName]),
      };
    });
  },
  
  removeLoadingAgent: (agentName) => {
    set((state) => {
      if (!state.loadingAgents.has(agentName)) return state;
      const newLoadingAgents = new Set(state.loadingAgents);
      newLoadingAgents.delete(agentName);
      return { loadingAgents: newLoadingAgents };
    });
  },
  
  clearLoadingAgents: () => set({ loadingAgents: new Set<string>() }),
  
  // Removed Agents Panel actions
  setRemovedAgentsPanelOpen: (open) => set({ removedAgentsPanelOpen: open }),
  
  setRemovedAgents: (agents) => set({ removedAgents: agents }),
  
  removeAgentFromConversation: (agentId) => {
    // This will be implemented to send the remove command
    // For now, we'll just log it - the actual implementation will be in the component
    console.log('Removing agent from conversation:', agentId);
  },
  
  recallAgent: (agentId) => {
    // This will be implemented to send the recall command
    // For now, we'll just log it - the actual implementation will be in the component
    console.log('Recalling agent:', agentId);
  },
  
  // Streaming actions — deltas coalesced per rAF to avoid one React commit per token
  appendStreamDelta: (msg) => {
    const id = msg.id;
    const meta = msg.metadata ?? {};
    const isReasoning = isReasoningStreamDelta(meta);
    const reasoningChunk =
      typeof meta[REASONING_APPEND_METADATA_KEY] === 'string'
        ? (meta[REASONING_APPEND_METADATA_KEY] as string)
        : '';

    const mergeDelta = (base: Message): Message => {
      if (isReasoning) {
        const prev = getReasoningText(base.metadata as Record<string, unknown> | undefined);
        return {
          ...base,
          type: 'chat' as Message['type'],
          metadata: {
            ...base.metadata,
            [REASONING_TEXT_METADATA_KEY]: prev + reasoningChunk,
          },
        };
      }
      const chunk = msg.content ?? '';
      return {
        ...base,
        type: 'chat' as Message['type'],
        content: capStreamContent((base.content ?? '') + chunk),
      };
    };

    const curPending = streamPending.get(id);
    const state = get();
    if (curPending) {
      streamPending.set(id, mergeDelta(curPending));
    } else if (state.streamingMessages[id]) {
      streamPending.set(id, mergeDelta(state.streamingMessages[id]));
    } else if (isReasoning) {
      streamPending.set(id, mergeDelta({
        ...msg,
        type: 'chat' as Message['type'],
        content: '',
        metadata: { ...msg.metadata },
      }));
    } else {
      const chunk = msg.content ?? '';
      streamPending.set(id, {
        ...msg,
        type: 'chat' as Message['type'],
        content: capStreamContent(chunk),
        metadata: { ...msg.metadata },
      });
    }
    scheduleStreamFlush();
  },

  finalizeStream: (streamId) => {
    if (streamFlushRaf.current !== 0) {
      if (typeof cancelAnimationFrame !== 'undefined') {
        cancelAnimationFrame(streamFlushRaf.current);
      } else {
        clearTimeout(streamFlushRaf.current);
      }
      streamFlushRaf.current = 0;
    }
    const pendingBatch =
      streamPending.size > 0 ? new Map(streamPending) : null;
    if (pendingBatch) {
      streamPending.clear();
    }
    set((state) => {
      let streamingMessages = state.streamingMessages;
      if (pendingBatch) {
        const next = { ...streamingMessages };
        for (const [id, m] of pendingBatch) {
          next[id] = { ...m, content: capStreamContent(m.content ?? '') };
        }
        streamingMessages = next;
      }
      const streamed = streamingMessages[streamId];
      // Unknown stream id (already finalized, or never started): no-op.
      if (!streamed) {
        return pendingBatch ? { ...state, streamingMessages } : state;
      }
      const { [streamId]: _removed, ...rest } = streamingMessages;
      const reasoning = getReasoningText(streamed.metadata as Record<string, unknown> | undefined);
      if (!streamed.content?.trim() && !reasoning.trim()) {
        return { ...state, streamingMessages: rest };
      }
      const alreadyInMessages = state.messages.some((m) => m.id === streamId);
      if (alreadyInMessages) {
        return { ...state, streamingMessages: rest };
      }
      const finalized: Message = {
        ...streamed,
        type: 'chat' as Message['type'],
        content: capStreamContent(streamed.content ?? ''),
        metadata: reasoning
          ? { ...streamed.metadata, [REASONING_TEXT_METADATA_KEY]: reasoning }
          : streamed.metadata,
      };
      return {
        streamingMessages: rest,
        messages: trimMessagesToMax([...state.messages, finalized], MAX_UI_CHANNEL_MESSAGES),
      };
    });
  },
  
  // Provider switching actions
  switchAgentProvider: async (agentId, provider, model) => {
    const { serverAddr } = get();
    const api = new ChatAPI(serverAddr);
    
    try {
      await api.switchAgentProvider(agentId, provider, model);
      set((state) => ({
        agents: state.agents.map((a) =>
          a.id === agentId ? { ...a, ai_provider: provider, ai_model: model } : a
        ),
      }));
    } catch (error) {
      console.error('Failed to switch agent provider:', error);
      throw error;
    }
  },
  
  switchAllAgentProviders: async (provider, model) => {
    const { serverAddr } = get();
    const api = new ChatAPI(serverAddr);
    
    try {
      await api.switchAllAgentProviders(provider, model);
      set((state) => ({
        agents: state.agents.map((a) => ({ ...a, ai_provider: provider, ai_model: model })),
      }));
    } catch (error) {
      console.error('Failed to switch all agent providers:', error);
      throw error;
    }
  },
  
  logout: () => {
    console.log('[ChatStore] Logging out user');
    clearStreamCoalesceState(streamPending, streamFlushRaf);
    set({ 
      ...initialState, 
      channelThinkingAgents: new Map<string, Map<string, ThinkingAgent>>(),
      threadMessages: new Map<string, Message[]>(),
      threadMetadata: new Map<string, ThreadMetadata>(),
      channelMessages: new Map<string, Message[]>(),
      unreadChannels: new Set<string>(),
      streamingMessages: {},
    });
  },
  
  reset: () => {
    clearStreamCoalesceState(streamPending, streamFlushRaf);
    set({ 
    ...initialState, 
    channelThinkingAgents: new Map<string, Map<string, ThinkingAgent>>(),
    threadMessages: new Map<string, Message[]>(),
    threadMetadata: new Map<string, ThreadMetadata>(),
    channelMessages: new Map<string, Message[]>(),
    unreadChannels: new Set<string>(),
    streamingMessages: {},
  });
  },
};
});

