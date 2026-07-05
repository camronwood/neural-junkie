import { create } from 'zustand';

export interface ProjectSet {
  id: string;
  name: string;
  primary_workspace_id: string;
  member_workspace_ids: string[];
  created_at: string;
}

interface ProjectSetsState {
  projectSets: ProjectSet[];
  activeProjectSetId: string | null;
  loaded: boolean;
  loadProjectSets: () => Promise<void>;
  setActiveProjectSet: (id: string | null) => void;
  getMemberIds: (id: string) => string[];
  createProjectSet: (input: {
    name: string;
    primaryWorkspaceId: string;
    memberWorkspaceIds: string[];
  }) => Promise<ProjectSet | null>;
  updateProjectSet: (
    id: string,
    input: { name?: string; primaryWorkspaceId?: string; memberWorkspaceIds?: string[] }
  ) => Promise<ProjectSet | null>;
  deleteProjectSet: (id: string) => Promise<boolean>;
}

const ACTIVE_KEY = 'nj-active-project-set';

async function hubFetch(path: string, init?: RequestInit): Promise<Response> {
  const { getHubBaseURL } = await import('../config/hubUrl');
  return fetch(`${getHubBaseURL()}${path}`, init);
}

export const useProjectSetsStore = create<ProjectSetsState>((set, get) => ({
  projectSets: [],
  activeProjectSetId: (() => {
    try {
      return localStorage.getItem(ACTIVE_KEY);
    } catch {
      return null;
    }
  })(),
  loaded: false,

  loadProjectSets: async () => {
    try {
      const res = await hubFetch('/api/project-sets');
      if (!res.ok) {
        set({ loaded: true });
        return;
      }
      const data = (await res.json()) as { project_sets?: ProjectSet[] };
      set({ projectSets: data.project_sets ?? [], loaded: true });
    } catch {
      set({ loaded: true });
    }
  },

  setActiveProjectSet: (id) => {
    try {
      if (id) localStorage.setItem(ACTIVE_KEY, id);
      else localStorage.removeItem(ACTIVE_KEY);
    } catch {
      /* ignore */
    }
    set({ activeProjectSetId: id });
  },

  getMemberIds: (id) => {
    const ps = get().projectSets.find((p) => p.id === id);
    if (!ps) return [];
    const seen = new Set<string>();
    const out: string[] = [];
    for (const wsId of ps.member_workspace_ids) {
      if (wsId && wsId !== ps.primary_workspace_id && !seen.has(wsId)) {
        seen.add(wsId);
        out.push(wsId);
      }
    }
    return out;
  },

  createProjectSet: async (input) => {
    try {
      const res = await hubFetch('/api/project-sets/create', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name: input.name,
          primary_workspace_id: input.primaryWorkspaceId,
          member_workspace_ids: input.memberWorkspaceIds,
        }),
      });
      if (!res.ok) return null;
      const ps = (await res.json()) as ProjectSet;
      set((s) => ({ projectSets: [...s.projectSets, ps] }));
      return ps;
    } catch {
      return null;
    }
  },

  updateProjectSet: async (id, input) => {
    try {
      const res = await hubFetch(`/api/project-sets/${encodeURIComponent(id)}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name: input.name,
          primary_workspace_id: input.primaryWorkspaceId,
          member_workspace_ids: input.memberWorkspaceIds,
        }),
      });
      if (!res.ok) return null;
      const ps = (await res.json()) as ProjectSet;
      set((s) => ({
        projectSets: s.projectSets.map((p) => (p.id === id ? ps : p)),
      }));
      return ps;
    } catch {
      return null;
    }
  },

  deleteProjectSet: async (id) => {
    try {
      const res = await hubFetch(`/api/project-sets/${encodeURIComponent(id)}`, {
        method: 'DELETE',
      });
      if (!res.ok) return false;
      set((s) => ({
        projectSets: s.projectSets.filter((p) => p.id !== id),
        activeProjectSetId: s.activeProjectSetId === id ? null : s.activeProjectSetId,
      }));
      if (get().activeProjectSetId === id) {
        try {
          localStorage.removeItem(ACTIVE_KEY);
        } catch {
          /* ignore */
        }
      }
      return true;
    } catch {
      return false;
    }
  },
}));
