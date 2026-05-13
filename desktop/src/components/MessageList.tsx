import React, { useEffect, useMemo, useRef, useState, useCallback, forwardRef } from 'react';
import { Virtuoso, type VirtuosoHandle } from 'react-virtuoso';
import type { Message as MessageType } from '../types/protocol';
import { channelTimelineAllowsEmptyContent } from '../types/protocol';
import { Message } from './Message';
import { useChatStore } from '../stores/chatStore';
import { shallow } from 'zustand/shallow';

type ListRow =
  | { kind: 'message'; message: MessageType }
  | { kind: 'stream'; message: MessageType };

/** Stable Scroller so wheel handling does not remount Virtuoso every render */
const VirtuosoScroller = forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  function VirtuosoScroller(props, ref) {
    const { onWheel, ...rest } = props;
    return (
      <div
        {...rest}
        ref={ref}
        onWheel={(e) => {
          onWheel?.(e);
          if (e.deltaY < 0) {
            window.dispatchEvent(new CustomEvent('nj-chat-scroll-up'));
          }
        }}
      />
    );
  }
);

interface MessageListProps {
  searchQuery?: string;
}

export function MessageList({ searchQuery = '' }: MessageListProps) {
  const { messages, threadMetadata, streamingMessages, openThread } = useChatStore(
    (s) => ({
      messages: s.messages,
      threadMetadata: s.threadMetadata,
      streamingMessages: s.streamingMessages,
      openThread: s.openThread,
    }),
    shallow
  );

  const virtuosoRef = useRef<VirtuosoHandle>(null);
  const lastRenderCountRef = useRef(0);
  const [isNearBottom, setIsNearBottom] = useState(true);
  const [pendingMessageCount, setPendingMessageCount] = useState(0);
  const [showJumpButton, setShowJumpButton] = useState(false);

  const normalizedSearchQuery = searchQuery.trim().toLowerCase();

  const channelMessages = useMemo(() => {
    return messages.filter((m) => {
      if (m.is_thread_reply) return false;
      if (!m.content?.trim() && !channelTimelineAllowsEmptyContent(m.type)) return false;
      if (!normalizedSearchQuery) return true;
      const content = m.content?.toLowerCase() || '';
      const authorName = m.from?.name?.toLowerCase() || '';
      return content.includes(normalizedSearchQuery) || authorName.includes(normalizedSearchQuery);
    });
  }, [messages, normalizedSearchQuery]);

  const activeStreams = useMemo(() => Object.values(streamingMessages), [streamingMessages]);

  /**
   * Total length of all in-flight stream content. Used as a render-cheap
   * dependency so we can re-pin the scroll to bottom while a streaming row
   * grows in height — Virtuoso's followOutput only fires on data length
   * changes, not on item resize.
   */
  const streamContentBytes = useMemo(() => {
    let n = 0;
    for (const s of activeStreams) n += s.content?.length ?? 0;
    return n;
  }, [activeStreams]);

  const rows: ListRow[] = useMemo(() => {
    const out: ListRow[] = [];
    for (const m of channelMessages) {
      out.push({ kind: 'message', message: m });
    }
    for (const m of activeStreams) {
      out.push({ kind: 'stream', message: m });
    }
    return out;
  }, [channelMessages, activeStreams]);

  const totalVisible = rows.length;

  useEffect(() => {
    const onScrollUp = () => {
      setIsNearBottom((near) => (near ? false : near));
      setShowJumpButton((show) => (show ? show : true));
    };
    window.addEventListener('nj-chat-scroll-up', onScrollUp);
    return () => window.removeEventListener('nj-chat-scroll-up', onScrollUp);
  }, []);

  useEffect(() => {
    const prevCount = lastRenderCountRef.current;
    const delta = Math.max(0, totalVisible - prevCount);
    lastRenderCountRef.current = totalVisible;

    if (isNearBottom && totalVisible > 0) {
      virtuosoRef.current?.scrollToIndex({
        index: totalVisible - 1,
        align: 'end',
        behavior: 'auto',
      });
      if (showJumpButton) setShowJumpButton(false);
      if (pendingMessageCount !== 0) setPendingMessageCount(0);
      return;
    }

    if (!isNearBottom && delta > 0) {
      setPendingMessageCount((count) => count + delta);
      if (!showJumpButton) setShowJumpButton(true);
    }
  }, [totalVisible, isNearBottom, showJumpButton, pendingMessageCount]);

  /**
   * Keep the view pinned while a streaming row grows in height. We only nudge
   * the scroll position when the user is near the bottom; otherwise we leave
   * them where they are (the jump button handles re-engagement).
   */
  useEffect(() => {
    if (!isNearBottom || totalVisible === 0 || streamContentBytes === 0) return;
    virtuosoRef.current?.scrollToIndex({
      index: totalVisible - 1,
      align: 'end',
      behavior: 'auto',
    });
  }, [streamContentBytes, isNearBottom, totalVisible]);

  const jumpToCurrent = useCallback(() => {
    if (rows.length === 0) return;
    virtuosoRef.current?.scrollToIndex({
      index: rows.length - 1,
      align: 'end',
      behavior: 'smooth',
    });
    setShowJumpButton(false);
    setPendingMessageCount(0);
  }, [rows.length]);

  const itemContent = useCallback(
    (_index: number, row: ListRow) => {
      if (row.kind === 'message') {
        return (
          <Message
            message={row.message}
            threadMetadata={threadMetadata.get(row.message.id)}
            onOpenThread={openThread}
          />
        );
      }
      return <Message message={row.message} onOpenThread={openThread} isStreaming />;
    },
    [threadMetadata, openThread]
  );

  if (rows.length === 0) {
    return (
      <div className="relative flex-1 min-h-0 bg-slack-bg">
        <div className="h-full overflow-y-auto flex items-center justify-center text-slack-textMuted p-8">
          <div className="text-center">
            <p className="text-lg mb-2">{normalizedSearchQuery ? 'No matches in this chat' : 'No messages yet'}</p>
            <p className="text-sm">
              {normalizedSearchQuery ? 'Try a different search.' : 'Start the conversation!'}
            </p>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="relative flex-1 min-h-0 bg-slack-bg">
      <Virtuoso
        ref={virtuosoRef}
        className="h-full"
        style={{ scrollbarWidth: 'thin' }}
        data={rows}
        computeItemKey={(_, row) => (row.kind === 'message' ? row.message.id : `stream-${row.message.id}`)}
        components={{ Scroller: VirtuosoScroller }}
        followOutput={isNearBottom ? 'auto' : false}
        atBottomStateChange={(atBottom) => {
          setIsNearBottom(atBottom);
          if (atBottom) {
            setShowJumpButton(false);
            setPendingMessageCount(0);
          }
        }}
        increaseViewportBy={{ top: 200, bottom: 400 }}
        itemContent={itemContent}
      />

      {showJumpButton && (
        <button
          type="button"
          onClick={jumpToCurrent}
          className="absolute bottom-3 right-3 rounded-full bg-slack-accent hover:bg-slack-accentHover text-white text-xs px-3 py-1.5 shadow-lg z-10"
          title="Jump to current message"
        >
          {pendingMessageCount > 0 ? `Jump to current (${pendingMessageCount})` : 'Jump to current'}
        </button>
      )}
    </div>
  );
}
