import { useState, useEffect } from 'react';
import { shallow } from 'zustand/shallow';
import { useSettingsStore } from '../../stores/settingsStore';
import { useChatStore } from '../../stores/chatStore';
import { ChatAPI } from '../../api/chatAPI';
import type {
  AnthropicSettings,
  GitHubSettings,
  ConfluenceSettings,
  GoogleMeetNotesSettings,
  GoogleMeetNotesStatus,
  SlackStatus,
  SlackBinding,
  SlackChannelInfo,
  SlackPolicy,
  SlackConfigResponse,
  SlackConnectionResponse,
  SlackInboxConfig,
} from '../../types/protocol';
import {
  defaultSlackInboxForm,
  mergeSlackInboxForm,
  slackCanListChannelsFrom,
  updateForwardRule,
} from './slackInboxHelpers';
import { openExternalLink, type SettingsTabProps } from './settingsShared';

export function IntegrationsSettingsTab({ hubHttp, isActive }: SettingsTabProps) {
  const {
    integrations,
    loadIntegrations,
    updateAnthropicSettings,
    updateGitHubSettings,
    updateConfluenceSettings,
    updateGoogleMeetNotesSettings,
    clearIntegrationSettings,
    testAnthropicConnection,
    testGitHubConnection,
    testConfluenceConnection,
  } = useSettingsStore();
  useEffect(() => {
    if (!isActive) return;
    loadIntegrations();
  }, [isActive, loadIntegrations]);

  const { agents, setChannels } = useChatStore(
    (s) => ({ agents: s.agents, setChannels: s.setChannels }),
    shallow
  );

    // Integration form states
    const [anthropicForm, setAnthropicForm] = useState<AnthropicSettings>(integrations.anthropic);
    const [githubForm, setGitHubForm] = useState<GitHubSettings>(integrations.github);
    const [confluenceForm, setConfluenceForm] = useState<ConfluenceSettings>(integrations.confluence);
    const [googleOAuthForm, setGoogleOAuthForm] = useState<GoogleMeetNotesSettings>(integrations.googleMeetNotes);
    const [googleOAuthSecretSet, setGoogleOAuthSecretSet] = useState(false);
    const [showPasswords, setShowPasswords] = useState<Record<string, boolean>>({});
    const [testResults, setTestResults] = useState<Record<string, { success: boolean; message: string }>>({});
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


    // Update form states when integrations change
    useEffect(() => {
      setAnthropicForm(integrations.anthropic);
      setGitHubForm(integrations.github);
      setConfluenceForm(integrations.confluence);
      setGoogleOAuthForm(integrations.googleMeetNotes);
          }, [integrations]);

    useEffect(() => {
      if (!isActive) return;
      void refreshGoogleMeetNotesStatus();
      void refreshSlackIntegration();
    }, [isActive, hubHttp]);

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
        openExternalLink(url);
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
        openExternalLink(url);
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
        openExternalLink(url);
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

  if (!isActive) return null;

  return (
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
              onClick={() => openExternalLink('https://console.anthropic.com/')}
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
              onClick={() => openExternalLink('https://github.com/settings/tokens')}
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
              onClick={() => openExternalLink('https://id.atlassian.com/manage-profile/security/api-tokens')}
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
  );
}
