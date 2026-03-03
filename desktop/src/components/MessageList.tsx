import { useEffect, useRef, useMemo, useState } from 'react';
import type { Message as MessageType, ThreadMetadata } from '../types/protocol';
import { Message } from './Message';

interface MessageListProps {
  messages: MessageType[];
  threadMetadata: Map<string, ThreadMetadata>;
  onOpenThread: (threadId: string) => void;
  streamingMessages?: Record<string, MessageType>;
  searchQuery?: string;
}

export function MessageList({ messages, threadMetadata, onOpenThread, streamingMessages, searchQuery = '' }: MessageListProps) {
  const scrollRef = useRef<HTMLDivElement>(null);
  const bottomRef = useRef<HTMLDivElement>(null);
  const lastRenderCountRef = useRef(0);
  const lastScrollTopRef = useRef(0);
  const [isNearBottom, setIsNearBottom] = useState(true);
  const [pendingMessageCount, setPendingMessageCount] = useState(0);
  const [showJumpButton, setShowJumpButton] = useState(false);

  const normalizedSearchQuery = searchQuery.trim().toLowerCase();

  // Filter out thread replies and empty messages - only show visible channel messages.
  const channelMessages = useMemo(() => {
    return messages.filter(m => {
      if (m.is_thread_reply) return false;
      if (!m.content?.trim() && m.type !== 'agent_join' && m.type !== 'agent_leave' && m.type !== 'system_info') return false;
      if (!normalizedSearchQuery) return true;
      const content = m.content?.toLowerCase() || '';
      const authorName = m.from?.name?.toLowerCase() || '';
      return content.includes(normalizedSearchQuery) || authorName.includes(normalizedSearchQuery);
    });
  }, [messages, normalizedSearchQuery]);

  const activeStreams = useMemo(() => {
    if (!streamingMessages) return [];
    return Object.values(streamingMessages);
  }, [streamingMessages]);

  const totalVisibleMessages = channelMessages.length + activeStreams.length;

  const handleScroll = () => {
    const container = scrollRef.current;
    if (!container) return;
    const currentScrollTop = container.scrollTop;
    const scrollingUp = currentScrollTop < lastScrollTopRef.current;
    lastScrollTopRef.current = currentScrollTop;
    const distanceFromBottom = container.scrollHeight - container.scrollTop - container.clientHeight;
    const nearBottom = distanceFromBottom < 16;
    if (scrollingUp && distanceFromBottom > 0) {
      if (isNearBottom) setIsNearBottom(false);
      if (!showJumpButton) setShowJumpButton(true);
      return;
    }
    setIsNearBottom(nearBottom);
    if (nearBottom) {
      setShowJumpButton(false);
      setPendingMessageCount(0);
    }
  };

  const jumpToCurrent = () => {
    if (bottomRef.current) {
      bottomRef.current.scrollIntoView({ behavior: 'smooth' });
    }
    setShowJumpButton(false);
    setPendingMessageCount(0);
  };

  // Auto-scroll only if user is already near bottom.
  useEffect(() => {
    const newCount = totalVisibleMessages;
    const prevCount = lastRenderCountRef.current;
    const delta = Math.max(0, newCount - prevCount);
    lastRenderCountRef.current = newCount;

    if (isNearBottom && bottomRef.current) {
      // Streaming can update many times per second; avoid smooth-scroll jank.
      bottomRef.current.scrollIntoView({ behavior: 'auto' });
      if (showJumpButton) setShowJumpButton(false);
      if (pendingMessageCount !== 0) setPendingMessageCount(0);
      return;
    }

    if (!isNearBottom && (delta > 0 || activeStreams.length > 0)) {
      if (delta > 0) {
        setPendingMessageCount((count) => count + delta);
      }
      if (!showJumpButton) setShowJumpButton(true);
    }
  }, [channelMessages, activeStreams, isNearBottom, totalVisibleMessages, showJumpButton, pendingMessageCount]);

  return (
    <div className="relative flex-1 min-h-0 bg-slack-bg">
      <div
        ref={scrollRef}
        className="h-full overflow-y-auto"
        style={{ scrollbarWidth: 'thin' }}
        onScroll={handleScroll}
        onWheel={(e) => {
          if (e.deltaY < 0) {
            // Immediately disengage sticky autoscroll when user scrolls up.
            if (isNearBottom) setIsNearBottom(false);
            if (!showJumpButton) setShowJumpButton(true);
          }
        }}
      >
        <div className="flex flex-col">
          {channelMessages.length === 0 && activeStreams.length === 0 ? (
            <div className="flex items-center justify-center h-full text-slack-textMuted p-8">
              <div className="text-center">
                <p className="text-lg mb-2">{normalizedSearchQuery ? 'No matches in this chat' : 'No messages yet'}</p>
                <p className="text-sm">
                  {normalizedSearchQuery ? 'Try a different search.' : 'Start the conversation!'}
                </p>
              </div>
            </div>
          ) : (
            <>
              {channelMessages.map((message) => (
                <Message
                  key={message.id}
                  message={message}
                  threadMetadata={threadMetadata.get(message.id)}
                  onOpenThread={onOpenThread}
                />
              ))}
              {activeStreams.map((msg) => (
                <Message
                  key={`stream-${msg.id}`}
                  message={msg}
                  onOpenThread={onOpenThread}
                  isStreaming
                />
              ))}
              <div ref={bottomRef} />
            </>
          )}
        </div>
      </div>

      {showJumpButton && (
        <button
          onClick={jumpToCurrent}
          className="absolute bottom-3 right-3 rounded-full bg-slack-accent hover:bg-slack-accentHover text-white text-xs px-3 py-1.5 shadow-lg"
          title="Jump to current message"
        >
          {pendingMessageCount > 0 ? `Jump to current (${pendingMessageCount})` : 'Jump to current'}
        </button>
      )}
    </div>
  );
}

