import type { AgentInfo, CachedAgentInfo } from '../types/protocol';
import { normalizeWorkspacePath } from './repoAgentWorkspace';

/** True when this cached row already has a live (or loading) hub agent. */
export function isCachedAgentAlreadyLoaded(
  cached: CachedAgentInfo,
  agents: AgentInfo[],
  loadingAgentNames: Iterable<string>
): boolean {
  const loading = new Set(
    Array.from(loadingAgentNames, (n) => n.toLowerCase())
  );
  if (loading.has(cached.name.toLowerCase())) {
    return true;
  }

  const cachedPath = cached.path ? normalizeWorkspacePath(cached.path) : '';
  const cachedName = cached.name.toLowerCase();
  const metaNames = Array.isArray(cached.metadata?.agent_names)
    ? (cached.metadata.agent_names as string[]).map((n) => String(n).toLowerCase())
    : [];

  for (const a of agents) {
    if (a.type !== cached.type) continue;
    if (a.status === 'removed') continue;

    const liveName = a.name.toLowerCase();
    if (liveName === cachedName || metaNames.includes(liveName)) {
      return true;
    }

    if (cached.type === 'repo' && cachedPath) {
      if (a.repository_path && normalizeWorkspacePath(a.repository_path) === cachedPath) {
        return true;
      }
    }

    if (cached.type === 'confluence' && cached.path) {
      const key = (a.confluence_space_key || a.knowledge_path || '').trim();
      if (key && normalizeWorkspacePath(key) === cachedPath) {
        return true;
      }
    }

    if (cached.type === 'cli' && cached.path && a.repository_path) {
      if (normalizeWorkspacePath(a.repository_path) === cachedPath) {
        return true;
      }
    }

    if (cached.type === 'expert' && a.type === 'expert' && liveName === cachedName) {
      return true;
    }
  }

  return false;
}

/** Resolve a running hub agent for a cached disk entry, if any. */
export function findLiveAgentForCached(
  cached: CachedAgentInfo,
  agents: AgentInfo[]
): AgentInfo | undefined {
  if (!isCachedAgentAlreadyLoaded(cached, agents, [])) {
    return undefined;
  }
  const cachedPath = cached.path ? normalizeWorkspacePath(cached.path) : '';
  const cachedName = cached.name.toLowerCase();

  return agents.find((a) => {
    if (a.type !== cached.type || a.status === 'removed') return false;
    if (a.name.toLowerCase() === cachedName) return true;
    if (cached.type === 'repo' && cachedPath && a.repository_path) {
      return normalizeWorkspacePath(a.repository_path) === cachedPath;
    }
    return false;
  });
}
