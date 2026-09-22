/** ChatAPI-facing DTOs extracted from chatAPI.ts for smaller modules. */
import type { ResolvedCapability } from '../../types/protocol';

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

