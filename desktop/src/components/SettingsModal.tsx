import { useState, useEffect } from 'react';
import { shallow } from 'zustand/shallow';
import { useSettingsStore, type ColorTheme, type FontSizeScope } from '../stores/settingsStore';
import { useChatStore } from '../stores/chatStore';
import { APP_INFO, TECH_STACK, getAppVersion } from '../utils/appInfo';
import {
  checkForAppUpdate,
  getUpdateChannelLabel,
  installAppUpdate,
} from '../utils/appUpdater';
import type {
  AnthropicSettings,
  GitHubSettings,
  ConfluenceSettings,
  OllamaSettings,
  LMStudioSettings,
  GoogleMeetNotesSettings,
  GoogleMeetNotesStatus,
  SlackStatus,
  SlackBinding,
  SlackChannelInfo,
  SlackPolicy,
  SlackConfigResponse,
  SlackConnectionResponse,
  SlackInboxConfig,
  SlackForwardRule,
} from '../types/protocol';
import { ChatAPI, type UserLearning } from '../api/chatAPI';
import { PackStoreBrowse } from './pack-store/PackStoreBrowse';
import { PackDevStudio } from './pack-store/dev/PackDevStudio';
import { usePacksStore } from '../stores/packsStore';
import { PACK_CAP } from '../stores/packCapabilities';
import { agentSidebarHideKey, parseDMDisplayName } from '../utils/dmChannelDisplay';
import { ProviderManager } from './ProviderManager';
import { CLIAgentsManager } from './CLIAgentsManager';
import { getHubBaseURL, getHubWebSocketURL } from '../config/hubUrl';
import { open } from '@tauri-apps/api/dialog';
import { useShortcutOverlay } from '../shortcuts/useShortcutOverlay';
import { getShortcutsForDisplay, formatChord } from '../shortcuts';
import {
  fetchHardwareSnapshot,
  fetchModelLookup,
  formatModelResourceHint,
  type HardwareSnapshot,
  type ModelLookup,
} from '../utils/hardwareRecommendations';

export type SettingsTab =
  | 'appearance'
  | 'layout'
  | 'keyboard'
  | 'chat'
  | 'integrations'
  | 'ai-providers'
  | 'domain-packs'
  | 'about';

const SETTINGS_NAV: Array<{ id: SettingsTab; label: string }> = [
  { id: 'appearance', label: 'Appearance' },
  { id: 'layout', label: 'Layout' },
  { id: 'keyboard', label: 'Keyboard' },
  { id: 'chat', label: 'Chat & agents' },
  { id: 'integrations', label: 'Integrations' },
  { id: 'ai-providers', label: 'AI Providers' },
  { id: 'domain-packs', label: 'Domain packs' },
  { id: 'about', label: 'About' },
];

interface SettingsModalProps {
  isOpen: boolean;
  onClose: () => void;
  initialTab?: SettingsTab;
}

function slackCanListChannelsFrom(
  status: SlackStatus | null | undefined,
  cfg: SlackConfigResponse | null | undefined
): boolean {
  return Boolean(cfg?.bot_token_set || status?.token_set || status?.configured);
}

function defaultSlackInboxForm(): SlackInboxConfig {
  return {
    enabled: false,
    agent_id: '',
    forward_rules: [
      { id: 'mentions', type: 'mention_of_me', enabled: false, slack_channel_ids: [] },
      { id: 'nj-prefix', type: 'prefix', enabled: false, prefix: 'nj:', slack_channel_ids: ['*'] },
      { id: 'robot-react', type: 'reaction', enabled: false, emoji: 'robot_face', slack_channel_ids: [] },
    ],
    human_dm_away: {
      enabled: false,
      away_enabled: false,
      schedule_enabled: false,
      schedule_timezone: Intl.DateTimeFormat().resolvedOptions().timeZone || 'America/Los_Angeles',
    },
  };
}

function mergeSlackInboxForm(inbox: SlackInboxConfig | null | undefined): SlackInboxConfig {
  const base = defaultSlackInboxForm();
  if (!inbox) return base;
  const rules =
    (inbox.forward_rules?.length ? inbox.forward_rules : base.forward_rules) ??
    base.forward_rules ??
    [];
  const byId = new Map(rules.map((r) => [r.id ?? r.type, r]));
  for (const def of base.forward_rules ?? []) {
    if (!byId.has(def.id ?? def.type)) {
      byId.set(def.id ?? def.type, def);
    }
  }
  return {
    ...base,
    ...inbox,
    forward_rules: Array.from(byId.values()),
    human_dm_away: { ...base.human_dm_away, ...inbox.human_dm_away },
  };
}

function updateForwardRule(
  inbox: SlackInboxConfig,
  ruleId: string,
  patch: Partial<SlackForwardRule>
): SlackInboxConfig {
  const rules = (inbox.forward_rules ?? []).map((r) =>
    (r.id ?? r.type) === ruleId ? { ...r, ...patch } : r
  );
  return { ...inbox, forward_rules: rules };
}

