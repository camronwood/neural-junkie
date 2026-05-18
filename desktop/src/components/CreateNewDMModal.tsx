import { useCallback, useEffect, useState } from 'react';
import type { Channel } from '../types/protocol';
import { ChatAPI } from '../api/chatAPI';

const PRESET_EXPERT_TYPES: { value: string; label: string }[] = [
  { value: 'assistant', label: 'Assistant' },
  { value: 'rust', label: 'Rust' },
  { value: 'backend', label: 'Backend' },
  { value: 'frontend', label: 'Frontend' },
  { value: 'devops', label: 'DevOps' },
  { value: 'database', label: 'Database' },
  { value: 'security', label: 'Security' },
];

const CUSTOM_EXPERT_VALUE = '__custom__';

/** Must stay in sync with `internal/agent/cli_registry.go` ListCLIAgentTypes (sorted). */
const CLI_TYPES_FALLBACK: readonly string[] = ['claude', 'copilot', 'cursor', 'gemini'];

type RuntimeTab = 'expert' | 'cli';

export interface CreateNewDMModalProps {
  api: ChatAPI;
  username: string;
  isOpen: boolean;
  onClose: () => void;
  /** Called after the server created the DM channel (parent refreshes lists / navigates). */
  onCreated: (channel: Channel) => void | Promise<void>;
}

