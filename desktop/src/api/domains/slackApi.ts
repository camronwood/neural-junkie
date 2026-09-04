import type {
  SlackBinding,
  SlackChannelInfo,
  SlackConfigResponse,
  SlackConnectionResponse,
  SlackDiagnoseResult,
  SlackInboxConfig,
  SlackPolicy,
  SlackSmokeResult,
  SlackStatus,
} from '../../types/protocol';
import type { HubFetchFn } from './packsApi';

/** Slack bridge config, bindings, inbox, and diagnostics HTTP surface. */
export class SlackApi {
  constructor(private readonly hubFetch: HubFetchFn) {}

  async getSlackConfig(): Promise<SlackConfigResponse> {
    const response = await this.hubFetch(`/api/slack/config`);
    if (!response.ok) {
      throw new Error(`Failed to fetch Slack config: ${response.statusText}`);
    }
    return response.json();
  }

  async saveSlackConfig(body: {
    enabled?: boolean;
    app_token?: string;
    bot_token?: string;
    display_name?: string;
    display_icon_url?: string;
    default_policy?: SlackPolicy;
    client_id?: string;
    client_secret?: string;
    redirect_url?: string;
  }): Promise<{ status: string }> {
    const response = await this.hubFetch(`/api/slack/config`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    const data = await response.json();
    if (!response.ok) {
      throw new Error(data.error || `Failed to save Slack config: ${response.statusText}`);
    }
    return data;
  }

  async getSlackStatus(): Promise<SlackStatus> {
    const response = await this.hubFetch(`/api/slack/status`);
    if (!response.ok) {
      throw new Error(`Failed to fetch Slack status: ${response.statusText}`);
    }
    return response.json();
  }

  async getSlackConnection(): Promise<SlackConnectionResponse> {
    const response = await this.hubFetch(`/api/slack/connection`);
    if (!response.ok) {
      throw new Error(`Failed to fetch Slack connection: ${response.statusText}`);
    }
    return response.json();
  }

  async getSlackBindings(): Promise<SlackBinding[]> {
    const response = await this.hubFetch(`/api/slack/bindings`);
    if (!response.ok) {
      throw new Error(`Failed to fetch Slack bindings: ${response.statusText}`);
    }
    return response.json();
  }

  async getSlackChannels(): Promise<SlackChannelInfo[]> {
    const response = await this.hubFetch(`/api/slack/channels`);
    const data = await response.json();
    if (!response.ok) {
      throw new Error(
        typeof data?.error === 'string' ? data.error : `Failed to list Slack channels: ${response.statusText}`
      );
    }
    return Array.isArray(data) ? data : [];
  }

  async saveSlackBinding(binding: {
    slack_channel_id: string;
    slack_channel_name?: string;
    agent_id: string;
    agent_name?: string;
    policy?: SlackPolicy;
    enabled?: boolean;
  }): Promise<SlackBinding> {
    const response = await this.hubFetch(`/api/slack/bindings`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(binding),
    });
    const data = await response.json();
    if (!response.ok) {
      throw new Error(data.error || `Failed to save Slack binding: ${response.statusText}`);
    }
    return data;
  }

  async deleteSlackBinding(slackChannelId: string): Promise<void> {
    const response = await this.hubFetch(
      `/api/slack/bindings?slack_channel_id=${encodeURIComponent(slackChannelId)}`,
      { method: 'DELETE' }
    );
    if (!response.ok) {
      const data = await response.json().catch(() => ({}));
      throw new Error(data.error || `Failed to delete binding: ${response.statusText}`);
    }
  }

  async getSlackOAuthURL(): Promise<string> {
    const response = await this.hubFetch(`/api/slack/oauth/start?json=1`);
    const data = await response.json();
    if (!response.ok) {
      throw new Error(data.error || `Failed to get Slack OAuth URL: ${response.statusText}`);
    }
    return data.url;
  }

  async getSlackUserDMOAuthURL(): Promise<string> {
    const response = await this.hubFetch(`/api/slack/oauth/user-dm/start?json=1`);
    const data = await response.json();
    if (!response.ok) {
      throw new Error(data.error || `Failed to get Slack user DM OAuth URL: ${response.statusText}`);
    }
    return data.url;
  }

  async disconnectSlack(): Promise<void> {
    const response = await this.hubFetch(`/api/slack/disconnect`, { method: 'POST' });
    if (!response.ok) {
      const data = await response.json().catch(() => ({}));
      throw new Error(data.error || `Slack disconnect failed: ${response.statusText}`);
    }
  }

  async restartSlackBridge(): Promise<void> {
    const response = await this.hubFetch(`/api/slack/restart`, { method: 'POST' });
    if (!response.ok) {
      const data = await response.json().catch(() => ({}));
      throw new Error(data.error || `Slack restart failed: ${response.statusText}`);
    }
  }

  async getSlackInbox(): Promise<SlackInboxConfig> {
    const response = await this.hubFetch(`/api/slack/inbox`);
    if (!response.ok) {
      throw new Error(`Failed to fetch Slack inbox: ${response.statusText}`);
    }
    return response.json();
  }

  async saveSlackInbox(body: SlackInboxConfig): Promise<SlackInboxConfig> {
    const response = await this.hubFetch(`/api/slack/inbox`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    const data = await response.json();
    if (!response.ok) {
      throw new Error(data.error || `Failed to save Slack inbox: ${response.statusText}`);
    }
    return data;
  }

  /** Toggle manual away mode for human DM away (GET + merge + PUT). */
  async setSlackInboxAwayEnabled(awayEnabled: boolean): Promise<SlackInboxConfig> {
    const current = await this.getSlackInbox();
    return this.saveSlackInbox({
      ...current,
      human_dm_away: {
        ...current.human_dm_away,
        away_enabled: awayEnabled,
      },
    });
  }

  /** Toggle channel message forwarding into the personal inbox (reply from NJ). */
  async setSlackInboxForwardEnabled(forwardEnabled: boolean): Promise<SlackInboxConfig> {
    const current = await this.getSlackInbox();
    return this.saveSlackInbox({
      ...current,
      forward_enabled: forwardEnabled,
    });
  }

  async testSlackInboxDM(text?: string): Promise<void> {
    const response = await this.hubFetch(`/api/slack/inbox/test-dm`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ text: text ?? '' }),
    });
    const data = await response.json();
    if (!response.ok) {
      throw new Error(data.error || `Failed to send inbox test DM: ${response.statusText}`);
    }
  }

  async slackTestPost(slackChannelId: string, text?: string): Promise<void> {
    const response = await this.hubFetch(`/api/slack/test-post`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ slack_channel_id: slackChannelId, text: text ?? '' }),
    });
    const data = await response.json();
    if (!response.ok) {
      throw new Error(data.error || `Slack test post failed: ${response.statusText}`);
    }
  }

  async getSlackDiagnose(): Promise<SlackDiagnoseResult> {
    const response = await this.hubFetch(`/api/slack/diagnose`);
    if (!response.ok) {
      throw new Error(`Slack diagnose failed: ${response.statusText}`);
    }
    return response.json();
  }

  async runSlackSmoke(options?: {
    channel_id?: string;
    outbound?: boolean;
  }): Promise<SlackSmokeResult> {
    const response = await this.hubFetch(`/api/slack/smoke/run`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        channel_id: options?.channel_id,
        outbound: options?.outbound ?? false,
        allow_outbound: options?.outbound ?? false,
      }),
    });
    const data = await response.json();
    if (!response.ok && !data.checks) {
      throw new Error(data.error || `Slack smoke failed: ${response.statusText}`);
    }
    return data;
  }
}
