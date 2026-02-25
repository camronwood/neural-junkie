import { create } from 'zustand';
import type { Message, AgentInfo, ThinkingAgent, AgentType, ThreadMetadata, CachedAgentInfo, Channel } from '../types/protocol';
import type { ConnectionStatus } from '../hooks/useWebSocket';
import { ChatAPI } from '../api/chatAPI';

interface ChatState {
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

const initialState = {
  connectionStatus: 'disconnected' as ConnectionStatus,
  serverAddr: 'localhost:8080',
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

export const useChatStore = create<ChatState>((set, get) => ({
  ...initialState,
  
  setConnectionStatus: (status) => set({ connectionStatus: status }),
  
  setServerAddr: (addr) => set({ serverAddr: addr }),
  
  setChannel: (channel) => set({ channel }),
  
  setUsername: (username) => set({ username }),
  
  addMessage: (message) =>
    set((state) => {
      // Skip empty messages (some CLI agents send blank status messages)
      if (!message.content?.trim() && message.type !== 'agent_join' && message.type !== 'agent_leave' && message.type !== 'system_info') {
        return state;
      }

      // Prevent duplicate messages (can happen with React StrictMode double-mounting)
      const isDuplicate = state.messages.some(m => m.id === message.id);
      if (isDuplicate) {
        console.log('[ChatStore] Skipping duplicate message:', message.id);
        return state;
      }
      
      return {
        messages: [...state.messages, message],
        isTyping: false,
      };
    }),
  
  setMessages: (messages) => set({
    messages: messages.filter(m =>
      !!m.content?.trim() || m.type === 'agent_join' || m.type === 'agent_leave' || m.type === 'system_info'
    ),
  }),
  
  setAgents: (agents) => set({ agents }),
  
  setIsTyping: (isTyping) => set({ isTyping }),
  
  setErrorMessage: (message) => set({ errorMessage: message }),
  
  addThinkingAgent: (channelName, agentId, agentName, agentType) =>
    set((state) => {
      const outer = new Map(state.channelThinkingAgents);
      const inner = new Map(outer.get(channelName) || []);
      inner.set(agentId, { id: agentId, name: agentName, type: agentType });
      outer.set(channelName, inner);
      return { channelThinkingAgents: outer };
    }),
  
  removeThinkingAgent: (channelName, agentId) =>
    set((state) => {
      const outer = new Map(state.channelThinkingAgents);
      const inner = outer.get(channelName);
      if (!inner) return state;
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
      const updatedAgents = state.agents.map((agent) =>
        agent.id === agentId ? { ...agent, ...updates } : agent
      );
      return { agents: updatedAgents };
    }),
  
  // Channel actions
  setChannels: (channels) => set({ channels }),

  switchChannel: (channelName) =>
    set((state) => {
      // Cache current channel's messages before switching
      const newCache = new Map(state.channelMessages);
      newCache.set(state.channel, state.messages);

      // Restore cached messages for the target channel (or empty)
      const cachedMessages = newCache.get(channelName) || [];

      // Clear unread for the channel we're switching to
      const newUnread = new Set(state.unreadChannels);
      newUnread.delete(channelName);

      return {
        channel: channelName,
        messages: cachedMessages,
        channelMessages: newCache,
        unreadChannels: newUnread,
        openThreadId: null,
      };
    }),

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
      newCache.set(channelName, [...cached, message]);
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
      newThreadMessages.set(threadId, [...currentMessages, message]);
      return { threadMessages: newThreadMessages };
    }),
  
  setThreadMessages: (threadId, messages) =>
    set((state) => {
      const newThreadMessages = new Map(state.threadMessages);
      newThreadMessages.set(threadId, messages);
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
    set((state) => ({
      loadingAgents: new Set([...state.loadingAgents, agentName])
    }));
  },
  
  removeLoadingAgent: (agentName) => {
    set((state) => {
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
  
  // Streaming actions
  appendStreamDelta: (msg) =>
    set((state) => {
      const existing = state.streamingMessages[msg.id];
      if (existing) {
        return {
          streamingMessages: {
            ...state.streamingMessages,
            [msg.id]: { ...existing, content: existing.content + msg.content },
          },
        };
      }
      // First delta -- create the streaming message entry
      return {
        streamingMessages: {
          ...state.streamingMessages,
          [msg.id]: { ...msg, type: 'chat' as Message['type'] },
        },
      };
    }),

  finalizeStream: (streamId) =>
    set((state) => {
      const { [streamId]: _removed, ...rest } = state.streamingMessages;
      return { streamingMessages: rest };
    }),
  
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
  
  reset: () => set({ 
    ...initialState, 
    channelThinkingAgents: new Map<string, Map<string, ThinkingAgent>>(),
    threadMessages: new Map<string, Message[]>(),
    threadMetadata: new Map<string, ThreadMetadata>(),
    channelMessages: new Map<string, Message[]>(),
    unreadChannels: new Set<string>(),
    streamingMessages: {},
  }),
}));

