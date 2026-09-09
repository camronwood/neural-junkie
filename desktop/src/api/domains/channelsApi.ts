import type { Channel } from '../../types/protocol';
import { buildChannelWebSocketURL, buildThreadWebSocketURL } from '../chatAPI/wsUrl';
import type { HubFetchFn } from './packsApi';

/** Channel list, create/DM, history, membership, and WebSocket URL surface. */
export class ChannelsApi {
  constructor(
    private readonly hubFetch: HubFetchFn,
    private readonly baseURL: string,
  ) {}

  async fetchChannels(): Promise<Channel[]> {
    const response = await this.hubFetch('/api/channels');
    if (!response.ok) {
      throw new Error(`Failed to fetch channels: ${response.statusText}`);
    }
    return response.json();
  }

  async createChannel(
    name: string,
    description: string,
    type: 'public' | 'dm' | 'custom',
    members: string[] = [],
    createdBy: string = ''
  ): Promise<Channel> {
    const response = await this.hubFetch(`/api/channels/create`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name, description, type, members, created_by: createdBy }),
    });

    if (!response.ok) {
      if (response.status === 429) {
        throw new Error('Too Many Requests — wait a moment and try again.');
      }
      throw new Error(`Failed to create channel: ${response.statusText}`);
    }

    return response.json();
  }

  /** Find-or-create DM with an agent (rate-limit exempt on the hub). */
  async openDM(agentId: string, createdBy: string): Promise<Channel> {
    const response = await this.hubFetch(`/api/channels/open-dm`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ agent_id: agentId, created_by: createdBy }),
    });

    if (!response.ok) {
      if (response.status === 429) {
        throw new Error('Too Many Requests — wait a moment and try again.');
      }
      const detail = (await response.text()).trim();
      throw new Error(detail || `Failed to open DM: ${response.statusText}`);
    }

    return response.json();
  }

  /** Create a new expert or CLI agent scoped to a fresh DM channel. */
  async createDMAgent(payload: {
    created_by: string;
    mode: 'expert' | 'cli';
    display_name: string;
    expert_type?: string;
    /** Optional extra instructions for custom (non-preset) experts. */
    persona?: string;
    provider_id?: string;
    provider?: string;
    model?: string;
    capability_allow?: string[];
    capability_deny?: string[];
    cli_type?: string;
    work_dir?: string;
  }): Promise<Channel> {
    const body: Record<string, unknown> = {
      created_by: payload.created_by,
      mode: payload.mode,
      display_name: payload.display_name,
    };
    if (payload.mode === 'expert') {
      body.expert_type = payload.expert_type ?? '';
      body.persona = payload.persona ?? '';
      body.provider_id = payload.provider_id ?? '';
      body.provider = payload.provider ?? '';
      body.model = payload.model ?? '';
      body.capability_allow = payload.capability_allow ?? [];
      body.capability_deny = payload.capability_deny ?? [];
    } else {
      body.cli_type = payload.cli_type ?? '';
      body.work_dir = payload.work_dir ?? '';
    }

    const response = await this.hubFetch(`/api/channels/create-dm-agent`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });

    if (!response.ok) {
      const detail = await response.text().catch(() => '');
      throw new Error(detail.trim() || `Failed to create DM agent: ${response.statusText}`);
    }

    return response.json();
  }

  async clearChannelHistory(name: string): Promise<void> {
    const response = await this.hubFetch(`/api/channels/clear-history`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name }),
    });

    if (!response.ok) {
      const text = await response.text();
      throw new Error(text || `Failed to clear channel history: ${response.statusText}`);
    }
  }

  async exportChannelHistory(channel: string, format: 'markdown' | 'json' = 'markdown'): Promise<Blob> {
    const q = new URLSearchParams({ channel, format });
    const response = await this.hubFetch(`/api/channel-export?${q.toString()}`);
    if (!response.ok) {
      const text = await response.text();
      throw new Error(text || `Failed to export channel history: ${response.statusText}`);
    }
    return response.blob();
  }

  async getChannelDurable(channel: string): Promise<boolean> {
    const q = new URLSearchParams({ channel });
    const response = await this.hubFetch(`/api/channel-durable/status?${q.toString()}`);
    if (!response.ok) {
      const text = await response.text();
      throw new Error(text || `Failed to read channel durable flag: ${response.statusText}`);
    }
    const data = (await response.json()) as { durable?: boolean };
    return !!data.durable;
  }

  async setChannelDurable(channel: string, durable: boolean): Promise<void> {
    const response = await this.hubFetch(`/api/channel-durable`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ channel, durable }),
    });
    if (!response.ok) {
      const text = await response.text();
      throw new Error(text || `Failed to update channel durable flag: ${response.statusText}`);
    }
  }

  async deleteChannel(name: string): Promise<void> {
    const response = await this.hubFetch(`/api/channels/delete`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name }),
    });

    if (!response.ok) {
      throw new Error(`Failed to delete channel: ${response.statusText}`);
    }
  }

  async archiveChannel(name: string): Promise<void> {
    const response = await this.hubFetch('/api/channels/archive', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name }),
    });
    if (!response.ok) {
      const detail = await response.text().catch(() => '');
      throw new Error(detail.trim() || `Failed to archive channel: ${response.statusText}`);
    }
  }

  async addAgentsToChannel(channelName: string, agentIds: string[]): Promise<void> {
    const response = await this.hubFetch(
      `/api/channels/agents?channel=${encodeURIComponent(channelName)}`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ agent_ids: agentIds }),
      }
    );

    if (!response.ok) {
      throw new Error(`Failed to add agents to channel: ${response.statusText}`);
    }
  }

  async removeAgentFromChannel(channelName: string, agentId: string): Promise<void> {
    const response = await this.hubFetch(
      `/api/channels/agents?channel=${encodeURIComponent(channelName)}&agent_id=${encodeURIComponent(agentId)}`,
      { method: 'DELETE' }
    );

    if (!response.ok) {
      throw new Error(`Failed to remove agent from channel: ${response.statusText}`);
    }
  }

  async testConnection(): Promise<boolean> {
    try {
      const response = await this.hubFetch(`/api/channels`);
      return response.ok;
    } catch (error) {
      return false;
    }
  }

  getWebSocketURL(channel: string, extraChannels: string[] = []): string {
    return buildChannelWebSocketURL(this.baseURL, channel, extraChannels);
  }

  getThreadWebSocketURL(channel: string, threadId: string): string {
    return buildThreadWebSocketURL(this.baseURL, channel, threadId);
  }
}
