import type { AgentInfo, CachedAgentInfo } from '../types/protocol';

export function isConsultOnlyRepoAgent(agent: Pick<AgentInfo, 'type' | 'consult_only'>): boolean {
  return agent.type === 'repo' && agent.consult_only === true;
}

/** Auto workspace indexes (`index-*` / consult-only) — not user-facing chat agents. */
export function isWorkspaceIndexAgent(
  agent: Pick<AgentInfo, 'type' | 'name' | 'consult_only'> | Pick<CachedAgentInfo, 'type' | 'name'>
): boolean {
  if (agent.type !== 'repo') return false;
  if ('consult_only' in agent && agent.consult_only === true) return true;
  const name = (agent.name || '').trim().toLowerCase();
  return name.startsWith('index-') || name.startsWith('__index');
}

export function isUserFacingAgent(agent: AgentInfo): boolean {
  return !isConsultOnlyRepoAgent(agent) && !isWorkspaceIndexAgent(agent);
}

export function isUserFacingCachedAgent(agent: CachedAgentInfo): boolean {
  return !isWorkspaceIndexAgent(agent);
}
