import type { ParticipantAddRequest } from '../../types/protocol';
import { taskActionBtnStyle } from './collabStyles';

export interface CollabParticipantRequestsProps {
  requests: ParticipantAddRequest[];
  isTerminal: boolean;
  isSubmitting: boolean;
  onApprove: (agentId: string, agentName: string) => void | Promise<void>;
  onDeny: (agentId: string, agentName: string) => void | Promise<void>;
}

/** Pending agent join requests awaiting human approve/deny. */
export function CollabParticipantRequests({
  requests: pendingParticipantRequests,
  isTerminal,
  isSubmitting,
  onApprove,
  onDeny,
}: CollabParticipantRequestsProps) {
  if (pendingParticipantRequests.length === 0 || isTerminal) return null;

  return (
    <div
      data-testid="collaboration-participant-requests"
      style={{
        marginBottom: 16,
        padding: 12,
        borderRadius: 8,
        border: '1px solid #3b82f6',
        backgroundColor: 'rgba(59, 130, 246, 0.12)',
      }}
    >
      <div style={{ fontSize: 13, fontWeight: 600, color: '#93c5fd', marginBottom: 8 }}>
        Pending join requests
      </div>
      {pendingParticipantRequests.map((req) => (
        <div
          key={`${req.agent_id}-${req.created_at}`}
          style={{
            display: 'flex',
            flexWrap: 'wrap',
            alignItems: 'center',
            gap: 8,
            marginBottom: 8,
            fontSize: 12,
            color: 'var(--text-secondary, #ccc)',
          }}
        >
          <span style={{ flex: '1 1 140px' }}>
            <strong>@{req.requested_by_name || 'An agent'}</strong> wants to add{' '}
            <strong>@{req.agent_name}</strong>
          </span>
          <button
            type="button"
            data-testid={`collaboration-participant-approve-${req.agent_id}`}
            onClick={() => void onApprove(req.agent_id, req.agent_name)}
            disabled={isSubmitting}
            style={taskActionBtnStyle('#10b981')}
          >
            Approve
          </button>
          <button
            type="button"
            data-testid={`collaboration-participant-deny-${req.agent_id}`}
            onClick={() => void onDeny(req.agent_id, req.agent_name)}
            disabled={isSubmitting}
            style={taskActionBtnStyle('#ef4444')}
          >
            Deny
          </button>
        </div>
      ))}
    </div>
  );
}
