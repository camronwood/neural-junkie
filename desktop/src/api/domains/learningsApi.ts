/** LearningsApi — extracted from ChatAPI for smaller modules. */
import type { HubFetchFn } from './packsApi';
import type { LearningCategory, LearningScope, LearningStats, UserLearning } from '../types/chatApiTypes';

export class LearningsApi {
  constructor(private readonly hubFetch: HubFetchFn) {}

  async fetchLearnings(options?: {
    agentId?: string;
    agentType?: string;
    agentName?: string;
  }): Promise<UserLearning[]> {
    const params = new URLSearchParams();
    if (options?.agentId) params.set('agent_id', options.agentId);
    if (options?.agentType) params.set('agent_type', options.agentType);
    if (options?.agentName) params.set('agent_name', options.agentName);
    const q = params.toString() ? `?${params.toString()}` : '';
    const response = await this.hubFetch(`/api/learnings${q}`);
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return response.json();
  }

  async createLearning(body: {
    scope?: LearningScope;
    agent_id: string;
    agent_type?: string;
    agent_name?: string;
    collaboration_id?: string;
    content: string;
    category?: LearningCategory;
    source_channel?: string;
    source_message_id?: string;
  }): Promise<UserLearning> {
    const response = await this.hubFetch(`/api/learnings`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return response.json();
  }

  async deleteLearning(id: string): Promise<void> {
    const response = await this.hubFetch(`/api/learnings/${encodeURIComponent(id)}`, {
      method: 'DELETE',
    });
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
  }

  async fetchLearningStats(agentId: string): Promise<LearningStats> {
    const response = await this.hubFetch(
      `/api/learnings/stats?agent_id=${encodeURIComponent(agentId)}`,
    );
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return response.json();
  }

  async updateLearning(
    id: string,
    body: {
      content?: string;
      category?: LearningCategory;
      scope?: LearningScope;
      collaboration_id?: string;
    },
  ): Promise<UserLearning> {
    const response = await this.hubFetch(`/api/learnings/${encodeURIComponent(id)}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return response.json();
  }

  async queryLearnings(params: {
    q?: string;
    agent_id?: string;
    scope?: LearningScope;
    channel?: string;
    collaboration_id?: string;
  }): Promise<{ query: string; count: number; results: UserLearning[] }> {
    const q = new URLSearchParams();
    if (params.q) q.set('q', params.q);
    if (params.agent_id) q.set('agent_id', params.agent_id);
    if (params.scope) q.set('scope', params.scope);
    if (params.channel) q.set('channel', params.channel);
    if (params.collaboration_id) q.set('collaboration_id', params.collaboration_id);
    const response = await this.hubFetch(`/api/learnings/query?${q.toString()}`);
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return response.json();
  }

  async exportLearnings(): Promise<{ version: number; user_id: string; entries: UserLearning[] }> {
    const response = await this.hubFetch(`/api/learnings/export`, { method: 'POST' });
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return response.json();
  }

  async importLearnings(bundle: { entries: UserLearning[] }): Promise<{ added: number; skipped: number }> {
    const response = await this.hubFetch(`/api/learnings/import`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(bundle),
    });
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return response.json();
  }

}
