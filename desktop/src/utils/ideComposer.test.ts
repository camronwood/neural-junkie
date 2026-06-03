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

describe('buildImplementationSessionMetadata', () => {
  it('sets session metadata for team-channel style sends', () => {
    const metadata = buildImplementationSessionMetadata({
      content: 'implement light/dark theme toggle in the sidebar',
      agents: [{ name: 'FrontendEngineer', type: 'frontend' } as never],
      activeTab: null,
      editorAgentMode: 'agent',
      editorAgentTrust: 'auto_apply_edits',
    });
    expect(metadata.implementation_session).toBe(true);
    expect(metadata.editor_mode).toBe('agent');
    expect(metadata.ide_route_agent_type).toBe('frontend');
  });

  it('skips ide_route when user @mentions an agent', () => {
    const metadata = buildImplementationSessionMetadata({
      content: '@BackendEngineer please implement a handler',
      agents: [{ name: 'BackendEngineer', type: 'backend' } as never],
      activeTab: { path: 'src/App.tsx', language: 'typescript' } as never,
      editorAgentMode: 'agent',
      editorAgentTrust: 'interactive',
    });
    expect(metadata.ide_route_agent_type).toBeUndefined();
    expect(metadata.implementation_session).toBe(true);
  });
});

describe('pickAgentTypeForImplementation', () => {
  it('uses @mention agent type when present', () => {
    const t = pickAgentTypeForImplementation(
      '@FrontendEngineer implement themes',
      null,
      [{ name: 'FrontendEngineer', type: 'frontend' } as never]
    );
    expect(t).toBe('frontend');
  });
});
