import { useState, useEffect } from 'react';
import {
  agentsForTrack,
  modelsToEnsureForTrack,
  ollamaModelForTrack,
  packsEnabledForTrack,
  type WizardTrack,
} from '../config/wizardProfiles';

interface ProviderChoice {
  id: string;
  type: string;
  name: string;
  endpoint?: string;
  apiKey?: string;
  model?: string;
}

interface AgentChoice {
  type: string;
  name: string;
  enabled: boolean;
}

interface SetupWizardProps {
  onComplete: () => void;
  serverAddr: string;
}

const BIO_HF_REPO = 'aaditya/Llama3-OpenBioLLM-8B';

export function SetupWizard({ onComplete, serverAddr }: SetupWizardProps) {
  const [step, setStep] = useState(0);
  const [wizardTrack, setWizardTrack] = useState<WizardTrack>('developer');
  const [providerType, setProviderType] = useState<'ollama' | 'cloud'>('ollama');
  const [ollamaStatus, setOllamaStatus] = useState<{ installed: boolean; running: boolean } | null>(null);
  const [apiKey, setApiKey] = useState('');
  const [hfToken, setHfToken] = useState('');
  const [agents, setAgents] = useState<AgentChoice[]>(() => agentsForTrack('developer'));
  const [pulling, setPulling] = useState(false);
  const [pullStatus, setPullStatus] = useState('');
  const [saving, setSaving] = useState(false);
  const [cliInstalled, setCliInstalled] = useState<Record<string, boolean>>({});
  const [cliTypes, setCliTypes] = useState<string[]>([]);

  const defaultOllamaModel = ollamaModelForTrack(wizardTrack);

  useEffect(() => {
    if (step === 3 && providerType === 'ollama') {
      checkOllama();
    }
  }, [step, providerType]);

  useEffect(() => {
    if (step !== 5) return;
    void (async () => {
      try {
        const resp = await fetch(`${serverAddr}/api/cli-agent-types`);
        const data = await resp.json();
        setCliTypes((data.types as string[]) ?? []);
        setCliInstalled((data.installed as Record<string, boolean>) ?? {});
      } catch {
        setCliTypes([]);
        setCliInstalled({});
      }
    })();
  }, [step, serverAddr]);

  async function checkOllama() {
    try {
      const resp = await fetch(`${serverAddr}/api/ollama/install-status`);
      const data = await resp.json();
      setOllamaStatus(data);
    } catch {
      setOllamaStatus({ installed: false, running: false });
    }
  }

  async function startOllama() {
    try {
      await fetch(`${serverAddr}/api/ollama/start`, { method: 'POST' });
      await checkOllama();
    } catch (e) {
      console.error('Failed to start Ollama:', e);
    }
  }

  async function pullDefaultModel() {
    setPulling(true);
    setPullStatus(`Pulling ${defaultOllamaModel}...`);
    try {
      const resp = await fetch(`${serverAddr}/api/ollama/pull`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ model: defaultOllamaModel }),
      });
      const reader = resp.body?.getReader();
      const decoder = new TextDecoder();
      if (reader) {
        while (true) {
          const { done, value } = await reader.read();
          if (done) break;
          const text = decoder.decode(value);
          const lines = text.split('\n').filter(l => l.startsWith('data: '));
          for (const line of lines) {
            try {
              const data = JSON.parse(line.replace('data: ', ''));
              if (data.percent) {
                setPullStatus(`Pulling... ${data.percent.toFixed(1)}%`);
              } else if (data.status === 'success') {
                setPullStatus('Model ready!');
              } else if (data.error) {
                setPullStatus(
                  wizardTrack === 'lifeSciences'
                    ? `Pull may require Model Library: download OpenBioLLM GGUF and import as ${defaultOllamaModel}`
                    : `Pull failed: ${data.error}`,
                );
              }
            } catch { /* ignore parse errors */ }
          }
        }
      }
    } catch (e) {
      setPullStatus(`Pull failed: ${e}`);
    }
    setPulling(false);
  }

  function selectTrack(track: WizardTrack) {
    setWizardTrack(track);
    setAgents(agentsForTrack(track));
    setStep(2);
  }

  async function saveAndFinish() {
    setSaving(true);
    const providers: ProviderChoice[] = [];

    if (providerType === 'ollama') {
      providers.push({
        id: 'ollama-local',
        type: 'ollama',
        name:
          wizardTrack === 'lifeSciences'
            ? 'Local Ollama (Bio 8B)'
            : wizardTrack === 'general'
              ? 'Local Ollama (utility)'
              : 'Local Ollama (Coder)',
        endpoint: 'http://localhost:11434',
        model: defaultOllamaModel,
      });
    } else if (wizardTrack === 'lifeSciences' && hfToken.trim()) {
      providers.push({
        id: 'hf-bio',
        type: 'huggingface',
        name: 'Hugging Face (OpenBioLLM)',
        apiKey: hfToken.trim(),
        model: BIO_HF_REPO,
      });
    } else {
      providers.push({
        id: 'anthropic',
        type: 'anthropic',
        name: 'Claude (Anthropic)',
        apiKey: apiKey,
        model: 'claude-3-5-sonnet-20241022',
      });
    }

    const config: Record<string, unknown> = {
      server: { host: 'localhost', port: 18765 },
      ai: {
        default_provider_id: providers[0].id,
        providers,
      },
      agents: agents.map(a => ({ type: a.type, name: a.name, enabled: a.enabled })),
      packs: {
        enabled: packsEnabledForTrack(wizardTrack),
      },
      mcp: { enabled: true },
      ollama: {
        auto_start: providerType === 'ollama',
        models_to_ensure: modelsToEnsureForTrack(wizardTrack, providerType),
      },
      updates: { auto_check: true },
    };
    if (hfToken.trim()) {
      config.hf = { token: hfToken.trim() };
    }

    try {
      await fetch(`${serverAddr}/api/settings`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(config),
      });
      await fetch(`${serverAddr}/api/agents/restart`, { method: 'POST' });
    } catch (e) {
      console.error('Failed to save config:', e);
    }
    setSaving(false);
    onComplete();
  }

  const toggleAgent = (type: string) => {
    setAgents(prev => prev.map(a => a.type === type ? { ...a, enabled: !a.enabled } : a));
  };

  const steps = ['Welcome', 'Focus', 'Provider', 'Setup', 'Agents', 'Done'];

  return (
    <div className="flex items-center justify-center w-full h-screen bg-gray-950">
      <div className="w-full max-w-lg p-8 space-y-6">
        <div className="flex gap-2 justify-center">
          {steps.map((s, i) => (
            <div key={s} className={`h-1 w-10 rounded ${i <= step ? 'bg-blue-500' : 'bg-gray-700'}`} />
          ))}
        </div>

        {step === 0 && (
          <div className="text-center space-y-4">
            <h1 className="text-3xl font-bold text-white">Welcome to Neural Junkie</h1>
            <p className="text-gray-400">
              Let's set up your multi-agent AI collaboration environment.
            </p>
            <button onClick={() => setStep(1)} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-500 transition-colors">
              Get Started
            </button>
          </div>
        )}

        {step === 1 && (
          <div className="space-y-4">
            <h2 className="text-xl font-semibold text-white text-center">What are you here for?</h2>
            <div className="grid grid-cols-1 gap-4">
              <button
                onClick={() => selectTrack('developer')}
                className="p-4 rounded-lg border border-gray-700 hover:border-blue-500 text-left space-y-2 transition-colors"
              >
                <div className="font-medium text-white">Software development</div>
                <div className="text-xs text-gray-400">
                  Coding specialists, repo context, and Qwen Coder models.
                </div>
              </button>
              <button
                onClick={() => selectTrack('lifeSciences')}
                className="p-4 rounded-lg border border-gray-700 hover:border-teal-500 text-left space-y-2 transition-colors"
              >
                <div className="font-medium text-white">Life sciences &amp; lab work</div>
                <div className="text-xs text-gray-400">
                  Neural Junkie Bio 8B, BiologyExpert, sequence tools, and structure prediction.
                </div>
              </button>
              <button
                onClick={() => selectTrack('general')}
                className="p-4 rounded-lg border border-gray-700 hover:border-violet-500 text-left space-y-2 transition-colors"
              >
                <div className="font-medium text-white">Team chat &amp; productivity</div>
                <div className="text-xs text-gray-400">
                  ChatModerator, Assistant, and auto-detected CLI tools (Cursor, Gemini, Claude, Copilot) when installed.
                  Enable the Software development pack later for in-process coding specialists.
                </div>
              </button>
            </div>
          </div>
        )}

        {step === 2 && (
          <div className="space-y-4">
            <h2 className="text-xl font-semibold text-white text-center">Choose your AI backend</h2>
            <div className="grid grid-cols-2 gap-4">
              <button
                onClick={() => { setProviderType('ollama'); setStep(3); }}
                className={`p-4 rounded-lg border text-left space-y-2 transition-colors ${
                  providerType === 'ollama' ? 'border-blue-500 bg-blue-500/10' : 'border-gray-700 hover:border-gray-500'
                }`}
              >
                <div className="font-medium text-white">Local Models</div>
                <div className="text-xs text-gray-400">
                  {wizardTrack === 'lifeSciences'
                    ? 'Run Bio 8B locally with Ollama. Private, research use.'
                    : 'Run AI locally with Ollama. Free, private, no API key needed.'}
                </div>
              </button>
              <button
                onClick={() => { setProviderType('cloud'); setStep(3); }}
                className={`p-4 rounded-lg border text-left space-y-2 transition-colors ${
                  providerType === 'cloud' ? 'border-blue-500 bg-blue-500/10' : 'border-gray-700 hover:border-gray-500'
                }`}
              >
                <div className="font-medium text-white">Cloud API</div>
                <div className="text-xs text-gray-400">
                  {wizardTrack === 'lifeSciences'
                    ? 'Hugging Face hosted OpenBioLLM or Anthropic Claude (API key).'
                    : 'Use Anthropic Claude. Requires an API key.'}
                </div>
              </button>
            </div>
          </div>
        )}

        {step === 3 && providerType === 'ollama' && (
          <div className="space-y-4">
            <h2 className="text-xl font-semibold text-white text-center">Ollama Setup</h2>
            {ollamaStatus === null ? (
              <div className="text-center text-gray-400">Checking Ollama installation...</div>
            ) : !ollamaStatus.installed ? (
              <div className="space-y-3">
                <div className="text-yellow-400 text-sm">Ollama is not installed.</div>
                <p className="text-gray-400 text-xs">
                  Install Ollama from <a href="https://ollama.com" className="text-blue-400 underline">ollama.com</a>, then click "Check Again".
                </p>
                <button onClick={checkOllama} className="px-4 py-2 bg-gray-700 text-white rounded hover:bg-gray-600">
                  Check Again
                </button>
              </div>
            ) : !ollamaStatus.running ? (
              <div className="space-y-3">
                <div className="text-green-400 text-sm">Ollama is installed but not running.</div>
                <button onClick={startOllama} className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-500">
                  Start Ollama
                </button>
              </div>
            ) : (
              <div className="space-y-3">
                <div className="text-green-400 text-sm">Ollama is installed and running.</div>
                {wizardTrack === 'lifeSciences' && (
                  <p className="text-xs text-gray-500">
                    If pull fails, open Model Library (⇧⌘M), download OpenBioLLM GGUF, and import as <code className="text-teal-400">{defaultOllamaModel}</code>.
                  </p>
                )}
                {!pullStatus.includes('ready') && (
                  <button onClick={pullDefaultModel} disabled={pulling} className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-500 disabled:opacity-50">
                    {pulling
                      ? pullStatus
                      : `Pull ${
                          wizardTrack === 'lifeSciences'
                            ? 'Neural Junkie Bio 8B'
                            : wizardTrack === 'general'
                              ? 'utility model'
                              : 'Coder model'
                        } (${defaultOllamaModel})`}
                  </button>
                )}
                {pullStatus && <div className="text-xs text-gray-400">{pullStatus}</div>}
                <button onClick={() => setStep(4)} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-500">
                  Next
                </button>
              </div>
            )}
          </div>
        )}

        {step === 3 && providerType === 'cloud' && (
          <div className="space-y-4">
            <h2 className="text-xl font-semibold text-white text-center">
              {wizardTrack === 'lifeSciences' ? 'Cloud API Keys' : 'Anthropic API Key'}
            </h2>
            {wizardTrack === 'lifeSciences' && (
              <>
                <label className="block text-xs text-gray-400">Hugging Face token (recommended for Bio 8B hosted)</label>
                <input
                  type="password"
                  value={hfToken}
                  onChange={(e) => setHfToken(e.target.value)}
                  placeholder="hf_..."
                  className="w-full px-4 py-2 bg-gray-800 border border-gray-700 rounded-lg text-white focus:border-teal-500 focus:outline-none"
                />
                <p className="text-xs text-gray-500">Also used for ESMFold structure prediction (saved in Settings).</p>
              </>
            )}
            <label className="block text-xs text-gray-400">
              {wizardTrack === 'lifeSciences' && hfToken.trim() ? 'Anthropic key (optional fallback)' : 'Anthropic API key'}
            </label>
            <input
              type="password"
              value={apiKey}
              onChange={(e) => setApiKey(e.target.value)}
              placeholder="sk-ant-..."
              className="w-full px-4 py-2 bg-gray-800 border border-gray-700 rounded-lg text-white focus:border-blue-500 focus:outline-none"
            />
            <button
              onClick={() => setStep(4)}
              disabled={wizardTrack === 'lifeSciences' ? !hfToken.trim() && !apiKey.trim() : !apiKey}
              className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-500 disabled:opacity-50"
            >
              Next
            </button>
          </div>
        )}

        {step === 4 && (
          <div className="space-y-4">
            <h2 className="text-xl font-semibold text-white text-center">Configure Agents</h2>
            {wizardTrack === 'general' ? (
              <div className="space-y-3 text-sm text-gray-400">
                <p>
                  <strong className="text-white">ChatModerator</strong> is always on with the hub (commands and chat help).
                </p>
                <p>
                  <strong className="text-white">CLI agents</strong> (Cursor, Gemini, Claude, Copilot) auto-join #general when their binaries are on your PATH.
                </p>
                <p>Toggle Assistant below. Coding specialists (GoExpert, RustExpert, …) are available later via Settings → Domain packs → Software development.</p>
              </div>
            ) : wizardTrack === 'developer' ? (
              <p className="text-sm text-gray-400 text-center">
                Assistant is configured here. Six engineering specialists are added via the Software development pack (enabled for this track).
              </p>
            ) : (
              <p className="text-sm text-gray-400 text-center">Choose which specialist agents to enable.</p>
            )}
            <div className="space-y-2">
              {agents.map(a => (
                <label key={a.type} className="flex items-center justify-between p-3 bg-gray-800 rounded-lg cursor-pointer hover:bg-gray-750">
                  <div>
                    <div className="text-white text-sm font-medium">{a.name}</div>
                    <div className="text-xs text-gray-500 capitalize">{a.type} specialist</div>
                  </div>
                  <input
                    type="checkbox"
                    checked={a.enabled}
                    onChange={() => toggleAgent(a.type)}
                    className="w-4 h-4 accent-blue-500"
                  />
                </label>
              ))}
            </div>
            <button onClick={() => setStep(5)} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-500">
              Next
            </button>
          </div>
        )}

        {step === 5 && (
          <div className="text-center space-y-4">
            <h2 className="text-2xl font-bold text-white">All Set!</h2>
            <p className="text-gray-400 text-sm">
              {providerType === 'ollama'
                ? wizardTrack === 'lifeSciences'
                  ? 'BiologyExpert will use Neural Junkie Bio 8B locally.'
                  : 'Your agents will use local Ollama models.'
                : wizardTrack === 'lifeSciences' && hfToken.trim()
                  ? 'BiologyExpert will use hosted OpenBioLLM via Hugging Face.'
                  : 'Your agents will use the Anthropic Claude API.'}
            </p>
            <p className="text-gray-500 text-xs">
              {agents.filter(a => a.enabled).length} configured agent(s) in settings.
              {wizardTrack === 'lifeSciences' && ' Research use only — not for clinical diagnosis.'}
            </p>
            {cliTypes.length > 0 && (
              <div className="text-left text-xs text-gray-500 space-y-1 max-w-sm mx-auto">
                <div className="text-gray-400 font-medium">CLI tools on PATH</div>
                {cliTypes.map((t) => (
                  <div key={t} className="flex justify-between gap-2">
                    <span className="capitalize">{t}</span>
                    <span className={cliInstalled[t] ? 'text-green-400' : 'text-gray-600'}>
                      {cliInstalled[t] ? 'detected' : 'not installed'}
                    </span>
                  </div>
                ))}
                <p className="text-gray-600 pt-1">After launch, detected CLIs join #general automatically.</p>
              </div>
            )}
            <button
              onClick={saveAndFinish}
              disabled={saving}
              className="px-8 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-500 disabled:opacity-50 font-medium"
            >
              {saving ? 'Saving...' : 'Launch Neural Junkie'}
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
