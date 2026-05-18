import React, { useEffect, useMemo, useRef, useState, useCallback, forwardRef } from 'react';
import { Virtuoso, type VirtuosoHandle } from 'react-virtuoso';
import type { Message as MessageType } from '../types/protocol';
import { channelTimelineAllowsEmptyContent } from '../types/protocol';
import { isHumanJoinAnnouncement } from '../utils/joinMessage';
import { Message } from './Message';
import { useChatStore } from '../stores/chatStore';
import { shallow } from 'zustand/shallow';

type ListRow =
  | { kind: 'message'; message: MessageType }
  | { kind: 'stream'; message: MessageType };

/** DOM scroller element for reliable scroll-to-bottom (Virtuoso height estimates can lag). */
export const chatScrollerElRef: { current: HTMLDivElement | null } = { current: null };

function mergeScrollerRefs(
  node: HTMLDivElement | null,
  forwardedRef: React.ForwardedRef<HTMLDivElement>
) {
  chatScrollerElRef.current = node;
  if (typeof forwardedRef === 'function') {
    forwardedRef(node);
  } else if (forwardedRef) {
    forwardedRef.current = node;
  }
}

/** Stable Scroller — must not be recreated each render or Virtuoso remounts. */
const VirtuosoScroller = forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  function VirtuosoScroller(props, ref) {
    const { className, ...rest } = props;
    return (
      <div
        {...rest}
        ref={(node) => mergeScrollerRefs(node, ref)}
        className={[className, 'nj-chat-scroller overscroll-y-contain'].filter(Boolean).join(' ')}
      />
    );
  }
);

