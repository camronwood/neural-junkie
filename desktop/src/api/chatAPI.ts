import type { Message, AgentInfo, Channel, ThreadMetadata, CachedAgentInfo, ConnectionTestResult, FileChange, FileChangeDiff, GitChangeProposal, CommandDefinition, AssistantStateResponse, GoogleMeetNotesStatus, GoogleMeetNotesAppConfig, WebSearchConfigResponse, SlackConfigResponse, SlackConnectionResponse, SlackStatus, SlackBinding, SlackChannelInfo, SlackPolicy, SlackInboxConfig, SlackDiagnoseResult, SlackSmokeResult, Collaboration, CollaborationTask, AssignSuggestion, ExecutionPolicy, GraphLayout, RunbookDefinition, RunbookDefinitionSummary, RunbookRunRecord, RunbookDefinitionBundle, RunbookRunProvenance, ConnectorProfile, StreamManagerStatus, StreamSubscription, StreamDispatchResult, AgentToolCapabilities, ChannelToolsResponse, CapabilityPolicyResponse, CapabilityPolicyUpdate, ResolvedCapability, StoredArtifact, StoredArtifactRevision } from '../types/protocol';
export type { ResolvedCapability } from '../types/protocol';
import {
  getHubBaseURL,
  hubAuthHeaders,
  hubSessionHeaders,
  normalizeHubBaseURL,
  setHubSessionToken,
} from '../config/hubUrl';
import { buildChannelWebSocketURL, buildThreadWebSocketURL } from './chatAPI/wsUrl';
import { PacksApi } from './domains/packsApi';
import { ChannelsApi } from './domains/channelsApi';
import { MessagesApi } from './domains/messagesApi';
import { CollabApi } from './domains/collabApi';
import { AgentsApi } from './domains/agentsApi';
import { ArtifactsApi } from './domains/artifactsApi';
import { RunbooksApi } from './domains/runbooksApi';
import { RoomsApi } from './domains/roomsApi';
import { ConnectorsApi } from './domains/connectorsApi';
import { StreamsApi } from './domains/streamsApi';
import { GitChangesApi } from './domains/gitChangesApi';

/** Successful POST /api/send response; optional fields when a slash command requests a channel switch. */
export interface SendMessageResponse {
  status?: string;
  collaboration_channel?: string;
  collaboration_id?: string;
  /** Set when /create-expert succeeds; client should open this DM. */
  dm_channel?: string;
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
  custom?: boolean;
  requires_packs?: string[];
  dev_linked?: boolean;
  dev_source_path?: string;
}

export interface PackManifestSummary {
  id: string;
  version?: string;
  title: string;
  description?: string;
  publisher?: string;
  pack_kind?: string;
  layout_profile?: string;
  capabilities?: string[];
  requires_packs?: string[];
  settings_overlay?: Record<string, string>;
  agents?: Array<{ type: string; name?: string; implementation?: string; ollama_model?: string }>;
  mcp_agents?: string[];
}

export interface PackValidationReport {
  valid: boolean;
  errors?: string[];
  warnings?: string[];
  manifest?: PackManifestSummary;
  assets: {
    workspace_guide_found: boolean;
    workspace_guide_path?: string;
    workspace_guide_preview?: string;
    runbooks_count: number;
    runbook_paths?: string[];
  };
  resolved_overlay?: Record<string, string>;
  requires_packs?: Array<{ id: string; installed: boolean; enabled: boolean }>;
  preview?: {
    agents?: Array<{ type: string; name?: string }>;
    effective_capabilities?: string[];
  };
}

export interface CustomerPackContext {
  id: string;
  title: string;
  publisher?: string;
  version?: string;
  requires_packs?: string[];
  workspace_guide?: string;
  settings_overlay?: Record<string, string>;
}

export interface CustomerPackContextResponse {
  packs: CustomerPackContext[];
}

