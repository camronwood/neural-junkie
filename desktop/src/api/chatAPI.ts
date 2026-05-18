import type { Message, AgentInfo, Channel, ThreadMetadata, CachedAgentInfo, ConnectionTestResult, FileChange, FileChangeDiff, CommandDefinition, AssistantStateResponse, Collaboration } from '../types/protocol';
import { getHubBaseURL, normalizeHubBaseURL } from '../config/hubUrl';

/** Successful POST /api/send response; optional fields when a slash command requests a channel switch. */
export interface SendMessageResponse {
  status?: string;
  collaboration_channel?: string;
  collaboration_id?: string;
}

export class ChatAPI {
  private baseURL: string;
  private commandsCache: CommandDefinition[] | null = null;

  constructor(serverAddr: string = getHubBaseURL()) {
    this.baseURL = normalizeHubBaseURL(serverAddr);
  }

  // Fetch existing messages for a channel
  async fetchMessages(channel: string, limit: number = 50): Promise<Message[]> {
    const response = await fetch(
      `${this.baseURL}/api/messages?channel=${encodeURIComponent(channel)}&limit=${limit}`
    );
    
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
    const response = await fetch(`${this.baseURL}/api/collaborations${query ? `?${query}` : ''}`);

    if (!response.ok) {
      throw new Error(`Failed to fetch collaborations: ${response.statusText}`);
    }

    return response.json();
  }

