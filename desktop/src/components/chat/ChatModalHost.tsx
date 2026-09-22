import type { ChatAPI, LearningProposalAction } from '../../api/chatAPI';
import type { LoraTrainPrefill } from '../LoraTrainingPanel';
import type {
  AgentInfo,
  AssistantTask,
  Channel,
  Collaboration,
  CommandDefinition,
  FileChange,
} from '../../types/protocol';
import type { HubDataAccessOption } from '../../utils/hubDataAccess';
import { useChatStore } from '../../stores/chatStore';
import { useToastStore } from '../../stores/toastStore';
import { GitModal } from '../GitPanel';
import { QuickOpenModal } from '../QuickOpenModal';
import { SymbolModal } from '../SymbolModal';
import { ProblemsPanel } from '../ProblemsPanel';
import { FastEditModal } from '../FastEditModal';
import { CommandPalette } from '../CommandPalette';
import { CreateChannelModal } from '../CreateChannelModal';
import {
  findCollaborateCommand,
  StartCollaborationModal,
} from '../StartCollaborationModal';
import { ChannelInfoModal } from '../ChannelInfoModal';
import { CreateNewDMModal } from '../CreateNewDMModal';
import { LearningProposalModal } from '../LearningProposalModal';
import { HubDataAccessModal } from '../HubDataAccessModal';
import {
  LazyAIInterviewPrepModal,
  LazyDomainPacksModal,
  LazyModelArenaModal,
  LazyModelLibraryModal,
  LazyPanelShell,
  LazyPhoenixBrowserModal,
  LazyRoomChatModal,
  LazyRunbookLibraryModal,
} from '../lazyPanels';
import { agentsToCollaborationAgents } from './chatWindowHelpers';

export interface ChatModalHostProps {
  api: ChatAPI;
  hubHttp: string;
  channel: string;
  username: string;
  agents: AgentInfo[];
  channels: Channel[];
  trackedCollaborations: Collaboration[];
  assistantTasks: AssistantTask[];
  pendingChanges: FileChange[];
  commandDefs: CommandDefinition[];
  ideEnabled: boolean;
  activeWorkspaceId: string | null;
  explorerWorkspaces: Array<{ id: string; path?: string }>;
  gitModalOpen: boolean;
  setGitModalOpen: (open: boolean) => void;
  quickOpenOpen: boolean;
  setQuickOpenOpen: (open: boolean) => void;
  symbolModalOpen: boolean;
  setSymbolModalOpen: (open: boolean) => void;
  problemsOpen: boolean;
  setProblemsOpen: (open: boolean) => void;
  fastEditOpen: boolean;
  setFastEditOpen: (open: boolean) => void;
  commandPaletteOpen: boolean;
  commandPaletteFilter: string;
  closeCommandPalette: () => void;
  handleCommandExecute: (commandString: string, metadata?: Record<string, unknown>) => void | Promise<void>;
  startCollaborationOpen: boolean;
  setStartCollaborationOpen: (open: boolean) => void;
  createChannelOpen: boolean;
  setCreateChannelOpen: (open: boolean) => void;
  handleCreateChannel: (name: string, description: string, agentIds: string[]) => void | Promise<void>;
  channelInfoModal: Channel | null;
  setChannelInfoModal: (ch: Channel | null) => void;
  createNewDmOpen: boolean;
  setCreateNewDmOpen: (open: boolean) => void;
  handleNewDmCreated: (channel: Channel) => void | Promise<void>;
  modelLibraryOpen: boolean;
  closeModelLibrary: () => void;
  switchAllAgentProviders: ChatAPI['switchAllAgentProviders'];
  switchAgentProvider: ChatAPI['switchAgentProvider'];
  modelLibraryInitialTab?: 'installed' | 'ollama' | 'huggingface' | 'train';
  loraTrainPrefill: LoraTrainPrefill | null;
  domainPacksOpen: boolean;
  closeDomainPacks: () => void;
  phoenixModalOpen: boolean;
  roomChatModalOpen: boolean;
  modelArenaModalOpen: boolean;
  aiInterviewModalOpen: boolean;
  setActivePackModal: (modal: string | null) => void;
  openArenaWorkbench: (workspaceId: string, path: string) => void;
  setCodeEditorOpen: (open: boolean) => void;
  updateLayoutSettings: (patch: { editorPanelVisible?: boolean }) => void | Promise<void>;
  learningProposalOpen: boolean;
  learningProposal: LearningProposalAction | null;
  setLearningProposalOpen: (open: boolean) => void;
  setLearningProposal: (p: LearningProposalAction | null) => void;
  activeCollabForChannel: Collaboration | null | undefined;
  hubAccessPending: {
    mode: 'thread' | 'main';
    threadId?: string;
    content: string;
    metadata?: Record<string, unknown>;
    options: HubDataAccessOption[];
  } | null;
  hubAccessLoading: boolean;
  hubAccessError: string | null;
  setHubAccessPending: (
    v: {
      mode: 'thread' | 'main';
      threadId?: string;
      content: string;
      metadata?: Record<string, unknown>;
      options: HubDataAccessOption[];
    } | null
  ) => void;
  setHubAccessError: (e: string | null) => void;
  handleHubAccessConfirm: (selected: HubDataAccessOption[]) => void | Promise<void>;
  runbookLibraryOpen: boolean;
  setRunbookLibraryOpen: (open: boolean) => void;
  handleSwitchChannel: (name: string) => Promise<void>;
  setActiveCollab: (c: Collaboration | null) => void;
  loadCollaborations: (channel: string) => void | Promise<void>;
  handleCreateBlankRunbook: () => void | Promise<void>;
  handleQuickOpenPath: (path: string) => void;
  handleOpenAtLine: (path: string, line: number, column?: number) => void;
}

