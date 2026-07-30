import { useState, useEffect, useRef } from 'react';
import {
  agentsForTrack,
  modelsToEnsureForTrack,
  ollamaModelForTrack,
  packsEnabledForTrack,
  inferWizardTrackFromPacks,
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
import { hubMutationPut } from '../utils/hubMutation';

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
  onCancel?: () => void;
  serverAddr: string;
  mode?: 'first-run' | 'rerun';
}

const BIO_HF_REPO = 'aaditya/Llama3-OpenBioLLM-8B';

function ollamaProviderName(track: WizardTrack): string {
  if (track === 'lifeSciences') return 'Local Ollama (Bio 8B)';
  if (track === 'cad') return 'Local Ollama (CAD)';
  if (track === 'general') return 'Local Ollama (utility)';
  return 'Local Ollama (Coder)';
}

function agentsMatchTrack(agents: AgentChoice[], track: WizardTrack): boolean {
  const expected = agentsForTrack(track);
  const types = new Set(agents.map((a) => a.type));
  return expected.every((e) => types.has(e.type));
}

function buildWizardProvider(
  providerType: 'ollama' | 'cloud',
  wizardTrack: WizardTrack,
  selectedOllamaModel: string,
  apiKey: string,
  hfToken: string,
  preferExistingHf = false,
): ProviderChoice {
  if (providerType === 'ollama') {
    return {
      id: 'ollama-local',
      type: 'ollama',
      name: ollamaProviderName(wizardTrack),
      endpoint: 'http://localhost:11434',
      model: selectedOllamaModel,
    };
  }
  if (wizardTrack === 'lifeSciences' && (hfToken.trim() || preferExistingHf)) {
    const provider: ProviderChoice = {
      id: 'hf-bio',
      type: 'huggingface',
      name: 'Hugging Face (OpenBioLLM)',
      model: BIO_HF_REPO,
    };
    if (hfToken.trim()) {
      provider.apiKey = hfToken.trim();
    }
    return provider;
  }
  const provider: ProviderChoice = {
    id: 'anthropic',
    type: 'anthropic',
    name: 'Claude (Anthropic)',
    model: 'claude-3-5-sonnet-20241022',
  };
  if (apiKey.trim()) {
    provider.apiKey = apiKey.trim();
  }
  return provider;
}

