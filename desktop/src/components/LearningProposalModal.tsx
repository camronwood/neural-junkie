import { useEffect, useState } from 'react';
import type { LearningCategory, LearningProposalAction, LearningScope } from '../api/chatAPI';
import { ChatAPI } from '../api/chatAPI';

const CATEGORIES: { value: LearningCategory; label: string }[] = [
  { value: 'preference', label: 'Preference' },
  { value: 'fact', label: 'Fact' },
  { value: 'workflow', label: 'Workflow' },
  { value: 'communication', label: 'Communication' },
];

const SCOPES: { value: LearningScope; label: string }[] = [
  { value: 'agent', label: 'This expert only' },
  { value: 'global', label: 'All experts' },
  { value: 'collaboration', label: 'This collaboration' },
];

interface LearningProposalModalProps {
  isOpen: boolean;
  proposal: LearningProposalAction | null;
  serverAddr: string;
  onClose: () => void;
  onSaved?: (agentId: string) => void;
  /** When set, show collab scope option */
  collaborationId?: string;
  /** Edit existing learning id */
  editLearningId?: string;
}

export function LearningProposalModal({
  isOpen,
  proposal,
  serverAddr,
  onClose,
  onSaved,
  collaborationId,
  editLearningId,
}: LearningProposalModalProps) {
  const [content, setContent] = useState('');
  const [category, setCategory] = useState<LearningCategory>('preference');
  const [scope, setScope] = useState<LearningScope>('agent');
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!isOpen || !proposal) return;
    setContent(proposal.draft?.trim() ?? '');
    setCategory(proposal.category ?? 'preference');
    setScope(proposal.scope ?? 'agent');
    setError(null);
  }, [isOpen, proposal]);

  if (!isOpen || !proposal) {
    return null;
  }

  const handleSave = async () => {
    const trimmed = content.trim();
    if (!trimmed) {
      setError('Enter something to remember.');
      return;
    }
    if (scope === 'collaboration' && !collaborationId && !proposal.collaboration_id) {
      setError('Collaboration scope requires an active collaboration channel.');
      return;
    }
    setSaving(true);
    setError(null);
    try {
      const api = new ChatAPI(serverAddr);
      if (editLearningId) {
        await api.updateLearning(editLearningId, { content: trimmed, category, scope, collaboration_id: collaborationId || proposal.collaboration_id });
      } else {
        await api.createLearning({
          scope,
          agent_id: proposal.agent_id,
          agent_type: proposal.agent_type,
          agent_name: proposal.agent_name,
          collaboration_id: scope === 'collaboration' ? (collaborationId || proposal.collaboration_id) : undefined,
          content: trimmed,
          category,
          source_channel: proposal.source_channel,
          source_message_id: proposal.source_message_id,
        });
      }
      onSaved?.(proposal.agent_id);
      onClose();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to save learning');
    } finally {
      setSaving(false);
    }
  };

  const scopeOptions = SCOPES.filter((s) => {
    if (s.value === 'collaboration') {
      return !!(collaborationId || proposal.collaboration_id);
    }
    return true;
  });

  return (
    <div className="fixed inset-0 z-[100] flex items-center justify-center bg-black/50">
      <div
        className="bg-slack-bg border border-slack-border rounded-lg shadow-xl w-full max-w-lg mx-4 p-6"
        role="dialog"
        aria-labelledby="learning-proposal-title"
      >
        <h2 id="learning-proposal-title" className="text-lg font-semibold text-slack-text mb-1">
          {editLearningId ? 'Edit learning' : `Save learning for ${proposal.agent_name}`}
        </h2>
        <p className="text-sm text-slack-textMuted mb-4">
          Edit and confirm — nothing is saved until you click Save.
        </p>

        <label className="block text-sm text-slack-text mb-1" htmlFor="learning-content">
          Learning
        </label>
        <textarea
          id="learning-content"
          value={content}
          onChange={(e) => setContent(e.target.value)}
          rows={4}
          disabled={saving}
          className="w-full px-3 py-2 text-sm border border-slack-border rounded bg-slack-bg text-slack-text mb-3 resize-y"
          placeholder="e.g. Always use tabs for indentation"
        />

        <label className="block text-sm text-slack-text mb-1" htmlFor="learning-scope">
          Scope
        </label>
        <select
          id="learning-scope"
          value={scope}
          onChange={(e) => setScope(e.target.value as LearningScope)}
          disabled={saving}
          className="w-full px-3 py-2 text-sm border border-slack-border rounded bg-slack-bg text-slack-text mb-3"
        >
          {scopeOptions.map((s) => (
            <option key={s.value} value={s.value}>
              {s.label}
            </option>
          ))}
        </select>

        <label className="block text-sm text-slack-text mb-1" htmlFor="learning-category">
          Category
        </label>
        <select
          id="learning-category"
          value={category}
          onChange={(e) => setCategory(e.target.value as LearningCategory)}
          disabled={saving}
          className="w-full px-3 py-2 text-sm border border-slack-border rounded bg-slack-bg text-slack-text mb-4"
        >
          {CATEGORIES.map((c) => (
            <option key={c.value} value={c.value}>
              {c.label}
            </option>
          ))}
        </select>

        {error && <p className="text-sm text-red-600 mb-3">{error}</p>}

        <div className="flex justify-end gap-2">
          <button
            type="button"
            onClick={onClose}
            disabled={saving}
            className="px-4 py-2 text-sm border border-slack-border rounded text-slack-text hover:bg-slack-bgHover disabled:opacity-50"
          >
            Not now
          </button>
          <button
            type="button"
            onClick={() => void handleSave()}
            disabled={saving}
            className="px-4 py-2 text-sm bg-slack-accent text-white rounded hover:bg-slack-accentHover disabled:opacity-50"
          >
            {saving ? 'Saving…' : 'Save'}
          </button>
        </div>
      </div>
    </div>
  );
}
