import { createWithEqualityFn as create } from 'zustand/traditional';
import type { Message, AgentInfo, ThinkingAgent, AgentType, ThreadMetadata, CachedAgentInfo, Channel, TurnTelemetryEvent } from '../types/protocol';
import {
  channelTimelineAllowsEmptyContent,
  getReasoningText,
  getToolSteps,
  isReasoningStreamDelta,
  isToolStepStreamDelta,
  REASONING_APPEND_METADATA_KEY,
  REASONING_TEXT_METADATA_KEY,
  TOOL_STEPS_METADATA_KEY,
  type ToolStepMeta,
} from '../types/protocol';
import type { ConnectionStatus } from '../hooks/useWebSocket';
import { ChatAPI } from '../api/chatAPI';
import { getHubBaseURL, normalizeHubBaseURL } from '../config/hubUrl';
import { isHumanJoinAnnouncement } from '../utils/joinMessage';
import { mergeMessagePreservingImages } from '../utils/mergeMessageImageMetadata';
import {
  capStreamContent,
  MAX_UI_CHANNEL_MESSAGES,
  MAX_UI_THREAD_MESSAGES,
  trimMessagesToMax,
} from '../config/messageLimits';

function isPendingLocalSlashCommand(message: Message): boolean {
  return (
    message.metadata?.slash_command === true &&
    message.metadata?.client_only === true
  );
}

function slashCommandContentKey(content: string): string {
  return content.trim();
}

