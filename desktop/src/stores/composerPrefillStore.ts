import { create } from 'zustand';

interface ComposerPrefillState {
  pendingText: string | null;
  requestPrefill: (text: string) => void;
  consumePrefill: () => string | null;
}

export const useComposerPrefillStore = create<ComposerPrefillState>((set, get) => ({
  pendingText: null,
  requestPrefill: (text) => set({ pendingText: text }),
  consumePrefill: () => {
    const text = get().pendingText;
    if (text) set({ pendingText: null });
    return text;
  },
}));
