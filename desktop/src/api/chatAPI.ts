import type { Message, AgentInfo, Channel, ThreadMetadata, CachedAgentInfo, ConnectionTestResult, FileChange, FileChangeDiff, CommandDefinition, AssistantStateResponse, GoogleMeetNotesStatus, GoogleMeetNotesAppConfig, SlackConfigResponse, SlackConnectionResponse, SlackStatus, SlackBinding, SlackChannelInfo, SlackPolicy, SlackInboxConfig, Collaboration, CollaborationTask, AssignSuggestion, ExecutionPolicy, GraphLayout, RunbookTemplate, AgentToolCapabilities, ChannelToolsResponse } from '../types/protocol';
import {
  getHubBaseURL,
  hubAuthHeaders,
  hubSessionHeaders,
  normalizeHubBaseURL,
  setHubSessionToken,
} from '../config/hubUrl';

/** Successful POST /api/send response; optional fields when a slash command requests a channel switch. */
export interface SendMessageResponse {
  status?: string;
  collaboration_channel?: string;
  collaboration_id?: string;
}

export interface PackStatus {
  id: string;
  title: string;
  description: string;
  installed: boolean;
  enabled: boolean;
  layout_profile?: string;
  capabilities?: string[];
  expert_slug?: string;
  expert_label?: string;
  version?: string;
}

export interface PackCatalogEntry {
  id: string;
  version: string;
  title: string;
  description: string;
  icon_key?: string;
  publisher?: string;
  builtin?: boolean;
  installed: boolean;
  enabled: boolean;
  lora_adapter_count?: number;
  lora_base_tags?: string[];
}

export interface InstallPackLoRAResult {
  agent_type?: string;
  repo_id: string;
  ollama_tag: string;
  status: string;
  error?: string;
}

export interface InstallPackLoRAsResponse {
  status: string;
  pack_id: string;
  results: InstallPackLoRAResult[];
}

export interface LoraExpertContext {
  agent_id: string;
  agent_name: string;
  agent_type: string;
  source: 'repo' | 'channel' | 'collaboration';
  source_id?: string;
  suggested_base_ollama_tag: string;
  suggested_ollama_tag?: string;
  preview_rows: number;
  min_rows: number;
  ready: boolean;
}

export interface LoraTrainJob {
  id: string;
  status: string;
  source: string;
  source_id: string;
  base_ollama_tag: string;
  ollama_tag: string;
  row_count?: number;
  log_tail?: string[];
  error?: string;
}

export type LearningCategory = 'preference' | 'fact' | 'workflow' | 'communication';
export type LearningScope = 'agent' | 'global' | 'collaboration';

export interface UserLearning {
  id: string;
  scope?: LearningScope;
  user_id?: string;
  agent_id: string;
  agent_type?: string;
  agent_name?: string;
  collaboration_id?: string;
  content: string;
  category: LearningCategory;
  source_channel?: string;
  source_message_id?: string;
  created_at: string;
  confirmed_at: string;
  updated_at?: string;
  use_count?: number;
  active: boolean;
}

export interface LearningStats {
  agent_id: string;
  learning_count: number;
  global_count?: number;
  collab_count?: number;
  embedding_index_ready?: boolean;
  preview_rows: number;
  min_rows: number;
  ready_for_lora: boolean;
}

export interface LearningProposalAction {
  type: 'learning_proposal';
  source?: string;
  agent_id: string;
  agent_name: string;
  agent_type?: string;
  draft?: string;
  category?: LearningCategory;
  scope?: LearningScope;
  source_message_id?: string;
  source_channel?: string;
  collaboration_id?: string;
}

export interface LoraTrainStartRequest {
  source: 'channel' | 'collaboration' | 'repo';
  source_id: string;
  thread_id?: string;
  agent_name?: string;
  agent_id?: string;
  include_learnings?: boolean;
  base_ollama_tag: string;
  ollama_tag: string;
  hyperparams?: { rank?: number; epochs?: number; learning_rate?: number; max_seq_len?: number };
}

export interface PacksAPIResponse {
  packs: PackStatus[];
  layout_owner?: string;
  layout_profile?: string;
  capabilities?: string[];
}

export interface ExpertPresetOption {
  slug: string;
  label: string;
  from_pack?: string;
}

export class ChatAPI {
  private baseURL: string;
  private commandsCache: CommandDefinition[] | null = null;

  constructor(serverAddr: string = getHubBaseURL()) {
    this.baseURL = normalizeHubBaseURL(serverAddr);
  }

  /** JSON + hub token + session for authenticated hub calls. */
  private hubHeaders(extra?: Record<string, string>): Record<string, string> {
    return {
      'Content-Type': 'application/json',
      ...hubAuthHeaders(),
      ...hubSessionHeaders(),
      ...extra,
    };
  }

  /** Hub fetch with auth headers on every request. */
  private hubFetch(path: string, init?: RequestInit): Promise<Response> {
    const extra = (init?.headers as Record<string, string> | undefined) ?? {};
    const url = path.startsWith('http') ? path : `${this.baseURL}${path}`;
    return fetch(url, {
      ...init,
      headers: { ...this.hubHeaders(), ...extra },
    });
  }

  /** Create or refresh a hub user session (channel ACL). */
  async createSession(username: string): Promise<{ token: string; username: string }> {
    const response = await this.hubFetch('/api/auth/session', {
      method: 'POST',
      body: JSON.stringify({ username }),
    });
    if (!response.ok) {
      throw new Error(`Failed to create session: ${response.statusText}`);
    }
    const data = (await response.json()) as { token: string; username: string };
    if (data.token) {
      setHubSessionToken(data.token);
    }
    return data;
  }

  // Fetch existing messages for a channel
  async fetchMessages(channel: string, limit: number = 50, beforeId?: string): Promise<Message[]> {
    const params = new URLSearchParams({ channel, limit: String(limit) });
    if (beforeId?.trim()) {
      params.set('before', beforeId.trim());
    }
    const response = await this.hubFetch(`/api/messages?${params}`);
    
    if (!response.ok) {
      throw new Error(`Failed to fetch messages: ${response.statusText}`);
    }
    
    return response.json();
  }

