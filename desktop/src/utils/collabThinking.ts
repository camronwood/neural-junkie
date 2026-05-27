import type { AgentInfo, Collaboration, CollaborationAgent } from '../types/protocol';
import { useChatStore } from '../stores/chatStore';

/** Participant who should be speaking next in a collab discussion. */
export function getCurrentTurnAgent(collab: Collaboration): CollaborationAgent | null {
  const agents = collab.agents ?? [];
  if (agents.length === 0) return null;

  const disc = collab.discussion;
  if (disc?.participants?.length) {
    const idx = disc.current_turn_index ?? 0;
    const agentId = disc.participants[idx];
    if (agentId) {
      const found = agents.find((a) => a.agent_id === agentId);
      if (found) return found;
    }
  }
  return agents[0] ?? null;
}

function agentInfoFromCollabParticipant(p: CollaborationAgent): Pick<AgentInfo, 'id' | 'name' | 'type'> {
  return {
    id: p.agent_id,
    name: p.agent_name,
    type: (p.agent_type as AgentInfo['type']) || 'assistant',
  };
}

/**
 * Shows the typing indicator for whoever holds the current collab turn during planning/review/executing.
 */
export function syncCollabTurnThinking(
  collab: Collaboration | null | undefined,
  channelName: string
): void {
  if (!collab?.id || !channelName) return;
  const phase = collab.phase;
  if (phase !== 'planning' && phase !== 'reviewing' && phase !== 'executing') {
    return;
  }
  if (collab.channel && collab.channel !== channelName) {
    return;
  }

  const turn = getCurrentTurnAgent(collab);
  if (!turn) return;

  const st = useChatStore.getState();
  const inner = st.channelThinkingAgents.get(channelName);
  if (inner?.has(turn.agent_id)) {
    return;
  }

  const info = agentInfoFromCollabParticipant(turn);
  st.addThinkingAgent(channelName, info.id, info.name, info.type);
}
