import { useEffect, useState, type CSSProperties } from 'react';
import { shallow } from 'zustand/shallow';
import { useChatStore } from '../stores/chatStore';
import { ChatAPI } from '../api/chatAPI';
import { RichMarkdownView } from './RichMarkdownView';
import type {
  Collaboration,
  CollaborationTask,
  CollaborationPhase,
} from '../types/protocol';
import { confirmReplaceCollaborationExecution } from '../utils/collaborationConfirm';
import {
  canSubmitCollaborationForReview,
  collaborationPrimaryActionLabel,
  collaborationPrimaryActionTitle,
  collaborationSubmitForReviewTitle,
} from '../utils/collaborationActionLabels';
import {
  isApprovedAwaitingDispatch,
  isAwaitingWorkspaceConfirmation,
  isPlanningAwaitingFirstTurn,
  planningStalledParticipantNames,
  taskNeedsFileDeliverable,
} from '../utils/collaborationPanelState';
import { buildCollabChannelOutboundMetadata } from '../utils/collaborationOutboundMetadata';
import { loadWorkspaceContextMode } from '../utils/outboundChatMetadata';
import { taskOrchestrationLabel } from '../utils/collaborationTaskOrchestration';
import { shrinkablePanelStyle } from '../utils/panelLayout';
import { RunbookGraphModal } from './runbook-graph';

interface CollaborationPanelProps {
  collaboration: Collaboration;
  /** Active collaborations (planning/review) for extend dropdown. */
  extendableCollaborations?: Collaboration[];
  /** If set and different from `collaboration`, approving will replace that running execution (after user confirms). */
  executingCollaboration?: Collaboration | null;
  onClose: () => void;
  /** Refetch collaboration snapshots after approve/revise/cancel (keeps UI in sync). */
  onAfterCollaborationCommand?: () => Promise<void>;
  /** Opens the desktop workspace confirmation dialog (executing phase, pre-ack). */
  onConfirmWorkspace?: () => void;
}

const phaseLabels: Record<CollaborationPhase, string> = {
  draft: 'Draft (Runbook)',
  planning: 'Planning',
  reviewing: 'Reviewing Plan',
  approved: 'Approved',
  executing: 'Executing',
  completed: 'Completed',
  cancelled: 'Cancelled',
};

const phaseColors: Record<CollaborationPhase, string> = {
  draft: '#64748b',
  planning: '#f59e0b',
  reviewing: '#3b82f6',
  approved: '#10b981',
  executing: '#8b5cf6',
  completed: '#059669',
  cancelled: '#ef4444',
};

function taskIcon(status: string): string {
  switch (status) {
    case 'in_progress': return '🔄';
    case 'completed': return '✅';
    case 'blocked': return '🚫';
    default: return '⬜';
  }
}

