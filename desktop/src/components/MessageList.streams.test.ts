import { describe, expect, it } from 'vitest';
import { streamingMessagesForChannel } from './MessageList';
import type { Message } from '../types/protocol';

describe('streamingMessagesForChannel', () => {
  const base = {
    type: 'chat' as const,
    from: { id: 'a', name: 'Agent', type: 'cli' as const },
    content: 'x',
    timestamp: new Date().toISOString(),
  };

  it('keeps streams for the active channel and channel-less streams', () => {
    const streaming: Record<string, Message> = {
      a: { ...base, id: 'a', channel: 'general' },
      b: { ...base, id: 'b', channel: 'reviews' },
      c: { ...base, id: 'c' },
    };
    const active = streamingMessagesForChannel(streaming, 'general');
    expect(active.map((m) => m.id).sort()).toEqual(['a', 'c']);
  });

  it('excludes other-channel streams (failure path)', () => {
    const streaming: Record<string, Message> = {
      a: { ...base, id: 'a', channel: 'general' },
      b: { ...base, id: 'b', channel: 'reviews' },
    };
    expect(streamingMessagesForChannel(streaming, 'reviews').map((m) => m.id)).toEqual(['b']);
  });
});
