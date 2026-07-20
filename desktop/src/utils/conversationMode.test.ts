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

  it('infers code for knowledge-graph relate questions', () => {
    expect(
      inferResolvedConversationMode(
        "How does CISO relate to the rest of the codebase? CISO (repo) in community 'root' — degree 1, 1 neighbors"
      )
    ).toBe('code');
  });

  it('keeps general questions as chat', () => {
    expect(
      inferResolvedConversationMode('What is AWS SSO and how do I use it in our dev account?')
    ).toBe('chat');
  });

  it('infers code for review verbs', () => {
    expect(inferResolvedConversationMode('review cmd/server/main.go')).toBe('code');
  });

  it('infers code for scan tool requests', () => {
    expect(
      inferResolvedConversationMode('Use summarize_scan_analysis on the file I have open please')
    ).toBe('code');
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
