import { create } from 'zustand';
import { ChatAPI, type PackStatus } from '../api/chatAPI';
import { getHubBaseURL } from '../config/hubUrl';

const PACK_LIFE_SCIENCES = 'life-sciences';

interface PacksState {
  packs: PackStatus[];
  loading: boolean;
  error: string | null;
  fetchPacks: () => Promise<void>;
  lifeSciencesEnabled: () => boolean;
}

export const usePacksStore = create<PacksState>((set, get) => ({
  packs: [],
  loading: false,
  error: null,

  fetchPacks: async () => {
    set({ loading: true, error: null });
    try {
      const api = new ChatAPI(getHubBaseURL());
      const packs = await api.fetchPacks();
      set({ packs, loading: false });
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to load domain packs';
      set({ error: message, loading: false });
    }
  },

  lifeSciencesEnabled: () => {
    const pack = get().packs.find((p) => p.id === PACK_LIFE_SCIENCES);
    return pack?.enabled === true;
  },
}));
