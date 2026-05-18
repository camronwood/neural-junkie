import { useEffect, useState } from 'react';
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

interface CollaborationPanelProps {
  collaboration: Collaboration;
  /** Active collaborations (planning/review) for extend dropdown. */
  extendableCollaborations?: Collaboration[];
  /** If set and different from `collaboration`, approving will replace that running execution (after user confirms). */
  executingCollaboration?: Collaboration | null;
  onClose: () => void;
  /** Refetch collaboration snapshots after approve/revise/cancel (keeps UI in sync). */
  onAfterCollaborationCommand?: () => Promise<void>;
}

const phaseLabels: Record<CollaborationPhase, string> = {
  planning: 'Planning',
  reviewing: 'Reviewing Plan',
  approved: 'Approved',
  executing: 'Executing',
  completed: 'Completed',
  cancelled: 'Cancelled',
};

const phaseColors: Record<CollaborationPhase, string> = {
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
  const [extendTargetId, setExtendTargetId] = useState('');
  const [extendRounds, setExtendRounds] = useState('1');
  const [extendMessages, setExtendMessages] = useState('');
  const from = { name: username || 'User', type: 'human' };

  const c = collaboration;
  const extendCandidates =
    extendableCollaborations.length > 0
      ? extendableCollaborations
      : c.phase === 'planning' || c.phase === 'reviewing'
        ? [c]
        : [];
  const collabChannel = c.channel?.trim() || channel;
  const completedTasks = c.tasks?.filter(t => t.status === 'completed').length ?? 0;
  const totalTasks = c.tasks?.length ?? 0;
  const progress = totalTasks > 0 ? Math.round((completedTasks / totalTasks) * 100) : 0;
  const isTerminal = c.phase === 'completed' || c.phase === 'cancelled';

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
      await api.sendMessage(collabChannel, `/resume-plan ${c.id.slice(0, 8)}`, from);
      await onAfterCollaborationCommand?.();
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleRevise = async () => {
    if (!feedback.trim()) return;
    setIsSubmitting(true);
    try {
      await api.sendMessage(collabChannel, `/revise-plan ${c.id.slice(0, 8)} ${feedback}`, from);
      setFeedback('');
      await onAfterCollaborationCommand?.();
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleCancel = async () => {
    setIsSubmitting(true);
    try {
      await api.sendMessage(collabChannel, `/cancel-plan ${c.id.slice(0, 8)}`, from);
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
      await api.sendMessage(collabChannel, `/complete-collab ${c.id.slice(0, 8)}${force}`, from);
      await onAfterCollaborationCommand?.();
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleTaskDone = async (taskIndex: number) => {
    setIsSubmitting(true);
    try {
      await api.sendMessage(
        collabChannel,
        `/collab-task-done ${c.id.slice(0, 8)} ${taskIndex + 1}`,
        from
      );
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
      await api.sendMessage(
        collabChannel,
        `/collab-rename ${c.id.slice(0, 8)} ${title}`,
        from
      );
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
      await api.sendMessage(collabChannel, cmd, from);
      await onAfterCollaborationCommand?.();
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="collaboration-panel" style={{
      width: 380,
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
      <div style={{ flex: 1, overflow: 'auto', padding: 16 }}>
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
            <h4 style={{ margin: '0 0 8px 0', fontSize: 12, textTransform: 'uppercase', color: 'var(--text-secondary, #888)', letterSpacing: 0.5 }}>
              Tasks
            </h4>
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
                    {!isTerminal && task.status !== 'completed' && c.phase === 'executing' && (
                      <button
                        type="button"
                        onClick={() => void handleTaskDone(i)}
                        disabled={isSubmitting}
                        style={{
                          marginTop: 6,
                          padding: '4px 8px',
                          borderRadius: 4,
                          border: '1px solid var(--border-color, #444)',
                          background: 'transparent',
                          color: '#10b981',
                          fontSize: 11,
                          cursor: 'pointer',
                          opacity: isSubmitting ? 0.6 : 1,
                        }}
                      >
                        Mark task done
                      </button>
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
              ) : (
                <>
                  <div style={{ gridColumn: '1 / -1' }}>Execution — limits off</div>
                  <div style={{ gridColumn: '1 / -1' }}>Messages: {c.discussion.total_message_count}</div>
                  <div style={{ gridColumn: '1 / -1' }}>Execution Q&A: {c.discussion.status}</div>
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
          {(c.phase === 'reviewing' || c.phase === 'approved' || c.phase === 'executing') && (
            <button
              type="button"
              onClick={() => void handleResume()}
              disabled={isSubmitting}
              title={
                c.phase === 'executing'
                  ? 'Re-send task prompts for open work (pending, in progress, or blocked)'
                  : undefined
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
                opacity: isSubmitting ? 0.6 : 1,
              }}
            >
              {c.phase === 'executing'
                ? 'Resume plan'
                : executingCollaboration &&
                    executingCollaboration.phase === 'executing' &&
                    executingCollaboration.id !== c.id
                  ? 'Resume plan (stop other run)'
                  : 'Resume plan'}
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
    </div>
  );
}
