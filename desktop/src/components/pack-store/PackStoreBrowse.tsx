import { useCallback, useEffect, useMemo, useState } from 'react';
import type { ACEStepStatus, InstallPackLoRAResult } from '../../api/chatAPI';
import { ChatAPI } from '../../api/chatAPI';
import { usePacksStore } from '../../stores/packsStore';
import { PACK_CAP } from '../../stores/packCapabilities';
import { getHubBaseURL } from '../../config/hubUrl';

export function PackStoreBrowse() {
  const catalog = usePacksStore((s) => s.catalog);
  const loading = usePacksStore((s) => s.loading);
  const error = usePacksStore((s) => s.error);
  const fetchPackCatalog = usePacksStore((s) => s.fetchPackCatalog);
  const fetchPacks = usePacksStore((s) => s.fetchPacks);
  const installPack = usePacksStore((s) => s.installPack);
  const installPackLoRAs = usePacksStore((s) => s.installPackLoRAs);
  const uninstallPack = usePacksStore((s) => s.uninstallPack);
  const setPackEnabled = usePacksStore((s) => s.setPackEnabled);
  const hasLoRAAdapters = usePacksStore((s) => s.hasCapability(PACK_CAP.LORA_ADAPTERS));
  const [busyId, setBusyId] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);
  const [loraResults, setLoraResults] = useState<Record<string, InstallPackLoRAResult[]>>({});
  const [aceStepStatus, setAceStepStatus] = useState<ACEStepStatus | null>(null);

  useEffect(() => {
    void fetchPackCatalog();
    void fetchPacks();
  }, [fetchPackCatalog, fetchPacks]);

  const musicPackEnabled = useMemo(
    () => catalog.some((e) => e.id === 'music-creation' && e.installed && e.enabled),
    [catalog],
  );

  useEffect(() => {
    if (!musicPackEnabled) {
      setAceStepStatus(null);
      return;
    }
    let cancelled = false;
    (async () => {
      try {
        const st = await new ChatAPI(getHubBaseURL()).fetchACEStepStatus();
        if (!cancelled) setAceStepStatus(st);
      } catch {
        if (!cancelled) setAceStepStatus(null);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [musicPackEnabled, catalog]);

  const run = useCallback(
    async (packId: string, fn: () => Promise<void>) => {
      setBusyId(packId);
      setActionError(null);
      try {
        await fn();
      } catch (e) {
        setActionError(e instanceof Error ? e.message : 'Pack action failed');
      } finally {
        setBusyId(null);
      }
    },
    [],
  );

  const sorted = useMemo(
    () => [...catalog].sort((a, b) => a.title.localeCompare(b.title)),
    [catalog],
  );

  if (loading && sorted.length === 0) {
    return <p className="text-sm text-gray-400">Loading pack catalog…</p>;
  }

  return (
    <div className="space-y-4">
      {(error || actionError) && (
        <p className="text-sm text-red-400">{actionError ?? error}</p>
      )}
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        {sorted.map((entry) => {
          const busy = busyId === entry.id;
          const loraCount = entry.lora_adapter_count ?? 0;
          const results = loraResults[entry.id];
          let primaryLabel = 'Install';
          let primaryAction: () => void = () => run(entry.id, () => installPack(entry.id));
          if (entry.custom && entry.installed) {
            primaryLabel = entry.enabled ? 'Disable' : 'Enable';
            primaryAction = () =>
              run(entry.id, () => setPackEnabled(entry.id, !entry.enabled));
          } else if (entry.installed && !entry.enabled) {
            primaryLabel = 'Enable';
            primaryAction = () => run(entry.id, () => setPackEnabled(entry.id, true));
          } else if (entry.installed && entry.enabled) {
            primaryLabel = 'Disable';
            primaryAction = () => run(entry.id, () => setPackEnabled(entry.id, false));
          }
          return (
            <article
              key={entry.id}
              className="flex flex-col rounded-xl border border-slack-border bg-slack-bgHover/50 p-4"
            >
              <h4 className="text-sm font-semibold text-white">{entry.title}</h4>
              {entry.publisher && (
                <p className="text-[10px] uppercase tracking-wide text-gray-500 mt-1">{entry.publisher}</p>
              )}
              <p className="text-xs text-gray-400 mt-2 flex-1">{entry.description}</p>
              <p className="text-[10px] text-gray-500 mt-2 font-mono">v{entry.version}</p>
              <div className="flex flex-wrap gap-1 mt-2">
                {entry.custom && (
                  <span className="text-[10px] px-1.5 py-0.5 rounded bg-teal-900/50 text-teal-200">Custom</span>
                )}
                {entry.installed && (
                  <span className="text-[10px] px-1.5 py-0.5 rounded bg-gray-700 text-gray-300">Installed</span>
                )}
                {entry.enabled && (
                  <span className="text-[10px] px-1.5 py-0.5 rounded bg-green-900/50 text-green-300">Enabled</span>
                )}
                {loraCount > 0 && (
                  <span className="text-[10px] px-1.5 py-0.5 rounded bg-purple-900/40 text-purple-200">
                    {loraCount} LoRA{loraCount === 1 ? '' : 's'}
                  </span>
                )}
              </div>
              {entry.requires_packs && entry.requires_packs.length > 0 && (
                <p className="text-[10px] text-amber-400/90 mt-2">
                  Requires: {entry.requires_packs.join(', ')}
                </p>
              )}
              {loraCount > 0 && !entry.installed && (
                <p className="text-[10px] text-gray-500 mt-2">Install pack first to compose pack LoRAs.</p>
              )}
              {results && results.length > 0 && (
                <ul className="mt-2 space-y-1 text-[10px] font-mono">
                  {results.map((r) => (
                    <li
                      key={`${r.repo_id}-${r.ollama_tag}`}
                      className={r.status === 'imported' ? 'text-green-400' : 'text-red-400'}
                    >
                      {r.ollama_tag}: {r.status}
                      {r.error ? ` — ${r.error}` : ''}
                    </li>
                  ))}
                </ul>
              )}
              <div className="flex flex-wrap gap-2 mt-3">
                <button
                  type="button"
                  disabled={busy}
                  onClick={primaryAction}
                  className="flex-1 min-w-[5rem] px-3 py-1.5 text-xs font-medium rounded-lg bg-blue-600 text-white hover:bg-blue-500 disabled:opacity-40"
                >
                  {busy ? '…' : primaryLabel}
                </button>
                {entry.installed && loraCount > 0 && entry.id === 'specialist-tuning' && hasLoRAAdapters && (
                  <button
                    type="button"
                    disabled={busy}
                    onClick={() => {
                      void run(entry.id, async () => {
                        const resp = await installPackLoRAs(entry.id);
                        setLoraResults((prev) => ({ ...prev, [entry.id]: resp.results ?? [] }));
                      });
                    }}
                    className="flex-1 min-w-[5rem] px-3 py-1.5 text-xs font-medium rounded-lg bg-purple-700 text-white hover:bg-purple-600 disabled:opacity-40"
                    title={
                      entry.lora_base_tags?.length
                        ? `Requires base: ${entry.lora_base_tags.join(', ')}`
                        : 'Download and compose pack LoRA adapters in Ollama'
                    }
                  >
                    {busy ? '…' : 'Install LoRAs'}
                  </button>
                )}
                {entry.id === 'music-creation' && entry.installed && entry.enabled && !aceStepStatus?.ready && !aceStepStatus?.demo_mode && (
                  <button
                    type="button"
                    disabled={busy || aceStepStatus?.installing}
                    onClick={() => {
                      if (
                        !window.confirm(
                          'Install ACE-Step 1.5? Downloads several GB and may take 10–30 minutes.',
                        )
                      ) {
                        return;
                      }
                      void run(entry.id, async () => {
                        const resp = await new ChatAPI(getHubBaseURL()).installACEStep();
                        setAceStepStatus(resp.acestep);
                      });
                    }}
                    className="flex-1 min-w-[5rem] px-3 py-1.5 text-xs font-medium rounded-lg bg-indigo-700 text-white hover:bg-indigo-600 disabled:opacity-40"
                    title="Download ACE-Step weights and Python dependencies"
                  >
                    {busy || aceStepStatus?.installing ? '…' : 'Install ACE-Step'}
                  </button>
                )}
                {entry.installed && (
                  <button
                    type="button"
                    disabled={busy}
                    onClick={() => {
                      if (!window.confirm(`Uninstall ${entry.title}? This removes specialists and pack tools.`)) {
                        return;
                      }
                      void run(entry.id, () => uninstallPack(entry.id));
                    }}
                    className="px-3 py-1.5 text-xs font-medium rounded-lg bg-red-900/50 text-red-200 hover:bg-red-800/60 disabled:opacity-40"
                  >
                    Uninstall
                  </button>
                )}
              </div>
            </article>
          );
        })}
      </div>
    </div>
  );
}
