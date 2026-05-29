import { describe, expect, it } from 'vitest';
import {
  applyIdeAskPrefix,
  buildIdeDispatchPayload,
  ensureIdeAgentMention,
  messageRequestsCodebase,
  pickAgentTypeFromTab,
} from './ideComposer';
import { IDE_ROUTE_AGENT_TYPE_KEY } from '../constants/promptMetadata';

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

  it('does not set IDE route when user already @mentions an agent', () => {
    const payload = buildIdeDispatchPayload({
      content: '@Assistant hello',
      agents: [
        { id: 'a1', name: 'Assistant', type: 'assistant' } as never,
        { id: 'b1', name: 'BackendEngineer', type: 'backend' } as never,
      ],
      activeTab: { path: 'cmd/main.go', language: 'go' } as never,
      editorAgentMode: 'agent',
      editorAgentTrust: 'default',
    });
    expect(payload.content).toBe('@Assistant hello');
    expect(payload.metadata[IDE_ROUTE_AGENT_TYPE_KEY]).toBeUndefined();
  });

  it('sets IDE route for implicit specialist routing', () => {
    const payload = buildIdeDispatchPayload({
      content: 'fix the bug',
      agents: [{ id: 'b1', name: 'BackendEngineer', type: 'backend' } as never],
      activeTab: { path: 'cmd/main.go', language: 'go' } as never,
      editorAgentMode: 'agent',
      editorAgentTrust: 'default',
    });
    expect(payload.metadata[IDE_ROUTE_AGENT_TYPE_KEY]).toBe('backend');
  });
});
