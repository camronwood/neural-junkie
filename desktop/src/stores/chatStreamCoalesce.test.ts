import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useChatStore } from './chatStore';
import type { AgentInfo, Message } from '../types/protocol';

const agent: AgentInfo = {
  id: 'a1',
  name: 'Test',
  type: 'backend',
  expertise: [],
  status: 'active',
  model: 'test',
  is_paused: false,
};

function streamDelta(id: string, chunk: string, ch = 'general'): Message {
  return {
    id,
    type: 'stream_delta',
    channel: ch,
    from: agent,
    content: chunk,
    timestamp: new Date().toISOString(),
  };
}

describe('chatStore stream coalescing', () => {
  beforeEach(() => {
    useChatStore.getState().reset();
  });

  it('buffers deltas until rAF flush merges content', () => {
    let scheduled: FrameRequestCallback | null = null;
    vi.stubGlobal('requestAnimationFrame', (cb: FrameRequestCallback) => {
      scheduled = cb;
      return 42 as unknown as number;
    });
    vi.stubGlobal('cancelAnimationFrame', vi.fn());

    const id = 'stream-msg-1';
    useChatStore.getState().appendStreamDelta(streamDelta(id, 'hello'));
    useChatStore.getState().appendStreamDelta(streamDelta(id, ' '));
    useChatStore.getState().appendStreamDelta(streamDelta(id, 'world'));

    expect(useChatStore.getState().streamingMessages[id]).toBeUndefined();

    expect(typeof scheduled).toBe('function');
    scheduled!(0);

    expect(useChatStore.getState().streamingMessages[id]?.content).toBe('hello world');
  });

  it('finalizeStream merges pending then promotes to messages', () => {
    let scheduled: FrameRequestCallback | null = null;
    vi.stubGlobal('requestAnimationFrame', (cb: FrameRequestCallback) => {
      scheduled = cb;
      return 1 as unknown as number;
    });
    vi.stubGlobal('cancelAnimationFrame', vi.fn());

    const id = 'stream-msg-2';
    useChatStore.getState().appendStreamDelta(streamDelta(id, 'done'));
    expect(scheduled).toBeTruthy();
    useChatStore.getState().finalizeStream(id);

    expect(useChatStore.getState().streamingMessages[id]).toBeUndefined();
    const promoted = useChatStore.getState().messages.find((m) => m.id === id);
    expect(promoted?.content).toBe('done');
  });

  it('finalizeStream is a no-op for unknown stream ids', () => {
    vi.stubGlobal('requestAnimationFrame', vi.fn(() => 0 as unknown as number));
    vi.stubGlobal('cancelAnimationFrame', vi.fn());

    const before = useChatStore.getState();
    const streamingRef = before.streamingMessages;
    const messagesRef = before.messages;

    useChatStore.getState().finalizeStream('never-existed');

    const after = useChatStore.getState();
    expect(after.streamingMessages).toBe(streamingRef);
    expect(after.messages).toBe(messagesRef);
  });
});
