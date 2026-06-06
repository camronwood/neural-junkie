import { createWithEqualityFn as create } from 'zustand/traditional';
import type { ShortcutHandlerMap } from '../shortcuts/types';

type PartialHandlers = Partial<ShortcutHandlerMap>;

interface ShortcutHandlersState {
  handlers: PartialHandlers;
  gates: {
    devPackEnabled: boolean;
    ideLayout: boolean;
    codeEditorOpen: boolean;
    showAgentStop: boolean;
    hasPendingApprovals: boolean;
    threadOpen: boolean;
    terminalFocused: boolean;
    monacoFocused: boolean;
    chatConnected: boolean;
  };
  registerHandlers: (handlers: PartialHandlers) => void;
  unregisterHandlers: (ids: Array<keyof ShortcutHandlerMap>) => void;
  setGates: (gates: Partial<ShortcutHandlersState['gates']>) => void;
}

const noop = () => {};

const defaultHandlers: ShortcutHandlerMap = {
  openSettings: noop,
  toggleTerminal: noop,
  openCommandPalette: noop,
  openModelLibrary: noop,
  openChatFind: noop,
  handleEscape: noop,
  toggleChannelSidebar: noop,
  toggleFileExplorer: noop,
  toggleTaskManagement: noop,
  toggleGitPanel: noop,
  toggleProblemsPanel: noop,
  togglePendingChanges: noop,
  toggleMyAgents: noop,
  toggleChatPanel: noop,
  toggleToolbarSidebar: noop,
  toggleIdeLayout: noop,
  prevChannel: noop,
  nextChannel: noop,
  focusChannelSearch: noop,
  createChannel: noop,
  createNewDm: noop,
  openWorkspaceSwitcher: noop,
  quickOpen: noop,
  goToSymbol: noop,
  fastEdit: noop,
  focusComposer: noop,
  saveActiveTab: noop,
  saveAllTabs: noop,
  closeActiveTab: noop,
  openCodeEditor: noop,
  cycleEditorTabForward: noop,
  cycleEditorTabBackward: noop,
  newRunbook: noop,
  approveFirstPending: noop,
  rejectFirstPending: noop,
  closeThread: noop,
  clearTerminal: noop,
};

export const useShortcutHandlersStore = create<ShortcutHandlersState>((set) => ({
  handlers: { ...defaultHandlers },
  gates: {
    devPackEnabled: false,
    ideLayout: false,
    codeEditorOpen: false,
    showAgentStop: false,
    hasPendingApprovals: false,
    threadOpen: false,
    terminalFocused: false,
    monacoFocused: false,
    chatConnected: false,
  },

  registerHandlers: (handlers) => {
    set((state) => ({
      handlers: { ...state.handlers, ...handlers },
    }));
  },

  unregisterHandlers: (ids) => {
    set((state) => {
      const next = { ...state.handlers };
      for (const id of ids) {
        next[id] = defaultHandlers[id];
      }
      return { handlers: next };
    });
  },

  setGates: (gates) => {
    set((state) => ({ gates: { ...state.gates, ...gates } }));
  },
}));

export function getShortcutHandler(id: keyof ShortcutHandlerMap) {
  const handler = useShortcutHandlersStore.getState().handlers[id];
  return handler ?? defaultHandlers[id];
}

export function getShortcutGates() {
  return useShortcutHandlersStore.getState().gates;
}