export function CollaborationPanel({
  collaboration,
  extendableCollaborations = [],
  executingCollaboration,
  onClose,
  onAfterCollaborationCommand,
  onConfirmWorkspace,
}: CollaborationPanelProps) {
  const { serverAddr, channel, username } = useChatStore(
    (s) => ({ serverAddr: s.serverAddr, channel: s.channel, username: s.username }),
    shallow
  );
  const [api] = useState(() => new ChatAPI(serverAddr));
  const [feedback, setFeedback] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [renaming, setRenaming] = useState(false);
  const [renameDraft, setRenameDraft] = useState('');
  const [graphOpen, setGraphOpen] = useState(false);
  const [extendTargetId, setExtendTargetId] = useState('');
  const [extendRounds, setExtendRounds] = useState('1');
  const [extendMessages, setExtendMessages] = useState('');
  const from = { name: username || 'User', type: 'human' };
  const workspaceContextMode = loadWorkspaceContextMode();

  const c = collaboration;
  const extendCandidates =
    extendableCollaborations.length > 0
      ? extendableCollaborations
      : c.phase === 'planning' || c.phase === 'reviewing'
        ? [c]
        : [];
  const collabChannel = c.channel?.trim() || channel;

  const sendCollabCommand = async (content: string) => {
    const meta = buildCollabChannelOutboundMetadata(c, content, {
      contextMode: workspaceContextMode,
    });
    await api.sendMessage(collabChannel, content, from, 'question', meta);
  };
  const completedTasks = c.tasks?.filter(t => t.status === 'completed').length ?? 0;
  const totalTasks = c.tasks?.length ?? 0;
  const progress = totalTasks > 0 ? Math.round((completedTasks / totalTasks) * 100) : 0;
  const isTerminal = c.phase === 'completed' || c.phase === 'cancelled';
  const planningRecapPending = c.planning_recap_status === 'pending';
  const recapFacilitatorName =
    c.agents?.find(a => a.agent_id === c.planning_recap_agent_id)?.agent_name ?? 'agent';

  useEffect(() => {
    setExtendTargetId(c.id);
  }, [c.id]);

  const handleResume = async () => {
    if (c.phase === 'reviewing' || c.phase === 'approved') {
      if (!confirmReplaceCollaborationExecution(executingCollaboration ?? null, c)) {
        return;
      }
    }
    setIsSubmitting(true);
    try {
      await sendCollabCommand(`/resume-plan ${c.id.slice(0, 8)}`);
      await onAfterCollaborationCommand?.();
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleSubmitForReview = async () => {
    setIsSubmitting(true);
    try {
      await sendCollabCommand(`/submit-plan ${c.id.slice(0, 8)}`);
      await onAfterCollaborationCommand?.();
    } finally {
      setIsSubmitting(false);
    }
  };

  const anotherCollabExecuting =
    executingCollaboration != null &&
    executingCollaboration.phase === 'executing' &&
    executingCollaboration.id !== c.id;
  const awaitingWorkspaceConfirmation = isAwaitingWorkspaceConfirmation(c);
  const primaryActionLabel = collaborationPrimaryActionLabel(c.phase, {
    anotherCollabExecuting,
    awaitingWorkspaceConfirmation,
  });
  const showPrimaryAction =
    primaryActionLabel != null &&
    (c.phase === 'reviewing' || c.phase === 'approved' || c.phase === 'executing');
  const approveBlocked = c.phase === 'reviewing' && planningRecapPending;
  const planningAwaitingFirstTurn = isPlanningAwaitingFirstTurn(c);
  const planningStalled = planningStalledParticipantNames(c);
  const approvedAwaitingDispatch = isApprovedAwaitingDispatch(c);
  const chatOnlyCompletedFileTasks =
    c.tasks?.filter(
      (t) =>
        t.status === 'completed' &&
        taskNeedsFileDeliverable(t) &&
        !(t.output?.trim())
    ) ?? [];
  const executingStuck =
    c.phase === 'executing' &&
    c.workspace_acknowledged &&
    totalTasks > 0 &&
    completedTasks === 0 &&
    (c.tasks?.some((t) => t.status === 'in_progress') ?? false);
  const submitForReviewEnabled = canSubmitCollaborationForReview(c.phase, c.discussion);
  const submitForReviewBlocked = c.phase === 'planning' && !submitForReviewEnabled;
  const sessionRecapText = (c.session_recap || c.planning_recap || '').trim();
  const showSessionSummaryCard =
    sessionRecapText.length > 0 && !(c.phase === 'reviewing' && planningRecapPending && !c.planning_recap?.trim());
  const setupInProgress =
    isSubmitting &&
    (c.phase === 'reviewing' || c.phase === 'approved' || primaryActionLabel?.toLowerCase().includes('approve'));

  const handleRevise = async () => {
    if (!feedback.trim()) return;
    setIsSubmitting(true);
    try {
      await sendCollabCommand(`/revise-plan ${c.id.slice(0, 8)} ${feedback}`);
      setFeedback('');
      await onAfterCollaborationCommand?.();
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleCancel = async () => {
    setIsSubmitting(true);
    try {
      await sendCollabCommand(`/cancel-plan ${c.id.slice(0, 8)}`);
      await onAfterCollaborationCommand?.();
    } finally {
      setIsSubmitting(false);
    }
  };

  const openTasks = c.tasks?.filter(t => t.status !== 'completed') ?? [];

  const handleMarkDone = async () => {
    if (openTasks.length > 0) {
      const lines = openTasks.map((t, i) => `• ${t.title || `Task ${i + 1}`}`).join('\n');
      const ok = window.confirm(
        `Mark this collaboration done?\n\n${openTasks.length} task(s) are not completed yet:\n${lines}\n\nOpen tasks will be marked complete.`
      );
      if (!ok) return;
    }
    setIsSubmitting(true);
    try {
      const force = openTasks.length > 0 ? ' --force' : '';
      await sendCollabCommand(`/complete-collab ${c.id.slice(0, 8)}${force}`);
      await onAfterCollaborationCommand?.();
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleTaskDone = async (task: CollaborationTask) => {
    setIsSubmitting(true);
    try {
      await api.collabTaskComplete(c.id, task.id);
      await onAfterCollaborationCommand?.();
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleTaskSkip = async (task: CollaborationTask) => {
    setIsSubmitting(true);
    try {
      await api.collabTaskSkip(c.id, task.id);
      await onAfterCollaborationCommand?.();
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleTaskRedispatch = async (task: CollaborationTask) => {
    setIsSubmitting(true);
    try {
      await api.collabTaskRedispatch(c.id, task.id);
      await onAfterCollaborationCommand?.();
    } finally {
      setIsSubmitting(false);
    }
  };

  const handlePauseResume = async () => {
    setIsSubmitting(true);
    try {
      if (c.dispatch_paused) {
        await api.collabResume(c.id);
      } else {
        await api.collabPause(c.id);
      }
      await onAfterCollaborationCommand?.();
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleRename = async () => {
    const title = renameDraft.trim();
    if (!title) return;
    setIsSubmitting(true);
    try {
      await sendCollabCommand(`/collab-rename ${c.id.slice(0, 8)} ${title}`);
      setRenaming(false);
      await onAfterCollaborationCommand?.();
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleExtend = async () => {
    const target = (extendTargetId || c.id).trim();
    if (!target) return;
    const rounds = parseInt(extendRounds, 10);
    const messages = extendMessages.trim() ? parseInt(extendMessages, 10) : 0;
    if (!Number.isFinite(rounds) || rounds <= 0) return;
    if (extendMessages.trim() && (!Number.isFinite(messages) || messages <= 0)) return;
    const prefix = target.length > 8 ? target.slice(0, 8) : target;
    let cmd = `/collab-extend ${prefix} --rounds ${rounds}`;
    if (messages > 0) {
      cmd += ` --messages ${messages}`;
    }
    setIsSubmitting(true);
    try {
      await sendCollabCommand(cmd);
      await onAfterCollaborationCommand?.();
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="collaboration-panel" style={{
      ...shrinkablePanelStyle(380, 240),
      borderLeft: '1px solid var(--border-color, #333)',
      display: 'flex',
      flexDirection: 'column',
      backgroundColor: 'var(--bg-secondary, #1e1e1e)',
      height: '100%',
    }}>
      {/* Header */}
      <div style={{
        padding: '12px 16px',
        borderBottom: '1px solid var(--border-color, #333)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
      }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <span style={{ fontSize: 16 }}>🤝</span>
          <span style={{ fontWeight: 600, fontSize: 14 }}>Collaboration</span>
          <span style={{
            fontSize: 11,
            padding: '2px 8px',
            borderRadius: 10,
            backgroundColor: phaseColors[c.phase] + '22',
            color: phaseColors[c.phase],
            fontWeight: 500,
          }}>
            {phaseLabels[c.phase]}
          </span>
        </div>
        <button
          onClick={onClose}
          style={{
            background: 'none', border: 'none', cursor: 'pointer',
            color: 'var(--text-secondary, #888)', fontSize: 18,
          }}
        >
          ×
        </button>
      </div>

      {/* Content */}
      <div data-testid="collaboration-panel-scroll" style={{ flex: 1, overflow: 'auto', padding: 16 }}>
        {/* Title and description */}
        <div style={{ marginBottom: 16 }}>
          {renaming ? (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
              <input
                type="text"
                value={renameDraft}
                onChange={(e) => setRenameDraft(e.target.value)}
                placeholder="Collaboration title"
                maxLength={120}
                style={{
                  width: '100%',
                  boxSizing: 'border-box',
                  padding: '8px 10px',
                  borderRadius: 6,
                  border: '1px solid var(--border-color, #444)',
                  backgroundColor: 'var(--bg-tertiary, #2a2a2a)',
                  color: 'var(--text-primary, #eee)',
                  fontSize: 13,
                }}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') void handleRename();
                  if (e.key === 'Escape') setRenaming(false);
                }}
              />
              <div style={{ display: 'flex', gap: 8 }}>
                <button
                  type="button"
                  onClick={() => void handleRename()}
                  disabled={isSubmitting || !renameDraft.trim()}
                  style={{
                    padding: '6px 12px',
                    borderRadius: 6,
                    border: 'none',
                    backgroundColor: '#10b981',
                    color: '#fff',
                    fontSize: 12,
                    cursor: 'pointer',
                  }}
                >
                  Save title
                </button>
                <button
                  type="button"
                  onClick={() => setRenaming(false)}
                  style={{
                    padding: '6px 12px',
                    borderRadius: 6,
                    border: '1px solid var(--border-color, #444)',
                    background: 'transparent',
                    color: 'var(--text-secondary, #999)',
                    fontSize: 12,
                    cursor: 'pointer',
                  }}
                >
                  Cancel
                </button>
              </div>
            </div>
          ) : (
            <div style={{ display: 'flex', alignItems: 'flex-start', gap: 8 }}>
              <h3 style={{ margin: 0, fontSize: 15, color: 'var(--text-primary, #eee)', flex: 1 }}>
                {c.title}
              </h3>
              {!isTerminal && (
                <button
                  type="button"
                  onClick={() => {
                    setRenameDraft(c.title);
                    setRenaming(true);
                  }}
                  style={{
                    background: 'none',
                    border: 'none',
                    color: 'var(--text-secondary, #888)',
                    fontSize: 12,
                    cursor: 'pointer',
                    textDecoration: 'underline',
                    flexShrink: 0,
                  }}
                >
                  Rename
                </button>
              )}
            </div>
          )}
          <p style={{ margin: '8px 0 0 0', fontSize: 12, color: 'var(--text-secondary, #999)' }}>
            <code style={{ fontSize: 11 }}>{c.id.slice(0, 8)}</code>
            {isTerminal && c.updated_at && (
              <span style={{ marginLeft: 8 }}>
                Closed {new Date(c.updated_at).toLocaleString()}
              </span>
            )}
          </p>
          <p style={{ margin: '8px 0 0 0', fontSize: 13, color: 'var(--text-secondary, #999)', lineHeight: 1.4 }}>
            {c.description}
          </p>
        </div>

        {(c.planning_recap || c.session_recap) && showSessionSummaryCard && (
          <div
            data-testid="collaboration-session-summary"
            style={{
              marginBottom: 16,
              padding: 12,
              borderRadius: 8,
              border: '1px solid var(--border-color, #444)',
              backgroundColor: 'var(--bg-tertiary, #252525)',
            }}
          >
            <div style={{ fontSize: 12, fontWeight: 600, marginBottom: 8, color: 'var(--text-secondary, #aaa)' }}>
              Session summary
            </div>
            <RichMarkdownView content={sessionRecapText} compact />
          </div>
        )}

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

        {c.phase === 'planning' && planningStalled.length > 0 && (
          <div
            data-testid="collaboration-planning-stall-banner"
            style={{
              marginBottom: 16,
              padding: 12,
              borderRadius: 8,
              border: '1px solid #b45309',
              backgroundColor: 'rgba(180, 83, 9, 0.12)',
            }}
          >
            <div style={{ fontSize: 13, fontWeight: 600, color: '#fcd34d', marginBottom: 6 }}>
              Waiting on {planningStalled.map((n) => `@${n}`).join(', ')}
            </div>
            <p style={{ margin: 0, fontSize: 12, color: 'var(--text-secondary, #ccc)', lineHeight: 1.45 }}>
              These participants have not posted this round. Check agents are online and the model is reachable
              (Ollama: <code style={{ fontSize: 11 }}>ollama serve</code>). Debug:{' '}
              <code style={{ fontSize: 11 }}>make debug-collab COLAB={c.id.slice(0, 8)} LIVE=1</code>
            </p>
          </div>
        )}

        {c.phase === 'planning' && planningAwaitingFirstTurn && (
          <div
            data-testid="collaboration-planning-wait-banner"
            style={{
              marginBottom: 16,
              padding: 12,
              borderRadius: 8,
              border: '1px solid #b45309',
              backgroundColor: 'rgba(180, 83, 9, 0.12)',
            }}
          >
            <div style={{ fontSize: 13, fontWeight: 600, color: '#fcd34d', marginBottom: 6 }}>
              Waiting for the first agent turn
            </div>
            <p style={{ margin: 0, fontSize: 12, color: 'var(--text-secondary, #ccc)', lineHeight: 1.45 }}>
              The hub prompted the first participant. If nothing appears after ~30s, check that agents are online
              or run <code style={{ fontSize: 11 }}>make debug-collab LIVE=1</code>.
            </p>
          </div>
        )}

        {c.phase === 'executing' && executingStuck && (
          <div
            data-testid="collaboration-executing-stuck-banner"
            style={{
              marginBottom: 16,
              padding: 12,
              borderRadius: 8,
              border: '1px solid #b45309',
              backgroundColor: 'rgba(180, 83, 9, 0.12)',
            }}
          >
            <div style={{ fontSize: 13, fontWeight: 600, color: '#fcd34d', marginBottom: 6 }}>
              Execution may be stuck
            </div>
            <p style={{ margin: 0, fontSize: 12, color: 'var(--text-secondary, #ccc)', lineHeight: 1.45 }}>
              No tasks completed yet while one is in progress. Use <strong>Resume plan</strong> if the watchdog
              stopped dispatching, or check hub logs and{' '}
              <code style={{ fontSize: 11 }}>make debug-collab COLAB={c.id.slice(0, 8)} LIVE=1</code>.
            </p>
          </div>
        )}

        {c.phase === 'executing' && chatOnlyCompletedFileTasks.length > 0 && (
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
              {chatOnlyCompletedFileTasks.length} task(s) marked completed without file output. Approve pending
              file changes in chat or ask the assignee to emit a [FILE_CHANGE] block. See{' '}
              <code style={{ fontSize: 11 }}>docs/COLLABORATION.md</code> troubleshooting.
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

        {/* Participants */}
        <div style={{ marginBottom: 16 }}>
          <h4 style={{ margin: '0 0 8px 0', fontSize: 12, textTransform: 'uppercase', color: 'var(--text-secondary, #888)', letterSpacing: 0.5 }}>
            Participants
          </h4>
          {c.agents.map(agent => (
            <div key={agent.agent_id} style={{
              display: 'flex', alignItems: 'center', gap: 8,
              padding: '4px 0', fontSize: 13,
            }}>
              <span style={{ fontWeight: 500, color: 'var(--text-primary, #eee)' }}>@{agent.agent_name}</span>
              <span style={{ color: 'var(--text-secondary, #888)', fontSize: 12 }}>{agent.role}</span>
            </div>
          ))}
        </div>

        {/* Progress bar (during execution) */}
        {c.phase === 'executing' && totalTasks > 0 && (
          <div style={{ marginBottom: 16 }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
              <span style={{ fontSize: 12, color: 'var(--text-secondary, #888)' }}>Progress</span>
              <span style={{ fontSize: 12, color: 'var(--text-secondary, #888)' }}>{completedTasks}/{totalTasks} tasks ({progress}%)</span>
            </div>
            <div style={{
              height: 6, borderRadius: 3,
              backgroundColor: 'var(--bg-tertiary, #2a2a2a)',
              overflow: 'hidden',
            }}>
              <div style={{
                height: '100%', borderRadius: 3,
                width: `${progress}%`,
                backgroundColor: '#8b5cf6',
                transition: 'width 0.3s ease',
              }} />
            </div>
          </div>
        )}

        {/* Tasks */}
        {c.tasks && c.tasks.length > 0 && (
          <div style={{ marginBottom: 16 }}>
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 8 }}>
              <h4 style={{ margin: 0, fontSize: 12, textTransform: 'uppercase', color: 'var(--text-secondary, #888)', letterSpacing: 0.5 }}>
                Tasks
              </h4>
              <button
                type="button"
                onClick={() => setGraphOpen(true)}
                style={{
                  padding: '4px 8px',
                  fontSize: 11,
                  borderRadius: 4,
                  border: '1px solid var(--border-color, #444)',
                  background: 'transparent',
                  color: 'var(--text-primary, #ccc)',
                  cursor: 'pointer',
                }}
              >
                View graph
              </button>
            </div>
            {c.approve_warnings && c.approve_warnings.length > 0 ? (
              <div
                style={{
                  marginBottom: 10,
                  padding: '8px 10px',
                  borderRadius: 6,
                  border: '1px solid #b45309',
                  backgroundColor: 'rgba(180, 83, 9, 0.12)',
                  fontSize: 12,
                  color: '#fcd34d',
                }}
              >
                <div style={{ fontWeight: 600, marginBottom: 4 }}>Plan validation</div>
                <ul style={{ margin: 0, paddingLeft: 18 }}>
                  {c.approve_warnings.map((w) => (
                    <li key={w}>{w}</li>
                  ))}
                </ul>
              </div>
            ) : null}
            {c.tasks.map((task: CollaborationTask, i: number) => (
              <div key={task.id} style={{
                padding: '8px 10px', marginBottom: 6,
                borderRadius: 6,
                backgroundColor: 'var(--bg-tertiary, #2a2a2a)',
                fontSize: 13,
              }}>
                <div style={{ display: 'flex', alignItems: 'flex-start', gap: 6 }}>
                  <span>{taskIcon(task.status)}</span>
                  <div style={{ flex: 1 }}>
                    <div style={{ fontWeight: 500, color: 'var(--text-primary, #eee)' }}>
                      Task {i + 1}: {task.title}
                    </div>
                    <div style={{ fontSize: 12, color: 'var(--text-secondary, #999)', marginTop: 2 }}>
                      Assigned to @{task.assigned_name || 'unassigned'}
                    </div>
                    {task.options?.expected_model || task.options?.expected_provider_id ? (
                      <div style={{ fontSize: 11, color: '#a5b4fc', marginTop: 4 }}>
                        Model: {task.options.expected_model || 'agent default'}
                        {task.options.expected_provider_id
                          ? ` · provider ${task.options.expected_provider_id}`
                          : ''}
                        {task.options.routing_reason ? ` · ${task.options.routing_reason}` : ''}
                      </div>
                    ) : null}
                    {c.tasks && taskOrchestrationLabel(task, c.tasks, c.phase) ? (
                      <div style={{ fontSize: 11, color: '#94a3b8', marginTop: 4 }}>
                        {taskOrchestrationLabel(task, c.tasks, c.phase)}
                      </div>
                    ) : null}
                    {c.phase === 'executing' &&
                    task.status === 'in_progress' &&
                    taskNeedsFileDeliverable(task) ? (
                      <div style={{ fontSize: 11, color: '#fbbf24', marginTop: 4, lineHeight: 1.4 }}>
                        File deliverable: assignee must emit a <strong>[FILE_CHANGE]</strong> proposal — approve it in{' '}
                        <strong>Pending changes</strong>. Chat-only replies do not write to disk.
                      </div>
                    ) : null}
                    {!isTerminal && task.status !== 'completed' && c.phase === 'executing' && (
                      <div style={{ marginTop: 6, display: 'flex', flexWrap: 'wrap', gap: 4 }}>
                        <button
                          type="button"
                          onClick={() => void handleTaskDone(task)}
                          disabled={isSubmitting}
                          style={taskActionBtnStyle('#10b981')}
                        >
                          Complete
                        </button>
                        <button
                          type="button"
                          onClick={() => void handleTaskSkip(task)}
                          disabled={isSubmitting}
                          style={taskActionBtnStyle('#94a3b8')}
                        >
                          Skip
                        </button>
                        <button
                          type="button"
                          onClick={() => void handleTaskRedispatch(task)}
                          disabled={isSubmitting}
                          style={taskActionBtnStyle('#3b82f6')}
                        >
                          Redispatch
                        </button>
                      </div>
                    )}
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}

        {/* Plan artifact */}
        {c.plan && c.plan.content && (
          <div style={{ marginBottom: 16 }}>
            <h4 style={{ margin: '0 0 8px 0', fontSize: 12, textTransform: 'uppercase', color: 'var(--text-secondary, #888)', letterSpacing: 0.5 }}>
              Plan (v{c.plan.version})
            </h4>
            <div style={{
              padding: 12, borderRadius: 6,
              backgroundColor: 'var(--bg-tertiary, #2a2a2a)',
              maxHeight: 400, overflow: 'auto',
            }}>
              <RichMarkdownView content={c.plan.content} compact />
            </div>
          </div>
        )}

        {/* Discussion stats */}
        {c.discussion && (
          <div style={{ marginBottom: 16 }}>
            <h4 style={{ margin: '0 0 8px 0', fontSize: 12, textTransform: 'uppercase', color: 'var(--text-secondary, #888)', letterSpacing: 0.5 }}>
              Discussion
            </h4>
            <div style={{
              display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8,
              fontSize: 12, color: 'var(--text-secondary, #999)',
            }}>
              {c.phase === 'planning' || c.phase === 'reviewing' ? (
                <>
                  <div>Round: {c.discussion.current_round}/{c.discussion.max_rounds}</div>
                  <div>Messages: {c.discussion.total_message_count}/{c.discussion.max_total_messages}</div>
                  <div style={{ gridColumn: '1 / -1' }}>Status: {c.discussion.status}</div>
                  {(c.discussion.status === 'budget_exhausted' || c.discussion.status === 'timed_out') && (
                    <div style={{ gridColumn: '1 / -1', marginTop: 8 }}>
                      <div style={{ color: '#fbbf24', fontSize: 12, marginBottom: 8 }}>
                        Discussion limits reached — extend or cancel.
                      </div>
                      <label style={{ display: 'block', fontSize: 11, color: 'var(--text-secondary, #888)', marginBottom: 4 }}>
                        Collaboration
                      </label>
                      <select
                        value={extendTargetId || c.id}
                        onChange={(e) => setExtendTargetId(e.target.value)}
                        style={{
                          width: '100%',
                          marginBottom: 8,
                          padding: '6px 8px',
                          borderRadius: 6,
                          border: '1px solid var(--border-color, #444)',
                          backgroundColor: 'var(--bg-tertiary, #2a2a2a)',
                          color: 'var(--text-primary, #eee)',
                          fontSize: 12,
                        }}
                      >
                        {extendCandidates.map((collab) => (
                          <option key={collab.id} value={collab.id}>
                            {(collab.title || 'Collaboration').slice(0, 48)} ({collab.id.slice(0, 8)})
                          </option>
                        ))}
                      </select>
                      <div style={{ display: 'flex', gap: 8, marginBottom: 8 }}>
                        <label style={{ flex: 1, fontSize: 11, color: 'var(--text-secondary, #888)' }}>
                          + Rounds
                          <input
                            type="number"
                            min={1}
                            value={extendRounds}
                            onChange={(e) => setExtendRounds(e.target.value)}
                            style={{
                              display: 'block',
                              width: '100%',
                              marginTop: 4,
                              padding: '6px 8px',
                              borderRadius: 6,
                              border: '1px solid var(--border-color, #444)',
                              backgroundColor: 'var(--bg-tertiary, #2a2a2a)',
                              color: 'var(--text-primary, #eee)',
                              fontSize: 12,
                            }}
                          />
                        </label>
                        <label style={{ flex: 1, fontSize: 11, color: 'var(--text-secondary, #888)' }}>
                          + Messages (optional)
                          <input
                            type="number"
                            min={1}
                            placeholder="—"
                            value={extendMessages}
                            onChange={(e) => setExtendMessages(e.target.value)}
                            style={{
                              display: 'block',
                              width: '100%',
                              marginTop: 4,
                              padding: '6px 8px',
                              borderRadius: 6,
                              border: '1px solid var(--border-color, #444)',
                              backgroundColor: 'var(--bg-tertiary, #2a2a2a)',
                              color: 'var(--text-primary, #eee)',
                              fontSize: 12,
                            }}
                          />
                        </label>
                      </div>
                      <button
                        type="button"
                        onClick={() => void handleExtend()}
                        disabled={isSubmitting}
                        style={{
                          width: '100%',
                          padding: '8px 12px',
                          borderRadius: 6,
                          border: 'none',
                          backgroundColor: '#f59e0b',
                          color: '#111',
                          fontWeight: 600,
                          fontSize: 12,
                          cursor: 'pointer',
                          opacity: isSubmitting ? 0.6 : 1,
                        }}
                      >
                        Extend discussion
                      </button>
                    </div>
                  )}
                </>
              ) : c.phase === 'executing' ? (
                <div style={{ gridColumn: '1 / -1' }}>
                  Task-driven execution — agents respond to assigned tasks only (no open planning discussion).
                  {c.workspace_acknowledged && c.source_repo_path?.trim() && c.execution_mode !== 'worktree' ? (
                    <span style={{ display: 'block', marginTop: 6, color: 'var(--text-secondary, #9ca3af)', fontSize: 12 }}>
                      Tasks dispatch automatically after approve when a project workspace is bound.
                    </span>
                  ) : null}
                  {c.phase === 'executing' && !c.workspace_acknowledged && (c.working_directory?.trim() || c.execution_mode === 'worktree') ? (
                    <span style={{ display: 'block', marginTop: 6, color: '#fbbf24', fontSize: 12 }}>
                      Confirm the execution workspace (Continue in chat) before agents receive task prompts.
                    </span>
                  ) : null}
                </div>
              ) : (
                <>
                  <div style={{ gridColumn: '1 / -1' }}>Execution — limits off</div>
                  <div style={{ gridColumn: '1 / -1' }}>Messages: {c.discussion.total_message_count}</div>
                  <div style={{ gridColumn: '1 / -1' }}>Status: {c.discussion.status}</div>
                </>
              )}
            </div>
          </div>
        )}
      </div>

      {isTerminal && (
        <div style={{
          padding: '12px 16px',
          borderTop: '1px solid var(--border-color, #333)',
          display: 'flex', flexDirection: 'column', gap: 8,
        }}>
          <p style={{ margin: 0, fontSize: 13, color: 'var(--text-secondary, #aaa)' }}>
            This collaboration is closed. Review tasks and plan above, then dismiss the panel.
          </p>
          <button
            type="button"
            onClick={onClose}
            style={{
              padding: '8px 16px',
              borderRadius: 6,
              border: '1px solid var(--border-color, #444)',
              backgroundColor: 'transparent',
              color: 'var(--text-primary, #eee)',
              fontWeight: 500,
              cursor: 'pointer',
              fontSize: 13,
            }}
          >
            Dismiss
          </button>
        </div>
      )}

      {/* Action buttons */}
      {!isTerminal && (
        <div style={{
          padding: '12px 16px',
          borderTop: '1px solid var(--border-color, #333)',
          display: 'flex', flexDirection: 'column', gap: 8,
        }}>
          {c.phase === 'planning' && (
            <button
              type="button"
              data-testid="collaboration-submit-for-review"
              onClick={() => void handleSubmitForReview()}
              disabled={isSubmitting || submitForReviewBlocked}
              title={collaborationSubmitForReviewTitle(c.discussion)}
              style={{
                padding: '8px 16px',
                borderRadius: 6,
                border: 'none',
                backgroundColor: submitForReviewBlocked ? '#4b5563' : '#3b82f6',
                color: '#fff',
                fontWeight: 600,
                cursor: submitForReviewBlocked ? 'not-allowed' : 'pointer',
                fontSize: 13,
                opacity: isSubmitting || submitForReviewBlocked ? 0.6 : 1,
              }}
            >
              Submit for review
            </button>
          )}
          {showPrimaryAction && primaryActionLabel && (
            <button
              type="button"
              onClick={() => {
                if (awaitingWorkspaceConfirmation && onConfirmWorkspace) {
                  onConfirmWorkspace();
                  return;
                }
                void handleResume();
              }}
              disabled={isSubmitting || approveBlocked}
              title={
                approveBlocked
                  ? `Approve unlocks when the session summary is posted (waiting on @${recapFacilitatorName})`
                  : collaborationPrimaryActionTitle(c.phase, { awaitingWorkspaceConfirmation })
              }
              style={{
                padding: '8px 16px',
                borderRadius: 6,
                border: 'none',
                backgroundColor: c.phase === 'executing' ? '#8b5cf6' : '#10b981',
                color: '#fff',
                fontWeight: 500,
                cursor: 'pointer',
                fontSize: 13,
                opacity: isSubmitting || approveBlocked ? 0.6 : 1,
              }}
            >
              {primaryActionLabel}
            </button>
          )}
          {c.phase === 'reviewing' && (
            <>
              <textarea
                value={feedback}
                onChange={e => setFeedback(e.target.value)}
                placeholder="Feedback for revision… (⌘↵ or Ctrl+↵ to send)"
                rows={4}
                style={{
                  width: '100%',
                  boxSizing: 'border-box',
                  padding: '8px 10px',
                  borderRadius: 6,
                  border: '1px solid var(--border-color, #444)',
                  backgroundColor: 'var(--bg-tertiary, #2a2a2a)',
                  color: 'var(--text-primary, #eee)',
                  fontSize: 13,
                  lineHeight: 1.45,
                  resize: 'vertical',
                  minHeight: 88,
                  fontFamily: 'inherit',
                }}
                onKeyDown={e => {
                  if (e.key !== 'Enter') return;
                  if (!(e.metaKey || e.ctrlKey)) return;
                  e.preventDefault();
                  void handleRevise();
                }}
              />
              <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
                <button
                  type="button"
                  onClick={() => void handleRevise()}
                  disabled={isSubmitting || !feedback.trim()}
                  style={{
                    padding: '6px 12px', borderRadius: 6, border: 'none',
                    backgroundColor: '#3b82f6', color: '#fff',
                    fontWeight: 500, cursor: 'pointer', fontSize: 13,
                    opacity: isSubmitting || !feedback.trim() ? 0.6 : 1,
                  }}
                >
                  Revise
                </button>
              </div>
            </>
          )}
          {c.phase === 'executing' && (
            <button
              type="button"
              onClick={() => void handlePauseResume()}
              disabled={isSubmitting}
              style={{
                padding: '6px 12px', borderRadius: 6,
                border: '1px solid var(--border-color, #444)',
                backgroundColor: 'transparent',
                color: 'var(--text-primary, #ccc)',
                fontWeight: 500, cursor: 'pointer', fontSize: 13,
              }}
            >
              {c.dispatch_paused ? 'Resume dispatch' : 'Pause dispatch'}
            </button>
          )}
          {(c.phase === 'executing' || c.phase === 'reviewing' || c.phase === 'approved') && (
            <button
              type="button"
              onClick={() => void handleMarkDone()}
              disabled={isSubmitting}
              style={{
                padding: '8px 16px',
                borderRadius: 6,
                border: 'none',
                backgroundColor: '#059669',
                color: '#fff',
                fontWeight: 500,
                cursor: 'pointer',
                fontSize: 13,
                opacity: isSubmitting ? 0.6 : 1,
              }}
            >
              Mark collaboration done
            </button>
          )}
          <button
            type="button"
            onClick={() => void handleCancel()}
            disabled={isSubmitting}
            style={{
              padding: '6px 16px', borderRadius: 6,
              border: '1px solid var(--border-color, #444)',
              backgroundColor: 'transparent', color: '#ef4444',
              fontWeight: 500, cursor: 'pointer', fontSize: 13,
              opacity: isSubmitting ? 0.6 : 1,
            }}
          >
            Cancel Collaboration
          </button>
        </div>
      )}

      <RunbookGraphModal
        isOpen={graphOpen}
        collaboration={c}
        agents={c.agents}
        tasks={c.tasks ?? []}
        api={api}
        editable={false}
        onClose={() => setGraphOpen(false)}
        onTasksChange={() => {}}
      />
    </div>
  );
}

function taskActionBtnStyle(color: string): CSSProperties {
  return {
    padding: '4px 8px',
    borderRadius: 4,
    border: '1px solid var(--border-color, #444)',
    background: 'transparent',
    color,
    fontSize: 11,
    cursor: 'pointer',
  };
}
