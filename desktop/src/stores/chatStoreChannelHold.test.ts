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

describe('chatStore channel hold / interject', () => {
  beforeEach(() => {
    useChatStore.getState().reset();
    useChatStore.getState().setChannel('general');
  });

  it('setChannelHold and isChannelHeld track per channel', () => {
    const st = useChatStore.getState();
    expect(st.isChannelHeld('general')).toBe(false);
    st.setChannelHold('general', true);
    expect(useChatStore.getState().isChannelHeld('general')).toBe(true);
    expect(useChatStore.getState().isChannelHeld('other')).toBe(false);
    st.setChannelHold('general', false);
    expect(useChatStore.getState().isChannelHeld('general')).toBe(false);
  });

  it('clearThinkingAgents removes agents for a channel', () => {
    const st = useChatStore.getState();
    st.addThinkingAgent('general', 'a1', 'Agent One', 'backend');
    st.addThinkingAgent('general', 'a2', 'Agent Two', 'rust');
    expect(useChatStore.getState().channelThinkingAgents.get('general')?.size).toBe(2);
    st.clearThinkingAgents('general');
    expect(useChatStore.getState().channelThinkingAgents.has('general')).toBe(false);
  });

  it('stopAllStreamsForChannel finalizes streaming messages on that channel', () => {
    let scheduled: FrameRequestCallback | null = null;
    vi.stubGlobal('requestAnimationFrame', (cb: FrameRequestCallback) => {
      scheduled = cb;
      return 42 as unknown as number;
    });
    vi.stubGlobal('cancelAnimationFrame', vi.fn());

    const streamId = 'stream-hold-1';
    const delta: Message = {
      id: streamId,
      type: 'stream_delta',
      channel: 'general',
      from: agent,
      content: 'partial',
      timestamp: new Date().toISOString(),
    };
    useChatStore.getState().appendStreamDelta(delta);
    expect(typeof scheduled).toBe('function');
    scheduled!(0);
    expect(useChatStore.getState().streamingMessages[streamId]?.content).toBe('partial');

    useChatStore.getState().stopAllStreamsForChannel('general');
    expect(useChatStore.getState().streamingMessages[streamId]).toBeUndefined();
    const promoted = useChatStore.getState().messages.find((m) => m.id === streamId);
    expect(promoted?.content).toBe('partial');
  });

  it('upsertUserQuestionMessage replaces pending card with answered state', () => {
    const pending: Message = {
      id: 'question-card-1',
      type: 'user_question',
      channel: 'general',
      from: agent,
      content: '**RustExpert** asks: Which genre?',
      timestamp: new Date().toISOString(),
      metadata: { question_id: 'q1', status: 'pending', question: 'Which genre?' },
    };
    const answered: Message = {
      ...pending,
      id: 'question-update-1',
      content: '**Answer:** RTS',
      metadata: { ...pending.metadata, status: 'answered', answer: 'RTS' },
    };

    const st = useChatStore.getState();
    st.upsertUserQuestionMessage(pending);
    st.upsertUserQuestionMessage(answered);

    const cards = useChatStore
      .getState()
      .messages.filter((m) => m.type === 'user_question' && m.metadata?.question_id === 'q1');
    expect(cards).toHaveLength(1);
    expect(cards[0].content).toBe('**Answer:** RTS');
    expect(cards[0].metadata?.status).toBe('answered');
  });

  it('addMessageToCache upserts user questions for inactive channels', () => {
    const pending: Message = {
      id: 'off-channel-pending',
      type: 'user_question',
      channel: 'rustgame',
      from: agent,
      content: 'Which genre?',
      timestamp: new Date().toISOString(),
      metadata: { question_id: 'q-off', status: 'pending' },
    };
    const answered: Message = {
      ...pending,
      id: 'off-channel-answered',
      content: '**Answer:** RTS',
      metadata: { question_id: 'q-off', status: 'answered', answer: 'RTS' },
    };

    const st = useChatStore.getState();
    st.addMessageToCache('rustgame', pending);
    st.addMessageToCache('rustgame', answered);

    const cached = useChatStore.getState().channelMessages.get('rustgame') ?? [];
    const cards = cached.filter(
      (m) => m.type === 'user_question' && m.metadata?.question_id === 'q-off'
    );
    expect(cards).toHaveLength(1);
    expect(cards[0].metadata?.status).toBe('answered');
  });
});
