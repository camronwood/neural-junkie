import { useCallback, useEffect, useState } from 'react';
import { ChatAPI, type LoraTrainJob, type LoraTrainStartRequest } from '../api/chatAPI';

type SourceKind = 'channel' | 'collaboration' | 'repo';

interface LoraTrainingPanelProps {
  serverAddr: string;
  defaultChannel?: string;
  switchAgentProvider?: (agentId: string, provider: string, model: string) => Promise<void>;
  runtimeAgents?: { id: string; name: string; type: string }[];
}

export function LoraTrainingPanel({
  serverAddr,
  defaultChannel = '',
  switchAgentProvider,
  runtimeAgents = [],
}: LoraTrainingPanelProps) {
  const [source, setSource] = useState<SourceKind>('channel');
  const [sourceId, setSourceId] = useState(defaultChannel);
  const [threadId, setThreadId] = useState('');
  const [agentName, setAgentName] = useState('');
  const [baseTag, setBaseTag] = useState('qwen2.5-coder:14b');
  const [ollamaTag, setOllamaTag] = useState('nj-repo-custom:14b');
  const [rank, setRank] = useState(16);
  const [epochs, setEpochs] = useState(1);
  const [previewCount, setPreviewCount] = useState<number | null>(null);
  const [job, setJob] = useState<LoraTrainJob | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (defaultChannel && !sourceId) {
      setSourceId(defaultChannel);
    }
  }, [defaultChannel, sourceId]);

  const api = useCallback(() => new ChatAPI(serverAddr), [serverAddr]);

  const refreshPreview = useCallback(async () => {
    if (!sourceId.trim()) return;
    try {
      const n = await api().previewLoraTrain({
        source,
        source_id: sourceId.trim(),
        thread_id: threadId.trim() || undefined,
        agent_name: agentName.trim() || undefined,
      });
      setPreviewCount(n);
      setError(null);
    } catch (e) {
      setPreviewCount(null);
      setError(e instanceof Error ? e.message : 'Preview failed');
    }
  }, [api, source, sourceId, threadId, agentName]);

  useEffect(() => {
    void refreshPreview();
  }, [refreshPreview]);

  useEffect(() => {
    if (!job?.id || job.status === 'done' || job.status === 'failed' || job.status === 'cancelled') {
      return;
    }
    const t = window.setInterval(async () => {
      try {
        const updated = await api().fetchLoraTrainJob(job.id);
        setJob(updated);
      } catch {
        // ignore poll errors
      }
    }, 2000);
    return () => window.clearInterval(t);
  }, [api, job?.id, job?.status]);

  const start = async () => {
    setBusy(true);
    setError(null);
    try {
      const body: LoraTrainStartRequest = {
        source,
        source_id: sourceId.trim(),
        thread_id: threadId.trim() || undefined,
        agent_name: agentName.trim() || undefined,
        base_ollama_tag: baseTag.trim(),
        ollama_tag: ollamaTag.trim(),
        hyperparams: { rank, epochs, learning_rate: 2e-4 },
      };
      const started = await api().startLoraTrain(body);
      setJob(started);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Start failed');
    } finally {
      setBusy(false);
    }
  };

  const assignToAgent = async (agentId: string) => {
    if (!switchAgentProvider || !ollamaTag.trim()) return;
    await switchAgentProvider(agentId, 'ollama-local', ollamaTag.trim());
  };

  return (
    <div className="space-y-4 text-sm text-gray-300">
      <p className="text-xs text-gray-400">
        Export chat or collaboration data, fine-tune with Unsloth, then compose into Ollama. Python deps install with{' '}
        <span className="font-mono text-gray-500">make deps</span> (.venv-lora). See{' '}
        <span className="font-mono text-gray-500">docs/LORA_TRAINING.md</span>.
      </p>

      <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
        <label className="flex flex-col gap-1">
          <span className="text-xs text-gray-500">Source</span>
          <select
            value={source}
            onChange={(e) => setSource(e.target.value as SourceKind)}
            className="rounded border border-slack-border bg-slack-bg px-2 py-1.5"
          >
            <option value="channel">Channel / DM</option>
            <option value="collaboration">Collaboration</option>
            <option value="repo">Repo agent</option>
          </select>
        </label>
        <label className="flex flex-col gap-1">
          <span className="text-xs text-gray-500">Source ID</span>
          <input
            value={sourceId}
            onChange={(e) => setSourceId(e.target.value)}
            placeholder="channel name or collaboration id"
            className="rounded border border-slack-border bg-slack-bg px-2 py-1.5 font-mono text-xs"
          />
        </label>
        {source === 'channel' && (
          <label className="flex flex-col gap-1 sm:col-span-2">
            <span className="text-xs text-gray-500">Thread ID (optional)</span>
            <input
              value={threadId}
              onChange={(e) => setThreadId(e.target.value)}
              className="rounded border border-slack-border bg-slack-bg px-2 py-1.5 font-mono text-xs"
            />
          </label>
        )}
        {source === 'repo' && (
          <label className="flex flex-col gap-1 sm:col-span-2">
            <span className="text-xs text-gray-500">Agent name filter</span>
            <input
              value={agentName}
              onChange={(e) => setAgentName(e.target.value)}
              placeholder="MyAppExpert"
              className="rounded border border-slack-border bg-slack-bg px-2 py-1.5"
            />
          </label>
        )}
        <label className="flex flex-col gap-1">
          <span className="text-xs text-gray-500">Base Ollama tag</span>
          <input
            value={baseTag}
            onChange={(e) => setBaseTag(e.target.value)}
            className="rounded border border-slack-border bg-slack-bg px-2 py-1.5 font-mono text-xs"
          />
        </label>
        <label className="flex flex-col gap-1">
          <span className="text-xs text-gray-500">Output composed tag</span>
          <input
            value={ollamaTag}
            onChange={(e) => setOllamaTag(e.target.value)}
            className="rounded border border-slack-border bg-slack-bg px-2 py-1.5 font-mono text-xs"
          />
        </label>
        <label className="flex flex-col gap-1">
          <span className="text-xs text-gray-500">LoRA rank</span>
          <input
            type="number"
            min={4}
            max={128}
            value={rank}
            onChange={(e) => setRank(Number(e.target.value))}
            className="rounded border border-slack-border bg-slack-bg px-2 py-1.5"
          />
        </label>
        <label className="flex flex-col gap-1">
          <span className="text-xs text-gray-500">Epochs</span>
          <input
            type="number"
            min={1}
            max={10}
            value={epochs}
            onChange={(e) => setEpochs(Number(e.target.value))}
            className="rounded border border-slack-border bg-slack-bg px-2 py-1.5"
          />
        </label>
      </div>

      {previewCount != null && (
        <p className="text-xs text-gray-500">
          Preview: {previewCount} training rows (minimum 10 required)
        </p>
      )}
      {error && <p className="text-xs text-red-400">{error}</p>}

      <button
        type="button"
        disabled={busy || !sourceId.trim()}
        onClick={() => void start()}
        className="px-4 py-2 rounded-lg bg-purple-700 text-white text-xs font-medium hover:bg-purple-600 disabled:opacity-40"
      >
        {busy ? 'Starting…' : 'Start training'}
      </button>

      {job && (
        <div className="rounded-lg border border-slack-border bg-slack-bgHover/40 p-3 space-y-2">
          <p className="text-xs font-mono">
            Job {job.id.slice(0, 8)} — <span className="text-purple-300">{job.status}</span>
            {job.row_count ? ` (${job.row_count} rows)` : ''}
          </p>
          {job.error && <p className="text-xs text-red-400">{job.error}</p>}
          {job.log_tail && job.log_tail.length > 0 && (
            <pre className="text-[10px] font-mono text-gray-500 max-h-32 overflow-y-auto whitespace-pre-wrap">
              {job.log_tail.slice(-12).join('\n')}
            </pre>
          )}
          {job.status === 'done' && switchAgentProvider && runtimeAgents.length > 0 && (
            <div className="flex flex-wrap gap-2 pt-1">
              {runtimeAgents.slice(0, 6).map((a) => (
                <button
                  key={a.id}
                  type="button"
                  onClick={() => void assignToAgent(a.id)}
                  className="px-2 py-1 text-[10px] rounded bg-teal-800/60 text-teal-100 hover:bg-teal-700/60"
                >
                  Assign to {a.name}
                </button>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
