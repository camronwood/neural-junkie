import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { RunbookBuilderPanel } from './RunbookBuilderPanel';
import type { Collaboration, CollaborationAgent } from '../types/protocol';

const fullCollabId = 'aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee';

const {
  updateRunbookMock,
  submitRunbookMock,
  startRunbookMock,
  getRunbookMock,
  acknowledgeWorkspaceMock,
} = vi.hoisted(() => ({
  updateRunbookMock: vi.fn(),
  submitRunbookMock: vi.fn(),
  startRunbookMock: vi.fn(),
  getRunbookMock: vi.fn(),
  acknowledgeWorkspaceMock: vi.fn(),
}));

vi.mock('../stores/chatStore', () => ({
  useChatStore: (selector: (s: { serverAddr: string }) => unknown) =>
    selector({ serverAddr: 'http://127.0.0.1:9' }),
}));

vi.mock('../api/chatAPI', () => ({
  ChatAPI: class {
    updateRunbook = updateRunbookMock;
    submitRunbook = submitRunbookMock;
    startRunbook = startRunbookMock;
    getRunbook = getRunbookMock;
    acknowledgeCollaborationWorkspace = acknowledgeWorkspaceMock;
  },
}));

vi.mock('../utils/collaborationExecutionWorkspace', () => ({
  ensureCollaborationExecutionWorkspace: vi.fn().mockResolvedValue(undefined),
}));

function makeCollaboration(overrides: Partial<Collaboration> = {}): Collaboration {
  const now = new Date().toISOString();
  return {
    id: fullCollabId,
    title: 'Runbook title',
    description: 'Ship the feature',
    phase: 'reviewing',
    source: 'runbook',
    agents: [
      {
        agent_id: 'ag1',
        agent_name: 'RustExpert',
        agent_type: 'rust',
        expertise: ['rust'],
        role: 'Rust',
      },
    ],
    tasks: [
      {
        id: 'task-1',
        title: 'Implement',
        description: 'Build it',
        assigned_to: 'ag1',
        assigned_name: 'RustExpert',
        status: 'pending',
        dependencies: [],
        created_at: now,
        updated_at: now,
      },
    ],
    channel: 'collab-test',
    created_by: 'tester',
    created_at: now,
    updated_at: now,
    ...overrides,
  };
}

const hubAgents: CollaborationAgent[] = [
  {
    agent_id: 'ag1',
    agent_name: 'RustExpert',
    agent_type: 'rust',
    expertise: ['rust'],
    role: 'Rust',
  },
];

afterEach(() => {
  cleanup();
});

beforeEach(() => {
  vi.clearAllMocks();
  updateRunbookMock.mockImplementation(async (id: string, body: Record<string, unknown>) =>
    makeCollaboration({
      id,
      description: String(body.description ?? 'Ship the feature'),
      tasks: (body.tasks as Collaboration['tasks']) ?? makeCollaboration().tasks,
    })
  );
  submitRunbookMock.mockResolvedValue(makeCollaboration({ phase: 'reviewing' }));
});

function clickThroughStartModal() {
  fireEvent.click(screen.getByRole('button', { name: 'Start execution' }));
  const modalStartButtons = screen.getAllByRole('button', { name: 'Start execution' });
  fireEvent.click(modalStartButtons[modalStartButtons.length - 1]!);
}

describe('RunbookBuilderPanel start execution', () => {
  it('opens workspace gate for sandbox runbooks after start', async () => {
    const started = makeCollaboration({
      phase: 'executing',
      working_directory: '/tmp/collab-sandbox',
      workspace_acknowledged: false,
    });
    startRunbookMock.mockResolvedValue(started);

    const onWorkspaceGateRequest = vi.fn();
    const onSaved = vi.fn();

    render(
      <RunbookBuilderPanel
        collaboration={makeCollaboration()}
        hubAgents={hubAgents}
        onClose={() => undefined}
        onSaved={onSaved}
        onWorkspaceGateRequest={onWorkspaceGateRequest}
      />
    );

    clickThroughStartModal();

    await waitFor(() => expect(startRunbookMock).toHaveBeenCalledWith(fullCollabId, {}));
    expect(acknowledgeWorkspaceMock).not.toHaveBeenCalled();
    expect(onWorkspaceGateRequest).toHaveBeenCalledWith(
      expect.objectContaining({
        id: fullCollabId,
        phase: 'executing',
        workspace_acknowledged: false,
      })
    );
    expect(onSaved).toHaveBeenCalled();
  });

  it('auto-acknowledges workspace when project repo is bound', async () => {
    const started = makeCollaboration({
      phase: 'executing',
      working_directory: '/tmp/collab-sandbox',
      source_repo_path: '/repo/project',
      workspace_acknowledged: false,
    });
    const acked = makeCollaboration({
      phase: 'executing',
      working_directory: '/tmp/collab-sandbox',
      source_repo_path: '/repo/project',
      workspace_acknowledged: true,
    });
    startRunbookMock.mockResolvedValue(started);
    acknowledgeWorkspaceMock.mockResolvedValue(undefined);
    getRunbookMock.mockResolvedValue(acked);

    const onWorkspaceGateRequest = vi.fn();
    const onSaved = vi.fn();

    render(
      <RunbookBuilderPanel
        collaboration={makeCollaboration()}
        hubAgents={hubAgents}
        onClose={() => undefined}
        onSaved={onSaved}
        onWorkspaceGateRequest={onWorkspaceGateRequest}
      />
    );

    clickThroughStartModal();

    await waitFor(() => expect(acknowledgeWorkspaceMock).toHaveBeenCalledWith(fullCollabId));
    expect(onWorkspaceGateRequest).not.toHaveBeenCalled();
    expect(onSaved).toHaveBeenLastCalledWith(
      expect.objectContaining({ workspace_acknowledged: true })
    );
  });

  it('submits draft runbooks before starting', async () => {
    const draft = makeCollaboration({ phase: 'draft' });
    const reviewing = makeCollaboration({ phase: 'reviewing' });
    const executing = makeCollaboration({
      phase: 'executing',
      working_directory: '/tmp/collab-sandbox',
      workspace_acknowledged: true,
    });
    updateRunbookMock.mockResolvedValue(makeCollaboration({ phase: 'draft' }));
    submitRunbookMock.mockResolvedValue(reviewing);
    startRunbookMock.mockResolvedValue(executing);

    render(
      <RunbookBuilderPanel
        collaboration={draft}
        hubAgents={hubAgents}
        onClose={() => undefined}
        onSaved={() => undefined}
      />
    );

    clickThroughStartModal();

    await waitFor(() => expect(submitRunbookMock).toHaveBeenCalledWith(fullCollabId));
    await waitFor(() => expect(startRunbookMock).toHaveBeenCalledWith(fullCollabId, {}));
  });
});
