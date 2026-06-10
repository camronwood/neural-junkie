import { useState, useEffect } from 'react';
import { shallow } from 'zustand/shallow';
import { useSettingsStore } from '../../stores/settingsStore';
import { useChatStore } from '../../stores/chatStore';
import { usePacksStore } from '../../stores/packsStore';
import { PACK_CAP } from '../../stores/packCapabilities';
import { ChatAPI, type UserLearning } from '../../api/chatAPI';
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
import { open } from '@tauri-apps/api/dialog';
import { mergeSettingsPut, type SettingsTabProps } from './settingsShared';

export function AIProvidersSettingsTab({ hubHttp, isActive }: SettingsTabProps) {
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
  const hasPersonalLearning = usePacksStore((s) => s.hasCapability(PACK_CAP.PERSONAL_LEARNING));
  const hasLoRATraining = usePacksStore((s) => s.hasCapability(PACK_CAP.LORA_TRAINING));

    const [ollamaForm, setOllamaForm] = useState<OllamaSettings>(integrations.ollama);
    const [hardwareSnapshot, setHardwareSnapshot] = useState<HardwareSnapshot | null>(null);
    const [defaultModelLookup, setDefaultModelLookup] = useState<ModelLookup | null>(null);
    const [lmstudioForm, setLMStudioForm] = useState<LMStudioSettings>(integrations.lmstudio);
    const [testResults, setTestResults] = useState<Record<string, { success: boolean; message: string }>>({});
    const [isSwitching, setIsSwitching] = useState(false);
    const [collabSmartRouting, setCollabSmartRouting] = useState(false);
    const [collabPlanningProviderId, setCollabPlanningProviderId] = useState('');
    const [configuredProviders, setConfiguredProviders] = useState<Array<{ id: string; name: string }>>([]);
    const [implRoutingEnabled, setImplRoutingEnabled] = useState(true);
    const [implRoutingEnabledPersisted, setImplRoutingEnabledPersisted] = useState(true);
    const [implLocalToolModel, setImplLocalToolModel] = useState('qwen2.5-coder:7b');
    const [implLocalToolModelPersisted, setImplLocalToolModelPersisted] = useState('qwen2.5-coder:7b');
    const [collabAutoApproveDeliverables, setCollabAutoApproveDeliverables] = useState(true);
    const [collabRoutingSaving, setCollabRoutingSaving] = useState(false);
    const [collabRoutingErr, setCollabRoutingErr] = useState<string | null>(null);
    const [delegationEnabled, setDelegationEnabled] = useState(false);
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
    const [delegationSaving, setDelegationSaving] = useState(false);
    const [collabAssetsRoot, setCollabAssetsRoot] = useState('');
    const [collabAssetsPersisted, setCollabAssetsPersisted] = useState('');
    const [collabAssetsSaving, setCollabAssetsSaving] = useState(false);
    const [collabAssetsErr, setCollabAssetsErr] = useState<string | null>(null);
    const [collabAssetsOk, setCollabAssetsOk] = useState<string | null>(null);
    const [specialistModelsAdvancedOpen, setSpecialistModelsAdvancedOpen] = useState(false);
    const [personalLearningEnabled, setPersonalLearningEnabled] = useState(false);
    const [personalLearningSuggestEnabled, setPersonalLearningSuggestEnabled] = useState(false);
    const [conversationMemoryEnabled, setConversationMemoryEnabled] = useState(true);
    const [conversationMemorySaving, setConversationMemorySaving] = useState(false);
    const [personalLearningSaving, setPersonalLearningSaving] = useState(false);
    const [personalLearningsOpen, setPersonalLearningsOpen] = useState(false);
    const [allLearnings, setAllLearnings] = useState<UserLearning[]>([]);
    const [allLearningsLoading, setAllLearningsLoading] = useState(false);
    const [allLearningsErr, setAllLearningsErr] = useState<string | null>(null);

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
      setCollabRoutingErr(null);
      (async () => {
        try {
          const r = await fetch(`${hubHttp}/api/settings`);
          if (!r.ok) {
            throw new Error(await r.text());
          }
          const cfg = await r.json();
          if (!cancelled) {
            setCollabSmartRouting(!!cfg.collaboration?.smart_routing_enabled);
            setCollabPlanningProviderId(
              typeof cfg.collaboration?.planning_provider_id === 'string'
                ? cfg.collaboration.planning_provider_id
                : ''
            );
            const provRows = Array.isArray(cfg.ai?.providers) ? cfg.ai.providers : [];
            setConfiguredProviders(
              provRows.map((p: { id?: string; name?: string }) => ({
                id: String(p.id ?? ''),
                name: String(p.name ?? p.id ?? ''),
              })).filter((p: { id: string }) => p.id)
            );
            setImplRoutingEnabled(cfg.implementation?.routing_enabled !== false);
            setImplRoutingEnabledPersisted(cfg.implementation?.routing_enabled !== false);
            const toolModel =
              typeof cfg.implementation?.local_tool_model === 'string' &&
              cfg.implementation.local_tool_model.trim()
                ? cfg.implementation.local_tool_model.trim()
                : 'qwen2.5-coder:7b';
            setImplLocalToolModel(toolModel);
            setImplLocalToolModelPersisted(toolModel);
            setCollabAutoApproveDeliverables(cfg.collaboration?.auto_approve_deliverables !== false);
            setDelegationEnabled(!!cfg.delegation?.enabled);
            const root =
              typeof cfg.collaboration?.assets_root === 'string' ? cfg.collaboration.assets_root : '';
            setCollabAssetsRoot(root);
            setCollabAssetsPersisted(root);
            setCollabAssetsOk(null);
            const hfTok = typeof cfg.hf?.token === 'string' ? cfg.hf.token : '';
            const redacted = hfTok.includes('...') || hfTok === '***';
            setHfHubToken(redacted ? '' : hfTok);
            setHfHubTokenPersisted(redacted ? '***' : hfTok);
            setPersonalLearningEnabled(!!cfg.features?.personal_learning_enabled);
            setPersonalLearningSuggestEnabled(!!cfg.features?.personal_learning_suggest_enabled);
            setConversationMemoryEnabled(cfg.features?.conversation_memory_enabled !== false);
          }
        } catch (e) {
          if (!cancelled) {
            setCollabRoutingErr(e instanceof Error ? e.message : String(e));
          }
        }
      })();
      return () => {
        cancelled = true;
      };
    }, [isActive, hubHttp]);
    useEffect(() => {
      if (!isActive || !personalLearningEnabled || !hasPersonalLearning) {
        return;
      }
      void refreshAllLearnings();
    }, [isActive, personalLearningEnabled, hasPersonalLearning, hubHttp]);
    // Auto-fetch available models when AI Providers tab is selected
    useEffect(() => {
      if (!isActive) return;
      let cancelled = false;

      const loadModels = async () => {
        try {
          const ollamaModels = await fetchOllamaModels();
          if (!cancelled) setOllamaForm(prev => ({ ...prev, availableModels: ollamaModels }));
        } catch { /* Ollama may not be running */ }

        try {
          const lmModels = await fetchLMStudioModels();
          if (!cancelled) setLMStudioForm(prev => ({ ...prev, availableModels: lmModels }));
        } catch { /* LM Studio may not be running */ }
      };

      loadModels();
      return () => { cancelled = true; };
    }, [isActive, fetchOllamaModels, fetchLMStudioModels]);

    useEffect(() => {
      if (!isActive) return;
      let cancelled = false;
      void fetchHardwareSnapshot(hubHttp).then((snap) => {
        if (!cancelled) setHardwareSnapshot(snap);
      });
      return () => {
        cancelled = true;
      };
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
      return () => {
        cancelled = true;
      };
    }, [isActive, hubHttp, ollamaForm.defaultModel]);

    const handleConversationMemoryToggle = async (enabled: boolean) => {
      setConversationMemorySaving(true);
      setCollabRoutingErr(null);
      try {
        await mergeSettingsPut(hubHttp, (cfg) => ({
          ...cfg,
          features: {
            ...(typeof cfg.features === 'object' && cfg.features ? cfg.features : {}),
            conversation_memory_enabled: enabled,
          },
        }));
        setConversationMemoryEnabled(enabled);
      } catch (e) {
        setCollabRoutingErr(e instanceof Error ? e.message : String(e));
      } finally {
        setConversationMemorySaving(false);
      }
    };

    const handlePersonalLearningToggle = async (enabled: boolean) => {
      setPersonalLearningSaving(true);
      setCollabRoutingErr(null);
      try {
        await mergeSettingsPut(hubHttp, (cfg) => ({
          ...cfg,
          features: {
            ...(typeof cfg.features === 'object' && cfg.features ? cfg.features : {}),
            personal_learning_enabled: enabled,
          },
        }));
        setPersonalLearningEnabled(enabled);
      } catch (e) {
        setCollabRoutingErr(e instanceof Error ? e.message : String(e));
      } finally {
        setPersonalLearningSaving(false);
      }
    };

    const refreshAllLearnings = async () => {
      if (!personalLearningEnabled || !hasPersonalLearning) return;
      setAllLearningsLoading(true);
      setAllLearningsErr(null);
      try {
        const api = new ChatAPI(hubHttp);
        const rows = await api.fetchLearnings();
        setAllLearnings(rows);
      } catch (e) {
        setAllLearningsErr(e instanceof Error ? e.message : String(e));
      } finally {
        setAllLearningsLoading(false);
      }
    };

    const handlePersonalLearningSuggestToggle = async (enabled: boolean) => {
      setPersonalLearningSaving(true);
      setCollabRoutingErr(null);
      try {
        await mergeSettingsPut(hubHttp, (cfg) => ({
          ...cfg,
          features: {
            ...(typeof cfg.features === 'object' && cfg.features ? cfg.features : {}),
            personal_learning_suggest_enabled: enabled,
          },
        }));
        setPersonalLearningSuggestEnabled(enabled);
      } catch (e) {
        setCollabRoutingErr(e instanceof Error ? e.message : String(e));
      } finally {
        setPersonalLearningSaving(false);
      }
    };

    const handleDelegationToggle = async (enabled: boolean) => {
      setDelegationSaving(true);
      setCollabRoutingErr(null);
      try {
        const r = await fetch(`${hubHttp}/api/settings`);
        if (!r.ok) {
          throw new Error(await r.text());
        }
        const cfg = await r.json();
        const next = {
          ...cfg,
          delegation: {
            ...(cfg.delegation ?? {}),
            enabled,
          },
        };
        const put = await fetch(`${hubHttp}/api/settings`, {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(next),
        });
        if (!put.ok) {
          throw new Error(await put.text());
        }
        setDelegationEnabled(enabled);
      } catch (e) {
        setCollabRoutingErr(e instanceof Error ? e.message : String(e));
      } finally {
        setDelegationSaving(false);
      }
    };

    const saveImplementationSettings = async () => {
      setCollabRoutingSaving(true);
      setCollabRoutingErr(null);
      try {
        const r = await fetch(`${hubHttp}/api/settings`);
        if (!r.ok) {
          throw new Error(await r.text());
        }
        const cfg = await r.json();
        const next = {
          ...cfg,
          implementation: {
            ...(cfg.implementation ?? {}),
            routing_enabled: implRoutingEnabled,
            local_tool_model: implLocalToolModel.trim() || 'qwen2.5-coder:7b',
          },
        };
        const put = await fetch(`${hubHttp}/api/settings`, {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(next),
        });
        if (!put.ok) {
          throw new Error(await put.text());
        }
        setImplLocalToolModelPersisted(implLocalToolModel.trim() || 'qwen2.5-coder:7b');
        setImplRoutingEnabledPersisted(implRoutingEnabled);
      } catch (e) {
        setCollabRoutingErr(e instanceof Error ? e.message : String(e));
      } finally {
        setCollabRoutingSaving(false);
      }
    };

    const handleCollabPlanningProviderChange = async (providerId: string) => {
      setCollabRoutingSaving(true);
      setCollabRoutingErr(null);
      try {
        const r = await fetch(`${hubHttp}/api/settings`);
        if (!r.ok) {
          throw new Error(await r.text());
        }
        const cfg = await r.json();
        const next = {
          ...cfg,
          collaboration: {
            ...(cfg.collaboration ?? {}),
            planning_provider_id: providerId.trim(),
          },
        };
        const put = await fetch(`${hubHttp}/api/settings`, {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(next),
        });
        if (!put.ok) {
          throw new Error(await put.text());
        }
        setCollabPlanningProviderId(providerId.trim());
      } catch (e) {
        setCollabRoutingErr(e instanceof Error ? e.message : String(e));
      } finally {
        setCollabRoutingSaving(false);
      }
    };

    const handleCollabSmartRoutingToggle = async (enabled: boolean) => {
      setCollabRoutingSaving(true);
      setCollabRoutingErr(null);
      try {
        const r = await fetch(`${hubHttp}/api/settings`);
        if (!r.ok) {
          throw new Error(await r.text());
        }
        const cfg = await r.json();
        const next = {
          ...cfg,
          collaboration: {
            ...(cfg.collaboration ?? {}),
            smart_routing_enabled: enabled,
          },
        };
        const put = await fetch(`${hubHttp}/api/settings`, {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(next),
        });
        if (!put.ok) {
          throw new Error(await put.text());
        }
        setCollabSmartRouting(enabled);
      } catch (e) {
        setCollabRoutingErr(e instanceof Error ? e.message : String(e));
      } finally {
        setCollabRoutingSaving(false);
      }
    };

    const handleCollabAutoApproveToggle = async (enabled: boolean) => {
      setCollabRoutingSaving(true);
      setCollabRoutingErr(null);
      try {
        const r = await fetch(`${hubHttp}/api/settings`);
        if (!r.ok) {
          throw new Error(await r.text());
        }
        const cfg = await r.json();
        const next = {
          ...cfg,
          collaboration: {
            ...(cfg.collaboration ?? {}),
            auto_approve_deliverables: enabled,
          },
        };
        const put = await fetch(`${hubHttp}/api/settings`, {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(next),
        });
        if (!put.ok) {
          throw new Error(await put.text());
        }
        setCollabAutoApproveDeliverables(enabled);
      } catch (e) {
        setCollabRoutingErr(e instanceof Error ? e.message : String(e));
      } finally {
        setCollabRoutingSaving(false);
      }
    };

    const saveHfHubToken = async () => {
      setHfTokenSaving(true);
      setHfTokenErr(null);
      setHfTokenOk(null);
      try {
        const trimmed = hfHubToken.trim();
        await mergeSettingsPut(hubHttp, (cfg) => ({
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
      const persistCollabAssetsRoot = async (path: string): Promise<boolean> => {
      setCollabAssetsSaving(true);
      setCollabAssetsErr(null);
      setCollabAssetsOk(null);
      try {
        const r = await fetch(`${hubHttp}/api/settings`);
        if (!r.ok) {
          throw new Error(await r.text());
        }
        const cfg = await r.json();
        const trimmed = path.trim();
        const next = {
          ...cfg,
          collaboration: {
            ...(cfg.collaboration ?? {}),
            assets_root: trimmed,
          },
        };
        const put = await fetch(`${hubHttp}/api/settings`, {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(next),
        });
        if (!put.ok) {
          throw new Error(await put.text());
        }
        setCollabAssetsRoot(trimmed);
        setCollabAssetsPersisted(trimmed);
        setCollabAssetsOk(
          trimmed
            ? 'Saved to hub. New collaborations will use this folder.'
            : 'Saved. New collaborations will use the default ~/.neural-junkie/collaborations.'
        );
        return true;
      } catch (e) {
        setCollabAssetsErr(e instanceof Error ? e.message : String(e));
        return false;
      } finally {
        setCollabAssetsSaving(false);
      }
    };

    const handleCollabAssetsRootSave = async () => {
      await persistCollabAssetsRoot(collabAssetsRoot);
    };

    const handleCollabAssetsRootBlur = () => {
      if (collabAssetsSaving) return;
      if (collabAssetsRoot.trim() === collabAssetsPersisted.trim()) return;
      void persistCollabAssetsRoot(collabAssetsRoot);
    };

    const handleBrowseCollabAssetsRoot = async () => {
      setCollabAssetsErr(null);
      setCollabAssetsOk(null);
      if (!(typeof window !== 'undefined' && (window as { __TAURI__?: unknown }).__TAURI__)) {
        setCollabAssetsErr('Folder picker requires the desktop app');
        return;
      }
      try {
        const selected = await open({
          directory: true,
          multiple: false,
          title: 'Collaboration output folder',
        });
        if (selected && typeof selected === 'string') {
          setCollabAssetsRoot(selected);
          await persistCollabAssetsRoot(selected);
        }
      } catch (e) {
        setCollabAssetsErr(e instanceof Error ? e.message : String(e));
      }
    };

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
    const saveConfiguredAgentModels = async () => {
      setAgentModelsSaving(true);
      setAgentModelsErr(null);
      setAgentModelsOk(null);
      try {
        await mergeSettingsPut(hubHttp, (cfg) => {
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
      } catch (e) {
        setAgentModelsErr(e instanceof Error ? e.message : String(e));
      } finally {
        setAgentModelsSaving(false);
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

    <div className="border border-slack-border rounded-lg p-6">
      <h3 className="text-lg font-semibold text-slack-text mb-2">Collaboration output folder</h3>
      <p className="text-sm text-slack-textMuted mb-4">
        When a plan is approved, each collaboration gets a sandbox at{' '}
        <code className="font-mono text-xs bg-slack-bgHover px-1 rounded">&lt;folder&gt;/&lt;collaboration-id&gt;/</code>.
        Leave empty to use{' '}
        <code className="font-mono text-xs bg-slack-bgHover px-1 rounded">~/.neural-junkie/collaborations</code>.
        <strong className="text-slack-text"> Browse saves immediately.</strong> Typed paths save when you click Save or leave the field.
        Hub env <code className="font-mono text-xs bg-slack-bgHover px-1 rounded">NEURAL_JUNKIE_COLLAB_ASSETS_DIR</code> overrides this if set at server start.
      </p>
      <div className="flex flex-col sm:flex-row gap-2 mb-3">
        <input
          type="text"
          value={collabAssetsRoot}
          onChange={(e) => {
            setCollabAssetsRoot(e.target.value);
            setCollabAssetsOk(null);
          }}
          onBlur={handleCollabAssetsRootBlur}
          placeholder="~/development/collab-output"
          disabled={collabAssetsSaving}
          className="flex-1 px-3 py-2 text-sm border border-slack-border rounded bg-slack-bg text-slack-text font-mono"
          spellCheck={false}
        />
        <button
          type="button"
          onClick={() => void handleBrowseCollabAssetsRoot()}
          disabled={collabAssetsSaving}
          className="px-3 py-2 text-sm border border-slack-border rounded text-slack-text hover:bg-slack-bgHover disabled:opacity-50"
        >
          Browse…
        </button>
        <button
          type="button"
          onClick={() => void handleCollabAssetsRootSave()}
          disabled={collabAssetsSaving}
          className="px-4 py-2 text-sm bg-slack-accent text-white rounded hover:bg-slack-accentHover disabled:opacity-50"
        >
          {collabAssetsSaving ? 'Saving…' : 'Save'}
        </button>
      </div>
      {collabAssetsErr && (
        <p className="text-sm text-red-600">{collabAssetsErr}</p>
      )}
      {collabAssetsOk && !collabAssetsErr && (
        <p className="text-sm text-green-600">{collabAssetsOk}</p>
      )}
      {!collabAssetsSaving &&
        !collabAssetsErr &&
        collabAssetsRoot.trim() !== collabAssetsPersisted.trim() && (
          <p className="text-sm text-amber-600">Unsaved changes — Save or tab out of the field.</p>
        )}
    </div>

    <div className="border border-slack-border rounded-lg p-6">
      <h3 className="text-lg font-semibold text-slack-text mb-2">Agent delegation</h3>
      <p className="text-sm text-slack-textMuted mb-4">
        When enabled, any in-process agent may consult other specialists via the hub (by relevance), then
        synthesize one reply. Applies to normal chat and DMs — not collaboration task orchestration.
      </p>
      <label className="flex items-center gap-3 cursor-pointer">
        <input
          type="checkbox"
          checked={delegationEnabled}
          disabled={delegationSaving}
          onChange={(e) => void handleDelegationToggle(e.target.checked)}
          className="rounded border-slack-border"
        />
        <span className="text-slack-text">Enable cross-agent delegation</span>
      </label>
    </div>

    <div className="border border-slack-border rounded-lg p-6">
      <h3 className="text-lg font-semibold text-slack-text mb-2">Collaboration planning provider</h3>
      <p className="text-sm text-slack-textMuted mb-2">
        Local Ollama models vary in plan quality. For harder collaborations, route <strong>planning discussion</strong>{' '}
        turns through a cloud or larger local provider. Execution tasks still use smart routing / agent defaults.
      </p>
      <p className="text-sm text-slack-textMuted mb-4">
        Recommended: 14B+ local or a configured Claude/OpenAI provider. See{' '}
        <a href="https://github.com/camronwood/neural-junkie/blob/main/docs/HARDWARE.md" className="text-slack-accent hover:underline" target="_blank" rel="noreferrer">
          HARDWARE.md
        </a>{' '}
        for RAM tiers.
      </p>
      <label className="block text-sm text-slack-text mb-1" htmlFor="collab-planning-provider">
        Planning provider
      </label>
      <select
        id="collab-planning-provider"
        className="w-full max-w-md rounded border border-slack-border bg-slack-bg px-3 py-2 text-sm text-slack-text mb-2"
        value={collabPlanningProviderId}
        disabled={collabRoutingSaving}
        onChange={(e) => void handleCollabPlanningProviderChange(e.target.value)}
      >
        <option value="">Use each agent&apos;s default</option>
        {configuredProviders.map((p) => (
          <option key={p.id} value={p.id}>
            {p.name} ({p.id})
          </option>
        ))}
      </select>
    </div>

    <div className="border border-slack-border rounded-lg p-6">
      <h3 className="text-lg font-semibold text-slack-text mb-2">Collaboration smart routing</h3>
      <p className="text-sm text-slack-textMuted mb-4">
        When enabled, the hub picks a configured AI provider for each <strong>collaboration execution task</strong>{' '}
        (assigned task messages after the plan is approved). Normal chat and DMs still use each agent's configured provider.
      </p>
      <label className="flex items-center gap-3 cursor-pointer">
        <input
          type="checkbox"
          checked={collabSmartRouting}
          disabled={collabRoutingSaving}
          onChange={(e) => void handleCollabSmartRoutingToggle(e.target.checked)}
          className="rounded border-slack-border"
        />
        <span className="text-slack-text">Enable smart routing for collaboration tasks</span>
      </label>
      <label className="flex items-center gap-3 cursor-pointer mt-4">
        <input
          type="checkbox"
          checked={collabAutoApproveDeliverables}
          disabled={collabRoutingSaving}
          onChange={(e) => void handleCollabAutoApproveToggle(e.target.checked)}
          className="rounded border-slack-border"
        />
        <span className="text-slack-text">Auto-approve deliverables under collabs/&lt;id&gt;/</span>
      </label>
      {collabRoutingErr && (
        <p className="text-sm text-red-600 mt-2">{collabRoutingErr}</p>
      )}
    </div>

    <div className="border border-slack-border rounded-lg p-6">
      <h3 className="text-lg font-semibold text-slack-text mb-2">Implementation sessions</h3>
      <p className="text-sm text-slack-textMuted mb-4">
        IDE Agent mode runs multi-step implementation sessions (read → edit → verify). Local Ollama is
        preferred; fallbacks use configured cloud providers when local tool calling is unavailable.
      </p>
      <label className="flex items-center gap-3 cursor-pointer">
        <input
          type="checkbox"
          checked={implRoutingEnabled}
          disabled={collabRoutingSaving}
          onChange={(e) => setImplRoutingEnabled(e.target.checked)}
          className="rounded border-slack-border"
        />
        <span className="text-slack-text">Enable local-first implementation routing</span>
      </label>
      <div className="mt-4">
        <label className="block text-sm text-slack-textMuted mb-1">Implementation tool model (Ollama tag)</label>
        <input
          type="text"
          value={implLocalToolModel}
          disabled={collabRoutingSaving}
          onChange={(e) => setImplLocalToolModel(e.target.value)}
          className="w-full max-w-md px-3 py-2 rounded border border-slack-border bg-slack-bg text-slack-text text-sm"
          placeholder="qwen2.5-coder:7b"
        />
      </div>
      <button
        type="button"
        disabled={
          collabRoutingSaving ||
          (implRoutingEnabled === implRoutingEnabledPersisted &&
            (implLocalToolModel.trim() || 'qwen2.5-coder:7b') === implLocalToolModelPersisted)
        }
        onClick={() => void saveImplementationSettings()}
        className="mt-4 px-4 py-2 text-sm rounded bg-slack-accent text-white disabled:opacity-50"
      >
        Save implementation settings
      </button>
    </div>

    <div className="border border-slack-border rounded-lg p-6">
      <h3 className="text-lg font-semibold text-slack-text mb-2">Conversation memory</h3>
      <p className="text-sm text-slack-textMuted mb-4">
        Index persisted chat and collab artifacts locally, then retrieve relevant past context when you ask
        about earlier decisions (requires Ollama embed model).
      </p>
      <label className="flex items-center gap-3 cursor-pointer">
        <input
          type="checkbox"
          checked={conversationMemoryEnabled}
          disabled={conversationMemorySaving}
          onChange={(e) => void handleConversationMemoryToggle(e.target.checked)}
          className="rounded border-slack-border"
        />
        <span className="text-slack-text">Retrieve relevant past messages</span>
      </label>
    </div>

    {hasPersonalLearning && (
      <div className="border border-slack-border rounded-lg p-6">
        <h3 className="text-lg font-semibold text-slack-text mb-2">Personal learning</h3>
        <p className="text-sm text-slack-textMuted mb-4">
          Agents will ask before saving anything — each expert keeps its own learnings.
        </p>
        <label className="flex items-center gap-3 cursor-pointer">
          <input
            type="checkbox"
            checked={personalLearningEnabled}
            disabled={personalLearningSaving}
            onChange={(e) => void handlePersonalLearningToggle(e.target.checked)}
            className="rounded border-slack-border"
          />
          <span className="text-slack-text">Enable personal learning for experts</span>
        </label>

        {personalLearningEnabled && (
          <label className="flex items-center gap-3 cursor-pointer mt-3">
            <input
              type="checkbox"
              checked={personalLearningSuggestEnabled}
              disabled={personalLearningSaving}
              onChange={(e) => void handlePersonalLearningSuggestToggle(e.target.checked)}
              className="rounded border-slack-border"
            />
            <span className="text-slack-text">Allow agents to suggest learnings (still requires your approval)</span>
          </label>
        )}

        {personalLearningEnabled && (
          <div className="flex gap-2 mt-4">
            <button
              type="button"
              className="px-3 py-1.5 text-sm border border-slack-border rounded hover:bg-slack-bgHover"
              onClick={async () => {
                try {
                  const api = new ChatAPI(hubHttp);
                  const bundle = await api.exportLearnings();
                  const blob = new Blob([JSON.stringify(bundle, null, 2)], { type: 'application/json' });
                  const url = URL.createObjectURL(blob);
                  const a = document.createElement('a');
                  a.href = url;
                  a.download = 'neural-junkie-learnings.json';
                  a.click();
                  URL.revokeObjectURL(url);
                } catch (e) {
                  setAllLearningsErr(e instanceof Error ? e.message : String(e));
                }
              }}
            >
              Export learnings
            </button>
            <label className="px-3 py-1.5 text-sm border border-slack-border rounded hover:bg-slack-bgHover cursor-pointer">
              Import learnings
              <input
                type="file"
                accept="application/json,.json"
                className="hidden"
                onChange={async (e) => {
                  const file = e.target.files?.[0];
                  if (!file) return;
                  try {
                    const text = await file.text();
                    const bundle = JSON.parse(text) as { entries: UserLearning[] };
                    const api = new ChatAPI(hubHttp);
                    await api.importLearnings(bundle);
                    const rows = await api.fetchLearnings();
                    setAllLearnings(rows);
                    setAllLearningsErr(null);
                  } catch (err) {
                    setAllLearningsErr(err instanceof Error ? err.message : 'Import failed');
                  } finally {
                    e.target.value = '';
                  }
                }}
              />
            </label>
          </div>
        )}

        {personalLearningEnabled && (
          <details
            open={personalLearningsOpen}
            onToggle={(e) => setPersonalLearningsOpen(e.currentTarget.open)}
            className="mt-4"
          >
            <summary className="cursor-pointer text-sm font-medium text-slack-text">
              Saved learnings ({allLearnings.length})
            </summary>
            {allLearningsLoading && (
              <p className="text-sm text-slack-textMuted mt-2">Loading…</p>
            )}
            {allLearningsErr && (
              <p className="text-sm text-red-600 mt-2">{allLearningsErr}</p>
            )}
            {!allLearningsLoading && allLearnings.length === 0 && (
              <p className="text-sm text-slack-textMuted mt-2">No learnings saved yet.</p>
            )}
            {allLearnings.length > 0 && (
              <div className="mt-2 space-y-4 max-h-64 overflow-y-auto">
                {(['global', 'agent', 'collaboration'] as const).map((scopeKey) => {
                  const rows = allLearnings.filter((e) => (e.scope || 'agent') === scopeKey);
                  if (rows.length === 0) return null;
                  const title =
                    scopeKey === 'global'
                      ? 'All experts'
                      : scopeKey === 'collaboration'
                        ? 'By collaboration'
                        : 'By expert';
                  return (
                    <div key={scopeKey}>
                      <p className="text-xs font-semibold text-slack-textMuted uppercase mb-1">{title}</p>
                      <ul className="space-y-1">
                        {rows.map((e) => (
                          <li
                            key={e.id}
                            className="flex justify-between gap-2 text-sm text-slack-textMuted"
                          >
                            <span>
                              {scopeKey === 'agent' && (
                                <span className="text-slack-text mr-1">{e.agent_name || e.agent_id}:</span>
                              )}
                              [{e.category}] {e.content}
                            </span>
                            <button
                              type="button"
                              className="text-red-500 hover:text-red-400 shrink-0"
                              onClick={async () => {
                                try {
                                  const api = new ChatAPI(hubHttp);
                                  await api.deleteLearning(e.id);
                                  setAllLearnings((prev) => prev.filter((x) => x.id !== e.id));
                                } catch (err) {
                                  setAllLearningsErr(
                                    err instanceof Error ? err.message : 'Forget failed'
                                  );
                                }
                              }}
                            >
                              Forget
                            </button>
                          </li>
                        ))}
                      </ul>
                    </div>
                  );
                })}
              </div>
            )}
            {hasLoRATraining && (
              <p className="text-xs text-slack-textMuted mt-3">
                When an expert has 10+ chat turns, open agent info (ℹ️) → Train LoRA to bake history into weights.
              </p>
            )}
          </details>
        )}
      </div>
    )}

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
                placeholder={ollamaForm.defaultModel || 'qwen2.5-coder:14b'}
                className="flex-1 px-3 py-2 text-sm border border-slack-border rounded bg-slack-bg text-slack-text font-mono"
              />
            </li>
          ))}
        </ul>
      )}
      <datalist id="nj-ollama-model-options">
        {ollamaForm.availableModels.map((m) => (
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

    {/* CLI agent install & auth */}
    <div className="border border-slack-border rounded-lg p-6">
      <CLIAgentsManager serverAddr={hubHttp} />
    </div>

    {/* Dynamic Provider Registry */}
    <div className="border border-slack-border rounded-lg p-6">
      <ProviderManager serverAddr={hubHttp} />
    </div>

    <div className="border border-slack-border rounded-lg p-4 bg-slack-bgHover/30">
      <p className="text-sm text-slack-text">
        <strong className="font-medium">Model library</strong> — browse, download, and install Ollama and
        Hugging Face models from the chat toolbar (amber icon),{' '}
        <kbd className="font-mono text-xs px-1 rounded bg-slack-bgHover">⇧⌘M</kbd> /{' '}
        <kbd className="font-mono text-xs px-1 rounded bg-slack-bgHover">Ctrl+Shift+M</kbd>, or{' '}
        <code className="font-mono text-xs bg-slack-bgHover px-1 rounded">/nj-open-model-library</code>.
      </p>
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
