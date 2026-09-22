/** PlansApi — extracted from ChatAPI for smaller modules. */
import type { HubFetchFn } from './packsApi';

export class PlansApi {
  constructor(private readonly hubFetch: HubFetchFn) {}

  async setAgentApprovalMode(agentId: string, mode: 'interactive' | 'auto_edit' | 'yolo'): Promise<void> {
    const response = await this.hubFetch(`/api/agents/${agentId}/approval-mode`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ mode }),
    });

    if (!response.ok) {
      throw new Error(`Failed to set approval mode: ${response.statusText}`);
    }
  }

  async setAgentCustomRulesMarkdown(agentId: string, markdown: string): Promise<void> {
    const response = await this.hubFetch(`/api/agents/${encodeURIComponent(agentId)}/rules`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ markdown }),
    });

    if (!response.ok) {
      throw new Error(`Failed to save agent rules: ${response.statusText}`);
    }
  }

  async setUserRulesMarkdown(markdown: string): Promise<void> {
    const response = await this.hubFetch('/api/user-rules', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ markdown }),
    });

    if (!response.ok) {
      throw new Error(`Failed to save user rules: ${response.statusText}`);
    }
  }

  async getUserRulesMarkdown(): Promise<string> {
    const response = await this.hubFetch('/api/user-rules');
    if (!response.ok) {
      throw new Error(`Failed to load user rules: ${response.statusText}`);
    }
    const data = (await response.json()) as { markdown?: string };
    return data.markdown ?? '';
  }

  async getPlan(id: string): Promise<{
    id: string;
    name: string;
    overview: string;
    todos: Array<{ id: string; content: string; status: string }>;
    markdown: string;
  }> {
    const response = await this.hubFetch(`/api/plans/${encodeURIComponent(id)}`);
    if (!response.ok) {
      throw new Error(`Failed to load plan: ${response.statusText}`);
    }
    return response.json();
  }

  async putPlan(
    id: string,
    markdown: string
  ): Promise<{
    id: string;
    name: string;
    overview: string;
    todos: Array<{ id: string; content: string; status: string }>;
    markdown: string;
  }> {
    const response = await this.hubFetch(`/api/plans/${encodeURIComponent(id)}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ markdown }),
    });
    if (!response.ok) {
      throw new Error((await response.text()) || `Failed to save plan: ${response.statusText}`);
    }
    return response.json();
  }

}
