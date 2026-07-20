import { describe, expect, it } from 'vitest';
import {
  cycleConversationModeSetting,
  formatContextIndicator,
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

  it('clarifies weak verb + question without a path', () => {
    expect(isConversationModeAmbiguous('How do I update my AWS SSO credentials?')).toBe(true);
    expect(inferResolvedConversationMode('How do I update my AWS SSO credentials?')).toBe('clarify');
  });

  it('infers code for review verbs with paths', () => {
    expect(inferResolvedConversationMode('review cmd/server/main.go')).toBe('code');
  });

  it('infers code for scan tool requests', () => {
    expect(
      inferResolvedConversationMode('Use summarize_scan_analysis on the file I have open please')
    ).toBe('code');
  });

  it('honors explicit chat setting over ambiguity', () => {
    expect(resolveConversationMode('chat', 'How do I update my SSO?')).toBe('chat');
    expect(resolveConversationMode('code', 'How do I update my SSO?')).toBe('code');
  });

  it('formats context indicator including clarify', () => {
    const label = formatContextIndicator({
      modeSetting: 'auto',
      resolvedMode: 'clarify',
      scope: 'hint',
    });
    expect(label).toContain('Auto→clarify');
    expect(label).toContain('hint');
  });
});
