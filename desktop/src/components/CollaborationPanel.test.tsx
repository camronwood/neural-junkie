import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { CollaborationPanel } from './CollaborationPanel';
import type { Collaboration, DiscussionSession, SharedArtifact } from '../types/protocol';

const fullCollabId = 'aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee';
const collabPrefix = 'aaaaaaaa';

const { sendMessageMock, confirmReplaceMock } = vi.hoisted(() => ({
  sendMessageMock: vi.fn().mockResolvedValue({}),
  confirmReplaceMock: vi.fn((_a: unknown, _b: unknown) => true),
}));

vi.mock('../stores/chatStore', () => ({
  useChatStore: (selector: (s: Record<string, string>) => unknown) =>
    selector({
      serverAddr: 'http://127.0.0.1:9',
      channel: 'collab-test-channel',
      username: 'PanelTester',
    }),
}));

vi.mock('../api/chatAPI', () => ({
  ChatAPI: class {
    sendMessage = sendMessageMock;
  },
}));

vi.mock('../utils/collaborationConfirm', () => ({
  confirmReplaceCollaborationExecution: (executing: unknown, incoming: unknown) =>
    confirmReplaceMock(executing, incoming) as boolean,
}));

function discussion(overrides: Partial<DiscussionSession> = {}): DiscussionSession {
  return {
    id: 'd1',
    collaboration_id: fullCollabId,
    topic: 't',
    participants: [],
    max_rounds: 3,
    current_round: 1,
    turn_budget: 1,
    total_message_count: 2,
    max_total_messages: 20,
    status: 'active',
    current_turn_index: 0,
    consensus: {},
    ...overrides,
  };
}

function makeCollaboration(overrides: Partial<Collaboration> = {}): Collaboration {
  return {
    id: fullCollabId,
    title: 'Panel collab title',
    description: 'Panel collab description body',
    phase: 'reviewing',
    agents: [
      {
        agent_id: 'ag1',
        agent_name: 'RustExpert',
        agent_type: 'rust',
        expertise: ['rust'],
        role: 'Rust Architecture & Systems Design',
      },
    ],
    tasks: [],
    channel: 'collab-test-channel',
    created_by: 'u1',
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
    ...overrides,
  };
}

afterEach(() => {
  cleanup();
});

beforeEach(() => {
  sendMessageMock.mockClear();
  confirmReplaceMock.mockClear();
  confirmReplaceMock.mockReturnValue(true);
});

