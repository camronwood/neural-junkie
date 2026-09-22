import type { Message, AgentInfo, Channel, ThreadMetadata, CachedAgentInfo, ConnectionTestResult, FileChange, FileChangeDiff, FileChangeRequest, GitChangeProposal, CommandDefinition, AssistantStateResponse, GoogleMeetNotesStatus, GoogleMeetNotesAppConfig, WebSearchConfigResponse, SlackConfigResponse, SlackConnectionResponse, SlackStatus, SlackBinding, SlackChannelInfo, SlackPolicy, SlackInboxConfig, SlackDiagnoseResult, SlackSmokeResult, Collaboration, CollaborationTask, AssignSuggestion, ExecutionPolicy, GraphLayout, RunbookDefinition, RunbookDefinitionSummary, RunbookRunRecord, RunbookDefinitionBundle, RunbookRunProvenance, ConnectorProfile, StreamManagerStatus, StreamSubscription, StreamDispatchResult, AgentToolCapabilities, ChannelToolsResponse, CapabilityPolicyResponse, CapabilityPolicyUpdate, StoredArtifact, StoredArtifactRevision } from '../types/protocol';
export type { ResolvedCapability } from '../types/protocol';
import {
  getHubBaseURL,
  hubAuthHeaders,
  hubSessionHeaders,
  normalizeHubBaseURL,
  setHubSessionToken,
} from '../config/hubUrl';
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
import { AssistantApi } from './domains/assistantApi';
import { SlackApi } from './domains/slackApi';
import { ProvidersApi } from './domains/providersApi';
import { WebSearchApi } from './domains/webSearchApi';
import { WorkspaceApi } from './domains/workspaceApi';
import { FilesApi } from './domains/filesApi';
import { CadApi } from './domains/cadApi';
import { GitWorkspaceApi } from './domains/gitWorkspaceApi';
import { IdeApi } from './domains/ideApi';
import { ToolApprovalsApi } from './domains/toolApprovalsApi';
import { MapsApi } from './domains/mapsApi';
import { PlansApi } from './domains/plansApi';
import { FileChangesApi } from './domains/fileChangesApi';
import { PackExtrasApi } from './domains/packExtrasApi';
import { PhoenixApi } from './domains/phoenixApi';
import { LoraApi } from './domains/loraApi';
import { SecondaryAnalysisApi } from './domains/secondaryAnalysisApi';
import { LearningsApi } from './domains/learningsApi';
import { AgentsExtrasApi } from './domains/agentsExtrasApi';
import type {
  PacksAPIResponse,
  AgentShareBundle,
  CadParam,
  ACEStepStatus,
  ArenaSidecarStatus,
  AIInterviewProgressResponse,
  ImageGenStatus,
  PackValidationReport,
  CustomerPackContextResponse,
  ExpertPresetOption,
  LoraExpertContext,
  LoraTrainingBase,
  LoraTrainDatasetPreview,
  LoraTrainJob,
  LoraTrainStartRequest,
  UserLearning,
  LearningCategory,
  LearningScope,
  LearningStats,
  InstallACEStepResponse,
  InstallArenaSidecarResponse,
  SendMessageResponse,
  PackCatalogEntry,
  PackUpdatesResponse,
  InstallPackLoRAsResponse,
} from './types/chatApiTypes';

export type {
  SendMessageResponse,
  PackStatus,
  PackManifestSummary,
  PackValidationReport,
  CustomerPackContext,
  CustomerPackContextResponse,
  PackCatalogEntry,
  InstallPackLoRAResult,
  InstallPackLoRAsResponse,
  ACEStepPaths,
  ACEStepStatus,
  InstallACEStepResponse,
  ArenaSidecarPaths,
  ArenaSidecarStatus,
  InstallArenaSidecarResponse,
  AIInterviewDayStatus,
  AIInterviewProgressResponse,
  ImageGenStatus,
  PackUpdateInfo,
  PackUpdatesResponse,
  LoraTrainingBase,
  LoraExpertContext,
  LoraTrainJob,
  LearningCategory,
  LearningScope,
  UserLearning,
  AgentShareBundle,
  LearningStats,
  LearningProposalAction,
  LoraTrainStartRequest,
  LoraTrainDatasetRow,
  LoraTrainDatasetPreview,
  PacksAPIResponse,
  ExpertPresetOption,
  CadParam,
} from './types/chatApiTypes';

export class ChatAPI {
  private baseURL: string;
  private commandsCache: CommandDefinition[] | null = null;
  private packsApi: PacksApi;
  private channelsApi: ChannelsApi;
  private messagesApi: MessagesApi;
  private collabApi: CollabApi;
  private agentsApi: AgentsApi;
  private artifactsApi: ArtifactsApi;
  private runbooksApi: RunbooksApi;
  private roomsApi: RoomsApi;
  private connectorsApi: ConnectorsApi;
  private streamsApi: StreamsApi;
  private gitChangesApi: GitChangesApi;
  private assistantApi: AssistantApi;
  private slackApi: SlackApi;
  private providersApi: ProvidersApi;
  private webSearchApi: WebSearchApi;
  private workspaceApi: WorkspaceApi;
  private filesApi: FilesApi;
  private cadApi: CadApi;
  private gitWorkspaceApi: GitWorkspaceApi;
  private ideApi: IdeApi;
  private toolApprovalsApi: ToolApprovalsApi;
  private mapsApi: MapsApi;
  private plansApi: PlansApi;
  private fileChangesApi: FileChangesApi;
  private packExtrasApi: PackExtrasApi;
  private phoenixApi: PhoenixApi;
  private loraApi: LoraApi;
  private secondaryAnalysisApi: SecondaryAnalysisApi;
  private learningsApi: LearningsApi;
  private agentsExtrasApi: AgentsExtrasApi;

