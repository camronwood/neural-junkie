import { useCallback, useEffect, useState } from 'react';
import type { ChatAPI } from '../../api/chatAPI';
import type { RunbookRunRecord } from '../../types/protocol';

interface RunbookHistoryPanelProps {
  api: ChatAPI;
  definitionId?: string;
  onOpenRun: (collabId: string, channel?: string) => void;
}

export function RunbookHistoryPanel({ api, definitionId, onOpenRun }: RunbookHistoryPanelProps) {
  const [runs, setRuns] = useState<RunbookRunRecord[]>([]);
  const [error, setError] = useState('');
  const [provenance, setProvenance] = useState<string>('');

  const load = useCallback(async () => {
    try {
      setRuns(await api.listRunbookRuns(definitionId));
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }, [api, definitionId]);

  useEffect(() => {
    void load();
  }, [load]);

  const replay = async (collabId: string) => {
    try {
      const result = await api.replayRunbookRun(collabId);
      onOpenRun(result.collaboration_id, result.collaboration_channel);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  };

  const showProvenance = async (collabId: string) => {
    try {
      const p = await api.getRunbookRunProvenance(collabId);
      const defId = (p as { definition_id?: string }).definition_id || '—';
      const ver = (p as { definition_version?: number }).definition_version;
      const events = (p as { event_count?: number }).event_count;
      setProvenance(
        `Run ${collabId.slice(0, 8)}… → definition ${defId}` +
          (ver != null ? ` v${ver}` : '') +
          (events != null ? ` · ${events} events` : '')
      );
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  };

  if (runs.length === 0 && !error) return null;

  return (
    <div className="mt-4 border-t border-slack-border pt-3">
      <h4 className="text-xs font-semibold text-slack-textMuted uppercase tracking-wide mb-2">Run history</h4>
      {error ? <p className="text-xs text-red-400 mb-2">{error}</p> : null}
      {provenance ? <p className="text-xs text-slack-textMuted mb-2">{provenance}</p> : null}
      <ul className="space-y-1 max-h-40 overflow-y-auto">
        {runs.map((r) => (
          <li key={r.id} className="flex items-center justify-between gap-2 text-xs">
            <button type="button" className="text-slack-accent hover:underline truncate" onClick={() => onOpenRun(r.id, r.channel)}>
              Run #{r.run_number} · {r.phase}
            </button>
            <div className="flex gap-2 shrink-0">
              <button type="button" className="text-slack-textMuted hover:text-slack-text" onClick={() => void showProvenance(r.id)}>
                Provenance
              </button>
              <button type="button" className="text-slack-textMuted hover:text-slack-text" onClick={() => void replay(r.id)}>
                Run again
              </button>
            </div>
          </li>
        ))}
      </ul>
    </div>
  );
}
