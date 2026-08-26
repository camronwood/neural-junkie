import type {
  AgentInfo,
  AgentToolCapabilities,
  CachedAgentInfo,
  CapabilityPolicyResponse,
  CapabilityPolicyUpdate,
  ChannelToolsResponse,
} from '../../types/protocol';
import type { HubFetchFn } from './packsApi';

/** Agent roster and capability HTTP surface. */
export class AgentsApi {
  constructor(private readonly hubFetch: HubFetchFn) {}

  async fetchAgents(options?: { includeToolCounts?: boolean }): Promise<AgentInfo[]> {
    const params = new URLSearchParams();
    if (options?.includeToolCounts) {
      params.set('include_tool_counts', 'true');
    }
    const qs = params.toString();
    const response = await this.hubFetch(`/api/agents${qs ? `?${qs}` : ''}`);
    if (!response.ok) {
      throw new Error(`Failed to fetch agents: ${response.statusText}`);
    }
    return response.json();
  }

  async fetchMyAgents(): Promise<CachedAgentInfo[]> {
    const response = await this.hubFetch('/api/my-agents');
    if (!response.ok) {
      throw new Error(`Failed to fetch my agents: ${response.statusText}`);
    }
    const data = await response.json();
    return data.my_agents || [];
  }

  async fetchRemovedAgents(): Promise<AgentInfo[]> {
    const response = await this.hubFetch('/api/removed-agents');
    if (!response.ok) {
      throw new Error(`Failed to fetch removed agents: ${response.statusText}`);
    }
    const data = await response.json();
    return data.removed_agents || [];
  }

  async fetchAgentTools(agentId: string): Promise<AgentToolCapabilities> {
    const response = await this.hubFetch(`/api/agent-tools?agent_id=${encodeURIComponent(agentId)}`);
    if (!response.ok) {
      const detail = await response.text().catch(() => '');
      throw new Error(detail.trim() || `Failed to fetch agent tools: ${response.statusText}`);
    }
    return response.json();
  }

  async fetchChannelTools(channel: string): Promise<ChannelToolsResponse> {
    const response = await this.hubFetch(`/api/channel-tools?channel=${encodeURIComponent(channel)}`);
    if (!response.ok) {
      const detail = await response.text().catch(() => '');
      throw new Error(detail.trim() || `Failed to fetch channel tools: ${response.statusText}`);
    }
    return response.json();
  }

  async fetchCapabilityPolicy(): Promise<CapabilityPolicyResponse> {
    const response = await this.hubFetch('/api/capability-policy');
    if (!response.ok) {
      const detail = await response.text().catch(() => '');
      throw new Error(detail.trim() || `Failed to fetch capability policy: ${response.statusText}`);
    }
    return response.json();
  }

  async updateCapabilityPolicy(update: CapabilityPolicyUpdate): Promise<CapabilityPolicyResponse> {
    const response = await this.hubFetch('/api/capability-policy', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(update),
    });
    if (!response.ok) {
      const detail = await response.text().catch(() => '');
      throw new Error(detail.trim() || `Failed to update capability policy: ${response.statusText}`);
    }
    return response.json();
  }
}
