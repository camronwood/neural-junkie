import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useChatStore } from './chatStore';
import type { AgentInfo, Message } from '../types/protocol';
import {
  REASONING_APPEND_METADATA_KEY,
  REASONING_DELTA_METADATA_KEY,
  REASONING_TEXT_METADATA_KEY,
} from '../types/protocol';

const agent: AgentInfo = {
  id: 'a1',
  name: 'Test',
  type: 'backend',
  expertise: [],
  status: 'active',
  model: 'test',
  is_paused: false,
};

function reasoningDelta(id: string, chunk: string, ch = 'general'): Message {
  return {
    id,
    type: 'stream_delta',
    channel: ch,
    from: agent,
    content: '',
    timestamp: new Date().toISOString(),
    metadata: {
      [REASONING_DELTA_METADATA_KEY]: true,
      [REASONING_APPEND_METADATA_KEY]: chunk,
    },
  };
}

describe('chatStore reasoning stream', () => {
  beforeEach(() => {
    useChatStore.getState().reset();
  });

  it('accumulates reasoning deltas into streaming metadata', () => {
    let scheduled: FrameRequestCallback | null = null;
    vi.stubGlobal('requestAnimationFrame', (cb: FrameRequestCallback) => {
      scheduled = cb;
      return 42 as unknown as number;
    });
    vi.stubGlobal('cancelAnimationFrame', vi.fn());

    const id = 'stream-r1';
    useChatStore.getState().appendStreamDelta(reasoningDelta(id, 'think '));
    useChatStore.getState().appendStreamDelta(reasoningDelta(id, 'step'));

    expect(typeof scheduled).toBe('function');
    scheduled!(0);

    const streamed = useChatStore.getState().streamingMessages[id];
    expect(streamed?.metadata?.[REASONING_TEXT_METADATA_KEY]).toBe('think step');
  });

  it('finalizeStream preserves reasoning_text on promoted message', () => {
    vi.stubGlobal('requestAnimationFrame', (cb: FrameRequestCallback) => {
      cb(0);
      return 1 as unknown as number;
    });
    vi.stubGlobal('cancelAnimationFrame', vi.fn());

    const id = 'stream-r2';
    useChatStore.getState().appendStreamDelta(reasoningDelta(id, 'reasoning'));
    useChatStore.getState().finalizeStream(id);

    const msg = useChatStore.getState().messages.find((m) => m.id === id);
    expect(msg?.metadata?.[REASONING_TEXT_METADATA_KEY]).toBe('reasoning');
  });
});
