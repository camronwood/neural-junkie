import { createWithEqualityFn as create } from 'zustand/traditional';

export type ShortcutOverlayType =
  | 'settings'
  | 'commandPalette'
  | 'modelLibrary'
  | 'quickOpen'
  | 'symbol'
  | 'fastEdit'
  | 'createChannel'
  | 'createNewDm'
  | 'channelInfo'
  | 'agentInfo'
  | 'git'
  | 'phoenix'
  | 'problems'
  | 'pendingChanges'
  | 'workspaceSwitcher'
  | 'learningProposal'
  | 'mermaid'
  | 'runbookGraph'
  | 'runbookImport'
  | 'hubDataAccess'
  | 'thread'
  | 'chatFind';

export interface ShortcutOverlayEntry {
  type: ShortcutOverlayType;
  onClose: () => void;
}

interface ShortcutContextState {
  stack: ShortcutOverlayEntry[];
  pushOverlay: (entry: ShortcutOverlayEntry) => void;
  popOverlay: (type: ShortcutOverlayType) => void;
  closeTopOverlay: () => boolean;
  hasOverlay: () => boolean;
}

export const useShortcutContextStore = create<ShortcutContextState>((set, get) => ({
  stack: [],

  pushOverlay: (entry) => {
    set((state) => {
      const without = state.stack.filter((e) => e.type !== entry.type);
      return { stack: [...without, entry] };
    });
  },

  popOverlay: (type) => {
    set((state) => ({ stack: state.stack.filter((e) => e.type !== type) }));
  },

  closeTopOverlay: () => {
    const stack = get().stack;
    const top = stack[stack.length - 1];
    if (!top) return false;
    top.onClose();
    return true;
  },

  hasOverlay: () => get().stack.length > 0,
}));
