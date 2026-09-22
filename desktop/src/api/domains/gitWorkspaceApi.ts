/** GitWorkspaceApi — extracted from ChatAPI for smaller modules. */
import type { HubFetchFn } from './packsApi';

export class GitWorkspaceApi {
  constructor(private readonly hubFetch: HubFetchFn) {}

  async getGitStatus(workspaceId: string): Promise<any> {
    const response = await this.hubFetch(`/api/git-status?workspace=${encodeURIComponent(workspaceId)}`, {
      method: 'POST',
    });

    if (!response.ok) {
      throw new Error(`Failed to get git status: ${response.statusText}`);
    }
    
    return response.json();
  }

  async getGitDiff(workspaceId: string, path: string, staged = false): Promise<string> {
    const params = new URLSearchParams({
      workspace: workspaceId,
      path,
    });
    if (staged) params.set('staged', 'true');
    const response = await this.hubFetch(`/api/git-diff?${params}`, { method: 'POST' });
    if (!response.ok) {
      throw new Error(`Failed to get git diff: ${response.statusText}`);
    }
    const data = await response.json();
    return data.diff;
  }

  async getGitFileSides(
    workspaceId: string,
    path: string,
    staged: boolean
  ): Promise<{ original: string; modified: string }> {
    const params = new URLSearchParams({
      workspace: workspaceId,
      path,
    });
    if (staged) params.set('staged', 'true');
    const response = await this.hubFetch(`/api/git-file-sides?${params}`, { method: 'GET' });
    if (!response.ok) {
      throw new Error(`Failed to get file sides: ${response.statusText}`);
    }
    return response.json();
  }

  async gitAdd(workspaceId: string, paths: string[]): Promise<void> {
    const response = await this.hubFetch('/api/git-add', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ workspace_id: workspaceId, paths }),
    });
    if (!response.ok) {
      throw new Error(`Failed to stage: ${response.statusText}`);
    }
  }

  async gitReset(workspaceId: string, paths: string[]): Promise<void> {
    const response = await this.hubFetch('/api/git-reset', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ workspace_id: workspaceId, paths }),
    });
    if (!response.ok) {
      throw new Error(`Failed to unstage: ${response.statusText}`);
    }
  }

  async commitChanges(workspaceId: string, message: string): Promise<void> {
    const response = await this.hubFetch(`/api/git-commit`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        workspace_id: workspaceId,
        message,
      }),
    });

    if (!response.ok) {
      throw new Error(`Failed to commit changes: ${response.statusText}`);
    }
  }

  async pushChanges(workspaceId: string): Promise<void> {
    const response = await this.hubFetch(`/api/git-push`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        workspace_id: workspaceId,
      }),
    });

    if (!response.ok) {
      throw new Error(`Failed to push changes: ${response.statusText}`);
    }
  }

  async pullChanges(workspaceId: string): Promise<void> {
    const response = await this.hubFetch(`/api/git-pull`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        workspace_id: workspaceId,
      }),
    });

    if (!response.ok) {
      throw new Error(`Failed to pull changes: ${response.statusText}`);
    }
  }

}
