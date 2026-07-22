import { describe, expect, it } from 'vitest';
import { prepareOutboundPayload } from './prepareOutboundPayload';

describe('prepareOutboundPayload', () => {
  it('explicit export mode grants implementation permission', async () => {
    const { content, metadata } = await prepareOutboundPayload({
      content: 'save it to docs/test.md',
      composerMode: 'export',
      agents: [],
      activeTab: null,
      editorAgentTrust: 'interactive',
      ideEnabled: true,
    });
    expect(content).toBe('save it to docs/test.md');
    expect(metadata.editor_mode).toBe('export');
    expect(metadata.conversation_mode).toBeUndefined();
    expect(metadata.implementation_session).toBe(true);
  });

  it('does not infer export mode from save-to-file language', async () => {
    const { metadata } = await prepareOutboundPayload({
      content: 'the artical you wrote a few messages back please save it to a markdown file',
      composerMode: 'agent',
      agents: [],
      activeTab: null,
      editorAgentTrust: 'interactive',
      ideEnabled: true,
    });
    expect(metadata.editor_mode).toBe('agent');
    expect(metadata.implementation_session).toBeUndefined();
    expect(metadata.ide_route_agent_type).toBeUndefined();
  });

  it('combined write+save stays on agent metadata for chat-first path', async () => {
    const { metadata } = await prepareOutboundPayload({
      content:
        'Can you write me a LinkedIn artical about the app and save the file to the workspace root?',
      composerMode: 'agent',
      agents: [],
      activeTab: null,
      editorAgentTrust: 'interactive',
      ideEnabled: true,
    });
    expect(metadata.editor_mode).toBe('agent');
    expect(metadata.implementation_session).toBeUndefined();
  });

  it('does not infer implementation or recipient metadata from boot-fix language', async () => {
    const { metadata } = await prepareOutboundPayload({
      content:
        'Something is wrong with this code I am working on and the app will not boot up, can you sort me out here?',
      composerMode: 'agent',
      agents: [{ id: 'frontend-1', name: 'FrontendEngineer', type: 'frontend' }],
      activeTab: null,
      editorAgentTrust: 'auto_apply_edits',
      ideEnabled: true,
    });
    expect(metadata.editor_mode).toBe('agent');
    expect(metadata.editor_agent_trust).toBe('auto_apply_edits');
    expect(metadata.ide_route_agent_type).toBeUndefined();
    expect(metadata.implementation_session).toBeUndefined();
  });

  it('ask mode prefixes read-only banner', async () => {
    const { content, metadata } = await prepareOutboundPayload({
      content: 'what does main.go do?',
      composerMode: 'ask',
      agents: [],
      activeTab: null,
      editorAgentTrust: 'interactive',
    });
    expect(content).toContain('[ASK mode');
    expect(metadata.editor_mode).toBe('ask');
    expect(metadata.editor_agent_trust).toBe('interactive');
    expect(metadata.implementation_session).toBeUndefined();
  });

  it('preserves an explicit DM recipient without overriding an explicit mention', async () => {
    const agents = [{ id: 'architect-1', name: 'SoftwareArchitect', type: 'architecture' }];
    const channelMeta = {
      type: 'dm' as const,
      agents: ['architect-1'],
      name: 'dm-camron-softwarearchitect',
      description: '',
    };
    const dm = await prepareOutboundPayload({
      content: 'Can you help?',
      composerMode: 'agent',
      agents,
      activeTab: null,
      editorAgentTrust: 'yolo',
      channel: channelMeta.name,
      channelMeta,
    });
    expect(dm.metadata.ide_route_agent_type).toBe('architecture');
    expect(dm.metadata.editor_agent_trust).toBe('yolo');

    const mentioned = await prepareOutboundPayload({
      content: '@FrontendEngineer can you help?',
      composerMode: 'agent',
      agents,
      activeTab: null,
      editorAgentTrust: 'yolo',
      channel: channelMeta.name,
      channelMeta,
    });
    expect(mentioned.metadata.ide_route_agent_type).toBeUndefined();
  });

  it('preserves explicit attachments, mentions, and reply identifiers', async () => {
    const { metadata } = await prepareOutboundPayload({
      content: '@BackendEngineer review the attachment',
      composerMode: 'plan',
      agents: [],
      activeTab: null,
      editorAgentTrust: 'yolo',
      composerMetadata: {
        prompt_attachments: [{ path: 'src/main.rs', language: 'rust', content: 'fn main() {}' }],
        mentions: ['backend-1'],
        reply_to: 'message-1',
        reply_message_id: 'message-1',
      },
    });

    expect(metadata.prompt_attachments).toHaveLength(1);
    expect(metadata.mentions).toEqual(['backend-1']);
    expect(metadata.reply_to).toBe('message-1');
    expect(metadata.reply_message_id).toBe('message-1');
    expect(metadata.editor_mode).toBe('plan');
    expect(metadata.editor_agent_trust).toBe('interactive');
  });

  it('slash commands skip ask/plan prefixes and IDE routing metadata', async () => {
    const { content, metadata } = await prepareOutboundPayload({
      content: '/create-expert ios MyExpert ollama gemma3:12b',
      composerMode: 'ask',
      agents: [],
      activeTab: null,
      editorAgentTrust: 'interactive',
      ideEnabled: true,
    });
    expect(content).toBe('/create-expert ios MyExpert ollama gemma3:12b');
    expect(metadata.editor_mode).toBeUndefined();
    expect(metadata.ide_route_agent_type).toBeUndefined();
    expect(metadata.implementation_session).toBeUndefined();
  });
});
