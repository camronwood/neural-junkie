import { useCallback, useEffect, useMemo, useState } from 'react';
import { ModelStoreBrowse } from './model-library/ModelStoreBrowse';
import type { StoreModelAction, StoreModelItem } from './model-library/types';

export interface HfCatalogEntry {
  repo_id: string;
  title: string;
  description: string;
  tags: string[];
  size_hint?: string;
  modes: string[];
  icon_key?: string;
  publisher?: string;
  files?: { filename: string; quant?: string; size_hint?: string }[];
}

interface HubProvider {
  id: string;
  type: string;
  name: string;
  endpoint?: string;
  api_key?: string;
  model?: string;
}

interface HfLocalFile {
  repo_id: string;
  filename: string;
  path: string;
  size: number;
}

interface HfModelLibraryProps {
  serverAddr: string;
  switchAllAgentProviders: (provider: string, model: string) => Promise<void>;
  onAfterModelChange?: () => void;
  onViewChange?: (view: 'grid' | 'detail') => void;
  resetDetailSignal?: number;
}

type LibraryTab = 'hosted' | 'local';

async function parseSSEChunks(
  reader: ReadableStreamDefaultReader<Uint8Array>,
  onData: (obj: Record<string, unknown>) => void
): Promise<void> {
  const decoder = new TextDecoder();
  let buffer = '';
  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    buffer += decoder.decode(value, { stream: true });
    const lines = buffer.split('\n');
    buffer = lines.pop() ?? '';
    for (const line of lines) {
      if (!line.startsWith('data: ')) continue;
      const raw = line.slice(6).trim();
      if (!raw) continue;
      try {
        onData(JSON.parse(raw) as Record<string, unknown>);
      } catch {
        /* ignore */
      }
    }
  }
}

