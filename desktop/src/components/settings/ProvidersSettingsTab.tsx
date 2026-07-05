import { useState, useEffect } from 'react';
import { shallow } from 'zustand/shallow';
import { useSettingsStore } from '../../stores/settingsStore';
import { useChatStore } from '../../stores/chatStore';
import { ProviderManager } from '../ProviderManager';
import { CLIAgentsManager } from '../CLIAgentsManager';
import {
  fetchHardwareSnapshot,
  fetchModelLookup,
  formatModelResourceHint,
  type HardwareSnapshot,
  type ModelLookup,
} from '../../utils/hardwareRecommendations';
import type { OllamaSettings, LMStudioSettings } from '../../types/protocol';
import { putSystemSecurity, type SettingsTabProps } from './settingsShared';

type CLIAgentsForm = {
  disable_interactive: boolean;
  cursor_trust: boolean;
  disable_pty: boolean;
  gemini_cli_pty: boolean;
  gemini_cli_home: string;
};

export function ProvidersSettingsTab({ hubHttp, isActive }: SettingsTabProps) {
  const {
    integrations,
    loadIntegrations,
    updateOllamaSettings,
    updateLMStudioSettings,
    fetchOllamaModels,
    fetchLMStudioModels,
    testOllamaConnection,
    testLMStudioConnection,
  } = useSettingsStore();
  const { switchAllAgentProviders } = useChatStore(
    (s) => ({ switchAllAgentProviders: s.switchAllAgentProviders }),
    shallow
  );

  const [ollamaForm, setOllamaForm] = useState<OllamaSettings>(integrations.ollama);
  const [hardwareSnapshot, setHardwareSnapshot] = useState<HardwareSnapshot | null>(null);
  const [defaultModelLookup, setDefaultModelLookup] = useState<ModelLookup | null>(null);
  const [lmstudioForm, setLMStudioForm] = useState<LMStudioSettings>(integrations.lmstudio);
  const [testResults, setTestResults] = useState<Record<string, { success: boolean; message: string }>>({});
  const [isSwitching, setIsSwitching] = useState(false);
  const [cliAgents, setCliAgents] = useState<CLIAgentsForm>({
    disable_interactive: false,
    cursor_trust: true,
    disable_pty: false,
    gemini_cli_pty: false,
    gemini_cli_home: '',
  });
  const [cliAgentsSaving, setCliAgentsSaving] = useState(false);

  useEffect(() => {
    if (!isActive) return;
    let cancelled = false;
    (async () => {
      try {
        const r = await fetch(`${hubHttp}/api/system/security`);
        if (!r.ok) throw new Error(await r.text());
        const data = await r.json();
        const c = data.cli_agents ?? {};
        if (!cancelled) {
          setCliAgents({
            disable_interactive: !!c.disable_interactive,
            cursor_trust: c.cursor_trust !== false,
            disable_pty: !!c.disable_pty,
            gemini_cli_pty: !!c.gemini_cli_pty,
            gemini_cli_home: String(c.gemini_cli_home ?? ''),
          });
        }
      } catch {
        /* hub may be offline */
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [isActive, hubHttp]);

  const saveCLIAgentsSettings = async () => {
    setCliAgentsSaving(true);
    try {
      await putSystemSecurity(hubHttp, { cli_agents: cliAgents });
      setTestResults((prev) => ({
        ...prev,
        cliAgents: { success: true, message: 'CLI agent settings saved.' },
      }));
    } catch (error) {
      setTestResults((prev) => ({
        ...prev,
        cliAgents: {
          success: false,
          message: error instanceof Error ? error.message : 'Failed to save CLI agent settings',
        },
      }));
    } finally {
      setCliAgentsSaving(false);
    }
  };

  useEffect(() => {
    if (!isActive) return;
    loadIntegrations();
  }, [isActive, loadIntegrations]);

  useEffect(() => {
    setOllamaForm(integrations.ollama);
    setLMStudioForm(integrations.lmstudio);
  }, [integrations]);

  useEffect(() => {
    if (!isActive) return;
    let cancelled = false;
    const loadModels = async () => {
      try {
        const ollamaModels = await fetchOllamaModels();
        if (!cancelled) setOllamaForm((prev) => ({ ...prev, availableModels: ollamaModels }));
      } catch { /* Ollama may not be running */ }
      try {
        const lmModels = await fetchLMStudioModels();
        if (!cancelled) setLMStudioForm((prev) => ({ ...prev, availableModels: lmModels }));
      } catch { /* LM Studio may not be running */ }
    };
    void loadModels();
    return () => { cancelled = true; };
  }, [isActive, fetchOllamaModels, fetchLMStudioModels]);

  useEffect(() => {
    if (!isActive) return;
    let cancelled = false;
    void fetchHardwareSnapshot(hubHttp).then((snap) => {
      if (!cancelled) setHardwareSnapshot(snap);
    });
    return () => { cancelled = true; };
  }, [isActive, hubHttp]);

  useEffect(() => {
    if (!isActive) return;
    const model = ollamaForm.defaultModel?.trim();
    if (!model) {
      setDefaultModelLookup(null);
      return;
    }
    let cancelled = false;
    void fetchModelLookup(hubHttp, model).then((row) => {
      if (!cancelled) setDefaultModelLookup(row);
    });
    return () => { cancelled = true; };
  }, [isActive, hubHttp, ollamaForm.defaultModel]);

const handleOllamaChange = (field: keyof OllamaSettings, value: string | string[]) => {
      setOllamaForm(prev => ({ ...prev, [field]: value }));
    };

    const handleLMStudioChange = (field: keyof LMStudioSettings, value: string | string[]) => {
      setLMStudioForm(prev => ({ ...prev, [field]: value }));
    };
    const saveOllamaSettings = async () => {
      try {
        await updateOllamaSettings(ollamaForm);
        setTestResults(prev => ({ ...prev, ollama: { success: true, message: 'Settings saved successfully!' } }));
      } catch (error) {
        setTestResults(prev => ({ 
          ...prev, 
          ollama: { 
            success: false, 
            message: error instanceof Error ? error.message : 'Failed to save settings' 
          } 
        }));
      }
    };

    const saveLMStudioSettings = async () => {
      try {
        await updateLMStudioSettings(lmstudioForm);
        setTestResults(prev => ({ ...prev, lmstudio: { success: true, message: 'Settings saved successfully!' } }));
      } catch (error) {
        setTestResults(prev => ({ 
          ...prev, 
          lmstudio: { 
            success: false, 
            message: error instanceof Error ? error.message : 'Failed to save settings' 
          } 
        }));
      }
    };
    const testConnection = async (service: string) => {
      setTestResults(prev => ({ ...prev, [service]: { success: false, message: 'Testing...' } }));
    
      try {
        let result = false;
        switch (service) {
          case 'ollama':
            result = await testOllamaConnection();
            break;
          case 'lmstudio':
            result = await testLMStudioConnection();
            break;
        }
      
        setTestResults(prev => ({ 
          ...prev, 
          [service]: { 
            success: result, 
            message: result ? 'Connection successful!' : 'Connection failed. Check your credentials.' 
          } 
        }));
      } catch (error) {
        setTestResults(prev => ({ 
          ...prev, 
          [service]: { 
            success: false, 
            message: `Error: ${error instanceof Error ? error.message : 'Unknown error'}` 
          } 
        }));
      }
    };
  const handleSwitchAllToClaude = async () => {
      setIsSwitching(true);
      try {
        await switchAllAgentProviders('claude', 'claude-sonnet');
        setTestResults(prev => ({ 
          ...prev, 
          providerSwitch: { 
            success: true, 
            message: 'All agents switched to Claude successfully!' 
          } 
        }));
      } catch (error) {
        console.error('Failed to switch all agents to Claude:', error);
        setTestResults(prev => ({ 
          ...prev, 
          providerSwitch: { 
            success: false, 
            message: error instanceof Error ? error.message : 'Failed to switch all agents to Claude' 
          } 
        }));
      } finally {
        setIsSwitching(false);
      }
    };

    const handleSwitchAllToOllama = async () => {
      setIsSwitching(true);
      try {
        const model = ollamaForm.defaultModel || 'llama3.1';
        await switchAllAgentProviders('ollama', model);
        setTestResults(prev => ({ 
          ...prev, 
          providerSwitch: { 
            success: true, 
            message: `All agents switched to Ollama (${model}) successfully!` 
          } 
        }));
      } catch (error) {
        console.error('Failed to switch all agents to Ollama:', error);
        setTestResults(prev => ({ 
          ...prev, 
          providerSwitch: { 
            success: false, 
            message: error instanceof Error ? error.message : 'Failed to switch all agents to Ollama' 
          } 
        }));
      } finally {
        setIsSwitching(false);
      }
    };

    const handleSwitchAllToLMStudio = async () => {
      setIsSwitching(true);
      try {
        const model = lmstudioForm.defaultModel || (lmstudioForm.availableModels[0] ?? '');
        await switchAllAgentProviders('lmstudio', model);
        setTestResults(prev => ({ 
          ...prev, 
          providerSwitch: { 
            success: true, 
            message: `All agents switched to LM Studio${model ? ` (${model})` : ''} successfully!` 
          } 
        }));
      } catch (error) {
        console.error('Failed to switch all agents to LM Studio:', error);
        setTestResults(prev => ({ 
          ...prev, 
          providerSwitch: { 
            success: false, 
            message: error instanceof Error ? error.message : 'Failed to switch all agents to LM Studio' 
          } 
        }));
      } finally {
        setIsSwitching(false);
      }
    };

  if (!isActive) return null;

  return (
    <div className="space-y-8">
{/* CLI agent install & auth */}
    <div className="border border-slack-border rounded-lg p-6">
      <CLIAgentsManager serverAddr={hubHttp} />
    </div>
{/* CLI agent runtime (hub config) */}
    <div className="border border-slack-border rounded-lg p-6">
      <h3 className="text-lg font-semibold text-slack-text mb-2">CLI agent runtime</h3>
      <p className="text-sm text-slack-textMuted mb-4">
        PTY, trust, and Gemini CLI home directory. Environment variables override when set.
      </p>
      {testResults.cliAgents && (
        <div
          className={`mb-4 p-3 rounded text-sm ${
            testResults.cliAgents.success
              ? 'bg-green-100 text-green-800 border border-green-200'
              : 'bg-red-100 text-red-800 border border-red-200'
          }`}
        >
          {testResults.cliAgents.message}
        </div>
      )}
      <div className="space-y-3">
        <label className="flex items-center gap-2 text-sm text-slack-text">
          <input
            type="checkbox"
            checked={cliAgents.disable_interactive}
            onChange={(e) =>
              setCliAgents((p) => ({ ...p, disable_interactive: e.target.checked }))
            }
          />
          Disable interactive CLI prompts (non-TTY)
        </label>
        <label className="flex items-center gap-2 text-sm text-slack-text">
          <input
            type="checkbox"
            checked={cliAgents.cursor_trust}
            onChange={(e) => setCliAgents((p) => ({ ...p, cursor_trust: e.target.checked }))}
          />
          Cursor CLI trust workspace (default on)
        </label>
        <label className="flex items-center gap-2 text-sm text-slack-text">
          <input
            type="checkbox"
            checked={cliAgents.disable_pty}
            onChange={(e) => setCliAgents((p) => ({ ...p, disable_pty: e.target.checked }))}
          />
          Disable PTY for all CLI agents
        </label>
        <label className="flex items-center gap-2 text-sm text-slack-text">
          <input
            type="checkbox"
            checked={cliAgents.gemini_cli_pty}
            onChange={(e) => setCliAgents((p) => ({ ...p, gemini_cli_pty: e.target.checked }))}
          />
          Gemini CLI PTY mode
        </label>
        <label className="block text-sm text-slack-text">
          Gemini CLI home directory
          <input
            type="text"
            value={cliAgents.gemini_cli_home}
            onChange={(e) => setCliAgents((p) => ({ ...p, gemini_cli_home: e.target.value }))}
            placeholder="~/.gemini (empty = default)"
            className="mt-1 w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded font-mono text-xs"
          />
        </label>
        <button
          type="button"
          onClick={() => void saveCLIAgentsSettings()}
          disabled={cliAgentsSaving}
          className="w-full px-4 py-2 bg-slack-accent text-white rounded hover:bg-slack-accentHover disabled:opacity-50"
        >
          {cliAgentsSaving ? 'Saving…' : 'Save CLI agent runtime'}
        </button>
      </div>
    </div>
{/* Dynamic Provider Registry */}
    <div className="border border-slack-border rounded-lg p-6">
      <ProviderManager serverAddr={hubHttp} />
    </div>
{/* Ollama Settings (legacy) */}
    <div className="border border-slack-border rounded-lg p-6">
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-lg font-semibold text-slack-text">Ollama (Local LLM)</h3>
        <div className="flex items-center space-x-2">
          {ollamaForm.endpoint && (
            <span className="text-green-500 text-sm">✓ Configured</span>
          )}
          <button
            onClick={() => testConnection('ollama')}
            className="px-3 py-1 text-sm bg-slack-accent text-white rounded hover:bg-slack-accentHover transition-colors"
          >
            Test
          </button>
        </div>
      </div>
      
      {testResults.ollama && (
        <div className={`mb-4 p-3 rounded text-sm ${
          testResults.ollama.success 
            ? 'bg-green-100 text-green-800 border border-green-200' 
            : 'bg-red-100 text-red-800 border border-red-200'
        }`}>
          {testResults.ollama.message}
        </div>
      )}

      <div className="space-y-4">
        <div>
          <label className="block text-sm font-medium text-slack-text mb-2">
            Ollama Endpoint
          </label>
          <input
            type="text"
            value={ollamaForm.endpoint}
            onChange={(e) => handleOllamaChange('endpoint', e.target.value)}
            placeholder="http://localhost:11434"
            className="w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-slack-text focus:outline-none focus:ring-2 focus:ring-slack-accent"
          />
          <p className="text-xs text-slack-textMuted mt-1">
            URL where Ollama server is running (default: http://localhost:11434)
          </p>
        </div>

        <div>
          <label className="block text-sm font-medium text-slack-text mb-2">
            Default Model
          </label>
          <div className="flex items-center gap-2">
            <select
              value={ollamaForm.defaultModel}
              onChange={(e) => handleOllamaChange('defaultModel', e.target.value)}
              className="flex-1 px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-slack-text focus:outline-none focus:ring-2 focus:ring-slack-accent"
            >
              {ollamaForm.availableModels.length > 0 ? (
                ollamaForm.availableModels.map((model) => (
                  <option key={model} value={model}>{model}</option>
                ))
              ) : (
                <>
                  <option value="llama3.1">llama3.1</option>
                  <option value="mistral">mistral</option>
                  <option value="codellama">codellama</option>
                  <option value="phi3">phi3</option>
                  <option value="gemma">gemma</option>
                </>
              )}
            </select>
            <button
              onClick={async () => {
                try {
                  const models = await fetchOllamaModels();
                  setOllamaForm(prev => ({ ...prev, availableModels: models }));
                } catch (error) {
                  console.error('Failed to fetch Ollama models:', error);
                }
              }}
              className="px-3 py-2 text-sm bg-blue-500 text-white rounded hover:bg-blue-600 transition-colors whitespace-nowrap"
              title="Fetch available models from Ollama"
            >
              Refresh
            </button>
          </div>
          <p className="text-xs text-slack-textMuted mt-1">
            {ollamaForm.availableModels.length > 0
              ? `${ollamaForm.availableModels.length} models available`
              : 'Click Refresh to load models from Ollama'}
          </p>
          {(formatModelResourceHint(defaultModelLookup) || hardwareSnapshot) && (
            <p className="text-xs text-slack-textMuted mt-2">
              {formatModelResourceHint(defaultModelLookup)}
              {formatModelResourceHint(defaultModelLookup) && hardwareSnapshot ? ' · ' : ''}
              {hardwareSnapshot
                ? `Your system: ${hardwareSnapshot.total_memory_gb} GB RAM (${hardwareSnapshot.tier} tier)`
                : ''}
            </p>
          )}
        </div>

        <button
          onClick={saveOllamaSettings}
          className="w-full px-4 py-2 bg-slack-accent text-white rounded hover:bg-slack-accentHover transition-colors"
        >
          Save Ollama Settings
        </button>
      </div>
    </div>
{/* LM Studio Settings */}
    <div className="border border-slack-border rounded-lg p-6">
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-lg font-semibold text-slack-text">LM Studio (Local LLM)</h3>
        <div className="flex items-center space-x-2">
          {lmstudioForm.endpoint && (
            <span className="text-green-500 text-sm">✓ Configured</span>
          )}
          <button
            onClick={() => testConnection('lmstudio')}
            className="px-3 py-1 text-sm bg-slack-accent text-white rounded hover:bg-slack-accentHover transition-colors"
          >
            Test
          </button>
        </div>
      </div>
      
      {testResults.lmstudio && (
        <div className={`mb-4 p-3 rounded text-sm ${
          testResults.lmstudio.success 
            ? 'bg-green-100 text-green-800 border border-green-200' 
            : 'bg-red-100 text-red-800 border border-red-200'
        }`}>
          {testResults.lmstudio.message}
        </div>
      )}

      <div className="space-y-4">
        <div>
          <label className="block text-sm font-medium text-slack-text mb-2">
            LM Studio Endpoint
          </label>
          <input
            type="text"
            value={lmstudioForm.endpoint}
            onChange={(e) => handleLMStudioChange('endpoint', e.target.value)}
            placeholder="http://localhost:1234/v1"
            className="w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-slack-text focus:outline-none focus:ring-2 focus:ring-slack-accent"
          />
          <p className="text-xs text-slack-textMuted mt-1">
            URL where LM Studio server is running (default: http://localhost:1234/v1)
          </p>
        </div>

        <div>
          <label className="block text-sm font-medium text-slack-text mb-2">
            Default Model
          </label>
          <div className="flex items-center gap-2">
            {lmstudioForm.availableModels.length > 0 ? (
              <select
                value={lmstudioForm.defaultModel}
                onChange={(e) => handleLMStudioChange('defaultModel', e.target.value)}
                className="flex-1 px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-slack-text focus:outline-none focus:ring-2 focus:ring-slack-accent"
              >
                <option value="">Auto-select</option>
                {lmstudioForm.availableModels.map((model) => (
                  <option key={model} value={model}>{model}</option>
                ))}
              </select>
            ) : (
              <input
                type="text"
                value={lmstudioForm.defaultModel}
                onChange={(e) => handleLMStudioChange('defaultModel', e.target.value)}
                placeholder="Leave empty to auto-select"
                className="flex-1 px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-slack-text focus:outline-none focus:ring-2 focus:ring-slack-accent"
              />
            )}
            <button
              onClick={async () => {
                try {
                  const models = await fetchLMStudioModels();
                  setLMStudioForm(prev => ({ ...prev, availableModels: models }));
                } catch (error) {
                  console.error('Failed to fetch LM Studio models:', error);
                }
              }}
              className="px-3 py-2 text-sm bg-blue-500 text-white rounded hover:bg-blue-600 transition-colors whitespace-nowrap"
              title="Fetch available models from LM Studio"
            >
              Refresh
            </button>
          </div>
          <p className="text-xs text-slack-textMuted mt-1">
            {lmstudioForm.availableModels.length > 0
              ? `${lmstudioForm.availableModels.length} models available`
              : 'Click Refresh to load models from LM Studio'}
          </p>
        </div>

        <button
          onClick={saveLMStudioSettings}
          className="w-full px-4 py-2 bg-slack-accent text-white rounded hover:bg-slack-accentHover transition-colors"
        >
          Save LM Studio Settings
        </button>
      </div>
    </div>
{/* Global Provider Toggle */}
    <div className="border border-slack-border rounded-lg p-6">
      <h3 className="text-lg font-semibold text-slack-text mb-4">Global Provider Settings</h3>
      <div className="space-y-4">
        {testResults.providerSwitch && (
          <div className={`p-3 rounded text-sm ${
            testResults.providerSwitch.success 
              ? 'bg-green-100 text-green-800 border border-green-200' 
              : 'bg-red-100 text-red-800 border border-red-200'
          }`}>
            {testResults.providerSwitch.message}
          </div>
        )}
        <div className="flex items-center justify-between p-4 bg-slack-bgHover rounded">
          <div>
            <h4 className="font-medium text-slack-text">Switch All Agents</h4>
            <p className="text-sm text-slack-textMuted">
              Change all agents to use the same AI provider
            </p>
          </div>
          <div className="flex gap-2 flex-wrap">
            <button
              onClick={handleSwitchAllToClaude}
              disabled={isSwitching}
              className={`px-3 py-1 text-sm bg-purple-500 text-white rounded hover:bg-purple-600 transition-colors ${
                isSwitching ? 'opacity-50 cursor-not-allowed' : ''
              }`}
            >
              🧠 All to Claude
            </button>
            <button
              onClick={handleSwitchAllToOllama}
              disabled={isSwitching}
              className={`px-3 py-1 text-sm bg-blue-500 text-white rounded hover:bg-blue-600 transition-colors ${
                isSwitching ? 'opacity-50 cursor-not-allowed' : ''
              }`}
            >
              🤖 All to Ollama
            </button>
            <button
              onClick={handleSwitchAllToLMStudio}
              disabled={isSwitching}
              className={`px-3 py-1 text-sm bg-green-500 text-white rounded hover:bg-green-600 transition-colors ${
                isSwitching ? 'opacity-50 cursor-not-allowed' : ''
              }`}
            >
              🎨 All to LM Studio
            </button>
          </div>
        </div>
        {isSwitching && (
          <div className="flex items-center gap-2 text-sm text-slack-textMuted">
            <div className="w-4 h-4 border-2 border-slack-textMuted border-t-transparent rounded-full animate-spin" />
            <span>Switching providers...</span>
          </div>
        )}
      </div>
    </div>
    </div>
  );
}
