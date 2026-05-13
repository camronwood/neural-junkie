import { describe, expect, it } from 'vitest';
import {
  capStreamContent,
  MAX_STREAM_CONTENT_CHARS,
  trimMessagesToMax,
} from './messageLimits';
import type { Message } from '../types/protocol';

function msg(id: string, content: string): Message {
  return {
    id,
    type: 'chat',
    channel: 'general',
    from: {
      id: 'u',
      name: 'u',
      type: 'human',
      expertise: [],
      status: 'active',
      model: '',
      is_paused: false,
    },
    content,
    timestamp: new Date().toISOString(),
  };
}

describe('trimMessagesToMax', () => {
  it('returns same array when under cap', () => {
    const m = [msg('1', 'a'), msg('2', 'b')];
    expect(trimMessagesToMax(m, 500)).toBe(m);
  });

  it('keeps newest messages', () => {
    const arr = Array.from({ length: 10 }, (_, i) => msg(`id-${i}`, `${i}`));
    const out = trimMessagesToMax(arr, 3);
    expect(out.map((m) => m.content)).toEqual(['7', '8', '9']);
  });
});

describe('capStreamContent', () => {
  it('does not change short strings', () => {
    expect(capStreamContent('hello')).toBe('hello');
  });

  it('truncates past cap', () => {
    const s = 'x'.repeat(MAX_STREAM_CONTENT_CHARS + 1000);
    const out = capStreamContent(s);
    expect(out.length).toBeLessThanOrEqual(MAX_STREAM_CONTENT_CHARS);
    expect(out).toContain('truncated');
  });
});
