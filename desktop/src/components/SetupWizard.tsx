import { useState, useEffect } from 'react';

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

const defaultAgents: AgentChoice[] = [
  { type: 'backend', name: 'GoExpert', enabled: true },
  { type: 'frontend', name: 'ReactExpert', enabled: true },
  { type: 'devops', name: 'DevOpsPro', enabled: true },
  { type: 'database', name: 'SQLMaster', enabled: true },
  { type: 'security', name: 'SecurityExpert', enabled: true },
  { type: 'rust', name: 'RustExpert', enabled: true },
];

export function SetupWizard({ onComplete, serverAddr }: SetupWizardProps) {
  const [step, setStep] = useState(0);
  const [providerType, setProviderType] = useState<'ollama' | 'cloud'>('ollama');
  const [ollamaStatus, setOllamaStatus] = useState<{ installed: boolean; running: boolean } | null>(null);
  const [apiKey, setApiKey] = useState('');
  const [agents, setAgents] = useState<AgentChoice[]>(defaultAgents);
  const [pulling, setPulling] = useState(false);
  const [pullStatus, setPullStatus] = useState('');
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (step === 2 && providerType === 'ollama') {
      checkOllama();
    }
  }, [step, providerType]);

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
    setPullStatus('Pulling qwen2.5-coder:14b...');
    try {
      const resp = await fetch(`${serverAddr}/api/ollama/pull`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ model: 'qwen2.5-coder:14b' }),
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

  async function saveAndFinish() {
    setSaving(true);
    const providers: ProviderChoice[] = [];

    if (providerType === 'ollama') {
      providers.push({
        id: 'ollama-local',
        type: 'ollama',
        name: 'Local Ollama',
        endpoint: 'http://localhost:11434',
        model: 'qwen2.5-coder:14b',
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

    const config = {
      server: { host: 'localhost', port: 18765 },
      ai: {
        default_provider_id: providers[0].id,
        providers,
      },
      agents: agents.map(a => ({ type: a.type, name: a.name, enabled: a.enabled })),
      ollama: {
        auto_start: providerType === 'ollama',
        models_to_ensure: providerType === 'ollama' ? ['qwen2.5-coder:14b', 'qwen2.5:7b'] : [],
      },
      updates: { auto_check: true },
    };

    try {
      await fetch(`${serverAddr}/api/settings`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(config),
      });
      // Restart agents with new config
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

  const steps = ['Welcome', 'Provider', 'Setup', 'Agents', 'Done'];

  return (
    <div className="flex items-center justify-center w-full h-screen bg-gray-950">
      <div className="w-full max-w-lg p-8 space-y-6">
        {/* Progress */}
        <div className="flex gap-2 justify-center">
          {steps.map((s, i) => (
            <div key={s} className={`h-1 w-12 rounded ${i <= step ? 'bg-blue-500' : 'bg-gray-700'}`} />
          ))}
        </div>

        {/* Step 0: Welcome */}
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

        {/* Step 1: Provider choice */}
        {step === 1 && (
          <div className="space-y-4">
            <h2 className="text-xl font-semibold text-white text-center">Choose your AI backend</h2>
            <div className="grid grid-cols-2 gap-4">
              <button
                onClick={() => { setProviderType('ollama'); setStep(2); }}
                className={`p-4 rounded-lg border text-left space-y-2 transition-colors ${
                  providerType === 'ollama' ? 'border-blue-500 bg-blue-500/10' : 'border-gray-700 hover:border-gray-500'
                }`}
              >
                <div className="font-medium text-white">Local Models</div>
                <div className="text-xs text-gray-400">Run AI locally with Ollama. Free, private, no API key needed.</div>
              </button>
              <button
                onClick={() => { setProviderType('cloud'); setStep(2); }}
                className={`p-4 rounded-lg border text-left space-y-2 transition-colors ${
                  providerType === 'cloud' ? 'border-blue-500 bg-blue-500/10' : 'border-gray-700 hover:border-gray-500'
                }`}
              >
                <div className="font-medium text-white">Cloud API</div>
                <div className="text-xs text-gray-400">Use Anthropic Claude. Requires an API key.</div>
              </button>
            </div>
          </div>
        )}

        {/* Step 2: Provider setup */}
        {step === 2 && providerType === 'ollama' && (
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
                {!pullStatus.includes('ready') && (
                  <button onClick={pullDefaultModel} disabled={pulling} className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-500 disabled:opacity-50">
                    {pulling ? pullStatus : 'Pull Default Model (qwen2.5-coder:14b)'}
                  </button>
                )}
                {pullStatus && <div className="text-xs text-gray-400">{pullStatus}</div>}
                <button onClick={() => setStep(3)} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-500">
                  Next
                </button>
              </div>
            )}
          </div>
        )}

        {step === 2 && providerType === 'cloud' && (
          <div className="space-y-4">
            <h2 className="text-xl font-semibold text-white text-center">Anthropic API Key</h2>
            <input
              type="password"
              value={apiKey}
              onChange={(e) => setApiKey(e.target.value)}
              placeholder="sk-ant-..."
              className="w-full px-4 py-2 bg-gray-800 border border-gray-700 rounded-lg text-white focus:border-blue-500 focus:outline-none"
            />
            <p className="text-xs text-gray-500">
              Get your key from{' '}
              <a href="https://console.anthropic.com" className="text-blue-400 underline">console.anthropic.com</a>
            </p>
            <button
              onClick={() => setStep(3)}
              disabled={!apiKey}
              className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-500 disabled:opacity-50"
            >
              Next
            </button>
          </div>
        )}

        {/* Step 3: Agent config */}
        {step === 3 && (
          <div className="space-y-4">
            <h2 className="text-xl font-semibold text-white text-center">Configure Agents</h2>
            <p className="text-sm text-gray-400 text-center">Choose which specialist agents to enable.</p>
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
            <button onClick={() => setStep(4)} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-500">
              Next
            </button>
          </div>
        )}

        {/* Step 4: Done */}
        {step === 4 && (
          <div className="text-center space-y-4">
            <h2 className="text-2xl font-bold text-white">All Set!</h2>
            <p className="text-gray-400 text-sm">
              {providerType === 'ollama'
                ? 'Your agents will use local Ollama models.'
                : 'Your agents will use the Anthropic Claude API.'}
            </p>
            <p className="text-gray-500 text-xs">
              {agents.filter(a => a.enabled).length} agents enabled. You can change this later in Settings.
            </p>
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
