/** AgentsExtrasApi — extracted from ChatAPI for smaller modules. */
import type { HubFetchFn } from './packsApi';
import type { AgentShareBundle } from '../types/chatApiTypes';

export class AgentsExtrasApi {
  constructor(private readonly hubFetch: HubFetchFn) {}

  async deleteCachedAgent(payload: {
    type: string;
    name: string;
    path?: string;
  }): Promise<void> {
    const response = await this.hubFetch(`/api/my-agents`, {
      method: 'DELETE',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    });
    if (response.status === 404) {
      throw new Error('Cached agent not found');
    }
    if (!response.ok) {
      throw new Error(`Failed to delete cached agent: ${response.statusText}`);
    }
  }

  /**
   * Build a Share Agent bundle (export + custom rules + agent-scoped
   * learnings + LoRA metadata, when present) for a repo agent so it can be
   * offered as a download from Agent Info -> Share.
   */
  async shareAgent(agentId: string): Promise<AgentShareBundle> {
    const response = await this.hubFetch(`/api/agents/${agentId}/share`, { method: 'POST' });
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return response.json();
  }

  /**
   * Import an agent from an MCP export / Share Agent bundle file already
   * accessible on the hub's filesystem. Set `hydrate` to rebuild the
   * agent's knowledge from the bundle's embedded resources instead of
   * re-indexing the original repository path; the hub auto-hydrates when
   * the original path isn't available even if this isn't set.
   */
  async importAgentBundle(options: {
    filePath: string;
    hydrate?: boolean;
    repositoryPath?: string;
  }): Promise<{ success: boolean; message: string; name?: string; lora_train_suggestion?: unknown }> {
    const response = await this.hubFetch('/api/import', {
      method: 'POST',
      body: JSON.stringify({
        file_path: options.filePath,
        hydrate: options.hydrate ?? false,
        repository_path: options.repositoryPath ?? '',
      }),
    });
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return response.json();
  }

  async switchAgentProvider(agentId: string, provider: string, model: string): Promise<void> {
    const response = await this.hubFetch(`/api/agents/${agentId}/provider`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ provider, model }),
    });

    if (!response.ok) {
      throw new Error(`Failed to switch agent provider: ${response.statusText}`);
    }
  }

  async restartConfiguredAgents(): Promise<void> {
    const response = await this.hubFetch(`/api/agents/restart`, { method: 'POST' });
    if (!response.ok) {
      throw new Error(`Failed to restart agents: ${response.statusText}`);
    }
  }

  async fetchCliAgentTypes(): Promise<{ types: string[]; installed: Record<string, boolean> }> {
    const response = await this.hubFetch(`/api/cli-agent-types`);
    if (!response.ok) {
      throw new Error(`Failed to fetch CLI agent types: ${response.status} ${response.statusText}`);
    }
    const text = await response.text();
    let data: unknown;
    try {
      data = JSON.parse(text) as unknown;
    } catch {
      const preview = text.replace(/\s+/g, ' ').slice(0, 160);
      throw new Error(
        `Hub returned non-JSON from /api/cli-agent-types (wrong NEURAL_JUNKIE_HUB_URL / VITE_NJ_HUB_URL or stale sidecar?). ${preview}`
      );
    }
    if (!data || typeof data !== 'object' || !Array.isArray((data as { types?: unknown }).types)) {
      throw new Error('Hub JSON for CLI types is missing a "types" array.');
    }
    const obj = data as { types: string[]; installed?: Record<string, boolean> };
    return { types: obj.types, installed: obj.installed ?? {} };
  }

}