  // Fetch collaboration snapshots for task/collaboration management UIs
  async fetchCollaborations(channel?: string, includeTerminal: boolean = false): Promise<Collaboration[]> {
    const params = new URLSearchParams();
    if (channel) {
      params.set('channel', channel);
    }
    if (includeTerminal) {
      params.set('include_terminal', 'true');
    }
    const query = params.toString();
    const response = await this.hubFetch(`/api/collaborations${query ? `?${query}` : ''}`);

    if (!response.ok) {
      throw new Error(`Failed to fetch collaborations: ${response.statusText}`);
    }

    return response.json();
  }

  /** Read user-granted files/directories under ~/.neural-junkie for agent context. */
  async readHubDataAccess(
    targets: Array<{ kind: 'file' | 'directory'; relative_path: string }>
  ): Promise<{ root: string; entries: unknown[] }> {
    const response = await this.hubFetch(`/api/hub-data/read`, {
      method: 'POST',
      body: JSON.stringify({ targets }),
    });
    if (!response.ok) {
      const t = await response.text();
      if (response.status === 404) {
        throw new Error(
          'Hub does not expose /api/hub-data/read (404). Restart the hub (`make server`) or rebuild the packaged sidecar (`make build-sidecar`).'
        );
      }
      throw new Error(t.trim() || response.statusText);
    }
    return response.json();
  }

