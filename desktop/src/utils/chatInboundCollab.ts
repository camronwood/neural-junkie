/** Pure helpers for ChatWindow WebSocket collaboration inbound routing. */

import type { Collaboration, Message } from '../types/protocol';
import { getCollaborationId, isCollaborationMessage } from '../types/protocol';

export function isTerminalCollaborationPhase(phase?: Collaboration['phase']): boolean {
  return phase === 'completed' || phase === 'cancelled';
}

/** Agents present in next but not previous snapshot (same collab). */
export function collaboratorsAddedSince(
  previous: Collaboration | undefined,
  next: Collaboration
): Collaboration['agents'] {
  if (!next?.agents?.length) return [];
  const existing = new Set((previous?.agents || []).map((a) => a.agent_id));
  return next.agents.filter((a) => !existing.has(a.agent_id));
}

export type CollabInboundOpenDecision =
  | { action: 'noop' }
  | { action: 'update_open'; snapshot: Collaboration }
  | { action: 'open'; snapshot: Collaboration };

/**
 * Decide how inbound collaboration_data should affect the open side panel.
 * Pure: no store or toast side effects.
 */
export function decideCollabPanelOpen(args: {
  snapshot: Collaboration;
  activeChannel: string;
  currentlyOpen: Collaboration | null | undefined;
  message: Message;
}): CollabInboundOpenDecision {
  const { snapshot, activeChannel, currentlyOpen, message } = args;
  const collabChannel = snapshot.channel || message.channel || '';
  const isActiveChannelCollab = !collabChannel || collabChannel === activeChannel;

  if (currentlyOpen?.id === snapshot.id) {
    if (isActiveChannelCollab || isTerminalCollaborationPhase(snapshot.phase)) {
      return { action: 'update_open', snapshot };
    }
    return { action: 'noop' };
  }
  if (
    !currentlyOpen &&
    isActiveChannelCollab &&
    isCollaborationMessage(message) &&
    !isTerminalCollaborationPhase(snapshot.phase)
  ) {
    return { action: 'open', snapshot };
  }
  return { action: 'noop' };
}

export function shouldToastCollaboratorAdds(args: {
  previous: Collaboration | undefined;
  snapshot: Collaboration;
  isActiveChannelCollab: boolean;
}): boolean {
  if (!args.isActiveChannelCollab) return false;
  const phase = args.snapshot.phase;
  if (phase !== 'planning' && phase !== 'reviewing') return false;
  return collaboratorsAddedSince(args.previous, args.snapshot).length > 0;
}

/** Parse collab participant-add request fields from an inbound message. */
export function parseCollabParticipantAddRequest(message: Message): {
  collabID: string;
  agentID: string;
  agentName: string;
  requestedBy: string;
} | null {
  if (message.metadata?.event !== 'collab-participant-add-request') return null;
  const collabData = message.metadata?.collaboration_data as Collaboration | undefined;
  const collabID = getCollaborationId(message) || collabData?.id || '';
  const agentID =
    typeof message.metadata?.requested_agent_id === 'string'
      ? message.metadata.requested_agent_id
      : '';
  if (!collabID || !agentID) return null;
  return {
    collabID,
    agentID,
    agentName:
      typeof message.metadata?.requested_agent_name === 'string'
        ? message.metadata.requested_agent_name
        : 'the agent',
    requestedBy:
      typeof message.metadata?.requested_by_name === 'string'
        ? message.metadata.requested_by_name
        : 'An agent',
  };
}

/** Whether implementation-session completion metadata should surface in UI. */
export function inboundImplementationSessionCompleted(message: Message): boolean {
  const meta = message.metadata;
  if (!meta) return false;
  if (meta.implementation_session_complete === true) return true;
  if (meta.implementation_session_outcome && typeof meta.implementation_session_outcome === 'object') {
    const outcome = meta.implementation_session_outcome as Record<string, unknown>;
    return outcome.complete === true || outcome.status === 'complete';
  }
  return false;
}
