import { useState, useEffect, useMemo } from 'react';
import { useChatStore } from '../stores/chatStore';
import {
  CLI_PROVIDER_PROFILES,
  detectCLIProfileLabel,
  type CLIProviderProfileConfig,
} from '../utils/cliProviderProfiles';

interface ProviderConfig {
  id: string;
  type: string;
  name: string;
  endpoint?: string;
  api_key?: string;
  model?: string;
  headers?: Record<string, string>;
  work_dir?: string;
}

interface ProviderManagerProps {
  serverAddr: string;
}

const PROVIDER_TYPES = [
  { value: 'ollama', label: 'Ollama (Local)' },
  { value: 'huggingface', label: 'Hugging Face (Hosted Inference)' },
  { value: 'anthropic', label: 'Anthropic (Claude API)' },
  { value: 'openai-compatible', label: 'OpenAI-Compatible (Bedrock, Azure, Groq, etc.)' },
  { value: 'cursor-cli', label: 'Cursor CLI' },
  { value: 'gemini-cli', label: 'Gemini CLI' },
];

export function ProviderManager({ serverAddr }: ProviderManagerProps) {
  const [providers, setProviders] = useState<ProviderConfig[]>([]);
  const [editing, setEditing] = useState<ProviderConfig | null>(null);
  const [isNew, setIsNew] = useState(false);
  const [testResult, setTestResult] = useState<Record<string, { success: boolean; error?: string }>>({});
  const [profileStatus, setProfileStatus] = useState<Record<string, string>>({});
  const messages = useChatStore((state) => state.messages);
  const agents = useChatStore((state) => state.agents);

  useEffect(() => { loadProviders(); }, []);

  async function loadProviders() {
    try {
      const resp = await fetch(`${serverAddr}/api/providers`);
      const data = await resp.json();
      setProviders(data);
    } catch (e) {
      console.error('Failed to load providers:', e);
    }
  }

  async function saveProvider(p: ProviderConfig) {
    const method = isNew ? 'POST' : 'PUT';
    const url = isNew ? `${serverAddr}/api/providers` : `${serverAddr}/api/providers/${p.id}`;
    await fetch(url, {
      method,
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(p),
    });
    setEditing(null);
    setIsNew(false);
    loadProviders();
  }

  async function applyCLIProfile(config: CLIProviderProfileConfig, presetId: string) {
    const preset = config.presets.find((p) => p.id === presetId);
    if (!preset) return;

    const existing = providers.find((p) => p.type === config.providerType);
    const defaultName = config.title + ' (Auto-detected)';
    const payload: ProviderConfig = existing
      ? { ...existing, model: preset.model }
      : {
          id: config.providerType,
          type: config.providerType,
          name: defaultName,
          model: preset.model,
        };

    const method = existing ? 'PUT' : 'POST';
    const url = existing
      ? `${serverAddr}/api/providers/${payload.id}`
      : `${serverAddr}/api/providers`;

    setProfileStatus((prev) => ({ ...prev, [config.providerType]: 'Applying...' }));
    try {
      const resp = await fetch(url, {
        method,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });
      if (!resp.ok) {
        const err = await resp.text();
        throw new Error(err || 'Failed to apply profile');
      }
      await loadProviders();
      setProfileStatus((prev) => ({
        ...prev,
        [config.providerType]: `Set to ${preset.label} (${preset.model}).`,
      }));
    } catch (e) {
      setProfileStatus((prev) => ({
        ...prev,
        [config.providerType]: `Failed: ${e instanceof Error ? e.message : String(e)}`,
      }));
    }
  }

  async function deleteProvider(id: string) {
    if (!confirm(`Remove provider "${id}"?`)) return;
    const resp = await fetch(`${serverAddr}/api/providers/${id}`, { method: 'DELETE' });
    if (!resp.ok) {
      const err = await resp.text();
      alert(err);
      return;
    }
    loadProviders();
  }

  async function testProvider(id: string) {
    setTestResult(prev => ({ ...prev, [id]: { success: false } }));
    try {
      const resp = await fetch(`${serverAddr}/api/providers/${id}/test`, { method: 'POST' });
      const data = await resp.json();
      setTestResult(prev => ({ ...prev, [id]: data }));
    } catch (e) {
      setTestResult(prev => ({ ...prev, [id]: { success: false, error: String(e) } }));
    }
  }

  function startAdd() {
    setEditing({ id: '', type: 'openai-compatible', name: '' });
    setIsNew(true);
  }

  const visibleCLIProfiles = CLI_PROVIDER_PROFILES;

  const usageInsights = useMemo(() => {
    const providerByAgentId = new Map<string, string>();
    for (const agent of agents) {
      providerByAgentId.set(agent.id, (agent.ai_provider || '').toLowerCase());
    }

    const providerMessageCount = new Map<string, number>();
    let localMessages = 0;
    let cloudMessages = 0;
    let totalAgentMessages = 0;

    for (const msg of messages) {
      if (!msg.from || msg.from.type === 'human') continue;
      if (msg.type !== 'chat' && msg.type !== 'answer') continue;

      const provider = ((msg.from.ai_provider || providerByAgentId.get(msg.from.id) || 'unknown') as string).toLowerCase();
      providerMessageCount.set(provider, (providerMessageCount.get(provider) || 0) + 1);
      totalAgentMessages++;

      const isLocal = provider === 'ollama' || provider === 'lmstudio' || provider.endsWith('-cli');
      if (isLocal) {
        localMessages++;
      } else {
        cloudMessages++;
      }
    }

    const rankedProviders = Array.from(providerMessageCount.entries())
      .sort((a, b) => b[1] - a[1])
      .slice(0, 6);

    return { totalAgentMessages, localMessages, cloudMessages, rankedProviders };
  }, [agents, messages]);

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold text-gray-300">AI Providers</h3>
        <button onClick={startAdd} className="px-3 py-1 text-xs bg-blue-600 text-white rounded hover:bg-blue-500">
          Add Provider
        </button>
      </div>

      <div className="p-3 bg-gray-800/60 border border-gray-700 rounded-lg">
        <div className="text-sm text-white font-medium">Why two Claudes?</div>
        <p className="mt-1 text-xs text-gray-400 leading-relaxed">
          <span className="text-gray-300">Anthropic (Claude API)</span> is the in-process cloud provider for
          built-in specialists (moderator, repo agents, etc.) — it calls Anthropic&apos;s HTTP API with your API key.
          <span className="text-gray-300"> Claude Code CLI</span> is a separate auto-detected subprocess agent when
          the <code className="text-gray-300">claude</code> binary is on your PATH. They can both appear in this list
          because they are different integration paths, not duplicates of the same thing.
        </p>
      </div>

      {visibleCLIProfiles.length > 0 && (
        <div className="space-y-2">
          <div className="text-xs text-gray-400 uppercase tracking-wide">CLI model profiles</div>
          {visibleCLIProfiles.map((config) => {
            const provider = providers.find((p) => p.type === config.providerType);
            const status = profileStatus[config.providerType];
            return (
              <div
                key={config.providerType}
                className="p-3 bg-gray-800/60 border border-gray-700 rounded-lg"
              >
                <div className="flex items-center justify-between gap-3 flex-wrap">
                  <div>
                    <div className="text-sm text-white font-medium">{config.title}</div>
                    <div className="text-xs text-gray-400">{config.description}</div>
                  </div>
                  <div className="flex items-center gap-2 flex-wrap">
                    {config.presets.map((preset) => (
                      <button
                        key={preset.id}
                        onClick={() => applyCLIProfile(config, preset.id)}
                        className={`px-3 py-1 text-xs text-white rounded ${preset.buttonClass}`}
                      >
                        {preset.label}
                      </button>
                    ))}
                  </div>
                </div>
                {(provider || status) && (
                  <div className="mt-2 text-xs text-gray-300">
                    {provider
                      ? `Current: ${provider.model || config.presets[0]?.model} (${detectCLIProfileLabel(config, provider.model)})`
                      : 'Provider will be created on first apply.'}
                    {status ? ` ${status}` : ''}
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}

      <div className="p-3 bg-gray-800/60 border border-gray-700 rounded-lg">
        <div className="text-sm text-white font-medium">Usage & Cost Insights</div>
        <div className="text-xs text-gray-400 mt-1">
          Quick estimate by provider activity. Cloud providers typically bill by tokens; local providers generally do not.
        </div>
        <div className="mt-3 grid grid-cols-1 sm:grid-cols-3 gap-2">
          <div className="rounded border border-gray-700 bg-gray-900/60 p-2">
            <div className="text-[11px] uppercase tracking-wide text-gray-400">Agent messages</div>
            <div className="text-lg font-semibold text-white">{usageInsights.totalAgentMessages}</div>
          </div>
          <div className="rounded border border-gray-700 bg-gray-900/60 p-2">
            <div className="text-[11px] uppercase tracking-wide text-gray-400">Local activity</div>
            <div className="text-lg font-semibold text-emerald-400">{usageInsights.localMessages}</div>
          </div>
          <div className="rounded border border-gray-700 bg-gray-900/60 p-2">
            <div className="text-[11px] uppercase tracking-wide text-gray-400">Cloud activity</div>
            <div className="text-lg font-semibold text-amber-300">{usageInsights.cloudMessages}</div>
          </div>
        </div>
        <div className="mt-3">
          <div className="text-xs text-gray-300 mb-1">Top providers by activity</div>
          {usageInsights.rankedProviders.length === 0 ? (
            <div className="text-xs text-gray-500">No agent activity yet.</div>
          ) : (
            <div className="space-y-1">
              {usageInsights.rankedProviders.map(([provider, count]) => (
                <div key={provider} className="flex items-center justify-between text-xs">
                  <span className="text-gray-300">{provider || 'unknown'}</span>
                  <span className="text-white font-medium">{count}</span>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      <div className="space-y-2">
        {providers.map(p => (
          <div key={p.id} className="flex items-center justify-between p-3 bg-gray-800 rounded-lg">
            <div>
              <div className="text-sm text-white font-medium">{p.name || p.id}</div>
              <div className="text-xs text-gray-500">{p.type} {p.model ? `· ${p.model}` : ''}</div>
            </div>
            <div className="flex items-center gap-2">
              {testResult[p.id] && (
                <span className={`text-xs ${testResult[p.id].success ? 'text-green-400' : 'text-red-400'}`}>
                  {testResult[p.id].success ? 'Connected' : testResult[p.id].error?.slice(0, 30) || 'Failed'}
                </span>
              )}
              <button onClick={() => testProvider(p.id)} className="px-2 py-1 text-xs bg-gray-700 text-gray-300 rounded hover:bg-gray-600">
                Test
              </button>
              <button onClick={() => { setEditing(p); setIsNew(false); }} className="px-2 py-1 text-xs bg-gray-700 text-gray-300 rounded hover:bg-gray-600">
                Edit
              </button>
              <button onClick={() => deleteProvider(p.id)} className="px-2 py-1 text-xs bg-red-700/50 text-red-300 rounded hover:bg-red-700">
                Remove
              </button>
            </div>
          </div>
        ))}
      </div>

      {editing && (
        <div className="p-4 bg-gray-800/50 border border-gray-700 rounded-lg space-y-3">
          <div className="text-sm font-medium text-white">{isNew ? 'Add Provider' : 'Edit Provider'}</div>
          <div className="grid grid-cols-2 gap-3">
            <input
              value={editing.id}
              onChange={(e) => setEditing({ ...editing, id: e.target.value })}
              placeholder="ID (e.g. my-claude)"
              disabled={!isNew}
              className="px-3 py-2 bg-gray-900 border border-gray-700 rounded text-sm text-white disabled:opacity-50"
            />
            <select
              value={editing.type}
              onChange={(e) => setEditing({ ...editing, type: e.target.value })}
              className="px-3 py-2 bg-gray-900 border border-gray-700 rounded text-sm text-white"
            >
              {PROVIDER_TYPES.map(t => (
                <option key={t.value} value={t.value}>{t.label}</option>
              ))}
            </select>
            <input
              value={editing.name}
              onChange={(e) => setEditing({ ...editing, name: e.target.value })}
              placeholder="Display name"
              className="px-3 py-2 bg-gray-900 border border-gray-700 rounded text-sm text-white"
            />
            <input
              value={editing.model || ''}
              onChange={(e) => setEditing({ ...editing, model: e.target.value })}
              placeholder={editing.type === 'huggingface' ? 'Hub repo id (org/model)' : 'Model'}
              className="px-3 py-2 bg-gray-900 border border-gray-700 rounded text-sm text-white"
            />
            {(editing.type === 'ollama' || editing.type === 'openai-compatible' || editing.type === 'huggingface') && (
              <input
                value={editing.endpoint || ''}
                onChange={(e) => setEditing({ ...editing, endpoint: e.target.value })}
                placeholder={editing.type === 'huggingface' ? 'https://router.huggingface.co/v1' : 'Endpoint URL'}
                className="col-span-2 px-3 py-2 bg-gray-900 border border-gray-700 rounded text-sm text-white"
              />
            )}
            {(editing.type === 'anthropic' || editing.type === 'openai-compatible' || editing.type === 'huggingface') && (
              <input
                value={editing.api_key || ''}
                onChange={(e) => setEditing({ ...editing, api_key: e.target.value })}
                placeholder={editing.type === 'huggingface' ? 'HF token (or hub token in Settings)' : 'API Key'}
                type="password"
                className="col-span-2 px-3 py-2 bg-gray-900 border border-gray-700 rounded text-sm text-white"
              />
            )}
          </div>
          <div className="flex gap-2">
            <button onClick={() => saveProvider(editing)} className="px-4 py-2 bg-blue-600 text-white text-sm rounded hover:bg-blue-500">
              {isNew ? 'Add' : 'Save'}
            </button>
            <button onClick={() => { setEditing(null); setIsNew(false); }} className="px-4 py-2 bg-gray-700 text-gray-300 text-sm rounded hover:bg-gray-600">
              Cancel
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
