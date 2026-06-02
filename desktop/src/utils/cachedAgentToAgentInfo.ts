import type { AgentInfo, CachedAgentInfo } from '../types/protocol';

/** Maps a disk-cached agent row to AgentInfo for the shared info modal (offline view). */
export function cachedAgentToAgentInfo(cached: CachedAgentInfo): AgentInfo {
  const patterns = cached.metadata?.code_patterns;
  const expertise = Array.isArray(patterns) ? (patterns as string[]) : [];
  return {
    id: `cached:${cached.type}:${cached.path || cached.name}`,
    name: cached.name,
    type: cached.type,
    expertise,
    status: 'inactive',
    model: '',
    ai_provider: '',
    ai_model: '',
    is_paused: false,
    supports_vision: false,
    supports_image_generation: false,
    indexing_status: cached.type === 'repo' ? 'ready' : undefined,
    index_progress: cached.type === 'repo' ? 100 : undefined,
    repository_path: cached.type === 'repo' ? cached.path : '',
    knowledge_path: cached.type === 'confluence' ? cached.path : '',
    confluence_space_key: cached.metadata?.space_key as string | undefined,
    last_active_time: cached.last_used,
  };
}