  constructor(serverAddr: string = getHubBaseURL()) {
    this.baseURL = normalizeHubBaseURL(serverAddr);
    const hubFetch = (path: string, init?: RequestInit) => this.hubFetch(path, init);
    this.packsApi = new PacksApi(hubFetch);
    this.channelsApi = new ChannelsApi(hubFetch, this.baseURL);
    this.messagesApi = new MessagesApi(hubFetch);
    this.collabApi = new CollabApi(hubFetch);
    this.agentsApi = new AgentsApi(hubFetch);
    this.artifactsApi = new ArtifactsApi(hubFetch);
    this.runbooksApi = new RunbooksApi(hubFetch);
    this.roomsApi = new RoomsApi(hubFetch, this.baseURL);
    this.connectorsApi = new ConnectorsApi(hubFetch);
    this.streamsApi = new StreamsApi(hubFetch);
    this.gitChangesApi = new GitChangesApi(hubFetch);
    this.assistantApi = new AssistantApi(hubFetch);
    this.slackApi = new SlackApi(hubFetch);
    this.providersApi = new ProvidersApi(hubFetch);
    this.webSearchApi = new WebSearchApi(hubFetch);
    this.workspaceApi = new WorkspaceApi(hubFetch);
    this.filesApi = new FilesApi(hubFetch);
    this.cadApi = new CadApi(hubFetch);
    this.gitWorkspaceApi = new GitWorkspaceApi(hubFetch);
    this.ideApi = new IdeApi(hubFetch);
    this.toolApprovalsApi = new ToolApprovalsApi(hubFetch);
    this.mapsApi = new MapsApi(hubFetch);
    this.plansApi = new PlansApi(hubFetch);
    this.fileChangesApi = new FileChangesApi(hubFetch);
    this.packExtrasApi = new PackExtrasApi(hubFetch);
    this.phoenixApi = new PhoenixApi(hubFetch);
    this.loraApi = new LoraApi(hubFetch);
    this.secondaryAnalysisApi = new SecondaryAnalysisApi(hubFetch);
    this.learningsApi = new LearningsApi(hubFetch);
    this.agentsExtrasApi = new AgentsExtrasApi(hubFetch);
  }

  private hubHeaders(extra?: Record<string, string>): Record<string, string> {
    return {
      'Content-Type': 'application/json',
      ...hubAuthHeaders(),
      ...hubSessionHeaders(),
      ...extra,
    };
  }

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

  async createAdminSession(
    username: string,
    bootstrapToken: string
  ): Promise<{ token: string; username: string; role: string }> {
    return this.roomsApi.createAdminSession(username, bootstrapToken);
  }

  async fetchMessages(channel: string, limit: number = 50, beforeId?: string): Promise<Message[]> {
    return this.messagesApi.fetchMessages(channel, limit, beforeId);
  }

  async searchMessages(channel: string, query: string, limit: number = 50): Promise<Message[]> {
    return this.messagesApi.searchMessages(channel, query, limit);
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
    return this.messagesApi.fetchTurnTrace(channel, messageId, q);
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

  async fetchArtifactAssetDataUrl(artifactId: string, name: string): Promise<string> {
    return this.artifactsApi.fetchArtifactAssetDataUrl(artifactId, name);
  }

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

  async channelInterject(channel: string, heldBy?: string): Promise<{ channel: string; held: boolean }> {
    return this.messagesApi.channelInterject(channel, heldBy);
  }

  async sendMessage(
    channel: string,
    content: string,
    from: { name: string; type: string },
    type: string = 'question',
    credentials?: Record<string, any>
  ): Promise<SendMessageResponse> {
    return this.messagesApi.sendMessage(channel, content, from, type, credentials);
  }

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

  async dispatchTurn(
    channel: string,
    content: string,
    from: { name: string; type: string },
    type: string = 'question',
    metadata?: Record<string, any>
  ): Promise<SendMessageResponse> {
    return this.messagesApi.dispatchTurn(channel, content, from, type, metadata);
  }

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

  async fetchChannels(): Promise<Channel[]> {
    return this.channelsApi.fetchChannels();
  }

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
    return this.assistantApi.fetchAssistantState(channel);
  }

  async markAssistantTaskDone(taskID: string): Promise<void> {
    return this.assistantApi.markAssistantTaskDone(taskID);
  }

  async dismissAssistantReminder(reminderID: string): Promise<void> {
    return this.assistantApi.dismissAssistantReminder(reminderID);
  }

  async getGoogleMeetNotesAppConfig(): Promise<GoogleMeetNotesAppConfig> {
    return this.assistantApi.getGoogleMeetNotesAppConfig();
  }

  async saveGoogleMeetNotesAppConfig(
    clientId: string,
    clientSecret: string,
    redirectUrl?: string
  ): Promise<GoogleMeetNotesAppConfig> {
    return this.assistantApi.saveGoogleMeetNotesAppConfig(clientId, clientSecret, redirectUrl);
  }

  async getGoogleMeetNotesStatus(): Promise<GoogleMeetNotesStatus> {
    return this.assistantApi.getGoogleMeetNotesStatus();
  }

  async getGoogleMeetNotesAuthURL(): Promise<string> {
    return this.assistantApi.getGoogleMeetNotesAuthURL();
  }

  async disconnectGoogleMeetNotes(): Promise<void> {
    return this.assistantApi.disconnectGoogleMeetNotes();
  }

