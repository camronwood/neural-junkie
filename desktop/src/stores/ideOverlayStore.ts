/** Overlay visibility for IDE modals / panels (extracted from ChatWindow booleans). */

import { create } from 'zustand';

type IdeOverlayState = {
  gitModalOpen: boolean;
  quickOpenOpen: boolean;
  symbolModalOpen: boolean;
  problemsOpen: boolean;
  fastEditOpen: boolean;
  setGitModalOpen: (open: boolean) => void;
  setQuickOpenOpen: (open: boolean) => void;
  setSymbolModalOpen: (open: boolean) => void;
  setProblemsOpen: (open: boolean) => void;
  setFastEditOpen: (open: boolean) => void;
  closeAll: () => void;
};

export const useIdeOverlayStore = create<IdeOverlayState>((set) => ({
  gitModalOpen: false,
  quickOpenOpen: false,
  symbolModalOpen: false,
  problemsOpen: false,
  fastEditOpen: false,
  setGitModalOpen: (open) => set({ gitModalOpen: open }),
  setQuickOpenOpen: (open) => set({ quickOpenOpen: open }),
  setSymbolModalOpen: (open) => set({ symbolModalOpen: open }),
  setProblemsOpen: (open) => set({ problemsOpen: open }),
  setFastEditOpen: (open) => set({ fastEditOpen: open }),
  closeAll: () =>
    set({
      gitModalOpen: false,
      quickOpenOpen: false,
      symbolModalOpen: false,
      problemsOpen: false,
      fastEditOpen: false,
    }),
}));
