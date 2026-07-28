import { describe, expect, it } from 'vitest';
import {
  cycleConversationModeSetting,
  formatContextIndicator,
  hasStrongCodeTaskSignals,
  inferResolvedConversationMode,
  isConversationModeAmbiguous,
  resolveConversationMode,
} from './conversationMode';

describe('conversationMode', () => {
  it('cycles auto → chat → code → auto', () => {
    expect(cycleConversationModeSetting('auto')).toBe('chat');
    expect(cycleConversationModeSetting('chat')).toBe('code');
    expect(cycleConversationModeSetting('code')).toBe('auto');
  });

  it('Auto defaults to chat (no NL verb inference)', () => {
    expect(inferResolvedConversationMode('hello')).toBe('chat');
    expect(inferResolvedConversationMode('@Assistant hi')).toBe('chat');
    expect(
      inferResolvedConversationMode('What is AWS SSO and how do I use it in our dev account?')
    ).toBe('chat');
    expect(inferResolvedConversationMode('review cmd/server/main.go')).toBe('chat');
    expect(
      inferResolvedConversationMode('Use summarize_scan_analysis on the file I have open please')
    ).toBe('chat');
    expect(isConversationModeAmbiguous('How do I update my AWS SSO credentials?')).toBe(false);
  });

  it('infers chat for @here / social pings even in IDE layout', () => {
    expect(
      inferResolvedConversationMode('@here whats going on!?!', { ideCoding: true })
    ).toBe('chat');
    expect(inferResolvedConversationMode('@here', { ideCoding: true })).toBe('chat');
  });

  it('still forces code in IDE for real tasks', () => {
    expect(
      inferResolvedConversationMode('refactor cmd/server/main.go', { ideCoding: true })
    ).toBe('code');
  });

  it('detects structural @codebase / path signals without forcing Auto→code', () => {
    expect(hasStrongCodeTaskSignals('look at @codebase')).toBe(true);
    expect(hasStrongCodeTaskSignals('review cmd/server/main.go')).toBe(true);
    expect(inferResolvedConversationMode('look at @codebase')).toBe('chat');
  });

  it('honors explicit chat/code settings', () => {
    expect(resolveConversationMode('chat', 'How do I update my SSO?')).toBe('chat');
    expect(resolveConversationMode('code', 'How do I update my SSO?')).toBe('code');
  });

  it('formats context indicator', () => {
    const label = formatContextIndicator({
      modeSetting: 'auto',
      resolvedMode: 'chat',
      scope: 'hint',
    });
    expect(label).toContain('Auto→chat');
    expect(label).toContain('hint');
  });
});
