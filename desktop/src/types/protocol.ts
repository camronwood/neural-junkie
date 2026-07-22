// TypeScript types matching Go protocol types

export type MessageType =
  | 'chat'
  | 'question'
  | 'answer'
  | 'system_info'
  | 'agent_join'
  | 'agent_leave'
  | 'agent_status'
  | 'context_share'
  | 'request_help'
  | 'command_output'
  | 'command_suggestion'
  | 'design_output'
  | 'file_change'
  | 'tool_approval'
  | 'user_question'
  | 'stream_delta'
  | 'stream_end'
  | 'collaboration_plan'
  | 'collaboration_task'
  | 'collaboration_status'
  | 'collaboration_discussion'
  | 'collaboration_recap'
  | 'artifact_changed';

export type AgentType =
  | 'frontend'
  | 'backend'
  | 'devops'
  | 'database'
  | 'security'
  | 'rust'
  | 'architecture'
  | 'code-review'
  | 'biology'
  | 'cad'
  | 'general'
  | 'repo'
  | 'expert'
  | 'confluence'
  | 'moderator'
  | 'assistant'
  | 'cli'
  | 'loading'
  | 'human';

export type AIProviderType = 'claude' | 'ollama' | 'lmstudio';

export type IndexingStatus =
  | 'indexing'
  | 'ready'
  | 'reindexing'
  | 'error';

export interface AgentInfo {
  id: string;
  name: string;
  type: AgentType;
  expertise: string[];
  status: string; // "active", "busy", "idle", "paused", "removed"
  model: string;
  ai_provider?: string; // AI provider being used ("claude", "ollama")
  ai_model?: string; // Specific model name (e.g., "claude-sonnet", "llama3.1")
  is_paused: boolean;
  supports_vision?: boolean; // Whether the agent can process images
  supports_image_generation?: boolean;
  supports_music_generation?: boolean;
  indexing_status?: string;
  index_progress?: number;
  repository_path?: string;
  knowledge_path?: string;
  confluence_space_key?: string;
  last_active_time?: string;
  removed_from?: string[];
  approval_mode?: 'interactive' | 'auto_edit' | 'yolo';
  /** User-defined markdown instructions merged into this agent's system prompt (server-persisted). */
  custom_rules_markdown?: string;
  /** Hub MCP + built-in tool count (when fetched with include_tool_counts). */
  tool_count?: number;
  /** Internal consult-only repo index; hidden from user-facing agent lists. */
  consult_only?: boolean;
  capabilities?: string[];
  denied_capabilities?: string[];
}

export interface AgentToolParam {
  name: string;
  required: boolean;
  description?: string;
}

export interface AgentToolDefinition {
  name: string;
  description: string;
  parameters?: AgentToolParam[];
  source: 'mcp' | 'builtin';
}

export interface AgentToolCapabilities {
  agent_id: string;
  agent_name: string;
  agent_type: string;
  tools: AgentToolDefinition[];
  tool_count: number;
  mcp_enabled: boolean;
  mcp_port?: number;
  chat_model: string;
  chat_provider: string;
  chat_native_tools: boolean;
  tool_loop_model: string;
  tool_loop_uses_fallback: boolean;
  tool_loop_mode?: 'native' | 'react' | 'fallback';
  notes?: string[];
  discoverable_capabilities?: string[];
  available_capabilities?: string[];
  active_capabilities?: string[];
  denied_capabilities?: string[];
  unavailable_capabilities?: string[];
}

export interface ChannelToolsResponse {
  channel: string;
  agents: AgentToolCapabilities[];
}

export interface ResolvedCapability {
  id: string;
  qualified_id: string;
  pack_id?: string;
  label?: string;
  description?: string;
  exposure?: 'safe' | 'sensitive' | string;
  kind?: string;
  platform?: boolean;
  routes?: string[];
  ui?: {
    toolbar?: { id?: string; label?: string; icon?: string };
    modal?: string;
  };
  match_glob?: string;
  viewer?: string;
  settings?: string[];
  mcp_tools?: string[];
  mcp_agents?: string[];
  renderer?: string;
  media_types?: string[];
  renderer_api_version?: number;
  schema_version_min?: number;
  schema_version_max?: number;
  fallback?: string;
}

export interface AgentCapabilityState {
  discoverable: ResolvedCapability[];
  available: string[];
  effective: string[];
  denied: string[];
  unavailable: string[];
  allow: string[];
  deny: string[];
}

export interface CapabilityPolicyAgent {
  agent: AgentInfo;
  state: AgentCapabilityState;
}

export interface CapabilityPolicyResponse {
  allow_sensitive_by_default: boolean;
  handoffs_enabled: boolean;
  capability_registry: ResolvedCapability[];
  agents: CapabilityPolicyAgent[];
}

export interface CapabilityPolicyUpdate {
  allow_sensitive_by_default?: boolean;
  handoffs_enabled?: boolean;
  agent_key?: string;
  override?: {
    allow: string[];
    deny: string[];
  };
}

export interface ImplementationSessionOutcome {
  outcome?: string;
  failure_type?: string;
  verify_failed?: boolean;
  verify_skipped?: boolean;
  repair_used?: boolean;
  repair_attempts?: number;
  circuit_breaker_triggered?: boolean;
  playbook_used?: string;
  suggested_agent?: string;
  routing_reason?: string;
  routing_tool_model?: string;
  repro_command?: string;
  files_changed?: string[];
  command_failures?: Array<{ cmd: string; count: number }>;
}

export interface Message {
  id: string;
  type: MessageType;
  channel: string;
  from: AgentInfo;
  content: string;
  timestamp: string; // ISO date string
  reply_to?: string;
  thread_id?: string; // ID of the thread this message belongs to
  is_thread_reply?: boolean; // Whether this is a reply in a thread
  metadata?: Record<string, any>;
  tags?: string[];
  mentions?: string[];
}

export type ChangeProposalKind = 'file_change' | 'git_change';
export type ChangeProposalStatus =
  | 'pending'
  | 'applying'
  | 'approved'
  | 'rejected'
  | 'stale'
  | 'expired'
  | 'failed';

export interface ChangeProposalCard {
  version: number;
  kind: ChangeProposalKind;
  id: string;
  status: ChangeProposalStatus;
  operation: string;
  file_path?: string;
  old_path?: string;
  new_path?: string;
  message?: string;
  paths?: string[];
  workspace_id?: string;
  requested_at?: string;
  expires_at?: string;
  reason?: string;
  error?: string;
}