  /** Confirm collaboration sandbox so the hub sends task prompts to agents (after /approve-plan). */
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
    const response = await this.hubFetch(`/api/collaboration-workspace-ack`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t || response.statusText);
    }
  }

  async createRunbook(body: {
    description: string;
    agent_ids: string[];
    channel: string;
    created_by: string;
    tasks?: CollaborationTask[];
    execution_mode?: string;
    source_repo_path?: string;
  }): Promise<{ collaboration_id: string; collaboration_channel: string; collaboration: Collaboration }> {
    const response = await this.hubFetch(`/api/runbooks`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    if (!response.ok) {
      throw new Error(await response.text() || response.statusText);
    }
    return response.json();
  }

  async updateRunbook(
    collabId: string,
    body: {
      title?: string;
      description?: string;
      agent_ids?: string[];
      tasks?: CollaborationTask[];
      execution_policy?: ExecutionPolicy;
      graph_layout?: GraphLayout;
    }
  ): Promise<Collaboration> {
    const response = await this.hubFetch(`/api/runbooks/${encodeURIComponent(collabId)}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    if (!response.ok) {
      throw new Error(await response.text() || response.statusText);
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

  async suggestRunbookAssignee(
    collabId: string,
    title: string,
    description: string
  ): Promise<AssignSuggestion | null> {
    const response = await this.hubFetch(`/api/runbooks/${encodeURIComponent(collabId)}/suggest-assignee`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ title, description }),
      }
    );
    if (!response.ok) {
      throw new Error(await response.text() || response.statusText);
    }
    const data = await response.json();
    if (data.agent_id) {
      return data as AssignSuggestion;
    }
    return null;
  }

  async parseRunbookPlan(collabId: string, markdown: string): Promise<CollaborationTask[]> {
    const response = await this.hubFetch(`/api/runbooks/${encodeURIComponent(collabId)}/parse-plan`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ markdown }),
      }
    );
    if (!response.ok) {
      throw new Error(await response.text() || response.statusText);
    }
    const data = await response.json();
    return data.tasks ?? [];
  }

  async submitRunbook(collabId: string): Promise<Collaboration> {
    const response = await this.hubFetch(`/api/runbooks/${encodeURIComponent(collabId)}/submit`,
      { method: 'POST' }
    );
    if (!response.ok) {
      throw new Error(await response.text() || response.statusText);
    }
    return response.json();
  }

  async startRunbook(collabId: string): Promise<Collaboration> {
    const response = await this.hubFetch(`/api/runbooks/${encodeURIComponent(collabId)}/start`,
      { method: 'POST' }
    );
    if (!response.ok) {
      throw new Error(await response.text() || response.statusText);
    }
    return response.json();
  }

  async listRunbookTemplates(): Promise<RunbookTemplate[]> {
    const response = await this.hubFetch(`/api/runbook-templates`);
    if (!response.ok) {
      throw new Error(await response.text() || response.statusText);
    }
    return response.json();
  }

  async createRunbookFromTemplate(
    templateName: string,
    body: { channel: string; created_by: string; agent_ids: string[] }
  ): Promise<{ collaboration_id: string; collaboration_channel: string; collaboration: Collaboration }> {
    const response = await this.hubFetch(`/api/runbook-templates/${encodeURIComponent(templateName)}/instantiate`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      }
    );
    if (!response.ok) {
      throw new Error(await response.text() || response.statusText);
    }
    return response.json();
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

  async collabTaskReassign(collabId: string, taskId: string, agentId: string): Promise<Collaboration> {
    const response = await this.hubFetch(`/api/collaborations/${encodeURIComponent(collabId)}/tasks/${encodeURIComponent(taskId)}/reassign`,
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
    const response = await this.hubFetch(`/api/collaborations/${encodeURIComponent(collabId)}/pause`,
      { method: 'POST' }
    );
    if (!response.ok) {
      throw new Error(await response.text() || response.statusText);
    }
    return response.json();
  }

  async collabResume(collabId: string): Promise<Collaboration> {
    const response = await this.hubFetch(`/api/collaborations/${encodeURIComponent(collabId)}/resume`,
      { method: 'POST' }
    );
    if (!response.ok) {
      throw new Error(await response.text() || response.statusText);
    }
    return response.json();
  }

  async approveCollabParticipantRequest(collabId: string, agentId: string): Promise<Collaboration> {
    return this.collabParticipantRequestPost(collabId, agentId, 'approve');
  }

  async denyCollabParticipantRequest(collabId: string, agentId: string): Promise<Collaboration> {
    return this.collabParticipantRequestPost(collabId, agentId, 'deny');
  }

  /** Cursor-style Stop: pause agents on a channel until the user sends a message. */
  async channelInterject(channel: string, heldBy?: string): Promise<{ channel: string; held: boolean }> {
    const response = await this.hubFetch(
      `/api/channels/${encodeURIComponent(channel)}/interject`,
      {
        method: 'POST',
        body: JSON.stringify({ held_by: heldBy ?? '' }),
      }
    );
    if (!response.ok) {
      throw new Error(await response.text() || response.statusText);
    }
    return response.json();
  }

  private async collabTaskPost(collabId: string, taskId: string, action: string): Promise<Collaboration> {
    const response = await this.hubFetch(`/api/collaborations/${encodeURIComponent(collabId)}/tasks/${encodeURIComponent(taskId)}/${action}`,
      { method: 'POST' }
    );
    if (!response.ok) {
      throw new Error(await response.text() || response.statusText);
    }
    return response.json();
  }

  private async collabParticipantRequestPost(collabId: string, agentId: string, action: 'approve' | 'deny'): Promise<Collaboration> {
    const response = await this.hubFetch(
      `/api/collaborations/${encodeURIComponent(collabId)}/participant-requests/${encodeURIComponent(agentId)}/${action}`,
      { method: 'POST' }
    );
    if (!response.ok) {
      throw new Error(await response.text() || response.statusText);
    }
    return response.json();
  }

  // Send a message to the server
  async sendMessage(
    channel: string,
    content: string,
    from: { name: string; type: string },
    type: string = 'question',
    credentials?: Record<string, any>
  ): Promise<SendMessageResponse> {
    const body: any = {
      channel,
      content,
      type,
      from,
    };

    // Add credentials to metadata if provided
    if (credentials) {
      body.metadata = { ...credentials };
      const replyTo = credentials.reply_to;
      if (typeof replyTo === 'string' && replyTo.trim()) {
        body.reply_to = replyTo.trim();
      }
    }

    const response = await this.hubFetch(`/api/send`, {
      method: 'POST',
      body: JSON.stringify(body),
    });

    if (!response.ok) {
      throw new Error(`Failed to send message: ${response.statusText}`);
    }

    const text = await response.text();
    if (!text.trim()) {
      return { status: 'ok' };
    }
    try {
      return JSON.parse(text) as SendMessageResponse;
    } catch {
      return { status: 'ok' };
    }
  }

  // Fetch list of active agents
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
    
    const agents = await response.json();
    
    
    return agents;
  }

  async fetchAgentTools(agentId: string): Promise<AgentToolCapabilities> {
    const response = await this.hubFetch(`/api/agent-tools?agent_id=${encodeURIComponent(agentId)}`
    );
    if (!response.ok) {
      const detail = await response.text().catch(() => '');
      throw new Error(detail.trim() || `Failed to fetch agent tools: ${response.statusText}`);
    }
    return response.json();
  }

  async fetchChannelTools(channel: string): Promise<ChannelToolsResponse> {
    const response = await this.hubFetch(`/api/channel-tools?channel=${encodeURIComponent(channel)}`
    );
    if (!response.ok) {
      const detail = await response.text().catch(() => '');
      throw new Error(detail.trim() || `Failed to fetch channel tools: ${response.statusText}`);
    }
    return response.json();
  }

  // Fetch list of channels
  async fetchChannels(): Promise<Channel[]> {
    const response = await this.hubFetch(`/api/channels`);
    
    if (!response.ok) {
      throw new Error(`Failed to fetch channels: ${response.statusText}`);
    }
    
    return response.json();
  }

  // Fetch command definitions (cached unless forceRefresh is true)
  async fetchCommands(forceRefresh: boolean = false): Promise<CommandDefinition[]> {
    if (!forceRefresh && this.commandsCache) {
      return this.commandsCache;
    }

    const response = await this.hubFetch(`/api/commands`);

    if (!response.ok) {
      throw new Error(`Failed to fetch commands: ${response.statusText}`);
    }

    this.commandsCache = await response.json();
    return this.commandsCache!;
  }

  clearCommandsCache(): void {
    this.commandsCache = null;
  }

  async fetchAssistantState(channel?: string): Promise<AssistantStateResponse> {
    const params = new URLSearchParams();
    if (channel) {
      params.set('channel', channel);
    }
    const query = params.toString();
    const response = await this.hubFetch(`/api/assistant/state${query ? `?${query}` : ''}`);
    if (!response.ok) {
      throw new Error(`Failed to fetch assistant state: ${response.statusText}`);
    }
    return response.json();
  }

  async markAssistantTaskDone(taskID: string): Promise<void> {
    const response = await this.hubFetch(`/api/assistant/task-done`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ task_id: taskID }),
    });
    if (!response.ok) {
      throw new Error(`Failed to mark task done: ${response.statusText}`);
    }
  }

  async dismissAssistantReminder(reminderID: string): Promise<void> {
    const response = await this.hubFetch(`/api/assistant/reminder-dismiss`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ reminder_id: reminderID }),
    });
    if (!response.ok) {
      throw new Error(`Failed to dismiss reminder: ${response.statusText}`);
    }
  }

  async getGoogleMeetNotesAppConfig(): Promise<GoogleMeetNotesAppConfig> {
    const response = await this.hubFetch(`/api/assistant/google/config`);
    if (!response.ok) {
      throw new Error(`Failed to fetch Google OAuth config: ${response.statusText}`);
    }
    return response.json();
  }

  async saveGoogleMeetNotesAppConfig(
    clientId: string,
    clientSecret: string,
    redirectUrl?: string
  ): Promise<GoogleMeetNotesAppConfig> {
    const response = await this.hubFetch(`/api/assistant/google/config`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        client_id: clientId,
        client_secret: clientSecret,
        redirect_url: redirectUrl ?? '',
      }),
    });
    const data = await response.json();
    if (!response.ok) {
      throw new Error(data.error || `Failed to save Google OAuth config: ${response.statusText}`);
    }
    return data;
  }

  async getGoogleMeetNotesStatus(): Promise<GoogleMeetNotesStatus> {
    const response = await this.hubFetch(`/api/assistant/google/status`);
    if (!response.ok) {
      throw new Error(`Failed to fetch Google meet notes status: ${response.statusText}`);
    }
    return response.json();
  }

  async getGoogleMeetNotesAuthURL(): Promise<string> {
    const response = await this.hubFetch(`/api/assistant/google/auth?json=1`);
    const data = await response.json();
    if (!response.ok) {
      throw new Error(data.error || `Failed to get auth URL: ${response.statusText}`);
    }
    return data.url;
  }

  async disconnectGoogleMeetNotes(): Promise<void> {
    const response = await this.hubFetch(`/api/assistant/google/disconnect`, {
      method: 'POST',
    });
    if (!response.ok) {
      const data = await response.json().catch(() => ({}));
      throw new Error(data.error || `Disconnect failed: ${response.statusText}`);
    }
  }

  async syncGoogleMeetNotes(): Promise<number> {
    const response = await this.hubFetch(`/api/assistant/google/sync`, {
      method: 'POST',
    });
    const data = await response.json();
    if (!response.ok) {
      throw new Error(data.error || `Sync failed: ${response.statusText}`);
    }
    return data.ingested ?? 0;
  }

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
    const response = await this.hubFetch(`/api/slack/bindings?slack_channel_id=${encodeURIComponent(slackChannelId)}`,
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

  // Create a new channel
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
      throw new Error(`Failed to create channel: ${response.statusText}`);
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

  /** CLI agent registry keys and whether each binary appears on the server PATH. */
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

  // Delete a channel
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

  // Add agents to a channel
  async addAgentsToChannel(channelName: string, agentIds: string[]): Promise<void> {
    const response = await this.hubFetch(`/api/channels/agents?channel=${encodeURIComponent(channelName)}`,
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

  // Remove an agent from a channel
  async removeAgentFromChannel(channelName: string, agentId: string): Promise<void> {
    const response = await this.hubFetch(`/api/channels/agents?channel=${encodeURIComponent(channelName)}&agent_id=${encodeURIComponent(agentId)}`,
      { method: 'DELETE' }
    );

    if (!response.ok) {
      throw new Error(`Failed to remove agent from channel: ${response.statusText}`);
    }
  }

  // Test server connection
  async testConnection(): Promise<boolean> {
    try {
      const response = await this.hubFetch(`/api/channels`);
      return response.ok;
    } catch (error) {
      return false;
    }
  }

  // Get WebSocket URL for a channel
  getWebSocketURL(channel: string): string {
    const wsURL = this.baseURL.replace('http://', 'ws://').replace('https://', 'wss://');
    return `${wsURL}/ws?channel=${encodeURIComponent(channel)}`;
  }

  // Get WebSocket URL for a thread
  getThreadWebSocketURL(channel: string, threadId: string): string {
    const wsURL = this.baseURL.replace('http://', 'ws://').replace('https://', 'wss://');
    return `${wsURL}/ws?channel=${encodeURIComponent(channel)}&thread=${encodeURIComponent(threadId)}`;
  }

  // Fetch messages from a thread
  async fetchThreadMessages(threadId: string, limit: number = 50): Promise<Message[]> {
    const response = await this.hubFetch(`/api/threads/${encodeURIComponent(threadId)}/messages?limit=${limit}`
    );
    
    if (!response.ok) {
      throw new Error(`Failed to fetch thread messages: ${response.statusText}`);
    }
    
    return response.json();
  }

  // Send a reply to a thread
  async sendThreadReply(
    threadId: string,
    channel: string,
    content: string,
    from: { name: string; type: string },
    metadata?: Record<string, unknown>
  ): Promise<void> {
    const body: Record<string, unknown> = {
      channel,
      content,
      from,
    };
    if (metadata && Object.keys(metadata).length > 0) {
      body.metadata = metadata;
    }

    const response = await this.hubFetch(`/api/threads/${encodeURIComponent(threadId)}/reply`,
      {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(body),
      }
    );

    if (!response.ok) {
      throw new Error(`Failed to send thread reply: ${response.statusText}`);
    }
  }

  // Fetch thread metadata
  async fetchThreadMetadata(threadId: string): Promise<ThreadMetadata> {
    const response = await this.hubFetch(`/api/threads/${encodeURIComponent(threadId)}/metadata`
    );
    
    if (!response.ok) {
      throw new Error(`Failed to fetch thread metadata: ${response.statusText}`);
    }
    
    return response.json();
  }

  // Fetch my agents
  async fetchMyAgents(): Promise<CachedAgentInfo[]> {
    const response = await this.hubFetch(`/api/my-agents`);
    
    if (!response.ok) {
      throw new Error(`Failed to fetch my agents: ${response.statusText}`);
    }
    
    const data = await response.json();
    return data.my_agents || [];
  }

  /** Delete a cached agent entry (repo index, CLI record) without loading it. */
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

  // Fetch removed agents
  async fetchRemovedAgents(): Promise<AgentInfo[]> {
    const response = await this.hubFetch(`/api/removed-agents`);
    
    if (!response.ok) {
      throw new Error(`Failed to fetch removed agents: ${response.statusText}`);
    }
    
    const data = await response.json();
    return data.removed_agents || [];
  }

  // Remove an agent from conversation
  async removeAgent(
    channel: string,
    agentName: string,
    from: { name: string; type: string }
  ): Promise<void> {
    const command = `/remove-agent ${agentName}`;
    await this.sendMessage(channel, command, from, 'question');
  }

  /** Permanently delete an agent (unregister, cleanup repo cache when applicable). */
  async deleteAgent(
    channel: string,
    agentName: string,
    from: { name: string; type: string }
  ): Promise<void> {
    const command = `/delete-agent ${agentName}`;
    await this.sendMessage(channel, command, from, 'question');
  }

  // Recall a removed agent
  async recallAgent(
    channel: string,
    agentName: string,
    from: { name: string; type: string }
  ): Promise<void> {
    const command = `/recall-agent ${agentName}`;
    await this.sendMessage(channel, command, from, 'question');
  }

  // Export an agent to MCP format
  async exportAgent(channel: string, agentName: string): Promise<void> {
    await this.sendMessage(
      channel,
      `/export-agent-mcp ${agentName}`,
      { name: 'User', type: 'user' },
      'chat'
    );
  }

  // Test Anthropic connection
  async testAnthropicConnection(apiKey: string, useAIHub: boolean = true, aiHubEndpoint?: string): Promise<ConnectionTestResult> {
    try {
      const credentials = {
        anthropic_api_key: apiKey,
        use_ai_hub: useAIHub,
        ai_hub_endpoint: aiHubEndpoint,
      };

      const response = await this.hubFetch(`/api/test-anthropic-connection`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(credentials),
      });

      const result = await response.json();
      return {
        success: response.ok,
        message: result.message || (response.ok ? 'Connection successful' : 'Connection failed'),
        error: result.error,
      };
    } catch (error) {
      return {
        success: false,
        message: 'Connection test failed',
        error: error instanceof Error ? error.message : 'Unknown error',
      };
    }
  }

  // Test GitHub connection
  async testGitHubConnection(personalAccessToken: string): Promise<ConnectionTestResult> {
    try {
      const credentials = {
        github_token: personalAccessToken,
      };

      const response = await this.hubFetch(`/api/test-github-connection`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(credentials),
      });

      const result = await response.json();
      return {
        success: response.ok,
        message: result.message || (response.ok ? 'Connection successful' : 'Connection failed'),
        error: result.error,
      };
    } catch (error) {
      return {
        success: false,
        message: 'Connection test failed',
        error: error instanceof Error ? error.message : 'Unknown error',
      };
    }
  }

  // Test Confluence connection
  async testConfluenceConnection(domain: string, email: string, apiToken: string): Promise<ConnectionTestResult> {
    try {
      const credentials = {
        confluence_credentials: {
          domain,
          email,
          api_token: apiToken,
        },
      };

      const response = await this.hubFetch(`/api/test-confluence-connection`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(credentials),
      });

      const result = await response.json();
      return {
        success: response.ok,
        message: result.message || (response.ok ? 'Connection successful' : 'Connection failed'),
        error: result.error,
      };
    } catch (error) {
      return {
        success: false,
        message: 'Connection test failed',
        error: error instanceof Error ? error.message : 'Unknown error',
      };
    }
  }

  // Test Ollama connection
  async testOllamaConnection(endpoint: string, model: string): Promise<ConnectionTestResult> {
    try {
      const credentials = {
        endpoint,
        model,
      };

      const response = await this.hubFetch(`/api/test-ollama-connection`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(credentials),
      });

      const result = await response.json();
      return {
        success: response.ok,
        message: result.message || (response.ok ? 'Connection successful' : 'Connection failed'),
        error: result.error,
      };
    } catch (error) {
      return {
        success: false,
        message: 'Connection test failed',
        error: error instanceof Error ? error.message : 'Unknown error',
      };
    }
  }

  // Switch agent provider
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

  // Switch all agents to same provider
  async switchAllAgentProviders(provider: string, model: string): Promise<void> {
    const response = await this.hubFetch(`/api/agents/switch-all-providers`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ provider, model }),
    });

    if (!response.ok) {
      throw new Error(`Failed to switch all agents: ${response.statusText}`);
    }
  }

  // Get Ollama status
  async fetchOllamaStatus(): Promise<{ running: boolean; endpoint: string; error?: string }> {
    const response = await this.hubFetch(`/api/ollama/status`);
    
    if (!response.ok) {
      throw new Error(`Failed to fetch Ollama status: ${response.statusText}`);
    }
    
    return response.json();
  }

  // Get available Ollama models
  async fetchOllamaModels(endpoint?: string): Promise<string[]> {
    const path = endpoint
      ? `/api/ollama/models?endpoint=${encodeURIComponent(endpoint)}`
      : '/api/ollama/models';
    const response = await this.hubFetch(path);
    
    if (!response.ok) {
      throw new Error(`Failed to fetch Ollama models: ${response.statusText}`);
    }
    
    const result = await response.json();
    return result.models || [];
  }

  // Test LM Studio connection
  async testLMStudioConnection(endpoint: string, model: string): Promise<ConnectionTestResult> {
    try {
      const credentials = {
        endpoint,
        model,
      };

      const response = await this.hubFetch(`/api/test-lmstudio-connection`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(credentials),
      });

      const result = await response.json();
      return {
        success: response.ok,
        message: result.message || (response.ok ? 'Connection successful' : 'Connection failed'),
        error: result.error,
      };
    } catch (error) {
      return {
        success: false,
        message: 'Connection test failed',
        error: error instanceof Error ? error.message : 'Unknown error',
      };
    }
  }

  // Get LM Studio status
  async fetchLMStudioStatus(): Promise<{ running: boolean; endpoint: string; error?: string }> {
    const response = await this.hubFetch(`/api/lmstudio/status`);
    
    if (!response.ok) {
      throw new Error(`Failed to fetch LM Studio status: ${response.statusText}`);
    }
    
    return response.json();
  }

  // Get available LM Studio models
  async fetchLMStudioModels(endpoint?: string): Promise<string[]> {
    const path = endpoint
      ? `/api/lmstudio/models?endpoint=${encodeURIComponent(endpoint)}`
      : '/api/lmstudio/models';
    const response = await this.hubFetch(path);
    
    if (!response.ok) {
      throw new Error(`Failed to fetch LM Studio models: ${response.statusText}`);
    }
    
    const result = await response.json();
    return result.models || [];
  }

  async fetchHfCatalog(): Promise<
    {
      repo_id: string;
      title: string;
      description: string;
      tags: string[];
      modes: string[];
      files?: { filename: string; quant?: string }[];
    }[]
  > {
    const response = await this.hubFetch(`/api/hf/catalog`);
    if (!response.ok) {
      throw new Error(`Failed to fetch HF catalog: ${response.statusText}`);
    }
    return response.json();
  }

  async fetchHfStatus(): Promise<{
    token_configured: boolean;
    router_reachable: boolean;
    cache_dir?: string;
  }> {
    const response = await this.hubFetch(`/api/hf/status`);
    if (!response.ok) {
      throw new Error(`Failed to fetch HF status: ${response.statusText}`);
    }
    return response.json();
  }

  async fetchProviders(): Promise<
    { id: string; type: string; name: string; model?: string; endpoint?: string }[]
  > {
    const response = await this.hubFetch(`/api/providers`);
    if (!response.ok) {
      throw new Error(`Failed to fetch providers: ${response.statusText}`);
    }
    return response.json();
  }

  // Send message with credentials for agent creation
  async sendMessageWithCredentials(
    channel: string,
    content: string,
    from: { name: string; type: string },
    credentials?: Record<string, any>
  ): Promise<SendMessageResponse> {
    return this.sendMessage(channel, content, from, 'question', credentials);
  }

  // Utility function to clear credentials from memory
  static clearCredentials(credentials: Record<string, any>): void {
    for (const key in credentials) {
      if (typeof credentials[key] === 'string') {
        // Overwrite string values with random data to clear from memory
        credentials[key] = 'x'.repeat(credentials[key].length);
      } else if (typeof credentials[key] === 'object' && credentials[key] !== null) {
        // Recursively clear nested objects
        this.clearCredentials(credentials[key]);
      }
    }
  }

  // Workspace API methods
  async fetchWorkspaces(): Promise<any[]> {
    const response = await this.hubFetch(`/api/workspaces`);
    
    if (!response.ok) {
      throw new Error(`Failed to fetch workspaces: ${response.statusText}`);
    }
    
    return response.json();
  }

  async addWorkspace(name: string, path: string): Promise<any> {
    const response = await this.hubFetch(`/api/workspaces`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ name, path }),
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

  // File system API methods
  async fetchFiles(workspaceId: string, path: string = '/'): Promise<any[]> {
    const response = await this.hubFetch(`/api/files?workspace=${encodeURIComponent(workspaceId)}&path=${encodeURIComponent(path)}`
    );
    
    if (!response.ok) {
      throw new Error(`Failed to fetch files: ${response.statusText}`);
    }
    
    return response.json();
  }

  async fetchFileContent(workspaceId: string, path: string): Promise<string> {
    const response = await this.hubFetch(`/api/file-content?workspace=${encodeURIComponent(workspaceId)}&path=${encodeURIComponent(path)}`
    );
    
    if (!response.ok) {
      const body = await response.text().catch(() => '');
      const detail = body.trim();
      if (response.status === 403) {
        throw new Error(
          detail || 'Forbidden: path is outside the workspace. Use a path relative to the workspace root.'
        );
      }
      if (response.status === 404) {
        throw new Error(detail || `Not Found: ${path}`);
      }
      throw new Error(
        detail
          ? `Failed to fetch file content (${response.status}): ${detail}`
          : `Failed to fetch file content: ${response.statusText}`
      );
    }
    
    const data = await response.json();
    return data.content;
  }

  /** Load a workspace image as a data URL (for editor preview in browser dev). */
  async fetchScanSummaryWellImage(
    workspaceId: string,
    summaryDir: string,
    well: string
  ): Promise<string> {
    const params = new URLSearchParams({
      workspace: workspaceId,
      dir: summaryDir,
      well,
    });
    const response = await this.hubFetch(`/api/scan-summary/well-image?${params.toString()}`
    );
    if (!response.ok) {
      const text = await response.text();
      throw new Error(text || `Failed to load well image: ${response.statusText}`);
    }
    const data = (await response.json()) as { mime?: string; content_base64?: string };
    const b64 = data.content_base64 ?? '';
    if (!b64) {
      throw new Error('Empty well image payload from hub');
    }
    const mime = data.mime || 'image/png';
    return `data:${mime};base64,${b64}`;
  }

  async fetchWorkspaceImageDataUrl(workspaceId: string, path: string): Promise<string> {
    const response = await this.hubFetch(`/api/file-content?workspace=${encodeURIComponent(workspaceId)}&path=${encodeURIComponent(path)}&binary=1`
    );
    if (!response.ok) {
      throw new Error(`Failed to load image: ${response.statusText}`);
    }
    const data = (await response.json()) as { mime?: string; content_base64?: string };
    const b64 = data.content_base64 ?? '';
    if (!b64) {
      throw new Error('Empty image payload from hub');
    }
    const mime = data.mime || 'application/octet-stream';
    return `data:${mime};base64,${b64}`;
  }

  async saveFileContent(workspaceId: string, path: string, content: string): Promise<void> {
    const response = await this.hubFetch(`/api/file-content`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        workspace_id: workspaceId,
        path,
        content,
      }),
    });

    if (!response.ok) {
      throw new Error(`Failed to save file content: ${response.statusText}`);
    }
  }

  async createFile(workspaceId: string, path: string, content: string = ''): Promise<void> {
    const response = await this.hubFetch(`/api/file-create`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        workspace_id: workspaceId,
        path,
        content,
      }),
    });

    if (!response.ok) {
      throw new Error(`Failed to create file: ${response.statusText}`);
    }
  }

  async renameFile(workspaceId: string, oldPath: string, newPath: string): Promise<void> {
    const response = await this.hubFetch(`/api/file-rename`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        workspace_id: workspaceId,
        old_path: oldPath,
        new_path: newPath,
      }),
    });

    if (!response.ok) {
      throw new Error(`Failed to rename file: ${response.statusText}`);
    }
  }

  async deleteFile(workspaceId: string, path: string): Promise<void> {
    const response = await this.hubFetch(`/api/file-delete?workspace=${encodeURIComponent(workspaceId)}&path=${encodeURIComponent(path)}`,
      {
        method: 'DELETE',
      }
    );

    if (!response.ok) {
      throw new Error(`Failed to delete file: ${response.statusText}`);
    }
  }

  // Git operations API methods (stubs for now)
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

  async searchWorkspaceFiles(
    workspaceId: string,
    query: string,
    limit = 50
  ): Promise<string[]> {
    const params = new URLSearchParams({
      workspace: workspaceId,
      q: query,
      limit: String(limit),
    });
    const response = await this.hubFetch(`/api/workspaces/files/search?${params}`, {
      method: 'GET',
    });
    if (!response.ok) {
      throw new Error(`Failed to search files: ${response.statusText}`);
    }
    const data = (await response.json()) as { paths?: string[] };
    return data.paths ?? [];
  }

  async searchWorkspaceSymbols(
    workspaceId: string,
    query: string,
    limit = 50
  ): Promise<
    Array<{ name: string; path: string; line: number; kind: string; language: string }>
  > {
    const params = new URLSearchParams({
      workspace: workspaceId,
      q: query,
      limit: String(limit),
    });
    const response = await this.hubFetch(`/api/workspaces/symbols/search?${params}`, {
      method: 'GET',
    });
    if (!response.ok) {
      throw new Error(`Failed to search symbols: ${response.statusText}`);
    }
    const data = (await response.json()) as { symbols?: Array<{
      name: string;
      path: string;
      line: number;
      kind: string;
      language: string;
    }> };
    return data.symbols ?? [];
  }

  async devFastEdit(params: {
    workspaceId: string;
    path?: string;
    instruction: string;
    selection?: string;
    agentType?: string;
    metadata?: Record<string, unknown>;
  }): Promise<{
    response: string;
    proposed: boolean;
    change_id?: string;
    agent?: string;
    agent_type?: string;
  }> {
    const response = await this.hubFetch('/api/dev/fast-edit', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        workspace_id: params.workspaceId,
        path: params.path,
        instruction: params.instruction,
        selection: params.selection,
        agent_type: params.agentType,
        metadata: params.metadata,
      }),
    });
    if (!response.ok) {
      const text = await response.text();
      throw new Error(text || `Fast edit failed: ${response.statusText}`);
    }
    return response.json();
  }

  async getGoLSPDiagnostics(
    workspaceId: string
  ): Promise<
    Array<{ path: string; line: number; column: number; message: string; severity: string }>
  > {
    const params = new URLSearchParams({ workspace: workspaceId });
    const response = await this.hubFetch(`/api/lsp/go/diagnostics?${params}`, {
      method: 'GET',
    });
    if (!response.ok) {
      return [];
    }
    const data = (await response.json()) as {
      diagnostics?: Array<{
        path: string;
        line: number;
        column: number;
        message: string;
        severity: string;
      }>;
    };
    return data.diagnostics ?? [];
  }

  async getLSPDiagnostics(
    lang: 'rust' | 'python',
    workspaceId: string
  ): Promise<
    Array<{ path: string; line: number; column: number; message: string; severity: string }>
  > {
    const params = new URLSearchParams({ workspace: workspaceId });
    const response = await this.hubFetch(`/api/lsp/${lang}/diagnostics?${params}`, {
      method: 'GET',
    });
    if (!response.ok) return [];
    const data = (await response.json()) as {
      diagnostics?: Array<{
        path: string;
        line: number;
        column: number;
        message: string;
        severity: string;
      }>;
    };
    return data.diagnostics ?? [];
  }

  async devComplete(params: {
    prefix: string;
    suffix?: string;
    language?: string;
    path?: string;
    model?: string;
  }): Promise<{ completion: string }> {
    const response = await this.hubFetch('/api/dev/complete', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        prefix: params.prefix,
        suffix: params.suffix ?? '',
        language: params.language,
        path: params.path,
        model: params.model,
      }),
    });
    if (!response.ok) {
      return { completion: '' };
    }
    return response.json();
  }

  async devAgentTurn(params: {
    workspaceId: string;
    instruction: string;
    sessionId?: string;
    mode?: 'ask' | 'agent';
    path?: string;
    selection?: string;
    agentType?: string;
    metadata?: Record<string, unknown>;
    attachments?: Array<Record<string, unknown>>;
  }): Promise<{
    response: string;
    proposed: boolean;
    session_id: string;
    channel: string;
    change_ids?: string[];
    agent?: string;
    agent_type?: string;
  }> {
    const response = await this.hubFetch('/api/dev/agent-turn', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        workspace_id: params.workspaceId,
        instruction: params.instruction,
        session_id: params.sessionId,
        mode: params.mode ?? 'agent',
        path: params.path,
        selection: params.selection,
        agent_type: params.agentType,
        metadata: params.metadata,
        attachments: params.attachments,
      }),
    });
    if (!response.ok) {
      const text = await response.text();
      throw new Error(text || `Agent turn failed: ${response.statusText}`);
    }
    return response.json();
  }

  async repoSemanticSearch(params: {
    repoPath: string;
    query: string;
    limit?: number;
  }): Promise<{ chunks: Array<{ path: string; content: string }> }> {
    const response = await this.hubFetch('/api/repo/search/semantic', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        repo_path: params.repoPath,
        query: params.query,
        limit: params.limit ?? 8,
      }),
    });
    if (!response.ok) {
      throw new Error(`Semantic search failed: ${response.statusText}`);
    }
    return response.json();
  }

  async repoIndexStatus(repoPath: string): Promise<{
    ready: boolean;
    building: boolean;
    chunk_count: number;
    embedding_model?: string;
  }> {
    const params = new URLSearchParams({ repo_path: repoPath });
    const response = await this.hubFetch(`/api/repo/index/status?${params}`);
    if (!response.ok) {
      throw new Error(`Index status failed: ${response.statusText}`);
    }
    return response.json();
  }

  // Tool approval API methods

  async approveToolCall(approvalId: string): Promise<void> {
    const response = await this.hubFetch(`/api/tool-approvals/approve/${approvalId}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
    });

    if (!response.ok) {
      throw new Error(`Failed to approve tool call: ${response.statusText}`);
    }
  }

  async rejectToolCall(approvalId: string, reason: string = 'User rejected'): Promise<void> {
    const response = await this.hubFetch(`/api/tool-approvals/reject/${approvalId}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ reason }),
    });

    if (!response.ok) {
      throw new Error(`Failed to reject tool call: ${response.statusText}`);
    }
  }

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

  // File change API methods

  // Create a file change proposal directly from an agent message
  async proposeFileChangeFromMessage(params: {
    channel: string;
    messageId: string;
    workspaceId: string;
    targetPath?: string;
    userId?: string;
  }): Promise<FileChange> {
    const response = await this.hubFetch(`/api/file-changes/propose-from-message`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        channel: params.channel,
        message_id: params.messageId,
        workspace_id: params.workspaceId,
        target_path: params.targetPath || '',
        user_id: params.userId || 'default',
      }),
    });

    if (!response.ok) {
      const errText = await response.text();
      throw new Error(errText || `Failed to create proposal from message: ${response.statusText}`);
    }

    return response.json();
  }

  // List pending file changes
  async listPendingFileChanges(userId: string = 'default'): Promise<FileChange[]> {
    const response = await this.hubFetch(`/api/file-changes?user_id=${encodeURIComponent(userId)}`
    );
    
    if (!response.ok) {
      throw new Error(`Failed to fetch file changes: ${response.statusText}`);
    }
    
    return response.json();
  }

  // Approve a file change
  async approveFileChange(changeId: string, userId: string = 'default'): Promise<FileChange> {
    const response = await this.hubFetch(`/api/file-changes/approve/${changeId}?user_id=${encodeURIComponent(userId)}`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
    });

    if (!response.ok) {
      throw new Error(`Failed to approve file change: ${response.statusText}`);
    }

    return response.json();
  }

  // Reject a file change
  async rejectFileChange(changeId: string, reason: string = 'No reason provided', userId: string = 'default'): Promise<FileChange> {
    const response = await this.hubFetch(`/api/file-changes/reject/${changeId}`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        user_id: userId,
        reason: reason,
      }),
    });

    if (!response.ok) {
      throw new Error(`Failed to reject file change: ${response.statusText}`);
    }

    return response.json();
  }

  // Get file change diff
  async getFileDiff(changeId: string): Promise<FileChangeDiff> {
    const response = await this.hubFetch(`/api/file-changes/${changeId}`);
    
    if (!response.ok) {
      throw new Error(`Failed to get file diff: ${response.statusText}`);
    }
    
    return response.json();
  }

  async fetchPacks(): Promise<PacksAPIResponse> {
    const response = await this.hubFetch(`/api/packs`);
    if (!response.ok) {
      throw new Error(`Failed to fetch packs: ${response.statusText}`);
    }
    return response.json();
  }

  async fetchPackCatalog(): Promise<PackCatalogEntry[]> {
    const response = await this.hubFetch(`/api/packs/catalog`);
    if (!response.ok) {
      throw new Error(`Failed to fetch pack catalog: ${response.statusText}`);
    }
    const data = await response.json();
    return (data.packs as PackCatalogEntry[]) ?? [];
  }

  async installPack(packId: string): Promise<PacksAPIResponse> {
    const response = await this.hubFetch(`/api/packs/${encodeURIComponent(packId)}/install`, {
      method: 'POST',
    });
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return this.parsePacksMutationResponse(await response.json());
  }

  async installPackLoRAs(packId: string): Promise<InstallPackLoRAsResponse> {
    const response = await this.hubFetch(`/api/packs/${encodeURIComponent(packId)}/install-loras`, {
      method: 'POST',
    });
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return response.json();
  }

  async uninstallPack(packId: string): Promise<PacksAPIResponse> {
    const response = await this.hubFetch(`/api/packs/${encodeURIComponent(packId)}`, {
      method: 'DELETE',
    });
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return this.parsePacksMutationResponse(await response.json());
  }

  async setPackEnabled(packId: string, enabled: boolean): Promise<PacksAPIResponse> {
    const response = await this.hubFetch(`/api/packs/${encodeURIComponent(packId)}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ enabled }),
    });
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return this.parsePacksMutationResponse(await response.json());
  }

  private parsePacksMutationResponse(data: Record<string, unknown>): PacksAPIResponse {
    return {
      packs: (data.packs as PackStatus[]) ?? [],
      layout_owner: data.layout_owner as string | undefined,
      layout_profile: data.layout_profile as string | undefined,
      capabilities: (data.capabilities as string[]) ?? [],
    };
  }

  async fetchExpertPresets(): Promise<ExpertPresetOption[]> {
    const response = await this.hubFetch(`/api/expert-presets`);
    if (!response.ok) {
      throw new Error(`Failed to fetch expert presets: ${response.statusText}`);
    }
    return response.json();
  }

  async restartConfiguredAgents(): Promise<void> {
    const response = await this.hubFetch(`/api/agents/restart`, { method: 'POST' });
    if (!response.ok) {
      throw new Error(`Failed to restart agents: ${response.statusText}`);
    }
  }

  async fetchLoraExpertContext(agentId: string): Promise<LoraExpertContext> {
    const response = await this.hubFetch(
      `/api/lora/train/expert-context?agent_id=${encodeURIComponent(agentId)}`,
    );
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return response.json();
  }

  async previewLoraTrain(params: {
    source: string;
    source_id: string;
    thread_id?: string;
    agent_name?: string;
    agent_id?: string;
    include_learnings?: boolean;
  }): Promise<number> {
    const q = new URLSearchParams({
      source: params.source,
      source_id: params.source_id,
    });
    if (params.thread_id) q.set('thread_id', params.thread_id);
    if (params.agent_name) q.set('agent_name', params.agent_name);
    if (params.agent_id) q.set('agent_id', params.agent_id);
    if (params.include_learnings) q.set('include_learnings', '1');
    const response = await this.hubFetch(`/api/lora/train/preview?${q.toString()}`);
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    const data = await response.json();
    return Number(data.row_count ?? 0);
  }

  async startLoraTrain(body: LoraTrainStartRequest): Promise<LoraTrainJob> {
    const response = await this.hubFetch(`/api/lora/train`, {
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

  async fetchLoraTrainJob(jobId: string): Promise<LoraTrainJob> {
    const response = await this.hubFetch(`/api/lora/train/${encodeURIComponent(jobId)}`);
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return response.json();
  }

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

