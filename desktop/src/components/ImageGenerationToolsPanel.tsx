import { useCallback, useEffect, useState } from 'react';
import type { ImageGenStatus } from '../api/chatAPI';
import { ChatAPI } from '../api/chatAPI';

export interface ImageGenerationToolsPanelProps {
  hubHttp: string;
  isActive: boolean;
}

function statusLabel(st: ImageGenStatus | null): string {
  if (!st) return 'Checking…';
  if (st.disabled) return 'Disabled (NEURAL_JUNKIE_IMAGE_PROVIDER=none)';
  if (st.provider === 'openai') {
    return st.ready ? 'Ready (OpenAI)' : 'OpenAI API key not configured';
  }
  if (!st.ollama_running) return 'Ollama not running';
  if (!st.model_pulled) return 'Image model not pulled';
  return 'Ready';
}

function statusClass(st: ImageGenStatus | null): string {
  if (!st) return 'text-slack-textMuted';
  if (st.ready) return 'text-green-400';
  if (st.disabled) return 'text-gray-400';
  return 'text-amber-400';
}

async function parseSSEChunks(
  reader: ReadableStreamDefaultReader<Uint8Array>,
  onData: (data: Record<string, unknown>) => void,
): Promise<void> {
  const decoder = new TextDecoder();
  let buffer = '';
  for (;;) {
    const { done, value } = await reader.read();
    if (done) break;
    buffer += decoder.decode(value, { stream: true });
    const parts = buffer.split('\n\n');
    buffer = parts.pop() ?? '';
    for (const part of parts) {
      for (const line of part.split('\n')) {
        if (!line.startsWith('data: ')) continue;
        const raw = line.slice(6).trim();
        if (!raw) continue;
        try {
          onData(JSON.parse(raw) as Record<string, unknown>);
        } catch {
          // ignore malformed chunks
        }
      }
    }
  }
}

export function ImageGenerationToolsPanel({ hubHttp, isActive }: ImageGenerationToolsPanelProps) {
  const [status, setStatus] = useState<ImageGenStatus | null>(null);
  const [loading, setLoading] = useState(false);
  const [pulling, setPulling] = useState(false);
  const [pullProgress, setPullProgress] = useState('');
  const [startingOllama, setStartingOllama] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [ok, setOk] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const api = new ChatAPI(hubHttp);
      setStatus(await api.fetchImageGenStatus());
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }, [hubHttp]);

  useEffect(() => {
    if (!isActive) return;
    void refresh();
  }, [isActive, refresh]);

  const runPull = async () => {
    if (!status?.model) return;
    const confirmed = window.confirm(
      `Pull Ollama image model ${status.model}?\n\n` +
        'This downloads several GB and may take a while depending on your connection.',
    );
    if (!confirmed) return;
    setPulling(true);
    setPullProgress('Starting…');
    setError(null);
    setOk(null);
    let streamError: string | null = null;
    try {
      const resp = await fetch(`${hubHttp}/api/ollama/pull`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ model: status.model }),
      });
      if (!resp.ok) {
        const t = await resp.text();
        throw new Error(t.trim() || resp.statusText);
      }
      const reader = resp.body?.getReader();
      if (!reader) throw new Error('No response body');
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
        setError(streamError);
      } else {
        setOk(`Model ${status.model} pulled. Try /generate-image in chat.`);
      }
      await refresh();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      await refresh();
    } finally {
      setPulling(false);
      setPullProgress('');
    }
  };

  const runStartOllama = async () => {
    setStartingOllama(true);
    setError(null);
    setOk(null);
    try {
      const resp = await fetch(`${hubHttp}/api/ollama/start`, { method: 'POST' });
      if (!resp.ok) {
        const t = await resp.text();
        throw new Error(t.trim() || resp.statusText);
      }
      setOk('Ollama started.');
      await refresh();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setStartingOllama(false);
    }
  };

  const busy = loading || pulling || startingOllama;
  const ollamaMode = status?.provider === 'ollama';
  const title =
    status?.provider === 'openai'
      ? 'Image generation — OpenAI'
      : status?.disabled
        ? 'Image generation'
        : 'Image generation — Ollama';

  return (
    <div className="rounded-lg border border-slack-border p-4">
      <h3 className="text-base font-semibold text-slack-text mb-2">{title}</h3>
      <p className="text-sm text-slack-textMuted mb-3">
        Agents use <code className="font-mono text-xs">generate_image</code> or{' '}
        <code className="font-mono text-xs">/generate-image</code> to create visuals. Local mode uses
        an Ollama image model (default{' '}
        <code className="font-mono text-xs">x/flux2-klein:4b</code>). The model unloads after each
        run to free VRAM for chat models.
      </p>
      <p className={`text-sm font-medium mb-3 ${statusClass(status)}`}>
        {pullProgress ? `Pulling… ${pullProgress}` : statusLabel(status)}
      </p>
      {status && !status.disabled && (
        <ul className="mb-3 space-y-1 text-xs font-mono text-slack-textMuted">
          <li>Provider: {status.provider}</li>
          <li>Model: {status.model}</li>
          {status.endpoint && <li>Endpoint: {status.endpoint}</li>}
          {ollamaMode && (
            <>
              <li>Ollama: {status.ollama_running ? '✓ running' : '— not running'}</li>
              <li>Model pulled: {status.model_pulled ? '✓' : '—'}</li>
            </>
          )}
          {status.provider === 'openai' && (
            <li>OpenAI key: {status.openai_key_set ? '✓ configured' : '— set OPENAI_API_KEY'}</li>
          )}
        </ul>
      )}
      <div className="flex flex-wrap gap-2">
        {ollamaMode && !status?.ollama_running && (
          <button
            type="button"
            disabled={busy}
            onClick={() => void runStartOllama()}
            className="rounded bg-slack-accent px-4 py-2 text-sm text-white hover:bg-slack-accentHover disabled:opacity-50"
          >
            {startingOllama ? 'Starting…' : 'Start Ollama'}
          </button>
        )}
        {ollamaMode && status?.ollama_running && !status.model_pulled && (
          <button
            type="button"
            disabled={busy}
            onClick={() => void runPull()}
            className="rounded bg-slack-accent px-4 py-2 text-sm text-white hover:bg-slack-accentHover disabled:opacity-50"
          >
            {pulling ? 'Pulling…' : `Pull ${status.model}`}
          </button>
        )}
        <button
          type="button"
          disabled={busy}
          onClick={() => void refresh()}
          className="rounded border border-slack-border px-4 py-2 text-sm text-slack-text hover:bg-slack-bgHover disabled:opacity-50"
        >
          Refresh status
        </button>
      </div>
      {error && <p className="mt-2 text-sm text-red-500">{error}</p>}
      {ok && <p className="mt-2 text-sm text-green-600">{ok}</p>}
      {status && !status.ready && !status.disabled && ollamaMode && status.pull_command && (
        <p className="mt-3 text-xs text-slack-textMuted">
          CLI fallback: <code className="font-mono">{status.pull_command}</code>
        </p>
      )}
      {status?.provider === 'openai' && !status.ready && (
        <p className="mt-3 text-xs text-slack-textMuted">
          Set <code className="font-mono">OPENAI_API_KEY</code> and restart the hub, or switch to
          local Ollama by unsetting <code className="font-mono">NEURAL_JUNKIE_IMAGE_PROVIDER</code>.
        </p>
      )}
    </div>
  );
}
