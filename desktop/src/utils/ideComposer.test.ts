import { describe, expect, it } from 'vitest';
import {
  buildIdeDispatchPayload,
  buildImplementationSessionMetadata,
  pickAgentTypeForImplementation,
} from './ideComposer';

describe('buildIdeDispatchPayload', () => {
  it('does not infer implementation_session from natural language', () => {
    const { metadata } = buildIdeDispatchPayload({
      content: 'please implement theme support',
      agents: [],
      activeTab: { path: 'src/App.tsx', language: 'typescript' } as never,
      editorAgentMode: 'agent',
      editorAgentTrust: 'auto_apply_edits',
    });
    expect(metadata.implementation_session).toBeUndefined();
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
  it('sets explicit editor metadata and tab-based route without semantic inference', () => {
    const metadata = buildImplementationSessionMetadata({
      content: 'implement light/dark theme toggle in the sidebar',
      agents: [{ name: 'FrontendEngineer', type: 'frontend' } as never],
      activeTab: { path: 'src/App.tsx', language: 'typescript' } as never,
      editorAgentMode: 'agent',
      editorAgentTrust: 'auto_apply_edits',
    });
    expect(metadata.implementation_session).toBeUndefined();
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
    expect(metadata.implementation_session).toBeUndefined();
  });

  it('preserves explicit implementation_session from composer metadata', () => {
    const metadata = buildImplementationSessionMetadata({
      content: 'approved',
      agents: [{ name: 'FrontendEngineer', type: 'frontend' } as never],
      activeTab: null,
      editorAgentMode: 'agent',
      editorAgentTrust: 'interactive',
      composerMetadata: { implementation_session: true },
    });
    expect(metadata.implementation_session).toBe(true);
  });

  it('does not set implementation_session for affirmations without explicit flag', () => {
    const metadata = buildImplementationSessionMetadata({
      content: 'approved',
      agents: [{ name: 'FrontendEngineer', type: 'frontend' } as never],
      activeTab: null,
      editorAgentMode: 'agent',
      editorAgentTrust: 'interactive',
    });
    expect(metadata.implementation_session).toBeUndefined();
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
