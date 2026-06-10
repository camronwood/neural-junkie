import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { PendingChangesPanel } from './PendingChangesPanel';
import type { FileChange } from '../types/protocol';

function makeChange(id: string, path: string): FileChange {
  return {
    id,
    operation: 'edit',
    file_path: path,
    agent: { name: 'TestAgent', type: 'go' },
    channel: 'general',
    status: 'pending',
    requested_at: new Date().toISOString(),
    expires_at: new Date(Date.now() + 3600000).toISOString(),
  };
}

const selectChangeMock = vi.fn();
const approveChangeMock = vi.fn().mockResolvedValue(undefined);
const rejectChangeMock = vi.fn().mockResolvedValue(undefined);
const fetchPendingChangesMock = vi.fn();
const refreshChangesMock = vi.fn();

let pendingChanges: FileChange[] = [];
let loading = false;

vi.mock('../stores/fileChangeStore', () => ({
  useFileChangeStore: () => ({
    pendingChanges,
    loading,
    error: null,
    fetchPendingChanges: fetchPendingChangesMock,
    approveChange: approveChangeMock,
    rejectChange: rejectChangeMock,
    selectChange: selectChangeMock,
    clearError: vi.fn(),
    refreshChanges: refreshChangesMock,
    previewData: null,
  }),
}));

afterEach(() => {
  cleanup();
  pendingChanges = [];
  loading = false;
  vi.clearAllMocks();
});

describe('PendingChangesPanel preview close cascade', () => {
  beforeEach(() => {
    vi.stubGlobal(
      'setInterval',
      vi.fn(() => 0 as unknown as ReturnType<typeof setInterval>),
    );
    vi.stubGlobal('clearInterval', vi.fn());
  });

  it('closes the list modal when dismissing preview for a single pending change', async () => {
    pendingChanges = [makeChange('change-1', 'src/a.ts')];
    const onClose = vi.fn();

    render(
      <PendingChangesPanel onClose={onClose} initialChangeId="change-1" />,
    );

    await waitFor(() => {
      expect(screen.getByLabelText('Close preview')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByLabelText('Close preview'));

    expect(onClose).toHaveBeenCalledTimes(1);
    expect(selectChangeMock).toHaveBeenCalledWith(null);
  });

  it('keeps the list modal open when dismissing preview with multiple pending changes', async () => {
    pendingChanges = [
      makeChange('change-1', 'src/a.ts'),
      makeChange('change-2', 'src/b.ts'),
    ];
    const onClose = vi.fn();

    render(
      <PendingChangesPanel onClose={onClose} initialChangeId="change-1" />,
    );

    await waitFor(() => {
      expect(screen.getByLabelText('Close preview')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByLabelText('Close preview'));

    expect(onClose).not.toHaveBeenCalled();
    expect(screen.getByText('Pending File Changes')).toBeInTheDocument();
    expect(selectChangeMock).toHaveBeenCalledWith(null);
  });

  it('closes both modals after approving the only pending change from preview', async () => {
    pendingChanges = [makeChange('change-1', 'src/a.ts')];
    const onClose = vi.fn();

    render(
      <PendingChangesPanel onClose={onClose} initialChangeId="change-1" />,
    );

    await waitFor(() => {
      expect(screen.getAllByText('Approve').length).toBeGreaterThan(1);
    });

    const approveButtons = screen.getAllByText('Approve');
    fireEvent.click(approveButtons[approveButtons.length - 1]!);

    await waitFor(() => {
      expect(approveChangeMock).toHaveBeenCalledWith('change-1');
      expect(onClose).toHaveBeenCalledTimes(1);
    });
  });

  it('closes both modals after rejecting the only pending change from preview', async () => {
    pendingChanges = [makeChange('change-1', 'src/a.ts')];
    const onClose = vi.fn();

    render(
      <PendingChangesPanel onClose={onClose} initialChangeId="change-1" />,
    );

    await waitFor(() => {
      expect(screen.getAllByText('Reject').length).toBeGreaterThan(1);
    });

    const rejectButtons = screen.getAllByText('Reject');
    fireEvent.click(rejectButtons[rejectButtons.length - 1]!);
    fireEvent.change(screen.getByPlaceholderText('Enter reason for rejection...'), {
      target: { value: 'Not needed' },
    });
    const confirmRejectButtons = screen.getAllByRole('button', { name: 'Reject' });
    fireEvent.click(confirmRejectButtons[confirmRejectButtons.length - 1]!);

    await waitFor(() => {
      expect(rejectChangeMock).toHaveBeenCalledWith('change-1', 'Not needed');
      expect(onClose).toHaveBeenCalledTimes(1);
    });
  });
});
