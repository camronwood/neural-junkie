import { useCallback, useEffect, useMemo, useState } from 'react';
import { ModelStoreBrowse } from './model-library/ModelStoreBrowse';
import type { StoreModelAction, StoreModelItem } from './model-library/types';

export interface HfCatalogEntry {
  kind?: string;
  repo_id: string;
  title: string;
  description: string;
  tags: string[];
  size_hint?: string;
  modes: string[];
  icon_key?: string;
  publisher?: string;
  base_ollama_tag?: string;
  default_ollama_tag?: string;
  agent_type?: string;
  files?: { filename: string; quant?: string; size_hint?: string }[];
}

function isAdapterEntry(entry: HfCatalogEntry): boolean {
  const kind = (entry.kind ?? '').toLowerCase();
  return kind === 'adapter' || kind === 'lora';
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
  kind?: string;
}

interface ConfiguredAgent {
  type: string;
  name: string;
  enabled?: boolean;
  model?: string;
}

interface HfModelLibraryProps {
  serverAddr: string;
  switchAllAgentProviders: (provider: string, model: string) => Promise<void>;
  switchAgentProvider?: (agentId: string, provider: string, model: string) => Promise<void>;
  runtimeAgents?: { id: string; name: string; type: string }[];
  onAfterModelChange?: () => void;
  onViewChange?: (view: 'grid' | 'detail') => void;
  resetDetailSignal?: number;
  canComposeLoRA?: boolean;
}

type LibraryTab = 'hosted' | 'local';

type DownloadProgressRow = {
  status?: string;
  repo_id?: string;
  filename?: string;
  percent?: number;
  error?: string;
};

interface HfSearchHit {
  repo_id: string;
  title: string;
  description?: string;
  tags?: string[];
  modes?: string[];
  files?: { filename: string; quant?: string; size_hint?: string }[];
}

interface HfSearchResponse {
  query: string;
  mode: string;
  models: HfSearchHit[];
  has_more: boolean;
  offset: number;
}

