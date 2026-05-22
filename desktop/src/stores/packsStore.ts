import { createWithEqualityFn as create } from 'zustand/traditional';
import { ChatAPI, type PackStatus } from '../api/chatAPI';
import { getHubBaseURL } from '../config/hubUrl';

export const PACK_LIFE_SCIENCES = 'life-sciences';
export const PACK_SOFTWARE_DEVELOPMENT = 'software-development';

interface PacksState {
  packs: PackStatus[];
  loading: boolean;
  error: string | null;
  fetchPacks: () => Promise<void>;
  lifeSciencesEnabled: () => boolean;
  softwareDevelopmentEnabled: () => boolean;
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

  softwareDevelopmentEnabled: () => {
    const pack = get().packs.find((p) => p.id === PACK_SOFTWARE_DEVELOPMENT);
    return pack?.enabled === true;
  },
}));
