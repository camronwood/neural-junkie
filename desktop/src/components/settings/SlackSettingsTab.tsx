import { useState, useEffect } from 'react';
import { shallow } from 'zustand/shallow';
import { useChatStore } from '../../stores/chatStore';
import { ChatAPI } from '../../api/chatAPI';
import type {
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
import { mergeSettingsPut, openExternalLink, type SettingsTabProps } from './settingsShared';

export function SlackSettingsTab({ hubHttp, isActive }: SettingsTabProps) {
  const { agents, setChannels } = useChatStore(
    (s) => ({ agents: s.agents, setChannels: s.setChannels }),
    shallow
  );
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
    const [slackInboxFeedback, setSlackInboxFeedback] = useState<{ success: boolean; message: string } | null>(null);
    const [slackBindingFeedback, setSlackBindingFeedback] = useState<{ success: boolean; message: string } | null>(null);
    const [slackConnection, setSlackConnection] = useState<SlackConnectionResponse | null>(null);
    const [slackInbox, setSlackInbox] = useState<SlackInboxConfig>(() => defaultSlackInboxForm());
    const [slackAdvancedOpen, setSlackAdvancedOpen] = useState(false);
    const [slackHubOverridesOpen, setSlackHubOverridesOpen] = useState(false);
    const [slackHubOverrides, setSlackHubOverrides] = useState({
      force_disabled: false,
      debug: false,
      oauth_relay_base: '',
      use_oauth_relay: false,
    });
    const [slackHubOverridesSaving, setSlackHubOverridesSaving] = useState(false);
    const [testResults, setTestResults] = useState<Record<string, { success: boolean; message: string }>>({});
    const loadSlackHubOverrides = async () => {
      try {
        const r = await fetch(`${hubHttp}/api/settings`);
        if (!r.ok) return;
        const cfg = await r.json();
        const slack = cfg.slack ?? {};
        setSlackHubOverrides({
          force_disabled: !!slack.force_disabled,
          debug: !!slack.debug,
          oauth_relay_base: String(slack.oauth_relay_base ?? ''),
          use_oauth_relay: slack.use_oauth_relay === true,
        });
      } catch {
        /* ignore */
      }
    };

    const saveSlackHubOverrides = async () => {
      setSlackHubOverridesSaving(true);
      try {
        await mergeSettingsPut(hubHttp, (cfg) => ({
          ...cfg,
          slack: {
            ...(typeof cfg.slack === 'object' && cfg.slack ? cfg.slack : {}),
            force_disabled: slackHubOverrides.force_disabled,
            debug: slackHubOverrides.debug,
            oauth_relay_base: slackHubOverrides.oauth_relay_base.trim(),
            use_oauth_relay: slackHubOverrides.use_oauth_relay,
          },
        }));
        setTestResults((prev) => ({
          ...prev,
          slackHub: { success: true, message: 'Slack hub overrides saved. Restart hub if bridge was running.' },
        }));
      } catch (e) {
        setTestResults((prev) => ({
          ...prev,
          slackHub: {
            success: false,
            message: e instanceof Error ? e.message : 'Failed to save Slack hub overrides',
          },
        }));
      } finally {
        setSlackHubOverridesSaving(false);
      }
    };

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
        const message = 'Slack channel ID and agent are required.';
        setSlackBindingFeedback({ success: false, message });
        setTestResults((prev) => ({
          ...prev,
          slack: { success: false, message },
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
        setSlackBindingFeedback({ success: true, message: 'Channel binding saved.' });
        setTestResults((prev) => ({
          ...prev,
          slack: { success: true, message: 'Channel binding saved.' },
        }));
      } catch (e) {
        const message = e instanceof Error ? e.message : 'Failed to save binding';
        setSlackBindingFeedback({ success: false, message });
        setTestResults((prev) => ({
          ...prev,
          slack: { success: false, message },
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
        setSlackBindingFeedback({ success: true, message: 'Binding removed.' });
      } catch (e) {
        const message = e instanceof Error ? e.message : 'Failed to delete binding';
        setSlackBindingFeedback({ success: false, message });
        setTestResults((prev) => ({
          ...prev,
          slack: { success: false, message },
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
        setSlackInboxFeedback({ success: true, message: 'Personal inbox saved.' });
        setTestResults((prev) => ({
          ...prev,
          slack: { success: true, message: 'Personal inbox saved.' },
        }));
      } catch (e) {
        const message = e instanceof Error ? e.message : 'Failed to save personal inbox';
        setSlackInboxFeedback({ success: false, message });
        setTestResults((prev) => ({
          ...prev,
          slack: { success: false, message },
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
        setSlackInboxFeedback({ success: true, message: 'Test DM sent — check Slack.' });
        setTestResults((prev) => ({
          ...prev,
          slack: { success: true, message: 'Test DM sent — check Slack.' },
        }));
      } catch (e) {
        const message = e instanceof Error ? e.message : 'Inbox test DM failed';
        setSlackInboxFeedback({ success: false, message });
        setTestResults((prev) => ({
          ...prev,
          slack: { success: false, message },
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

    useEffect(() => {
      if (!isActive) return;
      void refreshSlackIntegration();
      void loadSlackHubOverrides();
    }, [isActive, hubHttp]);

  if (!isActive) return null;

  return (
    <div className="space-y-8 nj-settings-integrations text-slack-text">
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
      <details
        className="mb-6 border border-slack-border rounded-lg"
        open={slackHubOverridesOpen}
        onToggle={(e) => setSlackHubOverridesOpen((e.target as HTMLDetailsElement).open)}
      >
        <summary className="cursor-pointer px-4 py-3 text-sm font-medium text-slack-text hover:bg-slack-bgHover rounded-lg">
          Hub overrides (force-disable, debug, OAuth relay)
        </summary>
        <div className="px-4 pb-4 space-y-4 border-t border-slack-border pt-4">
          <p className="text-xs text-slack-textMuted">
            Stored in hub config. Environment variables still override at runtime.
          </p>
          {testResults.slackHub && (
            <div
              className={`p-2 rounded text-xs ${
                testResults.slackHub.success
                  ? 'bg-green-100 text-green-800 border border-green-200'
                  : 'bg-red-100 text-red-800 border border-red-200'
              }`}
            >
              {testResults.slackHub.message}
            </div>
          )}
          <label className="flex items-center gap-2 text-sm text-slack-text">
            <input
              type="checkbox"
              checked={slackHubOverrides.force_disabled}
              onChange={(e) =>
                setSlackHubOverrides((p) => ({ ...p, force_disabled: e.target.checked }))
              }
            />
            Force Slack bridge disabled (overrides enabled toggle)
          </label>
          <label className="flex items-center gap-2 text-sm text-slack-text">
            <input
              type="checkbox"
              checked={slackHubOverrides.debug}
              onChange={(e) => setSlackHubOverrides((p) => ({ ...p, debug: e.target.checked }))}
            />
            Debug logging for Slack bridge
          </label>
          <label className="flex items-center gap-2 text-sm text-slack-text">
            <input
              type="checkbox"
              checked={slackHubOverrides.use_oauth_relay}
              onChange={(e) =>
                setSlackHubOverrides((p) => ({ ...p, use_oauth_relay: e.target.checked }))
              }
            />
            Use OAuth relay for Slack app install
          </label>
          <label className="block text-sm text-slack-text">
            OAuth relay base URL
            <input
              type="text"
              value={slackHubOverrides.oauth_relay_base}
              onChange={(e) =>
                setSlackHubOverrides((p) => ({ ...p, oauth_relay_base: e.target.value }))
              }
              placeholder="https://your-relay.example.com"
              className="mt-1 w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded font-mono text-xs"
            />
          </label>
          <button
            type="button"
            onClick={() => void saveSlackHubOverrides()}
            disabled={slackHubOverridesSaving}
            className="w-full px-4 py-2 bg-slack-accent text-white rounded hover:bg-slack-accentHover disabled:opacity-50"
          >
            {slackHubOverridesSaving ? 'Saving…' : 'Save hub overrides'}
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
        {slackInboxFeedback && (
          <div
            className={`p-2 rounded text-xs ${
              slackInboxFeedback.success
                ? 'bg-green-100 text-green-800 border border-green-200'
                : 'bg-red-100 text-red-800 border border-red-200'
            }`}
          >
            {slackInboxFeedback.message}
          </div>
        )}
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
      {slackBindingFeedback && (
        <div
          className={`mb-3 p-2 rounded text-xs ${
            slackBindingFeedback.success
              ? 'bg-green-100 text-green-800 border border-green-200'
              : 'bg-red-100 text-red-800 border border-red-200'
          }`}
        >
          {slackBindingFeedback.message}
        </div>
      )}
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
    </div>
  );
}
