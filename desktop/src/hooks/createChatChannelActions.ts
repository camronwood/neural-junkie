import type { Dispatch, MutableRefObject, SetStateAction } from 'react';
import { ChatAPI } from '../api/chatAPI';
import type { WorkspaceContextMode } from '../constants/promptMetadata';
import { useChatStore } from '../stores/chatStore';
import { useFileExplorerStore } from '../stores/fileExplorerStore';
import type { Settings } from '../stores/settingsStore';
import { useSettingsStore } from '../stores/settingsStore';
import { useTerminalStore } from '../stores/terminalStore';
import type { Toast } from '../stores/toastStore';
import type { AgentInfo, Channel, Collaboration } from '../types/protocol';
import { isTerminalCollaborationPhase } from '../utils/chatInboundCollab';
import { ensureCollaborationExecutionWorkspace } from '../utils/collaborationExecutionWorkspace';
import { MAX_COLLAB_AGENTS } from '../utils/collaborationLimits';
import { syncCollabTurnThinking } from '../utils/collabThinking';
import { WORKSPACE_CONTEXT_MODE_KEY } from '../utils/outboundChatMetadata';
import { patchRevealForChannel } from '../utils/sidebarVisibility';
import { resolveTerminalCwd } from '../utils/terminalCwd';

export type ChatChannelActionsDeps = {
  api: ChatAPI;
  channel: string;
  username: string;
  agents: AgentInfo[];
  channels: Channel[];
  workspaceGateCollab: Collaboration | null;
  setWorkspaceGateBusy: Dispatch<SetStateAction<boolean>>;
  setWorkspaceGateCollab: Dispatch<SetStateAction<Collaboration | null>>;
  setWorkspaceContextMode: Dispatch<SetStateAction<WorkspaceContextMode>>;
  dismissedWorkspaceGateIdRef: MutableRefObject<string | null>;
  workspaceGateToastIdRef: MutableRefObject<string | null>;
  collaborationsByIDRef: MutableRefObject<Record<string, Collaboration>>;
  loadCollaborations: (targetChannel: string) => Promise<unknown>;
  loadChannels: () => Promise<unknown>;
  addToast: (toast: Omit<Toast, 'id' | 'count'>) => void;
  setActiveCollab: Dispatch<SetStateAction<Collaboration | null>>;
  setRunbookLibraryOpen: Dispatch<SetStateAction<boolean>>;
  setChannelInfoModal: Dispatch<SetStateAction<Channel | null>>;
  updateSettings: (patch: Partial<Settings>) => Promise<void>;
};

