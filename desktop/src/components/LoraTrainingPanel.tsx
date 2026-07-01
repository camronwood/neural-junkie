import { useCallback, useEffect, useMemo, useState } from 'react';
import { ChatAPI, type LoraTrainJob, type LoraTrainStartRequest, type LoraTrainingBase } from '../api/chatAPI';
import type { Channel, Collaboration, Message } from '../types/protocol';

interface SelectOption {
  id: string;
  label: string;
}

function channelLabel(name: string, displayName?: string): string {
  const display = displayName?.trim();
  if (display && display !== name) return `${display} (${name})`;
  return name;
}

function buildChannelSourceOptions(channels: Channel[]): SelectOption[] {
  return channels
    .filter((c) => c.type !== 'collaboration')
    .map((c) => ({ id: c.name, label: channelLabel(c.name, c.display_name) }))
    .sort((a, b) => a.label.localeCompare(b.label));
}

function buildCollaborationSourceOptions(collabs: Collaboration[]): SelectOption[] {
  return collabs
    .map((c) => ({
      id: c.id,
      label: c.title?.trim() ? `${c.title.trim()} (${c.id.slice(0, 8)})` : c.id,
      updated: c.updated_at,
    }))
    .sort((a, b) => b.updated.localeCompare(a.updated))
    .map(({ id, label }) => ({ id, label }));
}

function buildRepoSourceOptions(channels: Channel[]): SelectOption[] {
  const options: SelectOption[] = [];
  for (const c of channels) {
    const repoAgents = (c.agents ?? []).filter((a) => a.type === 'repo');
    if (repoAgents.length === 0) continue;
    const names = repoAgents.map((a) => a.name).join(', ');
    options.push({
      id: c.name,
      label: `${channelLabel(c.name, c.display_name)} — ${names}`,
    });
  }
  return options.sort((a, b) => a.label.localeCompare(b.label));
}

function sourceFieldLabel(source: SourceKind): string {
  switch (source) {
    case 'channel':
      return 'Channel';
    case 'collaboration':
      return 'Collaboration';
    case 'repo':
      return 'Repo expert channel';
  }
}

function pickSourceId(options: SelectOption[], current: string, fallback?: string): string {
  const cur = current.trim();
  if (cur && options.some((o) => o.id === cur)) return cur;
  const fb = fallback?.trim();
  if (fb && options.some((o) => o.id === fb)) return fb;
  return options[0]?.id ?? '';
}

function threadPreview(content: string, maxLen = 56): string {
  const oneLine = content.replace(/\s+/g, ' ').trim();
  if (!oneLine) return 'Thread';
  return oneLine.length <= maxLen ? oneLine : `${oneLine.slice(0, maxLen - 1)}…`;
}

function buildThreadOptions(messages: Message[]): SelectOption[] {
  const byId = new Map(messages.map((m) => [m.id, m]));
  const replyCounts = new Map<string, number>();
  for (const m of messages) {
    const tid = m.thread_id?.trim();
    if (tid) {
      replyCounts.set(tid, (replyCounts.get(tid) ?? 0) + 1);
    }
  }
  const options: SelectOption[] = [];
  for (const [tid, count] of replyCounts) {
    const parent = byId.get(tid);
    const preview = parent ? threadPreview(parent.content) : `Thread ${tid.slice(0, 8)}`;
    const replies = count === 1 ? '1 reply' : `${count} replies`;
    options.push({ id: tid, label: `${preview} (${replies})` });
  }
  options.sort((a, b) => {
    const ta = byId.get(a.id)?.timestamp ?? '';
    const tb = byId.get(b.id)?.timestamp ?? '';
    return tb.localeCompare(ta);
  });
  return options;
}

const DEFAULT_CODE_BASE = 'llama3.1:8b';

type SourceKind = 'channel' | 'collaboration' | 'repo';

export interface LoraTrainPrefill {
  source?: SourceKind;
  sourceId?: string;
  agentName?: string;
  baseTag?: string;
  ollamaTag?: string;
  expertName?: string;
  agentId?: string;
  previewRows?: number;
  ready?: boolean;
  prior_adapter_id?: string;
  active_adapter_version?: number;
  refresh_suggested?: boolean;
  supported_bases?: LoraTrainingBase[];
}

interface LoraTrainingPanelProps {
  serverAddr: string;
  defaultChannel?: string;
  switchAgentProvider?: (agentId: string, provider: string, model: string) => Promise<void>;
  runtimeAgents?: { id: string; name: string; type: string }[];
  prefill?: LoraTrainPrefill;
}

