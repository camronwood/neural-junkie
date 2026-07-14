import { describe, expect, it } from 'vitest';
import { prepareOutboundPayload } from './prepareOutboundPayload';

describe('prepareOutboundPayload', () => {
  it('export mode sets code conversation and implementation session', async () => {
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
    expect(metadata.conversation_mode).toBe('code');
    expect(metadata.implementation_session).toBe(true);
  });

  it('auto-routes agent chip to export metadata for save-to-file messages', async () => {
    const { metadata } = await prepareOutboundPayload({
      content: 'the artical you wrote a few messages back please save it to a markdown file',
      composerMode: 'agent',
      agents: [],
      activeTab: null,
      editorAgentTrust: 'interactive',
      ideEnabled: true,
    });
    expect(metadata.editor_mode).toBe('export');
    expect(metadata.implementation_session).toBe(true);
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
    expect(metadata.implementation_session).toBeUndefined();
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