export interface GitChangeProposal {
  id: string;
  operation: string;
  message?: string;
  paths?: string[];
  workspace_id?: string;
  channel?: string;
  status: ChangeProposalStatus;
  requested_at?: string;
  expires_at?: string;
  reason?: string;
  error?: string;
}

export function getChangeProposalCard(message: Message): ChangeProposalCard | null {
  const raw = message.metadata?.change_proposal;
  if (raw && typeof raw === 'object') {
    const card = raw as Partial<ChangeProposalCard>;
    if (
      typeof card.id === 'string' &&
      (card.kind === 'file_change' || card.kind === 'git_change')
    ) {
      return {
        version: typeof card.version === 'number' ? card.version : 1,
        kind: card.kind,
        id: card.id,
        status: card.status ?? 'pending',
        operation: typeof card.operation === 'string' ? card.operation : '',
        file_path: card.file_path,
        old_path: card.old_path,
        new_path: card.new_path,
        message: card.message,
        paths: Array.isArray(card.paths)
          ? card.paths.filter((path): path is string => typeof path === 'string')
          : undefined,
        workspace_id: card.workspace_id,
        requested_at: card.requested_at,
        expires_at: card.expires_at,
        reason: card.reason,
        error: card.error,
      };
    }
  }

  // Compatibility with proposal messages persisted before the typed card contract.
  if (message.type === 'file_change') {
    const proposal = message.metadata?.file_change_proposal as Record<string, unknown> | undefined;
    const id = message.metadata?.registered_change_id;
    if (proposal && typeof id === 'string') {
      return {
        version: 1,
        kind: 'file_change',
        id,
        status: message.metadata?.file_change_auto_approved === true ? 'approved' : 'pending',
        operation: typeof proposal.operation === 'string' ? proposal.operation : '',
        file_path: typeof proposal.file_path === 'string' ? proposal.file_path : undefined,
        old_path: typeof proposal.old_path === 'string' ? proposal.old_path : undefined,
        new_path: typeof proposal.new_path === 'string' ? proposal.new_path : undefined,
      };
    }
  }

  const git = message.metadata?.git_change_proposal as Record<string, unknown> | undefined;
  if (git && typeof git.id === 'string') {
    return {
      version: 1,
      kind: 'git_change',
      id: git.id,
      status: 'pending',
      operation: typeof git.operation === 'string' ? git.operation : '',
      message: typeof git.message === 'string' ? git.message : undefined,
      paths: Array.isArray(git.paths)
        ? git.paths.filter((path): path is string => typeof path === 'string')
        : undefined,
      workspace_id: typeof git.workspace_id === 'string' ? git.workspace_id : undefined,
    };
  }
  return null;
}

export interface ArtifactReference {
  id: string;
  title?: string;
  renderer_id?: string;
  renderer_api_version?: number;
  media_type?: string;
  revision?: number;
  workspace_id?: string;
  action?: 'created' | 'updated' | 'deleted' | string;
}

export interface StoredArtifactSource {
  kind: string;
  uri?: string;
  artifactId?: string;
  revision?: number;
  label?: string;
  metadata?: Record<string, string>;
}

export interface StoredArtifact {
  schemaVersion: number;
  id: string;
  revision: number;
  kind?: string;
  title?: string;
  description?: string;
  provenance?: StoredArtifactSource[];
  links?: {
    workspaceId?: string;
    projectId?: string;
    channelId?: string;
    collaborationId?: string;
  };
  renderer: {
    id: string;
    apiVersion: string;
    mediaType: string;
  };
  payload: unknown;
  fallback?: {
    mediaType: string;
    data: unknown;
  };
  capabilities?: string[];
  createdAt: string;
  updatedAt: string;
}

export interface StoredArtifactRevision {
  artifactId: string;
  revision: number;
  createdAt: string;
  artifact: StoredArtifact;
}

export function getArtifactReference(
  metadata?: Record<string, unknown>,
): ArtifactReference | null {
  const raw = metadata?.artifact_ref;
  if (!raw || typeof raw !== 'object') return null;
  const ref = raw as Record<string, unknown>;
  if (typeof ref.id !== 'string' || !ref.id.trim()) return null;
  return ref as unknown as ArtifactReference;
}

export interface MessageErrorMetadata {
  error_code?: 'timeout' | 'rate_limit' | 'workspace_trust' | 'provider_unavailable' | 'provider_error' | 'unknown';
  retryable?: boolean;
}

/** Persisted model reasoning (Ollama thinking / DeepSeek R1). */
export const REASONING_TEXT_METADATA_KEY = 'reasoning_text';
/** Stream delta carries a reasoning chunk when true. */
export const REASONING_DELTA_METADATA_KEY = 'reasoning_delta';
/** Per-delta reasoning append payload. */
export const REASONING_APPEND_METADATA_KEY = 'reasoning_append';

export function getReasoningText(metadata?: Record<string, unknown>): string {
  const v = metadata?.[REASONING_TEXT_METADATA_KEY];
  return typeof v === 'string' ? v : '';
}

export function isReasoningStreamDelta(metadata?: Record<string, unknown>): boolean {
  return metadata?.[REASONING_DELTA_METADATA_KEY] === true;
}

export const TOOL_STEPS_METADATA_KEY = 'tool_steps';

export type ToolStepMeta = {
  kind: string;
  name: string;
  iteration?: number;
  preview?: string;
};

export function getToolSteps(metadata?: Record<string, unknown>): ToolStepMeta[] {
  const raw = metadata?.[TOOL_STEPS_METADATA_KEY];
  if (!Array.isArray(raw)) return [];
  return raw.filter((x) => x && typeof x === 'object') as ToolStepMeta[];
}

export const ROUTING_MODEL_METADATA_KEY = 'routing_model';
export const ROUTING_TOOL_MODEL_METADATA_KEY = 'routing_tool_model';
export const ROUTING_REASON_METADATA_KEY = 'routing_reason';
export const ROUTING_SOURCE_METADATA_KEY = 'routing_source';
export const ROUTING_DOMAIN_METADATA_KEY = 'routing_domain';
export const ROUTING_COST_TIER_METADATA_KEY = 'routing_cost_tier';
export const ROUTING_PROVIDER_ID_METADATA_KEY = 'routing_provider_id';
export const ROUTING_KNOWLEDGE_ROUTE_METADATA_KEY = 'routing_knowledge_route';
export const ROUTING_KNOWLEDGE_REASON_METADATA_KEY = 'routing_knowledge_reason';
export const ROUTING_KNOWLEDGE_TARGETS_METADATA_KEY = 'routing_knowledge_targets';
export const ROUTING_KNOWLEDGE_EXECUTED_METADATA_KEY = 'routing_knowledge_executed';
export const ROUTING_COMPOSER_MODE_METADATA_KEY = 'routing_composer_mode';
export const ROUTING_CONTEXT_SCOPE_METADATA_KEY = 'routing_context_scope';
export const ROUTING_IMPL_SESSION_METADATA_KEY = 'routing_impl_session';
export const ROUTING_CONVERSATION_TIER_METADATA_KEY = 'routing_conversation_tier';
export const ROUTING_CONVERSATION_REASONS_METADATA_KEY = 'routing_conversation_reasons';
export const ROUTING_CONVERSATION_ESCALATED_FROM_METADATA_KEY = 'routing_conversation_escalated_from';
export const ROUTING_ATTEMPTS_METADATA_KEY = 'routing_attempts';
export const ROUTING_FAILURE_EVIDENCE_METADATA_KEY = 'routing_failure_evidence';

