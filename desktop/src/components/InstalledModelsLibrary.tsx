import { useCallback, useEffect, useMemo, useState } from 'react';
import { ModelStoreBrowse } from './model-library/ModelStoreBrowse';
import type { StoreModelAction, StoreModelItem } from './model-library/types';
import { useModelTransferStore } from '../stores/modelTransferStore';

interface HubProvider {
  id: string;
  type: string;
  name: string;
  model?: string;
  endpoint?: string;
  api_key?: string;
  headers?: Record<string, string>;
  work_dir?: string;
}

interface HfLocalFile {
  repo_id: string;
  filename: string;
  path: string;
  size: number;
  kind?: string;
}

type HfActiveDownload = {
  status?: string;
  repo_id?: string;
  filename?: string;
  percent?: number;
  error?: string;
};

interface InstalledModelsLibraryProps {
  serverAddr: string;
  switchAllAgentProviders: (provider: string, model: string) => Promise<void>;
  onAfterModelChange?: () => void;
  onViewChange?: (view: 'grid' | 'detail') => void;
  resetDetailSignal?: number;
}

function formatBytes(n: number): string {
  if (!Number.isFinite(n) || n <= 0) return '';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  let v = n;
  let i = 0;
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024;
    i += 1;
  }
  return `${v < 10 && i > 0 ? v.toFixed(1) : Math.round(v)} ${units[i]}`;
}

function titleFromRepoId(repoId: string): string {
  const base = repoId.split('/').pop() || repoId;
  return base.replace(/[-_]/g, ' ').replace(/\bgguf\b/gi, '').trim() || repoId;
}

/** True when Ollama already has this pull tag (exact or same family / :tag variants). */
function ollamaTagInstalled(installed: Iterable<string>, tag: string): boolean {
  const want = tag.trim().toLowerCase();
  if (!want) return false;
  const wantBase = want.split(':')[0] ?? want;
  for (const name of installed) {
    const have = name.trim().toLowerCase();
    if (!have) continue;
    if (have === want) return true;
    const haveBase = have.split(':')[0] ?? have;
    if (haveBase === wantBase) return true;
  }
  return false;
}

