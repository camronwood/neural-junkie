import { ChatAPI } from '../api/chatAPI';
import { getHubBaseURL } from '../config/hubUrl';
import { useApprovalStore } from '../stores/approvalStore';
import { useChatStore } from '../stores/chatStore';
import { useFileExplorerStore } from '../stores/fileExplorerStore';
import { useLocationShareStore } from '../stores/locationShareStore';
import { shouldSendChannelJoinMessage } from '../utils/joinMessage';
import type { Toast } from '../stores/toastStore';

/** Load hub agents into chatStore (shared by ChatWindow and tests). */
export async function loadAgentsFromHub(api: ChatAPI): Promise<void> {
  const agentList = await api.fetchAgents({ includeToolCounts: true });
  useChatStore.getState().setAgents(agentList);

  const { loadingAgents, removeLoadingAgent } = useChatStore.getState();
  const activeAgentNames = new Set(agentList.map((agent) => agent.name));
  loadingAgents.forEach((agentName) => {
    if (activeAgentNames.has(agentName)) {
      removeLoadingAgent(agentName);
    }
  });
}

/** Load channels into chatStore. */
export async function loadChannelsFromHub(api?: ChatAPI): Promise<ReturnType<ChatAPI['fetchChannels']>> {
  const channelList = await (api ?? new ChatAPI(getHubBaseURL())).fetchChannels();
  useChatStore.getState().setChannels(channelList);
  return channelList;
}

export type LoadChatWindowInitialDataOpts = {
  api: ChatAPI;
  loadCollaborations: (targetChannel: string) => Promise<void>;
  loadAgents: () => Promise<void>;
  loadCounts: () => Promise<void>;
  loadChannels: () => Promise<unknown>;
  loadCommands: () => Promise<void>;
  addToast: (toast: Omit<Toast, 'id' | 'count'>) => void;
};

/** Parallel initial hub load after WebSocket connect (extracted from ChatWindow). */
export async function loadChatWindowInitialData(opts: LoadChatWindowInitialDataOpts): Promise<void> {
  const activeCh = useChatStore.getState().channel;
  const results = await Promise.allSettled([
    opts.api.fetchMessages(activeCh, 50).then((msgs) => useChatStore.getState().setMessages(msgs)),
    opts.loadCollaborations(activeCh),
    opts.loadAgents(),
    opts.loadCounts(),
    opts.loadChannels(),
    useFileExplorerStore.getState().loadWorkspaces(),
  ]);

  results.forEach((r, i) => {
    if (r.status === 'rejected') {
      const label = ['messages', 'collaborations', 'agents', 'counts', 'channels', 'workspaces'][i];
      console.error(`[loadInitialData] ${label} failed:`, r.reason);
      if (label === 'messages' || label === 'channels') {
        opts.addToast({
          type: 'error',
          title: `Failed to load ${label}`,
          message: r.reason instanceof Error ? r.reason.message : 'Hub request failed.',
        });
      }
    }
  });

  await opts.loadCommands();

  void useApprovalStore.getState().syncPendingFromHub(opts.api);
  void useLocationShareStore.getState().syncPending(opts.api);

  const { channel: joinCh, username: joinUser } = useChatStore.getState();
  const joinName = joinUser?.trim() || 'User';
  if (shouldSendChannelJoinMessage(joinCh, joinName)) {
    void opts.api
      .sendMessage(joinCh, `${joinName} has joined the chat`, { name: joinName, type: 'human' }, 'system_info')
      .catch((e) => console.error('[loadInitialData] join message failed:', e));
  }
}
