import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { ModelStoreBrowse } from './model-library/ModelStoreBrowse';
import type { StoreModelAction, StoreModelItem } from './model-library/types';
import {
  actionLabelWithSize,
  estimateSizeHintFromName,
  formatSizeBytes,
} from './model-library/sizeHint';
import { useModelTransferStore } from '../stores/modelTransferStore';

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
  deprecated?: boolean;
  ollama_compose_supported?: boolean;
  files?: { filename: string; quant?: string; size_hint?: string; size_bytes?: number; role?: string }[];
  min_ollama_version?: string;
}

const DEFAULT_LORA_BASE = 'llama3.1:8b';

function adapterComposeSupported(entry: HfCatalogEntry): boolean {
  if (!isAdapterEntry(entry)) return false;
  if (entry.deprecated) return false;
  if (entry.ollama_compose_supported === false) return false;
  const base = (entry.base_ollama_tag ?? DEFAULT_LORA_BASE).toLowerCase();
  return (
    base.includes('llama') ||
    base.includes('mistral') ||
    base.includes('mixtral') ||
    base.includes('gemma') ||
    base.includes('codestral') ||
    base.includes('devstral')
  );
}

function isAdapterEntry(entry: HfCatalogEntry): boolean {
  const kind = (entry.kind ?? '').toLowerCase();
  return kind === 'adapter' || kind === 'lora';
}

function fileRole(f: { role?: string; filename?: string }): string {
  const role = (f.role ?? '').trim().toLowerCase();
  if (role === 'mmproj' || role === 'projector') return 'mmproj';
  if ((f.filename ?? '').toLowerCase().includes('mmproj')) return 'mmproj';
  return 'main';
}

function primaryFilename(entry: HfCatalogEntry): string | undefined {
  const files = entry.files ?? [];
  const explicit = files.find((f) => (f.role ?? '').trim().toLowerCase() === 'main');
  if (explicit?.filename) return explicit.filename;
  return files.find((f) => fileRole(f) === 'main')?.filename;
}

function mmprojFilename(entry: HfCatalogEntry): string | undefined {
  return (entry.files ?? []).find((f) => fileRole(f) === 'mmproj')?.filename;
}

function catalogFilenames(entry: HfCatalogEntry): string[] {
  const main = primaryFilename(entry);
  const mmproj = mmprojFilename(entry);
  return [main, mmproj].filter((x): x is string => Boolean(x));
}

function ollamaVersionAtLeast(installed: string | undefined, required: string | undefined): boolean {
  if (!required?.trim()) return true;
  if (!installed?.trim()) return false;
  const parts = (v: string) =>
    v
      .trim()
      .replace(/^v/i, '')
      .split(/[^\d]+/)
      .filter(Boolean)
      .map((n) => parseInt(n, 10) || 0);
  const a = parts(installed);
  const b = parts(required);
  for (let i = 0; i < Math.max(a.length, b.length, 3); i++) {
    const av = a[i] ?? 0;
    const bv = b[i] ?? 0;
    if (av !== bv) return av > bv;
  }
  return true;
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
  /** Called when a download starts so the parent can switch to Installed. */
  onDownloadStarted?: () => void;
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
  size_hint?: string;
  files?: { filename: string; quant?: string; size_hint?: string; size_bytes?: number; role?: string }[];
}

interface HfSearchResponse {
  query: string;
  mode: string;
  models: HfSearchHit[];
  has_more: boolean;
  offset: number;
}