export function CreateNewDMModal({ api, username, isOpen, onClose, onCreated }: CreateNewDMModalProps) {
  const [tab, setTab] = useState<RuntimeTab>('expert');
  const [displayName, setDisplayName] = useState('');
  const [expertType, setExpertType] = useState('assistant');
  const [customDomain, setCustomDomain] = useState('');
  const [customPersona, setCustomPersona] = useState('');
  const [provider, setProvider] = useState<'ollama' | 'lmstudio' | 'claude' | 'huggingface' | 'registry'>('ollama');
  const [providerId, setProviderId] = useState('');
  const [hubProviders, setHubProviders] = useState<{ id: string; type: string; name: string; model?: string }[]>([]);
  const [hfCatalog, setHfCatalog] = useState<{ repo_id: string; title: string }[]>([]);
  const [model, setModel] = useState('');
  const [ollamaModels, setOllamaModels] = useState<string[]>([]);
  const [lmModels, setLmModels] = useState<string[]>([]);
  const [modelsLoading, setModelsLoading] = useState(false);
  const [cliTypes, setCliTypes] = useState<string[]>([]);
  const [cliInstalled, setCliInstalled] = useState<Record<string, boolean>>({});
  const [cliType, setCliType] = useState('');
  const [cliTypesBanner, setCliTypesBanner] = useState<string | null>(null);
  const [workDir, setWorkDir] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  const reset = useCallback(() => {
    setTab('expert');
    setDisplayName('');
    setExpertType('assistant');
    setCustomDomain('');
    setCustomPersona('');
    setProvider('ollama');
    setProviderId('');
    setHubProviders([]);
    setHfCatalog([]);
    setModel('');
    setOllamaModels([]);
    setLmModels([]);
    setCliTypes([]);
    setCliInstalled({});
    setCliType('');
    setCliTypesBanner(null);
    setWorkDir('');
    setFormError(null);
    setSubmitting(false);
  }, []);

  useEffect(() => {
    if (!isOpen) return;
    setFormError(null);
  }, [isOpen, tab]);

  useEffect(() => {
    if (!isOpen || tab !== 'expert') return;
    let cancelled = false;
    (async () => {
      setModelsLoading(true);
      try {
        if (provider === 'registry') {
          const rows = await api.fetchProviders();
          if (!cancelled) setHubProviders(rows);
        } else if (provider === 'huggingface') {
          const cat = await api.fetchHfCatalog();
          if (!cancelled) {
            setHfCatalog(cat.filter((e) => e.modes?.includes('hosted')).map((e) => ({ repo_id: e.repo_id, title: e.title })));
          }
        } else if (provider === 'ollama') {
          const m = await api.fetchOllamaModels();
          if (!cancelled) setOllamaModels(m);
        } else if (provider === 'lmstudio') {
          const m = await api.fetchLMStudioModels();
          if (!cancelled) setLmModels(m);
        }
      } catch {
        if (!cancelled) {
          if (provider === 'ollama') setOllamaModels([]);
          else if (provider === 'lmstudio') setLmModels([]);
          else if (provider === 'huggingface') setHfCatalog([]);
          else setHubProviders([]);
        }
      } finally {
        if (!cancelled) setModelsLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [api, isOpen, tab, provider]);

  useEffect(() => {
    if (!isOpen || tab !== 'cli') return;
    let cancelled = false;
    setCliTypesBanner(null);
    (async () => {
      try {
        const res = await api.fetchCliAgentTypes();
        if (cancelled) return;
        const types = res.types?.length ? res.types : [...CLI_TYPES_FALLBACK];
        const installed = res.installed || {};
        setCliTypes(types);
        setCliInstalled(installed);
        if (!res.types?.length) {
          setCliTypesBanner('Hub returned an empty type list; using built-in CLI types. Binary status unknown.');
        }
        setCliType(prev => {
          if (prev && types.includes(prev)) return prev;
          const firstInstalled = types.find(t => installed[t]);
          return firstInstalled ?? types[0] ?? '';
        });
      } catch (err) {
        if (!cancelled) {
          const msg = err instanceof Error ? err.message : 'Could not load CLI types from the hub.';
          setCliTypesBanner(`${msg} Showing built-in types; install status unknown.`);
          setCliTypes([...CLI_TYPES_FALLBACK]);
          setCliInstalled({});
          setCliType(CLI_TYPES_FALLBACK[0] ?? '');
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [api, isOpen, tab]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const name = displayName.trim();
    if (!name) {
      setFormError('Enter a name for the agent.');
      return;
    }
    if (!username.trim()) {
      setFormError('Username is required (sign in / set display name).');
      return;
    }

    setSubmitting(true);
    setFormError(null);
    try {
      if (tab === 'expert') {
        const isCustom = expertType === CUSTOM_EXPERT_VALUE;
        const domain = isCustom ? customDomain.trim() : expertType;
        if (!domain) {
          setFormError(isCustom ? 'Enter a domain for your custom expert (e.g. guitar).' : 'Select a persona.');
          setSubmitting(false);
          return;
        }
        const payload: Parameters<typeof api.createDMAgent>[0] = {
          created_by: username.trim(),
          mode: 'expert',
          display_name: name,
          expert_type: domain,
          persona: isCustom ? customPersona.trim() : undefined,
          model: model.trim(),
        };
        if (provider === 'registry') {
          if (!providerId) {
            setFormError('Select a configured provider.');
            setSubmitting(false);
            return;
          }
          payload.provider_id = providerId;
        } else {
          payload.provider = provider === 'huggingface' ? 'huggingface' : provider;
        }
        const ch = await api.createDMAgent(payload);
        await onCreated(ch);
        reset();
        onClose();
      } else {
        if (!cliType) {
          setFormError('Select a CLI agent type.');
          setSubmitting(false);
          return;
        }
        const ch = await api.createDMAgent({
          created_by: username.trim(),
          mode: 'cli',
          display_name: name,
          cli_type: cliType,
          work_dir: workDir.trim(),
        });
        await onCreated(ch);
        reset();
        onClose();
      }
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Request failed');
    } finally {
      setSubmitting(false);
    }
  };

  if (!isOpen) return null;

  const modelSuggestions =
    provider === 'ollama' ? ollamaModels : provider === 'lmstudio' ? lmModels : [];

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={onClose}>
      <div
        className="bg-slack-bg border border-slack-border rounded-lg shadow-2xl w-full max-w-md mx-4 max-h-[90vh] overflow-y-auto"
        onClick={e => e.stopPropagation()}
      >
        <form onSubmit={handleSubmit}>
          <div className="px-5 py-4 border-b border-slack-border">
            <h2 className="text-lg font-bold text-slack-text">New direct message</h2>
            <p className="text-xs text-slack-textMuted mt-1">
              Create a fresh agent and open a 1:1 DM. The agent is not added to your current channel.
            </p>
          </div>

          <div className="px-5 py-4 space-y-4">
            <div className="flex rounded border border-slack-border overflow-hidden text-xs">
              <button
                type="button"
                onClick={() => setTab('expert')}
                className={`flex-1 py-2 font-medium transition-colors ${
                  tab === 'expert' ? 'bg-slack-accent text-white' : 'bg-slack-bgHover text-slack-textMuted hover:text-slack-text'
                }`}
              >
                Local / Claude API
              </button>
              <button
                type="button"
                onClick={() => setTab('cli')}
                className={`flex-1 py-2 font-medium transition-colors ${
                  tab === 'cli' ? 'bg-slack-accent text-white' : 'bg-slack-bgHover text-slack-textMuted hover:text-slack-text'
                }`}
              >
                CLI agent
              </button>
            </div>

            <div>
              <label className="block text-xs font-medium text-slack-textMuted mb-1">Agent name</label>
              <input
                type="text"
                value={displayName}
                onChange={e => setDisplayName(e.target.value)}
                placeholder="e.g. CodeReviewBuddy"
                className="w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-sm text-slack-text placeholder-slack-textMuted focus:outline-none focus:ring-1 focus:ring-slack-accent"
                autoFocus
                required
              />
            </div>

            {tab === 'expert' && (
              <>
                <div>
                  <label className="block text-xs font-medium text-slack-textMuted mb-1">Persona</label>
                  <select
                    value={expertType}
                    onChange={e => setExpertType(e.target.value)}
                    className="w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-sm text-slack-text focus:outline-none focus:ring-1 focus:ring-slack-accent"
                  >
                    {PRESET_EXPERT_TYPES.map(o => (
                      <option key={o.value} value={o.value}>
                        {o.label}
                      </option>
                    ))}
                    <option value={CUSTOM_EXPERT_VALUE}>Custom…</option>
                  </select>
                </div>
                {expertType === CUSTOM_EXPERT_VALUE && (
                  <>
                    <div>
                      <label className="block text-xs font-medium text-slack-textMuted mb-1">Domain</label>
                      <input
                        type="text"
                        value={customDomain}
                        onChange={e => setCustomDomain(e.target.value)}
                        placeholder="e.g. guitar, legal-advice, cooking"
                        className="w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-sm text-slack-text placeholder-slack-textMuted focus:outline-none focus:ring-1 focus:ring-slack-accent"
                      />
                      <p className="text-xs text-slack-textMuted mt-1">
                        Any topic — not limited to engineering presets.
                      </p>
                    </div>
                    <div>
                      <label className="block text-xs font-medium text-slack-textMuted mb-1">
                        Extra instructions (optional)
                      </label>
                      <textarea
                        value={customPersona}
                        onChange={e => setCustomPersona(e.target.value)}
                        placeholder="e.g. Focus on jazz chords and practice routines"
                        rows={3}
                        className="w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-sm text-slack-text placeholder-slack-textMuted focus:outline-none focus:ring-1 focus:ring-slack-accent resize-y"
                      />
                    </div>
                  </>
                )}
                <div>
                  <label className="block text-xs font-medium text-slack-textMuted mb-1">Provider</label>
                  <select
                    value={provider}
                    onChange={e => {
                      setProvider(e.target.value as typeof provider);
                      setModel('');
                      setProviderId('');
                    }}
                    className="w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-sm text-slack-text focus:outline-none focus:ring-1 focus:ring-slack-accent"
                  >
                    <option value="ollama">Ollama (local)</option>
                    <option value="lmstudio">LM Studio (local)</option>
                    <option value="huggingface">Hugging Face (hosted)</option>
                    <option value="registry">From hub providers…</option>
                    <option value="claude">Claude API (cloud)</option>
                  </select>
                </div>
                {provider === 'registry' && (
                  <div>
                    <label className="block text-xs font-medium text-slack-textMuted mb-1">
                      Configured provider {modelsLoading ? '(loading…)' : ''}
                    </label>
                    <select
                      value={providerId}
                      onChange={e => {
                        setProviderId(e.target.value);
                        const p = hubProviders.find(x => x.id === e.target.value);
                        if (p?.model) setModel(p.model);
                      }}
                      className="w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-sm text-slack-text"
                    >
                      <option value="">Select…</option>
                      {hubProviders.map(p => (
                        <option key={p.id} value={p.id}>
                          {p.name || p.id} ({p.type}{p.model ? ` · ${p.model}` : ''})
                        </option>
                      ))}
                    </select>
                  </div>
                )}
                {provider === 'huggingface' && (
                  <div>
                    <label className="block text-xs font-medium text-slack-textMuted mb-1">
                      HF model {modelsLoading ? '(loading…)' : ''}
                    </label>
                    <select
                      value={model}
                      onChange={e => setModel(e.target.value)}
                      className="w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-sm text-slack-text"
                    >
                      <option value="">Select catalog model…</option>
                      {hfCatalog.map(e => (
                        <option key={e.repo_id} value={e.repo_id}>
                          {e.title} ({e.repo_id})
                        </option>
                      ))}
                    </select>
                    <p className="text-xs text-slack-textMuted mt-1">Uses HF Inference (cloud). Set HF_TOKEN on the hub.</p>
                  </div>
                )}
                {provider !== 'claude' && provider !== 'huggingface' && provider !== 'registry' && (
                  <div>
                    <label className="block text-xs font-medium text-slack-textMuted mb-1">
                      Model {modelsLoading ? '(loading…)' : ''}
                    </label>
                    <input
                      type="text"
                      list="nj-dm-model-suggestions"
                      value={model}
                      onChange={e => setModel(e.target.value)}
                      placeholder="Leave blank for server default"
                      className="w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-sm text-slack-text placeholder-slack-textMuted focus:outline-none focus:ring-1 focus:ring-slack-accent"
                    />
                    <datalist id="nj-dm-model-suggestions">
                      {modelSuggestions.map(m => (
                        <option key={m} value={m} />
                      ))}
                    </datalist>
                  </div>
                )}
                {provider === 'claude' && (
                  <p className="text-xs text-slack-textMuted">Uses the Claude model configured on the server.</p>
                )}
              </>
            )}

            {tab === 'cli' && (
              <>
                <div>
                  <label className="block text-xs font-medium text-slack-textMuted mb-1">CLI type</label>
                  <select
                    value={cliType}
                    onChange={e => setCliType(e.target.value)}
                    className="w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-sm text-slack-text focus:outline-none focus:ring-1 focus:ring-slack-accent"
                  >
                    {cliTypes.length === 0 ? (
                      <option value="">No types available</option>
                    ) : (
                      cliTypes.map(t => (
                        <option key={t} value={t}>
                          {t}
                          {cliInstalled[t] === false ? ' (binary not found)' : ''}
                        </option>
                      ))
                    )}
                  </select>
                  {cliTypesBanner && (
                    <p className="text-xs text-amber-500/90 mt-1.5 whitespace-pre-wrap">{cliTypesBanner}</p>
                  )}
                </div>
                <div>
                  <label className="block text-xs font-medium text-slack-textMuted mb-1">
                    Work directory (optional)
                  </label>
                  <input
                    type="text"
                    value={workDir}
                    onChange={e => setWorkDir(e.target.value)}
                    placeholder="Defaults to server cwd or provider env"
                    className="w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-sm text-slack-text placeholder-slack-textMuted focus:outline-none focus:ring-1 focus:ring-slack-accent"
                  />
                </div>
              </>
            )}

            {formError && <p className="text-xs text-red-400 whitespace-pre-wrap">{formError}</p>}
          </div>

          <div className="px-5 py-3 border-t border-slack-border flex justify-end gap-2">
            <button
              type="button"
              onClick={() => {
                reset();
                onClose();
              }}
              className="px-4 py-1.5 text-sm text-slack-textMuted hover:text-slack-text rounded transition-colors"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={submitting || !displayName.trim()}
              className="px-4 py-1.5 text-sm bg-slack-accent hover:bg-slack-accentHover text-white rounded font-medium transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
            >
              {submitting ? 'Creating…' : 'Create DM'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