export type RoutingMeta = {
  provider_id?: string;
  model?: string;
  tool_model?: string;
  reason?: string;
  source?: string;
  domain?: string;
  cost_tier?: string;
  knowledge_route?: string;
  knowledge_reason?: string;
  knowledge_targets?: string[];
  knowledge_executed?: string[];
  composer_mode?: string;
  context_scope?: string;
  impl_session?: boolean;
  conversation_tier?: string;
  conversation_reasons?: string[];
  conversation_escalated_from?: string;
  attempts?: RoutingAttempt[];
  failure_evidence?: string[];
};

export type RoutingGovernanceMeta = {
  composer_mode?: string;
  context_scope?: string;
  impl_session?: boolean;
  can_propose_files?: boolean;
  can_run_impl_session?: boolean;
};

export type RoutingTelemetryPayload = {
  chat_model?: string;
  tool_model?: string;
  provider_id?: string;
  reason?: string;
  source?: string;
  domain?: string;
  cost_tier?: string;
  knowledge_route?: string;
  knowledge_reason?: string;
  knowledge_targets?: string[];
  knowledge_executed?: string[];
  conversation_tier?: string;
  conversation_reasons?: string[];
  conversation_escalated_from?: string;
  governance?: RoutingGovernanceMeta;
};

export type TurnTraceRouting = {
  model?: string;
  tool_model?: string;
  provider_id?: string;
  domain?: string;
  cost_tier?: string;
  reason?: string;
  source?: string;
  attempts?: RoutingAttempt[];
  failure_evidence?: string[];
};

export type RoutingAttempt = {
  provider_id?: string;
  model?: string;
  tier?: string;
  reason?: string;
  failure_reason?: string;
};

export type TurnTraceRetrieval = {
  mode?: string;
  reason?: string;
  targets?: string[];
  executed?: string[];
  memory_count?: number;
  codebase_count?: number;
};

export type TurnTraceSpan = {
  id?: string;
  parent_id?: string;
  name?: string;
  start_ms?: number;
  end_ms?: number;
  status?: string;
  attrs?: Record<string, unknown>;
};

export type TurnTraceContextSelection = {
  selected_context_ids?: string[];
  selected_sections?: string[];
  dropped_context_ids?: string[];
  provenance?: Array<{
    id?: string;
    section?: string;
    source?: string;
    score?: number;
    freshness?: string;
  }>;
  digest_version?: number;
  section_sizes?: Record<string, { items?: number; bytes?: number }>;
  section_budgets?: Record<string, number>;
  compression?: {
    summary_checkpoint?: boolean;
    applied?: boolean;
    original_bytes?: number;
    final_bytes?: number;
    compressed_sections?: string[];
    recoverable?: boolean;
  };
  recovery?: {
    active?: boolean;
    correction_count?: number;
    superseded_count?: number;
    unresolved_actions?: number;
  };
  omission_reasons?: Record<string, string>;
  budget_omission_reasons?: Record<string, string>;
  retrieval_counts?: {
    memory?: number;
    codebase?: number;
  };
};

export type TurnTraceResponse = {
  message_id?: string;
  channel?: string;
  query?: string;
  reply_message_id?: string;
  trace_id?: string;
  routing?: TurnTraceRouting & {
    classifier?: {
      intent?: string;
      tool_need?: boolean;
      confidence?: number;
      lora_tag?: string;
    };
  };
  retrieval?: TurnTraceRetrieval;
  context_selection?: TurnTraceContextSelection;
  governance?: RoutingGovernanceMeta;
  tool_steps?: unknown;
  inference_usage?: Record<string, unknown>;
  spans?: TurnTraceSpan[];
  reasoning_text?: string;
  compress?: {
    strategy?: string;
    bytes_in?: number;
    bytes_out?: number;
  };
};