/** Overlay modals / lazy panels hosted by ChatWindow (extract-by-cut). */
export function ChatModalHost(props: ChatModalHostProps) {
  const {
    api,
    hubHttp,
    channel,
    username,
    agents,
    channels,
    trackedCollaborations,
    assistantTasks,
    pendingChanges,
    commandDefs,
    ideEnabled,
    activeWorkspaceId,
    explorerWorkspaces,
    gitModalOpen,
    setGitModalOpen,
    quickOpenOpen,
    setQuickOpenOpen,
    symbolModalOpen,
    setSymbolModalOpen,
    problemsOpen,
    setProblemsOpen,
    fastEditOpen,
    setFastEditOpen,
    commandPaletteOpen,
    commandPaletteFilter,
    closeCommandPalette,
    handleCommandExecute,
    startCollaborationOpen,
    setStartCollaborationOpen,
    createChannelOpen,
    setCreateChannelOpen,
    handleCreateChannel,
    channelInfoModal,
    setChannelInfoModal,
    createNewDmOpen,
    setCreateNewDmOpen,
    handleNewDmCreated,
    modelLibraryOpen,
    closeModelLibrary,
    switchAllAgentProviders,
    switchAgentProvider,
    modelLibraryInitialTab,
    loraTrainPrefill,
    domainPacksOpen,
    closeDomainPacks,
    phoenixModalOpen,
    roomChatModalOpen,
    modelArenaModalOpen,
    aiInterviewModalOpen,
    setActivePackModal,
    openArenaWorkbench,
    setCodeEditorOpen,
    updateLayoutSettings,
    learningProposalOpen,
    learningProposal,
    setLearningProposalOpen,
    setLearningProposal,
    activeCollabForChannel,
    hubAccessPending,
    hubAccessLoading,
    hubAccessError,
    setHubAccessPending,
    setHubAccessError,
    handleHubAccessConfirm,
    runbookLibraryOpen,
    setRunbookLibraryOpen,
    handleSwitchChannel,
    setActiveCollab,
    loadCollaborations,
    handleCreateBlankRunbook,
    handleQuickOpenPath,
    handleOpenAtLine,
  } = props;

  const addToast = useToastStore((s) => s.addToast);
  const workspaceId =
    explorerWorkspaces.find((w) => w.id === activeWorkspaceId)?.id ?? explorerWorkspaces[0]?.id;

  return (
    <>
      <GitModal isOpen={gitModalOpen && ideEnabled} onClose={() => setGitModalOpen(false)} />

      <QuickOpenModal
        isOpen={quickOpenOpen && ideEnabled}
        workspaceId={workspaceId}
        onClose={() => setQuickOpenOpen(false)}
        onOpenPath={handleQuickOpenPath}
      />

      <SymbolModal
        isOpen={symbolModalOpen && ideEnabled}
        workspaceId={workspaceId}
        onClose={() => setSymbolModalOpen(false)}
        onOpenSymbol={handleOpenAtLine}
      />

      <ProblemsPanel
        isOpen={problemsOpen && ideEnabled}
        onClose={() => setProblemsOpen(false)}
        onOpenAt={handleOpenAtLine}
      />

      <FastEditModal
        isOpen={fastEditOpen && ideEnabled}
        workspaceId={workspaceId}
        onClose={() => setFastEditOpen(false)}
      />

      <CommandPalette
        commands={commandDefs}
        agents={agents}
        channels={channels}
        activeChannel={channel}
        collaborations={trackedCollaborations}
        assistantTasks={assistantTasks}
        pendingChanges={pendingChanges}
        api={api}
        isOpen={commandPaletteOpen}
        initialFilter={commandPaletteFilter}
        onClose={closeCommandPalette}
        onExecute={handleCommandExecute}
      />

      <StartCollaborationModal
        isOpen={startCollaborationOpen}
        command={findCollaborateCommand(commandDefs)}
        agents={agents}
        channels={channels}
        activeChannel={channel}
        collaborations={trackedCollaborations}
        pendingChanges={pendingChanges}
        api={api}
        onClose={() => setStartCollaborationOpen(false)}
        onSubmit={handleCommandExecute}
      />

      <CreateChannelModal
        agents={agents}
        isOpen={createChannelOpen}
        onClose={() => setCreateChannelOpen(false)}
        onCreate={handleCreateChannel}
      />

      {channelInfoModal && (
        <ChannelInfoModal
          channel={channelInfoModal}
          agents={agents}
          api={api}
          onClose={() => setChannelInfoModal(null)}
          onClearHistory={async (name) => {
            await api.clearChannelHistory(name);
            const msgs = await api.fetchMessages(name, 50);
            const st = useChatStore.getState();
            st.replaceChannelMessagesCache(name, msgs);
            if (name === st.channel) {
              st.setMessages(msgs);
              st.cleanupStaleThinking(name, msgs);
            }
            addToast({ type: 'success', title: 'Channel history cleared' });
          }}
        />
      )}

      <CreateNewDMModal
        api={api}
        username={username}
        isOpen={createNewDmOpen}
        onClose={() => setCreateNewDmOpen(false)}
        onCreated={handleNewDmCreated}
      />

      {modelLibraryOpen && (
        <LazyPanelShell
          panelName="Model library"
          Component={LazyModelLibraryModal}
          props={{
            isOpen: modelLibraryOpen,
            onClose: closeModelLibrary,
            serverAddr: hubHttp,
            switchAllAgentProviders,
            switchAgentProvider,
            runtimeAgents: agents.map((a) => ({ id: a.id, name: a.name, type: a.type })),
            defaultChannel: channel,
            initialTab: modelLibraryInitialTab,
            loraTrainPrefill,
          }}
        />
      )}

      {domainPacksOpen && (
        <LazyPanelShell
          panelName="Domain packs"
          Component={LazyDomainPacksModal}
          props={{
            isOpen: domainPacksOpen,
            onClose: closeDomainPacks,
            serverAddr: hubHttp,
          }}
        />
      )}

      {phoenixModalOpen && (
        <LazyPanelShell
          panelName="Web browser"
          Component={LazyPhoenixBrowserModal}
          props={{ isOpen: phoenixModalOpen, onClose: () => setActivePackModal(null) }}
        />
      )}
      {roomChatModalOpen && (
        <LazyPanelShell
          panelName="Room chat"
          Component={LazyRoomChatModal}
          props={{ isOpen: roomChatModalOpen, onClose: () => setActivePackModal(null) }}
        />
      )}
      {modelArenaModalOpen && (
        <LazyPanelShell
          panelName="Model arena"
          Component={LazyModelArenaModal}
          props={{
            isOpen: modelArenaModalOpen,
            onClose: () => setActivePackModal(null),
            onOpenInEditor: (workspaceIdArg: string) => {
              openArenaWorkbench(workspaceIdArg, 'arena/model-arena.nj-arena.json');
              setCodeEditorOpen(true);
              void updateLayoutSettings({ editorPanelVisible: true });
            },
          }}
        />
      )}
      {aiInterviewModalOpen && (
        <LazyPanelShell
          panelName="AI interview prep"
          Component={LazyAIInterviewPrepModal}
          props={{ isOpen: aiInterviewModalOpen, onClose: () => setActivePackModal(null) }}
        />
      )}

      <LearningProposalModal
        isOpen={learningProposalOpen}
        proposal={learningProposal}
        serverAddr={hubHttp}
        collaborationId={activeCollabForChannel?.id}
        onClose={() => {
          setLearningProposalOpen(false);
          setLearningProposal(null);
        }}
        onSaved={async (agentId) => {
          addToast({
            type: 'success',
            title: 'Learning saved',
            message: `Saved for ${learningProposal?.agent_name ?? 'expert'}.`,
          });
          try {
            const stats = await api.fetchLearningStats(agentId);
            if (stats.suggest_training || stats.ready_for_lora) {
              addToast({
                type: 'info',
                title: 'Sharpen expert',
                message: `10+ turns with ${learningProposal?.agent_name ?? 'this expert'} — open agent info to sharpen.`,
              });
            }
          } catch {
            /* stats optional */
          }
        }}
      />

      {hubAccessPending && (
        <HubDataAccessModal
          options={hubAccessPending.options}
          isLoading={hubAccessLoading}
          error={hubAccessError}
          onCancel={() => {
            setHubAccessPending(null);
            setHubAccessError(null);
          }}
          onConfirm={handleHubAccessConfirm}
        />
      )}

      {runbookLibraryOpen && (
        <LazyPanelShell
          panelName="Runbook library"
          Component={LazyRunbookLibraryModal}
          props={{
            isOpen: runbookLibraryOpen,
            api,
            hubAgents: agentsToCollaborationAgents(agents),
            channel,
            username: username || 'User',
            onClose: () => setRunbookLibraryOpen(false),
            onInstantiated: (collabId: string, collabChannel: string) => {
              void (async () => {
                if (collabChannel && collabChannel !== channel) {
                  await handleSwitchChannel(collabChannel);
                }
                try {
                  setActiveCollab(await api.getRunbook(collabId));
                } catch {
                  /* snapshot optional */
                }
                void loadCollaborations(channel);
              })();
            },
            onNewBlank: () => void handleCreateBlankRunbook(),
          }}
        />
      )}
    </>
  );
}
