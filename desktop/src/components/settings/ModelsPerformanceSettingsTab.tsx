import { useState, useEffect } from 'react';
import { useSettingsStore } from '../../stores/settingsStore';
import { type SettingsTabProps } from './settingsShared';
import { useHubSettingsSnapshot } from './useHubSettingsSnapshot';

export function ModelsPerformanceSettingsTab({ hubHttp, isActive }: SettingsTabProps) {

  const { layoutSettings, updateLayoutSettings, integrations, loadIntegrations, fetchOllamaModels } =
    useSettingsStore();
  const [ollamaModels, setOllamaModels] = useState<string[]>(integrations.ollama.availableModels ?? []);
  const [hfHubToken, setHfHubToken] = useState('');
  const [hfHubTokenPersisted, setHfHubTokenPersisted] = useState('');
  const [hfTokenSaving, setHfTokenSaving] = useState(false);
  const [hfTokenErr, setHfTokenErr] = useState<string | null>(null);
  const [hfTokenOk, setHfTokenOk] = useState<string | null>(null);
  const [configuredAgents, setConfiguredAgents] = useState<
    { type: string; name: string; enabled: boolean; model?: string }[]
  >([]);
  const [agentModelsSaving, setAgentModelsSaving] = useState(false);
  const [agentModelsErr, setAgentModelsErr] = useState<string | null>(null);
  const [agentModelsOk, setAgentModelsOk] = useState<string | null>(null);
  const [specialistModelsAdvancedOpen, setSpecialistModelsAdvancedOpen] = useState(false);
  const [perfForm, setPerfForm] = useState({
    contextBudgetKB: 32,
    ideContextBudgetKB: 48,
    implSessionBudgetKB: 64,
    maxHistoryMessages: 10,
    ollamaNumCtx: 0,
    ollamaNumPredict: 0,
    ollamaKeepAlive: '',
  });
  const [perfSaving, setPerfSaving] = useState(false);
  const [perfFeedback, setPerfFeedback] = useState<{ success: boolean; message: string } | null>(null);

  const { config, save, reload } = useHubSettingsSnapshot(hubHttp, isActive);

  useEffect(() => {
    if (!isActive) return;
    loadIntegrations();
  }, [isActive, loadIntegrations]);

  useEffect(() => {
    setOllamaModels(integrations.ollama.availableModels ?? []);
  }, [integrations.ollama.availableModels]);

  useEffect(() => {
    if (!isActive) return;
    let cancelled = false;
    void fetchOllamaModels()
      .then((models) => {
        if (!cancelled) setOllamaModels(models);
      })
      .catch(() => {});
    return () => {
      cancelled = true;
    };
  }, [isActive, fetchOllamaModels]);

  useEffect(() => {
    if (!isActive || !config) return;
    const hfTok = typeof config.hf === 'object' && config.hf && typeof (config.hf as { token?: string }).token === 'string'
      ? (config.hf as { token: string }).token
      : '';
    const redacted = hfTok.includes('...') || hfTok === '***';
    setHfHubToken(redacted ? '' : hfTok);
    setHfHubTokenPersisted(redacted ? '***' : hfTok);
    const perf = config.performance as Record<string, number> | undefined;
    const ollamaCfg = config.ollama as Record<string, unknown> | undefined;
    setPerfForm({
      contextBudgetKB: perf?.context_budget_kb || 32,
      ideContextBudgetKB: perf?.ide_context_budget_kb || 48,
      implSessionBudgetKB: perf?.impl_session_budget_kb || 64,
      maxHistoryMessages: perf?.max_history_messages || 10,
      ollamaNumCtx: typeof ollamaCfg?.num_ctx === 'number' ? ollamaCfg.num_ctx : 0,
      ollamaNumPredict: typeof ollamaCfg?.num_predict === 'number' ? ollamaCfg.num_predict : 0,
      ollamaKeepAlive: typeof ollamaCfg?.keep_alive === 'string' ? ollamaCfg.keep_alive : '',
    });
  }, [isActive, config]);

  useEffect(() => {
    if (!isActive) return;
    let cancelled = false;
    (async () => {
      try {
        const r = await fetch(`${hubHttp}/api/agents/configured`);
        if (!r.ok) throw new Error(await r.text());
        const rows = (await r.json()) as { type: string; name: string; enabled: boolean; model?: string }[];
        if (!cancelled) setConfiguredAgents(rows);
      } catch (e) {
        if (!cancelled) setAgentModelsErr(e instanceof Error ? e.message : String(e));
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [isActive, hubHttp]);

  const saveHfHubToken = async () => {
    setHfTokenSaving(true);
    setHfTokenErr(null);
    setHfTokenOk(null);
    try {
      const trimmed = hfHubToken.trim();
      await save((cfg) => ({
        ...cfg,
        hf: { ...(cfg.hf as object | undefined), token: trimmed },
      }));
      setHfHubTokenPersisted(trimmed ? '***' : '');
      setHfTokenOk(trimmed ? 'Hugging Face token saved.' : 'Cleared hub Hugging Face token.');
    } catch (e) {
      setHfTokenErr(e instanceof Error ? e.message : String(e));
    } finally {
      setHfTokenSaving(false);
    }
  };

  const savePerformanceSettings = async () => {
    setPerfSaving(true);
    setPerfFeedback(null);
    try {
      await save((cfg) => ({
        ...cfg,
        performance: {
          context_budget_kb: perfForm.contextBudgetKB,
          ide_context_budget_kb: perfForm.ideContextBudgetKB,
          impl_session_budget_kb: perfForm.implSessionBudgetKB,
          max_history_messages: perfForm.maxHistoryMessages,
        },
        ollama: {
          ...(typeof cfg.ollama === 'object' && cfg.ollama ? cfg.ollama : {}),
          num_ctx: perfForm.ollamaNumCtx > 0 ? perfForm.ollamaNumCtx : 0,
          num_predict: perfForm.ollamaNumPredict > 0 ? perfForm.ollamaNumPredict : 0,
          keep_alive: perfForm.ollamaKeepAlive.trim(),
        },
      }));
      setPerfFeedback({ success: true, message: 'Performance settings saved.' });
    } catch (e) {
      setPerfFeedback({
        success: false,
        message: e instanceof Error ? e.message : 'Failed to save performance settings',
      });
    } finally {
      setPerfSaving(false);
    }
  };

  const saveConfiguredAgentModels = async () => {
    setAgentModelsSaving(true);
    setAgentModelsErr(null);
    setAgentModelsOk(null);
    try {
      await save((cfg) => {
        const existing = Array.isArray(cfg.agents) ? cfg.agents : [];
        const byKey = new Map(configuredAgents.map((a) => [`${a.type}\x00${a.name}`, a]));
        const agents = existing.map((row: { type: string; name: string; model?: string }) => {
          const hit = byKey.get(`${row.type}\x00${row.name}`);
          if (!hit) return row;
          return { ...row, model: hit.model?.trim() || undefined };
        });
        return { ...cfg, agents };
      });
      await fetch(`${hubHttp}/api/agents/restart`, { method: 'POST' });
      setAgentModelsOk('Specialist models saved. Agents restarted.');
      await reload();
    } catch (e) {
      setAgentModelsErr(e instanceof Error ? e.message : String(e));
    } finally {
      setAgentModelsSaving(false);
    }
  };

  if (!isActive) return null;

  return (
    <div className="space-y-8">
    <div className="mb-4">
      <h3 className="text-lg font-semibold text-slack-text mb-2">Agent &amp; editor behavior</h3>
      <p className="text-sm text-slack-textMuted mb-3">
        Controls for memory monitoring, routing visibility, inline completion, and IDE file-change trust.
      </p>
    </div>

    <div className="flex items-center justify-between p-4 bg-slack-bgHover rounded-lg border border-slack-border">
      <div className="flex-1">
        <div className="font-medium text-slack-text">Memory monitor</div>
        <div className="text-sm text-slack-textMuted">
          Show live system RAM and Ollama loaded-model usage in the toolbar
        </div>
      </div>
      <label className="relative inline-flex items-center cursor-pointer">
        <input
          type="checkbox"
          checked={layoutSettings.memoryMonitorEnabled !== false}
          onChange={(e) => updateLayoutSettings({ memoryMonitorEnabled: e.target.checked })}
          className="sr-only peer"
        />
        <div className="w-11 h-6 bg-gray-600 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
      </label>
    </div>

    <div className="flex items-center justify-between p-4 bg-slack-bgHover rounded-lg border border-slack-border">
      <div className="flex-1">
        <div className="font-medium text-slack-text">Routing badges on messages</div>
        <div className="text-sm text-slack-textMuted">
          Show which model ran on each agent reply (chat model, tool model, routing reason)
        </div>
      </div>
      <label className="relative inline-flex items-center cursor-pointer">
        <input
          type="checkbox"
          checked={layoutSettings.showRoutingOnMessages !== false}
          onChange={(e) => updateLayoutSettings({ showRoutingOnMessages: e.target.checked })}
          className="sr-only peer"
        />
        <div className="w-11 h-6 bg-gray-600 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
      </label>
    </div>

    <div className="flex items-center justify-between p-4 bg-slack-bgHover rounded-lg border border-slack-border">
      <div className="flex-1">
        <div className="font-medium text-slack-text">Compression badges on messages</div>
        <div className="text-sm text-slack-textMuted">
          Show when tool output was compressed (strategy and byte savings)
        </div>
      </div>
      <label className="relative inline-flex items-center cursor-pointer">
        <input
          type="checkbox"
          checked={layoutSettings.showCompressOnMessages === true}
          onChange={(e) => updateLayoutSettings({ showCompressOnMessages: e.target.checked })}
          className="sr-only peer"
        />
        <div className="w-11 h-6 bg-gray-600 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
      </label>
    </div>

    <div className="flex items-center justify-between p-4 bg-slack-bgHover rounded-lg border border-slack-border">
      <div className="flex-1">
        <div className="font-medium text-slack-text">Inline completion (ghost text)</div>
        <div className="text-sm text-slack-textMuted">Ollama FIM via hub when Software development pack is on</div>
      </div>
      <label className="relative inline-flex items-center cursor-pointer">
        <input
          type="checkbox"
          checked={layoutSettings.inlineCompletionEnabled ?? false}
          onChange={(e) => updateLayoutSettings({ inlineCompletionEnabled: e.target.checked })}
          className="sr-only peer"
        />
        <div className="w-11 h-6 bg-gray-600 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
      </label>
    </div>

    <div className="flex items-center justify-between p-4 bg-slack-bgHover rounded-lg border border-slack-border">
      <div className="flex-1">
        <div className="font-medium text-slack-text">Editor agent trust</div>
        <div className="text-sm text-slack-textMuted">
          How file changes from IDE-mode chat are applied (Ask/Agent toggle on the main composer)
        </div>
      </div>
      <select
        value={layoutSettings.editorAgentTrust ?? 'interactive'}
        onChange={(e) =>
          updateLayoutSettings({
            editorAgentTrust: e.target.value as 'interactive' | 'auto_apply_edits' | 'yolo',
          })
        }
        className="text-sm bg-slack-bg border border-slack-border rounded px-2 py-1"
      >
        <option value="interactive">Interactive (approve each)</option>
        <option value="auto_apply_edits">Auto-apply edits</option>
        <option value="yolo">Yolo (tools)</option>
      </select>
    </div>
<div className="border border-slack-border rounded-lg p-6">
      <h3 className="text-lg font-semibold text-slack-text mb-2">Hugging Face hub token</h3>
      <p className="text-sm text-slack-textMuted mb-4">
        Used for gated model downloads, hosted inference, and <strong>ESMFold</strong> structure prediction. You can also add a{' '}
        <code className="font-mono text-xs bg-slack-bgHover px-1 rounded">huggingface</code> provider below.
      </p>
      {hfHubTokenPersisted === '***' && !hfHubToken && (
        <p className="text-xs text-slack-textMuted mb-2">A token is saved on the hub (hidden). Enter a new value to replace it.</p>
      )}
      <div className="flex flex-col sm:flex-row gap-2 mb-2">
        <input
          type="password"
          value={hfHubToken}
          onChange={(e) => {
            setHfHubToken(e.target.value);
            setHfTokenOk(null);
          }}
          placeholder="hf_…"
          disabled={hfTokenSaving}
          className="flex-1 px-3 py-2 text-sm border border-slack-border rounded bg-slack-bg text-slack-text font-mono"
          autoComplete="off"
        />
        <button
          type="button"
          onClick={() => void saveHfHubToken()}
          disabled={hfTokenSaving}
          className="px-4 py-2 text-sm bg-slack-accent text-white rounded hover:bg-slack-accentHover disabled:opacity-50"
        >
          {hfTokenSaving ? 'Saving…' : 'Save token'}
        </button>
      </div>
      {hfTokenErr && <p className="text-sm text-red-600">{hfTokenErr}</p>}
      {hfTokenOk && <p className="text-sm text-green-600">{hfTokenOk}</p>}
    </div>
<details
      open={specialistModelsAdvancedOpen}
      onToggle={(e) => setSpecialistModelsAdvancedOpen(e.currentTarget.open)}
      className="border border-slack-border rounded-lg p-6"
    >
      <summary className="cursor-pointer text-lg font-semibold text-slack-text">
        Advanced — specialist model overrides
      </summary>
      <p className="text-sm text-slack-textMuted mt-4 mb-4">
        Bulk-edit per-agent Ollama tags in hub config (including composed LoRA tags like{' '}
        <code className="font-mono text-xs bg-slack-bgHover px-1 rounded">nj-security:14b</code>).
        For most cases, use agent info (ℹ️) → provider/model or Model library assign after compose/train.
        Leave blank to use the provider default. Saves on button click and restarts agents.
      </p>
      {configuredAgents.length === 0 ? (
        <p className="text-sm text-slack-textMuted">No configured specialists. Enable a domain pack first.</p>
      ) : (
        <ul className="space-y-3">
          {configuredAgents.map((a) => (
            <li key={`${a.type}-${a.name}`} className="flex flex-col sm:flex-row sm:items-center gap-2">
              <span className="text-sm text-slack-text sm:w-48 shrink-0">
                {a.name}{' '}
                <span className="text-slack-textMuted">({a.type})</span>
              </span>
              <input
                type="text"
                list="nj-ollama-model-options"
                value={a.model ?? ''}
                onChange={(e) =>
                  setConfiguredAgents((prev) =>
                    prev.map((row) =>
                      row.type === a.type && row.name === a.name
                        ? { ...row, model: e.target.value }
                        : row
                    )
                  )
                }
                placeholder={integrations.ollama.defaultModel || 'qwen2.5-coder:14b'}
                className="flex-1 px-3 py-2 text-sm border border-slack-border rounded bg-slack-bg text-slack-text font-mono"
              />
            </li>
          ))}
        </ul>
      )}
      <datalist id="nj-ollama-model-options">
        {ollamaModels.map((m) => (
          <option key={m} value={m} />
        ))}
      </datalist>
      <div className="mt-4 flex items-center gap-3">
        <button
          type="button"
          disabled={agentModelsSaving || configuredAgents.length === 0}
          onClick={() => void saveConfiguredAgentModels()}
          className="px-4 py-2 text-sm bg-slack-accent text-white rounded hover:bg-slack-accentHover disabled:opacity-50"
        >
          {agentModelsSaving ? 'Saving…' : 'Save specialist models'}
        </button>
        {agentModelsErr && <p className="text-sm text-red-600">{agentModelsErr}</p>}
        {agentModelsOk && !agentModelsErr && (
          <p className="text-sm text-green-600">{agentModelsOk}</p>
        )}
      </div>
    </details>
<div className="border border-slack-border rounded-lg p-4 bg-slack-bgHover/30">
      <p className="text-sm text-slack-text">
        <strong className="font-medium">Model library</strong> — browse, download, and install Ollama and
        Hugging Face models from the chat toolbar (amber icon),{' '}
        <kbd className="font-mono text-xs px-1 rounded bg-slack-bgHover">⇧⌘M</kbd> /{' '}
        <kbd className="font-mono text-xs px-1 rounded bg-slack-bgHover">Ctrl+Shift+M</kbd>, or{' '}
        <code className="font-mono text-xs bg-slack-bgHover px-1 rounded">/nj-open-model-library</code>.
      </p>
    </div>
{/* Performance & context */}
    <div className="border border-slack-border rounded-lg p-6">
      <h3 className="text-lg font-semibold text-slack-text mb-2">Performance &amp; context</h3>
      <p className="text-sm text-slack-textMuted mb-4">
        Tune prompt size and Ollama runtime without switching to smaller models. Lower budgets reduce RAM
        and latency; IDE and implementation sessions keep higher caps when coding.
      </p>
      {perfFeedback && (
        <div
          className={`mb-4 p-3 rounded text-sm ${
            perfFeedback.success
              ? 'bg-green-100 text-green-800 border border-green-200'
              : 'bg-red-100 text-red-800 border border-red-200'
          }`}
        >
          {perfFeedback.message}
        </div>
      )}
      <div className="grid gap-4 sm:grid-cols-2">
        <label className="block text-sm text-slack-text">
          Default prompt budget (KB)
          <input
            type="number"
            min={8}
            max={256}
            value={perfForm.contextBudgetKB}
            onChange={(e) =>
              setPerfForm((p) => ({
                ...p,
                contextBudgetKB: Math.max(8, Math.min(256, Number(e.target.value) || 32)),
              }))
            }
            className="mt-1 w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded"
          />
        </label>
        <label className="block text-sm text-slack-text">
          IDE / file-tab budget (KB)
          <input
            type="number"
            min={16}
            max={512}
            value={perfForm.ideContextBudgetKB}
            onChange={(e) =>
              setPerfForm((p) => ({
                ...p,
                ideContextBudgetKB: Math.max(16, Math.min(512, Number(e.target.value) || 48)),
              }))
            }
            className="mt-1 w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded"
          />
        </label>
        <label className="block text-sm text-slack-text">
          Implementation session budget (KB)
          <input
            type="number"
            min={32}
            max={512}
            value={perfForm.implSessionBudgetKB}
            onChange={(e) =>
              setPerfForm((p) => ({
                ...p,
                implSessionBudgetKB: Math.max(32, Math.min(512, Number(e.target.value) || 64)),
              }))
            }
            className="mt-1 w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded"
          />
        </label>
        <label className="block text-sm text-slack-text">
          Max history messages
          <input
            type="number"
            min={2}
            max={50}
            value={perfForm.maxHistoryMessages}
            onChange={(e) =>
              setPerfForm((p) => ({
                ...p,
                maxHistoryMessages: Math.max(2, Math.min(50, Number(e.target.value) || 10)),
              }))
            }
            className="mt-1 w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded"
          />
        </label>
        <label className="block text-sm text-slack-text">
          Ollama num_ctx (0 = default)
          <input
            type="number"
            min={0}
            max={131072}
            step={1024}
            value={perfForm.ollamaNumCtx || ''}
            placeholder="4096"
            onChange={(e) =>
              setPerfForm((p) => ({
                ...p,
                ollamaNumCtx: Math.max(0, Number(e.target.value) || 0),
              }))
            }
            className="mt-1 w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded"
          />
        </label>
        <label className="block text-sm text-slack-text">
          Ollama num_predict (0 = auto)
          <input
            type="number"
            min={0}
            max={8192}
            value={perfForm.ollamaNumPredict || ''}
            placeholder="512"
            onChange={(e) =>
              setPerfForm((p) => ({
                ...p,
                ollamaNumPredict: Math.max(0, Number(e.target.value) || 0),
              }))
            }
            className="mt-1 w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded"
          />
        </label>
        <label className="block text-sm text-slack-text sm:col-span-2">
          Ollama keep_alive
          <input
            type="text"
            value={perfForm.ollamaKeepAlive}
            placeholder="5m (empty = Ollama default, 0 or -1 = unload immediately)"
            onChange={(e) => setPerfForm((p) => ({ ...p, ollamaKeepAlive: e.target.value }))}
            className="mt-1 w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded font-mono text-sm"
          />
        </label>
      </div>
      <button
        type="button"
        disabled={perfSaving}
        onClick={() => void savePerformanceSettings()}
        className="mt-4 w-full px-4 py-2 bg-slack-accent text-white rounded hover:bg-slack-accentHover disabled:opacity-50"
      >
        {perfSaving ? 'Saving…' : 'Save performance settings'}
      </button>
    </div>
    </div>
  );
}