export function getRoutingMeta(metadata?: Record<string, unknown>): RoutingMeta | null {
  if (!metadata) return null;
  const model = metadata[ROUTING_MODEL_METADATA_KEY];
  const reason = metadata[ROUTING_REASON_METADATA_KEY];
  if (typeof model !== 'string' && typeof reason !== 'string') return null;
  const out: RoutingMeta = {};
  if (typeof metadata[ROUTING_PROVIDER_ID_METADATA_KEY] === 'string') {
    out.provider_id = metadata[ROUTING_PROVIDER_ID_METADATA_KEY] as string;
  }
  if (typeof model === 'string') out.model = model;
  if (typeof metadata[ROUTING_TOOL_MODEL_METADATA_KEY] === 'string') {
    out.tool_model = metadata[ROUTING_TOOL_MODEL_METADATA_KEY] as string;
  }
  if (typeof reason === 'string') out.reason = reason;
  if (typeof metadata[ROUTING_SOURCE_METADATA_KEY] === 'string') {
    out.source = metadata[ROUTING_SOURCE_METADATA_KEY] as string;
  }
  if (typeof metadata[ROUTING_DOMAIN_METADATA_KEY] === 'string') {
    out.domain = metadata[ROUTING_DOMAIN_METADATA_KEY] as string;
  }
  if (typeof metadata[ROUTING_COST_TIER_METADATA_KEY] === 'string') {
    out.cost_tier = metadata[ROUTING_COST_TIER_METADATA_KEY] as string;
  }
  if (typeof metadata[ROUTING_KNOWLEDGE_ROUTE_METADATA_KEY] === 'string') {
    out.knowledge_route = metadata[ROUTING_KNOWLEDGE_ROUTE_METADATA_KEY] as string;
  }
  if (typeof metadata[ROUTING_KNOWLEDGE_REASON_METADATA_KEY] === 'string') {
    out.knowledge_reason = metadata[ROUTING_KNOWLEDGE_REASON_METADATA_KEY] as string;
  }
  if (Array.isArray(metadata[ROUTING_KNOWLEDGE_TARGETS_METADATA_KEY])) {
    out.knowledge_targets = (metadata[ROUTING_KNOWLEDGE_TARGETS_METADATA_KEY] as unknown[]).filter(
      (v): v is string => typeof v === 'string'
    );
  }
  if (Array.isArray(metadata[ROUTING_KNOWLEDGE_EXECUTED_METADATA_KEY])) {
    out.knowledge_executed = (metadata[ROUTING_KNOWLEDGE_EXECUTED_METADATA_KEY] as unknown[]).filter(
      (v): v is string => typeof v === 'string'
    );
  }
  if (typeof metadata[ROUTING_COMPOSER_MODE_METADATA_KEY] === 'string') {
    out.composer_mode = metadata[ROUTING_COMPOSER_MODE_METADATA_KEY] as string;
  }
  if (typeof metadata[ROUTING_CONTEXT_SCOPE_METADATA_KEY] === 'string') {
    out.context_scope = metadata[ROUTING_CONTEXT_SCOPE_METADATA_KEY] as string;
  }
  if (metadata[ROUTING_IMPL_SESSION_METADATA_KEY] === true) {
    out.impl_session = true;
  }
  if (typeof metadata[ROUTING_CONVERSATION_TIER_METADATA_KEY] === 'string') {
    out.conversation_tier = metadata[ROUTING_CONVERSATION_TIER_METADATA_KEY] as string;
  }
  if (Array.isArray(metadata[ROUTING_CONVERSATION_REASONS_METADATA_KEY])) {
    out.conversation_reasons = (metadata[ROUTING_CONVERSATION_REASONS_METADATA_KEY] as unknown[]).filter(
      (v): v is string => typeof v === 'string'
    );
  }
  if (typeof metadata[ROUTING_CONVERSATION_ESCALATED_FROM_METADATA_KEY] === 'string') {
    out.conversation_escalated_from = metadata[ROUTING_CONVERSATION_ESCALATED_FROM_METADATA_KEY] as string;
  }
  if (Array.isArray(metadata[ROUTING_ATTEMPTS_METADATA_KEY])) {
    out.attempts = (metadata[ROUTING_ATTEMPTS_METADATA_KEY] as unknown[]).filter(
      (value): value is RoutingAttempt => !!value && typeof value === 'object' && !Array.isArray(value)
    );
  }
  if (Array.isArray(metadata[ROUTING_FAILURE_EVIDENCE_METADATA_KEY])) {
    out.failure_evidence = (metadata[ROUTING_FAILURE_EVIDENCE_METADATA_KEY] as unknown[]).filter(
      (value): value is string => typeof value === 'string'
    );
  }
  return out;
}

export function parseRoutingTelemetryPayload(raw: unknown): RoutingTelemetryPayload | null {
  if (!raw || typeof raw !== 'object' || Array.isArray(raw)) return null;
  const p = raw as Record<string, unknown>;
  const out: RoutingTelemetryPayload = {};
  for (const key of [
    'chat_model',
    'tool_model',
    'provider_id',
    'reason',
    'source',
    'domain',
    'cost_tier',
    'knowledge_route',
    'knowledge_reason',
    'conversation_tier',
    'conversation_escalated_from',
  ] as const) {
    if (typeof p[key] === 'string') out[key] = p[key] as string;
  }
  if (Array.isArray(p.knowledge_targets)) {
    out.knowledge_targets = p.knowledge_targets.filter((v): v is string => typeof v === 'string');
  }
  if (Array.isArray(p.knowledge_executed)) {
    out.knowledge_executed = p.knowledge_executed.filter((v): v is string => typeof v === 'string');
  }
  if (Array.isArray(p.conversation_reasons)) {
    out.conversation_reasons = p.conversation_reasons.filter((v): v is string => typeof v === 'string');
  }
  if (p.governance && typeof p.governance === 'object' && !Array.isArray(p.governance)) {
    const g = p.governance as Record<string, unknown>;
    out.governance = {};
    if (typeof g.composer_mode === 'string') out.governance.composer_mode = g.composer_mode;
    if (typeof g.context_scope === 'string') out.governance.context_scope = g.context_scope;
    if (g.can_propose_files === true) out.governance.can_propose_files = true;
    if (g.can_run_impl_session === true) out.governance.can_run_impl_session = true;
  }
  return out;
}

export function formatRoutingBadgeLabel(meta: RoutingMeta): string {
  const model = meta.model?.trim();
  if (!model) return '';
  const retrieval = meta.knowledge_targets?.length
    ? meta.knowledge_targets.join('+')
    : meta.knowledge_route?.trim();
  if (retrieval && retrieval !== 'general' && retrieval !== 'conversation_memory') {
    return `${model} · ${retrieval}`;
  }
  const source = meta.source?.trim();
  return source ? `${model} · ${source}` : model;
}

export function formatRoutingTooltip(meta: RoutingMeta): string {
  const lines: string[] = [];
  if (meta.model) lines.push(`Chat model: ${meta.model}`);
  if (meta.tool_model) lines.push(`Tool model: ${meta.tool_model}`);
  if (meta.provider_id) lines.push(`Provider: ${meta.provider_id}`);
  if (meta.domain) lines.push(`Domain: ${meta.domain}`);
  if (meta.cost_tier) lines.push(`Cost tier: ${meta.cost_tier}`);
  if (meta.knowledge_targets?.length) lines.push(`Retrieval targets: ${meta.knowledge_targets.join(', ')}`);
  if (meta.knowledge_executed?.length) lines.push(`Retrieval executed: ${meta.knowledge_executed.join(', ')}`);
  if (meta.knowledge_route) lines.push(`Retrieval: ${meta.knowledge_route}`);
  if (meta.knowledge_reason) lines.push(`Retrieval reason: ${meta.knowledge_reason}`);
  if (meta.composer_mode) lines.push(`Composer mode: ${meta.composer_mode}`);
  if (meta.context_scope) lines.push(`Context scope: ${meta.context_scope}`);
  if (meta.impl_session) lines.push('Implementation session: yes');
  if (meta.conversation_tier) lines.push(`Conversation tier: ${meta.conversation_tier}`);
  if (meta.conversation_reasons?.length) lines.push(`Conversation signals: ${meta.conversation_reasons.join(', ')}`);
  if (meta.conversation_escalated_from) lines.push(`Escalated from: ${meta.conversation_escalated_from}`);
  if (meta.reason) lines.push(`Reason: ${meta.reason}`);
  if (meta.source) lines.push(`Classifier: ${meta.source}`);
  return lines.join('\n');
}

