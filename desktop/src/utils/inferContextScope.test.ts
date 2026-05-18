import { describe, expect, it } from 'vitest';
import { resolveContextScope } from './inferContextScope';

describe('resolveContextScope', () => {
  it('off mode returns none', () => {
    expect(resolveContextScope({ message: 'review main.go', mode: 'off', channelKind: 'general' }).scope).toBe('none');
  });

  it('always mode returns full', () => {
    expect(resolveContextScope({ message: 'hello', mode: 'always', channelKind: 'general' }).scope).toBe('full');
  });

  it('general AWS question returns none', () => {
    const r = resolveContextScope({
      message: 'What is AWS SSO and how do I use it in our dev account?',
      mode: 'auto',
      channelKind: 'general',
    });
    expect(r.scope).toBe('none');
  });

  it('path in message returns focus', () => {
    const r = resolveContextScope({
      message: 'Please review internal/hub/hub.go for race conditions',
      mode: 'auto',
      channelKind: 'general',
    });
    expect(r.scope).toBe('focus');
  });

  it('collab social question returns hint', () => {
    const r = resolveContextScope({
      message: '@Gemini @Cursor who is the better rust programmer?',
      mode: 'auto',
      channelKind: 'collaboration',
    });
    expect(r.scope).toBe('hint');
  });

  it('collab with path returns focus not full', () => {
    const r = resolveContextScope({
      message: 'review src/main.rs for bugs',
      mode: 'auto',
      channelKind: 'collaboration',
    });
    expect(r.scope).toBe('focus');
  });

  it('architecture question returns outline', () => {
    const r = resolveContextScope({
      message: 'What is the architecture of this repo?',
      mode: 'auto',
      channelKind: 'general',
    });
    expect(r.scope).toBe('outline');
  });

  it('manual override wins', () => {
    expect(
      resolveContextScope({
        message: 'hi',
        mode: 'auto',
        channelKind: 'general',
        messageOverride: 'full',
      }).scope
    ).toBe('full');
  });

  it('new document open without path returns focus', () => {
    const r = resolveContextScope({
      message: 'I have a new document open, can you review?',
      mode: 'auto',
      channelKind: 'general',
    });
    expect(r.scope).toBe('focus');
  });

  it('review typo with active tab returns focus', () => {
    const r = resolveContextScope({
      message: 'can you reivew what I have open?',
      mode: 'auto',
      channelKind: 'general',
      activeTabPath: '/Users/me/proj/rfc.md',
    });
    expect(r.scope).toBe('focus');
  });

  it('ambiguous chat without editor signals stays hint', () => {
    const r = resolveContextScope({
      message: 'thoughts on our roadmap for Q3?',
      mode: 'auto',
      channelKind: 'general',
    });
    expect(r.scope).toBe('hint');
  });
});
