import { describe, expect, it } from 'vitest';
import { isSlashCommandMessage } from '../types/protocol';
import type { Message } from '../types/protocol';

describe('isSlashCommandMessage', () => {
  it('detects metadata flag', () => {
    const msg: Message = {
      id: '1',
      type: 'question',
      channel: 'general',
      from: { id: 'u', name: 'Camron', type: 'human' },
      content: '/help',
      timestamp: new Date().toISOString(),
      metadata: { slash_command: true },
    };
    expect(isSlashCommandMessage(msg)).toBe(true);
  });

  it('detects human lines starting with /', () => {
    const msg: Message = {
      id: '2',
      type: 'question',
      channel: 'general',
      from: { id: 'u', name: 'Camron', type: 'human' },
      content: '/collaborate build feature',
      timestamp: new Date().toISOString(),
    };
    expect(isSlashCommandMessage(msg)).toBe(true);
  });

  it('ignores system responses', () => {
    const msg: Message = {
      id: '3',
      type: 'system_info',
      channel: 'general',
      from: { id: 'system', name: 'System', type: 'general' },
      content: 'Available Commands',
      timestamp: new Date().toISOString(),
    };
    expect(isSlashCommandMessage(msg)).toBe(false);
  });
});
