import { useState, useEffect } from 'react';
import { shallow } from 'zustand/shallow';
import { useSettingsStore, type FontSizeScope } from '../stores/settingsStore';
import { useChatStore } from '../stores/chatStore';
import { APP_INFO, TECH_STACK, getAppVersion } from '../utils/appInfo';
import type { AnthropicSettings, GitHubSettings, ConfluenceSettings, OllamaSettings, LMStudioSettings, GoogleMeetNotesSettings, GoogleMeetNotesStatus } from '../types/protocol';
import { ChatAPI, type PackStatus } from '../api/chatAPI';
import { patchRevealActiveAgentsInSidebar } from '../utils/sidebarVisibility';
import { agentSidebarHideKey } from '../utils/dmChannelDisplay';
import { ProviderManager } from './ProviderManager';
import { getHubBaseURL, getHubWebSocketURL } from '../config/hubUrl';
import { open } from '@tauri-apps/api/dialog';

interface SettingsModalProps {
  isOpen: boolean;
  onClose: () => void;
}

export function SettingsModal({ isOpen, onClose }: SettingsModalProps) {
  const { 
    settings, 
    integrations,
    layoutSettings,
    updateFontSize, 
    updateFontSizeScope, 
    loadSettings,
    updateSettings,
    loadIntegrations,
    loadLayoutSettings,
    updateLayoutSettings,
    updateAnthropicSettings,
    updateGitHubSettings,
    updateConfluenceSettings,
    updateGoogleMeetNotesSettings,
    updateOllamaSettings,
    updateLMStudioSettings,
    clearIntegrationSettings,
    testAnthropicConnection,
    testGitHubConnection,
    testConfluenceConnection,
    testOllamaConnection,
    testLMStudioConnection,
    fetchOllamaModels,
    fetchLMStudioModels
  } = useSettingsStore();
  const { switchAllAgentProviders, serverAddr: chatServerAddr, channels, agents, setAgents, setChannels } = useChatStore(
    (s) => ({
      switchAllAgentProviders: s.switchAllAgentProviders,
      serverAddr: s.serverAddr,
      channels: s.channels,
      agents: s.agents,
      setAgents: s.setAgents,
      setChannels: s.setChannels,
    }),
    shallow
  );
  const hubHttp =
    chatServerAddr.startsWith('http') ? chatServerAddr : `http://${chatServerAddr}`;
  const [activeTab, setActiveTab] = useState<
    'appearance' | 'layout' | 'chat' | 'integrations' | 'ai-providers' | 'domain-packs' | 'about'
  >('appearance');
  const [appVersion, setAppVersion] = useState<string>('1.0.0');
  
  // Integration form states
  const [anthropicForm, setAnthropicForm] = useState<AnthropicSettings>(integrations.anthropic);
  const [githubForm, setGitHubForm] = useState<GitHubSettings>(integrations.github);
  const [confluenceForm, setConfluenceForm] = useState<ConfluenceSettings>(integrations.confluence);
  const [googleOAuthForm, setGoogleOAuthForm] = useState<GoogleMeetNotesSettings>(integrations.googleMeetNotes);
  const [googleOAuthSecretSet, setGoogleOAuthSecretSet] = useState(false);
  const [ollamaForm, setOllamaForm] = useState<OllamaSettings>(integrations.ollama);
  const [lmstudioForm, setLMStudioForm] = useState<LMStudioSettings>(integrations.lmstudio);
  const [showPasswords, setShowPasswords] = useState<Record<string, boolean>>({});
  const [testResults, setTestResults] = useState<Record<string, { success: boolean; message: string }>>({});
  const [isSwitching, setIsSwitching] = useState(false);
  const [collabSmartRouting, setCollabSmartRouting] = useState(false);
  const [collabRoutingSaving, setCollabRoutingSaving] = useState(false);
  const [collabRoutingErr, setCollabRoutingErr] = useState<string | null>(null);
  const [collabAssetsRoot, setCollabAssetsRoot] = useState('');
  const [collabAssetsPersisted, setCollabAssetsPersisted] = useState('');
  const [collabAssetsSaving, setCollabAssetsSaving] = useState(false);
  const [collabAssetsErr, setCollabAssetsErr] = useState<string | null>(null);
  const [collabAssetsOk, setCollabAssetsOk] = useState<string | null>(null);
  const [googleMeetNotes, setGoogleMeetNotes] = useState<GoogleMeetNotesStatus | null>(null);
  const [googleMeetNotesLoading, setGoogleMeetNotesLoading] = useState(false);
  const [googleMeetNotesBusy, setGoogleMeetNotesBusy] = useState(false);
  const [domainPacks, setDomainPacks] = useState<PackStatus[]>([]);
  const [packsLoading, setPacksLoading] = useState(false);
  const [packsSaving, setPacksSaving] = useState<string | null>(null);
  const [packsErr, setPacksErr] = useState<string | null>(null);
  const [hfHubToken, setHfHubToken] = useState('');
  const [hfHubTokenPersisted, setHfHubTokenPersisted] = useState('');
  const [hfTokenSaving, setHfTokenSaving] = useState(false);
  const [hfTokenErr, setHfTokenErr] = useState<string | null>(null);
  const [hfTokenOk, setHfTokenOk] = useState<string | null>(null);
  const [mcpEnabled, setMcpEnabled] = useState(true);
  const [bioMaxFold, setBioMaxFold] = useState('400');
  const [bioMaxAnalyze, setBioMaxAnalyze] = useState('10000');
  const [bioEsmfoldModel, setBioEsmfoldModel] = useState('facebook/esmfold_v1');
  const [bioArtifactsDir, setBioArtifactsDir] = useState('');
  const [bioSettingsSaving, setBioSettingsSaving] = useState(false);
  const [bioSettingsErr, setBioSettingsErr] = useState<string | null>(null);
  const [bioSettingsOk, setBioSettingsOk] = useState<string | null>(null);

  const refreshDomainPacks = async () => {
    setPacksLoading(true);
    setPacksErr(null);
    try {
      const api = new ChatAPI(hubHttp);
      const rows = await api.fetchPacks();
      setDomainPacks(rows);
    } catch (e) {
      setPacksErr(e instanceof Error ? e.message : String(e));
    } finally {
      setPacksLoading(false);
    }
  };

  const mergeSettingsPut = async (patch: (cfg: Record<string, unknown>) => Record<string, unknown>) => {
    const r = await fetch(`${hubHttp}/api/settings`);
    if (!r.ok) {
      throw new Error(await r.text());
    }
    const cfg = (await r.json()) as Record<string, unknown>;
    const put = await fetch(`${hubHttp}/api/settings`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(patch(cfg)),
    });
    if (!put.ok) {
      throw new Error(await put.text());
    }
  };

  const handlePackToggle = async (packId: string, enabled: boolean) => {
    setPacksSaving(packId);
    setPacksErr(null);
    try {
      const api = new ChatAPI(hubHttp);
      const rows = await api.setPackEnabled(packId, enabled);
      setDomainPacks(rows);
      const [agentList, channelList] = await Promise.all([
        api.fetchAgents(),
        api.fetchChannels(),
      ]);
      setAgents(agentList);
      setChannels(channelList);
      if (enabled) {
        if (!layoutSettings.sidebarAgentsVisible) {
          await updateLayoutSettings({ sidebarAgentsVisible: true });
        }
        const revealPatch = patchRevealActiveAgentsInSidebar(
          useSettingsStore.getState().settings,
          agentList,
          channelList
        );
        if (revealPatch) {
          await updateSettings(revealPatch);
        }
      }
    } catch (e) {
      setPacksErr(e instanceof Error ? e.message : String(e));
    } finally {
      setPacksSaving(null);
    }
  };

  const refreshGoogleMeetNotesStatus = async () => {
    setGoogleMeetNotesLoading(true);
    try {
      const api = new ChatAPI(hubHttp);
      const [status, appConfig] = await Promise.all([
        api.getGoogleMeetNotesStatus(),
        api.getGoogleMeetNotesAppConfig().catch(() => null),
      ]);
      setGoogleMeetNotes(status);
      if (appConfig) {
        setGoogleOAuthForm((prev) => ({
          ...prev,
          clientId: appConfig.client_id || prev.clientId,
          redirectUrl: appConfig.redirect_url || prev.redirectUrl,
        }));
        setGoogleOAuthSecretSet(appConfig.secret_set);
      }
    } catch (e) {
      setGoogleMeetNotes({
        connected: false,
        oauth_configured: false,
      });
      setTestResults((prev) => ({
        ...prev,
        googleMeetNotes: {
          success: false,
          message: e instanceof Error ? e.message : 'Failed to load status',
        },
      }));
    } finally {
      setGoogleMeetNotesLoading(false);
    }
  };

  // Load settings when modal opens
  useEffect(() => {
    if (isOpen) {
      loadSettings();
      loadIntegrations();
      loadLayoutSettings();
      getAppVersion().then(setAppVersion);
    }
  }, [isOpen, loadSettings, loadIntegrations, loadLayoutSettings]);

  // Update form states when integrations change
  useEffect(() => {
    setAnthropicForm(integrations.anthropic);
    setGitHubForm(integrations.github);
    setConfluenceForm(integrations.confluence);
    setGoogleOAuthForm(integrations.googleMeetNotes);
    setOllamaForm(integrations.ollama);
    setLMStudioForm(integrations.lmstudio);
  }, [integrations]);

  useEffect(() => {
    if (!isOpen || activeTab !== 'integrations') return;
    void refreshGoogleMeetNotesStatus();
  }, [isOpen, activeTab, hubHttp]);

  const saveGoogleOAuthSettings = async () => {
    setGoogleMeetNotesBusy(true);
    try {
      const api = new ChatAPI(hubHttp);
      await api.saveGoogleMeetNotesAppConfig(
        googleOAuthForm.clientId,
        googleOAuthForm.clientSecret,
        googleOAuthForm.redirectUrl
      );
      await updateGoogleMeetNotesSettings(googleOAuthForm);
      setGoogleOAuthSecretSet(true);
      setGoogleOAuthForm((prev) => ({ ...prev, clientSecret: '' }));
      await refreshGoogleMeetNotesStatus();
      setTestResults((prev) => ({
        ...prev,
        googleMeetNotes: {
          success: true,
          message: 'OAuth app credentials saved on the hub.',
        },
      }));
    } catch (e) {
      setTestResults((prev) => ({
        ...prev,
        googleMeetNotes: {
          success: false,
          message: e instanceof Error ? e.message : 'Failed to save OAuth credentials',
        },
      }));
    } finally {
      setGoogleMeetNotesBusy(false);
    }
  };

  const connectGoogleMeetNotes = async () => {
    setGoogleMeetNotesBusy(true);
    try {
      const api = new ChatAPI(hubHttp);
      const url = await api.getGoogleMeetNotesAuthURL();
      openLink(url);
      setTestResults((prev) => ({
        ...prev,
        googleMeetNotes: {
          success: true,
          message: 'Complete sign-in in your browser, then refresh status.',
        },
      }));
    } catch (e) {
      setTestResults((prev) => ({
        ...prev,
        googleMeetNotes: {
          success: false,
          message: e instanceof Error ? e.message : 'Connect failed',
        },
      }));
    } finally {
      setGoogleMeetNotesBusy(false);
    }
  };

  const disconnectGoogleMeetNotes = async () => {
    setGoogleMeetNotesBusy(true);
    try {
      const api = new ChatAPI(hubHttp);
      await api.disconnectGoogleMeetNotes();
      await refreshGoogleMeetNotesStatus();
      setTestResults((prev) => ({
        ...prev,
        googleMeetNotes: { success: true, message: 'Disconnected from Google.' },
      }));
    } catch (e) {
      setTestResults((prev) => ({
        ...prev,
        googleMeetNotes: {
          success: false,
          message: e instanceof Error ? e.message : 'Disconnect failed',
        },
      }));
    } finally {
      setGoogleMeetNotesBusy(false);
    }
  };

  const syncGoogleMeetNotesNow = async () => {
    setGoogleMeetNotesBusy(true);
    try {
      const api = new ChatAPI(hubHttp);
      const n = await api.syncGoogleMeetNotes();
      await refreshGoogleMeetNotesStatus();
      setTestResults((prev) => ({
        ...prev,
        googleMeetNotes: {
          success: true,
          message: `Synced ${n} meeting note(s).`,
        },
      }));
    } catch (e) {
      setTestResults((prev) => ({
        ...prev,
        googleMeetNotes: {
          success: false,
          message: e instanceof Error ? e.message : 'Sync failed',
        },
      }));
    } finally {
      setGoogleMeetNotesBusy(false);
    }
  };

  useEffect(() => {
    if (!isOpen || activeTab !== 'domain-packs') return;
    void refreshDomainPacks();
  }, [isOpen, activeTab, hubHttp]);

  useEffect(() => {
    if (!isOpen || (activeTab !== 'ai-providers' && activeTab !== 'domain-packs')) return;
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
          const root =
            typeof cfg.collaboration?.assets_root === 'string' ? cfg.collaboration.assets_root : '';
          setCollabAssetsRoot(root);
          setCollabAssetsPersisted(root);
          setCollabAssetsOk(null);
          const hfTok = typeof cfg.hf?.token === 'string' ? cfg.hf.token : '';
          const redacted = hfTok.includes('...') || hfTok === '***';
          setHfHubToken(redacted ? '' : hfTok);
          setHfHubTokenPersisted(redacted ? '***' : hfTok);
          setMcpEnabled(cfg.mcp?.enabled !== false);
          const bio = cfg.mcp?.biology ?? {};
          setBioMaxFold(String(bio.max_fold_length || 400));
          setBioMaxAnalyze(String(bio.max_analyze_length || 10000));
          setBioEsmfoldModel(bio.esmfold_model || 'facebook/esmfold_v1');
          setBioArtifactsDir(bio.artifacts_dir || '');
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
  }, [isOpen, activeTab, hubHttp]);

  // Auto-fetch available models when AI Providers tab is selected
  useEffect(() => {
    if (activeTab !== 'ai-providers') return;
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
  }, [activeTab, fetchOllamaModels, fetchLMStudioModels]);

  // Handle escape key
  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && isOpen) {
        onClose();
      }
    };

    if (isOpen) {
      document.addEventListener('keydown', handleEscape);
      return () => document.removeEventListener('keydown', handleEscape);
    }
  }, [isOpen, onClose]);

  if (!isOpen) return null;

  const handleFontSizeChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    updateFontSize(parseInt(e.target.value));
  };

  const handleScopeChange = (scope: FontSizeScope) => {
    updateFontSizeScope(scope);
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

  const saveHfHubToken = async () => {
    setHfTokenSaving(true);
    setHfTokenErr(null);
    setHfTokenOk(null);
    try {
      const trimmed = hfHubToken.trim();
      await mergeSettingsPut((cfg) => ({
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

  const saveBioMcpSettings = async () => {
    setBioSettingsSaving(true);
    setBioSettingsErr(null);
    setBioSettingsOk(null);
    try {
      const maxFold = parseInt(bioMaxFold, 10);
      const maxAnalyze = parseInt(bioMaxAnalyze, 10);
      if (!Number.isFinite(maxFold) || maxFold <= 0 || !Number.isFinite(maxAnalyze) || maxAnalyze <= 0) {
        throw new Error('Max lengths must be positive integers');
      }
      await mergeSettingsPut((cfg) => ({
        ...cfg,
        mcp: {
          ...(cfg.mcp as object | undefined),
          enabled: mcpEnabled,
          biology: {
            esmfold_model: bioEsmfoldModel.trim() || 'facebook/esmfold_v1',
            max_fold_length: maxFold,
            max_analyze_length: maxAnalyze,
            artifacts_dir: bioArtifactsDir.trim(),
          },
        },
      }));
      setBioSettingsOk('Life sciences tool settings saved. Restart BiologyExpert if it is already running.');
    } catch (e) {
      setBioSettingsErr(e instanceof Error ? e.message : String(e));
    } finally {
      setBioSettingsSaving(false);
    }
  };

  const handleMcpMasterToggle = async (enabled: boolean) => {
    setMcpEnabled(enabled);
    try {
      await mergeSettingsPut((cfg) => ({
        ...cfg,
        mcp: { ...(cfg.mcp as object | undefined), enabled },
      }));
    } catch (e) {
      setMcpEnabled(!enabled);
      setBioSettingsErr(e instanceof Error ? e.message : String(e));
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

  const openLink = (url: string) => {
    if (typeof window !== 'undefined' && (window as any).__TAURI__) {
      import('@tauri-apps/api/shell').then(({ open }) => open(url));
    } else {
      window.open(url, '_blank');
    }
  };

  // Integration handlers
  const handleAnthropicChange = (field: keyof AnthropicSettings, value: string | boolean) => {
    setAnthropicForm(prev => ({ ...prev, [field]: value }));
  };

  const handleGitHubChange = (field: keyof GitHubSettings, value: string) => {
    setGitHubForm(prev => ({ ...prev, [field]: value }));
  };

  const handleConfluenceChange = (field: keyof ConfluenceSettings, value: string) => {
    setConfluenceForm(prev => ({ ...prev, [field]: value }));
  };

  const handleOllamaChange = (field: keyof OllamaSettings, value: string | string[]) => {
    setOllamaForm(prev => ({ ...prev, [field]: value }));
  };

  const handleLMStudioChange = (field: keyof LMStudioSettings, value: string | string[]) => {
    setLMStudioForm(prev => ({ ...prev, [field]: value }));
  };

  const saveAnthropicSettings = async () => {
    try {
      await updateAnthropicSettings(anthropicForm);
      setTestResults(prev => ({ ...prev, anthropic: { success: true, message: 'Settings saved successfully!' } }));
    } catch (error) {
      setTestResults(prev => ({ 
        ...prev, 
        anthropic: { 
          success: false, 
          message: error instanceof Error ? error.message : 'Failed to save settings' 
        } 
      }));
    }
  };

  const saveGitHubSettings = async () => {
    try {
      await updateGitHubSettings(githubForm);
      setTestResults(prev => ({ ...prev, github: { success: true, message: 'Settings saved successfully!' } }));
    } catch (error) {
      setTestResults(prev => ({ 
        ...prev, 
        github: { 
          success: false, 
          message: error instanceof Error ? error.message : 'Failed to save settings' 
        } 
      }));
    }
  };

  const saveConfluenceSettings = async () => {
    try {
      await updateConfluenceSettings(confluenceForm);
      setTestResults(prev => ({ ...prev, confluence: { success: true, message: 'Settings saved successfully!' } }));
    } catch (error) {
      setTestResults(prev => ({ 
        ...prev, 
        confluence: { 
          success: false, 
          message: error instanceof Error ? error.message : 'Failed to save settings' 
        } 
      }));
    }
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

  const togglePasswordVisibility = (field: string) => {
    setShowPasswords(prev => ({ ...prev, [field]: !prev[field] }));
  };

  const testConnection = async (service: string) => {
    setTestResults(prev => ({ ...prev, [service]: { success: false, message: 'Testing...' } }));
    
    try {
      let result = false;
      switch (service) {
        case 'anthropic':
          result = await testAnthropicConnection();
          break;
        case 'github':
          result = await testGitHubConnection();
          break;
        case 'confluence':
          result = await testConfluenceConnection();
          break;
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

  const clearAllIntegrations = async () => {
    if (confirm('Are you sure you want to clear all integration settings? This action cannot be undone.')) {
      await clearIntegrationSettings();
      setAnthropicForm(integrations.anthropic);
      setGitHubForm(integrations.github);
      setConfluenceForm(integrations.confluence);
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

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div 
        className="absolute inset-0 bg-black bg-opacity-50"
        onClick={onClose}
      />
      
      {/* Modal */}
      <div className="relative bg-slack-bg border border-slack-border rounded-lg shadow-xl w-full max-w-2xl max-h-[90vh] overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b border-slack-border">
          <h2 className="text-xl font-bold text-slack-text">Settings</h2>
          <button
            onClick={onClose}
            className="text-slack-textMuted hover:text-slack-text transition-colors"
          >
            <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        {/* Tabs — horizontal scroll so About stays reachable on narrow modals */}
        <div className="flex flex-nowrap overflow-x-auto overscroll-x-contain border-b border-slack-border shrink-0">
          <button
            onClick={() => setActiveTab('appearance')}
            className={`shrink-0 whitespace-nowrap px-4 py-3 text-sm font-medium transition-colors ${
              activeTab === 'appearance'
                ? 'text-slack-text border-b-2 border-slack-accent'
                : 'text-slack-textMuted hover:text-slack-text'
            }`}
          >
            Appearance
          </button>
          <button
            onClick={() => setActiveTab('layout')}
            className={`shrink-0 whitespace-nowrap px-4 py-3 text-sm font-medium transition-colors ${
              activeTab === 'layout'
                ? 'text-slack-text border-b-2 border-slack-accent'
                : 'text-slack-textMuted hover:text-slack-text'
            }`}
          >
            Layout
          </button>
          <button
            onClick={() => setActiveTab('chat')}
            className={`shrink-0 whitespace-nowrap px-4 py-3 text-sm font-medium transition-colors ${
              activeTab === 'chat'
                ? 'text-slack-text border-b-2 border-slack-accent'
                : 'text-slack-textMuted hover:text-slack-text'
            }`}
          >
            Chat &amp; agents
          </button>
          <button
            onClick={() => setActiveTab('integrations')}
            className={`shrink-0 whitespace-nowrap px-4 py-3 text-sm font-medium transition-colors ${
              activeTab === 'integrations'
                ? 'text-slack-text border-b-2 border-slack-accent'
                : 'text-slack-textMuted hover:text-slack-text'
            }`}
          >
            Integrations
          </button>
          <button
            onClick={() => setActiveTab('ai-providers')}
            className={`shrink-0 whitespace-nowrap px-4 py-3 text-sm font-medium transition-colors ${
              activeTab === 'ai-providers'
                ? 'text-slack-text border-b-2 border-slack-accent'
                : 'text-slack-textMuted hover:text-slack-text'
            }`}
          >
            AI Providers
          </button>
          <button
            onClick={() => setActiveTab('domain-packs')}
            className={`shrink-0 whitespace-nowrap px-4 py-3 text-sm font-medium transition-colors ${
              activeTab === 'domain-packs'
                ? 'text-slack-text border-b-2 border-slack-accent'
                : 'text-slack-textMuted hover:text-slack-text'
            }`}
          >
            Domain packs
          </button>
          <button
            onClick={() => setActiveTab('about')}
            className={`shrink-0 whitespace-nowrap px-4 py-3 text-sm font-medium transition-colors ${
              activeTab === 'about'
                ? 'text-slack-text border-b-2 border-slack-accent'
                : 'text-slack-textMuted hover:text-slack-text'
            }`}
          >
            About
          </button>
        </div>

        {/* Content */}
        <div className="p-6 max-h-[60vh] overflow-y-auto">
          {activeTab === 'appearance' && (
            <div className="space-y-6">
              {/* Font Size */}
              <div>
                <label className="block text-sm font-medium text-slack-text mb-3">
                  Font Size: {settings.fontSize}px
                </label>
                <input
                  type="range"
                  min="12"
                  max="24"
                  value={settings.fontSize}
                  onChange={handleFontSizeChange}
                  className="w-full h-2 bg-slack-bgHover rounded-lg appearance-none cursor-pointer slider"
                />
                <div className="flex justify-between text-xs text-slack-textMuted mt-1">
                  <span>12px</span>
                  <span>24px</span>
                </div>
              </div>

              {/* Font Size Scope */}
              <div>
                <label className="block text-sm font-medium text-slack-text mb-3">
                  Apply font size to:
                </label>
                <div className="space-y-2">
                  {[
                    { value: 'messages', label: 'Messages only', description: 'Only message content' },
                    { value: 'input', label: 'Messages & Input', description: 'Chat messages and input field' },
                    { value: 'global', label: 'Global', description: 'Entire application' },
                  ].map((option) => (
                    <label key={option.value} className="flex items-start space-x-3 cursor-pointer">
                      <input
                        type="radio"
                        name="fontScope"
                        value={option.value}
                        checked={settings.fontSizeScope === option.value}
                        onChange={() => handleScopeChange(option.value as FontSizeScope)}
                        className="mt-1 text-slack-accent focus:ring-slack-accent"
                      />
                      <div>
                        <div className="text-sm font-medium text-slack-text">{option.label}</div>
                        <div className="text-xs text-slack-textMuted">{option.description}</div>
                      </div>
                    </label>
                  ))}
                </div>
              </div>

              {/* Preview */}
              <div>
                <label className="block text-sm font-medium text-slack-text mb-3">
                  Preview:
                </label>
                <div 
                  className="p-4 bg-slack-bgHover rounded-lg border border-slack-border"
                  style={{ fontSize: `${settings.fontSize}px` }}
                >
                  <div className="text-slack-text">
                    This is how your messages will look with the current font size.
                  </div>
                  <div className="text-slack-textMuted text-sm mt-2">
                    Sample message content with different text styles.
                  </div>
                </div>
              </div>
            </div>
          )}

          {activeTab === 'layout' && (
            <div className="space-y-6">
              <div className="mb-4">
                <h3 className="text-lg font-semibold text-slack-text mb-2">Panel Visibility</h3>
                <p className="text-sm text-slack-textMuted">
                  Configure which panels are visible by default when the app starts. You can still toggle panels manually at any time.
                </p>
              </div>

              {/* Files Panel */}
              <div className="flex items-center justify-between p-4 bg-slack-bgHover rounded-lg border border-slack-border">
                <div className="flex-1">
                  <div className="font-medium text-slack-text">Files Panel</div>
                  <div className="text-sm text-slack-textMuted">File explorer for browsing workspaces and files</div>
                </div>
                <label className="relative inline-flex items-center cursor-pointer">
                  <input
                    type="checkbox"
                    checked={layoutSettings.filesPanelVisible}
                    onChange={(e) => updateLayoutSettings({ filesPanelVisible: e.target.checked })}
                    className="sr-only peer"
                  />
                  <div className="w-11 h-6 bg-gray-600 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-800 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
                </label>
              </div>

              {/* Editor Panel */}
              <div className="flex items-center justify-between p-4 bg-slack-bgHover rounded-lg border border-slack-border">
                <div className="flex-1">
                  <div className="font-medium text-slack-text">Editor Panel</div>
                  <div className="text-sm text-slack-textMuted">Code editor for viewing and editing files</div>
                </div>
                <label className="relative inline-flex items-center cursor-pointer">
                  <input
                    type="checkbox"
                    checked={layoutSettings.editorPanelVisible}
                    onChange={(e) => updateLayoutSettings({ editorPanelVisible: e.target.checked })}
                    className="sr-only peer"
                  />
                  <div className="w-11 h-6 bg-gray-600 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-800 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
                </label>
              </div>

              {/* Terminal Panel */}
              <div className="flex items-center justify-between p-4 bg-slack-bgHover rounded-lg border border-slack-border">
                <div className="flex-1">
                  <div className="font-medium text-slack-text">Terminal Panel</div>
                  <div className="text-sm text-slack-textMuted">Terminal for executing commands and viewing output</div>
                </div>
                <label className="relative inline-flex items-center cursor-pointer">
                  <input
                    type="checkbox"
                    checked={layoutSettings.terminalPanelVisible}
                    onChange={(e) => updateLayoutSettings({ terminalPanelVisible: e.target.checked })}
                    className="sr-only peer"
                  />
                  <div className="w-11 h-6 bg-gray-600 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-800 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
                </label>
              </div>

              {/* My Agents Panel */}
              <div className="flex items-center justify-between p-4 bg-slack-bgHover rounded-lg border border-slack-border">
                <div className="flex-1">
                  <div className="font-medium text-slack-text">My Agents Panel</div>
                  <div className="text-sm text-slack-textMuted">Manage and view your custom repository agents</div>
                </div>
                <label className="relative inline-flex items-center cursor-pointer">
                  <input
                    type="checkbox"
                    checked={layoutSettings.myAgentsPanelVisible}
                    onChange={(e) => updateLayoutSettings({ myAgentsPanelVisible: e.target.checked })}
                    className="sr-only peer"
                  />
                  <div className="w-11 h-6 bg-gray-600 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-800 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
                </label>
              </div>

              {/* Pending Changes Panel */}
              <div className="flex items-center justify-between p-4 bg-slack-bgHover rounded-lg border border-slack-border">
                <div className="flex-1">
                  <div className="font-medium text-slack-text">Pending Changes Panel</div>
                  <div className="text-sm text-slack-textMuted">View and manage pending file changes</div>
                </div>
                <label className="relative inline-flex items-center cursor-pointer">
                  <input
                    type="checkbox"
                    checked={layoutSettings.pendingChangesPanelVisible}
                    onChange={(e) => updateLayoutSettings({ pendingChangesPanelVisible: e.target.checked })}
                    className="sr-only peer"
                  />
                  <div className="w-11 h-6 bg-gray-600 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-800 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
                </label>
              </div>

              {/* Sidebar agent shortcuts */}
              <div className="flex items-center justify-between p-4 bg-slack-bgHover rounded-lg border border-slack-border">
                <div className="flex-1">
                  <div className="font-medium text-slack-text">Sidebar agent shortcuts</div>
                  <div className="text-sm text-slack-textMuted">
                    Show agents without a DM under Direct Messages. Turn off for a cleaner list; open DMs stay.
                  </div>
                </div>
                <label className="relative inline-flex items-center cursor-pointer">
                  <input
                    type="checkbox"
                    checked={layoutSettings.sidebarAgentsVisible}
                    onChange={(e) => updateLayoutSettings({ sidebarAgentsVisible: e.target.checked })}
                    className="sr-only peer"
                  />
                  <div className="w-11 h-6 bg-gray-600 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-800 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
                </label>
              </div>
            </div>
          )}

          {activeTab === 'chat' && (
            <div className="space-y-8">
              <div>
                <h3 className="text-lg font-semibold text-slack-text mb-2">User rules (markdown)</h3>
                <p className="text-sm text-slack-textMuted mb-2">
                  Included on every message you send (main chat and threads). Agents treat this as your standing instructions.
                </p>
                <textarea
                  value={settings.userRulesMarkdown ?? ''}
                  onChange={(e) => void updateSettings({ userRulesMarkdown: e.target.value })}
                  rows={10}
                  className="w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-sm text-slack-text font-mono focus:outline-none focus:ring-2 focus:ring-slack-accent"
                  placeholder={'- Prefer concise answers\n- Stack: Rust, TypeScript'}
                />
                <p className="text-xs text-slack-textMuted mt-1">{(settings.userRulesMarkdown ?? '').length} characters</p>
              </div>

              <div>
                <h3 className="text-lg font-semibold text-slack-text mb-2">Hidden from sidebar</h3>
                <p className="text-sm text-slack-textMuted mb-3">
                  DM rows, collaborations, or agent shortcuts you hid. They reappear automatically when you open them or when an agent becomes active again (e.g. enabling a domain pack). Use Unhide here anytime (nothing is deleted or cancelled).
                </p>
                {(settings.hiddenDmChannelNames?.length ?? 0) === 0 &&
                (settings.hiddenCollaborationChannelNames?.length ?? 0) === 0 &&
                (settings.hiddenAgentSidebarKeys?.length ?? 0) === 0 &&
                (settings.hiddenAgentIdsForSidebar?.length ?? 0) === 0 ? (
                  <p className="text-sm text-slack-textMuted">None. Use × on a row in the sidebar.</p>
                ) : (
                  <div className="space-y-2">
                    {(settings.hiddenDmChannelNames ?? []).map((name) => (
                      <div
                        key={`dm-${name}`}
                        className="flex items-center justify-between gap-2 p-2 bg-slack-bgHover rounded border border-slack-border"
                      >
                        <span className="text-sm text-slack-text truncate" title={name}>
                          DM: {channels.find((c) => c.name === name)?.description || name}
                        </span>
                        <button
                          type="button"
                          className="shrink-0 text-xs text-slack-accent hover:underline"
                          onClick={() =>
                            void updateSettings({
                              hiddenDmChannelNames: (settings.hiddenDmChannelNames ?? []).filter((n) => n !== name),
                            })
                          }
                        >
                          Unhide
                        </button>
                      </div>
                    ))}
                    {(settings.hiddenCollaborationChannelNames ?? []).map((name) => {
                      const ch = channels.find((c) => c.name === name);
                      const label =
                        ch?.description?.trim() && !ch.description.startsWith('collab-')
                          ? ch.description
                          : name.replace(/^collab-/, '').slice(0, 8) || name;
                      return (
                        <div
                          key={`collab-${name}`}
                          className="flex items-center justify-between gap-2 p-2 bg-slack-bgHover rounded border border-slack-border"
                        >
                          <span className="text-sm text-slack-text truncate" title={name}>
                            Collab: {label}
                          </span>
                          <button
                            type="button"
                            className="shrink-0 text-xs text-slack-accent hover:underline"
                            onClick={() =>
                              void updateSettings({
                                hiddenCollaborationChannelNames: (
                                  settings.hiddenCollaborationChannelNames ?? []
                                ).filter((n) => n !== name),
                              })
                            }
                          >
                            Unhide
                          </button>
                        </div>
                      );
                    })}
                    {(settings.hiddenAgentSidebarKeys ?? []).map((key) => {
                      const label = key.includes(':') ? key.slice(key.indexOf(':') + 1) : key;
                      return (
                        <div
                          key={`agk-${key}`}
                          className="flex items-center justify-between gap-2 p-2 bg-slack-bgHover rounded border border-slack-border"
                        >
                          <span className="text-sm text-slack-text truncate">
                            Agent shortcut: {label}
                          </span>
                          <button
                            type="button"
                            className="shrink-0 text-xs text-slack-accent hover:underline"
                            onClick={() =>
                              void updateSettings({
                                hiddenAgentSidebarKeys: (settings.hiddenAgentSidebarKeys ?? []).filter(
                                  (x) => x !== key
                                ),
                              })
                            }
                          >
                            Unhide
                          </button>
                        </div>
                      );
                    })}
                    {(settings.hiddenAgentIdsForSidebar ?? []).map((id) => (
                      <div
                        key={`ag-${id}`}
                        className="flex items-center justify-between gap-2 p-2 bg-slack-bgHover rounded border border-slack-border"
                      >
                        <span className="text-sm text-slack-text truncate">
                          Agent shortcut: {agents.find((a) => a.id === id)?.name || id}
                        </span>
                        <button
                          type="button"
                          className="shrink-0 text-xs text-slack-accent hover:underline"
                          onClick={() => {
                            const agent = agents.find((a) => a.id === id);
                            void updateSettings({
                              hiddenAgentIdsForSidebar: (settings.hiddenAgentIdsForSidebar ?? []).filter(
                                (x) => x !== id
                              ),
                              ...(agent
                                ? {
                                    hiddenAgentSidebarKeys: (settings.hiddenAgentSidebarKeys ?? []).filter(
                                      (k) => k !== agentSidebarHideKey(agent)
                                    ),
                                  }
                                : {}),
                            });
                          }}
                        >
                          Unhide
                        </button>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>
          )}

          {activeTab === 'integrations' && (
            <div className="space-y-8">
              {/* Anthropic Settings */}
              <div className="border border-slack-border rounded-lg p-6">
                <div className="flex items-center justify-between mb-4">
                  <h3 className="text-lg font-semibold text-slack-text">Anthropic API</h3>
                  <div className="flex items-center space-x-2">
                    {anthropicForm.apiKey && (
                      <span className="text-green-500 text-sm">✓ Configured</span>
                    )}
                    <button
                      onClick={() => testConnection('anthropic')}
                      className="px-3 py-1 text-sm bg-slack-accent text-white rounded hover:bg-slack-accentHover transition-colors"
                    >
                      Test
                    </button>
                  </div>
                </div>
                
                {testResults.anthropic && (
                  <div className={`mb-4 p-3 rounded text-sm ${
                    testResults.anthropic.success 
                      ? 'bg-green-100 text-green-800 border border-green-200' 
                      : 'bg-red-100 text-red-800 border border-red-200'
                  }`}>
                    {testResults.anthropic.message}
                  </div>
                )}

                <div className="space-y-4">
                  <div>
                    <label className="block text-sm font-medium text-slack-text mb-2">
                      API Key
                    </label>
                    <div className="relative">
                      <input
                        type={showPasswords.anthropic ? 'text' : 'password'}
                        value={anthropicForm.apiKey}
                        onChange={(e) => handleAnthropicChange('apiKey', e.target.value)}
                        placeholder="sk-ant-..."
                        className="w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-slack-text focus:outline-none focus:ring-2 focus:ring-slack-accent"
                      />
                      <button
                        type="button"
                        onClick={() => togglePasswordVisibility('anthropic')}
                        className="absolute right-3 top-1/2 transform -translate-y-1/2 text-slack-textMuted hover:text-slack-text"
                      >
                        {showPasswords.anthropic ? '👁️' : '👁️‍🗨️'}
                      </button>
                    </div>
                    <p className="text-xs text-slack-textMuted mt-1">
                      Get your API key from{' '}
                      <button
                        onClick={() => openLink('https://console.anthropic.com/')}
                        className="text-slack-accent hover:underline"
                      >
                        Anthropic Console
                      </button>
                    </p>
                  </div>

                  <div className="flex items-center space-x-2">
                    <input
                      type="checkbox"
                      id="useAIHub"
                      checked={anthropicForm.useAIHub}
                      onChange={(e) => handleAnthropicChange('useAIHub', e.target.checked)}
                      className="text-slack-accent focus:ring-slack-accent"
                    />
                    <label htmlFor="useAIHub" className="text-sm text-slack-text">
                      Use AI Hub (recommended)
                    </label>
                  </div>

                  {anthropicForm.useAIHub && (
                    <>
                      <div>
                        <label className="block text-sm font-medium text-slack-text mb-2">
                          AI Hub Endpoint
                        </label>
                        <input
                          type="text"
                          value={anthropicForm.aiHubEndpoint}
                          onChange={(e) => handleAnthropicChange('aiHubEndpoint', e.target.value)}
                          className="w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-slack-text focus:outline-none focus:ring-2 focus:ring-slack-accent"
                        />
                      </div>
                      <div>
                        <label className="block text-sm font-medium text-slack-text mb-2">
                          Model
                        </label>
                        <select
                          value={anthropicForm.aiHubModel}
                          onChange={(e) => handleAnthropicChange('aiHubModel', e.target.value)}
                          className="w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-slack-text focus:outline-none focus:ring-2 focus:ring-slack-accent"
                        >
                          <option value="claude-sonnet">Claude Sonnet (recommended)</option>
                          <option value="claude-haiku">Claude Haiku (faster)</option>
                        </select>
                      </div>
                    </>
                  )}

                  <button
                    onClick={saveAnthropicSettings}
                    className="w-full px-4 py-2 bg-slack-accent text-white rounded hover:bg-slack-accentHover transition-colors"
                  >
                    Save Anthropic Settings
                  </button>
                </div>
              </div>

              {/* GitHub Settings */}
              <div className="border border-slack-border rounded-lg p-6">
                <div className="flex items-center justify-between mb-4">
                  <h3 className="text-lg font-semibold text-slack-text">GitHub</h3>
                  <div className="flex items-center space-x-2">
                    {githubForm.personalAccessToken && (
                      <span className="text-green-500 text-sm">✓ Configured</span>
                    )}
                    <button
                      onClick={() => testConnection('github')}
                      className="px-3 py-1 text-sm bg-slack-accent text-white rounded hover:bg-slack-accentHover transition-colors"
                    >
                      Test
                    </button>
                  </div>
                </div>
                
                {testResults.github && (
                  <div className={`mb-4 p-3 rounded text-sm ${
                    testResults.github.success 
                      ? 'bg-green-100 text-green-800 border border-green-200' 
                      : 'bg-red-100 text-red-800 border border-red-200'
                  }`}>
                    {testResults.github.message}
                  </div>
                )}

                <div className="space-y-4">
                  <div>
                    <label className="block text-sm font-medium text-slack-text mb-2">
                      Personal Access Token
                    </label>
                    <div className="relative">
                      <input
                        type={showPasswords.github ? 'text' : 'password'}
                        value={githubForm.personalAccessToken}
                        onChange={(e) => handleGitHubChange('personalAccessToken', e.target.value)}
                        placeholder="ghp_..."
                        className="w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-slack-text focus:outline-none focus:ring-2 focus:ring-slack-accent"
                      />
                      <button
                        type="button"
                        onClick={() => togglePasswordVisibility('github')}
                        className="absolute right-3 top-1/2 transform -translate-y-1/2 text-slack-textMuted hover:text-slack-text"
                      >
                        {showPasswords.github ? '👁️' : '👁️‍🗨️'}
                      </button>
                    </div>
                    <p className="text-xs text-slack-textMuted mt-1">
                      Create a token at{' '}
                      <button
                        onClick={() => openLink('https://github.com/settings/tokens')}
                        className="text-slack-accent hover:underline"
                      >
                        GitHub Settings
                      </button>
                      {' '}with repo, read:org permissions
                    </p>
                  </div>

                  <button
                    onClick={saveGitHubSettings}
                    className="w-full px-4 py-2 bg-slack-accent text-white rounded hover:bg-slack-accentHover transition-colors"
                  >
                    Save GitHub Settings
                  </button>
                </div>
              </div>

              {/* Google Meet notes (Assistant) */}
              <div className="border border-slack-border rounded-lg p-6 mb-6">
                <div className="flex items-center justify-between mb-4">
                  <h3 className="text-lg font-semibold text-slack-text">Google Meet notes</h3>
                  <button
                    type="button"
                    onClick={() => void refreshGoogleMeetNotesStatus()}
                    disabled={googleMeetNotesLoading}
                    className="px-3 py-1 text-sm border border-slack-border rounded hover:bg-slack-bgHover text-slack-text"
                  >
                    Refresh
                  </button>
                </div>
                <p className="text-sm text-slack-textMuted mb-4">
                  Sync Gemini meeting notes from Gmail into the Assistant. Create a Google Cloud OAuth
                  web client, add the redirect URI below, then save your Client ID and Secret.
                </p>
                <div className="space-y-4 mb-4">
                  <div>
                    <label className="block text-sm font-medium text-slack-text mb-2">
                      OAuth Client ID
                    </label>
                    <input
                      type="text"
                      value={googleOAuthForm.clientId}
                      onChange={(e) =>
                        setGoogleOAuthForm((prev) => ({ ...prev, clientId: e.target.value }))
                      }
                      placeholder="xxxx.apps.googleusercontent.com"
                      className="w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-slack-text focus:outline-none focus:ring-2 focus:ring-slack-accent"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-slack-text mb-2">
                      OAuth Client Secret
                      {googleOAuthSecretSet && !googleOAuthForm.clientSecret && (
                        <span className="ml-2 text-xs text-green-600">(saved)</span>
                      )}
                    </label>
                    <input
                      type={showPasswords.googleOAuth ? 'text' : 'password'}
                      value={googleOAuthForm.clientSecret}
                      onChange={(e) =>
                        setGoogleOAuthForm((prev) => ({ ...prev, clientSecret: e.target.value }))
                      }
                      placeholder={googleOAuthSecretSet ? 'Leave blank to keep existing secret' : 'Client secret'}
                      className="w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-slack-text focus:outline-none focus:ring-2 focus:ring-slack-accent"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-slack-text mb-2">
                      Redirect URI
                    </label>
                    <input
                      type="text"
                      value={googleOAuthForm.redirectUrl}
                      onChange={(e) =>
                        setGoogleOAuthForm((prev) => ({ ...prev, redirectUrl: e.target.value }))
                      }
                      className="w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-slack-text font-mono text-xs focus:outline-none focus:ring-2 focus:ring-slack-accent"
                    />
                    <p className="text-xs text-slack-textMuted mt-1">
                      Add this exact URI in Google Cloud Console → Credentials → your OAuth client.
                    </p>
                  </div>
                  <button
                    type="button"
                    onClick={() => void saveGoogleOAuthSettings()}
                    disabled={
                      googleMeetNotesBusy ||
                      !googleOAuthForm.clientId.trim() ||
                      (!googleOAuthSecretSet && !googleOAuthForm.clientSecret.trim())
                    }
                    className="w-full px-4 py-2 bg-slack-accent text-white rounded hover:bg-slack-accentHover disabled:opacity-50"
                  >
                    Save OAuth credentials
                  </button>
                </div>
                {testResults.googleMeetNotes && (
                  <div
                    className={`mb-4 p-3 rounded text-sm ${
                      testResults.googleMeetNotes.success
                        ? 'bg-green-100 text-green-800 border border-green-200'
                        : 'bg-red-100 text-red-800 border border-red-200'
                    }`}
                  >
                    {testResults.googleMeetNotes.message}
                  </div>
                )}
                {googleMeetNotesLoading && !googleMeetNotes ? (
                  <p className="text-sm text-slack-textMuted">Loading status…</p>
                ) : googleMeetNotes ? (
                  <div className="space-y-3 text-sm text-slack-text">
                    <p>
                      <span className="font-medium">Hub OAuth:</span>{' '}
                      {googleMeetNotes.oauth_configured ? 'configured' : 'not configured on server'}
                    </p>
                    <p>
                      <span className="font-medium">Account:</span>{' '}
                      {googleMeetNotes.connected
                        ? googleMeetNotes.email || 'connected'
                        : 'not connected'}
                    </p>
                    {googleMeetNotes.connected && (
                      <>
                        <p>
                          <span className="font-medium">Stored notes:</span>{' '}
                          {googleMeetNotes.notes_count ?? 0}
                        </p>
                        {googleMeetNotes.last_sync_at && (
                          <p>
                            <span className="font-medium">Last sync:</span>{' '}
                            {new Date(googleMeetNotes.last_sync_at).toLocaleString()}
                          </p>
                        )}
                      </>
                    )}
                  </div>
                ) : null}
                <div className="flex flex-wrap gap-2 mt-4">
                  <button
                    type="button"
                    onClick={() => void connectGoogleMeetNotes()}
                    disabled={googleMeetNotesBusy || !googleMeetNotes?.oauth_configured}
                    className="px-4 py-2 bg-slack-accent text-white rounded hover:bg-slack-accentHover disabled:opacity-50"
                  >
                    Connect Google
                  </button>
                  <button
                    type="button"
                    onClick={() => void syncGoogleMeetNotesNow()}
                    disabled={googleMeetNotesBusy || !googleMeetNotes?.connected}
                    className="px-4 py-2 border border-slack-border rounded hover:bg-slack-bgHover disabled:opacity-50"
                  >
                    Sync now
                  </button>
                  <button
                    type="button"
                    onClick={() => void disconnectGoogleMeetNotes()}
                    disabled={googleMeetNotesBusy || !googleMeetNotes?.connected}
                    className="px-4 py-2 text-red-600 border border-red-300 rounded hover:bg-red-50 disabled:opacity-50"
                  >
                    Disconnect
                  </button>
                </div>
              </div>

              {/* Confluence Settings */}
              <div className="border border-slack-border rounded-lg p-6">
                <div className="flex items-center justify-between mb-4">
                  <h3 className="text-lg font-semibold text-slack-text">Confluence</h3>
                  <div className="flex items-center space-x-2">
                    {confluenceForm.domain && confluenceForm.email && confluenceForm.apiToken && (
                      <span className="text-green-500 text-sm">✓ Configured</span>
                    )}
                    <button
                      onClick={() => testConnection('confluence')}
                      className="px-3 py-1 text-sm bg-slack-accent text-white rounded hover:bg-slack-accentHover transition-colors"
                    >
                      Test
                    </button>
                  </div>
                </div>
                
                {testResults.confluence && (
                  <div className={`mb-4 p-3 rounded text-sm ${
                    testResults.confluence.success 
                      ? 'bg-green-100 text-green-800 border border-green-200' 
                      : 'bg-red-100 text-red-800 border border-red-200'
                  }`}>
                    {testResults.confluence.message}
                  </div>
                )}

                <div className="space-y-4">
                  <div>
                    <label className="block text-sm font-medium text-slack-text mb-2">
                      Domain
                    </label>
                    <input
                      type="text"
                      value={confluenceForm.domain}
                      onChange={(e) => handleConfluenceChange('domain', e.target.value)}
                      placeholder="yourcompany.atlassian.net"
                      className="w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-slack-text focus:outline-none focus:ring-2 focus:ring-slack-accent"
                    />
                  </div>

                  <div>
                    <label className="block text-sm font-medium text-slack-text mb-2">
                      Email
                    </label>
                    <input
                      type="email"
                      value={confluenceForm.email}
                      onChange={(e) => handleConfluenceChange('email', e.target.value)}
                      placeholder="your.email@company.com"
                      className="w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-slack-text focus:outline-none focus:ring-2 focus:ring-slack-accent"
                    />
                  </div>

                  <div>
                    <label className="block text-sm font-medium text-slack-text mb-2">
                      API Token
                    </label>
                    <div className="relative">
                      <input
                        type={showPasswords.confluence ? 'text' : 'password'}
                        value={confluenceForm.apiToken}
                        onChange={(e) => handleConfluenceChange('apiToken', e.target.value)}
                        placeholder="Your API token"
                        className="w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-slack-text focus:outline-none focus:ring-2 focus:ring-slack-accent"
                      />
                      <button
                        type="button"
                        onClick={() => togglePasswordVisibility('confluence')}
                        className="absolute right-3 top-1/2 transform -translate-y-1/2 text-slack-textMuted hover:text-slack-text"
                      >
                        {showPasswords.confluence ? '👁️' : '👁️‍🗨️'}
                      </button>
                    </div>
                    <p className="text-xs text-slack-textMuted mt-1">
                      Get your API token from{' '}
                      <button
                        onClick={() => openLink('https://id.atlassian.com/manage-profile/security/api-tokens')}
                        className="text-slack-accent hover:underline"
                      >
                        Atlassian Account Settings
                      </button>
                    </p>
                  </div>

                  <button
                    onClick={saveConfluenceSettings}
                    className="w-full px-4 py-2 bg-slack-accent text-white rounded hover:bg-slack-accentHover transition-colors"
                  >
                    Save Confluence Settings
                  </button>
                </div>
              </div>

              {/* Clear All Button */}
              <div className="pt-4 border-t border-slack-border">
                <button
                  onClick={clearAllIntegrations}
                  className="px-4 py-2 text-red-600 border border-red-300 rounded hover:bg-red-50 transition-colors"
                >
                  Clear All Integration Settings
                </button>
              </div>
            </div>
          )}

          {activeTab === 'domain-packs' && (
            <div className="space-y-8">
              <div className="border border-slack-border rounded-lg p-4 bg-slack-bgHover/30 text-sm text-slack-textMuted">
                <strong className="text-slack-text">Always on:</strong> ChatModerator, Assistant, and CLI agents (Cursor, Gemini, Claude, Copilot) when installed on your PATH. Domain packs add optional in-process specialists and tools below.
              </div>

              <div className="border border-slack-border rounded-lg p-6">
                <h3 className="text-lg font-semibold text-slack-text mb-2">Domain packs</h3>
                <p className="text-sm text-slack-textMuted mb-4">
                  Turn optional specialist packs on or off. Enabled packs add experts to <strong>New DM</strong>, channel invite, and the agent sidebar.
                </p>
                {packsLoading && <p className="text-sm text-slack-textMuted">Loading packs…</p>}
                {packsErr && <p className="text-sm text-red-600 mb-2">{packsErr}</p>}
                <div className="space-y-3">
                  {domainPacks.map((pack) => (
                    <label
                      key={pack.id}
                      className="flex items-start gap-3 p-3 rounded-lg border border-slack-border bg-slack-bgHover/30 cursor-pointer"
                    >
                      <input
                        type="checkbox"
                        checked={pack.enabled}
                        disabled={packsSaving === pack.id}
                        onChange={(e) => void handlePackToggle(pack.id, e.target.checked)}
                        className="mt-1 rounded border-slack-border"
                      />
                      <div>
                        <div className="text-slack-text font-medium">{pack.title}</div>
                        <p className="text-xs text-slack-textMuted mt-1">{pack.description}</p>
                        {pack.enabled && pack.id === 'life-sciences' && pack.expert_label && (
                          <p className="text-xs text-teal-600 mt-1">
                            Expert: {pack.expert_label} — install Bio 8B from Model Library when using local Ollama. MCP tools start automatically (no env vars).
                          </p>
                        )}
                        {pack.enabled && pack.id === 'software-development' && (
                          <p className="text-xs text-blue-600 mt-1">
                            Adds GoExpert, ReactExpert, RustExpert, and other in-process specialists. Pull{' '}
                            <code className="font-mono bg-slack-bgHover px-1 rounded">qwen2.5-coder:14b</code> from Model Library for local Ollama. Dev MCP tools start with enabled agents.
                          </p>
                        )}
                      </div>
                    </label>
                  ))}
                </div>
              </div>

              {domainPacks.some((p) => p.id === 'life-sciences' && p.enabled) && (
                <div className="border border-slack-border rounded-lg p-6">
                  <h3 className="text-lg font-semibold text-slack-text mb-2">Life sciences tools</h3>
                  <p className="text-sm text-slack-textMuted mb-4">
                    Limits for <code className="font-mono text-xs bg-slack-bgHover px-1 rounded">analyze_sequence</code> and{' '}
                    <code className="font-mono text-xs bg-slack-bgHover px-1 rounded">fold_protein</code> (BiologyExpert MCP).
                  </p>
                  <label className="flex items-center gap-3 cursor-pointer mb-4">
                    <input
                      type="checkbox"
                      checked={mcpEnabled}
                      onChange={(e) => void handleMcpMasterToggle(e.target.checked)}
                      className="rounded border-slack-border"
                    />
                    <span className="text-sm text-slack-text">Enable MCP tool servers (master)</span>
                  </label>
                  <div className="grid gap-3 sm:grid-cols-2">
                    <label className="block text-sm">
                      <span className="text-slack-textMuted">Max fold length (aa)</span>
                      <input
                        type="number"
                        value={bioMaxFold}
                        onChange={(e) => setBioMaxFold(e.target.value)}
                        className="mt-1 w-full px-3 py-2 border border-slack-border rounded bg-slack-bg text-slack-text"
                      />
                    </label>
                    <label className="block text-sm">
                      <span className="text-slack-textMuted">Max analyze length</span>
                      <input
                        type="number"
                        value={bioMaxAnalyze}
                        onChange={(e) => setBioMaxAnalyze(e.target.value)}
                        className="mt-1 w-full px-3 py-2 border border-slack-border rounded bg-slack-bg text-slack-text"
                      />
                    </label>
                    <label className="block text-sm sm:col-span-2">
                      <span className="text-slack-textMuted">ESMFold model (Hub id)</span>
                      <input
                        type="text"
                        value={bioEsmfoldModel}
                        onChange={(e) => setBioEsmfoldModel(e.target.value)}
                        className="mt-1 w-full px-3 py-2 border border-slack-border rounded bg-slack-bg text-slack-text font-mono text-sm"
                      />
                    </label>
                    <label className="block text-sm sm:col-span-2">
                      <span className="text-slack-textMuted">Artifacts directory (empty = ~/.neural-junkie/bio)</span>
                      <input
                        type="text"
                        value={bioArtifactsDir}
                        onChange={(e) => setBioArtifactsDir(e.target.value)}
                        placeholder="~/.neural-junkie/bio"
                        className="mt-1 w-full px-3 py-2 border border-slack-border rounded bg-slack-bg text-slack-text font-mono text-sm"
                      />
                    </label>
                  </div>
                  <button
                    type="button"
                    onClick={() => void saveBioMcpSettings()}
                    disabled={bioSettingsSaving}
                    className="mt-4 px-4 py-2 text-sm bg-slack-accent text-white rounded hover:bg-slack-accentHover disabled:opacity-50"
                  >
                    {bioSettingsSaving ? 'Saving…' : 'Save life sciences tools'}
                  </button>
                  {bioSettingsErr && <p className="text-sm text-red-600 mt-2">{bioSettingsErr}</p>}
                  {bioSettingsOk && <p className="text-sm text-green-600 mt-2">{bioSettingsOk}</p>}
                </div>
              )}
            </div>
          )}

          {activeTab === 'ai-providers' && (
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
                {collabRoutingErr && (
                  <p className="text-sm text-red-600 mt-2">{collabRoutingErr}</p>
                )}
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
          )}

          {activeTab === 'about' && (
            <div className="space-y-6">
              {/* App Info */}
              <div>
                <h3 className="text-lg font-semibold text-slack-text mb-2">{APP_INFO.name}</h3>
                <p className="text-slack-textMuted mb-4">{APP_INFO.description}</p>
                <div className="grid grid-cols-2 gap-4 text-sm">
                  <div>
                    <span className="text-slack-textMuted">Version:</span>
                    <span className="ml-2 text-slack-text">{appVersion}</span>
                  </div>
                  <div>
                    <span className="text-slack-textMuted">License:</span>
                    <span className="ml-2 text-slack-text">{APP_INFO.license}</span>
                  </div>
                </div>
              </div>

              <div>
                <h3 className="text-lg font-semibold text-slack-text mb-2">Hub connection</h3>
                <div className="space-y-2 text-sm">
                  <div className="p-3 bg-slack-bgHover rounded">
                    <span className="text-slack-textMuted">HTTP:</span>
                    <span className="ml-2 text-slack-text font-mono break-all">{getHubBaseURL()}</span>
                  </div>
                  <div className="p-3 bg-slack-bgHover rounded">
                    <span className="text-slack-textMuted">WebSocket:</span>
                    <span className="ml-2 text-slack-text font-mono break-all">{getHubWebSocketURL()}</span>
                  </div>
                </div>
              </div>

              {/* Technology Stack */}
              <div>
                <h4 className="text-md font-semibold text-slack-text mb-3">Technology Stack</h4>
                <div className="flex flex-wrap gap-2">
                  {TECH_STACK.map((tech) => (
                    <span
                      key={tech}
                      className="px-3 py-1 bg-slack-bgHover text-slack-text text-sm rounded-full border border-slack-border"
                    >
                      {tech}
                    </span>
                  ))}
                </div>
              </div>

              {/* Links */}
              <div>
                <h4 className="text-md font-semibold text-slack-text mb-3">Links</h4>
                <div className="space-y-2">
                  <button
                    onClick={() => openLink(APP_INFO.repository)}
                    className="block text-left text-slack-accent hover:text-slack-accentHover transition-colors"
                  >
                    📁 GitHub Repository
                  </button>
                  <button
                    onClick={() => openLink(APP_INFO.documentation)}
                    className="block text-left text-slack-accent hover:text-slack-accentHover transition-colors"
                  >
                    📚 Documentation
                  </button>
                </div>
              </div>

              {/* Copyright */}
              <div className="pt-4 border-t border-slack-border">
                <p className="text-xs text-slack-textMuted">
                  © 2025 {APP_INFO.author}. Licensed under {APP_INFO.license}.
                </p>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
