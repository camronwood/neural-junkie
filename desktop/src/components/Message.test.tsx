// @vitest-environment jsdom
import type { ImgHTMLAttributes } from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { cleanup, render, screen } from '@testing-library/react';
import type { Message as ChatMessage } from '../types/protocol';
import { useChatStore } from '../stores/chatStore';
import { useEditorStore } from '../stores/editorStore';
import { useFileChangeStore } from '../stores/fileChangeStore';
import { useFileExplorerStore } from '../stores/fileExplorerStore';
import { useSettingsStore } from '../stores/settingsStore';

vi.mock('./MessageContent', () => ({
  MessageContent: ({ content }: { content: string }) => <div>{content}</div>,
}));

vi.mock('./CommandOutput', () => ({ CommandOutput: () => null }));
vi.mock('./DesignOutput', () => ({ DesignOutput: () => null }));
vi.mock('./ToolApprovalCard', () => ({ ToolApprovalCard: () => null }));
vi.mock('./UserQuestionCard', () => ({ UserQuestionCard: () => null }));
vi.mock('./ImplementationSessionOutcomeCard', () => ({
  ImplementationSessionOutcomeCard: () => null,
  parseImplementationSessionOutcome: () => null,
}));
vi.mock('./ChangeProposalCard', () => ({ ChangeProposalMessageCard: () => null }));
vi.mock('./ImageLightboxModal', () => ({
  ChatClickableImage: (props: ImgHTMLAttributes<HTMLImageElement>) => <img alt={props.alt} />,
}));
vi.mock('./neural-canvas', () => ({
  ArtifactCard: ({ onOpen }: { onOpen?: () => void }) => (
    <button type="button" data-testid="artifact-card" onClick={onOpen}>
      artifact
    </button>
  ),
}));
vi.mock('./PlanCard', () => ({
  PlanCard: () => <div data-testid="plan-card">plan</div>,
}));
vi.mock('./PlanInvalidCard', () => ({
  PlanInvalidCard: () => <div data-testid="plan-invalid-card">invalid</div>,
  shouldShowPlanInvalidCard: (metadata?: Record<string, unknown>) => metadata?.plan_format_invalid === true,
}));

import { Message } from './Message';

function makeMessage(metadata?: Record<string, unknown>): ChatMessage {
  return {
    id: 'msg-1',
    type: 'chat',
    channel: 'general',
    from: { id: 'agent-1', name: 'Planner', type: 'assistant', expertise: [], status: 'idle', model: 'qwen' } as ChatMessage['from'],
    content: 'Plan ready',
    timestamp: new Date('2026-08-19T15:08:00Z').toISOString(),
    metadata,
  };
}

function resetStores() {
  useChatStore.setState({
    username: 'camron',
    highlightMessageId: null,
  });
  useFileExplorerStore.setState({
    activeWorkspaceId: 'ws-1',
    selectedPath: '',
    workspaces: [],
  });
  useFileChangeStore.setState({
    pendingChanges: [],
    changesById: {},
    busyById: {},
    errorsById: {},
  });
}

describe('Message plan artifacts', () => {
  beforeEach(() => {
    resetStores();
  });

  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
    resetStores();
  });

  it('renders ArtifactCard instead of PlanCard and auto-opens once', () => {
    const openArtifactSpy = vi.spyOn(useEditorStore.getState(), 'openArtifact');
    const updateLayoutSpy = vi.spyOn(useSettingsStore.getState(), 'updateLayoutSettings');

    const message = makeMessage({
      plan_id: 'hello_abc123',
      artifact_ref: {
        id: 'art-1',
        title: 'HelloWorld plan',
        renderer_id: 'nj.document',
        media_type: 'application/vnd.neural-junkie.document+json',
        revision: 1,
      },
    });

    const { rerender } = render(<Message message={message} />);

    expect(screen.getByTestId('artifact-card')).toBeTruthy();
    expect(screen.queryByTestId('plan-card')).toBeNull();
    expect(openArtifactSpy).toHaveBeenCalledTimes(1);
    expect(openArtifactSpy).toHaveBeenCalledWith('ws-1', 'art-1', 'HelloWorld plan', 'nj.document');
    expect(updateLayoutSpy).toHaveBeenCalledTimes(1);
    expect(updateLayoutSpy).toHaveBeenCalledWith({ editorPanelVisible: true });

    rerender(<Message message={message} />);
    expect(openArtifactSpy).toHaveBeenCalledTimes(1);
    expect(updateLayoutSpy).toHaveBeenCalledTimes(1);
  });

  it('renders PlanInvalidCard when plan_format_invalid is set', () => {
    const message = makeMessage({
      plan_format_invalid: true,
      editor_mode: 'plan',
    });
    render(<Message message={message} />);
    expect(screen.getByTestId('plan-invalid-card')).toBeTruthy();
    expect(screen.queryByTestId('plan-card')).toBeNull();
  });
});
