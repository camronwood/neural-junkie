import { describe, expect, it } from 'vitest';
import {
  collaboratorsAddedSince,
  decideCollabPanelOpen,
  inboundImplementationSessionCompleted,
  isTerminalCollaborationPhase,
  parseCollabParticipantAddRequest,
  shouldToastCollaboratorAdds,
} from './chatInboundCollab';
import type { Collaboration, Message } from '../types/protocol';

function snap(partial: Partial<Collaboration> & { id: string }): Collaboration {
  return {
    id: partial.id,
    title: partial.title ?? 't',
    phase: partial.phase ?? 'planning',
    channel: partial.channel ?? 'collab-1',
    updated_at: partial.updated_at ?? '2026-07-14T12:00:00Z',
    workspace_acknowledged: true,
    agents: partial.agents ?? [],
    tasks: [],
  } as Collaboration;
}

function msg(partial: Partial<Message> & { type: Message['type'] }): Message {
  return {
    id: partial.id ?? 'm1',
    type: partial.type,
    channel: partial.channel ?? 'collab-1',
    content: partial.content ?? '',
    from: partial.from ?? { id: 'a1', name: 'A', type: 'assistant' },
    timestamp: partial.timestamp ?? new Date().toISOString(),
    metadata: partial.metadata ?? {},
  } as Message;
}

describe('chatInboundCollab', () => {
  it('detects terminal phases', () => {
    expect(isTerminalCollaborationPhase('completed')).toBe(true);
    expect(isTerminalCollaborationPhase('planning')).toBe(false);
  });

  it('lists newly added collaborators', () => {
    const prev = snap({
      id: 'c1',
      agents: [{ agent_id: 'a1', agent_name: 'A', agent_type: 'assistant' } as never],
    });
    const next = snap({
      id: 'c1',
      agents: [
        { agent_id: 'a1', agent_name: 'A', agent_type: 'assistant' } as never,
        { agent_id: 'a2', agent_name: 'B', agent_type: 'architecture' } as never,
      ],
    });
    expect(collaboratorsAddedSince(prev, next).map((a) => a.agent_id)).toEqual(['a2']);
  });

  it('decides panel open/update', () => {
    const snapshot = snap({ id: 'c1', phase: 'planning', channel: 'collab-1' });
    const open = decideCollabPanelOpen({
      snapshot,
      activeChannel: 'collab-1',
      currentlyOpen: null,
      message: msg({ type: 'collaboration_discussion', channel: 'collab-1' }),
    });
    expect(open.action).toBe('open');

    const update = decideCollabPanelOpen({
      snapshot: snap({ id: 'c1', phase: 'executing', channel: 'collab-1' }),
      activeChannel: 'collab-1',
      currentlyOpen: snap({ id: 'c1' }),
      message: msg({ type: 'chat' }),
    });
    expect(update.action).toBe('update_open');
  });

  it('toasts collaborator adds only on active planning/reviewing', () => {
    const previous = snap({ id: 'c1', agents: [] });
    const snapshot = snap({
      id: 'c1',
      phase: 'planning',
      agents: [{ agent_id: 'a2', agent_name: 'B', agent_type: 'architecture' } as never],
    });
    expect(
      shouldToastCollaboratorAdds({ previous, snapshot, isActiveChannelCollab: true })
    ).toBe(true);
    expect(
      shouldToastCollaboratorAdds({ previous, snapshot, isActiveChannelCollab: false })
    ).toBe(false);
  });

  it('parses participant add request', () => {
    const parsed = parseCollabParticipantAddRequest(
      msg({
        type: 'system',
        metadata: {
          event: 'collab-participant-add-request',
          collaboration_id: 'c1',
          requested_agent_id: 'a9',
          requested_agent_name: 'Claude',
          requested_by_name: 'Gemini',
        },
      })
    );
    expect(parsed).toEqual({
      collabID: 'c1',
      agentID: 'a9',
      agentName: 'Claude',
      requestedBy: 'Gemini',
    });
  });

  it('detects implementation session completion metadata', () => {
    expect(
      inboundImplementationSessionCompleted(
        msg({ type: 'answer', metadata: { implementation_session_complete: true } })
      )
    ).toBe(true);
    expect(inboundImplementationSessionCompleted(msg({ type: 'answer' }))).toBe(false);
  });
});
