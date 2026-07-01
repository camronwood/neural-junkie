import { describe, expect, it } from 'vitest';
import { parseDMDisplayName } from './dmChannelDisplay';
import type { Channel } from '../types/protocol';

describe('parseDMDisplayName', () => {
  it('prefers channel display_name over slug parsing', () => {
    const ch: Channel = {
      id: '1',
      name: 'dm-user-guitar',
      type: 'dm',
      description: 'Direct message with GuitarExpert',
      display_name: 'My Guitar Coach',
      created: new Date().toISOString(),
    };
    expect(parseDMDisplayName(ch)).toBe('My Guitar Coach');
  });

  it('falls back to agent name when display_name is absent', () => {
    const ch: Channel = {
      id: '1',
      name: 'dm-user-guitar',
      type: 'dm',
      description: 'Direct message with GuitarExpert',
      agents: [{ id: 'a1', name: 'GuitarExpert', type: 'expert' }],
      created: new Date().toISOString(),
    };
    expect(parseDMDisplayName(ch)).toBe('GuitarExpert');
  });
});