function mergeHfCatalog(curated: HfCatalogEntry[], hits: HfSearchHit[], mode: LibraryTab): HfCatalogEntry[] {
  const out = curated.filter((e) => e.modes?.includes(mode));
  const seen = new Set(out.map((e) => e.repo_id));
  for (const hit of hits) {
    if (seen.has(hit.repo_id)) continue;
    if (hit.modes && !hit.modes.includes(mode)) continue;
    seen.add(hit.repo_id);
    out.push({
      repo_id: hit.repo_id,
      title: hit.title,
      description: hit.description || hit.repo_id,
      tags: hit.tags ?? [],
      modes: hit.modes ?? [mode],
      files: hit.files,
    });
  }
  return out;
}

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
  switchAgentProvider,
  runtimeAgents = [],
  onAfterModelChange,
  onViewChange,
  resetDetailSignal,
  canComposeLoRA = true,
}: HfModelLibraryProps) {
  const [tab, setTab] = useState<LibraryTab>('hosted');
  const [curated, setCurated] = useState<HfCatalogEntry[]>([]);
  const [searchHits, setSearchHits] = useState<HfSearchHit[]>([]);
  const [searchOffset, setSearchOffset] = useState(0);
  const [searchHasMore, setSearchHasMore] = useState(false);
  const [searchLoading, setSearchLoading] = useState(false);
  const [catalogError, setCatalogError] = useState<string | null>(null);
  const [searchError, setSearchError] = useState<string | null>(null);
  const [hfStatus, setHfStatus] = useState<{ token_configured: boolean; router_reachable: boolean } | null>(null);
  const [localFiles, setLocalFiles] = useState<HfLocalFile[]>([]);
  const [ollamaRunning, setOllamaRunning] = useState(false);
  const [query, setQuery] = useState('');
  const [debouncedQuery, setDebouncedQuery] = useState('');
  const [hfToken, setHfToken] = useState('');
  const [downloadingKey, setDownloadingKey] = useState<string | null>(null);
  const [downloadProgress, setDownloadProgress] = useState('');
  const [importingKey, setImportingKey] = useState<string | null>(null);
  const [actionMessage, setActionMessage] = useState<{ kind: 'ok' | 'err'; text: string } | null>(null);
  const [configuredAgents, setConfiguredAgents] = useState<ConfiguredAgent[]>([]);
  const [ollamaModels, setOllamaModels] = useState<string[]>([]);

  useEffect(() => {
    const id = setTimeout(() => setDebouncedQuery(query.trim()), 300);
    return () => clearTimeout(id);
  }, [query]);

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
      if (st.running) {
        const modelsR = await fetch(`${serverAddr}/api/ollama/models`);
        if (modelsR.ok) {
          const data = await modelsR.json();
          const names = Array.isArray(data.models)
            ? data.models.map((m: string | { name?: string }) => (typeof m === 'string' ? m : m.name)).filter(Boolean)
            : [];
          setOllamaModels(names as string[]);
        }
      } else {
        setOllamaModels([]);
      }
    } catch {
      setOllamaRunning(false);
      setOllamaModels([]);
    }
    try {
      const agentsR = await fetch(`${serverAddr}/api/agents/configured`);
      if (agentsR.ok) {
        const rows = await agentsR.json();
        setConfiguredAgents(Array.isArray(rows) ? rows : []);
      }
    } catch {
      setConfiguredAgents([]);
    }
  }, [serverAddr]);

  const syncActiveDownloads = useCallback(async () => {
    try {
      const r = await fetch(`${serverAddr}/api/hf/downloads/active`);
      if (!r.ok) return;
      const data = (await r.json()) as { downloads?: DownloadProgressRow[] };
      const rows = Array.isArray(data.downloads) ? data.downloads : [];
      if (rows.length === 0) {
        return;
      }
      const d = rows[0];
      if (!d.repo_id || !d.filename) return;
      const key = `${d.repo_id}:${d.filename}`;
      if (d.status === 'error') {
        setDownloadingKey(null);
        setDownloadProgress('');
        setActionMessage({ kind: 'err', text: d.error || 'Download failed' });
        await refreshLocal();
        return;
      }
      setDownloadingKey(key);
      if (typeof d.percent === 'number' && d.percent > 0) {
        setDownloadProgress(`${d.percent.toFixed(1)}%`);
      } else if (d.status) {
        setDownloadProgress(String(d.status));
      }
    } catch {
      /* hub may be restarting */
    }
  }, [serverAddr, refreshLocal]);

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
          setCurated(Array.isArray(rows) ? rows : []);
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
    void syncActiveDownloads();
    return () => {
      cancelled = true;
    };
  }, [serverAddr, refreshLocal, syncActiveDownloads]);

  const fetchSearch = useCallback(
    async (searchQuery: string, mode: LibraryTab, offset: number, append: boolean) => {
      setSearchLoading(true);
      setSearchError(null);
      try {
        const params = new URLSearchParams({
          mode,
          limit: '24',
          offset: String(offset),
        });
        if (searchQuery) params.set('q', searchQuery);
        const r = await fetch(`${serverAddr}/api/hf/search?${params}`);
        if (!r.ok) throw new Error(await r.text());
        const data = (await r.json()) as HfSearchResponse;
        setSearchHits((prev) => (append ? [...prev, ...(data.models ?? [])] : data.models ?? []));
        setSearchOffset(data.offset ?? offset);
        setSearchHasMore(Boolean(data.has_more));
      } catch (e) {
        setSearchError(e instanceof Error ? e.message : 'HF search failed');
        if (!append) setSearchHits([]);
        setSearchHasMore(false);
      } finally {
        setSearchLoading(false);
      }
    },
    [serverAddr]
  );

  useEffect(() => {
    void fetchSearch(debouncedQuery, tab, 0, false);
  }, [debouncedQuery, tab, fetchSearch]);

  useEffect(() => {
    const id = setInterval(() => {
      void syncActiveDownloads();
    }, 2000);
    return () => clearInterval(id);
  }, [syncActiveDownloads]);

  const catalog = useMemo(
    () => mergeHfCatalog(curated, searchHits, tab),
    [curated, searchHits, tab]
  );

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return catalog;
    return catalog.filter((e) => {
      const hay = `${e.repo_id} ${e.title} ${e.description} ${e.publisher ?? ''} ${(e.tags || []).join(' ')}`.toLowerCase();
      return hay.includes(q);
    });
  }, [catalog, query]);

  async function ensureEntryFiles(entry: HfCatalogEntry): Promise<HfCatalogEntry> {
    if (entry.files?.length) return entry;
    const kind = isAdapterEntry(entry) ? 'adapter' : '';
    const qs = kind ? `&kind=${encodeURIComponent(kind)}` : '';
    const r = await fetch(`${serverAddr}/api/hf/files?repo_id=${encodeURIComponent(entry.repo_id)}${qs}`);
    if (!r.ok) throw new Error(await r.text());
    const data = (await r.json()) as { files?: HfCatalogEntry['files'] };
    const files = data.files ?? [];
    if (files.length === 0) {
      throw new Error(
        isAdapterEntry(entry)
          ? `No LoRA adapter files found for ${entry.repo_id}`
          : `No GGUF files found for ${entry.repo_id}`
      );
    }
    return { ...entry, files };
  }

  function ollamaHasBase(baseTag: string): boolean {
    const want = baseTag.trim();
    if (!want) return true;
    return ollamaModels.some(
      (m) => m === want || m === `${want}:latest` || m.startsWith(`${want}:`)
    );
  }

  async function assignComposedTagToAgent(entry: HfCatalogEntry, tag: string) {
    const agentType = entry.agent_type?.trim();
    if (!agentType || !switchAgentProvider) return false;
    const cfg = configuredAgents.find((a) => a.type === agentType);
    const runtime = runtimeAgents.find((a) => a.type === agentType || a.name === cfg?.name);
    if (!runtime) return false;
    await switchAgentProvider(runtime.id, 'ollama', tag);
    return true;
  }

  function isDownloaded(entry: HfCatalogEntry): boolean {
    const fn = entry.files?.[0]?.filename;
    if (fn) {
      return localFiles.some((f) => f.repo_id === entry.repo_id && f.filename === fn);
    }
    return localFiles.some((f) => f.repo_id === entry.repo_id);
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

  async function pollDownloadStatus(repoId: string, filename: string): Promise<DownloadProgressRow | null> {
    const q = new URLSearchParams({ repo_id: repoId, filename });
    const r = await fetch(`${serverAddr}/api/hf/download/status?${q}`);
    if (!r.ok) return null;
    return (await r.json()) as DownloadProgressRow;
  }

  async function downloadModel(entry: HfCatalogEntry) {
    let resolved = entry;
    try {
      resolved = await ensureEntryFiles(entry);
    } catch (e) {
      setActionMessage({ kind: 'err', text: e instanceof Error ? e.message : String(e) });
      return;
    }
    const filename = resolved.files?.[0]?.filename;
    if (!filename) {
      setActionMessage({ kind: 'err', text: 'No GGUF file available for this model.' });
      return;
    }
    const key = `${resolved.repo_id}:${filename}`;
    setDownloadingKey(key);
    setDownloadProgress('Starting…');
    setActionMessage(null);

    const st = await pollDownloadStatus(resolved.repo_id, filename);
    if (st?.status === 'success') {
      setActionMessage({ kind: 'ok', text: `${filename} is already on disk.` });
      setDownloadingKey(null);
      setDownloadProgress('');
      await refreshLocal();
      return;
    }

    let streamError: string | null = null;
    try {
      const resp = await fetch(`${serverAddr}/api/hf/download`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ repo_id: resolved.repo_id, filename }),
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
        const done = await pollDownloadStatus(resolved.repo_id, filename);
        if (done?.status === 'success') {
          setActionMessage({ kind: 'ok', text: `Downloaded ${filename}` });
        }
      }
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e);
      const still = await pollDownloadStatus(resolved.repo_id, filename);
      if (still?.status === 'downloading' || still?.status === 'starting' || still?.status === 'queued') {
        setActionMessage({
          kind: 'ok',
          text: 'Download continues on the hub in the background. Reopen Model Library to see progress.',
        });
      } else if (still?.status === 'success') {
        setActionMessage({ kind: 'ok', text: `Downloaded ${filename}` });
      } else {
        setActionMessage({ kind: 'err', text: msg });
      }
    } finally {
      const finalSt = await pollDownloadStatus(resolved.repo_id, filename);
      if (finalSt?.status === 'success' || finalSt?.status === 'error' || finalSt?.status === 'idle') {
        setDownloadingKey(null);
        setDownloadProgress('');
      }
      await refreshLocal();
    }
  }

  async function importToOllama(entry: HfCatalogEntry) {
    let resolved = entry;
    try {
      resolved = await ensureEntryFiles(entry);
    } catch (e) {
      setActionMessage({ kind: 'err', text: e instanceof Error ? e.message : String(e) });
      return;
    }
    const filename = resolved.files?.[0]?.filename;
    if (!filename) return;
    const adapter = isAdapterEntry(resolved);
    if (adapter && !canComposeLoRA) {
      setActionMessage({
        kind: 'err',
        text: 'Enable Specialist tuning pack to compose LoRA adapters.',
      });
      return;
    }
    const baseTag = resolved.base_ollama_tag || 'qwen2.5-coder:14b';
    if (adapter && !ollamaHasBase(baseTag)) {
      setActionMessage({
        kind: 'err',
        text: `Base model ${baseTag} is not in Ollama. Pull it from the Ollama tab first.`,
      });
      return;
    }
    const key = `${resolved.repo_id}:${filename}`;
    setImportingKey(key);
    setActionMessage(null);
    try {
      const resp = await fetch(`${serverAddr}/api/hf/import-ollama`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          repo_id: resolved.repo_id,
          filename,
          kind: adapter ? 'adapter' : undefined,
          base_ollama_tag: adapter ? baseTag : undefined,
          ollama_tag: adapter ? resolved.default_ollama_tag : undefined,
        }),
      });
      if (!resp.ok) throw new Error(await resp.text());
      const data = await resp.json();
      const tag = data.ollama_tag as string;
      const assigned = adapter ? await assignComposedTagToAgent(resolved, tag) : false;
      if (!assigned) {
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
      }
      setActionMessage({
        kind: 'ok',
        text: assigned
          ? `Composed ${tag} and assigned to ${resolved.agent_type ?? 'specialist'}`
          : adapter
            ? `Composed LoRA tag ${tag} in Ollama`
            : `Imported to Ollama as ${tag}`,
      });
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

      const adapter = isAdapterEntry(entry);
      const adapterComposeBlocked = adapter && !canComposeLoRA;
      const detailRows = file
        ? [
            { label: 'Repository', value: entry.repo_id },
            ...(adapter
              ? [
                  { label: 'Kind', value: 'LoRA adapter' },
                  { label: 'Base model', value: entry.base_ollama_tag ?? 'qwen2.5-coder:14b' },
                  { label: 'Adapter file', value: file.filename },
                ]
              : [
                  { label: 'GGUF file', value: file.filename },
                  ...(file.quant ? [{ label: 'Quantization', value: file.quant }] : []),
                ]),
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
            label: `Download ${adapter ? 'adapter' : file.quant || 'GGUF'}`,
            disabled: downloadingKey === dlKey,
            busyLabel:
              downloadingKey === dlKey ? downloadProgress || 'Downloading…' : undefined,
            onClick: () => void downloadModel(entry),
          });
        } else {
          const importLabel = adapter ? 'Compose & import' : 'Import to Ollama';
          primaryAction = {
            id: 'import',
            label: importLabel,
            disabled: !ollamaRunning || importingKey === dlKey || adapterComposeBlocked,
            busyLabel: importingKey === dlKey ? (adapter ? 'Composing…' : 'Importing…') : undefined,
            onClick: () => void importToOllama(entry),
          };
          detailActions.push(
            {
              id: 'import',
              label: importLabel,
              disabled: !ollamaRunning || importingKey === dlKey || adapterComposeBlocked,
              busyLabel: importingKey === dlKey ? (adapter ? 'Composing…' : 'Importing…') : undefined,
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
    canComposeLoRA,
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
            placeholder="hf_… or set hub token in Settings"
            className="w-full px-2 py-1.5 text-sm bg-gray-900 border border-gray-600 rounded text-gray-200"
          />
        </div>
      )}

      {tab === 'local' && (
        <p className="text-xs text-amber-500/90">
          Downloads GGUF or LoRA adapter files, then imports or composes in Ollama. LoRA entries need the base
          model (e.g. qwen2.5-coder:14b) pulled first.
          {!ollamaRunning && ' (Ollama is not running.)'}
        </p>
      )}

      {catalogError && <p className="text-sm text-red-400">{catalogError}</p>}
      {searchError && <p className="text-sm text-amber-400">{searchError}</p>}
      {searchLoading && <p className="text-xs text-gray-500">Searching Hugging Face Hub…</p>}
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
      searchPlaceholder="Search Hugging Face models…"
      onViewChange={onViewChange}
      resetDetailSignal={resetDetailSignal}
      banner={banner}
      footer={
        searchHasMore ? (
          <button
            type="button"
            disabled={searchLoading}
            onClick={() => void fetchSearch(debouncedQuery, tab, searchOffset + 24, true)}
            className="w-full px-3 py-2 text-xs bg-gray-800 text-gray-300 rounded border border-gray-700 hover:bg-gray-700 disabled:opacity-40"
          >
            {searchLoading ? 'Loading…' : 'Load more models'}
          </button>
        ) : undefined
      }
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
