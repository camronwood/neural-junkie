import { useCallback, useEffect, useMemo, useState } from 'react';
import { ChatAPI } from '../../api/chatAPI';
import type {
  AgentInfo,
  Channel,
  ConnectorProfile,
  RunbookDefinitionSummary,
  StreamActionType,
  StreamManagerStatus,
  StreamProtocol,
  StreamSubscription,
} from '../../types/protocol';
import type { SettingsTabProps } from './settingsShared';

const emptyForm = (): StreamSubscription => ({
  id: '',
  label: '',
  enabled: true,
  protocol: 'mqtt',
  connector_id: '',
  topic: '',
  debounce_ms: 0,
  match: { json_path: '', op: 'equals', value: '' },
  action: {
    type: 'runbook',
    definition_id: '',
    agent_ids: [],
    channel: 'general',
    hub_channel: 'general',
    message_template: 'Stream event on {{topic}}:\n{{payload}}',
    webhook_connector_id: '',
    url_override: '',
  },
});

export function StreamsSettingsTab({ hubHttp, isActive }: SettingsTabProps) {
  const api = useMemo(() => new ChatAPI(hubHttp), [hubHttp]);
  const [status, setStatus] = useState<StreamManagerStatus | null>(null);
  const [subs, setSubs] = useState<StreamSubscription[]>([]);
  const [connectors, setConnectors] = useState<ConnectorProfile[]>([]);
  const [definitions, setDefinitions] = useState<RunbookDefinitionSummary[]>([]);
  const [agents, setAgents] = useState<AgentInfo[]>([]);
  const [channels, setChannels] = useState<Channel[]>([]);
  const [form, setForm] = useState<StreamSubscription>(emptyForm);
  const [editing, setEditing] = useState(false);
  const [error, setError] = useState('');
  const [feedback, setFeedback] = useState('');
  const [testPayload, setTestPayload] = useState('{"status":"ok"}');
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    try {
      const [st, list, conn, defs, ags, chs] = await Promise.all([
        api.getStreamStatus(),
        api.listStreamSubscriptions(),
        api.listConnectors(),
        api.listRunbookDefinitions(),
        api.fetchAgents(),
        api.fetchChannels(),
      ]);
      setStatus(st);
      setSubs(list);
      setConnectors(conn);
      setDefinitions(defs);
      setAgents(ags.filter((a) => a.status !== 'removed'));
      setChannels(chs);
      setError('');
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }, [api]);

  useEffect(() => {
    if (!isActive) return;
    void load();
  }, [isActive, load]);

  const statusById = useMemo(() => {
    const m = new Map<string, StreamManagerStatus['subscriptions'][number]>();
    for (const s of status?.subscriptions ?? []) {
      m.set(s.subscription_id, s);
    }
    return m;
  }, [status]);

  if (!isActive) return null;

  const brokerConnectors = connectors.filter((c) => c.type === 'mqtt' || c.type === 'kafka');
  const webhookConnectors = connectors.filter((c) => c.type === 'webhook' || c.type === 'http_auth');
  const protocolConnectors = brokerConnectors.filter((c) => c.type === form.protocol);

  const save = async () => {
    setBusy(true);
    setFeedback('');
    setError('');
    try {
      const payload: StreamSubscription = {
        ...form,
        label: form.label.trim() || `${form.protocol} ${form.topic}`,
        match:
          form.match?.json_path?.trim()
            ? {
                json_path: form.match.json_path.trim(),
                op: form.match.op || 'equals',
                value: form.match.value || '',
              }
            : null,
        action: {
          ...form.action,
          agent_ids: form.action.agent_ids?.filter(Boolean) ?? [],
        },
      };
      await api.saveStreamSubscription(payload, !editing || !form.id);
      setForm(emptyForm());
      setEditing(false);
      setFeedback('Subscription saved. Manager reloaded.');
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  };

  const remove = async (id: string) => {
    setBusy(true);
    setError('');
    try {
      await api.deleteStreamSubscription(id);
      if (form.id === id) {
        setForm(emptyForm());
        setEditing(false);
      }
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  };

  const restart = async () => {
    setBusy(true);
    setError('');
    try {
      await api.restartStreamManager();
      setFeedback('Stream manager restarted.');
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  };

  const testFire = async (id: string) => {
    setBusy(true);
    setError('');
    setFeedback('');
    try {
      const res = await api.testStreamSubscription(id, testPayload);
      setFeedback(
        `Test: matched=${res.matched} fired=${res.fired} skipped=${res.skipped}` +
          (res.reason ? ` reason=${res.reason}` : '') +
          (res.error ? ` error=${res.error}` : '')
      );
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  };

  const setActionType = (type: StreamActionType) => {
    setForm((prev) => ({ ...prev, action: { ...prev.action, type } }));
  };

  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-lg font-semibold text-slack-text">Streams</h3>
        <p className="text-sm text-slack-textMuted mt-1">
          Long-lived MQTT and Kafka subscriptions that trigger a runbook, post to a hub channel, or call a webhook.
        </p>
      </div>

      {error ? <p className="text-xs text-red-400">{error}</p> : null}
      {feedback ? <p className="text-xs text-green-500">{feedback}</p> : null}

      <section className="border border-slack-border rounded p-3 space-y-2">
        <div className="flex items-center justify-between gap-2">
          <div>
            <h4 className="text-sm font-medium">Manager</h4>
            <p className="text-xs text-slack-textMuted">
              {status?.running ? 'Running' : 'Stopped'} · {subs.filter((s) => s.enabled).length} enabled
            </p>
          </div>
          <button
            type="button"
            disabled={busy}
            className="px-3 py-1.5 text-sm rounded bg-slack-accent text-white disabled:opacity-50"
            onClick={() => void restart()}
          >
            Restart
          </button>
        </div>
      </section>

      <section className="border border-slack-border rounded p-3 space-y-2">
        <h4 className="text-sm font-medium">Broker connectors</h4>
        <p className="text-xs text-slack-textMuted">
          Create MQTT/Kafka connectors under Integrations → Connectors (or below via Settings → Integrations).
        </p>
        {brokerConnectors.length === 0 ? (
          <p className="text-xs text-amber-500">No mqtt/kafka connectors yet.</p>
        ) : (
          <ul className="space-y-1">
            {brokerConnectors.map((c) => (
              <li key={c.id} className="text-sm">
                <span className="font-medium">{c.label}</span>
                <span className="text-slack-textMuted ml-2">{c.type}</span>
              </li>
            ))}
          </ul>
        )}
      </section>

      <section className="border border-slack-border rounded p-3 space-y-3">
        <h4 className="text-sm font-medium">Subscriptions</h4>
        {subs.length === 0 ? (
          <p className="text-xs text-slack-textMuted">No subscriptions configured.</p>
        ) : (
          <ul className="space-y-2">
            {subs.map((sub) => {
              const st = statusById.get(sub.id);
              return (
                <li key={sub.id} className="border border-slack-border rounded p-2 space-y-1">
                  <div className="flex flex-wrap items-center gap-2 justify-between">
                    <div>
                      <span className="font-medium text-sm">{sub.label || sub.id}</span>
                      <span className="text-xs text-slack-textMuted ml-2">
                        {sub.protocol} · {sub.topic} · {sub.action.type}
                      </span>
                      {!sub.enabled ? (
                        <span className="text-xs text-amber-500 ml-2">disabled</span>
                      ) : st?.connected ? (
                        <span className="text-xs text-green-500 ml-2">connected</span>
                      ) : (
                        <span className="text-xs text-slack-textMuted ml-2">idle</span>
                      )}
                    </div>
                    <div className="flex gap-2">
                      <button
                        type="button"
                        className="text-xs px-2 py-1 rounded border border-slack-border"
                        onClick={() => {
                          setForm({
                            ...emptyForm(),
                            ...sub,
                            match: sub.match ?? { json_path: '', op: 'equals', value: '' },
                            action: { ...emptyForm().action, ...sub.action },
                          });
                          setEditing(true);
                        }}
                      >
                        Edit
                      </button>
                      <button
                        type="button"
                        className="text-xs px-2 py-1 rounded border border-slack-border"
                        disabled={busy}
                        onClick={() => void testFire(sub.id)}
                      >
                        Test
                      </button>
                      <button
                        type="button"
                        className="text-xs px-2 py-1 rounded border border-red-500/40 text-red-400"
                        disabled={busy}
                        onClick={() => void remove(sub.id)}
                      >
                        Delete
                      </button>
                    </div>
                  </div>
                  {st?.last_error ? (
                    <p className="text-xs text-red-400">Last error: {st.last_error}</p>
                  ) : null}
                  <p className="text-xs text-slack-textMuted">
                    fires={st?.fire_count ?? 0} skips={st?.skip_count ?? 0}
                  </p>
                </li>
              );
            })}
          </ul>
        )}
        <div>
          <label className="text-xs text-slack-textMuted block mb-1">Test payload (JSON)</label>
          <textarea
            className="w-full px-2 py-1 text-sm border border-slack-border rounded bg-slack-bgHover font-mono"
            rows={3}
            value={testPayload}
            onChange={(e) => setTestPayload(e.target.value)}
          />
        </div>
      </section>

      <section className="border border-slack-border rounded p-3 space-y-2">
        <h4 className="text-sm font-medium">{editing ? 'Edit subscription' : 'Add subscription'}</h4>
        <input
          className="w-full px-2 py-1 text-sm border border-slack-border rounded bg-slack-bgHover"
          placeholder="Label"
          value={form.label}
          onChange={(e) => setForm((p) => ({ ...p, label: e.target.value }))}
        />
        <div className="grid grid-cols-2 gap-2">
          <select
            className="w-full px-2 py-1 text-sm border border-slack-border rounded bg-slack-bgHover"
            value={form.protocol}
            onChange={(e) =>
              setForm((p) => ({
                ...p,
                protocol: e.target.value as StreamProtocol,
                connector_id: '',
              }))
            }
          >
            <option value="mqtt">MQTT</option>
            <option value="kafka">Kafka</option>
          </select>
          <label className="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={form.enabled}
              onChange={(e) => setForm((p) => ({ ...p, enabled: e.target.checked }))}
            />
            Enabled
          </label>
        </div>
        <select
          className="w-full px-2 py-1 text-sm border border-slack-border rounded bg-slack-bgHover"
          value={form.connector_id}
          onChange={(e) => setForm((p) => ({ ...p, connector_id: e.target.value }))}
        >
          <option value="">Select connector…</option>
          {protocolConnectors.map((c) => (
            <option key={c.id} value={c.id}>
              {c.label} ({c.type})
            </option>
          ))}
        </select>
        <input
          className="w-full px-2 py-1 text-sm border border-slack-border rounded bg-slack-bgHover"
          placeholder="Topic"
          value={form.topic}
          onChange={(e) => setForm((p) => ({ ...p, topic: e.target.value }))}
        />
        <div className="grid grid-cols-3 gap-2">
          <input
            className="px-2 py-1 text-sm border border-slack-border rounded bg-slack-bgHover"
            placeholder="Match JSON path"
            value={form.match?.json_path || ''}
            onChange={(e) =>
              setForm((p) => ({
                ...p,
                match: { ...(p.match || {}), json_path: e.target.value, op: p.match?.op || 'equals', value: p.match?.value || '' },
              }))
            }
          />
          <select
            className="px-2 py-1 text-sm border border-slack-border rounded bg-slack-bgHover"
            value={form.match?.op || 'equals'}
            onChange={(e) =>
              setForm((p) => ({
                ...p,
                match: {
                  json_path: p.match?.json_path || '',
                  op: e.target.value as 'equals' | 'contains',
                  value: p.match?.value || '',
                },
              }))
            }
          >
            <option value="equals">equals</option>
            <option value="contains">contains</option>
          </select>
          <input
            className="px-2 py-1 text-sm border border-slack-border rounded bg-slack-bgHover"
            placeholder="Match value"
            value={form.match?.value || ''}
            onChange={(e) =>
              setForm((p) => ({
                ...p,
                match: {
                  json_path: p.match?.json_path || '',
                  op: p.match?.op || 'equals',
                  value: e.target.value,
                },
              }))
            }
          />
        </div>
        <input
          className="w-full px-2 py-1 text-sm border border-slack-border rounded bg-slack-bgHover"
          type="number"
          min={0}
          placeholder="Debounce ms"
          value={form.debounce_ms || 0}
          onChange={(e) => setForm((p) => ({ ...p, debounce_ms: Number(e.target.value) || 0 }))}
        />

        <div className="flex flex-wrap gap-3 text-sm">
          {(['runbook', 'channel', 'webhook'] as StreamActionType[]).map((t) => (
            <label key={t} className="flex items-center gap-1">
              <input type="radio" name="stream-action" checked={form.action.type === t} onChange={() => setActionType(t)} />
              {t}
            </label>
          ))}
        </div>

        {form.action.type === 'runbook' ? (
          <div className="space-y-2 border-t border-slack-border pt-2">
            <select
              className="w-full px-2 py-1 text-sm border border-slack-border rounded bg-slack-bgHover"
              value={form.action.definition_id || ''}
              onChange={(e) =>
                setForm((p) => ({ ...p, action: { ...p.action, definition_id: e.target.value } }))
              }
            >
              <option value="">Runbook definition…</option>
              {definitions.map((d) => (
                <option key={d.id} value={d.id}>
                  {d.title || d.id}
                </option>
              ))}
            </select>
            <select
              className="w-full px-2 py-1 text-sm border border-slack-border rounded bg-slack-bgHover"
              value={form.action.agent_ids?.[0] || ''}
              onChange={(e) =>
                setForm((p) => ({
                  ...p,
                  action: { ...p.action, agent_ids: e.target.value ? [e.target.value] : [] },
                }))
              }
            >
              <option value="">Agent…</option>
              {agents.map((a) => (
                <option key={a.id} value={a.id}>
                  {a.name}
                </option>
              ))}
            </select>
            <select
              className="w-full px-2 py-1 text-sm border border-slack-border rounded bg-slack-bgHover"
              value={form.action.channel || 'general'}
              onChange={(e) =>
                setForm((p) => ({ ...p, action: { ...p.action, channel: e.target.value } }))
              }
            >
              {channels.length === 0 ? <option value="general">general</option> : null}
              {channels.map((c) => (
                <option key={c.name} value={c.name}>
                  {c.name}
                </option>
              ))}
            </select>
          </div>
        ) : null}

        {form.action.type === 'channel' ? (
          <div className="space-y-2 border-t border-slack-border pt-2">
            <select
              className="w-full px-2 py-1 text-sm border border-slack-border rounded bg-slack-bgHover"
              value={form.action.hub_channel || 'general'}
              onChange={(e) =>
                setForm((p) => ({ ...p, action: { ...p.action, hub_channel: e.target.value } }))
              }
            >
              {channels.length === 0 ? <option value="general">general</option> : null}
              {channels.map((c) => (
                <option key={c.name} value={c.name}>
                  {c.name}
                </option>
              ))}
            </select>
            <textarea
              className="w-full px-2 py-1 text-sm border border-slack-border rounded bg-slack-bgHover font-mono"
              rows={3}
              placeholder="Message template ({{payload}} {{topic}} {{key}})"
              value={form.action.message_template || ''}
              onChange={(e) =>
                setForm((p) => ({ ...p, action: { ...p.action, message_template: e.target.value } }))
              }
            />
            <select
              className="w-full px-2 py-1 text-sm border border-slack-border rounded bg-slack-bgHover"
              value={form.action.mention_agent_ids?.[0] || ''}
              onChange={(e) =>
                setForm((p) => ({
                  ...p,
                  action: {
                    ...p.action,
                    mention_agent_ids: e.target.value ? [e.target.value] : [],
                  },
                }))
              }
            >
              <option value="">Mention agent (optional)…</option>
              {agents.map((a) => (
                <option key={a.id} value={a.id}>
                  {a.name}
                </option>
              ))}
            </select>
          </div>
        ) : null}

        {form.action.type === 'webhook' ? (
          <div className="space-y-2 border-t border-slack-border pt-2">
            <select
              className="w-full px-2 py-1 text-sm border border-slack-border rounded bg-slack-bgHover"
              value={form.action.webhook_connector_id || ''}
              onChange={(e) =>
                setForm((p) => ({
                  ...p,
                  action: { ...p.action, webhook_connector_id: e.target.value },
                }))
              }
            >
              <option value="">Webhook connector (optional)…</option>
              {webhookConnectors.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.label}
                </option>
              ))}
            </select>
            <input
              className="w-full px-2 py-1 text-sm border border-slack-border rounded bg-slack-bgHover"
              placeholder="URL override"
              value={form.action.url_override || ''}
              onChange={(e) =>
                setForm((p) => ({ ...p, action: { ...p.action, url_override: e.target.value } }))
              }
            />
            <textarea
              className="w-full px-2 py-1 text-sm border border-slack-border rounded bg-slack-bgHover font-mono"
              rows={2}
              placeholder="Body template (optional; default JSON envelope)"
              value={form.action.body_template || ''}
              onChange={(e) =>
                setForm((p) => ({ ...p, action: { ...p.action, body_template: e.target.value } }))
              }
            />
          </div>
        ) : null}

        <div className="flex gap-2">
          <button
            type="button"
            disabled={busy}
            className="px-3 py-1.5 text-sm rounded bg-slack-accent text-white disabled:opacity-50"
            onClick={() => void save()}
          >
            {editing ? 'Update' : 'Save'} subscription
          </button>
          {editing ? (
            <button
              type="button"
              className="px-3 py-1.5 text-sm rounded border border-slack-border"
              onClick={() => {
                setForm(emptyForm());
                setEditing(false);
              }}
            >
              Cancel
            </button>
          ) : null}
        </div>
      </section>
    </div>
  );
}