export function SettingsModal({ isOpen, onClose, initialTab }: SettingsModalProps) {
  const { 
    settings, 
    integrations,
    layoutSettings,
    updateFontSize,
    updateFontSizeScope,
    updateColorTheme,
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
  const { switchAllAgentProviders, serverAddr: chatServerAddr, channels, agents, setChannels } = useChatStore(
    (s) => ({
      switchAllAgentProviders: s.switchAllAgentProviders,
      serverAddr: s.serverAddr,
      channels: s.channels,
      agents: s.agents,
      setChannels: s.setChannels,
    }),
    shallow
  );
  const hubHttp =
    chatServerAddr.startsWith('http') ? chatServerAddr : `http://${chatServerAddr}`;
  const [activeTab, setActiveTab] = useState<SettingsTab>('appearance');

  useEffect(() => {
    if (isOpen && initialTab) {
      setActiveTab(initialTab);
    }
  }, [isOpen, initialTab]);
  const [appVersion, setAppVersion] = useState<string>('1.0.0');
  const [updateCheckStatus, setUpdateCheckStatus] = useState<string | null>(null);
  const [updateChecking, setUpdateChecking] = useState(false);
  const [updateInstalling, setUpdateInstalling] = useState(false);
  const [updateProgress, setUpdateProgress] = useState(0);
  const [pendingUpdateVersion, setPendingUpdateVersion] = useState<string | null>(null);
  
  // Integration form states
  const [anthropicForm, setAnthropicForm] = useState<AnthropicSettings>(integrations.anthropic);
  const [githubForm, setGitHubForm] = useState<GitHubSettings>(integrations.github);
  const [confluenceForm, setConfluenceForm] = useState<ConfluenceSettings>(integrations.confluence);
  const [googleOAuthForm, setGoogleOAuthForm] = useState<GoogleMeetNotesSettings>(integrations.googleMeetNotes);
  const [googleOAuthSecretSet, setGoogleOAuthSecretSet] = useState(false);
  const [ollamaForm, setOllamaForm] = useState<OllamaSettings>(integrations.ollama);
  const [hardwareSnapshot, setHardwareSnapshot] = useState<HardwareSnapshot | null>(null);
  const [defaultModelLookup, setDefaultModelLookup] = useState<ModelLookup | null>(null);
  const [lmstudioForm, setLMStudioForm] = useState<LMStudioSettings>(integrations.lmstudio);
  const [showPasswords, setShowPasswords] = useState<Record<string, boolean>>({});
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
  const [delegationSaving, setDelegationSaving] = useState(false);
  const [collabAssetsRoot, setCollabAssetsRoot] = useState('');
  const [collabAssetsPersisted, setCollabAssetsPersisted] = useState('');
  const [collabAssetsSaving, setCollabAssetsSaving] = useState(false);
  const [collabAssetsErr, setCollabAssetsErr] = useState<string | null>(null);
  const [collabAssetsOk, setCollabAssetsOk] = useState<string | null>(null);
  const [googleMeetNotes, setGoogleMeetNotes] = useState<GoogleMeetNotesStatus | null>(null);
  const [googleMeetNotesLoading, setGoogleMeetNotesLoading] = useState(false);
  const [googleMeetNotesBusy, setGoogleMeetNotesBusy] = useState(false);
  const [googleAdvancedOpen, setGoogleAdvancedOpen] = useState(false);
  const [slackStatus, setSlackStatus] = useState<SlackStatus | null>(null);
  const [slackConfig, setSlackConfig] = useState<SlackConfigResponse | null>(null);
  const [slackBindings, setSlackBindings] = useState<SlackBinding[]>([]);
  const [slackLoading, setSlackLoading] = useState(false);
  const [slackBusy, setSlackBusy] = useState(false);
  const [slackForm, setSlackForm] = useState({
    enabled: false,
    appToken: '',
    botToken: '',
    displayName: 'Camron',
    defaultPolicy: 'mention_only' as SlackPolicy,
    clientId: '',
    clientSecret: '',
    redirectUrl: `${hubHttp}/api/slack/oauth/callback`,
  });
  const [slackBindingForm, setSlackBindingForm] = useState({
    slackChannelId: '',
    slackChannelName: '',
    agentId: '',
    policy: 'mention_only' as SlackPolicy,
  });
  const [slackChannels, setSlackChannels] = useState<SlackChannelInfo[]>([]);
  const [slackChannelsLoading, setSlackChannelsLoading] = useState(false);
  const [slackChannelsError, setSlackChannelsError] = useState<string | null>(null);
  const [slackConnection, setSlackConnection] = useState<SlackConnectionResponse | null>(null);
  const [slackInbox, setSlackInbox] = useState<SlackInboxConfig>(() => defaultSlackInboxForm());
  const [slackAdvancedOpen, setSlackAdvancedOpen] = useState(false);
  const [specialistModelsAdvancedOpen, setSpecialistModelsAdvancedOpen] = useState(false);
  const [packsLoading, setPacksLoading] = useState(false);
  const packs = usePacksStore((s) => s.packs);
  const layoutOwner = usePacksStore((s) => s.layoutOwner);
  const setLayoutOwner = usePacksStore((s) => s.setLayoutOwner);
  const bioPackTools = usePacksStore((s) =>
    s.packs.some((p) => p.id === 'life-sciences' && p.enabled),
  );
  const bioSecondaryAnalysisTools = usePacksStore(
    (s) =>
      s.hasCapability(PACK_CAP.SECONDARY_ANALYSIS_API) ||
      s.hasCapability(PACK_CAP.SECONDARY_ANALYSIS_PYTHON),
  );
  const cadPackTools = usePacksStore((s) => s.hasCapability(PACK_CAP.CAD_API));
  const hasPersonalLearning = usePacksStore((s) => s.hasCapability(PACK_CAP.PERSONAL_LEARNING));
  const hasLoRATraining = usePacksStore((s) => s.hasCapability(PACK_CAP.LORA_TRAINING));
  const [packsErr, setPacksErr] = useState<string | null>(null);
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
  const [mcpEnabled, setMcpEnabled] = useState(true);
  const [mcpAgents, setMcpAgents] = useState<Record<string, boolean>>({});
  const [bioChatModel, setBioChatModel] = useState('koesn/llama3-openbiollm-8b:latest');
  const [bioToolModel, setBioToolModel] = useState('qwen2.5:7b');
  const [bioMaxFold, setBioMaxFold] = useState('400');
  const [bioMaxAnalyze, setBioMaxAnalyze] = useState('10000');
  const [bioEsmfoldModel, setBioEsmfoldModel] = useState('facebook/esmfold_v1');
  const [bioArtifactsDir, setBioArtifactsDir] = useState('');
  const [bioSecondaryToolsPath, setBioSecondaryToolsPath] = useState('');
  const [bioPythonExecutable, setBioPythonExecutable] = useState('python3');
  const [bioCumulativeQCDir, setBioCumulativeQCDir] = useState('');
  const [bioDefaultPanelProfile, setBioDefaultPanelProfile] = useState('human-inflammatory-12plex-v1');
  const [cadOpenSCADPath, setCadOpenSCADPath] = useState('openscad');
  const [cadFreeCADPath, setCadFreeCADPath] = useState('');
  const [cadArtifactsDir, setCadArtifactsDir] = useState('');
  const [cadRenderTimeout, setCadRenderTimeout] = useState('120');
  const [cadChatModel, setCadChatModel] = useState('qwen2.5-coder:14b');
  const [cadToolModel, setCadToolModel] = useState('qwen2.5:7b');
  const [cadSettingsSaving, setCadSettingsSaving] = useState(false);
  const [cadSettingsErr, setCadSettingsErr] = useState<string | null>(null);
  const [cadSettingsOk, setCadSettingsOk] = useState<string | null>(null);
  const [cadTestResult, setCadTestResult] = useState<string | null>(null);
  const [bioSettingsSaving, setBioSettingsSaving] = useState(false);
  const [bioSettingsErr, setBioSettingsErr] = useState<string | null>(null);
  const [bioSettingsOk, setBioSettingsOk] = useState<string | null>(null);
  const [personalLearningEnabled, setPersonalLearningEnabled] = useState(false);
  const [personalLearningSuggestEnabled, setPersonalLearningSuggestEnabled] = useState(false);
  const [conversationMemoryEnabled, setConversationMemoryEnabled] = useState(true);
  const [conversationMemorySaving, setConversationMemorySaving] = useState(false);
  const [personalLearningSaving, setPersonalLearningSaving] = useState(false);
  const [personalLearningsOpen, setPersonalLearningsOpen] = useState(false);
  const [allLearnings, setAllLearnings] = useState<UserLearning[]>([]);
  const [allLearningsLoading, setAllLearningsLoading] = useState(false);
  const [allLearningsErr, setAllLearningsErr] = useState<string | null>(null);

  const refreshDomainPacks = async () => {
    setPacksLoading(true);
    setPacksErr(null);
    try {
      const api = new ChatAPI(hubHttp);
      const data = await api.fetchPacks();
      usePacksStore.getState().applyPacksResponse(data);
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

  const refreshGoogleMeetNotesStatus = async () => {
    setGoogleMeetNotesLoading(true);
    try {
      const api = new ChatAPI(hubHttp);
      const [status, appConfig] = await Promise.all([
        api.getGoogleMeetNotesStatus(),
        api.getGoogleMeetNotesAppConfig().catch(() => null),
      ]);
      setGoogleMeetNotes({
        ...status,
        connect_ready: status.connect_ready ?? appConfig?.connect_ready ?? appConfig?.configured ?? false,
        oauth_source: status.oauth_source ?? appConfig?.oauth_source,
        oauth_configured: status.oauth_configured || appConfig?.configured === true,
      });
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
    void refreshSlackIntegration();
  }, [isOpen, activeTab, hubHttp]);

  const loadSlackChannels = async () => {
    setSlackChannelsLoading(true);
    setSlackChannelsError(null);
    try {
      const api = new ChatAPI(hubHttp);
      const channels = await api.getSlackChannels();
      const sorted = [...channels].sort((a, b) => a.name.localeCompare(b.name));
      setSlackChannels(sorted);
      if (sorted.length === 0) {
        setSlackChannelsError(
          'No channels found — invite the bot with /invite @YourBot in each channel first.'
        );
      }
    } catch (e) {
      setSlackChannels([]);
      setSlackChannelsError(e instanceof Error ? e.message : 'Failed to load Slack channels');
    } finally {
      setSlackChannelsLoading(false);
    }
  };

  const refreshSlackIntegration = async () => {
    setSlackLoading(true);
    try {
      const api = new ChatAPI(hubHttp);
      const [status, cfg, bindings, connection, inbox] = await Promise.all([
        api.getSlackStatus(),
        api.getSlackConfig(),
        api.getSlackBindings(),
        api.getSlackConnection(),
        api.getSlackInbox().catch(() => defaultSlackInboxForm()),
      ]);
      setSlackStatus(status);
      setSlackConfig(cfg);
      setSlackBindings(bindings);
      setSlackConnection(connection);
      setSlackInbox(mergeSlackInboxForm(inbox));
      if (slackCanListChannelsFrom(status, cfg)) {
        void loadSlackChannels();
      } else {
        setSlackChannels([]);
        setSlackChannelsError(null);
      }
      const defaultPolicy = (cfg.default_policy || 'mention_only') as SlackPolicy;
      setSlackForm((prev) => ({
        ...prev,
        enabled: cfg.enabled,
        displayName: cfg.display_name || 'Camron',
        defaultPolicy,
        clientId: cfg.oauth?.client_id ?? prev.clientId,
        redirectUrl: cfg.oauth?.redirect_url || `${hubHttp}/api/slack/oauth/callback`,
      }));
      // New binding form: default to hub slack.default_policy (often "always"), not hardcoded mention_only.
      setSlackBindingForm((prev) =>
        prev.slackChannelId.trim() ? prev : { ...prev, policy: defaultPolicy }
      );
    } catch (e) {
      setTestResults((prev) => ({
        ...prev,
        slack: {
          success: false,
          message: e instanceof Error ? e.message : 'Failed to load Slack settings',
        },
      }));
    } finally {
      setSlackLoading(false);
    }
  };

  const saveSlackSettings = async () => {
    setSlackBusy(true);
    try {
      const api = new ChatAPI(hubHttp);
      await api.saveSlackConfig({
        enabled: slackForm.enabled,
        app_token: slackForm.appToken || undefined,
        bot_token: slackForm.botToken || undefined,
        display_name: slackForm.displayName,
        default_policy: slackForm.defaultPolicy,
        client_id: slackForm.clientId || undefined,
        client_secret: slackForm.clientSecret || undefined,
        redirect_url: slackForm.redirectUrl || undefined,
      });
      await api.restartSlackBridge();
      setSlackForm((prev) => ({ ...prev, appToken: '', botToken: '', clientSecret: '' }));
      await refreshSlackIntegration();
      setTestResults((prev) => ({
        ...prev,
        slack: { success: true, message: 'Slack settings saved and bridge restarted.' },
      }));
    } catch (e) {
      setTestResults((prev) => ({
        ...prev,
        slack: {
          success: false,
          message: e instanceof Error ? e.message : 'Failed to save Slack settings',
        },
      }));
    } finally {
      setSlackBusy(false);
    }
  };

  const pollSlackConnectionAfterOAuth = async (api: ChatAPI) => {
    const deadline = Date.now() + 60_000;
    while (Date.now() < deadline) {
      await new Promise((r) => setTimeout(r, 1500));
      try {
        const conn = await api.getSlackConnection();
        setSlackConnection(conn);
        if (conn.bot_token_set) {
          await api.restartSlackBridge();
          await refreshSlackIntegration();
          setTestResults((prev) => ({
            ...prev,
            slack: {
              success: true,
              message: conn.team_name
                ? `Connected to ${conn.team_name}. Bridge is starting.`
                : 'Slack connected. Bridge is starting.',
            },
          }));
          return;
        }
      } catch {
        /* keep polling */
      }
    }
    setTestResults((prev) => ({
      ...prev,
      slack: {
        success: false,
        message: 'OAuth window closed or timed out. Click Refresh or try Connect Slack again.',
      },
    }));
  };

  const connectSlackOAuth = async () => {
    setSlackBusy(true);
    try {
      const api = new ChatAPI(hubHttp);
      const url = await api.getSlackOAuthURL();
      openLink(url);
      setTestResults((prev) => ({
        ...prev,
        slack: {
          success: true,
          message: 'Complete Slack authorization in your browser…',
        },
      }));
      void pollSlackConnectionAfterOAuth(api);
    } catch (e) {
      setTestResults((prev) => ({
        ...prev,
        slack: {
          success: false,
          message: e instanceof Error ? e.message : 'Slack OAuth failed',
        },
      }));
    } finally {
      setSlackBusy(false);
    }
  };

  const saveSlackDisplaySettings = async () => {
    setSlackBusy(true);
    try {
      const api = new ChatAPI(hubHttp);
      await api.saveSlackConfig({
        enabled: slackForm.enabled,
        display_name: slackForm.displayName,
        default_policy: slackForm.defaultPolicy,
      });
      await refreshSlackIntegration();
      setTestResults((prev) => ({
        ...prev,
        slack: { success: true, message: 'Slack display settings saved.' },
      }));
    } catch (e) {
      setTestResults((prev) => ({
        ...prev,
        slack: {
          success: false,
          message: e instanceof Error ? e.message : 'Failed to save Slack settings',
        },
      }));
    } finally {
      setSlackBusy(false);
    }
  };

  const disconnectSlack = async () => {
    setSlackBusy(true);
    try {
      const api = new ChatAPI(hubHttp);
      await api.disconnectSlack();
      await refreshSlackIntegration();
      setTestResults((prev) => ({
        ...prev,
        slack: { success: true, message: 'Slack disconnected.' },
      }));
    } catch (e) {
      setTestResults((prev) => ({
        ...prev,
        slack: {
          success: false,
          message: e instanceof Error ? e.message : 'Disconnect failed',
        },
      }));
    } finally {
      setSlackBusy(false);
    }
  };

  const saveSlackBinding = async () => {
    if (!slackBindingForm.slackChannelId.trim() || !slackBindingForm.agentId) {
      setTestResults((prev) => ({
        ...prev,
        slack: { success: false, message: 'Slack channel ID and agent are required.' },
      }));
      return;
    }
    setSlackBusy(true);
    try {
      const api = new ChatAPI(hubHttp);
      const agent = agents.find((a) => a.id === slackBindingForm.agentId);
      await api.saveSlackBinding({
        slack_channel_id: slackBindingForm.slackChannelId.trim(),
        slack_channel_name: slackBindingForm.slackChannelName.trim() || undefined,
        agent_id: slackBindingForm.agentId,
        agent_name: agent?.name,
        policy: slackBindingForm.policy,
        enabled: true,
      });
      await refreshSlackIntegration();
      const channelList = await api.fetchChannels();
      setChannels(channelList);
      setTestResults((prev) => ({
        ...prev,
        slack: { success: true, message: 'Channel binding saved.' },
      }));
    } catch (e) {
      setTestResults((prev) => ({
        ...prev,
        slack: {
          success: false,
          message: e instanceof Error ? e.message : 'Failed to save binding',
        },
      }));
    } finally {
      setSlackBusy(false);
    }
  };

  const deleteSlackBindingRow = async (slackChannelId: string) => {
    setSlackBusy(true);
    try {
      const api = new ChatAPI(hubHttp);
      await api.deleteSlackBinding(slackChannelId);
      await refreshSlackIntegration();
      const channelList = await api.fetchChannels();
      setChannels(channelList);
    } catch (e) {
      setTestResults((prev) => ({
        ...prev,
        slack: {
          success: false,
          message: e instanceof Error ? e.message : 'Failed to delete binding',
        },
      }));
    } finally {
      setSlackBusy(false);
    }
  };

  const saveSlackInboxSettings = async () => {
    setSlackBusy(true);
    try {
      const api = new ChatAPI(hubHttp);
      const agent = agents.find((a) => a.id === slackInbox.agent_id);
      const mentionIds =
        slackInbox.forward_rules?.find((r) => r.type === 'mention_of_me')?.slack_channel_ids ?? [];
      const forwardRules = (slackInbox.forward_rules ?? []).map((r) =>
        r.type === 'reaction' && mentionIds.length > 0
          ? { ...r, slack_channel_ids: mentionIds }
          : r
      );
      const payload: SlackInboxConfig = {
        ...slackInbox,
        forward_rules: forwardRules,
        agent_name: agent?.name,
      };
      await api.saveSlackInbox(payload);
      await refreshSlackIntegration();
      window.dispatchEvent(new Event('nj-slack-inbox-updated'));
      setTestResults((prev) => ({
        ...prev,
        slack: { success: true, message: 'Personal inbox saved.' },
      }));
    } catch (e) {
      setTestResults((prev) => ({
        ...prev,
        slack: {
          success: false,
          message: e instanceof Error ? e.message : 'Failed to save personal inbox',
        },
      }));
    } finally {
      setSlackBusy(false);
    }
  };

  const testSlackInboxDM = async () => {
    setSlackBusy(true);
    try {
      const api = new ChatAPI(hubHttp);
      await api.testSlackInboxDM();
      setTestResults((prev) => ({
        ...prev,
        slack: { success: true, message: 'Test DM sent — check Slack.' },
      }));
    } catch (e) {
      setTestResults((prev) => ({
        ...prev,
        slack: {
          success: false,
          message: e instanceof Error ? e.message : 'Inbox test DM failed',
        },
      }));
    } finally {
      setSlackBusy(false);
    }
  };

  const authorizeSlackUserDM = async () => {
    setSlackBusy(true);
    try {
      const api = new ChatAPI(hubHttp);
      const url = await api.getSlackUserDMOAuthURL();
      openLink(url);
      setTestResults((prev) => ({
        ...prev,
        slack: {
          success: true,
          message: 'Authorize Slack DM access in your browser…',
        },
      }));
      window.setTimeout(async () => {
        try {
          const inbox = await api.getSlackInbox();
          setSlackInbox(mergeSlackInboxForm(inbox));
        } catch {
          /* ignore poll errors */
        }
      }, 4000);
    } catch (e) {
      setTestResults((prev) => ({
        ...prev,
        slack: {
          success: false,
          message: e instanceof Error ? e.message : 'User DM authorization failed',
        },
      }));
    } finally {
      setSlackBusy(false);
    }
  };

  const humanDMStatusLabel = (status?: string) => {
    switch (status) {
      case 'monitoring_active':
        return 'Monitoring active';
      case 'not_authorized':
        return 'Not authorized — click Authorize Slack DM access';
      case 'inside_work_hours':
        return 'Inside work hours (schedule)';
      case 'away_off':
        return 'Away mode off';
      case 'inbox_not_ready':
        return 'Enable personal inbox and pick an agent first';
      case 'disabled':
        return 'Disabled';
      default:
        return status ?? '';
    }
  };

  const toggleMentionWatchChannel = (channelId: string) => {
    setSlackInbox((prev) => {
      const rules = prev.forward_rules ?? [];
      const mention = rules.find((r) => r.id === 'mentions' || r.type === 'mention_of_me');
      const ids = new Set(mention?.slack_channel_ids ?? []);
      if (ids.has(channelId)) ids.delete(channelId);
      else ids.add(channelId);
      return updateForwardRule(prev, mention?.id ?? 'mentions', {
        slack_channel_ids: Array.from(ids),
      });
    });
  };

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

  const googleOAuthSourceLabel = (source?: string) => {
    switch (source) {
      case 'vendor':
        return 'Using Neural Junkie Google app';
      case 'env':
        return 'Using environment Google OAuth config';
      case 'config':
        return 'Using custom Google OAuth client';
      default:
        return 'Google OAuth unavailable';
    }
  };

  const googleConnectReady =
    googleMeetNotes?.connect_ready ?? googleMeetNotes?.oauth_configured ?? false;

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
          setMcpEnabled(cfg.mcp?.enabled !== false);
          setMcpAgents(
            cfg.mcp?.agents && typeof cfg.mcp.agents === 'object'
              ? (cfg.mcp.agents as Record<string, boolean>)
              : {}
          );
          const bio = cfg.mcp?.biology ?? {};
          setBioChatModel(bio.chat_model || 'koesn/llama3-openbiollm-8b:latest');
          setBioToolModel(bio.tool_model || 'qwen2.5:7b');
          setBioMaxFold(String(bio.max_fold_length || 400));
          setBioMaxAnalyze(String(bio.max_analyze_length || 10000));
          setBioEsmfoldModel(bio.esmfold_model || 'facebook/esmfold_v1');
          setBioArtifactsDir(bio.artifacts_dir || '');
          setBioSecondaryToolsPath(bio.secondary_analysis_tools_path || '');
          setBioPythonExecutable(bio.python_executable || 'python3');
          setBioCumulativeQCDir(bio.cumulative_qc_dir || '');
          setBioDefaultPanelProfile(bio.default_panel_profile || 'human-inflammatory-12plex-v1');
          const cadCfg = cfg.mcp?.cad ?? {};
          setCadOpenSCADPath(cadCfg.openscad_path || 'openscad');
          setCadFreeCADPath(cadCfg.freecad_path || '');
          setCadArtifactsDir(cadCfg.artifacts_dir || '');
          setCadRenderTimeout(String(cadCfg.render_timeout_sec || 120));
          setCadChatModel(cadCfg.chat_model || 'qwen2.5-coder:14b');
          setCadToolModel(cadCfg.tool_model || 'qwen2.5:7b');
          const agentRows = Array.isArray(cfg.agents) ? cfg.agents : [];
          setConfiguredAgents(
            agentRows.filter((a: { enabled?: boolean }) => a.enabled !== false)
          );
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
  }, [isOpen, activeTab, hubHttp]);

  useEffect(() => {
    if (!isOpen || activeTab !== 'ai-providers' || !personalLearningEnabled || !hasPersonalLearning) {
      return;
    }
    void refreshAllLearnings();
  }, [isOpen, activeTab, personalLearningEnabled, hasPersonalLearning, hubHttp]);

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

  useEffect(() => {
    if (activeTab !== 'ai-providers') return;
    let cancelled = false;
    void fetchHardwareSnapshot(hubHttp).then((snap) => {
      if (!cancelled) setHardwareSnapshot(snap);
    });
    return () => {
      cancelled = true;
    };
  }, [activeTab, hubHttp]);

  useEffect(() => {
    if (activeTab !== 'ai-providers') return;
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
  }, [activeTab, hubHttp, ollamaForm.defaultModel]);

  useShortcutOverlay('settings', isOpen, onClose);

  if (!isOpen) return null;

  const handleFontSizeChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    updateFontSize(parseInt(e.target.value));
  };

  const handleScopeChange = (scope: FontSizeScope) => {
    updateFontSizeScope(scope);
  };

  const handleColorThemeChange = (theme: ColorTheme) => {
    updateColorTheme(theme);
  };

  const activeColorTheme: ColorTheme = settings.colorTheme ?? 'slack';

  const handleConversationMemoryToggle = async (enabled: boolean) => {
    setConversationMemorySaving(true);
    setCollabRoutingErr(null);
    try {
      await mergeSettingsPut((cfg) => ({
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
      await mergeSettingsPut((cfg) => ({
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
      await mergeSettingsPut((cfg) => ({
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

  const saveConfiguredAgentModels = async () => {
    setAgentModelsSaving(true);
    setAgentModelsErr(null);
    setAgentModelsOk(null);
    try {
      await mergeSettingsPut((cfg) => {
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
            chat_model: bioChatModel.trim() || 'koesn/llama3-openbiollm-8b:latest',
            tool_model: bioToolModel.trim() || 'qwen2.5:7b',
            esmfold_model: bioEsmfoldModel.trim() || 'facebook/esmfold_v1',
            max_fold_length: maxFold,
            max_analyze_length: maxAnalyze,
            artifacts_dir: bioArtifactsDir.trim(),
            secondary_analysis_tools_path: bioSecondaryToolsPath.trim(),
            python_executable: bioPythonExecutable.trim() || 'python3',
            cumulative_qc_dir: bioCumulativeQCDir.trim(),
            default_panel_profile: bioDefaultPanelProfile.trim() || 'human-inflammatory-12plex-v1',
          },
        },
      }));
      setBioSettingsOk('Life sciences settings saved. Restart BiologyExpert if it is already running.');
    } catch (e) {
      setBioSettingsErr(e instanceof Error ? e.message : String(e));
    } finally {
      setBioSettingsSaving(false);
    }
  };

  const saveCadMcpSettings = async () => {
    setCadSettingsSaving(true);
    setCadSettingsErr(null);
    setCadSettingsOk(null);
    try {
      const timeout = parseInt(cadRenderTimeout, 10);
      if (!Number.isFinite(timeout) || timeout <= 0) {
        throw new Error('Render timeout must be a positive integer');
      }
      await mergeSettingsPut((cfg) => ({
        ...cfg,
        mcp: {
          ...(cfg.mcp as object | undefined),
          enabled: mcpEnabled,
          cad: {
            openscad_path: cadOpenSCADPath.trim() || 'openscad',
            freecad_path: cadFreeCADPath.trim(),
            artifacts_dir: cadArtifactsDir.trim(),
            render_timeout_sec: timeout,
            chat_model: cadChatModel.trim() || 'qwen2.5-coder:14b',
            tool_model: cadToolModel.trim() || 'qwen2.5:7b',
          },
        },
      }));
      setCadSettingsOk('CAD tool settings saved. Restart CADExpert if it is already running.');
    } catch (e) {
      setCadSettingsErr(e instanceof Error ? e.message : String(e));
    } finally {
      setCadSettingsSaving(false);
    }
  };

  const testCadOpenSCAD = async () => {
    setCadTestResult(null);
    try {
      const api = new ChatAPI(hubHttp);
      const res = await api.testOpenSCAD(cadOpenSCADPath.trim() || undefined);
      setCadTestResult(res.ok ? res.message : `Failed: ${res.message}`);
    } catch (e) {
      setCadTestResult(e instanceof Error ? e.message : String(e));
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

  const handleMcpAgentToggle = async (agentKey: string, enabled: boolean) => {
    setMcpAgents((prev) => ({ ...prev, [agentKey]: enabled }));
    try {
      await mergeSettingsPut((cfg) => {
        const mcp = (cfg.mcp ?? {}) as Record<string, unknown>;
        const prevAgents =
          mcp.agents && typeof mcp.agents === 'object'
            ? (mcp.agents as Record<string, boolean>)
            : {};
        return {
          ...cfg,
          mcp: {
            ...mcp,
            enabled: mcp.enabled !== false,
            agents: { ...prevAgents, [agentKey]: enabled },
          },
        };
      });
    } catch (e) {
      setMcpAgents((prev) => ({ ...prev, [agentKey]: !enabled }));
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

  const addUnique = (items: string[] | undefined, value: string): string[] => {
    const cur = items ?? [];
    return cur.includes(value) ? cur : [...cur, value];
  };

  const removeItem = (items: string[] | undefined, value: string): string[] =>
    (items ?? []).filter((x) => x !== value);

  const findAgentForDmChannel = (channelName: string) => {
    const ch = channels.find((c) => c.name === channelName);
    if (!ch) return undefined;
    const agentId = ch.agents?.[0]?.id ?? ch.members?.[0];
    return (
      (agentId ? agents.find((a) => a.id === agentId) : undefined) ??
      agents.find((a) => a.name.toLowerCase() === parseDMDisplayName(ch).toLowerCase())
    );
  };

  const unhideDmChannel = (name: string) => {
    const agent = findAgentForDmChannel(name);
    const key = agent ? agentSidebarHideKey(agent) : '';
    void updateSettings({
      hiddenDmChannelNames: removeItem(settings.hiddenDmChannelNames, name),
      deletedDmChannelNames: removeItem(settings.deletedDmChannelNames, name),
      ...(agent
        ? {
            hiddenAgentIdsForSidebar: removeItem(settings.hiddenAgentIdsForSidebar, agent.id),
            hiddenAgentSidebarKeys: removeItem(settings.hiddenAgentSidebarKeys, key),
            deletedAgentIdsForSidebar: removeItem(settings.deletedAgentIdsForSidebar, agent.id),
            deletedAgentSidebarKeys: removeItem(settings.deletedAgentSidebarKeys, key),
          }
        : {}),
    });
  };

  const deleteHiddenDmChannel = (name: string) => {
    const agent = findAgentForDmChannel(name);
    const key = agent ? agentSidebarHideKey(agent) : '';
    void updateSettings({
      hiddenDmChannelNames: removeItem(settings.hiddenDmChannelNames, name),
      deletedDmChannelNames: addUnique(settings.deletedDmChannelNames, name),
      ...(agent
        ? {
            hiddenAgentIdsForSidebar: removeItem(settings.hiddenAgentIdsForSidebar, agent.id),
            hiddenAgentSidebarKeys: removeItem(settings.hiddenAgentSidebarKeys, key),
            deletedAgentIdsForSidebar: addUnique(settings.deletedAgentIdsForSidebar, agent.id),
            deletedAgentSidebarKeys: addUnique(settings.deletedAgentSidebarKeys, key),
          }
        : {}),
    });
  };

  const unhideCollaborationChannel = (name: string) => {
    void updateSettings({
      hiddenCollaborationChannelNames: removeItem(settings.hiddenCollaborationChannelNames, name),
      deletedCollaborationChannelNames: removeItem(settings.deletedCollaborationChannelNames, name),
    });
  };

  const deleteHiddenCollaborationChannel = (name: string) => {
    void updateSettings({
      hiddenCollaborationChannelNames: removeItem(settings.hiddenCollaborationChannelNames, name),
      deletedCollaborationChannelNames: addUnique(settings.deletedCollaborationChannelNames, name),
    });
  };

  const unhideAgentShortcutKey = (key: string) => {
    void updateSettings({
      hiddenAgentSidebarKeys: removeItem(settings.hiddenAgentSidebarKeys, key),
      deletedAgentSidebarKeys: removeItem(settings.deletedAgentSidebarKeys, key),
    });
  };

  const deleteHiddenAgentShortcutKey = (key: string) => {
    void updateSettings({
      hiddenAgentSidebarKeys: removeItem(settings.hiddenAgentSidebarKeys, key),
      deletedAgentSidebarKeys: addUnique(settings.deletedAgentSidebarKeys, key),
    });
  };

  const unhideAgentShortcutId = (id: string) => {
    const agent = agents.find((a) => a.id === id);
    const key = agent ? agentSidebarHideKey(agent) : '';
    void updateSettings({
      hiddenAgentIdsForSidebar: removeItem(settings.hiddenAgentIdsForSidebar, id),
      deletedAgentIdsForSidebar: removeItem(settings.deletedAgentIdsForSidebar, id),
      ...(agent
        ? {
            hiddenAgentSidebarKeys: removeItem(settings.hiddenAgentSidebarKeys, key),
            deletedAgentSidebarKeys: removeItem(settings.deletedAgentSidebarKeys, key),
          }
        : {}),
    });
  };

  const deleteHiddenAgentShortcutId = (id: string) => {
    const agent = agents.find((a) => a.id === id);
    const key = agent ? agentSidebarHideKey(agent) : '';
    void updateSettings({
      hiddenAgentIdsForSidebar: removeItem(settings.hiddenAgentIdsForSidebar, id),
      deletedAgentIdsForSidebar: addUnique(settings.deletedAgentIdsForSidebar, id),
      ...(agent
        ? {
            hiddenAgentSidebarKeys: removeItem(settings.hiddenAgentSidebarKeys, key),
            deletedAgentSidebarKeys: addUnique(settings.deletedAgentSidebarKeys, key),
          }
        : {}),
    });
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
    <div className="fixed inset-0 z-50 p-4" role="presentation">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/50"
        onClick={onClose}
        aria-hidden
      />

      {/* Settings shell — near-full-screen overlay with sidebar nav */}
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="nj-settings-title"
        className="relative z-10 mx-auto flex h-full w-full max-w-[1600px] flex-col overflow-hidden rounded-xl border border-slack-border bg-slack-bg shadow-2xl"
      >
        {/* Header */}
        <div className="flex shrink-0 items-center justify-between border-b border-slack-border px-6 py-4">
          <h2 id="nj-settings-title" className="text-xl font-bold text-slack-text">
            Settings
          </h2>
          <button
            type="button"
            onClick={onClose}
            aria-label="Close settings"
            className="text-slack-textMuted transition-colors hover:text-slack-text"
          >
            <svg className="h-6 w-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <div className="flex min-h-0 flex-1">
          {/* Sidebar nav */}
          <nav
            className="flex w-[220px] shrink-0 flex-col overflow-y-auto border-r border-slack-border bg-slack-bgHover/20 py-2"
            aria-label="Settings sections"
          >
            {SETTINGS_NAV.map((tab) => (
              <button
                key={tab.id}
                type="button"
                onClick={() => setActiveTab(tab.id)}
                className={`border-l-2 px-4 py-2.5 text-left text-sm font-medium transition-colors ${
                  activeTab === tab.id
                    ? 'border-slack-accent bg-slack-bgHover text-slack-text'
                    : 'border-transparent text-slack-textMuted hover:bg-slack-bgHover/50 hover:text-slack-text'
                }`}
              >
                {tab.label}
              </button>
            ))}
          </nav>

          {/* Content */}
          <div className="min-w-0 flex-1 overflow-y-auto p-6">
          {activeTab === 'appearance' && (
            <div className="space-y-6">
              {/* Color theme */}
              <div>
                <label className="block text-sm font-medium text-slack-text mb-3">
                  Color theme
                </label>
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                  {(
                    [
                      {
                        id: 'slack' as const,
                        label: 'Slack',
                        description: 'Classic blue-accent dark UI',
                        previewBg: '#1a1d21',
                        previewAccent: '#1164a3',
                      },
                      {
                        id: 'flat' as const,
                        label: 'Flat',
                        description: 'Navy base, purple accent, mint status',
                        previewBg: '#0d0d1a',
                        previewAccent: '#a78bfa',
                      },
                    ] as const
                  ).map((option) => (
                    <label
                      key={option.id}
                      className={`flex cursor-pointer flex-col rounded-lg border p-3 transition-colors ${
                        activeColorTheme === option.id
                          ? 'border-slack-accent bg-slack-accent/10 ring-1 ring-slack-accent'
                          : 'border-slack-border bg-slack-bgHover hover:border-slack-textMuted'
                      }`}
                    >
                      <input
                        type="radio"
                        name="colorTheme"
                        value={option.id}
                        checked={activeColorTheme === option.id}
                        onChange={() => handleColorThemeChange(option.id)}
                        className="sr-only"
                      />
                      <div className="mb-2 flex items-center gap-2">
                        <span
                          className="h-8 w-8 shrink-0 rounded border border-white/10"
                          style={{ backgroundColor: option.previewBg }}
                          aria-hidden
                        />
                        <span
                          className="h-8 flex-1 rounded"
                          style={{ backgroundColor: option.previewAccent }}
                          aria-hidden
                        />
                      </div>
                      <div className="text-sm font-medium text-slack-text">{option.label}</div>
                      <div className="text-xs text-slack-textMuted mt-0.5">{option.description}</div>
                    </label>
                  ))}
                </div>
              </div>

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
                <h3 className="text-lg font-semibold text-slack-text mb-2">Layout preset</h3>
                <p className="text-sm text-slack-textMuted mb-3">
                  IDE puts the editor and agent first; Team keeps the classic chat-first workspace.
                </p>
                <div className="flex gap-2">
                  {(['team', 'ide'] as const).map((preset) => (
                    <button
                      key={preset}
                      type="button"
                      onClick={() => {
                        void import('../utils/layoutPresets').then(({ panelsForPreset }) =>
                          updateLayoutSettings(panelsForPreset(preset))
                        );
                      }}
                      className={`px-3 py-1.5 text-sm rounded border ${
                        (layoutSettings.layoutPreset ?? 'team') === preset
                          ? 'border-slack-accent bg-slack-accent/20 text-slack-text'
                          : 'border-slack-border text-slack-textMuted hover:text-slack-text'
                      }`}
                    >
                      {preset === 'ide' ? 'IDE (project-first)' : 'Team (chat-first)'}
                    </button>
                  ))}
                </div>
              </div>

              <div className="mb-4">
                <h3 className="text-lg font-semibold text-slack-text mb-2">Toolbar actions</h3>
                <p className="text-sm text-slack-textMuted mb-3">
                  On wide screens, show toolbar chips in the top bar or a right sidebar. Narrow windows
                  always use the sidebar.
                </p>
                <div className="flex gap-2">
                  {(['top', 'sidebar'] as const).map((placement) => (
                    <button
                      key={placement}
                      type="button"
                      onClick={() => updateLayoutSettings({ toolbarChipsPlacement: placement })}
                      className={`px-3 py-1.5 text-sm rounded border ${
                        (layoutSettings.toolbarChipsPlacement ?? 'top') === placement
                          ? 'border-slack-accent bg-slack-accent/20 text-slack-text'
                          : 'border-slack-border text-slack-textMuted hover:text-slack-text'
                      }`}
                    >
                      {placement === 'top' ? 'Top bar' : 'Right sidebar'}
                    </button>
                  ))}
                </div>
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

          {activeTab === 'keyboard' && (
            <div className="space-y-4">
              <div>
                <h3 className="text-lg font-semibold text-slack-text mb-2">Keyboard shortcuts</h3>
                <p className="text-sm text-slack-textMuted mb-4">
                  Fixed defaults (not customizable yet). Toolbar buttons show the same chords on hover.
                </p>
              </div>
              <div className="border border-slack-border rounded-lg overflow-hidden">
                <table className="w-full text-sm">
                  <thead className="bg-slack-bgHover/50 text-left text-slack-textMuted">
                    <tr>
                      <th className="px-4 py-2 font-medium">Action</th>
                      <th className="px-4 py-2 font-medium w-40">Shortcut</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slack-border">
                    {getShortcutsForDisplay().map((row) => (
                      <tr key={row.id} className="text-slack-text">
                        <td className="px-4 py-2">{row.label}</td>
                        <td className="px-4 py-2">
                          <kbd className="rounded bg-slack-bgHover px-1.5 py-0.5 font-mono text-xs">
                            {formatChord(row.chord)}
                          </kbd>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
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
                  DM rows, collaborations, or agent shortcuts you hid. Use Unhide to restore them, or Delete to keep them off the sidebar permanently.
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
                        <div className="shrink-0 flex items-center gap-3">
                          <button
                            type="button"
                            className="text-xs text-slack-accent hover:underline"
                            onClick={() => unhideDmChannel(name)}
                          >
                            Unhide
                          </button>
                          <button
                            type="button"
                            className="text-xs text-red-400 hover:text-red-300 hover:underline"
                            onClick={() => deleteHiddenDmChannel(name)}
                          >
                            Delete
                          </button>
                        </div>
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
                          <div className="shrink-0 flex items-center gap-3">
                            <button
                              type="button"
                              className="text-xs text-slack-accent hover:underline"
                              onClick={() => unhideCollaborationChannel(name)}
                            >
                              Unhide
                            </button>
                            <button
                              type="button"
                              className="text-xs text-red-400 hover:text-red-300 hover:underline"
                              onClick={() => deleteHiddenCollaborationChannel(name)}
                            >
                              Delete
                            </button>
                          </div>
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
                          <div className="shrink-0 flex items-center gap-3">
                            <button
                              type="button"
                              className="text-xs text-slack-accent hover:underline"
                              onClick={() => unhideAgentShortcutKey(key)}
                            >
                              Unhide
                            </button>
                            <button
                              type="button"
                              className="text-xs text-red-400 hover:text-red-300 hover:underline"
                              onClick={() => deleteHiddenAgentShortcutKey(key)}
                            >
                              Delete
                            </button>
                          </div>
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
                        <div className="shrink-0 flex items-center gap-3">
                          <button
                            type="button"
                            className="text-xs text-slack-accent hover:underline"
                            onClick={() => unhideAgentShortcutId(id)}
                          >
                            Unhide
                          </button>
                          <button
                            type="button"
                            className="text-xs text-red-400 hover:text-red-300 hover:underline"
                            onClick={() => deleteHiddenAgentShortcutId(id)}
                          >
                            Delete
                          </button>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>
          )}

          {activeTab === 'integrations' && (
            <div className="space-y-8 nj-settings-integrations text-slack-text">
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
                  Connect your Google account to sync Gemini meeting notes from Gmail into Assistant.
                </p>
                <div
                  className={`mb-4 rounded border p-3 text-sm ${
                    googleConnectReady
                      ? 'border-green-200 bg-green-50 text-green-800'
                      : 'border-yellow-200 bg-yellow-50 text-yellow-800'
                  }`}
                >
                  {googleOAuthSourceLabel(googleMeetNotes?.oauth_source)}
                  {!googleConnectReady && (
                    <span className="block mt-1">
                      Use a release build with bundled credentials, set env vars, or configure
                      Advanced Google OAuth.
                    </span>
                  )}
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
                <div className="flex flex-wrap gap-2 mb-4">
                  <button
                    type="button"
                    onClick={() => void connectGoogleMeetNotes()}
                    disabled={googleMeetNotesBusy || !googleConnectReady}
                    className="px-4 py-2 bg-slack-accent text-white rounded hover:bg-slack-accentHover disabled:opacity-50"
                  >
                    Connect Google
                  </button>
                  <button
                    type="button"
                    onClick={() => void syncGoogleMeetNotesNow()}
                    disabled={googleMeetNotesBusy || !googleMeetNotes?.connected}
                    className="px-4 py-2 border border-slack-border rounded hover:bg-slack-bgHover text-slack-text disabled:opacity-50"
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
                {googleMeetNotesLoading && !googleMeetNotes ? (
                  <p className="text-sm text-slack-textMuted">Loading status…</p>
                ) : googleMeetNotes ? (
                  <div className="space-y-3 text-sm text-slack-text">
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
                <details
                  open={googleAdvancedOpen}
                  onToggle={(e) => setGoogleAdvancedOpen(e.currentTarget.open)}
                  className="mt-4 border border-slack-border rounded-lg p-4"
                >
                  <summary className="cursor-pointer text-sm font-medium text-slack-text">
                    Advanced (bring your own Google OAuth client)
                  </summary>
                  <div className="space-y-4 mt-4">
                    <p className="text-sm text-slack-textMuted">
                      Create a Google Cloud OAuth web client, add the redirect URI below, then save
                      your Client ID and Secret.
                    </p>
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
                        placeholder={
                          googleOAuthSecretSet ? 'Leave blank to keep existing secret' : 'Client secret'
                        }
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
                        Add this exact URI in Google Cloud Console → Credentials → your OAuth
                        client.
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
                </details>
              </div>

              {/* Slack bridge */}
              <div className="border border-slack-border rounded-lg p-6 mb-6">
                <div className="flex items-center justify-between mb-4">
                  <h3 className="text-lg font-semibold text-slack-text">Slack</h3>
                  <button
                    type="button"
                    onClick={() => void refreshSlackIntegration()}
                    disabled={slackLoading}
                    className="px-3 py-1 text-sm border border-slack-border rounded hover:bg-slack-bgHover text-slack-text"
                  >
                    Refresh
                  </button>
                </div>
                <p className="text-sm text-slack-textMuted mb-4">
                  Connect your workspace with one click. Assign one primary agent per Slack channel. Replies post as
                  the bot with display name <strong className="text-slack-text">on behalf of you</strong>.
                </p>
                {testResults.slack && (
                  <div
                    className={`mb-4 p-3 rounded text-sm ${
                      testResults.slack.success
                        ? 'bg-green-100 text-green-800 border border-green-200'
                        : 'bg-red-100 text-red-800 border border-red-200'
                    }`}
                  >
                    {testResults.slack.message}
                  </div>
                )}
                {slackLoading && !slackStatus ? (
                  <p className="text-sm text-slack-textMuted mb-4">Loading…</p>
                ) : (
                  <div className="space-y-2 text-sm text-slack-text mb-4 p-3 bg-slack-bgHover rounded border border-slack-border">
                    <p>
                      <span className="font-medium">Workspace:</span>{' '}
                      {slackConnection?.team_name ||
                        (slackConfig?.bot_token_set ? 'connected' : 'not connected')}
                    </p>
                    <p>
                      <span className="font-medium">Bridge:</span>{' '}
                      {slackConnection?.bridge_connected
                        ? 'connected'
                        : slackStatus?.configured
                          ? 'configured (starting…)'
                          : 'not connected'}
                    </p>
                    {slackStatus?.bot_user_id && (
                      <p>
                        <span className="font-medium">Bot user:</span> {slackStatus.bot_user_id}
                      </p>
                    )}
                    <p>
                      <span className="font-medium">Bindings:</span> {slackStatus?.bindings_count ?? 0}
                    </p>
                  </div>
                )}
                <div className="flex flex-wrap gap-2 mb-4">
                  <button
                    type="button"
                    onClick={() => void connectSlackOAuth()}
                    disabled={
                      slackBusy ||
                      !(slackConfig?.connect_ready ?? slackConfig?.oauth?.connect_ready)
                    }
                    className="px-4 py-2 bg-slack-accent text-white rounded hover:bg-slack-accentHover disabled:opacity-50"
                  >
                    Connect Slack
                  </button>
                  <button
                    type="button"
                    onClick={() => void disconnectSlack()}
                    disabled={slackBusy || !slackConfig?.bot_token_set}
                    className="px-4 py-2 text-red-600 border border-red-300 rounded hover:bg-red-50 disabled:opacity-50"
                  >
                    Disconnect
                  </button>
                </div>
                <div className="space-y-4 mb-4">
                  <label className="flex items-center gap-2 text-sm text-slack-text">
                    <input
                      type="checkbox"
                      checked={slackForm.enabled}
                      onChange={(e) =>
                        setSlackForm((prev) => ({ ...prev, enabled: e.target.checked }))
                      }
                    />
                    Enable Slack bridge
                  </label>
                  <div>
                    <label className="block text-sm font-medium text-slack-text mb-2">
                      Display name (on behalf of)
                    </label>
                    <input
                      type="text"
                      value={slackForm.displayName}
                      onChange={(e) =>
                        setSlackForm((prev) => ({ ...prev, displayName: e.target.value }))
                      }
                      className="w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-slack-text"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-slack-text mb-2">
                      Default trigger policy
                    </label>
                    <select
                      value={slackForm.defaultPolicy}
                      onChange={(e) =>
                        setSlackForm((prev) => ({
                          ...prev,
                          defaultPolicy: e.target.value as SlackPolicy,
                        }))
                      }
                      className="w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-slack-text"
                    >
                      <option value="mention_only">Mention only (@app)</option>
                      <option value="questions">Questions / requests</option>
                      <option value="always">Every message</option>
                    </select>
                  </div>
                  <button
                    type="button"
                    onClick={() => void saveSlackDisplaySettings()}
                    disabled={slackBusy}
                    className="w-full px-4 py-2 border border-slack-border rounded hover:bg-slack-bgHover text-slack-text disabled:opacity-50"
                  >
                    Save display settings
                  </button>
                </div>
                <details
                  className="mb-6 border border-slack-border rounded-lg"
                  open={slackAdvancedOpen}
                  onToggle={(e) => setSlackAdvancedOpen((e.target as HTMLDetailsElement).open)}
                >
                  <summary className="cursor-pointer px-4 py-3 text-sm font-medium text-slack-text hover:bg-slack-bgHover rounded-lg">
                    Advanced (bring your own Slack app)
                  </summary>
                  <div className="px-4 pb-4 space-y-4 border-t border-slack-border pt-4">
                    <p className="text-xs text-slack-textMuted">
                      Paste tokens or configure a custom OAuth app. See{' '}
                      <a
                        href="https://github.com/camronwood/neural-junkie/blob/main/docs/SLACK_INTEGRATION.md"
                        className="text-slack-accent hover:underline"
                        target="_blank"
                        rel="noreferrer"
                      >
                        SLACK_INTEGRATION.md
                      </a>{' '}
                      for enterprise / BYO setup.
                    </p>
                    <div>
                      <label className="block text-sm font-medium text-slack-text mb-2">
                        App token (Socket Mode, xapp-…)
                        {slackConfig?.app_token_set && !slackForm.appToken && (
                          <span className="ml-2 text-xs text-green-600">(saved)</span>
                        )}
                      </label>
                      <input
                        type="password"
                        value={slackForm.appToken}
                        onChange={(e) =>
                          setSlackForm((prev) => ({ ...prev, appToken: e.target.value }))
                        }
                        placeholder={slackConfig?.app_token_set ? 'Leave blank to keep' : 'xapp-…'}
                        className="w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-slack-text font-mono text-xs"
                      />
                    </div>
                    <div>
                      <label className="block text-sm font-medium text-slack-text mb-2">
                        Bot token (xoxb-…)
                        {slackConfig?.bot_token_set && !slackForm.botToken && (
                          <span className="ml-2 text-xs text-green-600">(saved)</span>
                        )}
                      </label>
                      <input
                        type="password"
                        value={slackForm.botToken}
                        onChange={(e) =>
                          setSlackForm((prev) => ({ ...prev, botToken: e.target.value }))
                        }
                        placeholder={slackConfig?.bot_token_set ? 'Leave blank to keep' : 'xoxb-…'}
                        className="w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-slack-text font-mono text-xs"
                      />
                    </div>
                    <div>
                      <label className="block text-sm font-medium text-slack-text mb-2">
                        OAuth client ID
                      </label>
                      <input
                        type="text"
                        value={slackForm.clientId}
                        onChange={(e) =>
                          setSlackForm((prev) => ({ ...prev, clientId: e.target.value }))
                        }
                        placeholder={slackConfig?.oauth?.client_id || 'Slack app client ID'}
                        className="w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-slack-text font-mono text-xs"
                      />
                    </div>
                    <div>
                      <label className="block text-sm font-medium text-slack-text mb-2">
                        OAuth client secret
                        {slackConfig?.oauth?.secret_set && !slackForm.clientSecret && (
                          <span className="ml-2 text-xs text-green-600">(saved)</span>
                        )}
                      </label>
                      <input
                        type="password"
                        value={slackForm.clientSecret}
                        onChange={(e) =>
                          setSlackForm((prev) => ({ ...prev, clientSecret: e.target.value }))
                        }
                        placeholder="Leave blank to keep existing secret"
                        className="w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-slack-text font-mono text-xs"
                      />
                    </div>
                    <div>
                      <label className="block text-sm font-medium text-slack-text mb-2">
                        OAuth redirect URI
                      </label>
                      <input
                        type="text"
                        value={slackForm.redirectUrl}
                        onChange={(e) =>
                          setSlackForm((prev) => ({ ...prev, redirectUrl: e.target.value }))
                        }
                        className="w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-slack-text font-mono text-xs"
                      />
                    </div>
                    <button
                      type="button"
                      onClick={() => void saveSlackSettings()}
                      disabled={slackBusy}
                      className="w-full px-4 py-2 bg-slack-accent text-white rounded hover:bg-slack-accentHover disabled:opacity-50"
                    >
                      Save advanced tokens &amp; restart bridge
                    </button>
                  </div>
                </details>
                <h4 className="text-sm font-semibold text-slack-text mb-2 mt-4">Personal inbox</h4>
                <p className="text-xs text-slack-textMuted mb-3">
                  DM the NJ bot from Slack while away. Forwarded channel messages get agent replies in the
                  original Slack thread.
                </p>
                <div className="p-4 mb-4 bg-slack-bgHover rounded-lg border border-slack-border space-y-3">
                  <div className="flex items-center justify-between gap-3">
                    <div>
                      <div className="font-medium text-slack-text">Enable personal inbox</div>
                      <div className="text-xs text-slack-textMuted">
                        Owner:{' '}
                        {slackInbox.owner_slack_user_name ||
                          slackConnection?.owner_slack_user_name ||
                          slackInbox.owner_slack_user_id ||
                          slackConnection?.owner_slack_user_id ||
                          'Connect Slack first'}
                      </div>
                    </div>
                    <input
                      type="checkbox"
                      checked={slackInbox.enabled}
                      onChange={(e) =>
                        setSlackInbox((prev) => ({ ...prev, enabled: e.target.checked }))
                      }
                      className="h-4 w-4 text-slack-accent"
                    />
                  </div>
                  <select
                    value={slackInbox.agent_id ?? ''}
                    onChange={(e) =>
                      setSlackInbox((prev) => ({ ...prev, agent_id: e.target.value }))
                    }
                    className="w-full px-3 py-2 bg-slack-bg border border-slack-border rounded text-sm text-slack-text"
                  >
                    <option value="" className="bg-slack-bg text-slack-text">
                      Select inbox agent…
                    </option>
                    {agents.map((a) => (
                      <option key={a.id} value={a.id} className="bg-slack-bg text-slack-text">
                        {a.name} ({a.type})
                      </option>
                    ))}
                  </select>
                  <div className="flex flex-wrap gap-2">
                    <button
                      type="button"
                      onClick={() => void saveSlackInboxSettings()}
                      disabled={slackBusy}
                      className="px-4 py-2 bg-slack-accent text-white rounded hover:bg-slack-accentHover disabled:opacity-50 text-sm"
                    >
                      Save personal inbox
                    </button>
                    <button
                      type="button"
                      onClick={() => void testSlackInboxDM()}
                      disabled={slackBusy || !slackInbox.enabled}
                      className="px-4 py-2 border border-slack-border rounded hover:bg-slack-bg text-sm text-slack-text disabled:opacity-50"
                    >
                      Test DM
                    </button>
                  </div>
                  <div className="border-t border-slack-border pt-3 space-y-3">
                    <div className="text-sm font-medium text-slack-text">Forwarding rules</div>
                    <label className="flex items-start gap-2 text-sm text-slack-text">
                      <input
                        type="checkbox"
                        checked={
                          slackInbox.forward_rules?.find((r) => r.type === 'mention_of_me')?.enabled ??
                          false
                        }
                        onChange={(e) =>
                          setSlackInbox((prev) =>
                            updateForwardRule(prev, 'mentions', { enabled: e.target.checked })
                          )
                        }
                        className="mt-1"
                      />
                      <span>
                        <span className="font-medium">@mention of me</span>
                        <span className="block text-xs text-slack-textMuted">
                          Forward when someone @mentions you in watched channels
                        </span>
                      </span>
                    </label>
                    {slackChannels.length > 0 && (
                      <div className="max-h-28 overflow-y-auto border border-slack-border rounded p-2 space-y-1">
                        {slackChannels.map((c) => {
                          const mentionRule = slackInbox.forward_rules?.find(
                            (r) => r.type === 'mention_of_me'
                          );
                          const checked = (mentionRule?.slack_channel_ids ?? []).includes(c.id);
                          return (
                            <label key={c.id} className="flex items-center gap-2 text-xs text-slack-text">
                              <input
                                type="checkbox"
                                checked={checked}
                                onChange={() => toggleMentionWatchChannel(c.id)}
                              />
                              {c.is_private ? '🔒' : '#'}
                              {c.name}
                            </label>
                          );
                        })}
                      </div>
                    )}
                    <label className="flex items-start gap-2 text-sm text-slack-text">
                      <input
                        type="checkbox"
                        checked={
                          slackInbox.forward_rules?.find((r) => r.type === 'prefix')?.enabled ?? false
                        }
                        onChange={(e) =>
                          setSlackInbox((prev) =>
                            updateForwardRule(prev, 'nj-prefix', { enabled: e.target.checked })
                          )
                        }
                        className="mt-1"
                      />
                      <span>
                        <span className="font-medium">Prefix </span>
                        <code className="text-xs bg-slack-bg px-1 rounded">nj:</code>
                        <span className="block text-xs text-slack-textMuted">
                          Start a line with nj: in any channel the bot is in
                        </span>
                      </span>
                    </label>
                    <label className="flex items-start gap-2 text-sm text-slack-text">
                      <input
                        type="checkbox"
                        checked={
                          slackInbox.forward_rules?.find((r) => r.type === 'reaction')?.enabled ?? false
                        }
                        onChange={(e) =>
                          setSlackInbox((prev) =>
                            updateForwardRule(prev, 'robot-react', { enabled: e.target.checked })
                          )
                        }
                        className="mt-1"
                      />
                      <span className="flex-1">
                        <span className="font-medium">Reaction </span>
                        <input
                          type="text"
                          value={
                            slackInbox.forward_rules?.find((r) => r.type === 'reaction')?.emoji ??
                            'robot_face'
                          }
                          onChange={(e) =>
                            setSlackInbox((prev) =>
                              updateForwardRule(prev, 'robot-react', { emoji: e.target.value })
                            )
                          }
                          className="ml-1 px-1 py-0.5 w-28 bg-slack-bg border border-slack-border rounded text-xs font-mono"
                        />
                        <span className="block text-xs text-slack-textMuted">
                          You react with this emoji to forward a message (watchlist same as @mentions)
                        </span>
                      </span>
                    </label>
                  </div>
                  <details className="border-t border-slack-border pt-3">
                    <summary className="text-sm font-medium text-slack-text cursor-pointer">
                      Human DM away mode
                    </summary>
                    <p className="text-xs text-slack-textMuted mt-2 mb-3">
                      When away, NJ reads your 1:1 Slack DMs locally (encrypted token) and replies in the DM as
                      &quot;Assistant (for you)&quot;. Someone must DM <strong>you</strong> directly — not note-to-self
                      (&quot;Jot Something Down&quot;) and not the NJ bot.
                    </p>
                    <div className="space-y-3">
                      <label className="flex items-center justify-between gap-3 text-sm text-slack-text">
                        <span>Enable human DM away mode</span>
                        <input
                          type="checkbox"
                          checked={slackInbox.human_dm_away?.enabled ?? false}
                          onChange={(e) =>
                            setSlackInbox((prev) => ({
                              ...prev,
                              human_dm_away: { ...prev.human_dm_away, enabled: e.target.checked },
                            }))
                          }
                          className="h-4 w-4 text-slack-accent"
                        />
                      </label>
                      <div className="flex flex-wrap items-center gap-2">
                        <button
                          type="button"
                          onClick={() => void authorizeSlackUserDM()}
                          disabled={slackBusy || !(slackInbox.human_dm_away?.enabled ?? false)}
                          className="px-3 py-1.5 text-xs border border-slack-border rounded hover:bg-slack-bg text-slack-text disabled:opacity-50"
                        >
                          Authorize Slack DM access
                        </button>
                        <span className="text-xs text-slack-textMuted">
                          {slackInbox.human_dm_away?.user_token_set ? 'Authorized' : 'Not authorized'}
                        </span>
                      </div>
                      <label className="flex items-center justify-between gap-3 text-sm text-slack-text">
                        <span>I&apos;m away now</span>
                        <input
                          type="checkbox"
                          checked={slackInbox.human_dm_away?.away_enabled ?? false}
                          onChange={(e) =>
                            setSlackInbox((prev) => ({
                              ...prev,
                              human_dm_away: { ...prev.human_dm_away, away_enabled: e.target.checked },
                            }))
                          }
                          disabled={!(slackInbox.human_dm_away?.enabled ?? false)}
                          className="h-4 w-4 text-slack-accent disabled:opacity-50"
                        />
                      </label>
                      <label className="flex items-center justify-between gap-3 text-sm text-slack-text">
                        <span>
                          Schedule (monitor outside Mon–Fri 9am–5pm)
                        </span>
                        <input
                          type="checkbox"
                          checked={slackInbox.human_dm_away?.schedule_enabled ?? false}
                          onChange={(e) =>
                            setSlackInbox((prev) => ({
                              ...prev,
                              human_dm_away: {
                                ...prev.human_dm_away,
                                schedule_enabled: e.target.checked,
                              },
                            }))
                          }
                          disabled={!(slackInbox.human_dm_away?.enabled ?? false)}
                          className="h-4 w-4 text-slack-accent disabled:opacity-50"
                        />
                      </label>
                      <label className="block text-xs text-slack-text">
                        Timezone
                        <input
                          type="text"
                          value={
                            slackInbox.human_dm_away?.schedule_timezone ??
                            'America/Los_Angeles'
                          }
                          onChange={(e) =>
                            setSlackInbox((prev) => ({
                              ...prev,
                              human_dm_away: {
                                ...prev.human_dm_away,
                                schedule_timezone: e.target.value,
                              },
                            }))
                          }
                          disabled={!(slackInbox.human_dm_away?.enabled ?? false)}
                          className="mt-1 w-full px-2 py-1 bg-slack-bg border border-slack-border rounded font-mono text-xs disabled:opacity-50"
                        />
                      </label>
                      <p className="text-xs text-slack-textMuted">
                        Status:{' '}
                        {humanDMStatusLabel(slackInbox.human_dm_away?.monitoring_status)}
                      </p>
                    </div>
                  </details>
                </div>
                <h4 className="text-sm font-semibold text-slack-text mb-2">Channel bindings</h4>
                <p className="text-xs text-slack-textMuted mb-3">
                  Pick a channel the bot is in, or paste a channel ID (C…) from Slack → channel → About.
                </p>
                <div className="flex flex-wrap items-center gap-2 mb-2">
                  <button
                    type="button"
                    onClick={() => void loadSlackChannels()}
                    disabled={slackChannelsLoading || !slackCanListChannelsFrom(slackStatus, slackConfig)}
                    className="px-3 py-1 text-xs border border-slack-border rounded hover:bg-slack-bgHover text-slack-text disabled:opacity-50"
                  >
                    {slackChannelsLoading ? 'Loading channels…' : 'Load Slack channels'}
                  </button>
                  {slackChannels.length > 0 && (
                    <span className="text-xs text-slack-textMuted">
                      {slackChannels.length} channel{slackChannels.length === 1 ? '' : 's'} (bot is a member)
                    </span>
                  )}
                </div>
                {slackChannelsError && (
                  <p className="text-xs text-amber-400 mb-2">{slackChannelsError}</p>
                )}
                <div className="grid gap-2 mb-3 sm:grid-cols-2">
                  <select
                    value={
                      slackChannels.some((c) => c.id === slackBindingForm.slackChannelId)
                        ? slackBindingForm.slackChannelId
                        : ''
                    }
                    onChange={(e) => {
                      const id = e.target.value;
                      const ch = slackChannels.find((c) => c.id === id);
                      setSlackBindingForm((prev) => ({
                        ...prev,
                        slackChannelId: id,
                        slackChannelName: ch?.name ?? prev.slackChannelName,
                        policy: slackForm.defaultPolicy,
                      }));
                    }}
                    className="px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-sm text-slack-text sm:col-span-2"
                  >
                    <option value="" className="bg-slack-bg text-slack-text">
                      {slackChannels.length > 0
                        ? 'Select a Slack channel…'
                        : 'Load channels or paste ID below'}
                    </option>
                    {slackChannels.map((c) => (
                      <option key={c.id} value={c.id} className="bg-slack-bg text-slack-text">
                        {c.is_private ? '🔒 ' : '#'}
                        {c.name} ({c.id})
                      </option>
                    ))}
                  </select>
                  <input
                    type="text"
                    placeholder="Or paste channel ID (C…)"
                    value={slackBindingForm.slackChannelId}
                    onChange={(e) =>
                      setSlackBindingForm((prev) => ({
                        ...prev,
                        slackChannelId: e.target.value,
                      }))
                    }
                    className="px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-sm font-mono text-slack-text sm:col-span-2"
                  />
                  <input
                    type="text"
                    placeholder="Display label (optional)"
                    value={slackBindingForm.slackChannelName}
                    onChange={(e) =>
                      setSlackBindingForm((prev) => ({
                        ...prev,
                        slackChannelName: e.target.value,
                      }))
                    }
                    className="px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-sm text-slack-text"
                  />
                  <select
                    value={slackBindingForm.agentId}
                    onChange={(e) =>
                      setSlackBindingForm((prev) => ({ ...prev, agentId: e.target.value }))
                    }
                    className="px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-sm text-slack-text"
                  >
                    <option value="" className="bg-slack-bg text-slack-text">
                      Select primary agent…
                    </option>
                    {agents.map((a) => (
                      <option key={a.id} value={a.id} className="bg-slack-bg text-slack-text">
                        {a.name} ({a.type})
                      </option>
                    ))}
                  </select>
                  <select
                    value={slackBindingForm.policy}
                    onChange={(e) =>
                      setSlackBindingForm((prev) => ({
                        ...prev,
                        policy: e.target.value as SlackPolicy,
                      }))
                    }
                    className="px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-sm text-slack-text"
                  >
                    <option value="mention_only" className="bg-slack-bg text-slack-text">
                      Mention only
                    </option>
                    <option value="questions" className="bg-slack-bg text-slack-text">
                      Questions
                    </option>
                    <option value="always" className="bg-slack-bg text-slack-text">
                      Always
                    </option>
                  </select>
                  <button
                    type="button"
                    onClick={() => void saveSlackBinding()}
                    disabled={slackBusy}
                    className="px-4 py-2 bg-slack-accent text-white rounded hover:bg-slack-accentHover disabled:opacity-50 sm:col-span-2"
                  >
                    Add / update binding
                  </button>
                </div>
                {slackBindings.length > 0 ? (
                  <ul className="space-y-2 text-sm text-slack-text">
                    {slackBindings.map((b) => (
                      <li
                        key={b.slack_channel_id}
                        className="flex items-center justify-between gap-2 p-2 bg-slack-bgHover rounded border border-slack-border"
                      >
                        <button
                          type="button"
                          className="truncate text-left text-slack-text hover:underline"
                          title="Load into form to edit policy or agent"
                          onClick={() =>
                            setSlackBindingForm({
                              slackChannelId: b.slack_channel_id,
                              slackChannelName: b.slack_channel_name || '',
                              agentId: b.agent_id,
                              policy: b.policy,
                            })
                          }
                        >
                          {b.slack_channel_name ? `#${b.slack_channel_name}` : b.slack_channel_id} →{' '}
                          {b.agent_name || b.agent_id} ({b.policy})
                        </button>
                        <button
                          type="button"
                          onClick={() => void deleteSlackBindingRow(b.slack_channel_id)}
                          className="shrink-0 text-xs text-red-600 hover:underline"
                        >
                          Remove
                        </button>
                      </li>
                    ))}
                  </ul>
                ) : (
                  <p className="text-sm text-slack-textMuted">No bindings yet.</p>
                )}
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
                <strong className="text-slack-text">Always on:</strong> ChatModerator, Assistant, and CLI agents (Cursor, Claude, Copilot, Codex, Gemini, Aider, OpenCode, and more) when installed on your PATH. Domain packs add optional in-process specialists and tools below.
              </div>

              <div className="border border-slack-border rounded-lg p-6">
                <h3 className="text-lg font-semibold text-slack-text mb-2">Pack store</h3>
                <p className="text-sm text-slack-textMuted mb-4">
                  Install official domain packs from the catalog. You can enable multiple packs; choose which enabled pack controls the UI layout (IDE vs team). Later packs add specialists and tools without changing layout unless you switch the owner below.
                </p>
                {(() => {
                  const layoutCandidates = packs.filter((p) => p.enabled && !p.custom);
                  if (layoutCandidates.length < 2) return null;
                  return (
                    <div className="mb-4">
                      <label className="block text-sm font-medium text-slack-text mb-1" htmlFor="layout-owner-select">
                        UI layout owner
                      </label>
                      <select
                        id="layout-owner-select"
                        className="w-full max-w-md rounded border border-slack-border bg-slack-bg px-3 py-2 text-sm text-slack-text"
                        value={layoutOwner}
                        onChange={(e) => void setLayoutOwner(e.target.value)}
                      >
                        {layoutCandidates.map((p) => (
                          <option key={p.id} value={p.id}>
                            {p.title} ({p.layout_profile === 'ide' ? 'IDE' : 'Team'})
                          </option>
                        ))}
                      </select>
                      <p className="text-xs text-slack-textMuted mt-1">
                        Controls IDE vs team layout and which pack&apos;s default Ollama model applies when enabled.
                      </p>
                    </div>
                  );
                })()}
                {packsLoading && <p className="text-sm text-slack-textMuted mb-2">Loading packs…</p>}
                {packsErr && <p className="text-sm text-red-600 mb-2">{packsErr}</p>}
                <PackStoreBrowse />
              </div>

              <PackDevStudio />

              <div className="border border-slack-border rounded-lg p-6">
                <h3 className="text-lg font-semibold text-slack-text mb-2">MCP specialist tools</h3>
                <p className="text-sm text-slack-textMuted mb-4">
                  Per-agent MCP tool servers. Enablement follows domain packs by default; override individual specialists here. Repo and Confluence agents always use in-process search tools when indexed.
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
                {mcpEnabled && (
                  <div className="grid gap-2 sm:grid-cols-2">
                    {[
                      ['backend', 'BackendEngineer'],
                      ['frontend', 'FrontendEngineer'],
                      ['devops', 'PlatformEngineer'],
                      ['database', 'DatabaseSpecialist'],
                      ['security', 'SecurityReviewer'],
                      ['code-review', 'CodeReviewer'],
                      ['architecture', 'SoftwareArchitect'],
                      ['biology', 'BiologyExpert'],
                      ['cad', 'CADExpert'],
                      ['rust', 'RustExpert'],
                    ].map(([key, label]) => (
                      <label key={key} className="flex items-center gap-2 cursor-pointer text-sm">
                        <input
                          type="checkbox"
                          checked={mcpAgents[key] !== false}
                          onChange={(e) => void handleMcpAgentToggle(key, e.target.checked)}
                          className="rounded border-slack-border"
                        />
                        <span className="text-slack-text">{label}</span>
                      </label>
                    ))}
                  </div>
                )}
              </div>

              {bioPackTools && (
                <div className="border border-slack-border rounded-lg p-6">
                  <h3 className="text-lg font-semibold text-slack-text mb-2">Life sciences tools</h3>
                  <p className="text-sm text-slack-textMuted mb-4">
                    Model layering for BiologyExpert: OpenBio (or your chat tag) for reasoning; a tool-capable model
                    for MCP (<code className="font-mono text-xs bg-slack-bgHover px-1 rounded">analyze_sequence</code>,{' '}
                    <code className="font-mono text-xs bg-slack-bgHover px-1 rounded">fold_protein</code>, QC tools).
                  </p>
                  <div className="grid gap-3 sm:grid-cols-2">
                    <label className="block text-sm">
                      <span className="text-slack-textMuted">Chat model (domain reasoning)</span>
                      <input
                        type="text"
                        value={bioChatModel}
                        onChange={(e) => setBioChatModel(e.target.value)}
                        className="mt-1 w-full px-3 py-2 border border-slack-border rounded bg-slack-bg text-slack-text font-mono text-sm"
                      />
                    </label>
                    <label className="block text-sm">
                      <span className="text-slack-textMuted">Tool model (MCP loop)</span>
                      <input
                        type="text"
                        value={bioToolModel}
                        onChange={(e) => setBioToolModel(e.target.value)}
                        className="mt-1 w-full px-3 py-2 border border-slack-border rounded bg-slack-bg text-slack-text font-mono text-sm"
                      />
                    </label>
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
                    {bioSecondaryAnalysisTools && (
                      <>
                        <p className="text-xs text-slack-textMuted sm:col-span-2">
                          Customer pack (<code className="font-mono">settings_overlay</code>). Override below
                          if needed.
                        </p>
                        <label className="block text-sm sm:col-span-2">
                          <span className="text-slack-textMuted">Secondary analysis tools path</span>
                          <input
                            type="text"
                            value={bioSecondaryToolsPath}
                            onChange={(e) => setBioSecondaryToolsPath(e.target.value)}
                            placeholder="/path/to/secondary-analysis-tools"
                            className="mt-1 w-full px-3 py-2 border border-slack-border rounded bg-slack-bg text-slack-text font-mono text-sm"
                          />
                        </label>
                        <label className="block text-sm">
                          <span className="text-slack-textMuted">Python executable</span>
                          <input
                            type="text"
                            value={bioPythonExecutable}
                            onChange={(e) => setBioPythonExecutable(e.target.value)}
                            className="mt-1 w-full px-3 py-2 border border-slack-border rounded bg-slack-bg text-slack-text font-mono text-sm"
                          />
                        </label>
                        <label className="block text-sm">
                          <span className="text-slack-textMuted">Default panel profile</span>
                          <input
                            type="text"
                            value={bioDefaultPanelProfile}
                            onChange={(e) => setBioDefaultPanelProfile(e.target.value)}
                            className="mt-1 w-full px-3 py-2 border border-slack-border rounded bg-slack-bg text-slack-text font-mono text-sm"
                          />
                        </label>
                        <label className="block text-sm sm:col-span-2">
                          <span className="text-slack-textMuted">
                            Cumulative QC folder override (empty = workspace/.neural-junkie/cumulative-qc)
                          </span>
                          <input
                            type="text"
                            value={bioCumulativeQCDir}
                            onChange={(e) => setBioCumulativeQCDir(e.target.value)}
                            className="mt-1 w-full px-3 py-2 border border-slack-border rounded bg-slack-bg text-slack-text font-mono text-sm"
                          />
                        </label>
                      </>
                    )}
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

              {cadPackTools && (
                <div className="border border-slack-border rounded-lg p-6">
                  <h3 className="text-lg font-semibold text-slack-text mb-2">CAD tools</h3>
                  <p className="text-sm text-slack-textMuted mb-4">
                    OpenSCAD rendering for <code className="font-mono text-xs bg-slack-bgHover px-1 rounded">render_openscad</code> and the CAD workbench. Install OpenSCAD from{' '}
                    <a href="https://openscad.org" className="text-slack-accent hover:underline" target="_blank" rel="noreferrer">openscad.org</a>.
                  </p>
                  <div className="grid gap-3 sm:grid-cols-2">
                    <label className="block text-sm sm:col-span-2">
                      <span className="text-slack-textMuted">OpenSCAD path</span>
                      <input
                        type="text"
                        value={cadOpenSCADPath}
                        onChange={(e) => setCadOpenSCADPath(e.target.value)}
                        className="mt-1 w-full px-3 py-2 border border-slack-border rounded bg-slack-bg text-slack-text font-mono text-sm"
                      />
                    </label>
                    <label className="block text-sm">
                      <span className="text-slack-textMuted">Chat model</span>
                      <input
                        type="text"
                        value={cadChatModel}
                        onChange={(e) => setCadChatModel(e.target.value)}
                        className="mt-1 w-full px-3 py-2 border border-slack-border rounded bg-slack-bg text-slack-text font-mono text-sm"
                      />
                    </label>
                    <label className="block text-sm">
                      <span className="text-slack-textMuted">Tool model</span>
                      <input
                        type="text"
                        value={cadToolModel}
                        onChange={(e) => setCadToolModel(e.target.value)}
                        className="mt-1 w-full px-3 py-2 border border-slack-border rounded bg-slack-bg text-slack-text font-mono text-sm"
                      />
                    </label>
                    <label className="block text-sm">
                      <span className="text-slack-textMuted">Render timeout (sec)</span>
                      <input
                        type="number"
                        value={cadRenderTimeout}
                        onChange={(e) => setCadRenderTimeout(e.target.value)}
                        className="mt-1 w-full px-3 py-2 border border-slack-border rounded bg-slack-bg text-slack-text"
                      />
                    </label>
                    <label className="block text-sm sm:col-span-2">
                      <span className="text-slack-textMuted">Artifacts directory (empty = ~/.neural-junkie/cad)</span>
                      <input
                        type="text"
                        value={cadArtifactsDir}
                        onChange={(e) => setCadArtifactsDir(e.target.value)}
                        className="mt-1 w-full px-3 py-2 border border-slack-border rounded bg-slack-bg text-slack-text font-mono text-sm"
                      />
                    </label>
                    <label className="block text-sm sm:col-span-2">
                      <span className="text-slack-textMuted">FreeCAD path (optional, for STEP export)</span>
                      <input
                        type="text"
                        value={cadFreeCADPath}
                        onChange={(e) => setCadFreeCADPath(e.target.value)}
                        className="mt-1 w-full px-3 py-2 border border-slack-border rounded bg-slack-bg text-slack-text font-mono text-sm"
                      />
                    </label>
                  </div>
                  <div className="mt-4 flex flex-wrap gap-2">
                    <button
                      type="button"
                      onClick={() => void saveCadMcpSettings()}
                      disabled={cadSettingsSaving}
                      className="px-4 py-2 text-sm bg-slack-accent text-white rounded hover:bg-slack-accentHover disabled:opacity-50"
                    >
                      {cadSettingsSaving ? 'Saving…' : 'Save CAD tools'}
                    </button>
                    <button
                      type="button"
                      onClick={() => void testCadOpenSCAD()}
                      className="px-4 py-2 text-sm border border-slack-border rounded text-slack-text hover:bg-slack-bgHover"
                    >
                      Test OpenSCAD
                    </button>
                  </div>
                  {cadSettingsErr && <p className="text-sm text-red-600 mt-2">{cadSettingsErr}</p>}
                  {cadSettingsOk && <p className="text-sm text-green-600 mt-2">{cadSettingsOk}</p>}
                  {cadTestResult && <p className="text-sm text-slack-textMuted mt-2 font-mono whitespace-pre-wrap">{cadTestResult}</p>}
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
                  <div className="col-span-2">
                    <span className="text-slack-textMuted">Update channel:</span>
                    <span className="ml-2 text-slack-text">{getUpdateChannelLabel(appVersion)}</span>
                  </div>
                </div>
                <div className="mt-4 flex flex-wrap items-center gap-3">
                  <button
                    type="button"
                    disabled={updateChecking || updateInstalling}
                    onClick={async () => {
                      setUpdateChecking(true);
                      setUpdateCheckStatus(null);
                      setPendingUpdateVersion(null);
                      try {
                        const result = await checkForAppUpdate();
                        if (result.status === 'available') {
                          setPendingUpdateVersion(result.update.version ?? 'new version');
                          setUpdateCheckStatus(`Update available: ${result.update.version ?? 'new version'}`);
                        } else if (result.status === 'current') {
                          setUpdateCheckStatus('You are on the latest version.');
                        } else {
                          setUpdateCheckStatus(result.reason);
                        }
                      } catch (e) {
                        setUpdateCheckStatus(e instanceof Error ? e.message : 'Update check failed');
                      } finally {
                        setUpdateChecking(false);
                      }
                    }}
                    className="px-3 py-1.5 text-sm bg-slack-accent text-white rounded hover:bg-slack-accentHover transition-colors disabled:opacity-50"
                  >
                    {updateChecking ? 'Checking…' : 'Check for updates'}
                  </button>
                  {pendingUpdateVersion && (
                    <button
                      type="button"
                      disabled={updateInstalling}
                      onClick={async () => {
                        setUpdateInstalling(true);
                        setUpdateCheckStatus(null);
                        try {
                          await installAppUpdate(setUpdateProgress);
                        } catch (e) {
                          setUpdateCheckStatus(e instanceof Error ? e.message : 'Update install failed');
                          setUpdateInstalling(false);
                        }
                      }}
                      className="px-3 py-1.5 text-sm bg-blue-600 text-white rounded hover:bg-blue-700 transition-colors disabled:opacity-50"
                    >
                      {updateInstalling ? `Installing… ${updateProgress}%` : `Install ${pendingUpdateVersion}`}
                    </button>
                  )}
                  {updateCheckStatus && (
                    <span className="text-sm text-slack-textMuted">{updateCheckStatus}</span>
                  )}
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
    </div>
  );
}