  async syncGoogleMeetNotes(): Promise<number> {
    return this.assistantApi.syncGoogleMeetNotes();
  }

  async getSlackConfig(): Promise<SlackConfigResponse> {
    return this.slackApi.getSlackConfig();
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
    return this.slackApi.saveSlackConfig(body);
  }

  async getWebSearchConfig(): Promise<WebSearchConfigResponse> {
    return this.webSearchApi.getWebSearchConfig();
  }

  async saveWebSearchConfig(body: {
    enabled?: boolean;
    provider?: string;
    api_key?: string;
    max_results?: number;
    keyless?: boolean;
  }): Promise<{ status: string }> {
    return this.webSearchApi.saveWebSearchConfig(body);
  }

  async testWebSearchConnection(): Promise<{ status: string; results?: Array<{ title: string; url: string; description: string }> }> {
    return this.webSearchApi.testWebSearchConnection();
  }

  async getSlackStatus(): Promise<SlackStatus> {
    return this.slackApi.getSlackStatus();
  }

  async getSlackConnection(): Promise<SlackConnectionResponse> {
    return this.slackApi.getSlackConnection();
  }

  async getSlackBindings(): Promise<SlackBinding[]> {
    return this.slackApi.getSlackBindings();
  }

  async getSlackChannels(): Promise<SlackChannelInfo[]> {
    return this.slackApi.getSlackChannels();
  }

  async saveSlackBinding(binding: {
    slack_channel_id: string;
    slack_channel_name?: string;
    agent_id: string;
    agent_name?: string;
    policy?: SlackPolicy;
    enabled?: boolean;
  }): Promise<SlackBinding> {
    return this.slackApi.saveSlackBinding(binding);
  }

  async deleteSlackBinding(slackChannelId: string): Promise<void> {
    return this.slackApi.deleteSlackBinding(slackChannelId);
  }

  async getSlackOAuthURL(): Promise<string> {
    return this.slackApi.getSlackOAuthURL();
  }

  async getSlackUserDMOAuthURL(): Promise<string> {
    return this.slackApi.getSlackUserDMOAuthURL();
  }

  async disconnectSlack(): Promise<void> {
    return this.slackApi.disconnectSlack();
  }

  async restartSlackBridge(): Promise<void> {
    return this.slackApi.restartSlackBridge();
  }

  async getSlackInbox(): Promise<SlackInboxConfig> {
    return this.slackApi.getSlackInbox();
  }

  async saveSlackInbox(body: SlackInboxConfig): Promise<SlackInboxConfig> {
    return this.slackApi.saveSlackInbox(body);
  }

  async setSlackInboxAwayEnabled(awayEnabled: boolean): Promise<SlackInboxConfig> {
    return this.slackApi.setSlackInboxAwayEnabled(awayEnabled);
  }

  async setSlackInboxForwardEnabled(forwardEnabled: boolean): Promise<SlackInboxConfig> {
    return this.slackApi.setSlackInboxForwardEnabled(forwardEnabled);
  }

  async testSlackInboxDM(text?: string): Promise<void> {
    return this.slackApi.testSlackInboxDM(text);
  }

  async slackTestPost(slackChannelId: string, text?: string): Promise<void> {
    return this.slackApi.slackTestPost(slackChannelId, text);
  }

  async getSlackDiagnose(): Promise<SlackDiagnoseResult> {
    return this.slackApi.getSlackDiagnose();
  }

  async runSlackSmoke(options?: {
    channel_id?: string;
    outbound?: boolean;
  }): Promise<SlackSmokeResult> {
    return this.slackApi.runSlackSmoke(options);
  }

  async createChannel(
    name: string,
    description: string,
    type: 'public' | 'dm' | 'custom',
    members: string[] = [],
    createdBy: string = ''
  ): Promise<Channel> {
    return this.channelsApi.createChannel(name, description, type, members, createdBy);
  }