export function LoraTrainingPanel({
  serverAddr,
  defaultChannel = '',
  switchAgentProvider,
  runtimeAgents = [],
  prefill,
}: LoraTrainingPanelProps) {
  const [source, setSource] = useState<SourceKind>(prefill?.source ?? 'channel');
  const [sourceId, setSourceId] = useState(prefill?.sourceId ?? defaultChannel);
  const [threadId, setThreadId] = useState('');
  const [sourceOptions, setSourceOptions] = useState<SelectOption[]>([]);
  const [sourcesLoading, setSourcesLoading] = useState(false);
  const [threadOptions, setThreadOptions] = useState<SelectOption[]>([]);
  const [threadsLoading, setThreadsLoading] = useState(false);
  const [agentName, setAgentName] = useState(prefill?.agentName ?? '');
  const [baseTag, setBaseTag] = useState(prefill?.baseTag ?? DEFAULT_CODE_BASE);
  const [ollamaTag, setOllamaTag] = useState(prefill?.ollamaTag ?? 'nj-repo-custom:14b');
  const [trainingBases, setTrainingBases] = useState<LoraTrainingBase[]>([]);
  const [rank, setRank] = useState(16);
  const [epochs, setEpochs] = useState(1);
  const [includeLearnings, setIncludeLearnings] = useState(true);
  const [incremental, setIncremental] = useState(Boolean(prefill?.prior_adapter_id));
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [learningRate, setLearningRate] = useState(2e-4);
  const [maxSeqLen, setMaxSeqLen] = useState(2048);
  const [previewCount, setPreviewCount] = useState<number | null>(prefill?.previewRows ?? null);
  const [job, setJob] = useState<LoraTrainJob | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!prefill) return;
    if (prefill.source) setSource(prefill.source);
    if (prefill.sourceId) setSourceId(prefill.sourceId);
    if (prefill.agentName) setAgentName(prefill.agentName);
    if (prefill.baseTag) setBaseTag(prefill.baseTag);
    if (prefill.ollamaTag) setOllamaTag(prefill.ollamaTag);
    if (prefill.previewRows != null) setPreviewCount(prefill.previewRows);
  }, [prefill]);

  const api = useCallback(() => new ChatAPI(serverAddr), [serverAddr]);

  useEffect(() => {
    let cancelled = false;
    setSourcesLoading(true);
    void (async () => {
      try {
        let options: SelectOption[] = [];
        if (source === 'channel' || source === 'repo') {
          const channels = await api().fetchChannels();
          if (cancelled) return;
          options =
            source === 'channel'
              ? buildChannelSourceOptions(channels)
              : buildRepoSourceOptions(channels);
        } else {
          const collabs = await api().fetchCollaborations(undefined, true);
          if (cancelled) return;
          options = buildCollaborationSourceOptions(collabs);
        }
        setSourceOptions(options);
        setSourceId((current) => pickSourceId(options, current, defaultChannel));
      } catch {
        if (!cancelled) {
          setSourceOptions([]);
          setSourceId('');
        }
      } finally {
        if (!cancelled) setSourcesLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [api, source, defaultChannel]);

  useEffect(() => {
    if (source !== 'channel') {
      setThreadOptions([]);
      setThreadId('');
      return;
    }
    const channel = sourceId.trim();
    if (!channel) {
      setThreadOptions([]);
      return;
    }
    let cancelled = false;
    setThreadsLoading(true);
    void (async () => {
      try {
        const messages = await api().fetchMessages(channel, 500);
        if (cancelled) return;
        const options = buildThreadOptions(messages);
        setThreadOptions(options);
        setThreadId((current) => (current && options.some((o) => o.id === current) ? current : ''));
      } catch {
        if (!cancelled) setThreadOptions([]);
      } finally {
        if (!cancelled) setThreadsLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [api, source, sourceId]);

  useEffect(() => {
    void (async () => {
      try {
        const bases = await api().fetchLoraTrainBases();
        setTrainingBases(bases);
      } catch {
        // hub may be starting; expert-context prefill still works
      }
    })();
  }, [api]);

  useEffect(() => {
    if (prefill?.supported_bases?.length) {
      setTrainingBases(prefill.supported_bases);
    }
  }, [prefill?.supported_bases]);

  const selectedBase = useMemo(
    () => trainingBases.find((b) => b.ollama_tag === baseTag),
    [trainingBases, baseTag],
  );

  const baseSupported = useMemo(() => {
    if (trainingBases.length === 0) {
      return !/qwen/i.test(baseTag);
    }
    return trainingBases.some((b) => b.ollama_tag === baseTag);
  }, [trainingBases, baseTag]);

  const refreshPreview = useCallback(async () => {
    if (!sourceId.trim()) return;
    try {
      const n = await api().previewLoraTrain({
        source,
        source_id: sourceId.trim(),
        thread_id: threadId.trim() || undefined,
        agent_name: agentName.trim() || undefined,
        agent_id: prefill?.agentId,
        include_learnings: includeLearnings,
        incremental,
      });
      setPreviewCount(n);
      setError(null);
    } catch (e) {
      setPreviewCount(null);
      setError(e instanceof Error ? e.message : 'Preview failed');
    }
  }, [api, source, sourceId, threadId, agentName, includeLearnings, incremental, prefill?.agentId]);

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
        agent_id: prefill?.agentId,
        include_learnings: includeLearnings,
        incremental,
        prior_adapter_id: prefill?.prior_adapter_id,
        base_ollama_tag: baseTag.trim(),
        ollama_tag: ollamaTag.trim(),
        hyperparams: { rank, epochs, learning_rate: learningRate, max_seq_len: maxSeqLen },
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

  const minRows = 10;
  const canStart =
    sourceId.trim() !== '' &&
    baseSupported &&
    (previewCount == null || previewCount >= minRows);

  const assignAgents =
    prefill?.agentId && runtimeAgents.some((a) => a.id === prefill.agentId)
      ? runtimeAgents.filter((a) => a.id === prefill.agentId)
      : runtimeAgents.slice(0, 6);

  return (
    <div className="space-y-4 text-sm text-gray-300">
      {prefill?.expertName && (
        <p className="text-sm text-white">
          Training LoRA for <strong>{prefill.expertName}</strong>
        </p>
      )}
      <p className="text-xs text-gray-400">
        Export chat or collaboration data, fine-tune a specialist adapter, then compose it into Ollama. LoRA
        training requires a <strong className="text-gray-300">Llama / Mistral / Gemma</strong> base —{' '}
        <span className="text-amber-300/90">Qwen bases are not supported</span> for compose yet. Training
        dependencies install automatically the first time you start a job.
      </p>

      <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
        <label className="flex flex-col gap-1">
          <span className="text-xs text-gray-500">Source</span>
          <select
            value={source}
            onChange={(e) => {
              setSource(e.target.value as SourceKind);
              setThreadId('');
            }}
            className="rounded border border-slack-border bg-slack-bg px-2 py-1.5"
          >
            <option value="channel">Channel / DM</option>
            <option value="collaboration">Collaboration</option>
            <option value="repo">Repo agent</option>
          </select>
        </label>
        <label className="flex flex-col gap-1">
          <span className="text-xs text-gray-500">{sourceFieldLabel(source)}</span>
          <select
            value={sourceId}
            onChange={(e) => {
              setSourceId(e.target.value);
              setThreadId('');
            }}
            disabled={sourcesLoading || sourceOptions.length === 0}
            className="rounded border border-slack-border bg-slack-bg px-2 py-1.5 text-xs disabled:opacity-50"
          >
            {sourcesLoading && <option value="">Loading…</option>}
            {!sourcesLoading && sourceOptions.length === 0 && (
              <option value="">No options available</option>
            )}
            {sourceOptions.map((opt) => (
              <option key={opt.id} value={opt.id}>
                {opt.label}
              </option>
            ))}
          </select>
          {!sourcesLoading && sourceOptions.length === 0 && (
            <span className="text-[10px] text-gray-500">
              {source === 'repo'
                ? 'Add a repo expert to a channel first.'
                : source === 'collaboration'
                  ? 'Start a collaboration to train from its completed tasks.'
                  : 'Open or create a channel to train from chat history.'}
            </span>
          )}
        </label>
        {source === 'channel' && (
          <label className="flex flex-col gap-1 sm:col-span-2">
            <span className="text-xs text-gray-500">Conversation thread (optional)</span>
            <select
              value={threadId}
              onChange={(e) => setThreadId(e.target.value)}
              disabled={threadsLoading || threadOptions.length === 0}
              className="rounded border border-slack-border bg-slack-bg px-2 py-1.5 text-xs disabled:opacity-50"
            >
              <option value="">Entire channel</option>
              {threadOptions.map((opt) => (
                <option key={opt.id} value={opt.id}>
                  {opt.label}
                </option>
              ))}
            </select>
            <span className="text-[10px] text-gray-500">
              {threadsLoading
                ? 'Loading threads…'
                : threadOptions.length === 0
                  ? 'No threaded conversations in recent channel history — training uses the full channel.'
                  : 'Train on one thread instead of all messages in the channel.'}
            </span>
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
        <label className="flex flex-col gap-1 sm:col-span-2">
          <span className="text-xs text-gray-500">Base Ollama tag</span>
          {trainingBases.length > 0 ? (
            <select
              value={baseTag}
              onChange={(e) => setBaseTag(e.target.value)}
              className="rounded border border-slack-border bg-slack-bg px-2 py-1.5 font-mono text-xs"
            >
              {trainingBases.map((b) => (
                <option key={b.ollama_tag} value={b.ollama_tag}>
                  {b.label} ({b.ollama_tag})
                  {b.recommended ? ' — recommended' : ''}
                </option>
              ))}
            </select>
          ) : (
            <input
              value={baseTag}
              onChange={(e) => setBaseTag(e.target.value)}
              className="rounded border border-slack-border bg-slack-bg px-2 py-1.5 font-mono text-xs"
            />
          )}
          {selectedBase && (
            <span className="text-[10px] text-gray-500">{selectedBase.description}</span>
          )}
          {!baseSupported && (
            <span className="text-[10px] text-amber-400">
              {baseTag} is not supported for LoRA training — choose a listed base (not Qwen).
            </span>
          )}
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

      {prefill?.agentId && (
        <label className="flex items-center gap-2 text-xs text-gray-400">
          <input
            type="checkbox"
            checked={includeLearnings}
            onChange={(e) => setIncludeLearnings(e.target.checked)}
            className="rounded border-slack-border"
          />
          Include confirmed personal learnings (up to 50 rows)
        </label>
      )}

      {prefill?.prior_adapter_id && (
        <label className="flex items-center gap-2 text-xs text-gray-400">
          <input
            type="checkbox"
            checked={incremental}
            onChange={(e) => setIncremental(e.target.checked)}
            className="rounded border-slack-border"
          />
          Refresh adapter incrementally
          {prefill.active_adapter_version ? ` (current v${prefill.active_adapter_version})` : ''}
        </label>
      )}

      <button
        type="button"
        className="text-xs text-gray-500 underline"
        onClick={() => setShowAdvanced((v) => !v)}
      >
        {showAdvanced ? 'Hide advanced' : 'Advanced training options'}
      </button>
      {showAdvanced && (
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <label className="flex flex-col gap-1">
            <span className="text-xs text-gray-500">Learning rate</span>
            <input
              type="number"
              step={0.0001}
              value={learningRate}
              onChange={(e) => setLearningRate(Number(e.target.value))}
              className="rounded border border-slack-border bg-slack-bg px-2 py-1.5"
            />
          </label>
          <label className="flex flex-col gap-1">
            <span className="text-xs text-gray-500">Max sequence length</span>
            <input
              type="number"
              min={512}
              max={8192}
              value={maxSeqLen}
              onChange={(e) => setMaxSeqLen(Number(e.target.value))}
              className="rounded border border-slack-border bg-slack-bg px-2 py-1.5"
            />
          </label>
        </div>
      )}

      {previewCount != null && (
        <p className={`text-xs ${previewCount >= minRows ? 'text-gray-500' : 'text-amber-400'}`}>
          Preview: {previewCount} training rows (minimum {minRows} required)
        </p>
      )}
      {error && <p className="text-xs text-red-400">{error}</p>}

      <button
        type="button"
        disabled={busy || !canStart}
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
          {job.status !== 'done' && job.status !== 'failed' && job.status !== 'cancelled' && (
            <button
              type="button"
              onClick={() => void api().cancelLoraTrainJob(job.id).then(setJob)}
              className="px-2 py-1 text-[10px] rounded border border-red-500/40 text-red-300"
            >
              Cancel job
            </button>
          )}
          {job.log_tail && job.log_tail.length > 0 && (
            <pre className="text-[10px] font-mono text-gray-500 max-h-32 overflow-y-auto whitespace-pre-wrap">
              {job.log_tail.slice(-12).join('\n')}
            </pre>
          )}
          {job.status === 'done' && switchAgentProvider && assignAgents.length > 0 && (
            <div className="flex flex-wrap gap-2 pt-1">
              {assignAgents.map((a) => (
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