export interface PackCatalogEntry {
  id: string;
  version: string;
  installed_version?: string;
  update_available?: boolean;
  title: string;
  description: string;
  icon_key?: string;
  publisher?: string;
  builtin?: boolean;
  custom?: boolean;
  requires_packs?: string[];
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

export interface ACEStepPaths {
  music_root: string;
  venv: string;
  project: string;
  checkpoint: string;
  setup_script?: string;
}

export interface ACEStepStatus {
  ready: boolean;
  demo_mode: boolean;
  installing: boolean;
  python_ok: boolean;
  venv_ready: boolean;
  project_ready: boolean;
  checkpoint_ready: boolean;
  model_variant?: string;
  python_version?: string;
  last_error?: string;
  install_progress?: { phase: string; detail: string; updated_at?: string };
  paths: ACEStepPaths;
}

export interface InstallACEStepResponse {
  status: string;
  pack_id: string;
  acestep: ACEStepStatus;
}

export interface ArenaSidecarPaths {
  venv: string;
  python: string;
  requirements?: string;
}

export interface ArenaSidecarStatus {
  chess_available: boolean;
  venv_ready: boolean;
  installing: boolean;
  python_ok: boolean;
  python_version?: string;
  last_error?: string;
  paths: ArenaSidecarPaths;
}

export interface InstallArenaSidecarResponse {
  status: string;
  pack_id: string;
  sidecar: ArenaSidecarStatus;
}

export interface AIInterviewDayStatus {
  status?: string;
  concept?: boolean;
  drill?: boolean;
  completed_at?: string;
}

export interface AIInterviewProgressResponse {
  progress: {
    version?: number;
    started_at?: string;
    current_day: number;
    phase: number;
    days?: Record<string, AIInterviewDayStatus>;
    gates?: Record<string, { status?: string; passed_at?: string | null }>;
    certification?: { status?: string; badge_path?: string | null; issued_at?: string | null };
    streak_days?: number;
    last_active_at?: string | null;
  };
  today: {
    day: number;
    phase?: number;
    title?: string;
    kind?: string;
    summary?: string;
    has_drill?: boolean;
    day_status?: AIInterviewDayStatus;
    complete?: boolean;
  };
  stats?: {
    completed_days?: number;
    total_days?: number;
    phase?: number;
    streak_days?: number;
  };
}

export interface ImageGenStatus {
  ready: boolean;
  provider: string;
  model: string;
  endpoint?: string;
  disabled: boolean;
  ollama_running: boolean;
  model_pulled: boolean;
  openai_key_set: boolean;
  pull_command?: string;
}

export interface PackUpdateInfo {
  id: string;
  title: string;
  installed_version: string;
  latest_version: string;
  enabled: boolean;
}

export interface PackUpdatesResponse {
  updates: PackUpdateInfo[];
  count: number;
}

export interface LoraTrainingBase {
  ollama_tag: string;
  hf_model: string;
  label: string;
  description: string;
  code_focused: boolean;
  recommended?: boolean;
  size_hint?: string;
}

export interface LoraExpertContext {
  agent_id: string;
  agent_name: string;
  agent_type: string;
  source: 'repo' | 'channel' | 'collaboration';
  source_id?: string;
  suggested_base_ollama_tag: string;
  suggested_ollama_tag?: string;
  supported_bases?: LoraTrainingBase[];
  preview_rows: number;
  min_rows: number;
  ready: boolean;
  refresh_suggested?: boolean;
  active_adapter_version?: number;
  prior_adapter_id?: string;
  chat_rows?: number;
  learning_rows?: number;
  delta_rows?: number;
  turns?: number;
  suggest_training?: boolean;
  include_learnings_default?: boolean;
  eval_min_score?: number;
  require_eval_to_assign?: boolean;
}

export interface LoraTrainJob {
  id: string;
  status: string;
  source: string;
  source_id: string;
  base_ollama_tag: string;
  ollama_tag: string;
  row_count?: number;
  queue_position?: number;
  adapter_id?: string;
  eval_score?: number;
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

/** Share Agent bundle: extended MCP export with custom rules, learnings, and LoRA metadata. */
export interface AgentShareBundle {
  version: string;
  agent: {
    name: string;
    type: string;
    expertise?: string[];
    description?: string;
    createdAt?: string;
    repository?: string;
  };
  resources: Array<{ uri: string; name: string; mimeType: string; content: string; size?: number }>;
  prompts: Array<{ name: string; description: string; prompt: string }>;
  systemPrompt: string;
  exportedAt?: string;
  lora?: { composed_tag?: string; base_ollama_tag?: string; hf_repo_id?: string; training_manifest?: unknown };
  custom_rules_markdown?: string;
  learnings?: Array<{ content: string; category?: string; scope?: string; agent_name?: string; agent_type?: string }>;
  hydrated_from_resources?: boolean;
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
  refresh_suggested?: boolean;
  suggest_training?: boolean;
  active_adapter_version?: number;
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
  incremental?: boolean;
  prior_adapter_id?: string;
  row_ids?: string[];
  extra_rows?: Array<{
    row_id?: string;
    instruction: string;
    input?: string;
    output: string;
    source_kind?: string;
    source_ref?: string;
  }>;
  approved_tasks_only?: boolean;
  base_ollama_tag: string;
  ollama_tag: string;
  hyperparams?: { rank?: number; epochs?: number; learning_rate?: number; max_seq_len?: number };
}

export interface LoraTrainDatasetRow {
  row_id?: string;
  instruction: string;
  input?: string;
  output: string;
  source_kind?: string;
  source_ref?: string;
  included?: boolean;
  message_at?: string;
}

export interface LoraTrainDatasetPreview {
  rows: LoraTrainDatasetRow[];
  count: number;
  min_rows: number;
}

export interface PacksAPIResponse {
  packs: PackStatus[];
  pack_id?: string;
  layout_owner?: string;
  layout_profile?: string;
  capabilities?: string[];
  capability_registry?: ResolvedCapability[];
  short_id_collisions?: string[];
}

export interface ExpertPresetOption {
  slug: string;
  label: string;
  from_pack?: string;
}

export type CadParam = {
  name: string;
  value: string;
  section?: string;
  comment?: string;
  min?: number;
  max?: number;
  step?: number;
};

export class ChatAPI {
  private baseURL: string;
  private commandsCache: CommandDefinition[] | null = null;
  private readonly packsApi: PacksApi;
  private readonly channelsApi: ChannelsApi;
  private readonly messagesApi: MessagesApi;
  private readonly collabApi: CollabApi;
  private readonly agentsApi: AgentsApi;
  private readonly artifactsApi: ArtifactsApi;
  private readonly runbooksApi: RunbooksApi;
  private readonly roomsApi: RoomsApi;
  private readonly connectorsApi: ConnectorsApi;
  private readonly streamsApi: StreamsApi;
  private readonly gitChangesApi: GitChangesApi;

