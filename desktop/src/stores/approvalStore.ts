import { createWithEqualityFn as create } from 'zustand/traditional';
import { ChatAPI } from '../api/chatAPI';

export interface PendingToolApproval {
  id: string;
  agentId: string;
  agentName: string;
  toolName: string;
  toolInput: Record<string, unknown>;
  channel: string;
  messageId?: string;
  createdAt: string;
}

interface ApprovalState {
  pendingTools: PendingToolApproval[];
  upsertPendingTool: (approval: PendingToolApproval) => void;
  removePendingTool: (approvalId: string) => void;
  setPendingTools: (approvals: PendingToolApproval[]) => void;
  syncPendingFromHub: (api: ChatAPI) => Promise<void>;
}

export const useApprovalStore = create<ApprovalState>((set, get) => ({
  pendingTools: [],

  upsertPendingTool: (approval) =>
    set((state) => {
      const idx = state.pendingTools.findIndex((a) => a.id === approval.id);
      if (idx >= 0) {
        const next = [...state.pendingTools];
        next[idx] = { ...next[idx], ...approval };
        return { pendingTools: next };
      }
      return { pendingTools: [...state.pendingTools, approval] };
    }),

  removePendingTool: (approvalId) =>
    set((state) => ({
      pendingTools: state.pendingTools.filter((a) => a.id !== approvalId),
    })),

  setPendingTools: (approvals) => set({ pendingTools: approvals }),

  syncPendingFromHub: async (api) => {
    try {
      const rows = await api.fetchPendingToolApprovals();
      const mapped: PendingToolApproval[] = rows.map((row) => ({
        id: row.id,
        agentId: row.agent_id,
        agentName: row.agent_name,
        toolInput: row.tool_input ?? {},
        toolName: row.tool_name,
        channel: row.channel,
        createdAt: row.created_at,
      }));
      get().setPendingTools(mapped);
    } catch {
      // Hub may be unavailable during startup.
    }
  },
}));
