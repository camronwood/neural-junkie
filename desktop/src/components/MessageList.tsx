import { useEffect, useRef, useMemo } from 'react';
import type { Message as MessageType, ThreadMetadata } from '../types/protocol';
import { Message } from './Message';

interface MessageListProps {
  messages: MessageType[];
  threadMetadata: Map<string, ThreadMetadata>;
  onOpenThread: (threadId: string) => void;
}

export function MessageList({ messages, threadMetadata, onOpenThread }: MessageListProps) {
  const scrollRef = useRef<HTMLDivElement>(null);
  const bottomRef = useRef<HTMLDivElement>(null);

  // Filter out thread replies - only show channel messages
  const channelMessages = useMemo(() => {
    return messages.filter(m => !m.is_thread_reply);
  }, [messages]);

  // Auto-scroll to bottom when new messages arrive
  useEffect(() => {
    if (bottomRef.current) {
      bottomRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [channelMessages]);

  return (
    <div
      ref={scrollRef}
      className="flex-1 overflow-y-auto bg-slack-bg"
      style={{ scrollbarWidth: 'thin' }}
    >
      <div className="flex flex-col">
        {channelMessages.length === 0 ? (
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
            <div ref={bottomRef} />
          </>
        )}
      </div>
    </div>
  );
}

