import { useCallback, useEffect, useMemo, useState } from 'react';

export interface OllamaCatalogEntry {
  name: string;
  title: string;
  description: string;
  tags: string[];
  size_hint?: string;
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
}: OllamaModelLibraryProps) {
  const [catalog, setCatalog] = useState<OllamaCatalogEntry[]>([]);
  const [catalogError, setCatalogError] = useState<string | null>(null);
  const [ollamaRunning, setOllamaRunning] = useState(false);
  const [installed, setInstalled] = useState<Set<string>>(() => new Set());
  const [query, setQuery] = useState('');
  const [customTag, setCustomTag] = useState('');
  const [pullingName, setPullingName] = useState<string | null>(null);
  const [pullProgress, setPullProgress] = useState('');
  const [actionMessage, setActionMessage] = useState<{ kind: 'ok' | 'err'; text: string } | null>(null);
  const [deletingName, setDeletingName] = useState<string | null>(null);
  const [useBusyName, setUseBusyName] = useState<string | null>(null);

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
        if (!r.ok) throw new Error(r.statusText);
        const rows = (await r.json()) as OllamaCatalogEntry[];
        if (!cancelled) {
          setCatalog(Array.isArray(rows) ? rows : []);
          setCatalogError(null);
        }
      } catch (e) {
        if (!cancelled) {
          setCatalog([]);
          setCatalogError(e instanceof Error ? e.message : 'Failed to load catalog');
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [serverAddr]);

  useEffect(() => {
    void refreshInstalled();
  }, [refreshInstalled]);

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return catalog;
    return catalog.filter((row) => {
      const hay = [row.name, row.title, row.description, ...(row.tags || [])]
        .join(' ')
        .toLowerCase();
      return hay.includes(q);
    });
  }, [catalog, query]);

  async function pullModel(model: string) {
    const tag = model.trim();
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
      const updated: HubProvider = { ...ollama, model };
      const put = await fetch(`${serverAddr}/api/providers/${encodeURIComponent(ollama.id)}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(updated),
      });
      if (!put.ok) {
        const t = await put.text();
        throw new Error(t || put.statusText);
      }
      await switchAllAgentProviders('ollama', model);
      setActionMessage({
        kind: 'ok',
        text: `Hub Ollama provider set to ${model} and all agents switched.`,
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

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between gap-2 flex-wrap">
        <h3 className="text-sm font-semibold text-gray-300">Model library</h3>
        <button
          type="button"
          onClick={() => void refreshInstalled()}
          className="px-3 py-1 text-xs bg-gray-700 text-gray-300 rounded hover:bg-gray-600"
        >
          Refresh installed
        </button>
      </div>

      {catalogError && (
        <div className="text-sm text-red-400 border border-red-900/50 rounded p-2">{catalogError}</div>
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
          Start Ollama above to install models. Browse the catalog and pull when the server is running.
        </p>
      )}

      <div>
        <label className="block text-xs text-gray-500 mb-1">Search catalog</label>
        <input
          type="search"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Name, tag, or description…"
          className="w-full px-3 py-2 bg-gray-900 border border-gray-700 rounded text-sm text-white"
        />
      </div>

      <div className="max-h-[min(420px,50vh)] overflow-y-auto border border-gray-700 rounded-lg divide-y divide-gray-800">
        {filtered.length === 0 ? (
          <div className="p-4 text-sm text-gray-500">No catalog entries match your search.</div>
        ) : (
          filtered.map((row) => {
            const isIn = installed.has(row.name);
            const busy = pullingName === row.name || deletingName === row.name || useBusyName === row.name;
            return (
              <div key={row.name} className="p-3 flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2 flex-wrap">
                    <span className="text-sm font-medium text-white">{row.title}</span>
                    {isIn && (
                      <span className="text-[10px] uppercase tracking-wide px-1.5 py-0.5 rounded bg-green-900/40 text-green-300">
                        Installed
                      </span>
                    )}
                    {row.size_hint && (
                      <span className="text-xs text-gray-500">{row.size_hint}</span>
                    )}
                  </div>
                  <div className="text-xs text-gray-500 font-mono mt-0.5">{row.name}</div>
                  <p className="text-xs text-gray-400 mt-1">{row.description}</p>
                  {row.tags?.length ? (
                    <div className="flex flex-wrap gap-1 mt-1">
                      {row.tags.map((t) => (
                        <span key={t} className="text-[10px] px-1.5 py-0.5 rounded bg-gray-800 text-gray-400">
                          {t}
                        </span>
                      ))}
                    </div>
                  ) : null}
                </div>
                <div className="flex flex-wrap gap-2 shrink-0">
                  {!isIn && (
                    <button
                      type="button"
                      disabled={!ollamaRunning || busy || !!pullingName}
                      onClick={() => void pullModel(row.name)}
                      className="px-2 py-1 text-xs bg-blue-600 text-white rounded hover:bg-blue-500 disabled:opacity-40"
                    >
                      {pullingName === row.name ? pullProgress || 'Pulling…' : 'Install'}
                    </button>
                  )}
                  {isIn && (
                    <>
                      <button
                        type="button"
                        disabled={busy || !!pullingName}
                        onClick={() => void useForAgents(row.name)}
                        className="px-2 py-1 text-xs bg-emerald-700/80 text-white rounded hover:bg-emerald-600 disabled:opacity-40"
                      >
                        {useBusyName === row.name ? 'Applying…' : 'Use for agents'}
                      </button>
                      <button
                        type="button"
                        disabled={busy || !!pullingName}
                        onClick={() => void deleteModel(row.name)}
                        className="px-2 py-1 text-xs bg-red-900/50 text-red-200 rounded hover:bg-red-800/60 disabled:opacity-40"
                      >
                        {deletingName === row.name ? 'Removing…' : 'Remove'}
                      </button>
                    </>
                  )}
                </div>
              </div>
            );
          })
        )}
      </div>

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
}
