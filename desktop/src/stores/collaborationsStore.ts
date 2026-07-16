import { create } from 'zustand';
import type { Collaboration } from '../types/protocol';
import { mergeCollaborationSnapshotMap } from '../utils/collaborationSnapshots';

type CollaborationsState = {
  byID: Record<string, Collaboration>;
  setByID: (
    updater:
      | Record<string, Collaboration>
      | ((prev: Record<string, Collaboration>) => Record<string, Collaboration>)
  ) => void;
  mergeSnapshot: (
    snapshot: Collaboration,
    pruneTerminalForChannel?: (
      next: Record<string, Collaboration>,
      channel: string
    ) => Record<string, Collaboration>
  ) => void;
  clear: () => void;
};

export const useCollaborationsStore = create<CollaborationsState>((set) => ({
  byID: {},
  setByID: (updater) => {
    set((state) => ({
      byID: typeof updater === 'function' ? updater(state.byID) : updater,
    }));
  },
  mergeSnapshot: (snapshot, pruneTerminalForChannel) => {
    set((state) => ({
      byID: mergeCollaborationSnapshotMap(state.byID, snapshot, pruneTerminalForChannel),
    }));
  },
  clear: () => set({ byID: {} }),
}));

/** Ref-like accessor for sync paths that previously used collaborationsByIDRef. */
export function collaborationsByIDSnapshot(): Record<string, Collaboration> {
  return useCollaborationsStore.getState().byID;
}
