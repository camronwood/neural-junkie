import { useCallback, useEffect, useState, type CSSProperties } from 'react';
import type { ChatAPI } from '../../api/chatAPI';
import type { SlackChannelInfo, SlackConfigResponse, SlackStatus, TaskActionSpec } from '../../types/protocol';
import {
  RUNBOOK_ACTION_TYPES,
  actionConfigString,
  defaultActionSpec,
} from '../../utils/runbookActionUtils';

function slackCanListChannels(
  status: SlackStatus | null | undefined,
  cfg: SlackConfigResponse | null | undefined
): boolean {
  return Boolean(cfg?.bot_token_set || status?.token_set || status?.configured);
}

interface RunbookActionConfigEditorProps {
  action: TaskActionSpec;
  onChange: (action: TaskActionSpec) => void;
  api: ChatAPI;
  disabled?: boolean;
  inputStyle: CSSProperties;
  labelStyle?: CSSProperties;
}

export function RunbookActionConfigEditor({
  action,
  onChange,
  api,
  disabled = false,
  inputStyle,
  labelStyle = defaultLabelStyle,
}: RunbookActionConfigEditorProps) {
  const [slackChannels, setSlackChannels] = useState<SlackChannelInfo[]>([]);
  const [slackChannelsLoading, setSlackChannelsLoading] = useState(false);
  const [slackChannelsError, setSlackChannelsError] = useState<string | null>(null);
  const [slackReady, setSlackReady] = useState(false);
  const [connectors, setConnectors] = useState<{ id: string; label: string; type: string }[]>([]);

  const actionType = action.type || 'http_get';
  const config = action.config ?? {};

  const updateConfig = useCallback(
    (key: string, value: unknown) => {
      onChange({
        ...action,
        config: { ...config, [key]: value },
      });
    },
    [action, config, onChange]
  );

  const loadSlackChannels = useCallback(async () => {
    setSlackChannelsLoading(true);
    setSlackChannelsError(null);
    try {
      const [status, cfg, channels] = await Promise.all([
        api.getSlackStatus(),
        api.getSlackConfig(),
        api.getSlackChannels(),
      ]);
      setSlackReady(slackCanListChannels(status, cfg));
      const sorted = [...channels].sort((a, b) => a.name.localeCompare(b.name));
      setSlackChannels(sorted);
      if (sorted.length === 0 && slackCanListChannels(status, cfg)) {
        setSlackChannelsError('No channels found — invite the bot to a channel or paste a channel ID below.');
      }
    } catch (e) {
      setSlackChannels([]);
      setSlackReady(false);
      setSlackChannelsError(e instanceof Error ? e.message : 'Failed to load Slack channels');
    } finally {
      setSlackChannelsLoading(false);
    }
  }, [api]);

  useEffect(() => {
    void (async () => {
      try {
        const list = await api.listConnectors();
        setConnectors(list.map((p) => ({ id: p.id, label: p.label, type: p.type })));
      } catch {
        setConnectors([]);
      }
    })();
  }, [api]);

  useEffect(() => {
    if (actionType !== 'slack_message') return;
    void (async () => {
      try {
        const [status, cfg] = await Promise.all([api.getSlackStatus(), api.getSlackConfig()]);
        const ready = slackCanListChannels(status, cfg);
        setSlackReady(ready);
        if (ready) {
          void loadSlackChannels();
        }
      } catch {
        setSlackReady(false);
      }
    })();
  }, [actionType, api, loadSlackChannels]);

  return (
    <div style={{ marginBottom: 6 }}>
      <label style={labelStyle}>
        Action type
        <select
          value={actionType}
          onChange={(e) => onChange(defaultActionSpec(e.target.value))}
          disabled={disabled}
          style={inputStyle}
        >
          {RUNBOOK_ACTION_TYPES.map((t) => (
            <option key={t.value} value={t.value}>
              {t.label}
            </option>
          ))}
        </select>
      </label>

      {connectors.length > 0 &&
      (actionType === 'http_get' ||
        actionType === 'http_post' ||
        actionType === 'webhook' ||
        actionType === 'slack_message' ||
        actionType === 'sms') ? (
        <label style={labelStyle}>
          Connector profile (optional)
          <select
            value={action.connector_id ?? ''}
            onChange={(e) =>
              onChange({ ...action, connector_id: e.target.value || undefined })
            }
            disabled={disabled}
            style={inputStyle}
          >
            <option value="">None — inline config</option>
            {connectors.map((c) => (
              <option key={c.id} value={c.id}>
                {c.label} ({c.type})
              </option>
            ))}
          </select>
        </label>
      ) : null}

      {actionType === 'http_get' || actionType === 'http_post' || actionType === 'webhook' ? (
        <label style={labelStyle}>
          URL
          <input
            type="text"
            value={actionConfigString(config, 'url')}
            onChange={(e) => updateConfig('url', e.target.value)}
            disabled={disabled}
            placeholder="https://…"
            style={inputStyle}
          />
        </label>
      ) : null}

      {actionType === 'http_post' ? (
        <label style={labelStyle}>
          JSON body
          <textarea
            value={jsonFieldValue(config.body)}
            onChange={(e) => updateConfig('body', parseJsonField(e.target.value))}
            disabled={disabled}
            rows={3}
            placeholder='{"key": "value"}'
            style={{ ...inputStyle, fontFamily: 'monospace', resize: 'vertical' }}
          />
        </label>
      ) : null}

      {actionType === 'webhook' ? (
        <label style={labelStyle}>
          Payload
          <textarea
            value={jsonFieldValue(config.payload)}
            onChange={(e) => updateConfig('payload', parseJsonField(e.target.value))}
            disabled={disabled}
            rows={3}
            placeholder='{"event": "done"}'
            style={{ ...inputStyle, fontFamily: 'monospace', resize: 'vertical' }}
          />
        </label>
      ) : null}

      {actionType === 'web_search' ? (
        <label style={labelStyle}>
          Query
          <input
            type="text"
            value={actionConfigString(config, 'query')}
            onChange={(e) => updateConfig('query', e.target.value)}
            disabled={disabled}
            style={inputStyle}
          />
        </label>
      ) : null}

      {actionType === 'sms' ? (
        <>
          <label style={labelStyle}>
            To
            <input
              type="text"
              value={actionConfigString(config, 'to')}
              onChange={(e) => updateConfig('to', e.target.value)}
              disabled={disabled}
              placeholder="+1…"
              style={inputStyle}
            />
          </label>
          <label style={labelStyle}>
            Body
            <textarea
              value={actionConfigString(config, 'body')}
              onChange={(e) => updateConfig('body', e.target.value)}
              disabled={disabled}
              rows={2}
              style={{ ...inputStyle, resize: 'vertical' }}
            />
          </label>
        </>
      ) : null}

      {actionType === 'slack_message' ? (
        <>
          <div style={{ display: 'flex', gap: 8, alignItems: 'center', marginBottom: 6 }}>
            <button
              type="button"
              onClick={() => void loadSlackChannels()}
              disabled={disabled || slackChannelsLoading || !slackReady}
              style={secondaryBtnStyle}
            >
              {slackChannelsLoading ? 'Loading…' : 'Load Slack channels'}
            </button>
            {!slackReady ? (
              <span style={{ fontSize: 11, color: '#fbbf24' }}>Connect Slack in Settings first</span>
            ) : null}
          </div>
          {slackChannelsError ? (
            <p style={{ fontSize: 11, color: '#fbbf24', margin: '0 0 6px' }}>{slackChannelsError}</p>
          ) : null}
          {slackChannels.length > 0 ? (
            <label style={labelStyle}>
              Channel
              <select
                value={
                  slackChannels.some((c) => c.id === actionConfigString(config, 'channel_id'))
                    ? actionConfigString(config, 'channel_id')
                    : ''
                }
                onChange={(e) => updateConfig('channel_id', e.target.value)}
                disabled={disabled}
                style={inputStyle}
              >
                <option value="">Select a channel…</option>
                {slackChannels.map((c) => (
                  <option key={c.id} value={c.id}>
                    {c.is_private ? '🔒 ' : '#'}
                    {c.name} ({c.id})
                  </option>
                ))}
              </select>
            </label>
          ) : null}
          <label style={labelStyle}>
            Channel ID
            <input
              type="text"
              value={actionConfigString(config, 'channel_id')}
              onChange={(e) => updateConfig('channel_id', e.target.value)}
              disabled={disabled}
              placeholder="C01234567"
              style={inputStyle}
            />
          </label>
          <label style={labelStyle}>
            Message
            <textarea
              value={actionConfigString(config, 'text')}
              onChange={(e) => updateConfig('text', e.target.value)}
              disabled={disabled}
              rows={3}
              placeholder="Runbook step done: {{task.title}}"
              style={{ ...inputStyle, resize: 'vertical' }}
            />
          </label>
          <label style={labelStyle}>
            Thread ts (optional)
            <input
              type="text"
              value={actionConfigString(config, 'thread_ts')}
              onChange={(e) => updateConfig('thread_ts', e.target.value)}
              disabled={disabled}
              placeholder="1710000000.000000"
              style={inputStyle}
            />
          </label>
          <label style={labelStyle}>
            Bot display name (optional)
            <input
              type="text"
              value={actionConfigString(config, 'username')}
              onChange={(e) => updateConfig('username', e.target.value)}
              disabled={disabled}
              placeholder="Runbook Bot"
              style={inputStyle}
            />
          </label>
        </>
      ) : null}

      {actionType === 'shell' ? (
        <label style={labelStyle}>
          Command
          <input
            type="text"
            value={actionConfigString(config, 'command')}
            onChange={(e) => updateConfig('command', e.target.value)}
            disabled={disabled}
            placeholder="npm test"
            style={inputStyle}
          />
        </label>
      ) : null}

      {actionType === 'mcp_tool' ? (
        <>
          <label style={labelStyle}>
            Tool name
            <input
              type="text"
              value={actionConfigString(config, 'tool')}
              onChange={(e) => updateConfig('tool', e.target.value)}
              disabled={disabled}
              style={inputStyle}
            />
          </label>
          <label style={labelStyle}>
            Arguments (JSON)
            <textarea
              value={jsonFieldValue(config.arguments)}
              onChange={(e) => updateConfig('arguments', parseJsonField(e.target.value))}
              disabled={disabled}
              rows={3}
              style={{ ...inputStyle, fontFamily: 'monospace', resize: 'vertical' }}
            />
          </label>
        </>
      ) : null}

      {actionType === 'wait_human' ? (
        <label style={labelStyle}>
          Approval prompt
          <textarea
            value={actionConfigString(config, 'prompt')}
            onChange={(e) => updateConfig('prompt', e.target.value)}
            disabled={disabled}
            rows={2}
            style={{ ...inputStyle, resize: 'vertical' }}
          />
        </label>
      ) : null}

      {(actionType === 'git_status' || actionType === 'git_diff') ? (
        <label style={labelStyle}>
          Path (optional)
          <input
            type="text"
            value={actionConfigString(config, 'path')}
            onChange={(e) => updateConfig('path', e.target.value)}
            disabled={disabled}
            placeholder="src/"
            style={inputStyle}
          />
        </label>
      ) : null}
    </div>
  );
}

function jsonFieldValue(value: unknown): string {
  if (value == null) return '';
  if (typeof value === 'string') return value;
  try {
    return JSON.stringify(value, null, 2);
  } catch {
    return String(value);
  }
}

function parseJsonField(raw: string): unknown {
  const trimmed = raw.trim();
  if (!trimmed) return {};
  try {
    return JSON.parse(trimmed);
  } catch {
    return raw;
  }
}

const defaultLabelStyle: CSSProperties = {
  display: 'block',
  fontSize: 11,
  color: 'var(--text-secondary, #a3a3a3)',
  marginBottom: 8,
};

const secondaryBtnStyle: CSSProperties = {
  border: '1px solid var(--border-color, #444)',
  borderRadius: 6,
  backgroundColor: 'transparent',
  color: 'var(--text-primary, #eee)',
  fontSize: 11,
  padding: '4px 8px',
  cursor: 'pointer',
};
