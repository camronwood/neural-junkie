import { create } from 'zustand';

export type ModelTransferSource = 'ollama' | 'huggingface';

export interface ModelTransfer {
  id: string;
  source: ModelTransferSource;
  title: string;
  subtitle: string;
  progressLabel?: string;
  percent?: number;
  status: 'downloading' | 'error' | 'complete';
  error?: string;
  updatedAt: number;
}

interface ModelTransferState {
  transfers: Record<string, ModelTransfer>;
  start: (transfer: Omit<ModelTransfer, 'status' | 'updatedAt'> & { status?: ModelTransfer['status'] }) => void;
  update: (
    id: string,
    patch: Partial<Pick<ModelTransfer, 'progressLabel' | 'percent' | 'title' | 'subtitle'>>
  ) => void;
  fail: (id: string, error: string) => void;
  complete: (id: string) => void;
  remove: (id: string) => void;
  clearFinished: () => void;
}

export const useModelTransferStore = create<ModelTransferState>((set) => ({
  transfers: {},

  start: (transfer) =>
    set((state) => ({
      transfers: {
        ...state.transfers,
        [transfer.id]: {
          ...transfer,
          status: transfer.status ?? 'downloading',
          updatedAt: Date.now(),
        },
      },
    })),

  update: (id, patch) =>
    set((state) => {
      const cur = state.transfers[id];
      if (!cur) return state;
      return {
        transfers: {
          ...state.transfers,
          [id]: { ...cur, ...patch, status: 'downloading', updatedAt: Date.now() },
        },
      };
    }),

  fail: (id, error) =>
    set((state) => {
      const cur = state.transfers[id];
      if (!cur) return state;
      return {
        transfers: {
          ...state.transfers,
          [id]: { ...cur, status: 'error', error, progressLabel: error, updatedAt: Date.now() },
        },
      };
    }),

  complete: (id) =>
    set((state) => {
      const next = { ...state.transfers };
      delete next[id];
      return { transfers: next };
    }),

  remove: (id) =>
    set((state) => {
      const next = { ...state.transfers };
      delete next[id];
      return { transfers: next };
    }),

  clearFinished: () =>
    set((state) => {
      const next: Record<string, ModelTransfer> = {};
      for (const [id, t] of Object.entries(state.transfers)) {
        if (t.status === 'downloading') next[id] = t;
      }
      return { transfers: next };
    }),
}));