export const COMPRESS_BYTES_IN_METADATA_KEY = 'context_compress_bytes_in';
export const COMPRESS_BYTES_OUT_METADATA_KEY = 'context_compress_bytes_out';
export const COMPRESS_STRATEGY_METADATA_KEY = 'context_compress_strategy';
export const COMPRESS_REFS_METADATA_KEY = 'context_compress_refs';

export type CompressMeta = {
  bytes_in?: number;
  bytes_out?: number;
  strategy?: string;
  refs?: string;
};

export function getCompressMeta(metadata?: Record<string, unknown>): CompressMeta | null {
  if (!metadata) return null;
  const strategy = metadata[COMPRESS_STRATEGY_METADATA_KEY];
  const bytesIn = metadata[COMPRESS_BYTES_IN_METADATA_KEY];
  const bytesOut = metadata[COMPRESS_BYTES_OUT_METADATA_KEY];
  if (typeof strategy !== 'string' && bytesIn == null && bytesOut == null) return null;
  const out: CompressMeta = {};
  if (typeof bytesIn === 'number') out.bytes_in = bytesIn;
  if (typeof bytesOut === 'number') out.bytes_out = bytesOut;
  if (typeof strategy === 'string') out.strategy = strategy;
  if (typeof metadata[COMPRESS_REFS_METADATA_KEY] === 'string') {
    out.refs = metadata[COMPRESS_REFS_METADATA_KEY] as string;
  }
  return out;
}

function formatBytes(n: number): string {
  if (n >= 1024) return `${(n / 1024).toFixed(1)}KB`;
  return `${n}B`;
}

export function formatCompressBadgeLabel(meta: CompressMeta): string {
  const strategy = meta.strategy?.trim();
  if (!strategy || strategy === 'none') return '';
  const inB = meta.bytes_in ?? 0;
  const outB = meta.bytes_out ?? 0;
  if (inB > 0 && outB > 0 && inB > outB) {
    return `${strategy} · ${formatBytes(inB)}→${formatBytes(outB)}`;
  }
  return strategy;
}

export function formatCompressTooltip(meta: CompressMeta): string {
  const lines: string[] = [];
  if (meta.strategy) lines.push(`Strategy: ${meta.strategy}`);
  if (meta.bytes_in != null) lines.push(`Bytes in: ${meta.bytes_in}`);
  if (meta.bytes_out != null) lines.push(`Bytes out: ${meta.bytes_out}`);
  if (meta.refs) lines.push(`Refs: ${meta.refs}`);
  lines.push('Use nj_retrieve_context to expand compressed tool output.');
  return lines.join('\n');
}

export function isToolStepStreamDelta(metadata?: Record<string, unknown>): boolean {
  return typeof metadata?.tool_step === 'string';
}

export type ChannelType = 'public' | 'dm' | 'custom' | 'collaboration' | 'room' | 'delegation';

export interface Channel {
  id: string;
  name: string;
  /** Human label (e.g. Slack #cursor-test); hub name stays in `name`. */
  display_name?: string;
  description: string;
  project?: string;
  room_id?: string;
  type: ChannelType;
  source_channel?: string;
  source_message_id?: string;
  archived?: boolean;
  archived_at?: string;
  created_by?: string;
  created: string; // ISO date string
  agents: AgentInfo[];
  members?: string[]; // Explicitly added agent IDs
  human_members?: string[]; // Usernames allowed on private custom channels
  agents_muted?: boolean;
  tags?: string[];
}

export interface CommandOutput {
  command: string;
  plugin: string;
  exit_code: number;
  stdout: string;
  stderr: string;
  duration: number; // Duration in nanoseconds
  success: boolean;
}

export interface AssistantTask {
  id: string;
  title: string;
  description: string;
  priority: number;
  status: 'todo' | 'in_progress' | 'done';
  due_date?: string;
  created_at: string;
  channel: string;
  created_by: string;
}

export interface AssistantReminder {
  id: string;
  content: string;
  trigger_time: string;
  channel: string;
  created_by: string;
  active: boolean;
  created_at: string;
}

export interface AssistantStateResponse {
  channel: string;
  tasks: AssistantTask[];
  reminders: AssistantReminder[];
}

export interface ThinkingStatusMetadata {
  thinking_status: 'started' | 'completed' | 'error' | 'aborted';
  question_id: string;
  thinking_activity?: 'generating_image' | string;
  thinking_activity_detail?: string;
}

/** Ephemeral agent_status: channel agents paused until user sends a message. */
export const METADATA_CHANNEL_HOLD = 'channel_hold';

export const THINKING_ACTIVITY_GENERATING_IMAGE = 'generating_image';
export const THINKING_ACTIVITY_GENERATING_MUSIC = 'generating_music';
export const THINKING_ACTIVITY_USING_TOOL = 'using_tool';
export const THINKING_ACTIVITY_REASONING = 'reasoning';
export const THINKING_ACTIVITY_WRITING = 'writing';
export const THINKING_ACTIVITY_VERIFYING = 'verifying';
export const THINKING_ACTIVITY_PROPOSING_EDIT = 'proposing_edit';
export const THINKING_ACTIVITY_IMPLEMENTATION = 'implementation';
export const THINKING_ACTIVITY_DETAIL_KEY = 'thinking_activity_detail';

export interface ThinkingAgent {
  id: string;
  name: string;
  type: AgentType;
  activity?: string;
  activityDetail?: string;
  toolSteps?: ToolStepMeta[];
  /** Unix ms when this agent entered the typing footer. */
  startedAt?: number;
}

export const TELEMETRY_KIND_METADATA_KEY = 'telemetry_kind';
export const TELEMETRY_PAYLOAD_METADATA_KEY = 'telemetry_payload';

export type TurnTelemetryEvent = {
  id: string;
  at: number;
  channel: string;
  agentId: string;
  agentName: string;
  kind: string;
  detail: string;
  payload?: Record<string, unknown>;
};

export interface ThreadMetadata {
  thread_id: string;
  reply_count: number;
  last_reply_time: string; // ISO date string
  participants: string[]; // Agent/user names who participated in thread
}

export interface CachedAgentInfo {
  type: 'repo' | 'confluence' | 'cli' | 'expert';
  name: string;
  path: string;
  last_used: string; // ISO date string
  cache_size: number; // Size in bytes
  metadata: Record<string, any>;
}

export type AgentCategory = 'all' | 'repo' | 'confluence' | 'cli' | 'expert';

// Integration Settings Types
export interface AnthropicSettings {
  apiKey: string;
  useAIHub: boolean;
  aiHubEndpoint: string;
  aiHubModel: string;
}

