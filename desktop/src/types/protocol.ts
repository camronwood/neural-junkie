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
  | 'file_change';

export type AgentType =
  | 'frontend'
  | 'backend'
  | 'devops'
  | 'database'
  | 'security'
  | 'rust'
  | 'general'
  | 'repo'
  | 'helper'
  | 'confluence'
  | 'moderator'  // System moderator agent
  | 'assistant'  // Personal assistant agent
  | 'human'; // Added for human users

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
  indexing_status?: string;
  index_progress?: number;
  repository_path?: string;
  knowledge_path?: string;
  confluence_space_key?: string;
  last_active_time?: string;
  removed_from?: string[];
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

export type ChannelType = 'public' | 'dm' | 'custom';

export interface Channel {
  id: string;
  name: string;
  description: string;
  project?: string;
  type: ChannelType;
  created_by?: string;
  created: string; // ISO date string
  agents: AgentInfo[];
  members?: string[]; // Explicitly added agent IDs
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

export interface ThinkingStatusMetadata {
  thinking_status: 'started' | 'completed' | 'error';
  question_id: string;
}

export interface ThinkingAgent {
  id: string;
  name: string;
  type: AgentType;
}

export interface ThreadMetadata {
  thread_id: string;
  reply_count: number;
  last_reply_time: string; // ISO date string
  participants: string[]; // Agent/user names who participated in thread
}

export interface CachedAgentInfo {
  type: 'repo' | 'helper' | 'confluence';
  name: string;
  path: string;
  last_used: string; // ISO date string
  cache_size: number; // Size in bytes
  metadata: Record<string, any>;
}

export type AgentCategory = 'all' | 'repo' | 'helper' | 'confluence';

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

export interface OllamaSettings {
  endpoint: string;
  defaultModel: string;
  availableModels: string[];
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
    case 'moderator':
      return '#3b82f6'; // Blue for moderator
    case 'assistant':
      return '#10b981'; // Green for assistant
    case 'human':
      return '#148567'; // Green
    default:
      return '#a9b9ba'; // Gray
  }
}

// Helper to check if a message is a system message
export function isSystemMessage(type: MessageType): boolean {
  return type === 'system_info' || type === 'agent_join' || type === 'agent_leave' || type === 'command_output';
}

// Helper to check if a message is a thinking status message
export function isThinkingStatusMessage(message: Message): boolean {
  return message.type === 'agent_status' && message.metadata?.thinking_status !== undefined;
}

// File change types

export type FileOperation = 'create' | 'edit' | 'delete' | 'move';

export type FileChangeStatus = 'pending' | 'approved' | 'rejected' | 'expired';

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

export type CommandArgType = 'string' | 'path' | 'provider' | 'model' | 'agent-name';

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