describe('CollaborationPanel', () => {
  it('renders title, description, phase badge, and participants', () => {
    const collab = makeCollaboration();
    render(
      <CollaborationPanel collaboration={collab} onClose={() => {}} />
    );

    expect(screen.getByText('Collaboration')).toBeInTheDocument();
    expect(screen.getByText('Reviewing Plan')).toBeInTheDocument();
    expect(screen.getByText('Panel collab title')).toBeInTheDocument();
    expect(screen.getByText('Panel collab description body')).toBeInTheDocument();
    expect(screen.getByText('@RustExpert')).toBeInTheDocument();
    expect(screen.getByText('Rust Architecture & Systems Design')).toBeInTheDocument();
  });

  it('planning phase shows only cancel (no resume) until reviewing or executing', () => {
    const collab = makeCollaboration({
      phase: 'planning',
      discussion: discussion(),
    });
    render(<CollaborationPanel collaboration={collab} onClose={() => {}} />);

    expect(screen.queryByRole('button', { name: 'Resume plan' })).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Cancel Collaboration' })).toBeInTheDocument();
  });

  it('sends /resume-plan with 8-char id from reviewing and refreshes after', async () => {
    const onAfter = vi.fn().mockResolvedValue(undefined);
    const collab = makeCollaboration({ phase: 'reviewing' });

    render(
      <CollaborationPanel collaboration={collab} onClose={() => {}} onAfterCollaborationCommand={onAfter} />
    );

    fireEvent.click(screen.getByRole('button', { name: 'Resume plan' }));

    await waitFor(() => {
      expect(sendMessageMock).toHaveBeenCalledWith(
        'collab-test-channel',
        `/resume-plan ${collabPrefix}`,
        { name: 'PanelTester', type: 'human' }
      );
    });
    expect(onAfter).toHaveBeenCalled();
    expect(confirmReplaceMock).toHaveBeenCalled();
  });

  it('does not send resume when confirmReplaceCollaborationExecution returns false', async () => {
    confirmReplaceMock.mockReturnValue(false);
    const collab = makeCollaboration({ phase: 'reviewing' });

    render(<CollaborationPanel collaboration={collab} onClose={() => {}} />);

    fireEvent.click(screen.getByRole('button', { name: 'Resume plan' }));

    await waitFor(() => {
      expect(confirmReplaceMock).toHaveBeenCalled();
    });
    expect(sendMessageMock).not.toHaveBeenCalled();
  });

  it('labels resume when another collaboration is executing', () => {
    const collab = makeCollaboration({ phase: 'reviewing', id: '11111111-2222-3333-4444-555555555555' });
    const other = makeCollaboration({
      id: '99999999-aaaa-bbbb-cccc-dddddddddddd',
      phase: 'executing',
      title: 'Other run',
    });

    render(
      <CollaborationPanel
        collaboration={collab}
        executingCollaboration={other}
        onClose={() => {}}
      />
    );

    expect(screen.getByRole('button', { name: 'Resume plan (stop other run)' })).toBeInTheDocument();
  });

  it('revise is disabled without feedback; sends /revise-plan with trimmed body', async () => {
    const onAfter = vi.fn().mockResolvedValue(undefined);
    const collab = makeCollaboration({ phase: 'reviewing' });

    render(
      <CollaborationPanel collaboration={collab} onClose={() => {}} onAfterCollaborationCommand={onAfter} />
    );

    const reviseBtn = screen.getByRole('button', { name: 'Revise' });
    expect(reviseBtn).toBeDisabled();

    fireEvent.change(screen.getByPlaceholderText(/Feedback for revision/i), {
      target: { value: '  Add observability  ' },
    });
    expect(reviseBtn).not.toBeDisabled();

    fireEvent.click(reviseBtn);

    await waitFor(() => {
      expect(sendMessageMock).toHaveBeenCalledWith(
        'collab-test-channel',
        `/revise-plan ${collabPrefix}   Add observability  `,
        { name: 'PanelTester', type: 'human' }
      );
    });
    expect(onAfter).toHaveBeenCalled();
  });

  it('submits revision on Ctrl+Enter in the feedback textarea', async () => {
    const collab = makeCollaboration({ phase: 'reviewing' });
    render(<CollaborationPanel collaboration={collab} onClose={() => {}} />);

    const ta = screen.getByPlaceholderText(/Feedback for revision/i);
    fireEvent.change(ta, { target: { value: 'Ship in two phases' } });
    fireEvent.keyDown(ta, { key: 'Enter', ctrlKey: true });

    await waitFor(() => {
      expect(sendMessageMock).toHaveBeenCalledWith(
        'collab-test-channel',
        `/revise-plan ${collabPrefix} Ship in two phases`,
        { name: 'PanelTester', type: 'human' }
      );
    });
  });

  it('mark done sends /complete-collab with --force when tasks are open', async () => {
    const confirmSpy = vi.spyOn(window, 'confirm').mockReturnValue(true);
    const onAfter = vi.fn().mockResolvedValue(undefined);
    const collab = makeCollaboration({
      phase: 'executing',
      tasks: [
        {
          id: 't1',
          title: 'Build index',
          description: 'd',
          assigned_to: 'ag1',
          assigned_name: 'RustExpert',
          status: 'in_progress',
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
        },
      ],
    });

    render(
      <CollaborationPanel collaboration={collab} onClose={() => {}} onAfterCollaborationCommand={onAfter} />
    );

    fireEvent.click(screen.getByRole('button', { name: 'Mark collaboration done' }));

    await waitFor(() => {
      expect(sendMessageMock).toHaveBeenCalledWith(
        'collab-test-channel',
        `/complete-collab ${collabPrefix} --force`,
        expect.objectContaining({ name: 'PanelTester' })
      );
    });
    confirmSpy.mockRestore();
  });

  it('mark task done sends /collab-task-done with 1-based index', async () => {
    const onAfter = vi.fn().mockResolvedValue(undefined);
    const collab = makeCollaboration({
      phase: 'executing',
      tasks: [
        {
          id: 't1',
          title: 'Build index',
          description: 'd',
          assigned_to: 'ag1',
          assigned_name: 'RustExpert',
          status: 'pending',
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
        },
        {
          id: 't2',
          title: 'Review',
          description: 'd',
          assigned_to: 'ag1',
          assigned_name: 'RustExpert',
          status: 'completed',
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
        },
      ],
    });

    render(
      <CollaborationPanel collaboration={collab} onClose={() => {}} onAfterCollaborationCommand={onAfter} />
    );

    const buttons = screen.getAllByRole('button', { name: 'Mark task done' });
    expect(buttons).toHaveLength(1);
    fireEvent.click(buttons[0]);

    await waitFor(() => {
      expect(sendMessageMock).toHaveBeenCalledWith(
        'collab-test-channel',
        `/collab-task-done ${collabPrefix} 1`,
        expect.objectContaining({ name: 'PanelTester' })
      );
    });
  });

  it('completed phase shows dismiss and hides resume', () => {
    const collab = makeCollaboration({ phase: 'completed' });
    render(<CollaborationPanel collaboration={collab} onClose={() => {}} />);

    expect(screen.getByRole('button', { name: 'Dismiss' })).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Resume plan' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Mark collaboration done' })).not.toBeInTheDocument();
  });

  it('cancel sends /cancel-plan with id prefix', async () => {
    const onAfter = vi.fn().mockResolvedValue(undefined);
    const collab = makeCollaboration({ phase: 'planning', discussion: discussion() });

    render(
      <CollaborationPanel collaboration={collab} onClose={() => {}} onAfterCollaborationCommand={onAfter} />
    );

    fireEvent.click(screen.getByRole('button', { name: 'Cancel Collaboration' }));

    await waitFor(() => {
      expect(sendMessageMock).toHaveBeenCalledWith(
        'collab-test-channel',
        `/cancel-plan ${collabPrefix}`,
        { name: 'PanelTester', type: 'human' }
      );
    });
    expect(onAfter).toHaveBeenCalled();
  });

  it('executing phase shows task progress and resume for re-dispatch', () => {
    const collab = makeCollaboration({
      phase: 'executing',
      discussion: discussion({ status: 'active', total_message_count: 5 }),
      tasks: [
        {
          id: 't1',
          title: 'First',
          description: '',
          assigned_to: 'a1',
          assigned_name: 'AgentA',
          status: 'completed',
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
        },
        {
          id: 't2',
          title: 'Second',
          description: '',
          assigned_to: 'a2',
          assigned_name: 'AgentB',
          status: 'pending',
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
        },
      ],
    });

    render(<CollaborationPanel collaboration={collab} onClose={() => {}} />);

    expect(screen.getByText(/1\/2 tasks \(50%\)/)).toBeInTheDocument();
    expect(screen.getByText('Task 1: First')).toBeInTheDocument();
    expect(screen.getByText('Task 2: Second')).toBeInTheDocument();
    expect(screen.getByText('Execution — limits off')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Resume plan' })).toBeInTheDocument();
  });

  it('shows collab-extend hint when discussion hits budget in reviewing', () => {
    const collab = makeCollaboration({
      phase: 'reviewing',
      discussion: discussion({ status: 'budget_exhausted', total_message_count: 20, max_total_messages: 20 }),
    });

    render(<CollaborationPanel collaboration={collab} onClose={() => {}} />);

    expect(screen.getByText(/Discussion limits reached/)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Extend discussion' })).toBeInTheDocument();
  });

  it('renders plan version and uses @unassigned for tasks without assignee name', () => {
    const plan: SharedArtifact = {
      id: 'art1',
      title: 'Plan',
      content: '## Plan\n\n- Step one',
      version: 2,
      status: 'draft',
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    };
    const collab = makeCollaboration({
      phase: 'executing',
      plan,
      tasks: [
        {
          id: 't1',
          title: 'Unowned task',
          description: '',
          assigned_to: '',
          assigned_name: '',
          status: 'pending',
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
        },
      ],
      discussion: discussion(),
    });

    render(<CollaborationPanel collaboration={collab} onClose={() => {}} />);

    expect(screen.getByText('Plan (v2)')).toBeInTheDocument();
    expect(screen.getByRole('heading', { level: 2, name: 'Plan' })).toBeInTheDocument();
    expect(screen.getByText(/Step one/)).toBeInTheDocument();
    expect(screen.getByText(/@unassigned/)).toBeInTheDocument();
  });

  it('terminal phases hide action footer', () => {
    const completed = makeCollaboration({ phase: 'completed', discussion: discussion() });
    const { unmount: u1 } = render(<CollaborationPanel collaboration={completed} onClose={() => {}} />);
    expect(screen.queryByRole('button', { name: 'Cancel Collaboration' })).not.toBeInTheDocument();
    u1();

    const cancelled = makeCollaboration({ phase: 'cancelled' });
    render(<CollaborationPanel collaboration={cancelled} onClose={() => {}} />);
    expect(screen.queryByRole('button', { name: 'Cancel Collaboration' })).not.toBeInTheDocument();
  });

  it('close control invokes onClose', () => {
    const onClose = vi.fn();
    render(<CollaborationPanel collaboration={makeCollaboration()} onClose={onClose} />);

    fireEvent.click(screen.getByRole('button', { name: '×' }));
    expect(onClose).toHaveBeenCalledTimes(1);
  });
});
