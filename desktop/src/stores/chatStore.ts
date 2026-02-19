import { create } from 'zustand';
import type { Message, AgentInfo, ThinkingAgent, AgentType, ThreadMetadata, CachedAgentInfo } from '../types/protocol';
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
  
  // Threads
  openThreadId: string | null;
  threadMessages: Map<string, Message[]>; // Thread ID -> messages
  threadMetadata: Map<string, ThreadMetadata>; // Thread ID -> metadata
  
  // UI State
  isTyping: boolean;
  errorMessage: string | null;
  thinkingAgents: Map<string, ThinkingAgent>; // Agent ID -> Agent info
  
  // My Agents Panel
  myAgentsPanelOpen: boolean;
  myAgents: CachedAgentInfo[];
  
  // Removed Agents Panel
  removedAgentsPanelOpen: boolean;
  removedAgents: AgentInfo[];
  
  // Loading Agents
  loadingAgents: Set<string>; // Set of agent names currently loading
  
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
  addThinkingAgent: (agentId: string, agentName: string, agentType: AgentType) => void;
  removeThinkingAgent: (agentId: string) => void;
  clearThinkingAgents: () => void;
  updateAgentStatus: (agentId: string, updates: Partial<AgentInfo>) => void;
  
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
  channel: 'general',
  username: '',
  messages: [],
  agents: [],
  openThreadId: null,
  threadMessages: new Map<string, Message[]>(),
  threadMetadata: new Map<string, ThreadMetadata>(),
  isTyping: false,
  errorMessage: null,
  thinkingAgents: new Map<string, ThinkingAgent>(),
  myAgentsPanelOpen: false,
  myAgents: [],
  removedAgentsPanelOpen: false,
  removedAgents: [],
  loadingAgents: new Set<string>(),
};

export const useChatStore = create<ChatState>((set, get) => ({
  ...initialState,
  
  setConnectionStatus: (status) => set({ connectionStatus: status }),
  
  setServerAddr: (addr) => set({ serverAddr: addr }),
  
  setChannel: (channel) => set({ channel }),
  
  setUsername: (username) => set({ username }),
  
  addMessage: (message) =>
    set((state) => {
      // Prevent duplicate messages (can happen with React StrictMode double-mounting)
      const isDuplicate = state.messages.some(m => m.id === message.id);
      if (isDuplicate) {
        console.log('[ChatStore] Skipping duplicate message:', message.id);
        return state; // Return unchanged state
      }
      
      return {
        messages: [...state.messages, message],
        isTyping: false, // Hide typing indicator when new message arrives
      };
    }),
  
  setMessages: (messages) => set({ messages }),
  
  setAgents: (agents) => set({ agents }),
  
  setIsTyping: (isTyping) => set({ isTyping }),
  
  setErrorMessage: (message) => set({ errorMessage: message }),
  
  addThinkingAgent: (agentId, agentName, agentType) =>
    set((state) => {
      const newThinkingAgents = new Map(state.thinkingAgents);
      newThinkingAgents.set(agentId, { id: agentId, name: agentName, type: agentType });
      return { thinkingAgents: newThinkingAgents };
    }),
  
  removeThinkingAgent: (agentId) =>
    set((state) => {
      const newThinkingAgents = new Map(state.thinkingAgents);
      newThinkingAgents.delete(agentId);
      return { thinkingAgents: newThinkingAgents };
    }),
  
  clearThinkingAgents: () =>
    set({ thinkingAgents: new Map<string, ThinkingAgent>() }),
  
  updateAgentStatus: (agentId, updates) =>
    set((state) => {
      const updatedAgents = state.agents.map((agent) =>
        agent.id === agentId ? { ...agent, ...updates } : agent
      );
      return { agents: updatedAgents };
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
  
  // Provider switching actions
  switchAgentProvider: async (agentId, provider, model) => {
    const { serverAddr } = get();
    const api = new ChatAPI(serverAddr);
    
    try {
      await api.switchAgentProvider(agentId, provider, model);
      console.log(`Switched agent ${agentId} to ${provider} (${model})`);
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
      console.log(`Switched all agents to ${provider} (${model})`);
    } catch (error) {
      console.error('Failed to switch all agent providers:', error);
      throw error;
    }
  },
  
  logout: () => {
    console.log('[ChatStore] Logging out user');
    set({ 
      ...initialState, 
      thinkingAgents: new Map<string, ThinkingAgent>(),
      threadMessages: new Map<string, Message[]>(),
      threadMetadata: new Map<string, ThreadMetadata>(),
    });
  },
  
  reset: () => set({ 
    ...initialState, 
    thinkingAgents: new Map<string, ThinkingAgent>(),
    threadMessages: new Map<string, Message[]>(),
    threadMetadata: new Map<string, ThreadMetadata>(),
  }),
}));

