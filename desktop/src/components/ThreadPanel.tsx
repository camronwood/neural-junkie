import { useState, useEffect, useRef, useMemo } from 'react';
import { shallow } from 'zustand/shallow';
import { useChatStore } from '../stores/chatStore';
import { ChatAPI } from '../api/chatAPI';
import { useWebSocket } from '../hooks/useWebSocket';
import { Message } from './Message';
import { RichTextInput } from './RichTextInput';
import { getAgentColor } from '../types/protocol';
import type { Message as MessageType } from '../types/protocol';

interface ThreadPanelProps {
  threadId: string;
  parentMessage: MessageType;
  onClose: () => void;
  /** Sends a thread reply (hub data consent + metadata handled by parent). */
  onSendReply: (content: string, composerMetadata?: Record<string, unknown>) => Promise<void>;
}

const MIN_WIDTH = 250; // Minimum usable width
const DEFAULT_WIDTH = 400;
const STORAGE_KEY = 'thread-panel-width';

export function ThreadPanel({ threadId, parentMessage, onClose, onSendReply }: ThreadPanelProps) {
  const {
    serverAddr,
    channel,
    agents,
    threadMessages,
    threadMetadata,
    setThreadMessages,
    updateThreadMetadata,
    addThreadMessage,
    streamingMessages,
  } = useChatStore(
    (s) => ({
      serverAddr: s.serverAddr,
      channel: s.channel,
      agents: s.agents,
      threadMessages: s.threadMessages,
      threadMetadata: s.threadMetadata,
      setThreadMessages: s.setThreadMessages,
      updateThreadMetadata: s.updateThreadMetadata,
      addThreadMessage: s.addThreadMessage,
      streamingMessages: s.streamingMessages,
    }),
    shallow
  );

  const [api] = useState(() => new ChatAPI(serverAddr));
  const [isLoading, setIsLoading] = useState(true);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  
  // Resize state
  const [width, setWidth] = useState<number>(() => {
    const saved = localStorage.getItem(STORAGE_KEY);
    const savedWidth = saved ? parseInt(saved, 10) : DEFAULT_WIDTH;
    // Sanity check: ensure saved width is reasonable (not larger than screen)
    const maxReasonableWidth = window.innerWidth * 0.6; // Max 60% of screen
    return savedWidth > maxReasonableWidth ? DEFAULT_WIDTH : savedWidth;
  });
  const [isResizing, setIsResizing] = useState(false);
  const resizeStartX = useRef<number>(0);
  const resizeStartWidth = useRef<number>(0);
  const currentWidthRef = useRef<number>(width);
  
  // Keep ref in sync with state
  useEffect(() => {
    currentWidthRef.current = width;
  }, [width]);

  // Get messages for this thread
  const messages = threadMessages.get(threadId) || [];
  const metadata = threadMetadata.get(threadId);
  const threadStreams = useMemo(
    () =>
      Object.values(streamingMessages).filter(
        (m) => m.is_thread_reply && m.thread_id === threadId
      ),
    [streamingMessages, threadId]
  );

  // WebSocket URL for this thread
  const threadWsURL = api.getThreadWebSocketURL(channel, threadId);

  // Subscribe to thread WebSocket for real-time updates
  useWebSocket({
    url: threadWsURL,
    onMessage: async (message: MessageType) => {
      const st = useChatStore.getState();
      if (message.type === 'stream_delta') {
        st.appendStreamDelta(message);
        return;
      }
      if (message.type === 'stream_end') {
        st.finalizeStream(message.id);
        return;
      }

      addThreadMessage(message);

      try {
        const meta = await api.fetchThreadMetadata(threadId);
        updateThreadMetadata(threadId, meta);
      } catch (error) {
        console.error('Failed to fetch thread metadata:', error);
      }
    },
    onConnect: () => {
      console.log('Connected to thread WebSocket');
    },
    onDisconnect: () => {
      console.log('Disconnected from thread WebSocket');
    },
  });

  // Load thread data on mount
  useEffect(() => {
    const loadThreadData = async () => {
      setIsLoading(true);
      try {
        // Load messages and metadata in parallel
        const [msgs, meta] = await Promise.all([
          api.fetchThreadMessages(threadId),
          api.fetchThreadMetadata(threadId),
        ]);
        
        setThreadMessages(threadId, msgs);
        updateThreadMetadata(threadId, meta);
      } catch (error) {
        console.error('Failed to load thread data:', error);
      } finally {
        setIsLoading(false);
      }
    };

    loadThreadData();
  }, [threadId, api, setThreadMessages, updateThreadMetadata]);

  // Auto-scroll to bottom when messages change
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  // Resize handlers
  useEffect(() => {
    const handleMouseMove = (e: MouseEvent) => {
      if (!isResizing) return;
      
      const delta = resizeStartX.current - e.clientX;
      const newWidth = resizeStartWidth.current + delta;
      // Allow free resizing, but limit to reasonable maximum
      // Thread panel should not take more than 50% of screen
      const maxWidth = Math.min(window.innerWidth * 0.5, 800); // Max 50% of screen or 800px
      const clampedWidth = Math.max(MIN_WIDTH, Math.min(maxWidth, newWidth));
      
      setWidth(clampedWidth);
    };

    const handleMouseUp = () => {
      if (isResizing) {
        setIsResizing(false);
        localStorage.setItem(STORAGE_KEY, currentWidthRef.current.toString());
      }
    };

    if (isResizing) {
      document.addEventListener('mousemove', handleMouseMove);
      document.addEventListener('mouseup', handleMouseUp);
      document.body.style.cursor = 'col-resize';
      document.body.style.userSelect = 'none';
    }

    return () => {
      document.removeEventListener('mousemove', handleMouseMove);
      document.removeEventListener('mouseup', handleMouseUp);
      document.body.style.cursor = '';
      document.body.style.userSelect = '';
    };
  }, [isResizing]);

  const handleResizeStart = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setIsResizing(true);
    resizeStartX.current = e.clientX;
    resizeStartWidth.current = currentWidthRef.current;
  };

  const handleSendReply = async (content: string, composerMeta?: Record<string, unknown>) => {
    try {
      await onSendReply(content, composerMeta);
    } catch (error) {
      console.error('Failed to send thread reply:', error);
    }
  };

  const agentColor = getAgentColor(parentMessage.from.type);

  return (
    <div 
      className="border-l border-slack-border bg-slack-bg flex flex-col h-full relative flex-shrink-0"
      style={{ width: `${width}px`, minWidth: `${MIN_WIDTH}px` }}
    >
      {/* Resize Handle */}
      <div
        className="absolute left-0 top-0 bottom-0 cursor-col-resize z-[100] group"
        onMouseDown={handleResizeStart}
        style={{ 
          width: '6px', 
          marginLeft: '-3px',
          pointerEvents: 'auto',
        }}
      >
        <div className="absolute inset-0 bg-transparent group-hover:bg-blue-500/30 transition-colors" />
        <div className="absolute left-1/2 top-1/2 -translate-y-1/2 -translate-x-1/2 w-1 h-8 bg-gray-400 group-hover:bg-blue-500 rounded-full opacity-0 group-hover:opacity-100 transition-opacity" />
      </div>
      
      {/* Thread Header */}
      <div className="px-4 py-3 border-b border-slack-border flex items-center justify-between bg-slack-bgHover">
        <h2 className="font-bold text-slack-text">Thread</h2>
        <button
          onClick={onClose}
          className="p-1.5 rounded text-slack-textMuted hover:text-slack-text hover:bg-slack-bg transition-colors"
          aria-label="Close thread"
        >
          <svg
            className="w-5 h-5"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M6 18L18 6M6 6l12 12"
            />
          </svg>
        </button>
      </div>

      {/* Parent Message */}
      <div className="px-4 py-3 border-b border-slack-border bg-slack-bgHover">
        <div className="flex items-baseline gap-2 mb-1">
          <span
            className="font-semibold"
            style={{ color: agentColor }}
          >
            {parentMessage.from.name}
          </span>
          <span className="text-xs text-slack-textMuted">
            {new Date(parentMessage.timestamp).toLocaleTimeString('en-US', {
              hour: '2-digit',
              minute: '2-digit',
            })}
          </span>
        </div>
        <div className="text-slack-text whitespace-pre-wrap break-words">
          {parentMessage.content}
        </div>
        {metadata && metadata.reply_count > 0 && (
          <div className="mt-2 text-xs text-slack-textMuted">
            {metadata.reply_count} {metadata.reply_count === 1 ? 'reply' : 'replies'}
          </div>
        )}
      </div>

      {/* Thread Messages */}
      <div className="flex-1 min-h-0 overflow-y-auto bg-slack-bg">
        {isLoading ? (
          <div className="flex items-center justify-center h-full text-slack-textMuted">
            Loading thread...
          </div>
        ) : messages.length === 0 && threadStreams.length === 0 ? (
          <div className="flex items-center justify-center h-full text-slack-textMuted">
            No replies yet. Start the conversation!
          </div>
        ) : (
          <div className="py-2">
            {messages.map((msg) => (
              <Message key={msg.id} message={msg} />
            ))}
            {threadStreams.map((msg) => (
              <Message key={msg.id} message={msg} isStreaming />
            ))}
            <div ref={messagesEndRef} />
          </div>
        )}
      </div>

      {/* Reply Input */}
      <div className="border-t border-slack-border">
        <RichTextInput
          onSend={handleSendReply}
          placeholder="Reply to thread..."
          agents={agents}
        />
      </div>
    </div>
  );
}

