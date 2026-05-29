import { describe, expect, it } from 'vitest';
import {
  cycleConversationModeSetting,
  formatContextIndicator,
  inferResolvedConversationMode,
  resolveConversationMode,
} from './conversationMode';

describe('conversationMode', () => {
  it('cycles auto → chat → code → auto', () => {
    expect(cycleConversationModeSetting('auto')).toBe('chat');
    expect(cycleConversationModeSetting('chat')).toBe('code');
    expect(cycleConversationModeSetting('code')).toBe('auto');
  });

  it('infers chat for greetings', () => {
    expect(inferResolvedConversationMode('hello')).toBe('chat');
    expect(inferResolvedConversationMode('@Assistant hi')).toBe('chat');
  });

  it('infers code for review verbs', () => {
    expect(inferResolvedConversationMode('review cmd/server/main.go')).toBe('code');
  });

  it('honors explicit chat setting', () => {
    expect(resolveConversationMode('chat', 'review main.go')).toBe('chat');
  });

  it('formats context indicator', () => {
    const label = formatContextIndicator({
      modeSetting: 'auto',
      resolvedMode: 'chat',
      scope: 'none',
    });
    expect(label).toContain('Auto→chat');
    expect(label).toContain('no files');
  });
});
