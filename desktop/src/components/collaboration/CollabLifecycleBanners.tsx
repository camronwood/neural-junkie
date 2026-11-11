import type { Collaboration } from '../../types/protocol';
import {
  FILE_PROPOSAL_QUEUE_LABEL,
  type PlanningGenerationErrorEntry,
} from '../../utils/collaborationPanelState';
import { CollabPhaseStatusStack } from './CollabPhaseStatusStack';

export interface CollabLifecycleBannersProps {
  collaboration: Collaboration;
  collabChannel: string;
  setupInProgress: boolean;
  planningRecapPending: boolean;
  recapFacilitatorName: string;
  planningGenErrors: PlanningGenerationErrorEntry[];
  planningStalled: string[];
  planningAwaitingFirstTurn: boolean;
  executingStuck: boolean;
  chatOnlyCompletedFileTasksCount: number;
  fileTasksAwaitingProposalCount: number;
  awaitingWorkspaceConfirmation: boolean;
  approvedAwaitingDispatch: boolean;
  submitForReviewEnabled: boolean;
  onConfirmWorkspace?: () => void;
}

/** Lifecycle / phase banners shown above the participants list in the collaboration panel. */
export function CollabLifecycleBanners({
  collaboration: c,
  collabChannel,
  setupInProgress,
  planningRecapPending,
  recapFacilitatorName,
  planningGenErrors,
  planningStalled,
  planningAwaitingFirstTurn,
  executingStuck,
  chatOnlyCompletedFileTasksCount,
  fileTasksAwaitingProposalCount,
  awaitingWorkspaceConfirmation,
  approvedAwaitingDispatch,
  submitForReviewEnabled,
  onConfirmWorkspace,
}: CollabLifecycleBannersProps) {
  return (
    <>
      {setupInProgress && (
        <div
          data-testid="collaboration-setup-banner"
          role="status"
          style={{
            marginBottom: 16,
            padding: 12,
            borderRadius: 8,
            border: '1px solid #8b5cf6',
            backgroundColor: 'rgba(139, 92, 246, 0.12)',
          }}
        >
          <div style={{ fontSize: 13, fontWeight: 600, color: '#c4b5fd', marginBottom: 6 }}>
            Setting up collaboration workspace…
          </div>
          <p style={{ margin: 0, fontSize: 12, color: 'var(--text-secondary, #ccc)', lineHeight: 1.45 }}>
            Creating the sandbox or git worktree, materializing deliverable stub files, and preparing task
            assignments. This can take up to a minute on first run.
          </p>
        </div>
      )}

      {c.phase === 'reviewing' && (
        <div
          data-testid="collaboration-review-banner"
          style={{
            marginBottom: 16,
            padding: 12,
            borderRadius: 8,
            border: '1px solid #3b82f6',
            backgroundColor: 'rgba(59, 130, 246, 0.12)',
          }}
        >
          <div style={{ fontSize: 13, fontWeight: 600, color: '#93c5fd', marginBottom: 6 }}>
            Waiting for your approval
          </div>
          <p style={{ margin: 0, fontSize: 13, color: 'var(--text-secondary, #ccc)', lineHeight: 1.45 }}>
            {planningRecapPending && !c.planning_recap?.trim() ? (
              <>
                Planning is complete. <strong>@{recapFacilitatorName}</strong> is writing the session summary
                now — the plan is below. <strong>Approve &amp; start</strong> unlocks when the recap is posted.
              </>
            ) : (
              <>
                Planning is complete. Review the plan and session summary below, then{' '}
                <strong>Approve &amp; start</strong> to create the collaboration workspace and assign tasks, or{' '}
                <strong>Revise</strong> to send agents back to planning.
              </>
            )}
          </p>
          {planningRecapPending && !c.planning_recap?.trim() && (
            <p style={{ margin: '8px 0 0', fontSize: 12, color: '#fbbf24' }} role="status">
              Generating session summary with @{recapFacilitatorName}… This usually takes 30–90 seconds.
            </p>
          )}
        </div>
      )}

      <CollabPhaseStatusStack
        phase={c.phase}
        collabIdPrefix={c.id.slice(0, 8)}
        planningGenErrors={planningGenErrors}
        planningStalled={planningStalled}
        planningAwaitingFirstTurn={planningAwaitingFirstTurn}
        executingStuck={executingStuck}
      />

      {c.phase === 'executing' && chatOnlyCompletedFileTasksCount > 0 && (
        <div
          data-testid="collaboration-file-deliverable-hint-banner"
          style={{
            marginBottom: 16,
            padding: 12,
            borderRadius: 8,
            border: '1px solid #b45309',
            backgroundColor: 'rgba(180, 83, 9, 0.12)',
          }}
        >
          <div style={{ fontSize: 13, fontWeight: 600, color: '#fcd34d', marginBottom: 6 }}>
            File deliverable may be missing
          </div>
          <p style={{ margin: 0, fontSize: 12, color: 'var(--text-secondary, #ccc)', lineHeight: 1.45 }}>
            {chatOnlyCompletedFileTasksCount} task(s) marked completed without file output. Approve{' '}
            <strong>{FILE_PROPOSAL_QUEUE_LABEL}</strong> in the toolbar or chat proposal cards — not tool
            approvals. Ask the assignee to emit a [FILE_CHANGE] block if none appear. Prefer{' '}
            <strong>Redispatch</strong> over Mark collaboration done until files land.
          </p>
        </div>
      )}

      {c.phase === 'executing' && fileTasksAwaitingProposalCount > 0 && (
        <div
          data-testid="collaboration-file-awaiting-proposal-banner"
          style={{
            marginBottom: 16,
            padding: 12,
            borderRadius: 8,
            border: '1px solid #b45309',
            backgroundColor: 'rgba(180, 83, 9, 0.12)',
          }}
        >
          <div style={{ fontSize: 13, fontWeight: 600, color: '#fcd34d', marginBottom: 6 }}>
            Waiting for file proposals
          </div>
          <p style={{ margin: 0, fontSize: 12, color: 'var(--text-secondary, #ccc)', lineHeight: 1.45 }}>
            {fileTasksAwaitingProposalCount} in-progress task(s) need a <strong>[FILE_CHANGE]</strong> proposal.
            Chat markdown does not write to disk — approve proposals in{' '}
            <strong>{FILE_PROPOSAL_QUEUE_LABEL}</strong> (toolbar) when they appear.
          </p>
        </div>
      )}

      {awaitingWorkspaceConfirmation && (
        <div
          data-testid="collaboration-workspace-gate-banner"
          style={{
            marginBottom: 16,
            padding: 12,
            borderRadius: 8,
            border: '1px solid #b45309',
            backgroundColor: 'rgba(180, 83, 9, 0.12)',
          }}
        >
          <div style={{ fontSize: 13, fontWeight: 600, color: '#fcd34d', marginBottom: 6 }}>
            Confirm execution workspace
          </div>
          <p style={{ margin: '0 0 10px', fontSize: 12, color: 'var(--text-secondary, #ccc)', lineHeight: 1.45 }}>
            Agents are paused until you confirm where files will be written. Click <strong>Confirm workspace</strong>{' '}
            below or use the dialog on channel <span style={{ fontFamily: 'monospace' }}>{collabChannel}</span>.
          </p>
          {onConfirmWorkspace && (
            <button
              type="button"
              onClick={onConfirmWorkspace}
              style={{
                padding: '6px 12px',
                borderRadius: 6,
                border: 'none',
                backgroundColor: '#d97706',
                color: '#fff',
                fontWeight: 600,
                cursor: 'pointer',
                fontSize: 12,
              }}
            >
              Confirm workspace
            </button>
          )}
        </div>
      )}

      {approvedAwaitingDispatch && (
        <div
          data-testid="collaboration-dispatch-banner"
          style={{
            marginBottom: 16,
            padding: 12,
            borderRadius: 8,
            border: '1px solid #10b981',
            backgroundColor: 'rgba(16, 185, 129, 0.12)',
          }}
        >
          <div style={{ fontSize: 13, fontWeight: 600, color: '#6ee7b7', marginBottom: 6 }}>
            Tasks dispatching…
          </div>
          <p style={{ margin: 0, fontSize: 12, color: 'var(--text-secondary, #ccc)', lineHeight: 1.45 }}>
            Workspace confirmed. Task prompts are being sent to assignees — no Continue step needed for sandbox +
            bound project repos.
          </p>
        </div>
      )}

      {c.phase === 'planning' && !planningAwaitingFirstTurn && (
        <p style={{ margin: '0 0 16px', fontSize: 13, color: 'var(--text-secondary, #aaa)', lineHeight: 1.45 }}>
          Agents are discussing and refining the plan.{' '}
          {submitForReviewEnabled ? (
            <>
              Planning discussion is complete — use <strong>Submit for review</strong> for the session summary,
              or wait for automatic transition if consensus already moved you to review.
            </>
          ) : (
            <>
              <strong>Submit for review</strong> unlocks when planning finishes (consensus, message limit, or round
              limit). Most collaborations move to review automatically when that happens.
            </>
          )}
        </p>
      )}
    </>
  );
}