export function InstalledModelsLibrary({
  serverAddr,
  switchAllAgentProviders,
  onAfterModelChange,
  onViewChange,
  resetDetailSignal,
}: InstalledModelsLibraryProps) {
  const [query, setQuery] = useState('');
  const [ollamaRunning, setOllamaRunning] = useState(false);
  const [ollamaModels, setOllamaModels] = useState<string[]>([]);
  const [hfFiles, setHfFiles] = useState<HfLocalFile[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [actionMessage, setActionMessage] = useState<{ kind: 'ok' | 'err'; text: string } | null>(null);
  const [busyId, setBusyId] = useState<string | null>(null);
  const transfers = useModelTransferStore((s) => s.transfers);
  const transferComplete = useModelTransferStore((s) => s.complete);
  const transferUpdate = useModelTransferStore((s) => s.update);
  const transferFail = useModelTransferStore((s) => s.fail);
  const transferStart = useModelTransferStore((s) => s.start);
  const transferRemove = useModelTransferStore((s) => s.remove);

  const refresh = useCallback(async () => {
    setLoading(true);
    setLoadError(null);
    try {
      const [st, hfR] = await Promise.all([
        fetch(`${serverAddr}/api/ollama/install-status`).then((r) => r.json()),
        fetch(`${serverAddr}/api/hf/local`),
      ]);
      const running = Boolean(st.running);
      setOllamaRunning(running);
      if (running) {
        const mr = await fetch(`${serverAddr}/api/ollama/models`);
        if (mr.ok) {
          const data = await mr.json();
          const raw = data.models as unknown;
          const names: string[] = Array.isArray(raw)
            ? raw
                .map((m) => (typeof m === 'string' ? m : (m as { name?: string }).name))
                .filter((x): x is string => Boolean(x))
            : [];
          setOllamaModels(names.sort((a, b) => a.localeCompare(b)));
        } else {
          setOllamaModels([]);
        }
      } else {
        setOllamaModels([]);
      }
      if (hfR.ok) {
        const data = await hfR.json();
        setHfFiles(Array.isArray(data.files) ? data.files : []);
      } else {
        setHfFiles([]);
      }
    } catch (e) {
      setLoadError(e instanceof Error ? e.message : String(e));
      setOllamaModels([]);
      setHfFiles([]);
    } finally {
      setLoading(false);
    }
  }, [serverAddr]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  // Poll hub-side HF downloads and mirror into the transfer store.
  useEffect(() => {
    let cancelled = false;
    const tick = async () => {
      try {
        const r = await fetch(`${serverAddr}/api/hf/downloads/active`);
        if (!r.ok || cancelled) return;
        const data = (await r.json()) as { downloads?: HfActiveDownload[] };
        const rows = Array.isArray(data.downloads) ? data.downloads : [];
        for (const d of rows) {
          if (!d.repo_id) continue;
          const id = `hf:${d.repo_id}`;
          if (d.status === 'error') {
            transferFail(id, d.error || 'Download failed');
            continue;
          }
          if (d.status === 'success') {
            transferComplete(id);
            continue;
          }
          const progressLabel =
            typeof d.percent === 'number' && d.percent > 0
              ? `${d.filename ? `${d.filename}: ` : ''}${d.percent.toFixed(1)}%`
              : d.status
                ? String(d.status)
                : 'Downloading…';
          const existing = useModelTransferStore.getState().transfers[id];
          if (!existing) {
            transferStart({
              id,
              source: 'huggingface',
              title: titleFromRepoId(d.repo_id),
              subtitle: d.repo_id,
              progressLabel,
              percent: typeof d.percent === 'number' ? d.percent : undefined,
            });
          } else {
            transferUpdate(id, {
              progressLabel,
              percent: typeof d.percent === 'number' ? d.percent : undefined,
            });
          }
        }
        // Drop HF transfer cards once files are on disk and hub has no active job for that repo.
        for (const t of Object.values(useModelTransferStore.getState().transfers)) {
          if (t.source !== 'huggingface') continue;
          const repoId = t.subtitle;
          const onDisk = hfFiles.some((f) => f.repo_id === repoId);
          const stillActive = rows.some((d) => d.repo_id === repoId);
          if (onDisk && !stillActive) {
            transferComplete(t.id);
          }
        }
      } catch {
        /* hub may be restarting */
      }
    };
    void tick();
    const id = setInterval(() => void tick(), 1500);
    return () => {
      cancelled = true;
      clearInterval(id);
    };
  }, [serverAddr, hfFiles, transferComplete, transferFail, transferStart, transferUpdate]);

  // Drop leftover transfer cards once the model is actually installed / on disk.
  useEffect(() => {
    for (const t of Object.values(transfers)) {
      if (t.source === 'ollama' && ollamaTagInstalled(ollamaModels, t.subtitle)) {
        transferComplete(t.id);
      }
      if (
        t.source === 'huggingface' &&
        t.status !== 'downloading' &&
        hfFiles.some((f) => f.repo_id === t.subtitle)
      ) {
        transferComplete(t.id);
      }
    }
  }, [transfers, ollamaModels, hfFiles, transferComplete]);

  // Refresh installed list when a transfer finishes (or fails).
  const transferSignature = Object.values(transfers)
    .map((t) => `${t.id}:${t.status}`)
    .sort()
    .join('|');
  useEffect(() => {
    const downloading = Object.values(transfers).some((t) => t.status === 'downloading');
    if (!downloading) {
      void refresh();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- refresh on transfer lifecycle, not every progress tick
  }, [transferSignature, refresh]);

  async function useOllamaForAgents(tag: string) {
    setBusyId(`ollama:${tag}`);
    setActionMessage(null);
    try {
      const pr = await fetch(`${serverAddr}/api/providers`);
      if (!pr.ok) throw new Error(pr.statusText);
      const providers = (await pr.json()) as HubProvider[];
      const ollama = providers.find((p) => p.type === 'ollama');
      if (!ollama) {
        throw new Error('No Ollama provider in hub config. Add one under AI Providers.');
      }
      const put = await fetch(`${serverAddr}/api/providers/${encodeURIComponent(ollama.id)}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ...ollama, model: tag }),
      });
      if (!put.ok) throw new Error(await put.text());
      await switchAllAgentProviders('ollama', tag);
      setActionMessage({ kind: 'ok', text: `Agents set to Ollama model ${tag}` });
      onAfterModelChange?.();
    } catch (e) {
      setActionMessage({ kind: 'err', text: e instanceof Error ? e.message : String(e) });
    } finally {
      setBusyId(null);
    }
  }

  async function uninstallOllama(tag: string) {
    if (!ollamaRunning) return;
    if (!confirm(`Uninstall Ollama model "${tag}" from this machine?`)) return;
    setBusyId(`ollama:${tag}`);
    setActionMessage(null);
    try {
      const resp = await fetch(`${serverAddr}/api/ollama/delete`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ model: tag }),
      });
      if (!resp.ok) throw new Error(await resp.text());
      setActionMessage({ kind: 'ok', text: `Uninstalled ${tag}` });
      await refresh();
      onAfterModelChange?.();
    } catch (e) {
      setActionMessage({ kind: 'err', text: e instanceof Error ? e.message : String(e) });
    } finally {
      setBusyId(null);
    }
  }

  async function importHfToOllama(repoId: string, filename: string, kind?: string) {
    const id = `hf:${repoId}:${filename}`;
    setBusyId(id);
    setActionMessage(null);
    try {
      const resp = await fetch(`${serverAddr}/api/hf/import-ollama`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          repo_id: repoId,
          filename,
          kind: kind === 'adapter' || kind === 'lora' ? 'adapter' : undefined,
        }),
      });
      if (!resp.ok) throw new Error(await resp.text());
      const data = await resp.json();
      const tag = data.ollama_tag as string;
      const pr = await fetch(`${serverAddr}/api/providers`);
      const providers = (await pr.json()) as HubProvider[];
      const ollama = providers.find((p) => p.type === 'ollama');
      if (ollama && tag) {
        await fetch(`${serverAddr}/api/providers/${encodeURIComponent(ollama.id)}`, {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ ...ollama, model: tag }),
        });
        await switchAllAgentProviders('ollama', tag);
      }
      setActionMessage({
        kind: 'ok',
        text: tag ? `Imported to Ollama as ${tag}` : `Imported ${filename}`,
      });
      await refresh();
      onAfterModelChange?.();
    } catch (e) {
      setActionMessage({ kind: 'err', text: e instanceof Error ? e.message : String(e) });
    } finally {
      setBusyId(null);
    }
  }

  async function uninstallHfRepo(repoId: string, files: HfLocalFile[]) {
    if (
      !confirm(
        `Uninstall Hugging Face cache for "${repoId}" (${files.length} file${files.length === 1 ? '' : 's'})?`
      )
    ) {
      return;
    }
    setBusyId(`hf:${repoId}`);
    setActionMessage(null);
    try {
      for (const f of files) {
        const resp = await fetch(`${serverAddr}/api/hf/delete`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ repo_id: repoId, filename: f.filename }),
        });
        if (!resp.ok) throw new Error(await resp.text());
      }
      setActionMessage({ kind: 'ok', text: `Uninstalled ${repoId}` });
      await refresh();
    } catch (e) {
      setActionMessage({ kind: 'err', text: e instanceof Error ? e.message : String(e) });
    } finally {
      setBusyId(null);
    }
  }

  const storeItems = useMemo((): StoreModelItem[] => {
    const items: StoreModelItem[] = [];
    const installedHfRepos = new Set(hfFiles.map((f) => f.repo_id));

    for (const t of Object.values(transfers).sort((a, b) => b.updatedAt - a.updatedAt)) {
      if (t.status === 'complete') continue;
      // Prefer the real installed row over a leftover transfer card (keep live downloads visible).
      if (t.source === 'ollama' && ollamaTagInstalled(ollamaModels, t.subtitle)) {
        continue;
      }
      if (
        t.source === 'huggingface' &&
        installedHfRepos.has(t.subtitle) &&
        t.status !== 'downloading'
      ) {
        continue;
      }
      const isFailed = t.status === 'error';
      const progress = t.progressLabel || (isFailed ? t.error || 'Failed' : 'Downloading…');
      const dismissAction: StoreModelAction = {
        id: 'dismiss',
        label: isFailed ? 'Dismiss' : 'Cancel from list',
        variant: 'danger',
        onClick: () => transferRemove(t.id),
      };
      items.push({
        id: t.id,
        title: t.title,
        subtitle: t.subtitle,
        description: isFailed
          ? t.error || 'Download failed'
          : t.source === 'ollama'
            ? 'Pulling into local Ollama…'
            : 'Downloading from Hugging Face…',
        tags: [t.source, isFailed ? 'failed' : 'downloading'],
        publisher: t.source === 'ollama' ? 'Ollama' : 'Hugging Face',
        status: 'downloading',
        statusLabel: isFailed ? 'Failed' : 'Downloading',
        sizeHint: typeof t.percent === 'number' ? `${t.percent.toFixed(0)}%` : undefined,
        detailRows: [
          { label: 'Source', value: t.source === 'ollama' ? 'Ollama pull' : 'Hugging Face download' },
          { label: 'Progress', value: progress },
        ],
        primaryAction: isFailed
          ? dismissAction
          : {
              id: 'progress',
              label: progress,
              disabled: true,
              busyLabel: progress,
              onClick: () => undefined,
            },
        detailActions: isFailed
          ? [dismissAction]
          : [
              {
                id: 'progress',
                label: progress,
                disabled: true,
                onClick: () => undefined,
              },
              dismissAction,
            ],
      });
    }

    for (const tag of ollamaModels) {
      const id = `ollama:${tag}`;
      const busy = busyId === id;
      const primaryAction: StoreModelAction = {
        id: 'use',
        label: 'Use for agents',
        disabled: busy,
        busyLabel: busy ? 'Applying…' : undefined,
        onClick: () => void useOllamaForAgents(tag),
      };
      items.push({
        id,
        title: tag,
        subtitle: tag,
        description: 'Installed in local Ollama',
        tags: ['ollama', 'installed'],
        publisher: 'Ollama',
        iconKey: tag.split(/[:/-]/)[0]?.toLowerCase(),
        status: 'installed',
        statusLabel: 'Installed',
        detailRows: [
          { label: 'Source', value: 'Ollama' },
          { label: 'Tag', value: tag },
        ],
        primaryAction,
        detailActions: [
          primaryAction,
          {
            id: 'uninstall',
            label: 'Uninstall',
            variant: 'danger',
            disabled: !ollamaRunning || busy,
            busyLabel: busy ? 'Uninstalling…' : undefined,
            onClick: () => void uninstallOllama(tag),
          },
        ],
      });
    }

    const downloadingHfRepos = new Set(
      Object.values(transfers)
        .filter((t) => t.source === 'huggingface' && t.status === 'downloading')
        .map((t) => t.subtitle)
    );

    const byRepo = new Map<string, HfLocalFile[]>();
    for (const f of hfFiles) {
      const list = byRepo.get(f.repo_id) ?? [];
      list.push(f);
      byRepo.set(f.repo_id, list);
    }

    for (const [repoId, files] of [...byRepo.entries()].sort((a, b) => a[0].localeCompare(b[0]))) {
      if (downloadingHfRepos.has(repoId)) continue;
      const main = files[0];
      const id = `hf:${repoId}`;
      const totalSize = files.reduce((sum, f) => sum + (f.size || 0), 0);
      const kind = (main.kind ?? '').toLowerCase();
      const isAdapter = kind === 'adapter' || kind === 'lora';
      const repoBusy = busyId === id || files.some((f) => busyId === `hf:${repoId}:${f.filename}`);
      const primaryAction: StoreModelAction = {
        id: 'import',
        label: isAdapter ? 'Compose & import' : 'Import to Ollama',
        disabled: !ollamaRunning || repoBusy,
        busyLabel: repoBusy ? (isAdapter ? 'Composing…' : 'Importing…') : undefined,
        onClick: () => void importHfToOllama(repoId, main.filename, main.kind),
      };
      items.push({
        id,
        title: titleFromRepoId(repoId),
        subtitle: repoId,
        description: isAdapter
          ? 'LoRA adapter cached from Hugging Face'
          : 'GGUF cached from Hugging Face — import into Ollama to use with agents',
        tags: ['huggingface', isAdapter ? 'adapter' : 'gguf', 'on_disk'],
        sizeHint: formatBytes(totalSize) || undefined,
        publisher: 'Hugging Face',
        status: 'on_disk',
        statusLabel: 'On disk',
        externalUrl: `https://huggingface.co/${repoId}`,
        detailRows: [
          { label: 'Source', value: 'Hugging Face cache' },
          { label: 'Repository', value: repoId },
          ...files.map((f) => ({
            label: f.filename,
            value: formatBytes(f.size) || f.path,
          })),
        ],
        primaryAction,
        detailActions: [
          primaryAction,
          {
            id: 'uninstall',
            label: 'Uninstall',
            variant: 'danger',
            disabled: repoBusy,
            busyLabel: repoBusy ? 'Uninstalling…' : undefined,
            onClick: () => void uninstallHfRepo(repoId, files),
          },
        ],
      });
    }

    const q = query.trim().toLowerCase();
    if (!q) return items;
    return items.filter((item) => {
      const hay = `${item.title} ${item.subtitle} ${item.description} ${item.publisher ?? ''} ${item.tags.join(' ')}`.toLowerCase();
      return hay.includes(q);
    });
  }, [ollamaModels, hfFiles, ollamaRunning, busyId, query, transfers, transferRemove]);

  const downloadingCount = Object.values(transfers).filter((t) => t.status === 'downloading').length;

  const banner = (
    <>
      <div className="flex items-center justify-between gap-2 flex-wrap">
        <h3 className="text-sm font-semibold text-gray-300">Installed on this machine</h3>
        <p className="text-xs text-gray-500">
          {ollamaModels.length} Ollama · {hfFiles.length} HF file{hfFiles.length === 1 ? '' : 's'}
          {downloadingCount > 0 ? ` · ${downloadingCount} downloading` : ''}
          {!ollamaRunning && ' · Ollama not running'}
        </p>
      </div>
      <p className="text-xs text-gray-500">
        Downloads appear here while in progress. Use Uninstall to remove Ollama tags or HF cache files.
      </p>
      {loadError && <p className="text-sm text-red-400">{loadError}</p>}
      {loading && downloadingCount === 0 && (
        <p className="text-xs text-gray-500">Loading installed models…</p>
      )}
      {actionMessage && (
        <p className={`text-sm ${actionMessage.kind === 'ok' ? 'text-green-400' : 'text-red-400'}`}>
          {actionMessage.text}
        </p>
      )}
    </>
  );

  return (
    <ModelStoreBrowse
      items={storeItems}
      query={query}
      onQueryChange={setQuery}
      searchPlaceholder="Search installed models…"
      onViewChange={onViewChange}
      resetDetailSignal={resetDetailSignal}
      emptyMessage={
        loading
          ? 'Loading…'
          : 'Nothing installed yet. Pull from Ollama or download a GGUF from Hugging Face.'
      }
      banner={banner}
      headerRight={
        <button
          type="button"
          onClick={() => void refresh()}
          className="px-3 py-1.5 text-xs bg-gray-700 text-gray-300 rounded-lg hover:bg-gray-600 shrink-0"
        >
          Refresh
        </button>
      }
    />
  );
}