/** Virtuoso Footer gives the scroller a definite end when item margins confuse height math. */
function ChatListFooter() {
  return <div className="h-px w-full shrink-0" aria-hidden />;
}

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
  const isNearBottomRef = useRef(true);
  const pinScrollRafRef = useRef<number | null>(null);
  const [isNearBottom, setIsNearBottom] = useState(true);
  const [pendingMessageCount, setPendingMessageCount] = useState(0);
  const [showJumpButton, setShowJumpButton] = useState(false);

  const normalizedSearchQuery = searchQuery.trim().toLowerCase();

  const channelMessages = useMemo(() => {
    const filtered = messages.filter((m) => {
      if (m.is_thread_reply) return false;
      if (!m.content?.trim() && !channelTimelineAllowsEmptyContent(m.type)) return false;
      if (!normalizedSearchQuery) return true;
      const content = m.content?.toLowerCase() || '';
      const authorName = m.from?.name?.toLowerCase() || '';
      return content.includes(normalizedSearchQuery) || authorName.includes(normalizedSearchQuery);
    });
    const deduped: MessageType[] = [];
    for (const m of filtered) {
      const prev = deduped[deduped.length - 1];
      if (
        prev &&
        isHumanJoinAnnouncement(prev) &&
        isHumanJoinAnnouncement(m) &&
        prev.content === m.content &&
        prev.from?.name === m.from?.name
      ) {
        continue;
      }
      deduped.push(m);
    }
    return deduped;
  }, [messages, normalizedSearchQuery]);

  const activeStreams = useMemo(() => Object.values(streamingMessages), [streamingMessages]);

  const streamContentBytes = useMemo(() => {
    let n = 0;
    for (const s of activeStreams) {
      n += s.content?.length ?? 0;
      const rt = s.metadata?.reasoning_text;
      if (typeof rt === 'string') n += rt.length;
    }
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
    isNearBottomRef.current = isNearBottom;
  }, [isNearBottom]);

  /**
   * Scroll to the true bottom of the chat. Virtuoso's scrollToIndex can stop short when
   * items have margins or grow after mount (code highlight, mermaid). We combine LAST
   * index, autoscrollToBottom, and a direct scroller scrollTop as fallback.
   */
  const scrollToTrueBottom = useCallback((behavior: 'auto' | 'smooth' = 'auto') => {
    const virtuoso = virtuosoRef.current;
    if (!virtuoso || totalVisible === 0) return;

    virtuoso.scrollToIndex({ index: 'LAST', align: 'end', behavior });
    virtuoso.autoscrollToBottom();

    const scroller = chatScrollerElRef.current;
    if (scroller) {
      scroller.scrollTo({ top: scroller.scrollHeight, behavior });
    }
  }, [totalVisible]);

  const scheduleScrollToTrueBottom = useCallback(
    (behavior: 'auto' | 'smooth' = 'auto') => {
      if (!isNearBottomRef.current || totalVisible === 0) return;
      if (pinScrollRafRef.current != null) {
        cancelAnimationFrame(pinScrollRafRef.current);
      }
      pinScrollRafRef.current = requestAnimationFrame(() => {
        pinScrollRafRef.current = null;
        scrollToTrueBottom(behavior);
        // Second pass after layout (syntax highlighter / mermaid / images).
        requestAnimationFrame(() => scrollToTrueBottom('auto'));
      });
    },
    [scrollToTrueBottom, totalVisible]
  );

  useEffect(() => {
    return () => {
      if (pinScrollRafRef.current != null) {
        cancelAnimationFrame(pinScrollRafRef.current);
      }
    };
  }, []);

  const handleAtBottomStateChange = useCallback((atBottom: boolean) => {
    isNearBottomRef.current = atBottom;
    setIsNearBottom(atBottom);
    if (atBottom) {
      setShowJumpButton(false);
      setPendingMessageCount(0);
    } else {
      setShowJumpButton(true);
    }
  }, []);

  useEffect(() => {
    const prevCount = lastRenderCountRef.current;
    const delta = Math.max(0, totalVisible - prevCount);
    lastRenderCountRef.current = totalVisible;

    if (isNearBottomRef.current && totalVisible > 0) {
      scheduleScrollToTrueBottom('auto');
      setShowJumpButton(false);
      setPendingMessageCount(0);
      return;
    }

    if (!isNearBottomRef.current && delta > 0) {
      setPendingMessageCount((count) => count + delta);
      setShowJumpButton(true);
    }
  }, [totalVisible, scheduleScrollToTrueBottom]);

  useEffect(() => {
    if (streamContentBytes === 0) return;
    scheduleScrollToTrueBottom('auto');
  }, [streamContentBytes, scheduleScrollToTrueBottom]);

  const jumpToCurrent = useCallback(() => {
    if (rows.length === 0) return;
    isNearBottomRef.current = true;
    setIsNearBottom(true);
    setShowJumpButton(false);
    setPendingMessageCount(0);

    scrollToTrueBottom('auto');
    requestAnimationFrame(() => scrollToTrueBottom('auto'));
    window.setTimeout(() => scrollToTrueBottom('auto'), 50);
    window.setTimeout(() => scrollToTrueBottom('auto'), 200);
  }, [rows.length, scrollToTrueBottom]);

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

  const virtuosoComponents = useMemo(
    () => ({
      Scroller: VirtuosoScroller,
      Footer: ChatListFooter,
    }),
    []
  );

  if (rows.length === 0) {
    return (
      <div className="relative flex-1 min-h-0 bg-slack-bg">
        <div className="h-full overflow-y-auto overscroll-y-contain flex items-center justify-center text-slack-textMuted p-8">
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
        className="h-full min-h-0"
        style={{ scrollbarWidth: 'thin' }}
        data={rows}
        alignToBottom
        atBottomThreshold={200}
        computeItemKey={(_, row) => (row.kind === 'message' ? row.message.id : `stream-${row.message.id}`)}
        components={virtuosoComponents}
        followOutput={isNearBottom ? 'auto' : false}
        atBottomStateChange={handleAtBottomStateChange}
        totalListHeightChanged={() => scheduleScrollToTrueBottom('auto')}
        increaseViewportBy={{ top: 400, bottom: 800 }}
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
