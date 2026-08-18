import { describe, expect, it } from 'vitest';
import { resolveContextScope, scopeFromContextRequest } from './inferContextScope';

describe('resolveContextScope', () => {
  it('off mode returns none', () => {
    expect(resolveContextScope({ message: 'review main.go', mode: 'off', channelKind: 'general' }).scope).toBe('none');
  });

  it('always mode returns full', () => {
    expect(resolveContextScope({ message: 'hello', mode: 'always', channelKind: 'general' }).scope).toBe('full');
  });

  it('/collaborate returns outline workspace (tree only)', () => {
    const r = resolveContextScope({
      message: '/collaborate @Gemini @Assistant investigate schemas',
      mode: 'auto',
      channelKind: 'general',
    });
    expect(r.scope).toBe('outline');
    expect(r.reason).toContain('collaboration');
  });

  it('prepare envelope defaults to structural hint (not NL phrase focus)', () => {
    const r = resolveContextScope({
      message: 'What is AWS SSO and how do I use it in our dev account?',
      mode: 'auto',
      channelKind: 'general',
    });
    expect(r.scope).toBe('hint');
    expect(r.reason).toContain('prepare envelope');
  });

  it('explicit path tokens stay prepare hint (hub context_request upgrades)', () => {
    const r = resolveContextScope({
      message: 'Please review internal/hub/hub.go for race conditions',
      mode: 'auto',
      channelKind: 'general',
    });
    expect(r.scope).toBe('hint');
  });

  it('collab social question returns hint', () => {
    const r = resolveContextScope({
      message: '@Gemini @Cursor who is the better rust programmer?',
      mode: 'auto',
      channelKind: 'collaboration',
    });
    expect(r.scope).toBe('hint');
  });

  it('hub stamp tier overrides prepare envelope', () => {
    const r = resolveContextScope({
      message: 'please reivew the documents in the workspace',
      mode: 'auto',
      channelKind: 'general',
      stampContextTier: 'outline',
    });
    expect(r.scope).toBe('outline');
    expect(r.reason).toContain('stamp');
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

  it('IDE with active tab uses prepare hint until stamp fetch', () => {
    const r = resolveContextScope({
      message: 'can you reivew what I have open?',
      mode: 'auto',
      channelKind: 'general',
      activeTabPath: '/Users/me/proj/rfc.md',
      ideCoding: true,
    });
    expect(r.scope).toBe('hint');
  });

  it('scan tool + open tab stays prepare hint', () => {
    const r = resolveContextScope({
      message: 'use summarize_scan_analysis on the file I have open',
      mode: 'auto',
      channelKind: 'dm',
      activeTabPath: '/data/plate-1/reports/results.json',
    });
    expect(r.scope).toBe('hint');
    expect(r.reason).toContain('scan');
  });
});

describe('scopeFromContextRequest', () => {
  it('maps hub tiers and include flags', () => {
    expect(scopeFromContextRequest({ context_tier: 'outline' })).toBe('outline');
    expect(scopeFromContextRequest({ include_document_bodies: true })).toBe('full');
    expect(scopeFromContextRequest({ include_active_tab: true })).toBe('focus');
    expect(scopeFromContextRequest({ include_file_tree: true })).toBe('outline');
  });
});
