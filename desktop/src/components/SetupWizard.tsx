import { useState, useEffect } from 'react';
import {
  agentsForTrack,
  modelsToEnsureForTrack,
  ollamaModelForTrack,
  packsEnabledForTrack,
  CAD_OLLAMA_CHAT_MODEL,
  CAD_OLLAMA_CHAT_MODEL_LIGHT,
  DEV_OLLAMA_MODEL,
  type WizardTrack,
} from '../config/wizardProfiles';
import { CLIAgentsManager } from './CLIAgentsManager';
import {
  fetchHardwareSnapshot,
  HARDWARE_DOCS_URL,
  recommendationMessageForTrack,
  recommendedPrimaryForTrack,
  shouldAutoDowngradePrimary,
  type HardwareSnapshot,
} from '../utils/hardwareRecommendations';
import { installOllamaRuntime } from '../utils/ollamaRuntime';

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
  const [ollamaStatus, setOllamaStatus] = useState<{
    installed: boolean;
    running: boolean;
    bundled?: boolean;
    autoInstallSupported?: boolean;
  } | null>(null);
  const [installingOllama, setInstallingOllama] = useState(false);
  const [installStatus, setInstallStatus] = useState('');
  const [apiKey, setApiKey] = useState('');
  const [hfToken, setHfToken] = useState('');
  const [agents, setAgents] = useState<AgentChoice[]>(() => agentsForTrack('developer'));
  const [pulling, setPulling] = useState(false);
  const [pullStatus, setPullStatus] = useState('');
  const [saving, setSaving] = useState(false);
  const [hardware, setHardware] = useState<HardwareSnapshot | null>(null);
  const [useFullSizeModel, setUseFullSizeModel] = useState(false);

  const trackDefaultModel = ollamaModelForTrack(wizardTrack);
  const tierPrimary = recommendedPrimaryForTrack(hardware, wizardTrack, trackDefaultModel);
  const autoDowngrade =
    !useFullSizeModel &&
    (wizardTrack === 'developer' || wizardTrack === 'cad') &&
    shouldAutoDowngradePrimary(hardware?.tier) &&
    tierPrimary !== trackDefaultModel;
  const selectedOllamaModel = autoDowngrade ? tierPrimary : trackDefaultModel;
  const hardwareMessage = recommendationMessageForTrack(hardware, wizardTrack);

  useEffect(() => {
    void fetchHardwareSnapshot(serverAddr).then(setHardware);
  }, [serverAddr]);

  useEffect(() => {
    if (step === 3 && providerType === 'ollama') {
      checkOllama();
    }
  }, [step, providerType]);

  async function checkOllama() {
    try {
      const resp = await fetch(`${serverAddr}/api/ollama/install-status`);
      const data = await resp.json();
      setOllamaStatus({
        installed: Boolean(data.installed),
        running: Boolean(data.running),
        bundled: data.bundled,
        autoInstallSupported: data.auto_install_supported,
      });
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

  async function installOllama() {
    setInstallingOllama(true);
    setInstallStatus('Preparing Ollama install...');
    try {
      await installOllamaRuntime(serverAddr, (msg) => setInstallStatus(msg));
      await checkOllama();
      const resp = await fetch(`${serverAddr}/api/ollama/install-status`);
      const data = await resp.json();
      if (data.installed && !data.running) {
        setInstallStatus('Starting Ollama...');
        await startOllama();
      }
    } catch (e) {
      setInstallStatus(`Install failed: ${e instanceof Error ? e.message : String(e)}`);
    } finally {
      setInstallingOllama(false);
    }
  }

  async function pullDefaultModel() {
    setPulling(true);
    setPullStatus(`Pulling ${selectedOllamaModel}...`);
    try {
      const resp = await fetch(`${serverAddr}/api/ollama/pull`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ model: selectedOllamaModel }),
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
                    ? `Pull may require Model Library: download OpenBioLLM GGUF and import as ${trackDefaultModel}`
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
    setUseFullSizeModel(false);
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
            : wizardTrack === 'cad'
              ? 'Local Ollama (CAD)'
              : wizardTrack === 'general'
                ? 'Local Ollama (utility)'
                : 'Local Ollama (Coder)',
        endpoint: 'http://localhost:11434',
        model: selectedOllamaModel,
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
        models_to_ensure: providerType === 'ollama'
          ? modelsToEnsureForTrack(wizardTrack, providerType).map((m) =>
              m === trackDefaultModel ? selectedOllamaModel : m,
            )
          : [],
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
      <div className="w-full max-w-xl p-8 space-y-6">
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
                onClick={() => selectTrack('cad')}
                className="p-4 rounded-lg border border-gray-700 hover:border-indigo-500 text-left space-y-2 transition-colors"
              >
                <div className="font-medium text-white">CAD &amp; mechanical design</div>
                <div className="text-xs text-gray-400">
                  CADExpert, OpenSCAD workbench, and Qwen Coder models. Requires OpenSCAD installed.
                </div>
              </button>
              <button
                onClick={() => selectTrack('general')}
                className="p-4 rounded-lg border border-gray-700 hover:border-violet-500 text-left space-y-2 transition-colors"
              >
                <div className="font-medium text-white">Team chat &amp; productivity</div>
                <div className="text-xs text-gray-400">
                  ChatModerator, Assistant, and auto-detected CLI tools when their binaries are on your PATH.
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
                {ollamaStatus.autoInstallSupported ? (
                  <>
                    <p className="text-gray-400 text-xs">
                      Neural Junkie can install Ollama for you (internet required; Linux may ask for your password).
                      Windows runs a silent installer. Or install manually from{' '}
                      <a href="https://ollama.com" className="text-blue-400 underline">ollama.com</a>.
                    </p>
                    <button
                      onClick={installOllama}
                      disabled={installingOllama}
                      className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-500 disabled:opacity-50"
                    >
                      {installingOllama ? 'Installing Ollama…' : 'Install Ollama'}
                    </button>
                    {installStatus && <div className="text-xs text-gray-400">{installStatus}</div>}
                    <button
                      onClick={checkOllama}
                      disabled={installingOllama}
                      className="px-4 py-2 bg-gray-700 text-white rounded hover:bg-gray-600 disabled:opacity-50"
                    >
                      Check Again
                    </button>
                  </>
                ) : (
                  <>
                    <p className="text-gray-400 text-xs">
                      Install Ollama from{' '}
                      <a href="https://ollama.com" className="text-blue-400 underline">ollama.com</a>, then click
                      &quot;Check Again&quot;. Or choose Cloud API instead.
                    </p>
                    <button onClick={checkOllama} className="px-4 py-2 bg-gray-700 text-white rounded hover:bg-gray-600">
                      Check Again
                    </button>
                  </>
                )}
              </div>
            ) : !ollamaStatus.running ? (
              <div className="space-y-3">
                <div className="text-green-400 text-sm">
                  {ollamaStatus.bundled ? 'Ollama is bundled with Neural Junkie but not running.' : 'Ollama is installed but not running.'}
                </div>
                <button onClick={startOllama} className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-500">
                  Start Ollama
                </button>
              </div>
            ) : (
              <div className="space-y-3">
                {hardwareMessage && (
                  <div className="rounded-lg border border-blue-800/50 bg-blue-950/40 p-3 text-sm text-blue-100">
                    {hardwareMessage}
                    <div className="mt-2 text-xs text-blue-200/80">
                      Pull target: <code className="text-teal-300">{selectedOllamaModel}</code>
                      {' · '}
                      <a
                        href={HARDWARE_DOCS_URL}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="underline hover:text-white"
                      >
                        Full hardware guide
                      </a>
                    </div>
                  </div>
                )}
                {autoDowngrade && (
                  <button
                    type="button"
                    onClick={() => setUseFullSizeModel(true)}
                    className="text-xs text-amber-400 underline hover:text-amber-300"
                  >
                    Use {wizardTrack === 'cad' ? CAD_OLLAMA_CHAT_MODEL : DEV_OLLAMA_MODEL} anyway
                  </button>
                )}
                {useFullSizeModel &&
                  shouldAutoDowngradePrimary(hardware?.tier) &&
                  (wizardTrack === 'developer' || wizardTrack === 'cad') && (
                    <button
                      type="button"
                      onClick={() => setUseFullSizeModel(false)}
                      className="text-xs text-gray-400 underline hover:text-gray-300"
                    >
                      Use recommended {wizardTrack === 'cad' ? CAD_OLLAMA_CHAT_MODEL_LIGHT : 'qwen2.5-coder:7b'} instead
                    </button>
                  )}
                <div className="text-green-400 text-sm">
                  {ollamaStatus.bundled
                    ? 'Ollama is bundled and running. Pull a model once to start chatting locally.'
                    : 'Ollama is installed and running.'}
                </div>
                {wizardTrack === 'lifeSciences' && (
                  <p className="text-xs text-gray-500">
                    If pull fails, open Model Library (⇧⌘M), download OpenBioLLM GGUF, and import as <code className="text-teal-400">{trackDefaultModel}</code>.
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
                        } (${selectedOllamaModel})`}
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
                  <strong className="text-white">CLI agents</strong> auto-join #general when their binaries are on your PATH (see /list-cli-agents).
                </p>
                <p>Toggle Assistant below. Coding specialists (BackendEngineer, CodeReviewer, …) are available later via Settings → Domain packs → Software development.</p>
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
          <div className="space-y-4">
            <div className="text-center space-y-2">
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
                {agents.filter((a) => a.enabled).length} configured agent(s) in settings.
                {wizardTrack === 'lifeSciences' && ' Research use only — not for clinical diagnosis.'}
              </p>
            </div>

            <div className="text-left max-h-[50vh] overflow-y-auto pr-1">
              <CLIAgentsManager serverAddr={serverAddr} compact featuredOnly />
            </div>

            <div className="text-center">
              <button
                onClick={saveAndFinish}
                disabled={saving}
                className="px-8 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-500 disabled:opacity-50 font-medium"
              >
                {saving ? 'Saving...' : 'Launch Neural Junkie'}
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
