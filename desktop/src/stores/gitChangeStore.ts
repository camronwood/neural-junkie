import { create } from 'zustand';
import { ChatAPI } from '../api/chatAPI';
import { getHubBaseURL } from '../config/hubUrl';
import type { GitChangeProposal } from '../types/protocol';

interface GitChangeStore {
  pendingGitChanges: GitChangeProposal[];
  changesById: Record<string, GitChangeProposal>;
  busyById: Record<string, boolean>;
  errorsById: Record<string, string>;
  fetchPendingGitChanges: (userId?: string) => Promise<void>;
  approveGitChange: (id: string) => Promise<void>;
  rejectGitChange: (id: string, reason?: string) => Promise<void>;
}

export const useGitChangeStore = create<GitChangeStore>((set) => ({
  pendingGitChanges: [],
  changesById: {},
  busyById: {},
  errorsById: {},
  fetchPendingGitChanges: async (userId = 'default') => {
    const api = new ChatAPI(getHubBaseURL());
    const rows = await api.fetchGitChanges(userId);
    const pendingGitChanges = rows.filter((row) => row.status === 'pending');
    set((state) => ({
      pendingGitChanges,
      changesById: pendingGitChanges.reduce<Record<string, GitChangeProposal>>(
        (next, proposal) => ({ ...next, [proposal.id]: proposal }),
        { ...state.changesById },
      ),
    }));
  },
  approveGitChange: async (id: string) => {
    set((state) => ({
      busyById: { ...state.busyById, [id]: true },
      errorsById: { ...state.errorsById, [id]: '' },
    }));
    try {
      const api = new ChatAPI(getHubBaseURL());
      const proposal = await api.approveGitChange(id);
      set((state) => ({
        pendingGitChanges: state.pendingGitChanges.filter((row) => row.id !== id),
        changesById: { ...state.changesById, [id]: proposal },
        busyById: { ...state.busyById, [id]: false },
      }));
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to approve Git change';
      set((state) => ({
        busyById: { ...state.busyById, [id]: false },
        errorsById: { ...state.errorsById, [id]: message },
      }));
      throw error;
    }
  },
  rejectGitChange: async (id: string, reason?: string) => {
    set((state) => ({
      busyById: { ...state.busyById, [id]: true },
      errorsById: { ...state.errorsById, [id]: '' },
    }));
    try {
      const api = new ChatAPI(getHubBaseURL());
      const proposal = await api.rejectGitChange(id, reason);
      set((state) => ({
        pendingGitChanges: state.pendingGitChanges.filter((row) => row.id !== id),
        changesById: { ...state.changesById, [id]: proposal },
        busyById: { ...state.busyById, [id]: false },
      }));
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to reject Git change';
      set((state) => ({
        busyById: { ...state.busyById, [id]: false },
        errorsById: { ...state.errorsById, [id]: message },
      }));
      throw error;
    }
  },
}));