  async openDM(agentId: string, createdBy: string): Promise<Channel> {
    return this.channelsApi.openDM(agentId, createdBy);
  }

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
    return this.channelsApi.createDMAgent(payload);
  }

  async fetchCliAgentTypes(): Promise<{ types: string[]; installed: Record<string, boolean> }> {
    return this.agentsExtrasApi.fetchCliAgentTypes();
  }

  async clearChannelHistory(name: string): Promise<void> {
    return this.channelsApi.clearChannelHistory(name);
  }

  async exportChannelHistory(channel: string, format: 'markdown' | 'json' = 'markdown'): Promise<Blob> {
    return this.channelsApi.exportChannelHistory(channel, format);
  }

  async getChannelDurable(channel: string): Promise<boolean> {
    return this.channelsApi.getChannelDurable(channel);
  }

  async setChannelDurable(channel: string, durable: boolean): Promise<void> {
    return this.channelsApi.setChannelDurable(channel, durable);
  }

  async deleteChannel(name: string): Promise<void> {
    return this.channelsApi.deleteChannel(name);
  }

  async archiveChannel(name: string): Promise<void> {
    return this.channelsApi.archiveChannel(name);
  }

  async addAgentsToChannel(channelName: string, agentIds: string[]): Promise<void> {
    return this.channelsApi.addAgentsToChannel(channelName, agentIds);
  }

  async removeAgentFromChannel(channelName: string, agentId: string): Promise<void> {
    return this.channelsApi.removeAgentFromChannel(channelName, agentId);
  }

  async testConnection(): Promise<boolean> {
    return this.channelsApi.testConnection();
  }

  getWebSocketURL(channel: string, extraChannels: string[] = []): string {
    return this.channelsApi.getWebSocketURL(channel, extraChannels);
  }

  getThreadWebSocketURL(channel: string, threadId: string): string {
    return this.channelsApi.getThreadWebSocketURL(channel, threadId);
  }

  async fetchThreadMessages(threadId: string, limit: number = 50): Promise<Message[]> {
    return this.messagesApi.fetchThreadMessages(threadId, limit);
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

  async fetchThreadMetadata(threadId: string): Promise<ThreadMetadata> {
    return this.messagesApi.fetchThreadMetadata(threadId);
  }

  async fetchMyAgents(): Promise<CachedAgentInfo[]> {
    return this.agentsApi.fetchMyAgents();
  }

  async deleteCachedAgent(payload: {
    type: string;
    name: string;
    path?: string;
  }): Promise<void> {
    return this.agentsExtrasApi.deleteCachedAgent(payload);
  }

  async fetchRemovedAgents(): Promise<AgentInfo[]> {
    return this.agentsApi.fetchRemovedAgents();
  }

  async removeAgent(
    channel: string,
    agentName: string,
    from: { name: string; type: string }
  ): Promise<void> {
    const command = `/remove-agent ${agentName}`;
    await this.sendMessage(channel, command, from, 'question');
  }

  async deleteAgent(
    channel: string,
    agentName: string,
    from: { name: string; type: string }
  ): Promise<void> {
    const command = `/delete-agent ${agentName}`;
    await this.sendMessage(channel, command, from, 'question');
  }

  async recallAgent(
    channel: string,
    agentName: string,
    from: { name: string; type: string }
  ): Promise<void> {
    const command = `/recall-agent ${agentName}`;
    await this.sendMessage(channel, command, from, 'question');
  }

  async exportAgent(channel: string, agentName: string): Promise<void> {
    await this.sendMessage(
      channel,
      `/export-agent-mcp ${agentName}`,
      { name: 'User', type: 'user' },
      'chat'
    );
  }

  async shareAgent(agentId: string): Promise<AgentShareBundle> {
    return this.agentsExtrasApi.shareAgent(agentId);
  }

  async importAgentBundle(options: {
    filePath: string;
    hydrate?: boolean;
    repositoryPath?: string;
  }): Promise<{ success: boolean; message: string; name?: string; lora_train_suggestion?: unknown }> {
    return this.agentsExtrasApi.importAgentBundle(options);
  }

  async testAnthropicConnection(apiKey: string, useAIHub: boolean = true, aiHubEndpoint?: string): Promise<ConnectionTestResult> {
    return this.providersApi.testAnthropicConnection(apiKey, useAIHub, aiHubEndpoint);
  }

  async testGitHubConnection(personalAccessToken: string): Promise<ConnectionTestResult> {
    return this.providersApi.testGitHubConnection(personalAccessToken);
  }

  async testConfluenceConnection(domain: string, email: string, apiToken: string): Promise<ConnectionTestResult> {
    return this.providersApi.testConfluenceConnection(domain, email, apiToken);
  }

  async testOllamaConnection(endpoint: string, model: string): Promise<ConnectionTestResult> {
    return this.providersApi.testOllamaConnection(endpoint, model);
  }

  async switchAgentProvider(agentId: string, provider: string, model: string): Promise<void> {
    return this.agentsExtrasApi.switchAgentProvider(agentId, provider, model);
  }

  async switchAllAgentProviders(provider: string, model: string): Promise<void> {
    return this.providersApi.switchAllAgentProviders(provider, model);
  }

  async fetchOllamaStatus(): Promise<{ running: boolean; endpoint: string; error?: string }> {
    return this.providersApi.fetchOllamaStatus();
  }

  async fetchOllamaModels(endpoint?: string): Promise<string[]> {
    return this.providersApi.fetchOllamaModels(endpoint);
  }

  async testLMStudioConnection(endpoint: string, model: string): Promise<ConnectionTestResult> {
    return this.providersApi.testLMStudioConnection(endpoint, model);
  }

  async fetchLMStudioStatus(): Promise<{ running: boolean; endpoint: string; error?: string }> {
    return this.providersApi.fetchLMStudioStatus();
  }

  async fetchLMStudioModels(endpoint?: string): Promise<string[]> {
    return this.providersApi.fetchLMStudioModels(endpoint);
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
    return this.providersApi.fetchHfCatalog();
  }

  async fetchHfStatus(): Promise<{
    token_configured: boolean;
    router_reachable: boolean;
    cache_dir?: string;
  }> {
    return this.providersApi.fetchHfStatus();
  }

  async fetchProviders(): Promise<
    { id: string; type: string; name: string; model?: string; endpoint?: string }[]
  > {
    return this.providersApi.fetchProviders();
  }

  async sendMessageWithCredentials(
    channel: string,
    content: string,
    from: { name: string; type: string },
    credentials?: Record<string, any>
  ): Promise<SendMessageResponse> {
    return this.sendMessage(channel, content, from, 'question', credentials);
  }

  static clearCredentials(credentials: Record<string, any>): void {
    ProvidersApi.clearCredentials(credentials);
  }

  async fetchWorkspaces(): Promise<any[]> {
    return this.workspaceApi.fetchWorkspaces();
  }

  async addWorkspace(
    name: string,
    path: string,
    options?: { create?: boolean; parentPath?: string },
  ): Promise<any> {
    return this.workspaceApi.addWorkspace(name, path, options);
  }

  async removeWorkspace(workspaceId: string): Promise<void> {
    return this.workspaceApi.removeWorkspace(workspaceId);
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
    return this.workspaceApi.connectRemoteWorkspace(params);
  }

  async fetchDevcontainerPlan(workspaceId: string): Promise<any> {
    return this.workspaceApi.fetchDevcontainerPlan(workspaceId);
  }

  async fetchDevcontainerPlanByPath(repoPath: string): Promise<any> {
    return this.workspaceApi.fetchDevcontainerPlanByPath(repoPath);
  }

  async pingSidecar(sidecarUrl: string, token?: string): Promise<boolean> {
    return this.workspaceApi.pingSidecar(sidecarUrl, token);
  }

  async fetchFiles(workspaceId: string, path: string = '/'): Promise<any[]> {
    return this.filesApi.fetchFiles(workspaceId, path);
  }

  async fetchFileContent(workspaceId: string, path: string): Promise<string> {
    return this.filesApi.fetchFileContent(workspaceId, path);
  }

  async fetchScanSummaryWellImage(
    workspaceId: string,
    summaryDir: string,
    well: string
  ): Promise<string> {
    return this.filesApi.fetchScanSummaryWellImage(workspaceId, summaryDir, well);
  }

  async fetchWorkspaceImageDataUrl(workspaceId: string, path: string): Promise<string> {
    return this.fetchWorkspaceBinaryDataUrl(workspaceId, path);
  }

  async fetchWorkspaceBinaryDataUrl(
    workspaceId: string,
    path: string,
    fallbackMime = 'application/octet-stream',
  ): Promise<string> {
    return this.filesApi.fetchWorkspaceBinaryDataUrl(workspaceId, path, fallbackMime);
  }

  async saveFileContent(workspaceId: string, path: string, content: string): Promise<void> {
    return this.filesApi.saveFileContent(workspaceId, path, content);
  }

  async renderCAD(body: {
    workspace: string;
    path: string;
    project_id?: string;
    params?: Record<string, string>;
    output_path?: string;
  }): Promise<{ content_base64: string; params?: unknown[]; scad_path?: string; stl_path?: string }> {
    return this.cadApi.renderCAD(body);
  }

  async fetchCADMesh(
    workspaceId: string,
    scadPath: string,
    projectId?: string
  ): Promise<{ content_base64: string }> {
    return this.cadApi.fetchCADMesh(workspaceId, scadPath, projectId);
  }

  async fetchCADParams(
    workspaceId: string,
    scadPath: string,
    projectId?: string
  ): Promise<{ params: CadParam[] }> {
    return this.cadApi.fetchCADParams(workspaceId, scadPath, projectId);
  }

  async fetchCADVersions(projectId: string): Promise<{ versions: Array<{ id: string; label: string; created_at: string }> }> {
    return this.cadApi.fetchCADVersions(projectId);
  }

  async saveCADVersion(body: {
    workspace: string;
    path: string;
    project_id: string;
    label: string;
    params?: Record<string, string>;
  }): Promise<unknown> {
    return this.cadApi.saveCADVersion(body);
  }

  async restoreCADVersion(
    projectId: string,
    versionId: string
  ): Promise<{ content?: string; scad_path?: string }> {
    return this.cadApi.restoreCADVersion(projectId, versionId);
  }

  async testOpenSCAD(path?: string): Promise<{ ok: boolean; message: string }> {
    return this.cadApi.testOpenSCAD(path);
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
    return this.cadApi.checkCADPrintability(body);
  }

  async validateCADAssembly(body: {
    manifest_path: string;
    clearance_mm?: number;
  }): Promise<{ ok?: boolean; bom?: Array<{ part_id: string; name: string }>; fit_issues?: unknown[] }> {
    return this.cadApi.validateCADAssembly(body);
  }

  async createFile(workspaceId: string, path: string, content: string = '', isDir = false): Promise<void> {
    return this.filesApi.createFile(workspaceId, path, content, isDir);
  }

  async renameFile(workspaceId: string, oldPath: string, newPath: string): Promise<void> {
    return this.filesApi.renameFile(workspaceId, oldPath, newPath);
  }

  async deleteFile(workspaceId: string, path: string): Promise<void> {
    return this.filesApi.deleteFile(workspaceId, path);
  }

  async getGitStatus(workspaceId: string): Promise<any> {
    return this.gitWorkspaceApi.getGitStatus(workspaceId);
  }

  async getGitDiff(workspaceId: string, path: string, staged = false): Promise<string> {
    return this.gitWorkspaceApi.getGitDiff(workspaceId, path, staged);
  }

  async getGitFileSides(
    workspaceId: string,
    path: string,
    staged: boolean
  ): Promise<{ original: string; modified: string }> {
    return this.gitWorkspaceApi.getGitFileSides(workspaceId, path, staged);
  }

  async gitAdd(workspaceId: string, paths: string[]): Promise<void> {
    return this.gitWorkspaceApi.gitAdd(workspaceId, paths);
  }

  async gitReset(workspaceId: string, paths: string[]): Promise<void> {
    return this.gitWorkspaceApi.gitReset(workspaceId, paths);
  }

  async commitChanges(workspaceId: string, message: string): Promise<void> {
    return this.gitWorkspaceApi.commitChanges(workspaceId, message);
  }

  async pushChanges(workspaceId: string): Promise<void> {
    return this.gitWorkspaceApi.pushChanges(workspaceId);
  }

  async pullChanges(workspaceId: string): Promise<void> {
    return this.gitWorkspaceApi.pullChanges(workspaceId);
  }

  async searchWorkspaceFiles(
    workspaceId: string,
    query: string,
    limit = 50
  ): Promise<string[]> {
    return this.ideApi.searchWorkspaceFiles(workspaceId, query, limit);
  }

  async searchWorkspaceSymbols(
    workspaceId: string,
    query: string,
    limit = 50
  ): Promise<
    Array<{ name: string; path: string; line: number; kind: string; language: string }>
  > {
    return this.ideApi.searchWorkspaceSymbols(workspaceId, query, limit);
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
    return this.ideApi.devFastEdit(params);
  }

  async getGoLSPDiagnostics(
    workspaceId: string
  ): Promise<
    Array<{ path: string; line: number; column: number; message: string; severity: string }>
  > {
    return this.ideApi.getGoLSPDiagnostics(workspaceId);
  }

  async getLSPDiagnostics(
    lang: 'rust' | 'python',
    workspaceId: string
  ): Promise<
    Array<{ path: string; line: number; column: number; message: string; severity: string }>
  > {
    return this.ideApi.getLSPDiagnostics(lang, workspaceId);
  }

  async devComplete(params: {
    prefix: string;
    suffix?: string;
    language?: string;
    path?: string;
    context?: string;
    model?: string;
    neighbor_snippets?: Array<{ path: string; content: string; source?: string }>;
    n?: number;
    signal?: AbortSignal;
  }): Promise<{ completion: string; completions?: string[]; model?: string }> {
    return this.ideApi.devComplete(params);
  }

  async devCompleteStream(
    params: {
      prefix: string;
      suffix?: string;
      language?: string;
      path?: string;
      context?: string;
      model?: string;
      neighbor_snippets?: Array<{ path: string; content: string; source?: string }>;
      n?: number;
      signal?: AbortSignal;
    },
    onChunk?: (text: string) => void
  ): Promise<{ completion: string; completions: string[]; model?: string; streamed: boolean }> {
    return this.ideApi.devCompleteStream(params, onChunk);
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
    return this.ideApi.devAgentTurn(params);
  }

  async repoSemanticSearch(params: {
    repoPath?: string;
    repoPaths?: string[];
    query: string;
    limit?: number;
  }): Promise<{
    chunks: Array<{ path: string; content: string; repo_path?: string; repo_name?: string }>;
  }> {
    return this.ideApi.repoSemanticSearch(params);
  }

  async repoIndexStatus(repoPath: string): Promise<{
    ready: boolean;
    building: boolean;
    chunk_count: number;
    embedding_model?: string;
  }> {
    return this.ideApi.repoIndexStatus(repoPath);
  }

  async repoGraph(repoPath: string): Promise<import('../components/knowledge-graph/types').KnowledgeGraphSummary> {
    return this.ideApi.repoGraph(repoPath);
  }

  async repoGraphSubgraph(
    repoPath: string,
    q: string,
    hops = 1,
    limit = 120,
  ): Promise<import('../components/knowledge-graph/types').KnowledgeGraphSummary & { query?: string; nodes: import('../components/knowledge-graph/types').KnowledgeGraphNode[]; edges: import('../components/knowledge-graph/types').KnowledgeGraphEdge[] }> {
    return this.ideApi.repoGraphSubgraph(repoPath, q, hops, limit);
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
    return this.ideApi.repoGraphPath(repoPath, from, to);
  }

  async repoGraphExplain(
    repoPath: string,
    node: string,
  ): Promise<import('../components/knowledge-graph/types').KnowledgeGraphExplain> {
    return this.ideApi.repoGraphExplain(repoPath, node);
  }

  async repoGraphStatus(
    repoPath: string,
    rebuild = false,
  ): Promise<import('../components/knowledge-graph/types').KnowledgeGraphMeta> {
    return this.ideApi.repoGraphStatus(repoPath, rebuild);
  }

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
    return this.toolApprovalsApi.fetchPendingToolApprovals();
  }

  async approveToolCall(approvalId: string, scope: 'once' | 'always' = 'once'): Promise<void> {
    return this.toolApprovalsApi.approveToolCall(approvalId, scope);
  }

  async rejectToolCall(approvalId: string, reason: string = 'User rejected'): Promise<void> {
    return this.toolApprovalsApi.rejectToolCall(approvalId, reason);
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
    return this.mapsApi.publishDeviceLocation(body);
  }

  async clearDeviceLocation(): Promise<void> {
    return this.mapsApi.clearDeviceLocation();
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
    return this.mapsApi.fetchPendingLocationRequests();
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
    return this.mapsApi.fulfillLocationRequest(requestId, body);
  }

  async rejectLocationRequest(requestId: string, reason?: string): Promise<void> {
    return this.mapsApi.rejectLocationRequest(requestId, reason);
  }

  async reverseGeocode(lat: number, lon: number): Promise<{ display_name?: string }> {
    return this.mapsApi.reverseGeocode(lat, lon);
  }

  async answerUserQuestion(questionId: string, answer: string): Promise<void> {
    return this.messagesApi.answerUserQuestion(questionId, answer);
  }

  async setAgentApprovalMode(agentId: string, mode: 'interactive' | 'auto_edit' | 'yolo'): Promise<void> {
    return this.plansApi.setAgentApprovalMode(agentId, mode);
  }

  async setAgentCustomRulesMarkdown(agentId: string, markdown: string): Promise<void> {
    return this.plansApi.setAgentCustomRulesMarkdown(agentId, markdown);
  }

  async setUserRulesMarkdown(markdown: string): Promise<void> {
    return this.plansApi.setUserRulesMarkdown(markdown);
  }

  async getUserRulesMarkdown(): Promise<string> {
    return this.plansApi.getUserRulesMarkdown();
  }

  async getPlan(id: string): Promise<{
    id: string;
    name: string;
    overview: string;
    todos: Array<{ id: string; content: string; status: string }>;
    markdown: string;
  }> {
    return this.plansApi.getPlan(id);
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
    return this.plansApi.putPlan(id, markdown);
  }

  async proposeFileChangeFromMessage(params: {
    channel: string;
    messageId: string;
    workspaceId: string;
    targetPath?: string;
    userId?: string;
  }): Promise<FileChange> {
    return this.fileChangesApi.proposeFileChangeFromMessage(params);
  }

  async listPendingFileChanges(userId: string = 'default'): Promise<FileChange[]> {
    return this.fileChangesApi.listPendingFileChanges(userId);
  }

  async approveFileChange(
    changeId: string,
    userId: string = 'default',
    newContent?: string
  ): Promise<FileChange> {
    return this.fileChangesApi.approveFileChange(changeId, userId, newContent);
  }

  async approveFileChangeRequest(requestId: string, userId: string = 'default'): Promise<FileChangeRequest> {
    return this.fileChangesApi.approveFileChangeRequest(requestId, userId);
  }

  async rejectFileChangeRequest(
    requestId: string,
    reason: string = 'No reason provided',
    userId: string = 'default',
  ): Promise<FileChangeRequest> {
    return this.fileChangesApi.rejectFileChangeRequest(requestId, reason, userId);
  }

  async updateFileChangeContent(changeId: string, newContent: string): Promise<FileChangeDiff> {
    return this.fileChangesApi.updateFileChangeContent(changeId, newContent);
  }

  async rejectFileChange(changeId: string, reason: string = 'No reason provided', userId: string = 'default'): Promise<FileChange> {
    return this.fileChangesApi.rejectFileChange(changeId, reason, userId);
  }

  async getFileDiff(changeId: string): Promise<FileChangeDiff> {
    return this.fileChangesApi.getFileDiff(changeId);
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
    return this.packExtrasApi.installPackFromZip(packZipBase64);
  }

  async installPackLoRAs(packId: string): Promise<InstallPackLoRAsResponse> {
    return this.packsApi.installPackLoRAs(packId);
  }

  async fetchACEStepStatus(packId = 'music-creation'): Promise<ACEStepStatus> {
    return this.packExtrasApi.fetchACEStepStatus(packId);
  }

  async installACEStep(
    packId = 'music-creation',
    modelVariant?: string,
  ): Promise<InstallACEStepResponse> {
    return this.packExtrasApi.installACEStep(packId, modelVariant);
  }

  async restartMusicSidecar(packId = 'music-creation'): Promise<{ status: string; acestep: ACEStepStatus }> {
    return this.packExtrasApi.restartMusicSidecar(packId);
  }

  async fetchArenaSidecarStatus(packId = 'model-arena'): Promise<ArenaSidecarStatus> {
    return this.packExtrasApi.fetchArenaSidecarStatus(packId);
  }

  async installArenaSidecarDeps(
    packId = 'model-arena',
  ): Promise<InstallArenaSidecarResponse> {
    return this.packExtrasApi.installArenaSidecarDeps(packId);
  }

  async restartArenaSidecar(
    packId = 'model-arena',
  ): Promise<{ status: string; sidecar: ArenaSidecarStatus }> {
    return this.packExtrasApi.restartArenaSidecar(packId);
  }

  async fetchAIInterviewProgress(): Promise<AIInterviewProgressResponse> {
    return this.packExtrasApi.fetchAIInterviewProgress();
  }

  async startAIInterviewDay(): Promise<AIInterviewProgressResponse> {
    return this.packExtrasApi.startAIInterviewDay();
  }

  async completeAIInterviewDay(
    day: number,
    body: { concept?: boolean; drill?: boolean; complete?: boolean; advance?: boolean },
  ): Promise<AIInterviewProgressResponse> {
    return this.packExtrasApi.completeAIInterviewDay(day, body);
  }

  async submitAIInterviewGate(
    gateId: string,
    body: { eval_notes?: string; mock_notes?: string; score?: number },
  ): Promise<AIInterviewProgressResponse> {
    return this.packExtrasApi.submitAIInterviewGate(gateId, body);
  }

  async unlockAIInterviewCert(): Promise<{
    certification: Record<string, unknown>;
    credential: Record<string, unknown>;
  }> {
    return this.packExtrasApi.unlockAIInterviewCert();
  }

  async fetchImageGenStatus(): Promise<ImageGenStatus> {
    return this.packExtrasApi.fetchImageGenStatus();
  }

  async uninstallPack(packId: string): Promise<PacksAPIResponse> {
    return this.packExtrasApi.uninstallPack(packId);
  }

  async setPackEnabled(packId: string, enabled: boolean): Promise<PacksAPIResponse> {
    return this.packExtrasApi.setPackEnabled(packId, enabled);
  }

  async setLayoutOwner(packId: string): Promise<PacksAPIResponse> {
    return this.packsApi.setLayoutOwner(packId);
  }

  async validatePack(body: {
    pack_zip_base64?: string;
    pack_dir?: string;
    pack_yaml?: string;
  }): Promise<PackValidationReport> {
    return this.packExtrasApi.validatePack(body);
  }

  async devLinkPack(packDir: string): Promise<PacksAPIResponse> {
    return this.packExtrasApi.devLinkPack(packDir);
  }

  async devReloadPack(packId: string): Promise<PacksAPIResponse> {
    return this.packExtrasApi.devReloadPack(packId);
  }

  async devUnlinkPack(packId: string): Promise<PacksAPIResponse> {
    return this.packExtrasApi.devUnlinkPack(packId);
  }

  async fetchCustomerPackContext(): Promise<CustomerPackContextResponse> {
    return this.packExtrasApi.fetchCustomerPackContext();
  }

  async fetchPhoenixStatus(): Promise<{
    environment: string;
    credentials_path?: string;
    authenticated: boolean;
    logged_in: boolean;
    identity?: string;
    hint?: string;
  }> {
    return this.phoenixApi.fetchPhoenixStatus();
  }

  async fetchPhoenixAnalyses(): Promise<Array<{ id: string; label: string }>> {
    return this.phoenixApi.fetchPhoenixAnalyses();
  }

  async fetchPhoenixScanResults(): Promise<Array<{ id: string; label: string }>> {
    return this.phoenixApi.fetchPhoenixScanResults();
  }

  async phoenixLoginStart(): Promise<{
    session_id: string;
    user_code: string;
    verification_url: string;
    expires_in: number;
    environment: string;
  }> {
    return this.phoenixApi.phoenixLoginStart();
  }

  async phoenixLoginPoll(sessionId: string): Promise<{
    status: string;
    identity?: string;
    hint?: string;
    expires_in?: number;
  }> {
    return this.phoenixApi.phoenixLoginPoll(sessionId);
  }

  async phoenixLogout(): Promise<void> {
    return this.phoenixApi.phoenixLogout();
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
    return this.phoenixApi.phoenixImport(body);
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
    return this.phoenixApi.phoenixImportScan(body);
  }

  async fetchExpertPresets(): Promise<ExpertPresetOption[]> {
    return this.packExtrasApi.fetchExpertPresets();
  }

  async restartConfiguredAgents(): Promise<void> {
    return this.agentsExtrasApi.restartConfiguredAgents();
  }

  async fetchLoraExpertContext(agentId: string): Promise<LoraExpertContext> {
    return this.loraApi.fetchLoraExpertContext(agentId);
  }

  async fetchLoraTrainBases(): Promise<LoraTrainingBase[]> {
    return this.loraApi.fetchLoraTrainBases();
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
    return this.loraApi.previewLoraTrain(params);
  }

  async previewLoraTrainDataset(
    body: Omit<LoraTrainStartRequest, 'base_ollama_tag' | 'ollama_tag' | 'hyperparams'> & {
      base_ollama_tag?: string;
      ollama_tag?: string;
    },
  ): Promise<LoraTrainDatasetPreview> {
    return this.loraApi.previewLoraTrainDataset(body);
  }

  async bootstrapLoraTrainFromIndex(agentId: string): Promise<LoraTrainDatasetPreview> {
    return this.loraApi.bootstrapLoraTrainFromIndex(agentId);
  }

  async startLoraTrain(body: LoraTrainStartRequest): Promise<LoraTrainJob> {
    return this.loraApi.startLoraTrain(body);
  }

  async fetchLoraTrainJob(jobId: string): Promise<LoraTrainJob> {
    return this.loraApi.fetchLoraTrainJob(jobId);
  }

  async cancelLoraTrainJob(jobId: string): Promise<LoraTrainJob> {
    return this.loraApi.cancelLoraTrainJob(jobId);
  }

  async run12PlexQC(body: {
    workspace_id: string;
    analysis_dir: string;
    write_report?: boolean;
  }): Promise<import('../utils/secondaryAnalysis').PanelQCReport> {
    return this.secondaryAnalysisApi.run12PlexQC(body);
  }

  async runSecondaryAnalysis(body: {
    workflow: string;
    workspace_id: string;
    config?: Record<string, unknown>;
  }): Promise<import('../utils/secondaryAnalysis').SecondaryAnalysisJob> {
    return this.secondaryAnalysisApi.runSecondaryAnalysis(body);
  }

  async fetchSecondaryAnalysisJob(
    jobId: string
  ): Promise<import('../utils/secondaryAnalysis').SecondaryAnalysisJob> {
    return this.secondaryAnalysisApi.fetchSecondaryAnalysisJob(jobId);
  }

  async cancelSecondaryAnalysisJob(
    jobId: string
  ): Promise<import('../utils/secondaryAnalysis').SecondaryAnalysisJob> {
    return this.secondaryAnalysisApi.cancelSecondaryAnalysisJob(jobId);
  }

  async fetchComparatorSummary(
    workspaceId: string,
    dir: string
  ): Promise<import('../utils/secondaryAnalysis').ComparatorSummary> {
    return this.secondaryAnalysisApi.fetchComparatorSummary(workspaceId, dir);
  }

  async fetchLearnings(options?: {
    agentId?: string;
    agentType?: string;
    agentName?: string;
  }): Promise<UserLearning[]> {
    return this.learningsApi.fetchLearnings(options);
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
    return this.learningsApi.createLearning(body);
  }

  async deleteLearning(id: string): Promise<void> {
    return this.learningsApi.deleteLearning(id);
  }

  async fetchLearningStats(agentId: string): Promise<LearningStats> {
    return this.learningsApi.fetchLearningStats(agentId);
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
    return this.learningsApi.updateLearning(id, body);
  }

  async queryLearnings(params: {
    q?: string;
    agent_id?: string;
    scope?: LearningScope;
    channel?: string;
    collaboration_id?: string;
  }): Promise<{ query: string; count: number; results: UserLearning[] }> {
    return this.learningsApi.queryLearnings(params);
  }

  async exportLearnings(): Promise<{ version: number; user_id: string; entries: UserLearning[] }> {
    return this.learningsApi.exportLearnings();
  }

  async importLearnings(bundle: { entries: UserLearning[] }): Promise<{ added: number; skipped: number }> {
    return this.learningsApi.importLearnings(bundle);
  }

}
