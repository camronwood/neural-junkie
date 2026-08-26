import type { Collaboration } from '../../types/protocol';
import type { HubFetchFn } from './packsApi';

/** Collaboration snapshot and participant HTTP surface. */
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
}
