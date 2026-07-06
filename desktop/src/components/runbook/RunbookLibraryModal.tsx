import { useCallback, useEffect, useState } from 'react';
import type { ChatAPI } from '../../api/chatAPI';
import type { CollaborationAgent, RunbookDefinitionSummary } from '../../types/protocol';

interface RunbookLibraryModalProps {
  isOpen: boolean;
  api: ChatAPI;
  hubAgents: CollaborationAgent[];
  channel: string;
  username: string;
  onClose: () => void;
  onInstantiated: (collabId: string, channel: string) => void;
  onNewBlank: () => void;
}

export function RunbookLibraryModal({
  isOpen,
  api,
  hubAgents,
  channel,
  username,
  onClose,
  onInstantiated,
  onNewBlank,
}: RunbookLibraryModalProps) {
  const [defs, setDefs] = useState<RunbookDefinitionSummary[]>([]);
  const [packRunbooks, setPackRunbooks] = useState<{ pack_id: string; path: string; title: string }[]>([]);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  const load = useCallback(async () => {
    setError('');
    try {
      const [list, packs] = await Promise.all([api.listRunbookDefinitions(), api.listPackRunbooks()]);
      setDefs(Array.isArray(list) ? list : []);
      setPackRunbooks(Array.isArray(packs) ? packs : []);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }, [api]);

  useEffect(() => {
    if (isOpen) void load();
  }, [isOpen, load]);

  const importPackRunbook = async (packId: string, path: string) => {
    setBusy(true);
    setError('');
    try {
      await api.importPackRunbook(packId, path);
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  };

  const instantiate = async (id: string) => {
    const agentIds = hubAgents.map((a) => a.agent_id).slice(0, 3);
    if (agentIds.length < 1) {
      setError('No agents available');
      return;
    }
    setBusy(true);
    setError('');
    try {
      const result = await api.instantiateRunbookDefinition(id, {
        channel,
        created_by: username || 'user',
        agent_ids: agentIds,
      });
      onInstantiated(result.collaboration_id, result.collaboration_channel);
      onClose();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-[70] flex items-center justify-center bg-black/55 p-4" onClick={onClose}>
      <div
        className="bg-slack-bg border border-slack-border rounded-lg shadow-2xl w-full max-w-lg max-h-[80vh] flex flex-col"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="px-5 py-4 border-b border-slack-border">
          <h2 className="text-lg font-bold text-slack-text">Runbook library</h2>
          <p className="text-xs text-slack-textMuted mt-1">Starter example, saved definitions, and pack runbooks</p>
        </div>
        <div className="flex-1 overflow-y-auto p-4 space-y-2">
          {error ? <p className="text-xs text-red-400">{error}</p> : null}
          {defs.map((d) => (
            <button
              key={`${d.source}-${d.id}`}
              type="button"
              disabled={busy}
              onClick={() => void instantiate(d.id)}
              className="w-full text-left p-3 rounded border border-slack-border hover:bg-slack-bgHover disabled:opacity-50"
            >
              <div className="font-medium text-sm text-slack-text">{d.title || d.id}</div>
              <div className="text-xs text-slack-textMuted mt-1">
                {d.source} · v{d.version}
                {d.description ? ` — ${d.description.slice(0, 80)}` : ''}
              </div>
            </button>
          ))}
          {defs.length === 0 && packRunbooks.length === 0 && !error ? (
            <p className="text-sm text-slack-textMuted">
              No saved runbooks yet. Create a new blank runbook, or install a pack that ships runbook definitions.
            </p>
          ) : null}
          {packRunbooks.length > 0 ? (
            <div className="mt-4 pt-4 border-t border-slack-border">
              <h3 className="text-xs font-semibold text-slack-textMuted uppercase tracking-wide mb-2">Pack runbooks</h3>
              {packRunbooks.map((p) => (
                <div
                  key={`${p.pack_id}-${p.path}`}
                  className="flex items-center justify-between gap-2 p-3 rounded border border-slack-border mb-2"
                >
                  <div>
                    <div className="font-medium text-sm text-slack-text">{p.title}</div>
                    <div className="text-xs text-slack-textMuted">{p.pack_id} · {p.path}</div>
                  </div>
                  <button
                    type="button"
                    disabled={busy}
                    className="text-xs px-2 py-1 rounded border border-slack-border hover:bg-slack-bgHover"
                    onClick={() => void importPackRunbook(p.pack_id, p.path)}
                  >
                    Save to library
                  </button>
                </div>
              ))}
            </div>
          ) : null}
        </div>
        <div className="px-5 py-3 border-t border-slack-border flex justify-between gap-2">
          <button type="button" className="text-sm text-slack-textMuted" onClick={onClose} disabled={busy}>
            Cancel
          </button>
          <button
            type="button"
            className="px-3 py-1.5 rounded bg-[#8b5cf6] text-white text-sm"
            disabled={busy}
            onClick={() => {
              onClose();
              onNewBlank();
            }}
          >
            New blank runbook
          </button>
        </div>
      </div>
    </div>
  );
}