export interface GitHubSettings {
  personalAccessToken: string;
}

export interface ConfluenceSettings {
  domain: string;
  email: string;
  apiToken: string;
}

/** OAuth app credentials for Google Meet notes (saved on hub + locally). */
export interface GoogleMeetNotesSettings {
  clientId: string;
  clientSecret: string;
  redirectUrl: string;
}

/** Hub-stored OAuth app config (secret not returned). */
export interface GoogleMeetNotesAppConfig {
  client_id: string;
  redirect_url: string;
  secret_set: boolean;
  configured: boolean;
  connect_ready?: boolean;
  oauth_source?: string;
}

/** Server-reported Google Meet notes connection (Assistant). */
export interface GoogleMeetNotesStatus {
  connected: boolean;
  email?: string;
  notes_count?: number;
  last_sync_at?: string;
  oauth_configured: boolean;
  connect_ready?: boolean;
  oauth_source?: string;
}

export interface WebSearchConfigResponse {
  enabled: boolean;
  provider: string;
  max_results: number;
  api_key_set: boolean;
  keyless: boolean;
  ready: boolean;
}

export type SlackPolicy = 'mention_only' | 'questions' | 'always';

export interface SlackConfigResponse {
  enabled: boolean;
  display_name: string;
  display_icon_url?: string;
  default_policy: SlackPolicy;
  bot_token_set: boolean;
  app_token_set: boolean;
  connect_ready?: boolean;
  oauth?: {
    client_id: string;
    redirect_url: string;
    secret_set: boolean;
    configured: boolean;
    connect_ready?: boolean;
    oauth_source?: string;
  };
}

export interface SlackConnectionResponse {
  oauth_ready: boolean;
  oauth_source: string;
  bot_token_set: boolean;
  app_token_set: boolean;
  bridge_connected: boolean;
  team_id?: string;
  team_name?: string;
  owner_slack_user_id?: string;
  owner_slack_user_name?: string;
}

export type SlackForwardRuleType = 'mention_of_me' | 'prefix' | 'reaction';

export interface SlackForwardRule {
  id?: string;
  type: SlackForwardRuleType;
  enabled: boolean;
  slack_channel_ids?: string[];
  prefix?: string;
  emoji?: string;
}

export interface SlackWorkDayHours {
  weekday: number;
  start: string;
  end: string;
}

export interface SlackHumanDMAwayConfig {
  enabled?: boolean;
  away_enabled?: boolean;
  schedule_enabled?: boolean;
  schedule_timezone?: string;
  work_hours?: SlackWorkDayHours[];
  reply_prefix?: string;
  user_token_set?: boolean;
  monitoring_status?: string;
}

export interface SlackInboxConfig {
  enabled: boolean;
  owner_slack_user_id?: string;
  owner_slack_user_name?: string;
  agent_id?: string;
  agent_name?: string;
  nj_channel?: string;
  slack_dm_channel_id?: string;
  reply_in_thread?: boolean;
  forward_enabled?: boolean;
  forward_rules?: SlackForwardRule[];
  human_dm_away?: SlackHumanDMAwayConfig;
}

export interface SlackStatus {
  enabled: boolean;
  configured: boolean;
  connected?: boolean;
  token_set?: boolean;
  oauth_configured?: boolean;
  bot_user_id?: string;
  team_id?: string;
  bindings_count?: number;
  display_name?: string;
}

export interface SlackChannelInfo {
  id: string;
  name: string;
  is_private: boolean;
  is_member: boolean;
}

export interface SlackBinding {
  id?: string;
  workspace_id: string;
  slack_channel_id: string;
  slack_channel_name?: string;
  nj_channel: string;
  agent_id: string;
  agent_name?: string;
  policy: SlackPolicy;
  reply_in_thread: boolean;
  enabled: boolean;
}

export interface SlackDiagnoseCheck {
  id: string;
  status: 'pass' | 'warn' | 'fail';
  label: string;
  fix?: string;
}

export interface SlackDiagnoseResult {
  ready: boolean;
  app_token_format_ok: boolean;
  bot_token_format_ok: boolean;
  auth_test_ok: boolean;
  socket_open_ok: boolean;
  socket_open_error?: string;
  bridge_connected?: boolean;
  channels_found?: number;
  bindings_count?: number;
  checks?: SlackDiagnoseCheck[];
  recommendations?: string[];
}

export interface SlackSmokeCheck {
  id: string;
  status: string;
  detail?: string;
}

export interface SlackSmokeResult {
  ok: boolean;
  checks: SlackSmokeCheck[];
  duration_ms: number;
  outbound_skipped: boolean;
  channel_id?: string;
}

export interface OllamaSettings {
  endpoint: string;
  defaultModel: string;
  availableModels: string[];
}

export interface PerformanceSettings {
  contextBudgetKB: number;
  ideContextBudgetKB: number;
  implSessionBudgetKB: number;
  maxHistoryMessages: number;
  ollamaNumCtx: number;
  ollamaNumPredict: number;
  ollamaKeepAlive: string;
}

export interface LMStudioSettings {
  endpoint: string;
  defaultModel: string;
  availableModels: string[];
}

export interface IntegrationSettings {
  anthropic: AnthropicSettings;
  github: GitHubSettings;
  confluence: ConfluenceSettings;
  googleMeetNotes: GoogleMeetNotesSettings;
  ollama: OllamaSettings;
  lmstudio: LMStudioSettings;
}

// Connection Test Results
export interface ConnectionTestResult {
  success: boolean;
  message: string;
  error?: string;
}

// Helper function to get agent color based on type
export function getAgentColor(type: AgentType): string {
  switch (type) {
    case 'frontend':
      return '#52b6ef'; // Blue
    case 'backend':
      return '#af77ca'; // Purple
    case 'devops':
      return '#f09348'; // Orange
    case 'database':
      return '#fbd837'; // Yellow
    case 'security':
      return '#f16a5a'; // Red
    case 'rust':
      return '#dea584'; // Rust orange (official Rust color)
    case 'architecture':
      return '#8b5cf6'; // Violet
    case 'code-review':
      return '#06b6d4'; // Cyan
    case 'biology':
      return '#14b8a6'; // Teal for life sciences
    case 'cad':
      return '#6366f1'; // Indigo for CAD
    case 'expert':
      return '#af77ca'; // Purple for custom domain experts
    case 'moderator':
      return '#3b82f6'; // Blue for moderator
    case 'assistant':
      return '#10b981'; // Green for assistant
    case 'human':
      return '#148567'; // Green
    case 'loading':
      return '#3b82f6'; // Blue
    default:
      return '#a9b9ba'; // Gray
  }
}

