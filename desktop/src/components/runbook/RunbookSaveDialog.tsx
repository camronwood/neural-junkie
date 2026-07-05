import { useState } from 'react';
import type { ChatAPI } from '../../api/chatAPI';
import type { Collaboration, RunbookDefinition } from '../../types/protocol';

interface RunbookSaveDialogProps {
  isOpen: boolean;
  api: ChatAPI;
  collaboration: Collaboration;
  tasks: Collaboration['tasks'];
  executionPolicy: Collaboration['execution_policy'];
  graphLayout: Collaboration['graph_layout'];
  onClose: () => void;
  onSaved: (def: RunbookDefinition) => void;
}

export function RunbookSaveDialog({
  isOpen,
  api,
  collaboration,
  tasks,
  executionPolicy,
  graphLayout,
  onClose,
  onSaved,
}: RunbookSaveDialogProps) {
  const [title, setTitle] = useState(collaboration.description || 'My runbook');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  if (!isOpen) return null;

  const handleSave = async () => {
    setBusy(true);
    setError('');
    try {
      const def: RunbookDefinition = {
        id: collaboration.definition_id || '',
        title: title.trim() || 'Runbook',
        description: collaboration.description,
        version: 0,
        source: 'user',
        agent_ids: collaboration.agents.map((a) => a.agent_id),
        tasks: tasks ?? [],
        execution_policy: executionPolicy,
        graph_layout: graphLayout,
        inputs: [],
      };
      const saved = await api.saveRunbookDefinition(def);
      onSaved(saved);
      onClose();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="fixed inset-0 z-[70] flex items-center justify-center bg-black/55 p-4" onClick={onClose}>
      <div className="bg-slack-bg border border-slack-border rounded-lg p-5 w-full max-w-md" onClick={(e) => e.stopPropagation()}>
        <h3 className="text-lg font-bold text-slack-text mb-3">Save to library</h3>
        {error ? <p className="text-xs text-red-400 mb-2">{error}</p> : null}
        <label className="block text-xs text-slack-textMuted mb-1">Title</label>
        <input
          className="w-full mb-4 px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-sm"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
        />
        <div className="flex justify-end gap-2">
          <button type="button" className="text-sm text-slack-textMuted" onClick={onClose} disabled={busy}>
            Cancel
          </button>
          <button type="button" className="px-3 py-1.5 rounded bg-[#8b5cf6] text-white text-sm disabled:opacity-50" disabled={busy} onClick={() => void handleSave()}>
            {busy ? 'Saving…' : 'Save'}
          </button>
        </div>
      </div>
    </div>
  );
}
