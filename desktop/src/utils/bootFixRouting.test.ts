import { describe, expect, it } from 'vitest';
import { hasBootFixRoutingSignals } from './bootFixRouting';
import { prepareOutboundPayload } from './prepareOutboundPayload';

describe('hasBootFixRoutingSignals', () => {
  it('detects boot-fix messages', () => {
    expect(hasBootFixRoutingSignals('the app is not booting up can you fix it?')).toBe(true);
    expect(hasBootFixRoutingSignals('make start-all fails')).toBe(true);
    expect(hasBootFixRoutingSignals('what is rust?')).toBe(false);
  });
});

describe('desktop boot-fix routing metadata', () => {
  it('leaves boot-fix semantics to the hub', async () => {
    const { metadata } = await prepareOutboundPayload({
      content:
        'Something is wrong with this code I am working on and the app will not boot up, can you sort me out here?',
      composerMode: 'agent',
      agents: [
        { id: '1', name: 'SoftwareArchitect', type: 'architecture' },
        { id: '2', name: 'FrontendEngineer', type: 'frontend' },
      ],
      activeTab: null,
      editorAgentTrust: 'auto_apply_edits',
    });
    expect(metadata.ide_route_agent_type).toBeUndefined();
    expect(metadata.implementation_session).toBeUndefined();
  });

  it('preserves the explicit DM partner without a boot-fix override', async () => {
    const { metadata } = await prepareOutboundPayload({
      content: 'the app is not booting up can you help?',
      composerMode: 'agent',
      agents: [
        { id: '1', name: 'SoftwareArchitect', type: 'architecture' },
        { id: '2', name: 'FrontendEngineer', type: 'frontend' },
      ],
      activeTab: null,
      editorAgentTrust: 'auto_apply_edits',
      channel: 'dm-user-softwarearchitect',
      channelMeta: {
        type: 'dm',
        agents: ['1'],
        name: 'dm-user-softwarearchitect',
        description: '',
      },
    });
    expect(metadata.ide_route_agent_type).toBe('architecture');
    expect(metadata.implementation_session).toBeUndefined();
  });
});
