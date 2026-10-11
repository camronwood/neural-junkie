import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useChatStore } from './chatStore';
import type { AgentInfo, Message } from '../types/protocol';

const agent: AgentInfo = {
  id: 'agent-1',
  name: 'RustExpert',
  type: 'rust',
  expertise: [],
  status: 'active',
  model: 'test',
  is_paused: false,
};

function streamDelta(id: string, channel: string, content: string): Message {
  return {
    id,
    type: 'stream_delta',
    channel,
    from: agent,
    content,
    timestamp: new Date().toISOString(),
  };
}

describe('chatStore channel-safe finalizeStream', () => {
  beforeEach(() => {
    useChatStore.getState().reset();
    useChatStore.getState().setChannel('general');
    vi.stubGlobal('requestAnimationFrame', (cb: FrameRequestCallback) => {
      cb(0);
      return 1 as unknown as number;
    });
    vi.stubGlobal('cancelAnimationFrame', vi.fn());
  });

  it('promotes on-channel streams into the active timeline', () => {
    const id = 'stream-on-channel';
    useChatStore.getState().appendStreamDelta(streamDelta(id, 'general', 'hello'));
    useChatStore.getState().finalizeStream(id);

    expect(useChatStore.getState().streamingMessages[id]).toBeUndefined();
    expect(useChatStore.getState().messages.find((m) => m.id === id)?.content).toBe('hello');
    expect(useChatStore.getState().channelMessages.get('other')).toBeUndefined();
  });

  it('caches off-channel streams instead of appending to the active timeline', () => {
    const id = 'stream-off-channel';
    useChatStore.getState().appendStreamDelta(streamDelta(id, 'reviews', 'ghost'));
    useChatStore.getState().setChannel('general');
    useChatStore.getState().finalizeStream(id);

    expect(useChatStore.getState().streamingMessages[id]).toBeUndefined();
    expect(useChatStore.getState().messages.find((m) => m.id === id)).toBeUndefined();
    const cached = useChatStore.getState().channelMessages.get('reviews') ?? [];
    expect(cached.find((m) => m.id === id)?.content).toBe('ghost');
  });

  it('stopAllStreamsForChannel finalizes off-active streams into cache', () => {
    useChatStore.getState().setChannel('general');
    const id = 'stream-reviews-stop';
    useChatStore.getState().appendStreamDelta(streamDelta(id, 'reviews', 'partial'));
    useChatStore.getState().stopAllStreamsForChannel('reviews');

    expect(useChatStore.getState().streamingMessages[id]).toBeUndefined();
    expect(useChatStore.getState().messages.find((m) => m.id === id)).toBeUndefined();
    const cached = useChatStore.getState().channelMessages.get('reviews') ?? [];
    expect(cached.find((m) => m.id === id)?.content).toBe('partial');
  });

  it('switchChannel keeps in-flight streams so finalize can route later', () => {
    const id = 'stream-survives-switch';
    useChatStore.getState().appendStreamDelta(streamDelta(id, 'general', 'still going'));
    useChatStore.getState().switchChannel('reviews');

    expect(useChatStore.getState().channel).toBe('reviews');
    expect(useChatStore.getState().streamingMessages[id]?.content).toBe('still going');

    useChatStore.getState().finalizeStream(id);
    expect(useChatStore.getState().messages.find((m) => m.id === id)).toBeUndefined();
    const cached = useChatStore.getState().channelMessages.get('general') ?? [];
    expect(cached.find((m) => m.id === id)?.content).toBe('still going');
  });

  it('switchChannel clears sticky isTyping flag', () => {
    useChatStore.getState().setIsTyping(true);
    useChatStore.getState().switchChannel('reviews');
    expect(useChatStore.getState().isTyping).toBe(false);
  });

  it('setMessages keeps empty tool_approval and user_question cards', () => {
    const approval: Message = {
      id: 'ta-1',
      type: 'tool_approval',
      channel: 'general',
      from: agent,
      content: '',
      timestamp: new Date().toISOString(),
      metadata: { approval_id: 'appr-1', status: 'pending', tool_name: 'run_command' },
    };
    const question: Message = {
      id: 'uq-1',
      type: 'user_question',
      channel: 'general',
      from: agent,
      content: '   ',
      timestamp: new Date().toISOString(),
      metadata: { question_id: 'q-1', status: 'pending' },
    };
    useChatStore.getState().setMessages([approval, question]);
    const msgs = useChatStore.getState().messages;
    expect(msgs.some((m) => m.id === 'ta-1')).toBe(true);
    expect(msgs.some((m) => m.id === 'uq-1')).toBe(true);
  });

  it('addMessageToCache upserts tool_approval by approval_id', () => {
    const pending: Message = {
      id: 'ta-pending',
      type: 'tool_approval',
      channel: 'reviews',
      from: agent,
      content: 'Approve?',
      timestamp: new Date().toISOString(),
      metadata: { approval_id: 'appr-cache', status: 'pending' },
    };
    const resolved: Message = {
      ...pending,
      id: 'ta-resolved',
      content: 'Approved',
      metadata: { approval_id: 'appr-cache', status: 'approved' },
    };
    useChatStore.getState().addMessageToCache('reviews', pending);
    useChatStore.getState().addMessageToCache('reviews', resolved);
    const cached = useChatStore.getState().channelMessages.get('reviews') ?? [];
    const cards = cached.filter(
      (m) => m.type === 'tool_approval' && m.metadata?.approval_id === 'appr-cache',
    );
    expect(cards).toHaveLength(1);
    expect(cards[0].metadata?.status).toBe('approved');
  });
});
