import { create } from 'zustand';
import { ChatAPI } from '../api/chatAPI';
import { getHubBaseURL } from '../config/hubUrl';

export interface GitChangeProposal {
  id: string;
  operation: string;
  message?: string;
  paths?: string[];
  workspace_id?: string;
  channel?: string;
  status?: string;
}

interface GitChangeStore {
  pendingGitChanges: GitChangeProposal[];
  fetchPendingGitChanges: (userId?: string) => Promise<void>;
  approveGitChange: (id: string) => Promise<void>;
  rejectGitChange: (id: string) => Promise<void>;
}

export const useGitChangeStore = create<GitChangeStore>((set, get) => ({
  pendingGitChanges: [],
  fetchPendingGitChanges: async (userId = 'default') => {
    const api = new ChatAPI(getHubBaseURL());
    const rows = (await api.fetchGitChanges(userId)) as unknown as GitChangeProposal[];
    set({ pendingGitChanges: rows.filter((r) => r.status === 'pending' || !r.status) });
  },
  approveGitChange: async (id: string) => {
    const api = new ChatAPI(getHubBaseURL());
    await api.approveGitChange(id);
    await get().fetchPendingGitChanges();
  },
  rejectGitChange: async (id: string) => {
    const api = new ChatAPI(getHubBaseURL());
    await api.rejectGitChange(id);
    await get().fetchPendingGitChanges();
  },
}));