  /** Read user-granted files/directories under ~/.neural-junkie for agent context. */
  async readHubDataAccess(
    targets: Array<{ kind: 'file' | 'directory'; relative_path: string }>
  ): Promise<{ root: string; entries: unknown[] }> {
    const response = await fetch(`${this.baseURL}/api/hub-data/read`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
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
  async acknowledgeCollaborationWorkspace(collaborationId: string): Promise<void> {
    const response = await fetch(`${this.baseURL}/api/collaboration-workspace-ack`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ collaboration_id: collaborationId }),
    });
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t || response.statusText);
    }
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
      body.metadata = credentials;
    }

    const response = await fetch(`${this.baseURL}/api/send`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
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
  async fetchAgents(): Promise<AgentInfo[]> {
    const response = await fetch(`${this.baseURL}/api/agents`);
    
    if (!response.ok) {
      throw new Error(`Failed to fetch agents: ${response.statusText}`);
    }
    
    const agents = await response.json();
    
    
    return agents;
  }

  // Fetch list of channels
  async fetchChannels(): Promise<Channel[]> {
    const response = await fetch(`${this.baseURL}/api/channels`);
    
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

    const response = await fetch(`${this.baseURL}/api/commands`);

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
    const response = await fetch(`${this.baseURL}/api/assistant/state${query ? `?${query}` : ''}`);
    if (!response.ok) {
      throw new Error(`Failed to fetch assistant state: ${response.statusText}`);
    }
    return response.json();
  }

  async markAssistantTaskDone(taskID: string): Promise<void> {
    const response = await fetch(`${this.baseURL}/api/assistant/task-done`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ task_id: taskID }),
    });
    if (!response.ok) {
      throw new Error(`Failed to mark task done: ${response.statusText}`);
    }
  }

  async dismissAssistantReminder(reminderID: string): Promise<void> {
    const response = await fetch(`${this.baseURL}/api/assistant/reminder-dismiss`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ reminder_id: reminderID }),
    });
    if (!response.ok) {
      throw new Error(`Failed to dismiss reminder: ${response.statusText}`);
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
    const response = await fetch(`${this.baseURL}/api/channels/create`, {
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

    const response = await fetch(`${this.baseURL}/api/channels/create-dm-agent`, {
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
    const response = await fetch(`${this.baseURL}/api/cli-agent-types`);
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
  async deleteChannel(name: string): Promise<void> {
    const response = await fetch(`${this.baseURL}/api/channels/delete`, {
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
    const response = await fetch(
      `${this.baseURL}/api/channels/agents?channel=${encodeURIComponent(channelName)}`,
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
    const response = await fetch(
      `${this.baseURL}/api/channels/agents?channel=${encodeURIComponent(channelName)}&agent_id=${encodeURIComponent(agentId)}`,
      { method: 'DELETE' }
    );

    if (!response.ok) {
      throw new Error(`Failed to remove agent from channel: ${response.statusText}`);
    }
  }

  // Test server connection
  async testConnection(): Promise<boolean> {
    try {
      const response = await fetch(`${this.baseURL}/api/channels`);
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
    const response = await fetch(
      `${this.baseURL}/api/threads/${encodeURIComponent(threadId)}/messages?limit=${limit}`
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

    const response = await fetch(
      `${this.baseURL}/api/threads/${encodeURIComponent(threadId)}/reply`,
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
    const response = await fetch(
      `${this.baseURL}/api/threads/${encodeURIComponent(threadId)}/metadata`
    );
    
    if (!response.ok) {
      throw new Error(`Failed to fetch thread metadata: ${response.statusText}`);
    }
    
    return response.json();
  }

  // Fetch my agents
  async fetchMyAgents(): Promise<CachedAgentInfo[]> {
    const response = await fetch(`${this.baseURL}/api/my-agents`);
    
    if (!response.ok) {
      throw new Error(`Failed to fetch my agents: ${response.statusText}`);
    }
    
    const data = await response.json();
    return data.my_agents || [];
  }

  // Fetch removed agents
  async fetchRemovedAgents(): Promise<AgentInfo[]> {
    const response = await fetch(`${this.baseURL}/api/removed-agents`);
    
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

      const response = await fetch(`${this.baseURL}/api/test-anthropic-connection`, {
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

      const response = await fetch(`${this.baseURL}/api/test-github-connection`, {
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

      const response = await fetch(`${this.baseURL}/api/test-confluence-connection`, {
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

      const response = await fetch(`${this.baseURL}/api/test-ollama-connection`, {
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
    const response = await fetch(`${this.baseURL}/api/agents/${agentId}/provider`, {
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
    const response = await fetch(`${this.baseURL}/api/agents/switch-all-providers`, {
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
    const response = await fetch(`${this.baseURL}/api/ollama/status`);
    
    if (!response.ok) {
      throw new Error(`Failed to fetch Ollama status: ${response.statusText}`);
    }
    
    return response.json();
  }

  // Get available Ollama models
  async fetchOllamaModels(endpoint?: string): Promise<string[]> {
    const url = endpoint ? `${this.baseURL}/api/ollama/models?endpoint=${encodeURIComponent(endpoint)}` : `${this.baseURL}/api/ollama/models`;
    const response = await fetch(url);
    
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

      const response = await fetch(`${this.baseURL}/api/test-lmstudio-connection`, {
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
    const response = await fetch(`${this.baseURL}/api/lmstudio/status`);
    
    if (!response.ok) {
      throw new Error(`Failed to fetch LM Studio status: ${response.statusText}`);
    }
    
    return response.json();
  }

  // Get available LM Studio models
  async fetchLMStudioModels(endpoint?: string): Promise<string[]> {
    const url = endpoint ? `${this.baseURL}/api/lmstudio/models?endpoint=${encodeURIComponent(endpoint)}` : `${this.baseURL}/api/lmstudio/models`;
    const response = await fetch(url);
    
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
    const response = await fetch(`${this.baseURL}/api/hf/catalog`);
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
    const response = await fetch(`${this.baseURL}/api/hf/status`);
    if (!response.ok) {
      throw new Error(`Failed to fetch HF status: ${response.statusText}`);
    }
    return response.json();
  }

  async fetchProviders(): Promise<
    { id: string; type: string; name: string; model?: string; endpoint?: string }[]
  > {
    const response = await fetch(`${this.baseURL}/api/providers`);
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
    const response = await fetch(`${this.baseURL}/api/workspaces`);
    
    if (!response.ok) {
      throw new Error(`Failed to fetch workspaces: ${response.statusText}`);
    }
    
    return response.json();
  }

  async addWorkspace(name: string, path: string): Promise<any> {
    const response = await fetch(`${this.baseURL}/api/workspaces`, {
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
    const response = await fetch(`${this.baseURL}/api/workspaces?id=${encodeURIComponent(workspaceId)}`, {
      method: 'DELETE',
    });

    if (!response.ok) {
      throw new Error(`Failed to remove workspace: ${response.statusText}`);
    }
  }

  // File system API methods
  async fetchFiles(workspaceId: string, path: string = '/'): Promise<any[]> {
    const response = await fetch(
      `${this.baseURL}/api/files?workspace=${encodeURIComponent(workspaceId)}&path=${encodeURIComponent(path)}`
    );
    
    if (!response.ok) {
      throw new Error(`Failed to fetch files: ${response.statusText}`);
    }
    
    return response.json();
  }

  async fetchFileContent(workspaceId: string, path: string): Promise<string> {
    const response = await fetch(
      `${this.baseURL}/api/file-content?workspace=${encodeURIComponent(workspaceId)}&path=${encodeURIComponent(path)}`
    );
    
    if (!response.ok) {
      throw new Error(`Failed to fetch file content: ${response.statusText}`);
    }
    
    const data = await response.json();
    return data.content;
  }

  /** Load a workspace image as a data URL (for editor preview in browser dev). */
  async fetchWorkspaceImageDataUrl(workspaceId: string, path: string): Promise<string> {
    const response = await fetch(
      `${this.baseURL}/api/file-content?workspace=${encodeURIComponent(workspaceId)}&path=${encodeURIComponent(path)}&binary=1`
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
    const response = await fetch(`${this.baseURL}/api/file-content`, {
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
    const response = await fetch(`${this.baseURL}/api/file-create`, {
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
    const response = await fetch(`${this.baseURL}/api/file-rename`, {
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
    const response = await fetch(
      `${this.baseURL}/api/file-delete?workspace=${encodeURIComponent(workspaceId)}&path=${encodeURIComponent(path)}`,
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
    const response = await fetch(`${this.baseURL}/api/git-status?workspace=${encodeURIComponent(workspaceId)}`, {
      method: 'POST',
    });

    if (!response.ok) {
      throw new Error(`Failed to get git status: ${response.statusText}`);
    }
    
    return response.json();
  }

  async getGitDiff(workspaceId: string, path: string): Promise<string> {
    const response = await fetch(
      `${this.baseURL}/api/git-diff?workspace=${encodeURIComponent(workspaceId)}&path=${encodeURIComponent(path)}`,
      {
        method: 'POST',
      }
    );

    if (!response.ok) {
      throw new Error(`Failed to get git diff: ${response.statusText}`);
    }
    
    const data = await response.json();
    return data.diff;
  }

  async commitChanges(workspaceId: string, message: string): Promise<void> {
    const response = await fetch(`${this.baseURL}/api/git-commit`, {
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
    const response = await fetch(`${this.baseURL}/api/git-push`, {
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
    const response = await fetch(`${this.baseURL}/api/git-pull`, {
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

  // Tool approval API methods

  async approveToolCall(approvalId: string): Promise<void> {
    const response = await fetch(`${this.baseURL}/api/tool-approvals/approve/${approvalId}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
    });

    if (!response.ok) {
      throw new Error(`Failed to approve tool call: ${response.statusText}`);
    }
  }

  async rejectToolCall(approvalId: string, reason: string = 'User rejected'): Promise<void> {
    const response = await fetch(`${this.baseURL}/api/tool-approvals/reject/${approvalId}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ reason }),
    });

    if (!response.ok) {
      throw new Error(`Failed to reject tool call: ${response.statusText}`);
    }
  }

  async setAgentApprovalMode(agentId: string, mode: 'interactive' | 'auto_edit' | 'yolo'): Promise<void> {
    const response = await fetch(`${this.baseURL}/api/agents/${agentId}/approval-mode`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ mode }),
    });

    if (!response.ok) {
      throw new Error(`Failed to set approval mode: ${response.statusText}`);
    }
  }

  async setAgentCustomRulesMarkdown(agentId: string, markdown: string): Promise<void> {
    const response = await fetch(`${this.baseURL}/api/agents/${encodeURIComponent(agentId)}/rules`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ markdown }),
    });

    if (!response.ok) {
      throw new Error(`Failed to save agent rules: ${response.statusText}`);
    }
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
    const response = await fetch(`${this.baseURL}/api/file-changes/propose-from-message`, {
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
    const response = await fetch(
      `${this.baseURL}/api/file-changes?user_id=${encodeURIComponent(userId)}`
    );
    
    if (!response.ok) {
      throw new Error(`Failed to fetch file changes: ${response.statusText}`);
    }
    
    return response.json();
  }

  // Approve a file change
  async approveFileChange(changeId: string, userId: string = 'default'): Promise<FileChange> {
    const response = await fetch(`${this.baseURL}/api/file-changes/approve/${changeId}?user_id=${encodeURIComponent(userId)}`, {
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
    const response = await fetch(`${this.baseURL}/api/file-changes/reject/${changeId}`, {
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
    const response = await fetch(`${this.baseURL}/api/file-changes/${changeId}`);
    
    if (!response.ok) {
      throw new Error(`Failed to get file diff: ${response.statusText}`);
    }
    
    return response.json();
  }
}

