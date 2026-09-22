/** WorkspaceApi — extracted from ChatAPI for smaller modules. */
import type { HubFetchFn } from './packsApi';

export class WorkspaceApi {
  constructor(private readonly hubFetch: HubFetchFn) {}

  async fetchWorkspaces(): Promise<any[]> {
    const response = await this.hubFetch(`/api/workspaces`);
    
    if (!response.ok) {
      throw new Error(`Failed to fetch workspaces: ${response.statusText}`);
    }
    
    return response.json();
  }

  async addWorkspace(
    name: string,
    path: string,
    options?: { create?: boolean; parentPath?: string },
  ): Promise<any> {
    const response = await this.hubFetch(`/api/workspaces`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        name,
        path,
        create: options?.create === true,
        parent_path: options?.parentPath?.trim() || undefined,
      }),
    });

    if (!response.ok) {
      throw new Error(`Failed to add workspace: ${response.statusText}`);
    }
    
    return response.json();
  }

  async removeWorkspace(workspaceId: string): Promise<void> {
    const response = await this.hubFetch(`/api/workspaces?id=${encodeURIComponent(workspaceId)}`, {
      method: 'DELETE',
    });

    if (!response.ok) {
      throw new Error(`Failed to remove workspace: ${response.statusText}`);
    }
  }

  async connectRemoteWorkspace(params: {
    name: string;
    remoteHost: string;
    remoteUser: string;
    remotePath: string;
    sidecarUrl: string;
    token: string;
    kind?: 'ssh' | 'devcontainer';
  }): Promise<any> {
    const response = await this.hubFetch('/api/workspaces/connect-remote', {
      method: 'POST',
      body: JSON.stringify({
        name: params.name,
        remote_host: params.remoteHost,
        remote_user: params.remoteUser,
        remote_path: params.remotePath,
        sidecar_url: params.sidecarUrl,
        token: params.token,
        kind: params.kind ?? 'ssh',
      }),
    });
    if (!response.ok) {
      const text = await response.text();
      throw new Error(text || `Failed to connect remote workspace: ${response.statusText}`);
    }
    return response.json();
  }

  async fetchDevcontainerPlan(workspaceId: string): Promise<any> {
    const response = await this.hubFetch(
      `/api/workspaces/devcontainer-plan?workspace=${encodeURIComponent(workspaceId)}`
    );
    if (!response.ok) {
      throw new Error(`Failed to fetch devcontainer plan: ${response.statusText}`);
    }
    return response.json();
  }

  async fetchDevcontainerPlanByPath(repoPath: string): Promise<any> {
    const response = await this.hubFetch(
      `/api/workspaces/devcontainer-plan?path=${encodeURIComponent(repoPath)}`
    );
    if (!response.ok) {
      const text = await response.text();
      throw new Error(text || `Failed to fetch devcontainer plan: ${response.statusText}`);
    }
    return response.json();
  }

  async pingSidecar(sidecarUrl: string, token?: string): Promise<boolean> {
    const url = `${sidecarUrl.replace(/\/$/, '')}/health`;
    const headers: Record<string, string> = {};
    if (token) headers.Authorization = `Bearer ${token}`;
    try {
      const res = await fetch(url, { headers });
      return res.ok;
    } catch {
      return false;
    }
  }

}
