import { describe, expect, it } from 'vitest';
import { buildHumanOutboundMetadata } from './outboundChatMetadata';
import {
  hasImplementationContinuationSignals,
  hasImplementationRequestSignals,
} from './implementationContinuation';

describe('implementationContinuation', () => {
  it('detects go-ahead affirmations', () => {
    expect(hasImplementationContinuationSignals('yes please go ahead')).toBe(true);
    expect(hasImplementationContinuationSignals('ok please do it now')).toBe(true);
    expect(hasImplementationContinuationSignals('approved')).toBe(true);
    expect(hasImplementationContinuationSignals('please keep going')).toBe(true);
    expect(hasImplementationContinuationSignals('what?')).toBe(false);
    expect(hasImplementationContinuationSignals('ok goahead')).toBe(true);
    expect(hasImplementationContinuationSignals('looks good')).toBe(false);
    expect(hasImplementationContinuationSignals('ok')).toBe(false);
  });

  it('detects workspace directive as implementation request', () => {
    expect(
      hasImplementationRequestSignals('use the open workspace it has all the files you need')
    ).toBe(true);
  });

  it('detects implementation request phrases', () => {
    expect(
      hasImplementationRequestSignals(
        'yesterday we were working on adding a settings modal for font size and themes dark/light, pick up where we left off'
      )
    ).toBe(true);
    expect(hasImplementationRequestSignals('hello there')).toBe(false);
  });
});

describe('buildHumanOutboundMetadata continuation', () => {
  it('does not force scope none on chat mode when workspace is always', () => {
    const meta = buildHumanOutboundMetadata({
      contextMode: 'always',
      message: 'thoughts on lunch?',
      channel: 'general',
      channelType: 'public',
    });
    expect(meta?.context_scope).toBe('full');
  });

  it('DM settings modal request uses code mode and outline/focus scope', () => {
    const meta = buildHumanOutboundMetadata({
      contextMode: 'always',
      message:
        'adding a settings modal with font size and themes dark/light, pick up where we left off',
      channel: 'dm-camron-frontendengineer',
      channelType: 'dm',
    });
    expect(meta?.conversation_mode).toBe('code');
    expect(['outline', 'focus']).toContain(meta?.context_scope);
  });

  it('DM workspace directive uses code mode', () => {
    const meta = buildHumanOutboundMetadata({
      contextMode: 'always',
      message: 'use the open workspace it has all the files you need',
      channel: 'dm-camron-frontendengineer',
      channelType: 'dm',
    });
    expect(meta?.conversation_mode).toBe('code');
  });
});
