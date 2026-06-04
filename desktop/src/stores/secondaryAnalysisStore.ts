import { createWithEqualityFn as create } from 'zustand/traditional';

export interface FolderBasketEntry {
  path: string;
  condition?: string;
}

export interface SecondaryAnalysisHistoryEntry {
  id: string;
  workflow: string;
  status: string;
  output_dir?: string;
  created_at: string;
}

interface SecondaryAnalysisState {
  panelOpen: boolean;
  workflow: string;
  basket: FolderBasketEntry[];
  jobId: string | null;
  jobHistory: SecondaryAnalysisHistoryEntry[];
  setPanelOpen: (open: boolean) => void;
  setWorkflow: (workflow: string) => void;
  addToBasket: (path: string, condition?: string) => void;
  updateBasketCondition: (path: string, condition: string) => void;
  removeFromBasket: (path: string) => void;
  clearBasket: () => void;
  setJobId: (id: string | null) => void;
  setJobHistory: (entries: SecondaryAnalysisHistoryEntry[]) => void;
  appendJobHistory: (entry: SecondaryAnalysisHistoryEntry) => void;
}

export const useSecondaryAnalysisStore = create<SecondaryAnalysisState>((set, get) => ({
  panelOpen: false,
  workflow: 'comparator',
  basket: [],
  jobId: null,
  jobHistory: [],
  setPanelOpen: (open) => set({ panelOpen: open }),
  setWorkflow: (workflow) => set({ workflow }),
  addToBasket: (path, condition) => {
    const p = path.trim();
    if (!p) return;
    const basket = get().basket.filter((e) => e.path !== p);
    basket.push({ path: p, condition: condition ?? `Plate ${basket.length + 1}` });
    set({ basket });
  },
  updateBasketCondition: (path, condition) =>
    set({
      basket: get().basket.map((e) => (e.path === path ? { ...e, condition } : e)),
    }),
  removeFromBasket: (path) =>
    set({ basket: get().basket.filter((e) => e.path !== path) }),
  clearBasket: () => set({ basket: [] }),
  setJobId: (jobId) => set({ jobId }),
  setJobHistory: (jobHistory) => set({ jobHistory }),
  appendJobHistory: (entry) => {
    const next = [entry, ...get().jobHistory.filter((e) => e.id !== entry.id)].slice(0, 20);
    set({ jobHistory: next });
  },
}));

export const SECONDARY_ANALYSIS_HISTORY_FILE = '.neural-junkie/secondary-analysis-history.json';
