import { useShortcutOverlay } from '../shortcuts/useShortcutOverlay';

export interface ChatShortcutOverlayState {
  commandPaletteOpen: boolean;
  onCloseCommandPalette: () => void;
  quickOpenOpen: boolean;
  devPackEnabled: boolean;
  onCloseQuickOpen: () => void;
  symbolModalOpen: boolean;
  onCloseSymbol: () => void;
  fastEditOpen: boolean;
  onCloseFastEdit: () => void;
  createChannelOpen: boolean;
  onCloseCreateChannel: () => void;
  createNewDmOpen: boolean;
  onCloseCreateNewDm: () => void;
  channelInfoModal: unknown;
  onCloseChannelInfo: () => void;
  gitModalOpen: boolean;
  onCloseGit: () => void;
  problemsOpen: boolean;
  onCloseProblems: () => void;
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
  useShortcutOverlay('commandPalette', state.commandPaletteOpen, state.onCloseCommandPalette);
  useShortcutOverlay(
    'quickOpen',
    state.quickOpenOpen && state.devPackEnabled,
    state.onCloseQuickOpen
  );
  useShortcutOverlay(
    'symbol',
    state.symbolModalOpen && state.devPackEnabled,
    state.onCloseSymbol
  );
  useShortcutOverlay(
    'fastEdit',
    state.fastEditOpen && state.devPackEnabled,
    state.onCloseFastEdit
  );
  useShortcutOverlay('createChannel', state.createChannelOpen, state.onCloseCreateChannel);
  useShortcutOverlay('createNewDm', state.createNewDmOpen, state.onCloseCreateNewDm);
  useShortcutOverlay('channelInfo', Boolean(state.channelInfoModal), state.onCloseChannelInfo);
  useShortcutOverlay(
    'git',
    state.gitModalOpen && state.devPackEnabled,
    state.onCloseGit
  );
  useShortcutOverlay(
    'problems',
    state.problemsOpen && state.devPackEnabled,
    state.onCloseProblems
  );
  useShortcutOverlay('phoenix', state.phoenixModalOpen, state.onClosePhoenix);
  useShortcutOverlay('learningProposal', state.learningProposalOpen, state.onCloseLearningProposal);
  useShortcutOverlay('hubDataAccess', Boolean(state.hubAccessPending), state.onCloseHubAccess);
  useShortcutOverlay('chatFind', state.chatFindOpen, state.onCloseChatFind);
  useShortcutOverlay('thread', Boolean(state.openThreadId), state.onCloseThread);
}
