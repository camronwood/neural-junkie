import { describe, expect, it } from 'vitest';
import type { Collaboration, Message } from '../types/protocol';
import { registeredFileChangeId, shouldPromptFileChangeApproval } from './fileChangeApprovalPrompt';

function fileChangeMessage(overrides: Partial<Message> = {}): Message {
  return {
    id: 'msg-1',
    type: 'file_change',
    channel: 'collab-abc',
    content: 'Proposing file',
    from: { id: 'a1', name: 'Agent', type: 'backend' },
    timestamp: new Date().toISOString(),
    metadata: { registered_change_id: 'change-1' },
    ...overrides,
  };
}

describe('shouldPromptFileChangeApproval', () => {
  it('prompts on the active channel', () => {
    expect(
      shouldPromptFileChangeApproval(
        fileChangeMessage({ channel: 'general' }),
        'general',
        {},
      ),
    ).toBe(true);
  });

  it('prompts for executing collab channels even when not active', () => {
    const collabs: Record<string, Collaboration> = {
      c1: {
        id: 'c1',
        title: 'Run',
        phase: 'executing',
        channel: 'collab-abc',
        agents: [],
        created_at: '',
        updated_at: '',
      },
    };
    expect(
      shouldPromptFileChangeApproval(
        fileChangeMessage({ channel: 'collab-abc' }),
        'general',
        collabs,
      ),
    ).toBe(true);
  });

  it('skips unrelated channels', () => {
    expect(
      shouldPromptFileChangeApproval(
        fileChangeMessage({ channel: 'other' }),
        'general',
        {},
      ),
    ).toBe(false);
  });
});

describe('registeredFileChangeId', () => {
  it('reads registered_change_id from metadata', () => {
    expect(registeredFileChangeId(fileChangeMessage())).toBe('change-1');
  });
});
