import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import type { ChatAPI } from '../../api/chatAPI';
import type { CollaborationAgent, RunbookDefinitionSummary } from '../../types/protocol';
import { isTauriRuntime } from '../../utils/promptAttachments';

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
  const fileInputRef = useRef<HTMLInputElement>(null);

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

  const grouped = useMemo(() => {
    const buckets: Record<string, RunbookDefinitionSummary[]> = {
      bundled: [],
      user: [],
      pack: [],
      other: [],
    };
    for (const d of defs) {
      const src = (d.source || 'other').toLowerCase();
      if (src === 'bundled' || src === 'user' || src === 'pack') {
        buckets[src].push(d);
      } else {
        buckets.other.push(d);
      }
    }
    return buckets;
  }, [defs]);

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

  const exportDef = async (id: string, title: string) => {
    setBusy(true);
    setError('');
    try {
      const bundle = await api.exportRunbookDefinition(id);
      const blob = new Blob([JSON.stringify(bundle, null, 2)], { type: 'application/json' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `${(title || id).replace(/[^\w.-]+/g, '_')}-runbook.json`;
      a.click();
      URL.revokeObjectURL(url);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  };

  const importBundleText = async (text: string) => {
    setBusy(true);
    setError('');
    try {
      const parsed = JSON.parse(text) as unknown;
      await api.importRunbookDefinition(parsed as Parameters<ChatAPI['importRunbookDefinition']>[0]);
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  };

  const handleImportClick = async () => {
    setError('');
    if (isTauriRuntime()) {
      try {
        const { open } = await import('@tauri-apps/plugin-dialog');
        const { readTextFile } = await import('@tauri-apps/plugin-fs');
        const selected = await open({
          multiple: false,
          title: 'Import runbook package',
          filters: [{ name: 'JSON', extensions: ['json'] }],
        });
        if (!selected || typeof selected !== 'string') return;
        await importBundleText(await readTextFile(selected));
      } catch (e) {
        setError(e instanceof Error ? e.message : String(e));
      }
      return;
    }
    fileInputRef.current?.click();
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

  const renderGroup = (label: string, items: RunbookDefinitionSummary[]) => {
    if (items.length === 0) return null;
    return (
      <div className="mb-4">
        <h3 className="text-xs font-semibold text-slack-textMuted uppercase tracking-wide mb-2">{label}</h3>
        {items.map((d) => (
          <div
            key={`${d.source}-${d.id}`}
            className="flex items-start justify-between gap-2 p-3 rounded border border-slack-border mb-2"
          >
            <button
              type="button"
              disabled={busy}
              onClick={() => void instantiate(d.id)}
              className="flex-1 text-left disabled:opacity-50"
            >
              <div className="font-medium text-sm text-slack-text">{d.title || d.id}</div>
              <div className="text-xs text-slack-textMuted mt-1">
                {d.source} · v{d.version}
                {d.description ? ` — ${d.description.slice(0, 80)}` : ''}
              </div>
            </button>
            <button
              type="button"
              disabled={busy}
              className="text-xs px-2 py-1 rounded border border-slack-border shrink-0"
              onClick={() => void exportDef(d.id, d.title || d.id)}
            >
              Export
            </button>
          </div>
        ))}
      </div>
    );
  };

  return (
    <div className="fixed inset-0 z-[70] flex items-center justify-center bg-black/55 p-4" onClick={onClose}>
      <div
        className="bg-slack-bg border border-slack-border rounded-lg shadow-2xl w-full max-w-lg max-h-[80vh] flex flex-col"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="px-5 py-4 border-b border-slack-border">
          <h2 className="text-lg font-bold text-slack-text">Runbook library</h2>
          <p className="text-xs text-slack-textMuted mt-1">
            Starter templates, saved definitions, and pack runbooks — export/import packages to share
          </p>
        </div>
        <div className="flex-1 overflow-y-auto p-4 space-y-2">
          {error ? <p className="text-xs text-red-400">{error}</p> : null}
          {renderGroup('Bundled templates', grouped.bundled)}
          {renderGroup('Your runbooks', grouped.user)}
          {renderGroup('From packs', grouped.pack)}
          {renderGroup('Other', grouped.other)}
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
          <div className="flex gap-2">
            <button type="button" className="text-sm text-slack-textMuted" onClick={onClose} disabled={busy}>
              Cancel
            </button>
            <button
              type="button"
              className="text-sm px-2 py-1 rounded border border-slack-border"
              disabled={busy}
              onClick={() => void handleImportClick()}
            >
              Import package
            </button>
            <input
              ref={fileInputRef}
              type="file"
              accept="application/json,.json"
              className="hidden"
              onChange={(e) => {
                const f = e.target.files?.[0];
                if (!f) return;
                void f.text().then(importBundleText);
              }}
            />
          </div>
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
