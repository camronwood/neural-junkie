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
      devPackEnabled: true,
    });
    expect(content).toBe('save it to docs/test.md');
    expect(metadata.editor_mode).toBe('export');
    expect(metadata.conversation_mode).toBe('code');
    expect(metadata.implementation_session).toBe(true);
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
});
