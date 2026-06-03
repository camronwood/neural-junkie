import { describe, expect, it } from 'vitest';
import {
  buildIdeDispatchPayload,
  buildImplementationSessionMetadata,
  pickAgentTypeForImplementation,
} from './ideComposer';

describe('buildIdeDispatchPayload', () => {
  it('sets implementation_session for agent mode code tasks', () => {
    const { metadata } = buildIdeDispatchPayload({
      content: 'please implement theme support',
      agents: [],
      activeTab: { path: 'src/App.tsx', language: 'typescript' } as never,
      editorAgentMode: 'agent',
      editorAgentTrust: 'auto_apply_edits',
    });
    expect(metadata.implementation_session).toBe(true);
    expect(metadata.editor_mode).toBe('agent');
  });

  it('does not set implementation_session in ask mode', () => {
    const { metadata } = buildIdeDispatchPayload({
      content: 'please implement theme support',
      agents: [],
      activeTab: null,
      editorAgentMode: 'ask',
      editorAgentTrust: 'interactive',
    });
    expect(metadata.implementation_session).toBeUndefined();
  });
});
