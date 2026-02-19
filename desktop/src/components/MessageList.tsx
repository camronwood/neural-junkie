import { useEffect, useRef, useMemo } from 'react';
import type { Message as MessageType, ThreadMetadata } from '../types/protocol';
import { Message } from './Message';

interface MessageListProps {
  messages: MessageType[];
  threadMetadata: Map<string, ThreadMetadata>;
  onOpenThread: (threadId: string) => void;
  streamingMessages?: Record<string, MessageType>;
}

export function MessageList({ messages, threadMetadata, onOpenThread, streamingMessages }: MessageListProps) {
  const scrollRef = useRef<HTMLDivElement>(null);
  const bottomRef = useRef<HTMLDivElement>(null);

  // Filter out thread replies - only show channel messages
  const channelMessages = useMemo(() => {
    return messages.filter(m => !m.is_thread_reply);
  }, [messages]);

  const activeStreams = useMemo(() => {
    if (!streamingMessages) return [];
    return Object.values(streamingMessages);
  }, [streamingMessages]);

  // Auto-scroll to bottom when new messages or stream deltas arrive
  useEffect(() => {
    if (bottomRef.current) {
      bottomRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [channelMessages, activeStreams]);

  return (
    <div
      ref={scrollRef}
      className="flex-1 overflow-y-auto bg-slack-bg"
      style={{ scrollbarWidth: 'thin' }}
    >
      <div className="flex flex-col">
        {channelMessages.length === 0 && activeStreams.length === 0 ? (
          <div className="flex items-center justify-center h-full text-slack-textMuted p-8">
            <div className="text-center">
              <p className="text-lg mb-2">No messages yet</p>
              <p className="text-sm">Start the conversation!</p>
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
  );
}

