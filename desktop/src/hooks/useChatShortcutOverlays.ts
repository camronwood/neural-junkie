import { shallow } from 'zustand/shallow';
import { useShortcutOverlay } from '../shortcuts/useShortcutOverlay';
import { useIdeOverlayStore } from '../stores/ideOverlayStore';

export interface ChatShortcutOverlayState {
  commandPaletteOpen: boolean;
  onCloseCommandPalette: () => void;
  ideEnabled: boolean;
  createChannelOpen: boolean;
  onCloseCreateChannel: () => void;
  createNewDmOpen: boolean;
  onCloseCreateNewDm: () => void;
  channelInfoModal: unknown;
  onCloseChannelInfo: () => void;
  phoenixModalOpen: boolean;
  onClosePhoenix: () => void;
  learningProposalOpen: boolean;
  onCloseLearningProposal: () => void;
  hubAccessPending: unknown;
  onCloseHubAccess: () => void;
  chatFindOpen: boolean;
  onCloseChatFind: () => void;
  openThreadId: string | null;
  onCloseThread: () => void;
}

export function useChatShortcutOverlays(state: ChatShortcutOverlayState) {
  const {
    quickOpenOpen,
    symbolModalOpen,
    fastEditOpen,
    gitModalOpen,
    problemsOpen,
    setQuickOpenOpen,
    setSymbolModalOpen,
    setFastEditOpen,
    setGitModalOpen,
    setProblemsOpen,
  } = useIdeOverlayStore(
    (s) => ({
      quickOpenOpen: s.quickOpenOpen,
      symbolModalOpen: s.symbolModalOpen,
      fastEditOpen: s.fastEditOpen,
      gitModalOpen: s.gitModalOpen,
      problemsOpen: s.problemsOpen,
      setQuickOpenOpen: s.setQuickOpenOpen,
      setSymbolModalOpen: s.setSymbolModalOpen,
      setFastEditOpen: s.setFastEditOpen,
      setGitModalOpen: s.setGitModalOpen,
      setProblemsOpen: s.setProblemsOpen,
    }),
    shallow
  );

  useShortcutOverlay('commandPalette', state.commandPaletteOpen, state.onCloseCommandPalette);
  useShortcutOverlay(
    'quickOpen',
    quickOpenOpen && state.ideEnabled,
    () => setQuickOpenOpen(false)
  );
  useShortcutOverlay(
    'symbol',
    symbolModalOpen && state.ideEnabled,
    () => setSymbolModalOpen(false)
  );
  useShortcutOverlay(
    'fastEdit',
    fastEditOpen && state.ideEnabled,
    () => setFastEditOpen(false)
  );
  useShortcutOverlay('createChannel', state.createChannelOpen, state.onCloseCreateChannel);
  useShortcutOverlay('createNewDm', state.createNewDmOpen, state.onCloseCreateNewDm);
  useShortcutOverlay('channelInfo', Boolean(state.channelInfoModal), state.onCloseChannelInfo);
  useShortcutOverlay(
    'git',
    gitModalOpen && state.ideEnabled,
    () => setGitModalOpen(false)
  );
  useShortcutOverlay(
    'problems',
    problemsOpen && state.ideEnabled,
    () => setProblemsOpen(false)
  );
  useShortcutOverlay('phoenix', state.phoenixModalOpen, state.onClosePhoenix);
  useShortcutOverlay('learningProposal', state.learningProposalOpen, state.onCloseLearningProposal);
  useShortcutOverlay('hubDataAccess', Boolean(state.hubAccessPending), state.onCloseHubAccess);
  useShortcutOverlay('chatFind', state.chatFindOpen, state.onCloseChatFind);
  useShortcutOverlay('thread', Boolean(state.openThreadId), state.onCloseThread);
}
