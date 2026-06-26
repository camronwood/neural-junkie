import type { AgentInfo } from '../types/protocol';

export function isConsultOnlyRepoAgent(agent: Pick<AgentInfo, 'type' | 'consult_only'>): boolean {
  return agent.type === 'repo' && agent.consult_only === true;
}

export function isUserFacingAgent(agent: AgentInfo): boolean {
  return !isConsultOnlyRepoAgent(agent);
}
