import { describe, expect, it } from 'vitest';
import { buildHumanOutboundMetadata } from './outboundChatMetadata';
import { hasImplementationContinuationSignals } from './implementationContinuation';

describe('implementationContinuation', () => {
  it('detects go-ahead affirmations', () => {
    expect(hasImplementationContinuationSignals('yes please go ahead')).toBe(true);
    expect(hasImplementationContinuationSignals('ok please do it now')).toBe(true);
    expect(hasImplementationContinuationSignals('what?')).toBe(false);
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
});
