import { describe, expect, it } from 'vitest';
import { hasBootFixRoutingSignals } from './bootFixRouting';
import { buildImplementationSessionMetadata } from './ideComposer';

describe('hasBootFixRoutingSignals', () => {
  it('detects boot-fix messages', () => {
    expect(hasBootFixRoutingSignals('the app is not booting up can you fix it?')).toBe(true);
    expect(hasBootFixRoutingSignals('make start-all fails')).toBe(true);
    expect(hasBootFixRoutingSignals('what is rust?')).toBe(false);
  });
});

describe('buildImplementationSessionMetadata boot-fix routing', () => {
  it('routes boot-fix to frontend without mention', () => {
    const metadata = buildImplementationSessionMetadata({
      content: 'the app is not booting up can you help?',
      agents: [
        { id: '1', name: 'SoftwareArchitect', type: 'architecture' },
        { id: '2', name: 'FrontendEngineer', type: 'frontend' },
      ],
      activeTab: null,
      editorAgentMode: 'agent',
      editorAgentTrust: 'auto_apply_edits',
    });
    expect(metadata.ide_route_agent_type).toBe('frontend');
  });
});
