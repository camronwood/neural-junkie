import { describe, expect, it } from 'vitest';
import {
  applyIdeAskPrefix,
  ensureIdeAgentMention,
  messageRequestsCodebase,
  pickAgentTypeFromTab,
} from './ideComposer';

describe('ideComposer', () => {
  it('picks backend for go files', () => {
    expect(pickAgentTypeFromTab({ path: 'cmd/main.go', language: 'go' } as never)).toBe('backend');
  });

  it('auto-mentions specialist when none present', () => {
    expect(ensureIdeAgentMention('fix the bug', 'BackendEngineer')).toBe(
      '@BackendEngineer fix the bug'
    );
    expect(ensureIdeAgentMention('@FrontendEngineer hi', 'BackendEngineer')).toBe(
      '@FrontendEngineer hi'
    );
  });

  it('prefixes ask mode once', () => {
    const out = applyIdeAskPrefix('explain this', 'ask');
    expect(out).toContain('[ASK mode');
    expect(applyIdeAskPrefix(out, 'ask')).toBe(out);
  });

  it('detects @codebase', () => {
    expect(messageRequestsCodebase('search @codebase for auth')).toBe(true);
    expect(messageRequestsCodebase('hello')).toBe(false);
  });
});