export function HfModelLibrary({
  serverAddr,
  switchAllAgentProviders,
  onAfterModelChange,
  onViewChange,
  resetDetailSignal,
}: HfModelLibraryProps) {
  const [tab, setTab] = useState<LibraryTab>('hosted');
  const [catalog, setCatalog] = useState<HfCatalogEntry[]>([]);
  const [catalogError, setCatalogError] = useState<string | null>(null);
  const [hfStatus, setHfStatus] = useState<{ token_configured: boolean; router_reachable: boolean } | null>(null);
  const [localFiles, setLocalFiles] = useState<HfLocalFile[]>([]);
  const [ollamaRunning, setOllamaRunning] = useState(false);
  const [query, setQuery] = useState('');
  const [hfToken, setHfToken] = useState('');
  const [downloadingKey, setDownloadingKey] = useState<string | null>(null);
  const [downloadProgress, setDownloadProgress] = useState('');
  const [importingKey, setImportingKey] = useState<string | null>(null);
  const [actionMessage, setActionMessage] = useState<{ kind: 'ok' | 'err'; text: string } | null>(null);

  const refreshLocal = useCallback(async () => {
    try {
      const r = await fetch(`${serverAddr}/api/hf/local`);
      if (r.ok) {
        const data = await r.json();
        setLocalFiles(Array.isArray(data.files) ? data.files : []);
      }
    } catch {
      setLocalFiles([]);
    }
    try {
      const st = await fetch(`${serverAddr}/api/ollama/install-status`).then((res) => res.json());
      setOllamaRunning(Boolean(st.running));
    } catch {
      setOllamaRunning(false);
    }
  }, [serverAddr]);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const [catR, stR] = await Promise.all([
          fetch(`${serverAddr}/api/hf/catalog`),
          fetch(`${serverAddr}/api/hf/status`),
        ]);
        if (!catR.ok) throw new Error(catR.statusText);
        const rows = (await catR.json()) as HfCatalogEntry[];
        if (!cancelled) {
          setCatalog(Array.isArray(rows) ? rows : []);
          setCatalogError(null);
        }
        if (stR.ok && !cancelled) {
          setHfStatus(await stR.json());
        }
      } catch (e) {
        if (!cancelled) {
          setCatalogError(e instanceof Error ? e.message : String(e));
        }
      }
    })();
    void refreshLocal();
    return () => {
      cancelled = true;
    };
  }, [serverAddr, refreshLocal]);

  const filtered = useMemo(() => {
    const mode = tab === 'hosted' ? 'hosted' : 'local';
    const q = query.trim().toLowerCase();
    return catalog.filter((e) => {
      if (!e.modes?.includes(mode)) return false;
      if (!q) return true;
      const hay = `${e.repo_id} ${e.title} ${e.description} ${e.publisher ?? ''} ${(e.tags || []).join(' ')}`.toLowerCase();
      return hay.includes(q);
    });
  }, [catalog, query, tab]);

  function isDownloaded(entry: HfCatalogEntry): boolean {
    const fn = entry.files?.[0]?.filename;
    if (!fn) return false;
    return localFiles.some((f) => f.repo_id === entry.repo_id && f.filename === fn);
  }

  async function addHostedProvider(entry: HfCatalogEntry) {
    setActionMessage(null);
    const token = hfToken.trim();
    if (!token && !hfStatus?.token_configured) {
      setActionMessage({ kind: 'err', text: 'Enter HF token or configure one in a huggingface provider row.' });
      return;
    }
    const slug = entry.repo_id.replace(/\//g, '-').toLowerCase();
    const id = `hf-${slug}`.slice(0, 48);
    const payload: HubProvider = {
      id,
      type: 'huggingface',
      name: entry.title,
      model: entry.repo_id,
      api_key: token || undefined,
      endpoint: 'https://router.huggingface.co/v1',
    };
    try {
      const resp = await fetch(`${serverAddr}/api/providers`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });
      if (!resp.ok) {
        const t = await resp.text();
        throw new Error(t || resp.statusText);
      }
      setActionMessage({ kind: 'ok', text: `Added provider ${id} for ${entry.repo_id}` });
    } catch (e) {
      setActionMessage({ kind: 'err', text: e instanceof Error ? e.message : String(e) });
    }
  }

  async function useHostedForAgents(entry: HfCatalogEntry) {
    setActionMessage(null);
    try {
      const pr = await fetch(`${serverAddr}/api/providers`);
      if (!pr.ok) throw new Error(pr.statusText);
      const providers = (await pr.json()) as HubProvider[];
      let hf = providers.find((p) => p.type === 'huggingface' && p.model === entry.repo_id);
      if (!hf) {
        await addHostedProvider(entry);
        const pr2 = await fetch(`${serverAddr}/api/providers`);
        const providers2 = (await pr2.json()) as HubProvider[];
        hf = providers2.find((p) => p.type === 'huggingface' && p.model === entry.repo_id);
      }
      if (!hf) {
        throw new Error('Could not find or create huggingface provider');
      }
      const updated = { ...hf, model: entry.repo_id };
      const put = await fetch(`${serverAddr}/api/providers/${encodeURIComponent(hf.id)}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(updated),
      });
      if (!put.ok) throw new Error(await put.text());
      await switchAllAgentProviders('huggingface', entry.repo_id);
      setActionMessage({ kind: 'ok', text: `Agents set to HF model ${entry.repo_id}` });
      onAfterModelChange?.();
    } catch (e) {
      setActionMessage({ kind: 'err', text: e instanceof Error ? e.message : String(e) });
    }
  }

  async function downloadModel(entry: HfCatalogEntry) {
    const filename = entry.files?.[0]?.filename;
    if (!filename) {
      setActionMessage({ kind: 'err', text: 'No GGUF file defined in catalog for this model.' });
      return;
    }
    const key = `${entry.repo_id}:${filename}`;
    setDownloadingKey(key);
    setDownloadProgress('Starting…');
    setActionMessage(null);
    let streamError: string | null = null;
    try {
      const resp = await fetch(`${serverAddr}/api/hf/download`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ repo_id: entry.repo_id, filename }),
      });
      if (!resp.ok) {
        throw new Error(await resp.text());
      }
      const reader = resp.body?.getReader();
      if (!reader) throw new Error('No response body');
      await parseSSEChunks(reader, (data) => {
        if (data.status === 'error' || data.error) {
          streamError = String(data.error || 'Download failed');
          setDownloadProgress(streamError);
          return;
        }
        const pct = data.percent;
        if (typeof pct === 'number' && pct > 0) {
          setDownloadProgress(`${pct.toFixed(1)}%`);
        } else if (typeof data.status === 'string') {
          setDownloadProgress(String(data.status));
        }
      });
      if (streamError) {
        setActionMessage({ kind: 'err', text: streamError });
      } else {
        setActionMessage({ kind: 'ok', text: `Downloaded ${filename}` });
      }
    } catch (e) {
      setActionMessage({ kind: 'err', text: e instanceof Error ? e.message : String(e) });
    } finally {
      setDownloadingKey(null);
      setDownloadProgress('');
      await refreshLocal();
    }
  }

  async function importToOllama(entry: HfCatalogEntry) {
    const filename = entry.files?.[0]?.filename;
    if (!filename) return;
    const key = `${entry.repo_id}:${filename}`;
    setImportingKey(key);
    setActionMessage(null);
    try {
      const resp = await fetch(`${serverAddr}/api/hf/import-ollama`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ repo_id: entry.repo_id, filename }),
      });
      if (!resp.ok) throw new Error(await resp.text());
      const data = await resp.json();
      const tag = data.ollama_tag as string;
      const pr = await fetch(`${serverAddr}/api/providers`);
      const providers = (await pr.json()) as HubProvider[];
      const ollama = providers.find((p) => p.type === 'ollama');
      if (ollama) {
        await fetch(`${serverAddr}/api/providers/${encodeURIComponent(ollama.id)}`, {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ ...ollama, model: tag }),
        });
        await switchAllAgentProviders('ollama', tag);
      }
      setActionMessage({ kind: 'ok', text: `Imported to Ollama as ${tag}` });
      onAfterModelChange?.();
    } catch (e) {
      setActionMessage({ kind: 'err', text: e instanceof Error ? e.message : String(e) });
    } finally {
      setImportingKey(null);
    }
  }

  async function deleteLocal(entry: HfCatalogEntry) {
    const filename = entry.files?.[0]?.filename;
    if (!filename) return;
    try {
      const resp = await fetch(`${serverAddr}/api/hf/delete`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ repo_id: entry.repo_id, filename }),
      });
      if (!resp.ok) throw new Error(await resp.text());
      setActionMessage({ kind: 'ok', text: `Removed cached ${filename}` });
      await refreshLocal();
    } catch (e) {
      setActionMessage({ kind: 'err', text: e instanceof Error ? e.message : String(e) });
    }
  }

  const storeItems = useMemo((): StoreModelItem[] => {
    return filtered.map((entry) => {
      const file = entry.files?.[0];
      const dlKey = file ? `${entry.repo_id}:${file.filename}` : entry.repo_id;
      const downloaded = tab === 'local' && isDownloaded(entry);
      const isHosted = tab === 'hosted';

      const detailRows = file
        ? [
            { label: 'Repository', value: entry.repo_id },
            { label: 'GGUF file', value: file.filename },
            ...(file.quant ? [{ label: 'Quantization', value: file.quant }] : []),
            ...(file.size_hint ? [{ label: 'File size', value: file.size_hint }] : []),
          ]
        : [{ label: 'Repository', value: entry.repo_id }];

      if (entry.files && entry.files.length > 1) {
        for (const f of entry.files.slice(1)) {
          detailRows.push({
            label: f.quant || 'Variant',
            value: `${f.filename}${f.size_hint ? ` (${f.size_hint})` : ''}`,
          });
        }
      }

      let primaryAction: StoreModelAction | undefined;
      const detailActions: StoreModelAction[] = [];

      if (isHosted) {
        primaryAction = {
          id: 'use-hosted',
          label: 'Use for agents',
          onClick: () => void useHostedForAgents(entry),
        };
        detailActions.push(
          {
            id: 'add-provider',
            label: 'Add provider',
            variant: 'secondary',
            onClick: () => void addHostedProvider(entry),
          },
          {
            id: 'use-hosted',
            label: 'Use for agents',
            onClick: () => void useHostedForAgents(entry),
          }
        );
      } else if (file) {
        if (!downloaded) {
          primaryAction = {
            id: 'download',
            label: 'Download',
            disabled: downloadingKey === dlKey,
            busyLabel:
              downloadingKey === dlKey ? downloadProgress || 'Downloading…' : undefined,
            onClick: () => void downloadModel(entry),
          };
          detailActions.push({
            id: 'download',
            label: `Download ${file.quant || 'GGUF'}`,
            disabled: downloadingKey === dlKey,
            busyLabel:
              downloadingKey === dlKey ? downloadProgress || 'Downloading…' : undefined,
            onClick: () => void downloadModel(entry),
          });
        } else {
          primaryAction = {
            id: 'import',
            label: 'Import to Ollama',
            disabled: !ollamaRunning || importingKey === dlKey,
            busyLabel: importingKey === dlKey ? 'Importing…' : undefined,
            onClick: () => void importToOllama(entry),
          };
          detailActions.push(
            {
              id: 'import',
              label: 'Import to Ollama',
              disabled: !ollamaRunning || importingKey === dlKey,
              busyLabel: importingKey === dlKey ? 'Importing…' : undefined,
              onClick: () => void importToOllama(entry),
            },
            {
              id: 'delete',
              label: 'Delete file',
              variant: 'danger',
              onClick: () => void deleteLocal(entry),
            }
          );
        }
      }

      return {
        id: entry.repo_id,
        title: entry.title,
        subtitle: entry.repo_id,
        description: entry.description,
        tags: entry.tags ?? [],
        sizeHint: entry.size_hint,
        publisher: entry.publisher,
        iconKey: entry.icon_key,
        status: isHosted ? 'cloud' : downloaded ? 'on_disk' : 'available',
        statusLabel: isHosted ? 'Cloud' : downloaded ? 'On disk' : undefined,
        externalUrl: `https://huggingface.co/${entry.repo_id}`,
        detailRows,
        primaryAction,
        detailActions,
      };
    });
  }, [
    filtered,
    tab,
    localFiles,
    ollamaRunning,
    downloadingKey,
    downloadProgress,
    importingKey,
    hfStatus,
    hfToken,
  ]);

  const tabSwitcher = (
    <div className="flex rounded-md border border-gray-600 overflow-hidden text-xs shrink-0">
      <button
        type="button"
        onClick={() => setTab('hosted')}
        className={`px-3 py-1.5 ${tab === 'hosted' ? 'bg-blue-600 text-white' : 'bg-gray-800 text-gray-400'}`}
      >
        Hosted (cloud)
      </button>
      <button
        type="button"
        onClick={() => setTab('local')}
        className={`px-3 py-1.5 ${tab === 'local' ? 'bg-blue-600 text-white' : 'bg-gray-800 text-gray-400'}`}
      >
        Download (local)
      </button>
    </div>
  );

  const banner = (
  <>
      <div className="flex items-center justify-between gap-2 flex-wrap">
        <h3 className="text-sm font-semibold text-gray-300">Hugging Face models</h3>
        {tabSwitcher}
      </div>

      {hfStatus && (
        <p className="text-xs text-gray-500">
          Token: {hfStatus.token_configured ? 'configured' : 'not set'} · Router:{' '}
          {hfStatus.router_reachable ? 'reachable' : 'unreachable'}
        </p>
      )}

      {tab === 'hosted' && !hfStatus?.token_configured && (
        <div>
          <label className="block text-xs text-gray-500 mb-1">HF token (for new providers)</label>
          <input
            type="password"
            value={hfToken}
            onChange={(e) => setHfToken(e.target.value)}
            placeholder="hf_… or set HF_TOKEN on the hub"
            className="w-full px-2 py-1.5 text-sm bg-gray-900 border border-gray-600 rounded text-gray-200"
          />
        </div>
      )}

      {tab === 'local' && (
        <p className="text-xs text-amber-500/90">
          Downloads GGUF files, then imports into Ollama. Ollama must be running to import.
          {!ollamaRunning && ' (Ollama is not running.)'}
        </p>
      )}

      {catalogError && <p className="text-sm text-red-400">{catalogError}</p>}
      {actionMessage && (
        <p className={`text-sm ${actionMessage.kind === 'ok' ? 'text-green-400' : 'text-red-400'}`}>
          {actionMessage.text}
        </p>
      )}
    </>
  );

  return (
    <ModelStoreBrowse
      key={tab}
      items={storeItems}
      query={query}
      onQueryChange={setQuery}
      searchPlaceholder="Search catalog…"
      onViewChange={onViewChange}
      resetDetailSignal={resetDetailSignal}
      banner={banner}
      headerRight={
        <button
          type="button"
          onClick={() => void refreshLocal()}
          className="px-3 py-1.5 text-xs bg-gray-700 text-gray-300 rounded-lg hover:bg-gray-600 shrink-0"
        >
          Refresh cache
        </button>
      }
    />
  );
}
