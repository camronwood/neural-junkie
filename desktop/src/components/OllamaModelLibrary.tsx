import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { ModelStoreBrowse } from './model-library/ModelStoreBrowse';
import type { StoreModelAction, StoreModelItem } from './model-library/types';

export interface OllamaCatalogEntry {
  name: string;
  title: string;
  description: string;
  tags: string[];
  size_hint?: string;
  icon_key?: string;
  publisher?: string;
}

interface RegistryModel {
  name: string;
  title: string;
  description?: string;
}

interface RegistrySearchResponse {
  query: string;
  page: number;
  models: RegistryModel[];
  has_more: boolean;
}

interface RegistryTagsResponse {
  name: string;
  default_tag?: string;
  tags: { name: string }[];
}

interface HubProvider {
  id: string;
  type: string;
  name: string;
  endpoint?: string;
  api_key?: string;
  model?: string;
  headers?: Record<string, string>;
  work_dir?: string;
}

interface OllamaModelLibraryProps {
  serverAddr: string;
  switchAllAgentProviders: (provider: string, model: string) => Promise<void>;
  onAfterModelChange?: () => void;
  onViewChange?: (view: 'grid' | 'detail') => void;
  resetDetailSignal?: number;
}

function hubFetchError(status: number, fallback: string): string {
  if (status === 429) {
    return 'Hub rate limit exceeded — wait a moment and try again';
  }
  return fallback;
}

function familyName(tag: string): string {
  const i = tag.indexOf(':');
  return i >= 0 ? tag.slice(0, i) : tag;
}

