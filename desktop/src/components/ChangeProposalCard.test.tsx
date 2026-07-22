import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import type { ChangeProposalCard, Message } from '../types/protocol';
import { useFileChangeStore } from '../stores/fileChangeStore';
import { useGitChangeStore } from '../stores/gitChangeStore';
import { ChangeProposalMessageCard } from './ChangeProposalCard';

function messageWith(card: ChangeProposalCard): Message {
  return {
    id: `message-${card.id}`,
    type: card.kind === 'file_change' ? 'file_change' : 'chat',
    channel: 'general',
    from: { id: 'agent-1', name: 'Builder', type: 'cli' } as Message['from'],
    content: '',
    timestamp: new Date().toISOString(),
    metadata: { change_proposal: card },
  };
}

function proposal(
  status: ChangeProposalCard['status'] = 'pending',
  kind: ChangeProposalCard['kind'] = 'file_change',
): ChangeProposalCard {
  return {
    version: 1,
    kind,
    id: 'change-1',
    status,
    operation: kind === 'file_change' ? 'edit' : 'commit',
    file_path: kind === 'file_change' ? '/workspace/src/app.ts' : undefined,
    message: kind === 'git_change' ? 'Keep the change review in chat' : undefined,
  };
}

afterEach(() => {
  cleanup();
  useFileChangeStore.setState({
    pendingChanges: [],
    changesById: {},
    busyById: {},
    errorsById: {},
  });
  useGitChangeStore.setState({
    pendingGitChanges: [],
    changesById: {},
    busyById: {},
    errorsById: {},
  });
});

describe('ChangeProposalMessageCard', () => {
  it('accepts a pending file proposal from the chat card', () => {
    const approveChange = vi.fn().mockResolvedValue(undefined);
    useFileChangeStore.setState({ approveChange });
    render(<ChangeProposalMessageCard message={messageWith(proposal())} />);

    expect(screen.getByText('Proposed file change')).toBeTruthy();
    fireEvent.click(screen.getByRole('button', { name: 'Accept' }));
    expect(approveChange).toHaveBeenCalledWith('change-1');
  });

  it('collects an optional rejection reason for Git proposals', () => {
    const rejectGitChange = vi.fn().mockResolvedValue(undefined);
    useGitChangeStore.setState({ rejectGitChange });
    render(<ChangeProposalMessageCard message={messageWith(proposal('pending', 'git_change'))} />);

    fireEvent.click(screen.getByRole('button', { name: 'Reject' }));
    fireEvent.change(screen.getByRole('textbox', { name: 'Rejection reason' }), {
      target: { value: 'Split the commit first' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Confirm reject' }));
    expect(rejectGitChange).toHaveBeenCalledWith('change-1', 'Split the commit first');
  });

  it.each([
    ['approved', 'Accepted'],
    ['rejected', 'Rejected'],
    ['stale', 'Stale'],
    ['expired', 'Expired'],
    ['failed', 'Failed'],
  ] as const)('renders terminal %s state without actions', (status, label) => {
    render(<ChangeProposalMessageCard message={messageWith(proposal(status))} />);
    expect(screen.getByText(label)).toBeTruthy();
    expect(screen.queryByRole('button', { name: 'Accept' })).toBeNull();
  });

  it('labels destructive file operations explicitly', () => {
    const card = { ...proposal(), operation: 'delete' };
    render(<ChangeProposalMessageCard message={messageWith(card)} />);
    expect(screen.getByRole('button', { name: 'Accept destructive change' })).toBeTruthy();
  });
});