export function SetupWizard({
  onComplete,
  onCancel,
  serverAddr,
  mode = 'first-run',
}: SetupWizardProps) {
  const isRerun = mode === 'rerun';
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
  const [prefillReady, setPrefillReady] = useState(!isRerun);
  const [hasExistingAnthropicKey, setHasExistingAnthropicKey] = useState(false);
  const [hasExistingHfToken, setHasExistingHfToken] = useState(false);
  const initialTrackRef = useRef<WizardTrack>('developer');
  const initialDefaultProviderIdRef = useRef<string>('');
  const initialProviderTypeRef = useRef<'ollama' | 'cloud'>('ollama');

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
    if (!isRerun) return;
    let cancelled = false;
    (async () => {
      try {
        const resp = await fetch(`${serverAddr}/api/settings`);
        if (!resp.ok || cancelled) return;
        const config = await resp.json();
        const packsEnabled = (config.packs?.enabled ?? {}) as Record<string, boolean>;
        const track = inferWizardTrackFromPacks(packsEnabled);
        const ai = config.ai ?? {};
        const providers = (ai.providers ?? []) as Array<{
          id?: string;
          type?: string;
          apiKey?: string;
        }>;
        const defaultId = String(ai.default_provider_id ?? '');
        const defaultProvider =
          providers.find((p) => p.id === defaultId) ?? providers[0];
        const inferredProviderType: 'ollama' | 'cloud' =
          defaultProvider?.type === 'ollama' ? 'ollama' : 'cloud';
        const existingAgents = (config.agents ?? []) as AgentChoice[];
        const seededAgents =
          Array.isArray(existingAgents) &&
          existingAgents.length > 0 &&
          agentsMatchTrack(existingAgents, track)
            ? existingAgents.map((a) => ({
                type: a.type,
                name: a.name,
                enabled: Boolean(a.enabled),
              }))
            : agentsForTrack(track);

        const hfCfg = config.hf as { token?: string } | undefined;
        const hasHf =
          Boolean(hfCfg?.token) ||
          providers.some((p) => p.type === 'huggingface' && Boolean(p.apiKey));
        const hasAnthropic = providers.some(
          (p) => p.type === 'anthropic' && Boolean(p.apiKey),
        );

        if (cancelled) return;
        initialTrackRef.current = track;
        initialDefaultProviderIdRef.current = defaultId;
        initialProviderTypeRef.current = inferredProviderType;
        setWizardTrack(track);
        setProviderType(inferredProviderType);
        setAgents(seededAgents);
        setHasExistingHfToken(hasHf);
        setHasExistingAnthropicKey(hasAnthropic);
      } catch (e) {
        console.error('Failed to prefill setup wizard:', e);
      } finally {
        if (!cancelled) setPrefillReady(true);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [isRerun, serverAddr]);

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

  function cloudNextDisabled(): boolean {
    if (wizardTrack === 'lifeSciences') {
      if (hfToken.trim() || apiKey.trim()) return false;
      if (isRerun && (hasExistingHfToken || hasExistingAnthropicKey)) return false;
      return true;
    }
    if (apiKey.trim()) return false;
    if (isRerun && hasExistingAnthropicKey) return false;
    return true;
  }

  function chosenProviderId(): string {
    if (providerType === 'ollama') return 'ollama-local';
    if (wizardTrack === 'lifeSciences' && (hfToken.trim() || (isRerun && hasExistingHfToken && !apiKey.trim()))) {
      return 'hf-bio';
    }
    return 'anthropic';
  }

  function shouldConfirmRerun(): boolean {
    if (!isRerun) return false;
    const trackChanged = wizardTrack !== initialTrackRef.current;
    const providerChanged =
      providerType !== initialProviderTypeRef.current ||
      chosenProviderId() !== initialDefaultProviderIdRef.current;
    return trackChanged || providerChanged;
  }

  async function saveAndFinish() {
    if (shouldConfirmRerun()) {
      const ok = window.confirm(
        'This will update agents and domain packs for the selected focus. Continue?',
      );
      if (!ok) return;
    }

    setSaving(true);
    const preferExistingHf =
      providerType === 'cloud' &&
      wizardTrack === 'lifeSciences' &&
      !hfToken.trim() &&
      !apiKey.trim() &&
      hasExistingHfToken;
    const wizardProvider = buildWizardProvider(
      providerType,
      wizardTrack,
      selectedOllamaModel,
      apiKey,
      hfToken,
      preferExistingHf,
    );

    try {
      if (isRerun) {
        await saveRerunMerge(wizardProvider);
      } else {
        await saveFirstRun(wizardProvider);
      }
      await fetch(`${serverAddr}/api/agents/restart`, { method: 'POST' });
      setSaving(false);
      onComplete();
    } catch (e) {
      console.error('Failed to save config:', e);
      setSaving(false);
      window.alert(
        e instanceof Error
          ? `Could not save setup: ${e.message}`
          : 'Could not save setup. Check that the hub is running and try again.',
      );
    }
  }

  async function saveFirstRun(wizardProvider: ProviderChoice) {
    const config: Record<string, unknown> = {
      setup_completed: true,
      server: { host: 'localhost', port: 18765 },
      ai: {
        default_provider_id: wizardProvider.id,
        providers: [wizardProvider],
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

    const put = await hubMutationPut(serverAddr, '/api/settings', config);
    if (!put.ok) {
      throw new Error(`Failed to save settings: ${put.status} ${await put.text()}`);
    }
  }

  async function saveRerunMerge(wizardProvider: ProviderChoice) {
    const resp = await fetch(`${serverAddr}/api/settings`);
    if (!resp.ok) {
      throw new Error(`Failed to load settings: ${resp.status}`);
    }
    const current = await resp.json();
    const ai = (current.ai ?? {}) as {
      providers?: ProviderChoice[];
      default_provider_id?: string;
    };
    const existingProviders = Array.isArray(ai.providers) ? [...ai.providers] : [];

    const upsert: ProviderChoice = { ...wizardProvider };
    // Avoid blanking stored secrets when the form fields were left empty.
    if (!apiKey.trim() && upsert.type === 'anthropic') {
      delete upsert.apiKey;
    }
    if (!hfToken.trim() && upsert.type === 'huggingface') {
      delete upsert.apiKey;
    }

    const idx = existingProviders.findIndex((p) => p.id === upsert.id);
    if (idx >= 0) {
      const prev = existingProviders[idx];
      existingProviders[idx] = {
        ...prev,
        ...upsert,
        apiKey: upsert.apiKey !== undefined ? upsert.apiKey : prev.apiKey,
      };
    } else {
      existingProviders.push(upsert);
    }

    const packsEnabled = {
      ...((current.packs?.enabled ?? {}) as Record<string, boolean>),
      ...packsEnabledForTrack(wizardTrack),
    };

    const config: Record<string, unknown> = {
      setup_completed: true,
      ai: {
        default_provider_id: upsert.id,
        providers: existingProviders,
      },
      agents: agents.map(a => ({ type: a.type, name: a.name, enabled: a.enabled })),
      packs: { enabled: packsEnabled },
      mcp: {
        ...(typeof current.mcp === 'object' && current.mcp ? current.mcp : {}),
        enabled: true,
      },
      ollama: {
        ...(typeof current.ollama === 'object' && current.ollama ? current.ollama : {}),
        auto_start: providerType === 'ollama',
        models_to_ensure: providerType === 'ollama'
          ? modelsToEnsureForTrack(wizardTrack, providerType).map((m) =>
              m === trackDefaultModel ? selectedOllamaModel : m,
            )
          : ((current.ollama as { models_to_ensure?: string[] } | undefined)?.models_to_ensure ?? []),
      },
      updates: {
        ...(typeof current.updates === 'object' && current.updates ? current.updates : {}),
        auto_check: true,
      },
    };
    if (hfToken.trim()) {
      config.hf = {
        ...(typeof current.hf === 'object' && current.hf ? current.hf : {}),
        token: hfToken.trim(),
      };
    }

    const put = await hubMutationPut(serverAddr, '/api/settings', config);
    if (!put.ok) {
      throw new Error(`Failed to save settings: ${put.status} ${await put.text()}`);
    }
  }

  const toggleAgent = (type: string) => {
    setAgents(prev => prev.map(a => a.type === type ? { ...a, enabled: !a.enabled } : a));
  };

  const steps = ['Welcome', 'Focus', 'Provider', 'Setup', 'Agents', 'Done'];

  if (!prefillReady) {
    return (
      <div className="flex items-center justify-center w-full h-screen bg-gray-950">
        <div className="text-gray-400 text-sm">Loading current setup…</div>
      </div>
    );
  }

  return (
    <div className="flex items-center justify-center w-full h-screen bg-gray-950">
      <div className="w-full max-w-xl p-8 space-y-6">
        <div className="flex gap-2 justify-center">
          {steps.map((s, i) => (
            <div key={s} className={`h-1 w-10 rounded ${i <= step ? 'bg-blue-500' : 'bg-gray-700'}`} />
          ))}
        </div>

        {isRerun && onCancel && (
          <div className="flex justify-end">
            <button
              type="button"
              onClick={onCancel}
              className="text-sm text-gray-400 hover:text-white transition-colors"
            >
              Cancel
            </button>
          </div>
        )}

        {step === 0 && (
          <div className="text-center space-y-4">
            <h1 className="text-3xl font-bold text-white">
              {isRerun ? 'Run setup again' : 'Welcome to Neural Junkie'}
            </h1>
            <p className="text-gray-400">
              {isRerun
                ? 'Update your focus track, AI backend, agents, and domain packs.'
                : "Let's set up your multi-agent AI collaboration environment."}
            </p>
            <button onClick={() => setStep(1)} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-500 transition-colors">
              {isRerun ? 'Continue' : 'Get Started'}
            </button>
          </div>
        )}

        {step === 1 && (
          <div className="space-y-4">
            <h2 className="text-xl font-semibold text-white text-center">What are you here for?</h2>
            <div className="grid grid-cols-1 gap-4">
              <button
                onClick={() => selectTrack('developer')}
                className={`p-4 rounded-lg border text-left space-y-2 transition-colors ${
                  wizardTrack === 'developer' ? 'border-blue-500 bg-blue-500/10' : 'border-gray-700 hover:border-blue-500'
                }`}
              >
                <div className="font-medium text-white">Software development</div>
                <div className="text-xs text-gray-400">
                  Coding specialists, repo context, and Qwen Coder models.
                </div>
              </button>
              <button
                onClick={() => selectTrack('lifeSciences')}
                className={`p-4 rounded-lg border text-left space-y-2 transition-colors ${
                  wizardTrack === 'lifeSciences' ? 'border-teal-500 bg-teal-500/10' : 'border-gray-700 hover:border-teal-500'
                }`}
              >
                <div className="font-medium text-white">Life sciences &amp; lab work</div>
                <div className="text-xs text-gray-400">
                  Neural Junkie Bio 8B, BiologyExpert, sequence tools, and structure prediction.
                </div>
              </button>
              <button
                onClick={() => selectTrack('cad')}
                className={`p-4 rounded-lg border text-left space-y-2 transition-colors ${
                  wizardTrack === 'cad' ? 'border-indigo-500 bg-indigo-500/10' : 'border-gray-700 hover:border-indigo-500'
                }`}
              >
                <div className="font-medium text-white">CAD &amp; mechanical design</div>
                <div className="text-xs text-gray-400">
                  CADExpert, OpenSCAD workbench, and Qwen Coder models. Requires OpenSCAD installed.
                </div>
              </button>
              <button
                onClick={() => selectTrack('general')}
                className={`p-4 rounded-lg border text-left space-y-2 transition-colors ${
                  wizardTrack === 'general' ? 'border-violet-500 bg-violet-500/10' : 'border-gray-700 hover:border-violet-500'
                }`}
              >
                <div className="font-medium text-white">Team chat &amp; productivity</div>
                <div className="text-xs text-gray-400">
                  Assistant and auto-detected CLI tools when their binaries are on your PATH.
                  Enable the IDE and Software development packs later for editor depth and coding specialists.
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
                      Neural Junkie can install Ollama for you on Windows, macOS, and Linux (internet required).
                      Approve the system password / UAC dialog when it appears. Or install manually from{' '}
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
            {isRerun && (hasExistingHfToken || hasExistingAnthropicKey) && (
              <p className="text-xs text-gray-500">
                Existing keys are kept if you leave these fields blank.
              </p>
            )}
            {wizardTrack === 'lifeSciences' && (
              <>
                <label className="block text-xs text-gray-400">Hugging Face token (recommended for Bio 8B hosted)</label>
                <input
                  type="password"
                  value={hfToken}
                  onChange={(e) => setHfToken(e.target.value)}
                  placeholder={isRerun && hasExistingHfToken ? 'Leave blank to keep existing' : 'hf_...'}
                  className="w-full px-4 py-2 bg-gray-800 border border-gray-700 rounded-lg text-white focus:border-teal-500 focus:outline-none"
                />
                <p className="text-xs text-gray-500">Also used for ESMFold structure prediction (saved in Settings).</p>
              </>
            )}
            <label className="block text-xs text-gray-400">
              {wizardTrack === 'lifeSciences' && (hfToken.trim() || (isRerun && hasExistingHfToken))
                ? 'Anthropic key (optional fallback)'
                : 'Anthropic API key'}
            </label>
            <input
              type="password"
              value={apiKey}
              onChange={(e) => setApiKey(e.target.value)}
              placeholder={isRerun && hasExistingAnthropicKey ? 'Leave blank to keep existing' : 'sk-ant-...'}
              className="w-full px-4 py-2 bg-gray-800 border border-gray-700 rounded-lg text-white focus:border-blue-500 focus:outline-none"
            />
            <button
              onClick={() => setStep(4)}
              disabled={cloudNextDisabled()}
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
                  <strong className="text-white">Assistant</strong> is always on with the hub (productivity, platform help, and chat commands).
                </p>
                <p>
                  <strong className="text-white">CLI agents</strong> auto-join #general when their binaries are on your PATH (see /list-cli-agents).
                </p>
                <p>Toggle Assistant below. Coding specialists (BackendEngineer, CodeReviewer, …) are available later via Settings → Domain packs → Software development.</p>
              </div>
            ) : wizardTrack === 'developer' ? (
              <p className="text-sm text-gray-400 text-center">
                Assistant is configured here. The IDE pack (editor depth) and Software development pack (engineering specialists) are enabled for this track.
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
              <h2 className="text-2xl font-bold text-white">{isRerun ? 'Ready to apply' : 'All Set!'}</h2>
              <p className="text-gray-400 text-sm">
                {providerType === 'ollama'
                  ? wizardTrack === 'lifeSciences'
                    ? 'BiologyExpert will use Neural Junkie Bio 8B locally.'
                    : 'Your agents will use local Ollama models.'
                  : wizardTrack === 'lifeSciences' && (hfToken.trim() || (isRerun && hasExistingHfToken && !apiKey.trim()))
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
                onClick={() => void saveAndFinish()}
                disabled={saving}
                className="px-8 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-500 disabled:opacity-50 font-medium"
              >
                {saving ? 'Saving...' : isRerun ? 'Apply setup' : 'Launch Neural Junkie'}
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