  constructor(serverAddr: string = getHubBaseURL()) {
    this.baseURL = normalizeHubBaseURL(serverAddr);
    const hubFetch = (path: string, init?: RequestInit) => this.hubFetch(path, init);
    this.packsApi = new PacksApi(hubFetch);
    this.channelsApi = new ChannelsApi(hubFetch);
    this.messagesApi = new MessagesApi(hubFetch);
    this.collabApi = new CollabApi(hubFetch);
    this.agentsApi = new AgentsApi(hubFetch);
    this.artifactsApi = new ArtifactsApi(hubFetch);
    this.runbooksApi = new RunbooksApi(hubFetch);
    this.roomsApi = new RoomsApi(hubFetch, this.baseURL);
    this.connectorsApi = new ConnectorsApi(hubFetch);
    this.streamsApi = new StreamsApi(hubFetch);
    this.gitChangesApi = new GitChangesApi(hubFetch);
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

  /** Hub fetch with auth headers on every request. Clears session on 401. */
  private async hubFetch(path: string, init?: RequestInit): Promise<Response> {
    const extra = (init?.headers as Record<string, string> | undefined) ?? {};
    const url = path.startsWith('http') ? path : `${this.baseURL}${path}`;
    const response = await fetch(url, {
      ...init,
      headers: { ...this.hubHeaders(), ...extra },
    });
    if (response.status === 401) {
      setHubSessionToken(null);
      window.dispatchEvent(new CustomEvent('nj-hub-unauthorized'));
    }
    return response;
  }

  /** Create or refresh a hub user session (channel ACL). */
  async createSession(username: string): Promise<{ token: string; username: string; role?: string }> {
    return this.roomsApi.createSession(username);
  }

  async createRoom(params?: {
    name?: string;
    ttl_hours?: number;
    max_members?: number;
  }): Promise<{ room: any; channel: any }> {
    return this.roomsApi.createRoom(params);
  }

  async leaveRoom(roomId: string): Promise<void> {
    return this.roomsApi.leaveRoom(roomId);
  }

  async endRoom(roomId: string): Promise<void> {
    return this.roomsApi.endRoom(roomId);
  }

  async getRoom(roomId: string): Promise<any> {
    return this.roomsApi.getRoom(roomId);
  }

  async getRoomPresence(roomId: string): Promise<{ room_id: string; members: any[] }> {
    return this.roomsApi.getRoomPresence(roomId);
  }

  async joinRoom(
    hostHubUrl: string,
    joinCode: string,
    username: string
  ): Promise<{ room: any; session: { token: string; username: string }; hub_url: string; hub_token: string; room_channel: string }> {
    return this.roomsApi.joinRoom(hostHubUrl, joinCode, username);
  }

  /** Mint an admin session using the hub bootstrap secret (API keys, ACL admin). */
  async createAdminSession(
    username: string,
    bootstrapToken: string
  ): Promise<{ token: string; username: string; role: string }> {
    return this.roomsApi.createAdminSession(username, bootstrapToken);
  }

  // Fetch existing messages for a channel
  async fetchMessages(channel: string, limit: number = 50, beforeId?: string): Promise<Message[]> {
    return this.messagesApi.fetchMessages(channel, limit, beforeId);
  }

  async searchMessages(channel: string, query: string, limit: number = 50): Promise<Message[]> {
    const params = new URLSearchParams({ channel, q: query, limit: String(limit) });
    const response = await this.hubFetch(`/api/messages/search?${params}`);
    if (!response.ok) {
      throw new Error(`Failed to search messages: ${response.statusText}`);
    }
    return response.json();
  }

  async fetchGitChanges(userId: string): Promise<GitChangeProposal[]> {
    return this.gitChangesApi.fetchGitChanges(userId);
  }

  async approveGitChange(changeId: string): Promise<GitChangeProposal> {
    return this.gitChangesApi.approveGitChange(changeId);
  }

  async rejectGitChange(changeId: string, reason?: string): Promise<GitChangeProposal> {
    return this.gitChangesApi.rejectGitChange(changeId, reason);
  }

  async fetchTurnTrace(channel: string, messageId: string, q?: string): Promise<Record<string, unknown>> {
    const params = new URLSearchParams({ channel, message_id: messageId });
    if (q?.trim()) params.set('q', q.trim());
    const response = await this.hubFetch(`/api/debug/turn-trace?${params}`);
    if (!response.ok) {
      throw new Error(`Failed to fetch turn trace: ${response.statusText}`);
    }
    return response.json();
  }

  async listAPIKeys(): Promise<Array<Record<string, unknown>>> {
    return this.roomsApi.listAPIKeys();
  }

  async createAPIKey(name: string, role: string): Promise<{ api_key: string; record: Record<string, unknown> }> {
    return this.roomsApi.createAPIKey(name, role);
  }

  async revokeAPIKey(id: string): Promise<void> {
    return this.roomsApi.revokeAPIKey(id);
  }

  async fetchCollaborations(channel?: string, includeTerminal: boolean = false): Promise<Collaboration[]> {
    return this.collabApi.fetchCollaborations(channel, includeTerminal);
  }

  async fetchArtifacts(filters: {
    workspace_id?: string;
    project_id?: string;
    channel_id?: string;
    collaboration_id?: string;
    renderer_id?: string;
    kind?: string;
  } = {}): Promise<StoredArtifact[]> {
    return this.artifactsApi.fetchArtifacts(filters);
  }

  async fetchArtifact(id: string): Promise<StoredArtifact> {
    return this.artifactsApi.fetchArtifact(id);
  }

  async createArtifact(artifact: Partial<StoredArtifact>): Promise<StoredArtifact> {
    return this.artifactsApi.createArtifact(artifact);
  }

  async updateArtifact(artifact: StoredArtifact): Promise<StoredArtifact> {
    return this.artifactsApi.updateArtifact(artifact);
  }

  async deleteArtifact(id: string, revision: number): Promise<void> {
    return this.artifactsApi.deleteArtifact(id, revision);
  }

  async fetchArtifactRevisions(id: string): Promise<StoredArtifactRevision[]> {
    return this.artifactsApi.fetchArtifactRevisions(id);
  }

  async fetchArtifactRevision(id: string, revision: number): Promise<StoredArtifactRevision> {
    return this.artifactsApi.fetchArtifactRevision(id, revision);
  }

  async duplicateArtifact(id: string, newId = ''): Promise<StoredArtifact> {
    return this.artifactsApi.duplicateArtifact(id, newId);
  }

  async exportArtifact(id: string, workspaceId: string, path: string, channel = ''): Promise<FileChange> {
    return this.artifactsApi.exportArtifact(id, workspaceId, path, channel);
  }

  /** Load a Neural Canvas artifact asset as a data URL (auth via hub session). */
  async fetchArtifactAssetDataUrl(artifactId: string, name: string): Promise<string> {
    return this.artifactsApi.fetchArtifactAssetDataUrl(artifactId, name);
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
    return this.collabApi.acknowledgeCollaborationWorkspace(collaborationId, sourceRepoPath);
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
    return this.runbooksApi.createRunbook(body);
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
    return this.runbooksApi.updateRunbook(collabId, body);
  }

  async getRunbook(collabId: string): Promise<Collaboration> {
    return this.collabApi.getRunbook(collabId);
  }

  async suggestRunbookAssignee(
    collabId: string,
    title: string,
    description: string
  ): Promise<AssignSuggestion | null> {
    return this.runbooksApi.suggestRunbookAssignee(collabId, title, description);
  }

  async parseRunbookPlan(collabId: string, markdown: string): Promise<CollaborationTask[]> {
    return this.runbooksApi.parseRunbookPlan(collabId, markdown);
  }

  async submitRunbook(collabId: string): Promise<Collaboration> {
    return this.runbooksApi.submitRunbook(collabId);
  }

  async startRunbook(collabId: string, inputs?: Record<string, string>): Promise<Collaboration> {
    return this.runbooksApi.startRunbook(collabId, inputs);
  }

  async listRunbookDefinitions(): Promise<RunbookDefinitionSummary[]> {
    return this.runbooksApi.listRunbookDefinitions();
  }

  async getRunbookDefinition(id: string, version?: number): Promise<RunbookDefinition> {
    return this.runbooksApi.getRunbookDefinition(id, version);
  }

  async saveRunbookDefinition(def: RunbookDefinition): Promise<RunbookDefinition> {
    return this.runbooksApi.saveRunbookDefinition(def);
  }

  async exportRunbookDefinition(id: string, version?: number): Promise<RunbookDefinitionBundle> {
    return this.runbooksApi.exportRunbookDefinition(id, version);
  }

  async importRunbookDefinition(
    bundleOrDefinition: RunbookDefinitionBundle | RunbookDefinition,
    options?: { keepId?: boolean }
  ): Promise<RunbookDefinition> {
    return this.runbooksApi.importRunbookDefinition(bundleOrDefinition, options);
  }

  async getRunbookRunProvenance(collaborationId: string): Promise<RunbookRunProvenance> {
    return this.runbooksApi.getRunbookRunProvenance(collaborationId);
  }

  async instantiateRunbookDefinition(
    definitionId: string,
    body: { channel: string; created_by: string; agent_ids: string[]; inputs?: Record<string, string> }
  ): Promise<{ collaboration_id: string; collaboration_channel: string; collaboration: Collaboration }> {
    return this.runbooksApi.instantiateRunbookDefinition(definitionId, body);
  }

  async listRunbookRuns(definitionId?: string): Promise<RunbookRunRecord[]> {
    return this.runbooksApi.listRunbookRuns(definitionId);
  }

  async replayRunbookRun(collabId: string): Promise<{ collaboration_id: string; collaboration_channel: string; collaboration: Collaboration }> {
    return this.runbooksApi.replayRunbookRun(collabId);
  }

  async listConnectors(): Promise<ConnectorProfile[]> {
    return this.connectorsApi.listConnectors();
  }

  async saveConnector(profile: ConnectorProfile & { secret?: string }, isNew: boolean): Promise<ConnectorProfile> {
    return this.connectorsApi.saveConnector(profile, isNew);
  }

  async deleteConnector(id: string): Promise<void> {
    return this.connectorsApi.deleteConnector(id);
  }

  async getStreamStatus(): Promise<StreamManagerStatus> {
    return this.streamsApi.getStreamStatus();
  }

  async restartStreamManager(): Promise<void> {
    return this.streamsApi.restartStreamManager();
  }

  async listStreamSubscriptions(): Promise<StreamSubscription[]> {
    return this.streamsApi.listStreamSubscriptions();
  }

  async saveStreamSubscription(sub: StreamSubscription, isNew: boolean): Promise<StreamSubscription> {
    return this.streamsApi.saveStreamSubscription(sub, isNew);
  }

  async deleteStreamSubscription(id: string): Promise<void> {
    return this.streamsApi.deleteStreamSubscription(id);
  }

  async testStreamSubscription(
    id: string,
    payload: string,
    topic?: string
  ): Promise<StreamDispatchResult> {
    return this.streamsApi.testStreamSubscription(id, payload, topic);
  }

  async listPackRunbooks(): Promise<{ pack_id: string; path: string; title: string }[]> {
    return this.runbooksApi.listPackRunbooks();
  }

  async importPackRunbook(packId: string, path: string): Promise<RunbookDefinition> {
    return this.runbooksApi.importPackRunbook(packId, path);
  }

  async listRunbookTemplates(): Promise<RunbookDefinitionSummary[]> {
    return this.runbooksApi.listRunbookTemplates();
  }

  async createRunbookFromTemplate(
    templateName: string,
    body: { channel: string; created_by: string; agent_ids: string[] }
  ): Promise<{ collaboration_id: string; collaboration_channel: string; collaboration: Collaboration }> {
    return this.runbooksApi.createRunbookFromTemplate(templateName, body);
  }

  async collabTaskComplete(collabId: string, taskId: string): Promise<Collaboration> {
    return this.collabApi.collabTaskComplete(collabId, taskId);
  }

  async collabTaskSkip(collabId: string, taskId: string): Promise<Collaboration> {
    return this.collabApi.collabTaskSkip(collabId, taskId);
  }

  async collabTaskRedispatch(collabId: string, taskId: string): Promise<Collaboration> {
    return this.collabApi.collabTaskRedispatch(collabId, taskId);
  }

  async collabTaskReassign(collabId: string, taskId: string, agentId: string): Promise<Collaboration> {
    return this.collabApi.collabTaskReassign(collabId, taskId, agentId);
  }

  async collabPause(collabId: string): Promise<Collaboration> {
    return this.collabApi.collabPause(collabId);
  }

  async collabResume(collabId: string): Promise<Collaboration> {
    return this.collabApi.collabResume(collabId);
  }

  async approveCollabParticipantRequest(collabId: string, agentId: string): Promise<Collaboration> {
    return this.collabApi.approveCollabParticipantRequest(collabId, agentId);
  }

  async denyCollabParticipantRequest(collabId: string, agentId: string): Promise<Collaboration> {
    return this.collabApi.denyCollabParticipantRequest(collabId, agentId);
  }

  /** Cursor-style Stop: pause agents on a channel until the user sends a message. */
  async channelInterject(channel: string, heldBy?: string): Promise<{ channel: string; held: boolean }> {
    return this.messagesApi.channelInterject(channel, heldBy);
  }

  // Send a message to the server
  async sendMessage(
    channel: string,
    content: string,
    from: { name: string; type: string },
    type: string = 'question',
    credentials?: Record<string, any>
  ): Promise<SendMessageResponse> {
    return this.messagesApi.sendMessage(channel, content, from, type, credentials);
  }

  /** Classify a turn and return a context_request before uploading payloads. */
  async prepareTurn(
    channel: string,
    content: string,
    from: { name: string; type: string },
    type: string = 'question',
    metadata?: Record<string, any>
  ): Promise<{
    prepare_token: string;
    context_request: import('../utils/contextRequestAttach').ContextRequestPayload;
    decision?: Record<string, unknown>;
  }> {
    return this.messagesApi.prepareTurn(channel, content, from, type, metadata);
  }

  /** Finalize a prepared turn after uploading requested context. */
  async dispatchTurn(
    channel: string,
    content: string,
    from: { name: string; type: string },
    type: string = 'question',
    metadata?: Record<string, any>
  ): Promise<SendMessageResponse> {
    return this.messagesApi.dispatchTurn(channel, content, from, type, metadata);
  }

  // Fetch list of active agents
  async fetchAgents(options?: { includeToolCounts?: boolean }): Promise<AgentInfo[]> {
    return this.agentsApi.fetchAgents(options);
  }

  async fetchAgentTools(agentId: string): Promise<AgentToolCapabilities> {
    return this.agentsApi.fetchAgentTools(agentId);
  }

  async fetchChannelTools(channel: string): Promise<ChannelToolsResponse> {
    return this.agentsApi.fetchChannelTools(channel);
  }

  async fetchCapabilityPolicy(): Promise<CapabilityPolicyResponse> {
    return this.agentsApi.fetchCapabilityPolicy();
  }

  async updateCapabilityPolicy(update: CapabilityPolicyUpdate): Promise<CapabilityPolicyResponse> {
    return this.agentsApi.updateCapabilityPolicy(update);
  }

  // Fetch list of channels
  async fetchChannels(): Promise<Channel[]> {
    return this.channelsApi.fetchChannels();
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

  async getWebSearchConfig(): Promise<WebSearchConfigResponse> {
    const response = await this.hubFetch(`/api/web-search/config`);
    if (!response.ok) {
      throw new Error(`Failed to fetch web search config: ${response.statusText}`);
    }
    return response.json();
  }

  async saveWebSearchConfig(body: {
    enabled?: boolean;
    provider?: string;
    api_key?: string;
    max_results?: number;
    keyless?: boolean;
  }): Promise<{ status: string }> {
    const response = await this.hubFetch(`/api/web-search/config`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    const data = await response.json();
    if (!response.ok) {
      throw new Error(data.error || `Failed to save web search config: ${response.statusText}`);
    }
    return data;
  }

  async testWebSearchConnection(): Promise<{ status: string; results?: Array<{ title: string; url: string; description: string }> }> {
    const response = await this.hubFetch(`/api/web-search/test`, { method: 'POST' });
    const data = await response.json();
    if (!response.ok) {
      throw new Error(data.error || `Web search test failed: ${response.statusText}`);
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

  // Get WebSocket URL for a channel. extraChannels are additional hub channels to watch on the same socket.
  getWebSocketURL(channel: string, extraChannels: string[] = []): string {
    return buildChannelWebSocketURL(this.baseURL, channel, extraChannels);
  }

  // Get WebSocket URL for a thread
  getThreadWebSocketURL(channel: string, threadId: string): string {
    return buildThreadWebSocketURL(this.baseURL, channel, threadId);
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

  async sendThreadReply(
    threadId: string,
    channel: string,
    content: string,
    from: { name: string; type: string },
    metadata?: Record<string, unknown>
  ): Promise<void> {
    return this.messagesApi.sendThreadReply(threadId, channel, content, from, metadata);
  }

  // Fetch thread metadata
  async fetchThreadMetadata(threadId: string): Promise<ThreadMetadata> {
    return this.messagesApi.fetchThreadMetadata(threadId);
  }

  // Fetch my agents
  async fetchMyAgents(): Promise<CachedAgentInfo[]> {
    return this.agentsApi.fetchMyAgents();
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
    return this.agentsApi.fetchRemovedAgents();
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

  async addWorkspace(
    name: string,
    path: string,
    options?: { create?: boolean; parentPath?: string },
  ): Promise<any> {
    const response = await this.hubFetch(`/api/workspaces`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        name,
        path,
        create: options?.create === true,
        parent_path: options?.parentPath?.trim() || undefined,
      }),
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

  async connectRemoteWorkspace(params: {
    name: string;
    remoteHost: string;
    remoteUser: string;
    remotePath: string;
    sidecarUrl: string;
    token: string;
    kind?: 'ssh' | 'devcontainer';
  }): Promise<any> {
    const response = await this.hubFetch('/api/workspaces/connect-remote', {
      method: 'POST',
      body: JSON.stringify({
        name: params.name,
        remote_host: params.remoteHost,
        remote_user: params.remoteUser,
        remote_path: params.remotePath,
        sidecar_url: params.sidecarUrl,
        token: params.token,
        kind: params.kind ?? 'ssh',
      }),
    });
    if (!response.ok) {
      const text = await response.text();
      throw new Error(text || `Failed to connect remote workspace: ${response.statusText}`);
    }
    return response.json();
  }

  async fetchDevcontainerPlan(workspaceId: string): Promise<any> {
    const response = await this.hubFetch(
      `/api/workspaces/devcontainer-plan?workspace=${encodeURIComponent(workspaceId)}`
    );
    if (!response.ok) {
      throw new Error(`Failed to fetch devcontainer plan: ${response.statusText}`);
    }
    return response.json();
  }

  async fetchDevcontainerPlanByPath(repoPath: string): Promise<any> {
    const response = await this.hubFetch(
      `/api/workspaces/devcontainer-plan?path=${encodeURIComponent(repoPath)}`
    );
    if (!response.ok) {
      const text = await response.text();
      throw new Error(text || `Failed to fetch devcontainer plan: ${response.statusText}`);
    }
    return response.json();
  }

  async pingSidecar(sidecarUrl: string, token?: string): Promise<boolean> {
    const url = `${sidecarUrl.replace(/\/$/, '')}/health`;
    const headers: Record<string, string> = {};
    if (token) headers.Authorization = `Bearer ${token}`;
    try {
      const res = await fetch(url, { headers });
      return res.ok;
    } catch {
      return false;
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

  async renderCAD(body: {
    workspace: string;
    path: string;
    project_id?: string;
    params?: Record<string, string>;
    output_path?: string;
  }): Promise<{ content_base64: string; params?: unknown[]; scad_path?: string; stl_path?: string }> {
    const response = await this.hubFetch('/api/cad/render', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    if (!response.ok) {
      const text = await response.text();
      throw new Error(text || `CAD render failed: ${response.statusText}`);
    }
    return response.json();
  }

  async fetchCADMesh(
    workspaceId: string,
    scadPath: string,
    projectId?: string
  ): Promise<{ content_base64: string }> {
    const params = new URLSearchParams({ workspace: workspaceId, path: scadPath });
    if (projectId) params.set('project_id', projectId);
    const response = await this.hubFetch(`/api/cad/mesh?${params.toString()}`);
    if (!response.ok) {
      throw new Error(await response.text());
    }
    return response.json();
  }

  async fetchCADParams(
    workspaceId: string,
    scadPath: string,
    projectId?: string
  ): Promise<{ params: CadParam[] }> {
    const params = new URLSearchParams({ workspace: workspaceId, path: scadPath });
    if (projectId) params.set('project_id', projectId);
    const response = await this.hubFetch(`/api/cad/params?${params.toString()}`);
    if (!response.ok) {
      throw new Error(await response.text());
    }
    return response.json();
  }

  async fetchCADVersions(projectId: string): Promise<{ versions: Array<{ id: string; label: string; created_at: string }> }> {
    const response = await this.hubFetch(`/api/cad/versions?project_id=${encodeURIComponent(projectId)}`);
    if (!response.ok) {
      throw new Error(await response.text());
    }
    return response.json();
  }

  async saveCADVersion(body: {
    workspace: string;
    path: string;
    project_id: string;
    label: string;
    params?: Record<string, string>;
  }): Promise<unknown> {
    const response = await this.hubFetch('/api/cad/versions', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    if (!response.ok) {
      throw new Error(await response.text());
    }
    return response.json();
  }

  async restoreCADVersion(
    projectId: string,
    versionId: string
  ): Promise<{ content?: string; scad_path?: string }> {
    const response = await this.hubFetch('/api/cad/versions/restore', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ project_id: projectId, version_id: versionId }),
    });
    if (!response.ok) {
      throw new Error(await response.text());
    }
    return response.json();
  }

  async testOpenSCAD(path?: string): Promise<{ ok: boolean; message: string }> {
    const response = await this.hubFetch('/api/cad/test-openscad', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ path: path ?? '' }),
    });
    const data = (await response.json()) as { ok: boolean; message: string };
    return data;
  }

  async checkCADPrintability(body: {
    stl_path: string;
    min_wall_mm?: number;
  }): Promise<{
    printable?: boolean;
    warnings?: string[];
    overhang?: { max_angle_deg?: number; faces_over_limit?: number };
    estimated_min_wall_mm?: number;
  }> {
    const response = await this.hubFetch('/api/cad/printability', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    if (!response.ok) {
      throw new Error(await response.text());
    }
    return response.json();
  }

  async validateCADAssembly(body: {
    manifest_path: string;
    clearance_mm?: number;
  }): Promise<{ ok?: boolean; bom?: Array<{ part_id: string; name: string }>; fit_issues?: unknown[] }> {
    const response = await this.hubFetch('/api/cad/assembly/validate', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    if (!response.ok) {
      throw new Error(await response.text());
    }
    return response.json();
  }

  async createFile(workspaceId: string, path: string, content: string = '', isDir = false): Promise<void> {
    const response = await this.hubFetch(`/api/file-create`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        workspace_id: workspaceId,
        path,
        content,
        is_dir: isDir,
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
    context?: string;
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
        context: params.context,
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
    repoPath?: string;
    repoPaths?: string[];
    query: string;
    limit?: number;
  }): Promise<{
    chunks: Array<{ path: string; content: string; repo_path?: string; repo_name?: string }>;
  }> {
    const paths =
      params.repoPaths?.filter((p) => p?.trim()).map((p) => p.trim()) ??
      (params.repoPath?.trim() ? [params.repoPath.trim()] : []);
    const response = await this.hubFetch('/api/repo/search/semantic', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        repo_paths: paths.length > 1 ? paths : undefined,
        repo_path: paths.length === 1 ? paths[0] : params.repoPath,
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

  async repoGraph(repoPath: string): Promise<import('../components/knowledge-graph/types').KnowledgeGraphSummary> {
    const params = new URLSearchParams({ repo_path: repoPath });
    const response = await this.hubFetch(`/api/repo/graph?${params}`);
    if (!response.ok) {
      throw new Error(`Knowledge graph failed: ${response.statusText}`);
    }
    return response.json();
  }

  async repoGraphSubgraph(
    repoPath: string,
    q: string,
    hops = 1,
    limit = 120,
  ): Promise<import('../components/knowledge-graph/types').KnowledgeGraphSummary & { query?: string; nodes: import('../components/knowledge-graph/types').KnowledgeGraphNode[]; edges: import('../components/knowledge-graph/types').KnowledgeGraphEdge[] }> {
    const params = new URLSearchParams({
      repo_path: repoPath,
      q,
      hops: String(hops),
      limit: String(limit),
    });
    const response = await this.hubFetch(`/api/repo/graph/subgraph?${params}`);
    if (!response.ok) {
      throw new Error(`Graph subgraph failed: ${response.statusText}`);
    }
    return response.json();
  }

  async repoGraphPath(
    repoPath: string,
    from: string,
    to: string,
  ): Promise<{
    from: string;
    to: string;
    found: boolean;
    nodes: import('../components/knowledge-graph/types').KnowledgeGraphNode[];
    edges: import('../components/knowledge-graph/types').KnowledgeGraphEdge[];
  }> {
    const params = new URLSearchParams({ repo_path: repoPath, from, to });
    const response = await this.hubFetch(`/api/repo/graph/path?${params}`);
    if (!response.ok) {
      throw new Error(`Graph path failed: ${response.statusText}`);
    }
    return response.json();
  }

  async repoGraphExplain(
    repoPath: string,
    node: string,
  ): Promise<import('../components/knowledge-graph/types').KnowledgeGraphExplain> {
    const params = new URLSearchParams({ repo_path: repoPath, node });
    const response = await this.hubFetch(`/api/repo/graph/explain?${params}`);
    if (!response.ok) {
      throw new Error(`Graph explain failed: ${response.statusText}`);
    }
    return response.json();
  }

  async repoGraphStatus(
    repoPath: string,
    rebuild = false,
  ): Promise<import('../components/knowledge-graph/types').KnowledgeGraphMeta> {
    const params = new URLSearchParams({ repo_path: repoPath });
    if (rebuild) params.set('rebuild', '1');
    const response = await this.hubFetch(`/api/repo/graph/status?${params}`);
    if (!response.ok) {
      throw new Error(`Graph status failed: ${response.statusText}`);
    }
    return response.json();
  }

  // Tool approval API methods

  async fetchPendingToolApprovals(): Promise<
    Array<{
      id: string;
      agent_id: string;
      agent_name: string;
      session_id?: string;
      tool_name: string;
      tool_input?: Record<string, unknown>;
      channel: string;
      created_at: string;
    }>
  > {
    const response = await this.hubFetch('/api/tool-approvals/pending');
    if (!response.ok) {
      throw new Error(`Failed to list pending tool approvals: ${response.statusText}`);
    }
    return response.json();
  }

  async approveToolCall(approvalId: string, scope: 'once' | 'always' = 'once'): Promise<void> {
    const response = await this.hubFetch(`/api/tool-approvals/approve/${approvalId}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ scope }),
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

  async publishDeviceLocation(body: {
    lat: number;
    lon: number;
    accuracy_m?: number;
    display_name?: string;
    captured_at?: string;
    session_id?: string;
    shared?: boolean;
    source?: string;
  }): Promise<{ ok: boolean; location?: Record<string, unknown> }> {
    const response = await this.hubFetch('/api/maps/device-location', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    if (!response.ok) {
      throw new Error(`Failed to publish device location: ${response.statusText}`);
    }
    return response.json();
  }

  async clearDeviceLocation(): Promise<void> {
    const response = await this.hubFetch('/api/maps/device-location', { method: 'DELETE' });
    if (!response.ok) {
      throw new Error(`Failed to clear device location: ${response.statusText}`);
    }
  }

  async fetchPendingLocationRequests(): Promise<
    Array<{
      id: string;
      agent_id?: string;
      agent_name?: string;
      channel?: string;
      created_at: string;
      status: string;
    }>
  > {
    const response = await this.hubFetch('/api/maps/location-requests/pending');
    if (!response.ok) {
      throw new Error(`Failed to list location requests: ${response.statusText}`);
    }
    return response.json();
  }

  async fulfillLocationRequest(
    requestId: string,
    body: {
      lat: number;
      lon: number;
      accuracy_m?: number;
      display_name?: string;
      captured_at?: string;
    },
  ): Promise<void> {
    const response = await this.hubFetch(`/api/maps/location-requests/${requestId}/fulfill`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    if (!response.ok) {
      throw new Error(`Failed to fulfill location request: ${response.statusText}`);
    }
  }

  async rejectLocationRequest(requestId: string, reason?: string): Promise<void> {
    const response = await this.hubFetch(`/api/maps/location-requests/${requestId}/reject`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ reason: reason ?? 'User declined to share location' }),
    });
    if (!response.ok) {
      throw new Error(`Failed to reject location request: ${response.statusText}`);
    }
  }

  async reverseGeocode(lat: number, lon: number): Promise<{ display_name?: string }> {
    const response = await this.hubFetch('/api/maps/reverse', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ lat, lon }),
    });
    if (!response.ok) {
      return {};
    }
    return response.json();
  }

  async answerUserQuestion(questionId: string, answer: string): Promise<void> {
    return this.messagesApi.answerUserQuestion(questionId, answer);
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

  // Approve a file change (optional new_content when editor buffer was partially edited)
  async approveFileChange(
    changeId: string,
    userId: string = 'default',
    newContent?: string
  ): Promise<FileChange> {
    const body =
      newContent !== undefined && newContent !== ''
        ? JSON.stringify({ new_content: newContent })
        : undefined;
    const response = await this.hubFetch(`/api/file-changes/approve/${changeId}?user_id=${encodeURIComponent(userId)}`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body,
    });

    if (!response.ok) {
      const detail = (await response.text()).trim();
      throw new Error(detail || `Failed to approve file change: ${response.statusText}`);
    }

    return response.json();
  }

  // Update pending file change content (partial hunk accept)
  async updateFileChangeContent(changeId: string, newContent: string): Promise<FileChangeDiff> {
    const response = await this.hubFetch(`/api/file-changes/${changeId}`, {
      method: 'PATCH',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ new_content: newContent }),
    });

    if (!response.ok) {
      const detail = (await response.text()).trim();
      throw new Error(detail || `Failed to update file change: ${response.statusText}`);
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
      const detail = (await response.text()).trim();
      throw new Error(detail || `Failed to reject file change: ${response.statusText}`);
    }

    return response.json();
  }

  // Get file change diff
  async getFileDiff(changeId: string): Promise<FileChangeDiff> {
    const response = await this.hubFetch(`/api/file-changes/${changeId}`);
    
    if (!response.ok) {
      const detail = (await response.text()).trim();
      throw new Error(detail || `Failed to get file diff: ${response.statusText}`);
    }
    
    return response.json();
  }

  async fetchPacks(): Promise<PacksAPIResponse> {
    return this.packsApi.fetchPacks();
  }

  async fetchPackCatalog(): Promise<PackCatalogEntry[]> {
    return this.packsApi.fetchPackCatalog();
  }

  async refreshPackCatalog(): Promise<PackCatalogEntry[]> {
    return this.packsApi.refreshPackCatalog();
  }

  async fetchPackUpdates(): Promise<PackUpdatesResponse> {
    return this.packsApi.fetchPackUpdates();
  }

  async upgradePack(packId: string): Promise<PacksAPIResponse> {
    return this.packsApi.upgradePack(packId);
  }

  async installPack(packId: string): Promise<PacksAPIResponse> {
    return this.packsApi.installPack(packId);
  }

  async installPackFromZip(packZipBase64: string): Promise<PacksAPIResponse> {
    const response = await this.hubFetch(`/api/packs/install-zip`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ pack_zip_base64: packZipBase64 }),
    });
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return this.parsePacksMutationResponse(await response.json());
  }

  async installPackLoRAs(packId: string): Promise<InstallPackLoRAsResponse> {
    return this.packsApi.installPackLoRAs(packId);
  }

  async fetchACEStepStatus(packId = 'music-creation'): Promise<ACEStepStatus> {
    const response = await this.hubFetch(
      `/api/packs/${encodeURIComponent(packId)}/acestep-status`,
    );
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return response.json();
  }

  async installACEStep(
    packId = 'music-creation',
    modelVariant?: string,
  ): Promise<InstallACEStepResponse> {
    const response = await this.hubFetch(
      `/api/packs/${encodeURIComponent(packId)}/install-acestep`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ model_variant: modelVariant ?? 'sft' }),
      },
    );
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return response.json();
  }

  async restartMusicSidecar(packId = 'music-creation'): Promise<{ status: string; acestep: ACEStepStatus }> {
    const response = await this.hubFetch(
      `/api/packs/${encodeURIComponent(packId)}/restart-sidecar`,
      { method: 'POST' },
    );
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return response.json();
  }

  async fetchArenaSidecarStatus(packId = 'model-arena'): Promise<ArenaSidecarStatus> {
    const response = await this.hubFetch(
      `/api/packs/${encodeURIComponent(packId)}/sidecar-status`,
    );
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return response.json();
  }

  async installArenaSidecarDeps(
    packId = 'model-arena',
  ): Promise<InstallArenaSidecarResponse> {
    const response = await this.hubFetch(
      `/api/packs/${encodeURIComponent(packId)}/install-sidecar-deps`,
      { method: 'POST' },
    );
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return response.json();
  }

  async restartArenaSidecar(
    packId = 'model-arena',
  ): Promise<{ status: string; sidecar: ArenaSidecarStatus }> {
    const response = await this.hubFetch(
      `/api/packs/${encodeURIComponent(packId)}/arena-restart-sidecar`,
      { method: 'POST' },
    );
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return response.json();
  }

  async fetchAIInterviewProgress(): Promise<AIInterviewProgressResponse> {
    const response = await this.hubFetch('/api/ai-interview/progress');
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return response.json();
  }

  async startAIInterviewDay(): Promise<AIInterviewProgressResponse> {
    const response = await this.hubFetch('/api/ai-interview/start', { method: 'POST' });
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return response.json();
  }

  async completeAIInterviewDay(
    day: number,
    body: { concept?: boolean; drill?: boolean; complete?: boolean; advance?: boolean },
  ): Promise<AIInterviewProgressResponse> {
    const response = await this.hubFetch(`/api/ai-interview/days/${day}/complete`, {
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

  async submitAIInterviewGate(
    gateId: string,
    body: { eval_notes?: string; mock_notes?: string; score?: number },
  ): Promise<AIInterviewProgressResponse> {
    const response = await this.hubFetch(
      `/api/ai-interview/gates/${encodeURIComponent(gateId)}/submit`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      },
    );
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return response.json();
  }

  async unlockAIInterviewCert(): Promise<{
    certification: Record<string, unknown>;
    credential: Record<string, unknown>;
  }> {
    const response = await this.hubFetch('/api/ai-interview/cert/unlock', { method: 'POST' });
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return response.json();
  }

  async fetchImageGenStatus(): Promise<ImageGenStatus> {
    const response = await this.hubFetch('/api/image-gen/status');
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

  async setLayoutOwner(packId: string): Promise<PacksAPIResponse> {
    return this.packsApi.setLayoutOwner(packId);
  }

  async validatePack(body: {
    pack_zip_base64?: string;
    pack_dir?: string;
    pack_yaml?: string;
  }): Promise<PackValidationReport> {
    const response = await this.hubFetch(`/api/packs/validate`, {
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

  async devLinkPack(packDir: string): Promise<PacksAPIResponse> {
    const response = await this.hubFetch(`/api/packs/dev-link`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ pack_dir: packDir }),
    });
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return this.parsePacksMutationResponse(await response.json());
  }

  async devReloadPack(packId: string): Promise<PacksAPIResponse> {
    const response = await this.hubFetch(`/api/packs/dev-reload`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ pack_id: packId }),
    });
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return this.parsePacksMutationResponse(await response.json());
  }

  async devUnlinkPack(packId: string): Promise<PacksAPIResponse> {
    const response = await this.hubFetch(`/api/packs/dev-unlink`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ pack_id: packId }),
    });
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return this.parsePacksMutationResponse(await response.json());
  }

  async fetchCustomerPackContext(): Promise<CustomerPackContextResponse> {
    const response = await this.hubFetch(`/api/packs/customer-context`);
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return response.json();
  }

  async fetchPhoenixStatus(): Promise<{
    environment: string;
    credentials_path?: string;
    authenticated: boolean;
    logged_in: boolean;
    identity?: string;
    hint?: string;
  }> {
    const response = await this.hubFetch('/api/phoenix/status');
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return response.json();
  }

  async fetchPhoenixAnalyses(): Promise<Array<{ id: string; label: string }>> {
    const response = await this.hubFetch('/api/phoenix/analyses');
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    const data = (await response.json()) as { analyses?: Array<{ id: string; label: string }> };
    return data.analyses ?? [];
  }

  async fetchPhoenixScanResults(): Promise<Array<{ id: string; label: string }>> {
    const response = await this.hubFetch('/api/phoenix/scan-results');
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    const data = (await response.json()) as { scan_results?: Array<{ id: string; label: string }> };
    return data.scan_results ?? [];
  }

  async phoenixLoginStart(): Promise<{
    session_id: string;
    user_code: string;
    verification_url: string;
    expires_in: number;
    environment: string;
  }> {
    const response = await this.hubFetch('/api/phoenix/login/start', { method: 'POST' });
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return response.json();
  }

  async phoenixLoginPoll(sessionId: string): Promise<{
    status: string;
    identity?: string;
    hint?: string;
    expires_in?: number;
  }> {
    const response = await this.hubFetch(
      `/api/phoenix/login/poll?session_id=${encodeURIComponent(sessionId)}`,
    );
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return response.json();
  }

  async phoenixLogout(): Promise<void> {
    const response = await this.hubFetch('/api/phoenix/logout', { method: 'POST' });
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
  }

  async phoenixImport(body: {
    workspace_id: string;
    analysis_id: string;
    scan_results_id?: string;
    output_dir?: string;
  }): Promise<{
    analysis_dir: string;
    validation_dir?: string;
    scan_export_dir?: string;
    scan_results_id?: string;
    files_written?: string[];
    attachment_notes?: string[];
  }> {
    const response = await this.hubFetch('/api/phoenix/import', {
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

  async phoenixImportScan(body: {
    workspace_id: string;
    scan_results_id: string;
    output_dir?: string;
  }): Promise<{
    analysis_dir: string;
    scan_export_dir?: string;
    scan_results_id?: string;
    files_written?: string[];
  }> {
    const response = await this.hubFetch('/api/phoenix/import-scan', {
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

  private parsePacksMutationResponse(data: Record<string, unknown>): PacksAPIResponse {
    return {
      packs: (data.packs as PackStatus[]) ?? [],
      pack_id: data.pack_id as string | undefined,
      layout_owner: data.layout_owner as string | undefined,
      layout_profile: data.layout_profile as string | undefined,
      capabilities: (data.capabilities as string[]) ?? [],
      capability_registry: (data.capability_registry as ResolvedCapability[]) ?? [],
      short_id_collisions: (data.short_id_collisions as string[]) ?? [],
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

  async fetchLoraTrainBases(): Promise<LoraTrainingBase[]> {
    const response = await this.hubFetch('/api/lora/train/bases');
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    const data = await response.json();
    return Array.isArray(data.bases) ? data.bases : [];
  }

  async previewLoraTrain(params: {
    source: string;
    source_id: string;
    thread_id?: string;
    agent_name?: string;
    agent_id?: string;
    include_learnings?: boolean;
    incremental?: boolean;
  }): Promise<number> {
    const q = new URLSearchParams({
      source: params.source,
      source_id: params.source_id,
    });
    if (params.thread_id) q.set('thread_id', params.thread_id);
    if (params.agent_name) q.set('agent_name', params.agent_name);
    if (params.agent_id) q.set('agent_id', params.agent_id);
    if (params.include_learnings) q.set('include_learnings', '1');
    if (params.incremental) q.set('incremental', '1');
    const response = await this.hubFetch(`/api/lora/train/preview?${q.toString()}`);
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    const data = await response.json();
    return Number(data.row_count ?? 0);
  }

  async previewLoraTrainDataset(
    body: Omit<LoraTrainStartRequest, 'base_ollama_tag' | 'ollama_tag' | 'hyperparams'> & {
      base_ollama_tag?: string;
      ollama_tag?: string;
    },
  ): Promise<LoraTrainDatasetPreview> {
    const response = await this.hubFetch('/api/lora/train/dataset-preview', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    const data = await response.json();
    return {
      rows: Array.isArray(data.rows) ? data.rows : [],
      count: Number(data.count ?? 0),
      min_rows: Number(data.min_rows ?? 10),
    };
  }

  async bootstrapLoraTrainFromIndex(agentId: string): Promise<LoraTrainDatasetPreview> {
    const response = await this.hubFetch('/api/lora/train/index-bootstrap', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ agent_id: agentId }),
    });
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    const data = await response.json();
    return {
      rows: Array.isArray(data.rows) ? data.rows : [],
      count: Number(data.count ?? 0),
      min_rows: 10,
    };
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

  async cancelLoraTrainJob(jobId: string): Promise<LoraTrainJob> {
    const response = await this.hubFetch(`/api/lora/train/${encodeURIComponent(jobId)}`, {
      method: 'DELETE',
    });
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return response.json();
  }

  async run12PlexQC(body: {
    workspace_id: string;
    analysis_dir: string;
    write_report?: boolean;
  }): Promise<import('../utils/secondaryAnalysis').PanelQCReport> {
    const response = await this.hubFetch('/api/secondary-analysis/12plex-qc', {
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

  async runSecondaryAnalysis(body: {
    workflow: string;
    workspace_id: string;
    config?: Record<string, unknown>;
  }): Promise<import('../utils/secondaryAnalysis').SecondaryAnalysisJob> {
    const response = await this.hubFetch('/api/secondary-analysis/run', {
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

  async fetchSecondaryAnalysisJob(
    jobId: string
  ): Promise<import('../utils/secondaryAnalysis').SecondaryAnalysisJob> {
    const response = await this.hubFetch(
      `/api/secondary-analysis/jobs/${encodeURIComponent(jobId)}`
    );
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return response.json();
  }

  async cancelSecondaryAnalysisJob(
    jobId: string
  ): Promise<import('../utils/secondaryAnalysis').SecondaryAnalysisJob> {
    const response = await this.hubFetch(
      `/api/secondary-analysis/jobs/${encodeURIComponent(jobId)}`,
      { method: 'DELETE' }
    );
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return response.json();
  }

  async fetchComparatorSummary(
    workspaceId: string,
    dir: string
  ): Promise<import('../utils/secondaryAnalysis').ComparatorSummary> {
    const params = new URLSearchParams({ workspace: workspaceId, dir });
    const response = await this.hubFetch(
      `/api/secondary-analysis/comparator-summary?${params.toString()}`
    );
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

