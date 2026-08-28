import type { Collaboration } from '../../types/protocol';
import type { HubFetchFn } from './packsApi';

/** Collaboration snapshot, participant, task, and workspace HTTP surface. */
export class CollabApi {
  constructor(private readonly hubFetch: HubFetchFn) {}

  async fetchCollaborations(channel?: string, includeTerminal: boolean = false): Promise<Collaboration[]> {
    const params = new URLSearchParams();
    if (channel) params.set('channel', channel);
    if (includeTerminal) params.set('include_terminal', 'true');
    const query = params.toString();
    const response = await this.hubFetch(`/api/collaborations${query ? `?${query}` : ''}`);
    if (!response.ok) {
      throw new Error(`Failed to fetch collaborations: ${response.statusText}`);
    }
    return response.json();
  }

  async getRunbook(collabId: string): Promise<Collaboration> {
    const response = await this.hubFetch(`/api/runbooks/${encodeURIComponent(collabId)}`);
    if (!response.ok) {
      throw new Error(await response.text() || response.statusText);
    }
    return response.json();
  }

  async approveCollabParticipantRequest(collabId: string, agentId: string): Promise<Collaboration> {
    return this.participantRequestPost(collabId, agentId, 'approve');
  }

  async denyCollabParticipantRequest(collabId: string, agentId: string): Promise<Collaboration> {
    return this.participantRequestPost(collabId, agentId, 'deny');
  }

  async collabTaskComplete(collabId: string, taskId: string): Promise<Collaboration> {
    return this.collabTaskPost(collabId, taskId, 'complete');
  }

  async collabTaskSkip(collabId: string, taskId: string): Promise<Collaboration> {
    return this.collabTaskPost(collabId, taskId, 'skip');
  }

  async collabTaskRedispatch(collabId: string, taskId: string): Promise<Collaboration> {
    return this.collabTaskPost(collabId, taskId, 'redispatch');
  }

  async collabTaskReassign(
    collabId: string,
    taskId: string,
    agentId: string
  ): Promise<Collaboration> {
    const response = await this.hubFetch(
      `/api/collaborations/${encodeURIComponent(collabId)}/tasks/${encodeURIComponent(taskId)}/reassign`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ agent_id: agentId }),
      }
    );
    if (!response.ok) {
      throw new Error(await response.text() || response.statusText);
    }
    return response.json();
  }

  async collabPause(collabId: string): Promise<Collaboration> {
    const response = await this.hubFetch(
      `/api/collaborations/${encodeURIComponent(collabId)}/pause`,
      { method: 'POST' }
    );
    if (!response.ok) {
      throw new Error(await response.text() || response.statusText);
    }
    return response.json();
  }

  async collabResume(collabId: string): Promise<Collaboration> {
    const response = await this.hubFetch(
      `/api/collaborations/${encodeURIComponent(collabId)}/resume`,
      { method: 'POST' }
    );
    if (!response.ok) {
      throw new Error(await response.text() || response.statusText);
    }
    return response.json();
  }

  async acknowledgeCollaborationWorkspace(
    collaborationId: string,
    sourceRepoPath?: string
  ): Promise<void> {
    const body: { collaboration_id: string; source_repo_path?: string } = {
      collaboration_id: collaborationId,
    };
    if (sourceRepoPath?.trim()) {
      body.source_repo_path = sourceRepoPath.trim();
    }
    const response = await this.hubFetch('/api/collaboration-workspace-ack', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t || response.statusText);
    }
  }

  private async participantRequestPost(
    collabId: string,
    agentId: string,
    action: 'approve' | 'deny'
  ): Promise<Collaboration> {
    const response = await this.hubFetch(
      `/api/collaborations/${encodeURIComponent(collabId)}/participant-requests/${encodeURIComponent(agentId)}/${action}`,
      { method: 'POST' }
    );
    if (!response.ok) {
      throw new Error(await response.text() || response.statusText);
    }
    return response.json();
  }

  private async collabTaskPost(
    collabId: string,
    taskId: string,
    action: string
  ): Promise<Collaboration> {
    const response = await this.hubFetch(
      `/api/collaborations/${encodeURIComponent(collabId)}/tasks/${encodeURIComponent(taskId)}/${action}`,
      { method: 'POST' }
    );
    if (!response.ok) {
      throw new Error(await response.text() || response.statusText);
    }
    return response.json();
  }
}