// Helper to check if a message is a system message
export function isSystemMessage(type: MessageType): boolean {
  return type === 'system_info' || type === 'agent_join' || type === 'agent_leave' || type === 'command_output'
    || type === 'collaboration_status';
}

/** Hub channel name for a Slack mirror (e.g. slack:C06SNHFGQ5T). */
export function isSlackMirrorChannel(channel: string): boolean {
  return channel.startsWith('slack:');
}

/** Slack mirrors use reply_in_thread; show those replies in the main timeline, not only ThreadPanel. */
export function showThreadReplyInMainTimeline(channel: string): boolean {
  return isSlackMirrorChannel(channel);
}

// Helper to check if a message is a thinking status message
export function isThinkingStatusMessage(message: Message): boolean {
  return message.type === 'agent_status' && message.metadata?.thinking_status !== undefined;
}

/** Human slash-command line in channel history (command history, not agent chat). */
export function isSlashCommandMessage(message: Message): boolean {
  if (message.metadata?.slash_command === true) return true;
  const from = message.from;
  const isHuman =
    from.type === 'human' ||
    (from.type === 'general' && from.name !== 'System' && from.id !== 'system');
  return isHuman && message.content.trimStart().startsWith('/');
}

// File change types

export type FileOperation = 'create' | 'edit' | 'delete' | 'move';

export type FileChangeStatus =
  | 'pending'
  | 'approved'
  | 'rejected'
  | 'stale'
  | 'expired'
  | 'failed';

export interface FileChange {
  id: string;
  operation: FileOperation;
  file_path: string;
  old_path?: string;    // For move operations
  new_path?: string;    // For move operations
  old_content?: string; // For edit operations
  new_content?: string; // For create/edit operations
  agent: AgentInfo;
  channel: string;
  status: FileChangeStatus;
  requested_at: string;
  expires_at: string;
  approved_at?: string;
  rejected_at?: string;
  reason?: string;      // Reason for rejection
  metadata?: Record<string, any>;
}

export interface FileChangeRequest {
  id: string;
  changes: FileChange[];
  agent: AgentInfo;
  channel: string;
  requested_at: string;
  expires_at: string;
  status: FileChangeStatus;
}

export interface FileChangeProposal {
  change_id: string;
  operation: string;  // "create", "edit", "delete", "move"
  file_path: string;
  old_path?: string;    // For move operations
  new_path?: string;    // For move operations
  old_content?: string; // For edit operations
  new_content?: string; // For create/edit operations
  agent: AgentInfo;
  channel: string;
  requested_at: string;
  expires_at: string;
  is_delete: boolean;   // Special flag for delete operations
  metadata?: Record<string, any>;
}

export interface FileChangeDiff {
  change: FileChange;
  diff: string;
}

// Command palette types

export type CommandArgType =
  | 'string'
  | 'path'
  | 'provider'
  | 'model'
  | 'agent-name'
  | 'repo-agent-name'
  | 'channel-name'
  | 'collaboration-id'
  | 'assistant-task-id'
  | 'file-change-id'
  | 'collaboration-task';

export interface CommandArgument {
  name: string;
  description: string;
  type: CommandArgType;
  required: boolean;
  options?: string[];
  default?: string;
}

export interface CommandDefinition {
  name: string;
  description: string;
  category: string;
  arguments: CommandArgument[];
}

// ── Collaboration Types ──────────────────────────────────────────────

export type CollaborationPhase =
  | 'draft'
  | 'planning'
  | 'reviewing'
  | 'approved'
  | 'executing'
  | 'completed'
  | 'cancelled';

export type CollaborationSource = 'discussion' | 'runbook';

export type CollaborationTaskStatus =
  | 'pending'
  | 'in_progress'
  | 'completed'
  | 'blocked';

export type DiscussionStatus =
  | 'active'
  | 'converged'
  | 'budget_exhausted'
  | 'timed_out'
  | 'cancelled';

export type ArtifactStatus =
  | 'draft'
  | 'proposed'
  | 'approved'
  | 'superseded';

export type ConsensusState = 'undecided' | 'agrees' | 'disagrees';

export interface CollaborationAgent {
  agent_id: string;
  agent_name: string;
  agent_type: AgentType;
  expertise: string[];
  role: string;
}

export interface ParticipantAddRequest {
  agent_id: string;
  agent_name: string;
  agent_type: AgentType;
  requested_by_id: string;
  requested_by_name: string;
  created_at: string;
}

export interface AssignSuggestion {
  agent_id: string;
  agent_name: string;
  score: number;
  reason: string;
}

export type TaskKind = 'agent' | 'action';

export type BlockedUpstreamPolicy = 'block' | 'skip_branch' | 'fail_run';

export interface ExecutionPolicy {
  max_concurrent_tasks?: number;
  max_execution_messages?: number;
  blocked_upstream_policy?: BlockedUpstreamPolicy;
  strict_task_status?: boolean;
  handoff_max_chars?: number;
  sla_seconds?: number;
  retry_budget?: number;
}

export interface GraphLayoutNode {
  x: number;
  y: number;
}

export type GraphLayout = Record<string, GraphLayoutNode>;

export interface TaskExecutionOptions {
  provider_id?: string;
  requires_approval?: boolean;
  max_retries?: number;
  timeout_seconds?: number;
  queue?: string;
  capability_tags?: string[];
  cache_policy?: 'none' | 'result';
  cache_expiration_seconds?: number;
  refresh_cache?: boolean;
  idempotency_required?: boolean;
  retry_on?: Array<'error' | 'dispatch' | 'timeout' | 'lease_lost'>;
  context_paths?: string[];
  expected_provider_id?: string;
  expected_model?: string;
  routing_reason?: string;
}

export interface TaskActionSpec {
  type: string;
  config?: Record<string, unknown>;
  connector_id?: string;
}

export interface EdgeCondition {
  mode: 'always' | 'on_status' | 'on_output';
  status?: string;
  contains?: string;
  regex?: string;
}

export interface DependencyEdge {
  from_task_id: string;
  condition?: EdgeCondition;
}

export interface DependencyGroup {
  mode: 'all' | 'any';
  task_ids: string[];
}

export interface CollaborationTask {
  id: string;
  title: string;
  description: string;
  assigned_to: string;
  assigned_name: string;
  kind?: TaskKind;
  action?: TaskActionSpec;
  options?: TaskExecutionOptions;
  status: CollaborationTaskStatus;
  dependencies?: string[];
  dependency_edges?: DependencyEdge[];
  dependency_groups?: DependencyGroup[];
  prompt_dispatched?: boolean;
  awaiting_approval?: boolean;
  skipped_due_to_blocked?: boolean;
  output?: string;
  created_at: string;
  updated_at: string;
}