function mergeCatalogRows(curated: OllamaCatalogEntry[], registry: RegistryModel[]): OllamaCatalogEntry[] {
  const out = [...curated];
  const seenFamilies = new Set(curated.map((row) => familyName(row.name)));
  const seenNames = new Set(curated.map((row) => row.name));

  for (const reg of registry) {
    const base = familyName(reg.name);
    if (!base || seenFamilies.has(base) || seenNames.has(base)) continue;
    seenFamilies.add(base);
    seenNames.add(base);
    out.push({
      name: base,
      title: reg.title || base,
      description: reg.description || `Ollama library model (${base})`,
      tags: [],
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
  if (buffer.startsWith('data: ')) {
    const raw = buffer.slice(6).trim();
    if (raw) {
      try {
        onData(JSON.parse(raw) as Record<string, unknown>);
      } catch {
        /* ignore */
      }
    }
  }
}

export function OllamaModelLibrary({
  serverAddr,
  switchAllAgentProviders,
  onAfterModelChange,
  onViewChange,
  resetDetailSignal,
}: OllamaModelLibraryProps) {
  const [curated, setCurated] = useState<OllamaCatalogEntry[]>([]);
  const [registry, setRegistry] = useState<RegistryModel[]>([]);
  const [registryPage, setRegistryPage] = useState(1);
  const [registryHasMore, setRegistryHasMore] = useState(false);
  const [catalogError, setCatalogError] = useState<string | null>(null);
  const [registryError, setRegistryError] = useState<string | null>(null);
  const [registryLoading, setRegistryLoading] = useState(false);
  const [ollamaRunning, setOllamaRunning] = useState(false);
  const [installed, setInstalled] = useState<Set<string>>(() => new Set());
  const [query, setQuery] = useState('');
  const [debouncedQuery, setDebouncedQuery] = useState('');
  const [customTag, setCustomTag] = useState('');
  const [pullingName, setPullingName] = useState<string | null>(null);
  const [pullProgress, setPullProgress] = useState('');
  const [actionMessage, setActionMessage] = useState<{ kind: 'ok' | 'err'; text: string } | null>(null);
  const [deletingName, setDeletingName] = useState<string | null>(null);
  const [useBusyName, setUseBusyName] = useState<string | null>(null);
  const [tagCache, setTagCache] = useState<Record<string, RegistryTagsResponse>>({});
  const tagCacheRef = useRef(tagCache);
  tagCacheRef.current = tagCache;
  const tagsInflightRef = useRef(new Map<string, Promise<RegistryTagsResponse | null>>());

  const loadTagsForFamily = useCallback(
    async (family: string): Promise<RegistryTagsResponse | null> => {
      const fam = familyName(family);
      if (!fam) return null;
      const cached = tagCacheRef.current[fam];
      if (cached) return cached;
      const pending = tagsInflightRef.current.get(fam);
      if (pending) return pending;

      const work = (async () => {
        try {
          const r = await fetch(`${serverAddr}/api/ollama/library/tags?name=${encodeURIComponent(fam)}`);
          if (!r.ok) return null;
          const data = (await r.json()) as RegistryTagsResponse;
          setTagCache((prev) => ({ ...prev, [fam]: data }));
          return data;
        } catch {
          return null;
        } finally {
          tagsInflightRef.current.delete(fam);
        }
      })();
      tagsInflightRef.current.set(fam, work);
      return work;
    },
    [serverAddr]
  );

  useEffect(() => {
    const id = setTimeout(() => setDebouncedQuery(query.trim()), 300);
    return () => clearTimeout(id);
  }, [query]);

  const refreshInstalled = useCallback(async () => {
    try {
      const st = await fetch(`${serverAddr}/api/ollama/install-status`).then((r) => r.json());
      const running = Boolean(st.running);
      setOllamaRunning(running);
      if (!running) {
        setInstalled(new Set());
        return;
      }
      const mr = await fetch(`${serverAddr}/api/ollama/models`);
      if (!mr.ok) {
        setInstalled(new Set());
        return;
      }
      const data = await mr.json();
      const raw = data.models as unknown;
      const names: string[] = Array.isArray(raw)
        ? raw
            .map((m) => (typeof m === 'string' ? m : (m as { name?: string }).name))
            .filter((x): x is string => Boolean(x))
        : [];
      setInstalled(new Set(names));
    } catch {
      setOllamaRunning(false);
      setInstalled(new Set());
    }
  }, [serverAddr]);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const r = await fetch(`${serverAddr}/api/ollama/catalog`);
        if (!r.ok) throw new Error(hubFetchError(r.status, r.statusText));
        const rows = (await r.json()) as OllamaCatalogEntry[];
        if (!cancelled) {
          setCurated(Array.isArray(rows) ? rows : []);
          setCatalogError(null);
        }
      } catch (e) {
        if (!cancelled) {
          setCurated([]);
          setCatalogError(e instanceof Error ? e.message : 'Failed to load catalog');
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [serverAddr]);

  const fetchRegistryPage = useCallback(
    async (searchQuery: string, page: number, append: boolean) => {
      setRegistryLoading(true);
      setRegistryError(null);
      try {
        const params = new URLSearchParams({ page: String(page) });
        if (searchQuery) params.set('q', searchQuery);
        const r = await fetch(`${serverAddr}/api/ollama/library/search?${params}`);
        if (!r.ok) throw new Error(hubFetchError(r.status, await r.text()));
        const data = (await r.json()) as RegistrySearchResponse;
        setRegistry((prev) => (append ? [...prev, ...(data.models ?? [])] : data.models ?? []));
        setRegistryPage(data.page ?? page);
        setRegistryHasMore(Boolean(data.has_more));
      } catch (e) {
        setRegistryError(e instanceof Error ? e.message : 'Failed to search Ollama library');
        if (!append) setRegistry([]);
        setRegistryHasMore(false);
      } finally {
        setRegistryLoading(false);
      }
    },
    [serverAddr]
  );

  useEffect(() => {
    void fetchRegistryPage(debouncedQuery, 1, false);
  }, [debouncedQuery, fetchRegistryPage]);

  useEffect(() => {
    void refreshInstalled();
  }, [refreshInstalled]);

  const catalog = useMemo(() => mergeCatalogRows(curated, registry), [curated, registry]);

  const installedExtras = useMemo(() => {
    const known = new Set(catalog.map((row) => row.name));
    const knownFamilies = new Set(catalog.map((row) => familyName(row.name)));
    const extras: OllamaCatalogEntry[] = [];
    for (const name of installed) {
      if (known.has(name)) continue;
      const fam = familyName(name);
      if (knownFamilies.has(fam)) continue;
      extras.push({
        name,
        title: name,
        description: 'Installed on this machine',
        tags: ['installed'],
      });
    }
    return extras.sort((a, b) => a.name.localeCompare(b.name));
  }, [catalog, installed]);

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    const rows = [...installedExtras, ...catalog];
    if (!q) return rows;
    return rows.filter((row) => {
      const hay = [row.name, row.title, row.description, row.publisher, ...(row.tags || [])]
        .filter(Boolean)
        .join(' ')
        .toLowerCase();
      return hay.includes(q);
    });
  }, [catalog, installedExtras, query]);

  async function resolvePullTag(model: string): Promise<string> {
    if (model.includes(':')) return model;
    const fam = familyName(model);
    const data = (await loadTagsForFamily(fam)) ?? tagCacheRef.current[fam];
    if (data?.default_tag) return data.default_tag;
    return data?.tags?.[0]?.name || `${fam}:latest`;
  }

  async function pullModel(model: string) {
    const tag = (await resolvePullTag(model)).trim();
    if (!tag || !ollamaRunning) return;
    setPullingName(tag);
    setPullProgress('Starting…');
    setActionMessage(null);
    let streamError: string | null = null;
    try {
      const resp = await fetch(`${serverAddr}/api/ollama/pull`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ model: tag }),
      });
      if (!resp.ok) {
        const t = await resp.text();
        throw new Error(t || resp.statusText);
      }
      const reader = resp.body?.getReader();
      if (!reader) {
        throw new Error('No response body');
      }
      await parseSSEChunks(reader, (data) => {
        if (data.status === 'error') {
          streamError = typeof data.error === 'string' ? data.error : 'Pull failed';
          setPullProgress(streamError);
          return;
        }
        if (typeof data.error === 'string' && data.error) {
          streamError = data.error;
          setPullProgress(streamError);
          return;
        }
        const pct = data.percent;
        if (typeof pct === 'number' && pct > 0) {
          setPullProgress(`${pct.toFixed(1)}%`);
        } else if (typeof data.status === 'string') {
          setPullProgress(String(data.status));
        }
      });
      if (streamError) {
        setActionMessage({ kind: 'err', text: streamError });
      }
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e);
      setPullProgress(msg);
      setActionMessage({ kind: 'err', text: msg });
    } finally {
      setPullingName(null);
      setPullProgress('');
      await refreshInstalled();
      onAfterModelChange?.();
    }
  }

  async function deleteModel(model: string) {
    if (!ollamaRunning) return;
    if (!confirm(`Remove Ollama model "${model}" from this machine?`)) return;
    setDeletingName(model);
    setActionMessage(null);
    try {
      const resp = await fetch(`${serverAddr}/api/ollama/delete`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ model }),
      });
      if (!resp.ok) {
        const t = await resp.text();
        throw new Error(t || resp.statusText);
      }
      setActionMessage({ kind: 'ok', text: `Removed ${model}` });
    } catch (e) {
      setActionMessage({
        kind: 'err',
        text: e instanceof Error ? e.message : String(e),
      });
    } finally {
      setDeletingName(null);
      await refreshInstalled();
      onAfterModelChange?.();
    }
  }

  async function useForAgents(model: string) {
    const tag = model.includes(':') ? model : await resolvePullTag(model);
    setUseBusyName(model);
    setActionMessage(null);
    try {
      const pr = await fetch(`${serverAddr}/api/providers`);
      if (!pr.ok) throw new Error(pr.statusText);
      const providers = (await pr.json()) as HubProvider[];
      const ollama = providers.find((p) => p.type === 'ollama');
      if (!ollama) {
        throw new Error('No Ollama provider in hub config. Add one under AI Providers.');
      }
      const updated: HubProvider = { ...ollama, model: tag };
      const put = await fetch(`${serverAddr}/api/providers/${encodeURIComponent(ollama.id)}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(updated),
      });
      if (!put.ok) {
        const t = await put.text();
        throw new Error(t || put.statusText);
      }
      await switchAllAgentProviders('ollama', tag);
      setActionMessage({
        kind: 'ok',
        text: `Hub Ollama provider set to ${tag} and all agents switched.`,
      });
    } catch (e) {
      setActionMessage({
        kind: 'err',
        text: e instanceof Error ? e.message : String(e),
      });
    } finally {
      setUseBusyName(null);
      onAfterModelChange?.();
    }
  }

  const storeItems = useMemo((): StoreModelItem[] => {
    const globalPullBusy = !!pullingName;

    return filtered.map((row) => {
      const installedMatch =
        installed.has(row.name) ||
        [...installed].some((name) => name === row.name || name.startsWith(`${row.name}:`));
      const installedTag =
        [...installed].find((name) => name === row.name || name.startsWith(`${row.name}:`)) ?? row.name;
      const isIn = installedMatch;
      const rowBusy =
        pullingName === row.name ||
        pullingName === installedTag ||
        deletingName === row.name ||
        deletingName === installedTag ||
        useBusyName === row.name;
      const pullLabel =
        pullingName === row.name || pullingName === installedTag
          ? pullProgress || 'Pulling…'
          : 'Install';

      const primaryAction: StoreModelAction | undefined = isIn
        ? {
            id: 'use',
            label: 'Use for agents',
            variant: 'primary',
            disabled: rowBusy || globalPullBusy,
            busyLabel: useBusyName === row.name ? 'Applying…' : undefined,
            onClick: () => void useForAgents(installedTag),
          }
        : {
            id: 'install',
            label: 'Install',
            disabled: !ollamaRunning || rowBusy || globalPullBusy,
            busyLabel: pullingName === row.name ? pullLabel : undefined,
            onClick: () => void pullModel(row.name),
          };

      const detailActions: StoreModelAction[] = [];
      const tagInfo = tagCache[familyName(row.name)];
      if (tagInfo?.tags?.length) {
        for (const tag of tagInfo.tags) {
          detailActions.push({
            id: `install-${tag.name}`,
            label: `Install ${tag.name}`,
            disabled: !ollamaRunning || rowBusy || globalPullBusy,
            onClick: () => void pullModel(tag.name),
          });
        }
      }

      if (!isIn) {
        if (detailActions.length === 0) {
          detailActions.push({
            id: 'install',
            label: 'Install',
            disabled: !ollamaRunning || rowBusy || globalPullBusy,
            busyLabel: pullingName === row.name ? pullLabel : undefined,
            onClick: () => void pullModel(row.name),
          });
        }
      } else {
        detailActions.unshift({
          id: 'use',
          label: 'Use for agents',
          variant: 'primary',
          disabled: rowBusy || globalPullBusy,
          busyLabel: useBusyName === row.name ? 'Applying…' : undefined,
          onClick: () => void useForAgents(installedTag),
        });
        detailActions.push({
          id: 'remove',
          label: 'Remove',
          variant: 'danger',
          disabled: rowBusy || globalPullBusy,
          busyLabel: deletingName === installedTag ? 'Removing…' : undefined,
          onClick: () => void deleteModel(installedTag),
        });
      }

      return {
        id: row.name,
        title: row.title,
        subtitle: row.name,
        description: row.description,
        tags: row.tags ?? [],
        sizeHint: row.size_hint,
        publisher: row.publisher,
        iconKey: row.icon_key,
        status: isIn ? 'installed' : 'available',
        detailRows: [{ label: 'Ollama tag', value: row.name }],
        primaryAction,
        detailActions,
      };
    });
  }, [
    filtered,
    installed,
    ollamaRunning,
    pullingName,
    pullProgress,
    deletingName,
    useBusyName,
    tagCache,
  ]);

  const handleDetailOpen = useCallback(
    (item: StoreModelItem) => {
      void loadTagsForFamily(item.id);
    },
    [loadTagsForFamily]
  );

  const banner = (
    <>
      {catalogError && (
        <div className="text-sm text-red-400 border border-red-900/50 rounded p-2">{catalogError}</div>
      )}
      {registryError && (
        <div className="text-sm text-amber-400 border border-amber-900/50 rounded p-2">{registryError}</div>
      )}
      {actionMessage && (
        <div
          className={`text-sm rounded p-2 border ${
            actionMessage.kind === 'ok'
              ? 'bg-green-900/20 text-green-300 border-green-800/50'
              : 'bg-red-900/20 text-red-300 border-red-800/50'
          }`}
        >
          {actionMessage.text}
        </div>
      )}
      {!ollamaRunning && (
        <p className="text-xs text-gray-500">
          Start Ollama above to install models. Browse the full Ollama library and pull when the server is running.
        </p>
      )}
      {registryLoading && (
        <p className="text-xs text-gray-500">Loading models from ollama.com…</p>
      )}
    </>
  );

  const footer = (
    <div className="space-y-3">
      {registryHasMore && (
        <button
          type="button"
          disabled={registryLoading}
          onClick={() => void fetchRegistryPage(debouncedQuery, registryPage + 1, true)}
          className="w-full px-3 py-2 text-xs bg-gray-800 text-gray-300 rounded border border-gray-700 hover:bg-gray-700 disabled:opacity-40"
        >
          {registryLoading ? 'Loading…' : 'Load more models'}
        </button>
      )}
      <div className="border border-gray-700 rounded-lg p-3 space-y-2">
        <div className="text-xs font-medium text-gray-400">Custom model tag</div>
        <p className="text-xs text-gray-500">
          Pull any Ollama library name (e.g. <span className="font-mono text-gray-400">mistral-nemo:12b</span>).
        </p>
        <div className="flex gap-2 items-center">
          <input
            value={customTag}
            onChange={(e) => setCustomTag(e.target.value)}
            placeholder="model:tag"
            disabled={!ollamaRunning || !!pullingName}
            className="flex-1 px-3 py-2 bg-gray-900 border border-gray-700 rounded text-sm text-white disabled:opacity-50"
          />
          <button
            type="button"
            disabled={!ollamaRunning || !customTag.trim() || !!pullingName}
            onClick={() => {
              void (async () => {
                const tag = customTag.trim();
                if (!tag) return;
                await pullModel(tag);
                setCustomTag('');
              })();
            }}
            className="px-3 py-2 text-xs bg-blue-600 text-white rounded hover:bg-blue-500 disabled:opacity-40"
          >
            Pull
          </button>
        </div>
        {pullingName && (
          <div className="text-xs text-blue-300 font-mono">
            {pullingName}
            {pullProgress ? ` — ${pullProgress}` : ''}
          </div>
        )}
      </div>
    </div>
  );

  return (
    <ModelStoreBrowse
      items={storeItems}
      query={query}
      onQueryChange={setQuery}
      searchPlaceholder="Search all Ollama models…"
      onViewChange={onViewChange}
      onDetailOpen={handleDetailOpen}
      resetDetailSignal={resetDetailSignal}
      banner={banner}
      footer={footer}
      headerRight={
        <button
          type="button"
          onClick={() => void refreshInstalled()}
          className="px-3 py-1.5 text-xs bg-gray-700 text-gray-300 rounded-lg hover:bg-gray-600 shrink-0"
        >
          Refresh installed
        </button>
      }
    />
  );
}