function primarySizeHint(entry: {
  size_hint?: string;
  title?: string;
  repo_id?: string;
  files?: { filename?: string; quant?: string; size_hint?: string; size_bytes?: number; role?: string }[];
}): string | undefined {
  if (entry.size_hint?.trim()) return entry.size_hint.trim();
  const files = entry.files ?? [];
  const main =
    files.find((f) => (f.role ?? '').toLowerCase() === 'main') ??
    files.find((f) => !(f.filename ?? '').toLowerCase().includes('mmproj') && (f.role ?? '').toLowerCase() !== 'mmproj') ??
    files[0];
  if (main) {
    const fromFile = main.size_hint?.trim() || formatSizeBytes(main.size_bytes);
    if (fromFile) return fromFile;
  }
  return (
    estimateSizeHintFromName(entry.title || '') ||
    estimateSizeHintFromName(entry.repo_id || '')
  );
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
      size_hint: hit.size_hint || primarySizeHint(hit),
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
  onDownloadStarted,
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
  const [ollamaEffectiveVersion, setOllamaEffectiveVersion] = useState<string | undefined>();
  const [ollamaUpdateSupported, setOllamaUpdateSupported] = useState(false);
  const [query, setQuery] = useState('');
  const [debouncedQuery, setDebouncedQuery] = useState('');
  const [hfToken, setHfToken] = useState('');
  const [downloadingKey, setDownloadingKey] = useState<string | null>(null);
  const [downloadProgress, setDownloadProgress] = useState('');
  const [importingKey, setImportingKey] = useState<string | null>(null);
  const [actionMessage, setActionMessage] = useState<{ kind: 'ok' | 'err'; text: string } | null>(null);
  const [configuredAgents, setConfiguredAgents] = useState<ConfiguredAgent[]>([]);
  const [ollamaModels, setOllamaModels] = useState<string[]>([]);
  /** Hub-resolved GGUF/adapter files keyed by repo_id (search hits start without files). */
  const [resolvedFiles, setResolvedFiles] = useState<
    Record<string, NonNullable<HfCatalogEntry['files']>>
  >({});
  const [resolvingSizeIds, setResolvingSizeIds] = useState<Set<string>>(() => new Set());
  const resolveInflightRef = useRef(new Set<string>());
  const resolveFailedRef = useRef(new Set<string>());
  const resolveGenRef = useRef(0);

  useEffect(() => {
    const id = setTimeout(() => setDebouncedQuery(query.trim()), 300);
    return () => clearTimeout(id);
  }, [query]);

  useEffect(() => {
    resolveGenRef.current += 1;
    setResolvedFiles({});
    setResolvingSizeIds(new Set());
    resolveInflightRef.current.clear();
    resolveFailedRef.current.clear();
  }, [tab]);

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
      setOllamaEffectiveVersion(
        typeof st.effective_version === 'string' && st.effective_version
          ? st.effective_version
          : typeof st.version === 'string'
            ? st.version
            : undefined,
      );
      setOllamaUpdateSupported(Boolean(st.update_supported));
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
      setOllamaEffectiveVersion(undefined);
      setOllamaUpdateSupported(false);
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

  const catalogWithFiles = useMemo(() => {
    return catalog.map((entry) => {
      if (entry.files?.length) return entry;
      const files = resolvedFiles[entry.repo_id];
      return files?.length ? { ...entry, files } : entry;
    });
  }, [catalog, resolvedFiles]);

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return catalogWithFiles;
    return catalogWithFiles.filter((e) => {
      const hay = `${e.repo_id} ${e.title} ${e.description} ${e.publisher ?? ''} ${(e.tags || []).join(' ')}`.toLowerCase();
      return hay.includes(q);
    });
  }, [catalogWithFiles, query]);

  // Prefetch preferred GGUF sizes for Download-tab cards (search hits have no files until resolved).
  useEffect(() => {
    if (tab !== 'local') return;

    const pending = catalog.filter((entry) => {
      if (entry.files?.length || entry.size_hint) return false;
      if (resolvedFiles[entry.repo_id]?.length) return false;
      if (resolveFailedRef.current.has(entry.repo_id)) return false;
      if (resolveInflightRef.current.has(entry.repo_id)) return false;
      return true;
    });
    if (pending.length === 0) return;

    const gen = resolveGenRef.current;
    let stopQueue = false;
    const queue = [...pending];
    const concurrency = Math.min(4, queue.length);

    const markResolving = (repoId: string, on: boolean) => {
      if (gen !== resolveGenRef.current) return;
      setResolvingSizeIds((prev) => {
        const next = new Set(prev);
        if (on) next.add(repoId);
        else next.delete(repoId);
        return next;
      });
    };

    async function resolveOne(entry: HfCatalogEntry) {
      const repoId = entry.repo_id;
      resolveInflightRef.current.add(repoId);
      markResolving(repoId, true);
      try {
        const kind = isAdapterEntry(entry) ? 'adapter' : '';
        const qs = kind ? `&kind=${encodeURIComponent(kind)}` : '';
        const r = await fetch(
          `${serverAddr}/api/hf/files?repo_id=${encodeURIComponent(repoId)}${qs}`
        );
        if (!r.ok) throw new Error(await r.text());
        const data = (await r.json()) as { files?: HfCatalogEntry['files'] };
        const files = data.files ?? [];
        if (gen !== resolveGenRef.current) return;
        if (files.length === 0) {
          resolveFailedRef.current.add(repoId);
          return;
        }
        setResolvedFiles((prev) => (prev[repoId]?.length ? prev : { ...prev, [repoId]: files }));
      } catch {
        if (gen === resolveGenRef.current) resolveFailedRef.current.add(repoId);
      } finally {
        resolveInflightRef.current.delete(repoId);
        markResolving(repoId, false);
      }
    }

    async function worker() {
      while (!stopQueue) {
        const entry = queue.shift();
        if (!entry) return;
        await resolveOne(entry);
      }
    }

    void Promise.all(Array.from({ length: concurrency }, () => worker()));
    return () => {
      // Stop dequeuing more work; in-flight fetches still complete and write if gen matches.
      stopQueue = true;
    };
  }, [tab, catalog, resolvedFiles, serverAddr]);

  function rememberResolvedFiles(repoId: string, files: NonNullable<HfCatalogEntry['files']>) {
    setResolvedFiles((prev) => (prev[repoId]?.length ? prev : { ...prev, [repoId]: files }));
  }

  async function ensureEntryFiles(entry: HfCatalogEntry): Promise<HfCatalogEntry> {
    if (entry.files?.length) return entry;
    const cached = resolvedFiles[entry.repo_id];
    if (cached?.length) return { ...entry, files: cached };
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
    rememberResolvedFiles(entry.repo_id, files);
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
    const names = catalogFilenames(entry);
    if (names.length > 0) {
      return names.every((fn) => localFiles.some((f) => f.repo_id === entry.repo_id && f.filename === fn));
    }
    return localFiles.some((f) => f.repo_id === entry.repo_id);
  }

  async function downloadOneFile(
    repoId: string,
    filename: string,
    onProgress?: (percent?: number, status?: string) => void
  ): Promise<void> {
    const st = await pollDownloadStatus(repoId, filename);
    if (st?.status === 'success') {
      return;
    }

    let streamError: string | null = null;
    const resp = await fetch(`${serverAddr}/api/hf/download`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ repo_id: repoId, filename }),
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
        setDownloadProgress(`${filename}: ${pct.toFixed(1)}%`);
        onProgress?.(pct, undefined);
      } else if (typeof data.status === 'string') {
        setDownloadProgress(`${filename}: ${String(data.status)}`);
        onProgress?.(undefined, String(data.status));
      }
    });
    if (streamError) {
      throw new Error(streamError);
    }
    const done = await pollDownloadStatus(repoId, filename);
    if (done?.status !== 'success') {
      throw new Error(`Download of ${filename} did not complete`);
    }
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
    const transferId = `hf:${entry.repo_id}`;
    const transfers = useModelTransferStore.getState();
    transfers.start({
      id: transferId,
      source: 'huggingface',
      title: entry.title,
      subtitle: entry.repo_id,
      progressLabel: 'Resolving files…',
    });
    onDownloadStarted?.();
    setDownloadingKey(entry.repo_id);
    setDownloadProgress('Resolving files…');
    setActionMessage(null);
    let resolved = entry;
    try {
      resolved = await ensureEntryFiles(entry);
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e);
      setDownloadingKey(null);
      setDownloadProgress('');
      setActionMessage({ kind: 'err', text: msg });
      transfers.fail(transferId, msg);
      return;
    }
    const filenames = catalogFilenames(resolved);
    if (filenames.length === 0) {
      setDownloadingKey(null);
      setDownloadProgress('');
      const msg = 'No GGUF file available for this model.';
      setActionMessage({ kind: 'err', text: msg });
      transfers.fail(transferId, msg);
      return;
    }
    const key = `${resolved.repo_id}:${filenames[0]}`;
    setDownloadingKey(key);
    setDownloadProgress('Starting…');
    transfers.update(transferId, { progressLabel: 'Starting…' });

    try {
      for (const filename of filenames) {
        const label = `Downloading ${filename}…`;
        setDownloadProgress(label);
        transfers.update(transferId, { progressLabel: label });
        await downloadOneFile(resolved.repo_id, filename, (pct, status) => {
          if (typeof pct === 'number' && pct > 0) {
            const progressLabel = `${filename}: ${pct.toFixed(1)}%`;
            setDownloadProgress(progressLabel);
            transfers.update(transferId, { progressLabel, percent: pct });
          } else if (status) {
            const progressLabel = `${filename}: ${status}`;
            setDownloadProgress(progressLabel);
            transfers.update(transferId, { progressLabel });
          }
        });
      }
      setActionMessage({
        kind: 'ok',
        text:
          filenames.length > 1
            ? `Downloaded ${filenames.join(' + ')}`
            : `Downloaded ${filenames[0]}`,
      });
      transfers.complete(transferId);
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e);
      const filename = filenames[0];
      const still = await pollDownloadStatus(resolved.repo_id, filename);
      if (still?.status === 'downloading' || still?.status === 'starting' || still?.status === 'queued') {
        setActionMessage({
          kind: 'ok',
          text: 'Download continues on the hub in the background. Check the Installed tab for progress.',
        });
        transfers.update(transferId, {
          progressLabel: still.percent ? `${still.percent.toFixed(1)}%` : String(still.status),
          percent: typeof still.percent === 'number' ? still.percent : undefined,
        });
      } else if (still?.status === 'success' && filenames.length === 1) {
        setActionMessage({ kind: 'ok', text: `Downloaded ${filename}` });
        transfers.complete(transferId);
      } else {
        setActionMessage({ kind: 'err', text: msg });
        transfers.fail(transferId, msg);
      }
    } finally {
      setDownloadingKey(null);
      setDownloadProgress('');
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
    const filename = primaryFilename(resolved);
    if (!filename) return;
    const adapter = isAdapterEntry(resolved);
    if (adapter && !canComposeLoRA) {
      setActionMessage({
        kind: 'err',
        text: 'Enable Specialist tuning pack to compose LoRA adapters.',
      });
      return;
    }
    const baseTag = resolved.base_ollama_tag || DEFAULT_LORA_BASE;
    if (adapter && !adapterComposeSupported(resolved)) {
      setActionMessage({
        kind: 'err',
        text: 'This adapter cannot be composed in Ollama (Qwen safetensors LoRA is unsupported). Use a Llama or Mistral base — see docs/LORA_ADAPTERS.md.',
      });
      return;
    }
    if (adapter && !ollamaHasBase(baseTag)) {
      setActionMessage({
        kind: 'err',
        text: `Base model ${baseTag} is not in Ollama. Pull it from the Ollama tab first.`,
      });
      return;
    }
    const minVer = resolved.min_ollama_version?.trim();
    if (minVer && !ollamaVersionAtLeast(ollamaEffectiveVersion, minVer)) {
      setActionMessage({
        kind: 'err',
        text: `This model needs Ollama ${minVer}+ (current: ${ollamaEffectiveVersion || 'unknown'}). Use Update Ollama in the toolbar chip${
          ollamaUpdateSupported ? '' : ' or upgrade from https://ollama.com/download'
        }.`,
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
          ollama_tag: resolved.default_ollama_tag || undefined,
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
    let filenames = catalogFilenames(entry);
    if (filenames.length === 0) {
      filenames = localFiles.filter((f) => f.repo_id === entry.repo_id).map((f) => f.filename);
    }
    if (filenames.length === 0) return;
    try {
      for (const filename of filenames) {
        const resp = await fetch(`${serverAddr}/api/hf/delete`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ repo_id: entry.repo_id, filename }),
        });
        if (!resp.ok) throw new Error(await resp.text());
      }
      setActionMessage({
        kind: 'ok',
        text: filenames.length > 1 ? `Removed cached ${filenames.join(' + ')}` : `Removed cached ${filenames[0]}`,
      });
      await refreshLocal();
    } catch (e) {
      setActionMessage({ kind: 'err', text: e instanceof Error ? e.message : String(e) });
    }
  }

  function isBusyForEntry(entry: HfCatalogEntry, key: string | null, mainName?: string): boolean {
    if (!key) return false;
    if (mainName && key === `${entry.repo_id}:${mainName}`) return true;
    if (key === entry.repo_id) return true;
    return key.startsWith(`${entry.repo_id}:`);
  }

  async function resolveFilesForDetail(item: StoreModelItem) {
    if (tab !== 'local') return;
    const entry = filtered.find((e) => e.repo_id === item.id);
    if (!entry || entry.files?.length) return;
    try {
      await ensureEntryFiles(entry);
    } catch {
      /* detail still useful without files; download will surface the error */
    }
  }

  const storeItems = useMemo((): StoreModelItem[] => {
    return filtered.map((entry) => {
      const mainName = primaryFilename(entry);
      const file = (entry.files ?? []).find((f) => f.filename === mainName) ?? entry.files?.[0];
      const downloaded = tab === 'local' && isDownloaded(entry);
      const isHosted = tab === 'hosted';
      const downloading = isBusyForEntry(entry, downloadingKey, mainName);
      const importing = isBusyForEntry(entry, importingKey, mainName);

      const adapter = isAdapterEntry(entry);
      const adapterComposeBlocked = adapter && (!canComposeLoRA || !adapterComposeSupported(entry));
      const detailRows = file
        ? [
            { label: 'Repository', value: entry.repo_id },
            ...(adapter
              ? [
                  { label: 'Kind', value: 'LoRA adapter' },
                  { label: 'Base model', value: entry.base_ollama_tag ?? DEFAULT_LORA_BASE },
                  ...(entry.deprecated || entry.ollama_compose_supported === false
                    ? [{ label: 'Compose', value: 'Not supported in Ollama (use Llama/Mistral adapter)' }]
                    : []),
                  { label: 'Adapter file', value: file.filename },
                ]
              : [
                  { label: 'GGUF file', value: file.filename },
                  ...(file.quant ? [{ label: 'Quantization', value: file.quant }] : []),
                ]),
            ...(file.size_hint ? [{ label: 'File size', value: file.size_hint }] : []),
          ]
        : [
            { label: 'Repository', value: entry.repo_id },
            {
              label: 'Files',
              value: resolvingSizeIds.has(entry.repo_id)
                ? 'Looking up preferred GGUF size…'
                : 'Resolves preferred GGUF on download (Q4_K_M when available)',
            },
          ];

      if (entry.files && entry.files.length > 1) {
        for (const f of entry.files) {
          if (f.filename === file?.filename) continue;
          const label =
            fileRole(f) === 'mmproj' ? 'Vision projector' : f.quant || 'Variant';
          detailRows.push({
            label,
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
      } else if (!downloaded) {
        const sizeHint = primarySizeHint(entry);
        const downloadLabel = actionLabelWithSize(
          file ? `Download ${adapter ? 'adapter' : file.quant || 'GGUF'}` : 'Download',
          sizeHint
        );
        primaryAction = {
          id: 'download',
          label: actionLabelWithSize('Download', sizeHint),
          disabled: downloading,
          busyLabel: downloading ? downloadProgress || 'Resolving…' : undefined,
          onClick: () => void downloadModel(entry),
        };
        detailActions.push({
          id: 'download',
          label: downloadLabel,
          disabled: downloading,
          busyLabel: downloading ? downloadProgress || 'Resolving…' : undefined,
          onClick: () => void downloadModel(entry),
        });
      } else {
        const sizeHint = primarySizeHint(entry);
        const importLabel = actionLabelWithSize(
          adapter ? 'Compose & import' : 'Import to Ollama',
          sizeHint
        );
        primaryAction = {
          id: 'import',
          label: importLabel,
          disabled: !ollamaRunning || importing || adapterComposeBlocked,
          busyLabel: importing ? (adapter ? 'Composing…' : 'Importing…') : undefined,
          onClick: () => void importToOllama(entry),
        };
        detailActions.push(
          {
            id: 'import',
            label: importLabel,
            disabled: !ollamaRunning || importing || adapterComposeBlocked,
            busyLabel: importing ? (adapter ? 'Composing…' : 'Importing…') : undefined,
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

      const sizeHint =
        primarySizeHint(entry) ??
        (tab === 'local' && !downloaded && resolvingSizeIds.has(entry.repo_id)
          ? 'Looking up size…'
          : undefined);

      return {
        id: entry.repo_id,
        title: entry.title,
        subtitle: entry.repo_id,
        description: entry.description,
        tags: entry.tags ?? [],
        sizeHint,
        publisher: entry.publisher,
        iconKey: entry.icon_key,
        status: isHosted ? 'cloud' : downloaded ? 'on_disk' : 'available',
        statusLabel: isHosted ? 'Cloud' : downloaded ? 'On disk' : undefined,
        externalUrl: `https://huggingface.co/${entry.repo_id}`,
        detailRows: [
          ...detailRows,
          ...(sizeHint && sizeHint !== 'Looking up size…'
            ? [{ label: 'Download size', value: sizeHint }]
            : []),
        ],
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
    resolvingSizeIds,
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
          Search any GGUF on the Hub — Download resolves the preferred quant (Q4_K_M when available), then
          import or compose in Ollama. LoRA adapters compose on Llama/Mistral bases (default llama3.1:8b).
          {!ollamaRunning && ' (Ollama is not running.)'}
          {ollamaEffectiveVersion && ` Running Ollama ${ollamaEffectiveVersion}.`}
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
      onDetailOpen={(item) => void resolveFilesForDetail(item)}
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