/** Channel / workspace-gate / runbook handlers extracted from ChatWindow. */
export function createChatChannelActions(deps: ChatChannelActionsDeps) {
  const revealSidebarForChannel = (channelName: string) => {
    const { settings, isLoaded } = useSettingsStore.getState();
    if (!isLoaded) return;
    const patch = patchRevealForChannel(
      settings,
      channelName,
      useChatStore.getState().channels,
      useChatStore.getState().agents,
    );
    if (patch) {
      void deps.updateSettings(patch);
    }
  };

  const handleWorkspaceGateContinue = async () => {
    const c = deps.workspaceGateCollab;
    if (!c) return;
    deps.setWorkspaceGateBusy(true);
    try {
      let sourceRepoPath: string | undefined;
      if (c.execution_mode === 'worktree' && !c.source_repo_path?.trim()) {
        const active = useFileExplorerStore.getState().getActiveWorkspace();
        if (!active?.path?.trim()) {
          throw new Error('Select a git workspace in the file explorer before continuing.');
        }
        if (!active.is_git_repo) {
          throw new Error('Active workspace is not a git repository.');
        }
        sourceRepoPath = active.path;
      }
      const deferWorktree = c.execution_mode === 'worktree' && !c.working_directory?.trim();
      if (!deferWorktree) {
        await ensureCollaborationExecutionWorkspace(c);
      }
      await deps.api.acknowledgeCollaborationWorkspace(c.id, sourceRepoPath);
      deps.dismissedWorkspaceGateIdRef.current = null;
      if (useChatStore.getState().channel === c.channel) {
        deps.setWorkspaceContextMode('always');
        localStorage.setItem(WORKSPACE_CONTEXT_MODE_KEY, 'always');
      }
      await deps.loadCollaborations(deps.channel);
      if (deferWorktree) {
        const refreshed = deps.collaborationsByIDRef.current[c.id];
        if (refreshed?.working_directory?.trim()) {
          await ensureCollaborationExecutionWorkspace(refreshed);
        }
      }
      deps.setWorkspaceGateCollab(null);
      deps.workspaceGateToastIdRef.current = null;
    } catch (e) {
      console.error('[workspace gate]', e);
      deps.addToast({
        type: 'error',
        title: 'Workspace confirmation failed',
        message: e instanceof Error ? e.message : 'Could not confirm workspace',
      });
    } finally {
      deps.setWorkspaceGateBusy(false);
    }
  };

  const handleWorkspaceGateDismiss = () => {
    if (deps.workspaceGateCollab) {
      deps.dismissedWorkspaceGateIdRef.current = deps.workspaceGateCollab.id;
    }
    deps.setWorkspaceGateCollab(null);
    deps.addToast({
      type: 'info',
      title: 'Workspace confirmation pending',
      message: 'Use the banner in the collaboration panel or chat strip when you are ready.',
    });
  };

  const handleSwitchChannel = async (channelName: string) => {
    const prevChannel = useChatStore.getState().channel;
    if (channelName === prevChannel) return;
    revealSidebarForChannel(channelName);
    // Collaboration side panel is channel-scoped; clear when navigating.
    deps.setActiveCollab(null);
    useChatStore.getState().switchChannel(channelName);
    localStorage.setItem('last-channel', channelName);
    void import('../stores/activityLogStore').then(({ logActivity }) => {
      logActivity({
        kind: 'channel',
        title: 'Switched channel',
        channel: channelName,
        detail: prevChannel ? `from #${prevChannel}` : undefined,
      });
    });
    if (prevChannel && prevChannel !== channelName) {
      useChatStore.getState().clearThinkingAgents(prevChannel);
    }
    try {
      const msgs = await deps.api.fetchMessages(channelName, 50);
      useChatStore.getState().setMessages(msgs);
      useChatStore.getState().cleanupStaleThinking(channelName, msgs);
      await deps.loadCollaborations(channelName);
      const collab = Object.values(deps.collaborationsByIDRef.current).find(
        (c) => c.channel === channelName && !isTerminalCollaborationPhase(c.phase),
      );
      if (collab) {
        syncCollabTurnThinking(collab, channelName);
      }
      const cwd = resolveTerminalCwd({ collaboration: collab ?? null });
      useTerminalStore.getState().alignActiveTabCwd(cwd);
    } catch (error) {
      console.error('Failed to load messages for channel:', error);
    }
  };

  const handleNewRunbook = async () => {
    deps.setRunbookLibraryOpen(true);
  };

  const handleCreateBlankRunbook = async () => {
    const pool = deps.agents.filter((a) => a.status === 'active' || a.status === 'idle');
    if (pool.length < 1) {
      deps.addToast({
        type: 'error',
        title: 'No agents',
        message: 'Add at least one active agent before creating a runbook.',
      });
      return;
    }
    const currentChannel = deps.channels.find((c) => c.name === deps.channel);
    const channelAgentIds = new Set(
      currentChannel?.agents?.map((a) => a.id) ?? currentChannel?.members ?? [],
    );
    const channelPool = pool.filter((a) => channelAgentIds.has(a.id));
    const pickFrom = channelPool.length > 0 ? channelPool : pool;
    const picked = pickFrom.slice(0, Math.min(MAX_COLLAB_AGENTS, pickFrom.length));
    try {
      const result = await deps.api.createRunbook({
        description: 'New runbook',
        agent_ids: picked.map((a) => a.id),
        channel: deps.channel,
        created_by: deps.username || 'User',
      });
      if (result.collaboration_channel && result.collaboration_channel !== deps.channel) {
        await handleSwitchChannel(result.collaboration_channel);
      }
      deps.setActiveCollab(result.collaboration);
      deps.addToast({
        type: 'success',
        title: 'Runbook created',
        message: 'Define tasks in the runbook builder panel.',
      });
      void deps.loadCollaborations(deps.channel);
    } catch (e) {
      deps.addToast({
        type: 'error',
        title: 'Runbook failed',
        message: e instanceof Error ? e.message : String(e),
      });
    }
  };

  const handleCreateChannel = async (name: string, description: string, agentIds: string[]) => {
    try {
      await deps.api.createChannel(name, description, 'custom', agentIds, deps.username);
      await deps.loadChannels();
      await handleSwitchChannel(name);
    } catch (error) {
      console.error('Failed to create channel:', error);
      deps.addToast({
        type: 'error',
        title: 'Could not create channel',
        message: error instanceof Error ? error.message : 'Channel creation failed.',
      });
    }
  };

  const handleDeleteChannel = async (name: string) => {
    const ch = useChatStore.getState().channels.find((c) => c.name === name);
    const label = ch?.type === 'collaboration' ? 'collaboration' : 'channel';
    if (!window.confirm(`Delete ${label} #${name}? This cannot be undone.`)) return;
    try {
      await deps.api.deleteChannel(name);
      const wasActive = useChatStore.getState().channel === name;
      await deps.loadChannels();
      if (wasActive) {
        await handleSwitchChannel('general');
      }
      deps.setChannelInfoModal((cur) => (cur?.name === name ? null : cur));
      const { logActivity } = await import('../stores/activityLogStore');
      logActivity({
        kind: 'channel',
        title: `Deleted ${label}`,
        detail: name,
        channel: name,
      });
      deps.addToast({
        type: 'success',
        title: 'Channel deleted',
        message: `#${name} was removed.`,
      });
    } catch (error) {
      console.error('Failed to delete channel:', error);
      deps.addToast({
        type: 'error',
        title: 'Could not delete channel',
        message: error instanceof Error ? error.message : 'Unknown error',
      });
    }
  };

  const handleOpenChannelInfo = async (ch: Channel) => {
    try {
      await deps.loadChannels();
      const list = useChatStore.getState().channels;
      const fresh = list.find((c) => c.name === ch.name) ?? ch;
      deps.setChannelInfoModal(fresh);
    } catch {
      deps.setChannelInfoModal(ch);
    }
  };

  return {
    handleWorkspaceGateContinue,
    handleWorkspaceGateDismiss,
    handleSwitchChannel,
    handleNewRunbook,
    handleCreateBlankRunbook,
    handleCreateChannel,
    handleDeleteChannel,
    handleOpenChannelInfo,
  };
}
