import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { MessageList } from './MessageList';
import { FIRST_WIN_DISMISS_KEY } from '../config/firstWinSteps';
import { useChatStore } from '../stores/chatStore';
import { useFileExplorerStore } from '../stores/fileExplorerStore';
import { usePacksStore } from '../stores/packsStore';
import { useCollaborationsStore } from '../stores/collaborationsStore';

function resetStores() {
  localStorage.removeItem(FIRST_WIN_DISMISS_KEY);
  useChatStore.setState({
    channel: 'general',
    messages: [],
    streamingMessages: {},
    agents: [],
    myAgents: [],
    threadMetadata: new Map(),
    pendingScrollToMessageId: null,
    highlightMessageId: null,
  });
  useFileExplorerStore.setState({ workspaces: [] });
  usePacksStore.setState({ packs: [] });
  useCollaborationsStore.getState().clear();
}

beforeEach(() => {
  resetStores();
});

afterEach(() => {
  cleanup();
  resetStores();
});

describe('MessageList first-win empty state', () => {
  it('renders the coach when the channel is empty and not dismissed', () => {
    render(<MessageList />);
    expect(screen.getByTestId('first-win-coach')).toBeTruthy();
    expect(screen.queryByText('Start the conversation!')).toBeNull();
  });

  it('keeps search-empty copy unchanged', () => {
    render(<MessageList searchQuery="nope" />);
    expect(screen.getByText('No matches in this chat')).toBeTruthy();
    expect(screen.getByText('Try a different search.')).toBeTruthy();
    expect(screen.queryByTestId('first-win-coach')).toBeNull();
  });

  it('falls back to the legacy empty copy after dismiss', () => {
    render(<MessageList />);
    fireEvent.click(screen.getByTestId('first-win-dismiss'));
    expect(screen.queryByTestId('first-win-coach')).toBeNull();
    expect(screen.getByText('No messages yet')).toBeTruthy();
    expect(screen.getByText('Start the conversation!')).toBeTruthy();
  });

  it('hides the coach when developer setup steps are already complete', () => {
    usePacksStore.setState({
      packs: [
        {
          id: 'ide',
          title: 'IDE',
          description: '',
          installed: true,
          enabled: true,
        },
      ],
    });
    useFileExplorerStore.setState({
      workspaces: [
        {
          id: 'ws1',
          name: 'repo',
          path: '/tmp/repo',
          created_at: '',
          last_used: '',
          is_git_repo: true,
        },
      ],
    });
    useChatStore.setState({
      agents: [
        {
          id: 'r1',
          name: 'MyRepoExpert',
          type: 'repo',
          expertise: [],
          status: 'idle',
          model: 'qwen',
          is_paused: false,
        },
      ],
    });
    render(<MessageList />);
    expect(screen.queryByTestId('first-win-coach')).toBeNull();
    expect(screen.getByText('Start the conversation!')).toBeTruthy();
  });
});