/** Keep optimistic slash-command lines until the hub echoes them in a refresh. */
function mergePendingLocalSlashCommands(
  incoming: Message[],
  previous: Message[],
): Message[] {
  const echoed = new Set(
    incoming
      .filter((m) => m.metadata?.slash_command === true)
      .map((m) => slashCommandContentKey(m.content)),
  );
  const pending = previous.filter(
    (m) =>
      isPendingLocalSlashCommand(m) &&
      !echoed.has(slashCommandContentKey(m.content)),
  );
  if (!pending.length) {
    return incoming;
  }
  const merged = [...incoming];
  for (const local of pending) {
    if (
      merged.some(
        (m) =>
          m.metadata?.slash_command === true &&
          slashCommandContentKey(m.content) === slashCommandContentKey(local.content),
      )
    ) {
      continue;
    }
    merged.push(local);
  }
  merged.sort(
    (a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime(),
  );
  return merged;
}

function stripLocalSlashDuplicates(
  messages: Message[],
  incoming: Message,
): Message[] {
  if (incoming.metadata?.slash_command !== true || incoming.metadata?.client_only === true) {
    return messages;
  }
  const key = slashCommandContentKey(incoming.content);
  return messages.filter(
    (m) =>
      !(
        isPendingLocalSlashCommand(m) &&
        slashCommandContentKey(m.content) === key
      ),
  );
}

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
  unreadCounts: Record<string, number>;
  pendingScrollToMessageId: string | null;
  highlightMessageId: string | null;
  
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

  /** User Stop / interject — agents paused until next human message. */
  channelHeld: Map<string, boolean>;

  /** Per-channel ring buffer of live turn telemetry (routing, tools, activity). */
  turnTelemetryByChannel: Map<string, TurnTelemetryEvent[]>;
  
  // Actions
  setConnectionStatus: (status: ConnectionStatus) => void;
  setServerAddr: (addr: string) => void;
  setChannel: (channel: string) => void;
  setUsername: (username: string) => void;
  addMessage: (message: Message) => void;
  /** Merge tool_approval rows by metadata.approval_id (pending → approved/rejected). */
  upsertToolApprovalMessage: (message: Message) => void;
  setMessages: (messages: Message[]) => void;
  prependMessages: (messages: Message[]) => void;
  setAgents: (agents: AgentInfo[]) => void;
  setIsTyping: (isTyping: boolean) => void;
  setErrorMessage: (message: string | null) => void;
  addThinkingAgent: (
    channelName: string,
    agentId: string,
    agentName: string,
    agentType: AgentType,
    activity?: string,
    activityDetail?: string
  ) => void;
  updateThinkingAgentActivity: (
    channelName: string,
    agentId: string,
    patch: {
      activity?: string;
      activityDetail?: string;
      toolStep?: ToolStepMeta;
    }
  ) => void;
  removeThinkingAgent: (channelName: string, agentId: string) => void;
  clearThinkingAgents: (channelName?: string) => void;
  cleanupStaleThinking: (channelName: string, messages: Message[]) => void;
  appendTurnTelemetryEvent: (channelName: string, event: Omit<TurnTelemetryEvent, 'id' | 'at' | 'channel'>) => void;
  clearTurnTelemetry: (channelName?: string) => void;
  updateAgentStatus: (agentId: string, updates: Partial<AgentInfo>) => void;
  
  // Channel actions
  setChannels: (channels: Channel[]) => void;
  switchChannel: (channelName: string) => void;
  markChannelUnread: (channelName: string) => void;
  clearChannelUnread: (channelName: string) => void;
  getUnreadCount: (channelName: string) => number;
  setPendingScrollToMessageId: (messageId: string | null) => void;
  setHighlightMessageId: (messageId: string | null) => void;
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
  setChannelHold: (channelName: string, held: boolean) => void;
  isChannelHeld: (channelName: string) => boolean;
  stopAllStreamsForChannel: (channelName: string) => void;
  
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
  unreadCounts: {} as Record<string, number>,
  pendingScrollToMessageId: null as string | null,
  highlightMessageId: null as string | null,
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
  channelHeld: new Map<string, boolean>(),
  turnTelemetryByChannel: new Map<string, TurnTelemetryEvent[]>(),
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
  
  setServerAddr: (addr) => set({ serverAddr: normalizeHubBaseURL(addr) }),
  
  setChannel: (channel) => set({ channel }),
  
  setUsername: (username) => set({ username }),
  
  addMessage: (message) =>
    set((state) => {
      // Skip empty messages (some CLI agents send blank status messages)
      if (!message.content?.trim() && !channelTimelineAllowsEmptyContent(message.type)) {
        return state;
      }

      if (isHumanJoinAnnouncement(message)) {
        const dup = state.messages.some(
          (m) =>
            isHumanJoinAnnouncement(m) &&
            m.content === message.content &&
            m.from?.name === message.from?.name
        );
        if (dup) {
          return state;
        }
      }

      let baseMessages = stripLocalSlashDuplicates(state.messages, message);

      const existingIdx = baseMessages.findIndex(m => m.id === message.id);
      const streaming = state.streamingMessages[message.id];
      const { [message.id]: _removedStream, ...restStreams } = state.streamingMessages;
      const cleanedStreams = streaming ? restStreams : state.streamingMessages;

      if (existingIdx !== -1) {
        const updated = [...baseMessages];
        updated[existingIdx] = mergeMessagePreservingImages(updated[existingIdx], message);
        return {
          messages: trimMessagesToMax(updated, MAX_UI_CHANNEL_MESSAGES),
          streamingMessages: cleanedStreams,
          isTyping: false,
        };
      }

      return {
        messages: trimMessagesToMax([...baseMessages, message], MAX_UI_CHANNEL_MESSAGES),
        streamingMessages: cleanedStreams,
        isTyping: false,
      };
    }),

  upsertToolApprovalMessage: (message) =>
    set((state) => {
      const approvalId = message.metadata?.approval_id as string | undefined;
      if (!approvalId) {
        return state;
      }
      const existingIdx = state.messages.findIndex(
        (m) => m.type === 'tool_approval' && m.metadata?.approval_id === approvalId,
      );
      if (existingIdx !== -1) {
        const updated = [...state.messages];
        updated[existingIdx] = {
          ...updated[existingIdx],
          content: message.content,
          metadata: { ...updated[existingIdx].metadata, ...message.metadata },
          timestamp: message.timestamp,
        };
        return { messages: updated, isTyping: false };
      }
      return {
        messages: trimMessagesToMax([...state.messages, message], MAX_UI_CHANNEL_MESSAGES),
        isTyping: false,
      };
    }),

  setMessages: (messages) =>
    set((state) => {
      const byId = new Map(state.messages.map((m) => [m.id, m]));
      const mergedIncoming = mergePendingLocalSlashCommands(messages, state.messages).map((m) => {
        const prev = byId.get(m.id);
        return prev ? mergeMessagePreservingImages(prev, m) : m;
      });
      return {
        messages: trimMessagesToMax(
          mergedIncoming.filter(
            (m) => !!m.content?.trim() || channelTimelineAllowsEmptyContent(m.type)
          ),
          MAX_UI_CHANNEL_MESSAGES
        ),
      };
    }),

  prependMessages: (older) =>
    set((state) => {
      const ids = new Set(state.messages.map((m) => m.id));
      const byId = new Map(state.messages.map((m) => [m.id, m]));
      const mergedOlder = older
        .filter((m) => !ids.has(m.id))
        .map((m) => {
          const prev = byId.get(m.id);
          return prev ? mergeMessagePreservingImages(prev, m) : m;
        });
      const merged = [...mergedOlder, ...state.messages];
      return {
        messages: trimMessagesToMax(
          merged.filter((m) => !!m.content?.trim() || channelTimelineAllowsEmptyContent(m.type)),
          MAX_UI_CHANNEL_MESSAGES
        ),
      };
    }),
  
  setAgents: (agents) => set({ agents }),
  
  setIsTyping: (isTyping) => set({ isTyping }),
  
  setErrorMessage: (message) => set({ errorMessage: message }),
  
  addThinkingAgent: (channelName, agentId, agentName, agentType, activity, activityDetail) =>
    set((state) => {
      const innerExisting = state.channelThinkingAgents.get(channelName);
      const prev = innerExisting?.get(agentId);
      if (
        prev &&
        prev.name === agentName &&
        prev.type === agentType &&
        prev.activity === activity &&
        prev.activityDetail === activityDetail
      ) {
        return state;
      }
      const outer = new Map(state.channelThinkingAgents);
      const inner = new Map(outer.get(channelName) || []);
      inner.set(agentId, {
        id: agentId,
        name: agentName,
        type: agentType,
        activity,
        activityDetail,
        toolSteps: prev?.toolSteps,
        startedAt: prev?.startedAt ?? Date.now(),
      });
      outer.set(channelName, inner);
      return { channelThinkingAgents: outer };
    }),

  updateThinkingAgentActivity: (channelName, agentId, patch) =>
    set((state) => {
      const inner = state.channelThinkingAgents.get(channelName);
      if (!inner || !inner.has(agentId)) return state;
      const prev = inner.get(agentId)!;
      const toolSteps = patch.toolStep
        ? [...(prev.toolSteps ?? []), patch.toolStep].slice(-12)
        : prev.toolSteps;
      const next: ThinkingAgent = {
        ...prev,
        activity: patch.activity !== undefined ? patch.activity : prev.activity,
        activityDetail:
          patch.activityDetail !== undefined ? patch.activityDetail : prev.activityDetail,
        toolSteps,
      };
      if (
        next.activity === prev.activity &&
        next.activityDetail === prev.activityDetail &&
        next.toolSteps === prev.toolSteps
      ) {
        return state;
      }
      const outer = new Map(state.channelThinkingAgents);
      const newInner = new Map(inner);
      newInner.set(agentId, next);
      outer.set(channelName, newInner);
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

  appendTurnTelemetryEvent: (channelName, event) =>
    set((state) => {
      const prev = state.turnTelemetryByChannel.get(channelName) ?? [];
      const row: TurnTelemetryEvent = {
        ...event,
        id: `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
        at: Date.now(),
        channel: channelName,
      };
      const next = [...prev, row].slice(-100);
      const outer = new Map(state.turnTelemetryByChannel);
      outer.set(channelName, next);
      return { turnTelemetryByChannel: outer };
    }),

  clearTurnTelemetry: (channelName) =>
    set((state) => {
      if (!channelName) {
        return { turnTelemetryByChannel: new Map<string, TurnTelemetryEvent[]>() };
      }
      if (!state.turnTelemetryByChannel.has(channelName)) return state;
      const outer = new Map(state.turnTelemetryByChannel);
      outer.delete(channelName);
      return { turnTelemetryByChannel: outer };
    }),

  cleanupStaleThinking: (channelName, messages) =>
    set((state) => {
      const inner = state.channelThinkingAgents.get(channelName);
      if (!inner || inner.size === 0) return state;
      const respondedAgentIds = new Set(
        messages
          .filter(
            (m) =>
              m.type === 'chat' ||
              m.type === 'answer' ||
              m.type === 'collaboration_discussion'
          )
          .map((m) => m.from?.id)
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
      const newCounts = { ...state.unreadCounts };
      delete newCounts[channelName];

      return {
        channel: channelName,
        messages: cachedMessages,
        channelMessages: newCache,
        unreadChannels: newUnread,
        unreadCounts: newCounts,
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
      const newCounts = { ...state.unreadCounts };
      newCounts[channelName] = (newCounts[channelName] ?? 0) + 1;
      return { unreadChannels: newUnread, unreadCounts: newCounts };
    }),

  clearChannelUnread: (channelName) =>
    set((state) => {
      const newUnread = new Set(state.unreadChannels);
      newUnread.delete(channelName);
      const newCounts = { ...state.unreadCounts };
      delete newCounts[channelName];
      return { unreadChannels: newUnread, unreadCounts: newCounts };
    }),

  getUnreadCount: (channelName) => get().unreadCounts[channelName] ?? 0,

  setPendingScrollToMessageId: (messageId) => set({ pendingScrollToMessageId: messageId }),

  setHighlightMessageId: (messageId) => set({ highlightMessageId: messageId }),

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
    const isToolStep = isToolStepStreamDelta(meta);
    const reasoningChunk =
      typeof meta[REASONING_APPEND_METADATA_KEY] === 'string'
        ? (meta[REASONING_APPEND_METADATA_KEY] as string)
        : '';

    const mergeDelta = (base: Message): Message => {
      if (isToolStep) {
        const prev = getToolSteps(base.metadata as Record<string, unknown> | undefined);
        const step: ToolStepMeta = {
          kind: String(meta.tool_step ?? ''),
          name: String(meta.tool_name ?? ''),
          iteration: typeof meta.tool_iteration === 'number' ? meta.tool_iteration : undefined,
          preview: typeof meta.tool_preview === 'string' ? meta.tool_preview : undefined,
        };
        return {
          ...base,
          type: 'chat' as Message['type'],
          metadata: {
            ...base.metadata,
            [TOOL_STEPS_METADATA_KEY]: [...prev, step],
          },
        };
      }
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
    } else if (isReasoning || isToolStep) {
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

  setChannelHold: (channelName, held) =>
    set((state) => {
      const next = new Map(state.channelHeld);
      if (held) {
        next.set(channelName, true);
      } else {
        next.delete(channelName);
      }
      return { channelHeld: next };
    }),

  isChannelHeld: (channelName) => get().channelHeld.get(channelName) === true,

  stopAllStreamsForChannel: (channelName) => {
    const { streamingMessages, channel: activeChannel, finalizeStream: fin } = get();
    for (const [id, msg] of Object.entries(streamingMessages)) {
      const ch = msg.channel || activeChannel;
      if (ch === channelName) {
        fin(id);
      }
    }
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
      unreadCounts: {},
      pendingScrollToMessageId: null,
      highlightMessageId: null,
      streamingMessages: {},
      channelHeld: new Map<string, boolean>(),
      turnTelemetryByChannel: new Map<string, TurnTelemetryEvent[]>(),
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
    unreadCounts: {},
    pendingScrollToMessageId: null,
    highlightMessageId: null,
    streamingMessages: {},
    channelHeld: new Map<string, boolean>(),
    turnTelemetryByChannel: new Map<string, TurnTelemetryEvent[]>(),
  });
  },
};
});