export interface ArtifactEdit {
  editor_id: string;
  editor_name: string;
  content: string;
  version: number;
  timestamp: string;
}

export interface SharedArtifact {
  id: string;
  title: string;
  content: string;
  version: number;
  edit_history?: ArtifactEdit[];
  status: ArtifactStatus;
  created_at: string;
  updated_at: string;
}

export interface DiscussionSession {
  id: string;
  collaboration_id: string;
  topic: string;
  participants: string[];
  max_rounds: number;
  current_round: number;
  turn_budget: number;
  total_message_count: number;
  max_total_messages: number;
  status: DiscussionStatus;
  current_turn_index: number;
  consensus: Record<string, ConsensusState>;
  /** Messages per participant in the current discussion round (hub JSON: turns_this_round). */
  turns_this_round?: Record<string, number>;
}

export interface Collaboration {
  id: string;
  title: string;
  description: string;
  phase: CollaborationPhase;
  source?: CollaborationSource;
  agents: CollaborationAgent[];
  plan?: SharedArtifact;
  tasks?: CollaborationTask[];
  discussion?: DiscussionSession;
  channel: string;
  thread_id?: string;
  created_by: string;
  created_at: string;
  updated_at: string;
  /** sandbox (default) or worktree execution mode. */
  execution_mode?: 'sandbox' | 'worktree';
  /** Git repository root for worktree mode. */
  source_repo_path?: string;
  /** Branch created for worktree execution (e.g. nj/collab-abc12345). */
  worktree_branch?: string;
  /** Absolute execution directory (sandbox or git worktree). */
  working_directory?: string;
  /** True after user confirms workspace setup; until then task prompts are not sent to agents. */
  workspace_acknowledged?: boolean;
  /** True after the hub has sent initial task prompts to assignees. */
  tasks_dispatched?: boolean;
  execution_policy?: ExecutionPolicy;
  graph_layout?: GraphLayout;
  dispatch_paused?: boolean;
  execution_message_count?: number;
  allow_agent_participant_requests?: boolean;
  pending_participant_requests?: ParticipantAddRequest[];
  planning_recap?: string;
  session_recap?: string;
  planning_recap_status?: 'pending' | 'complete' | 'failed' | 'skipped';
  session_recap_status?: 'pending' | 'complete' | 'failed' | 'skipped';
  planning_recap_agent_id?: string;
  session_recap_agent_id?: string;
  /** Validation notices from the last plan approval (task hygiene). */
  approve_warnings?: string[];
  definition_id?: string;
  definition_version?: number;
  run_inputs?: Record<string, string>;
  run_number?: number;
}

export type RunbookDefinitionSource = 'bundled' | 'user' | 'pack';

export interface RunInputSpec {
  key: string;
  type: string;
  label?: string;
  default?: string;
  required?: boolean;
}

export interface RunbookDefinitionSummary {
  id: string;
  title: string;
  description?: string;
  version: number;
  source: RunbookDefinitionSource;
  pack_id?: string;
  updated_at?: string;
}

export interface RunbookDefinition extends RunbookDefinitionSummary {
  agent_ids?: string[];
  execution_policy?: ExecutionPolicy;
  graph_layout?: GraphLayout;
  inputs?: RunInputSpec[];
  tasks: CollaborationTask[];
}

export interface RunbookRunRecord {
  id: string;
  definition_id?: string;
  definition_version?: number;
  run_number: number;
  started_at: string;
  updated_at: string;
  phase: string;
  channel?: string;
  title?: string;
}

export interface ConnectorProfile {
  id: string;
  type: string;
  label: string;
  config?: Record<string, string>;
  secret_set?: boolean;
  created_at?: string;
  updated_at?: string;
}

export type StreamProtocol = 'mqtt' | 'kafka';
export type StreamActionType = 'runbook' | 'channel' | 'webhook';
export type StreamMatchOp = 'equals' | 'contains';

export interface StreamMatchSpec {
  json_path?: string;
  op?: StreamMatchOp;
  value?: string;
}

export interface StreamActionSpec {
  type: StreamActionType;
  definition_id?: string;
  version?: number;
  agent_ids?: string[];
  channel?: string;
  input_map?: Record<string, string>;
  hub_channel?: string;
  message_template?: string;
  mention_agent_ids?: string[];
  webhook_connector_id?: string;
  url_override?: string;
  body_template?: string;
}

export interface StreamSubscription {
  id: string;
  label: string;
  enabled: boolean;
  protocol: StreamProtocol;
  connector_id: string;
  topic: string;
  match?: StreamMatchSpec | null;
  debounce_ms?: number;
  action: StreamActionSpec;
  created_at?: string;
  updated_at?: string;
}

export interface StreamSubStatus {
  subscription_id: string;
  label?: string;
  enabled: boolean;
  connected: boolean;
  last_message_at?: string | null;
  last_fire_at?: string | null;
  last_error?: string;
  fire_count: number;
  skip_count: number;
}

export interface StreamManagerStatus {
  running: boolean;
  subscriptions: StreamSubStatus[];
}

export interface StreamDispatchResult {
  matched: boolean;
  fired: boolean;
  skipped: boolean;
  reason?: string;
  error?: string;
}

export interface RunbookTemplate {
  name: string;
  title: string;
  description: string;
  execution_policy?: ExecutionPolicy;
  tasks: CollaborationTask[];
}

export function isCollaborationMessage(message: Message): boolean {
  return (
    message.type === 'collaboration_plan' ||
    message.type === 'collaboration_task' ||
    message.type === 'collaboration_status' ||
    message.type === 'collaboration_discussion' ||
    message.type === 'collaboration_recap'
  );
}

/** Main-channel rows that must not be dropped when `content` is empty/whitespace (or missing). */
export function channelTimelineAllowsEmptyContent(
  type: MessageType,
  metadata?: Record<string, unknown>,
): boolean {
  return (
    metadata?.change_proposal !== undefined ||
    type === 'file_change' ||
    type === 'agent_join' ||
    type === 'agent_leave' ||
    type === 'system_info' ||
    type === 'collaboration_discussion' ||
    type === 'collaboration_plan' ||
    type === 'collaboration_task' ||
    type === 'collaboration_status'
  );
}

export function getCollaborationId(message: Message): string | undefined {
  return message.metadata?.collaboration_id as string | undefined;
}

export function getCollaborationPhase(message: Message): CollaborationPhase | undefined {
  return message.metadata?.collaboration_phase as CollaborationPhase | undefined;
}

