import { useCallback, useEffect, useMemo, useState } from 'react';
import type { ChatAPI } from '../../api/chatAPI';
import type { ConnectorProfile } from '../../types/protocol';
import {
  buildConnectorConfig,
  connectorListDetail,
  validateConnectorForm,
  type ConnectorFormFields,
} from './connectorFormUtils';

interface ConnectorsSettingsTabProps {
  api: ChatAPI;
}

const emptyFields = (): Omit<ConnectorFormFields, 'type'> => ({
  brokerUrl: '',
  clientId: '',
  username: '',
  brokers: '',
  groupId: '',
  webhookUrl: '',
  smtpHost: '',
  smtpPort: '587',
  smtpFrom: '',
  smsUrl: '',
  smsFrom: '',
  smsFormat: '',
  secret: '',
});

export function ConnectorsSettingsTab({ api }: ConnectorsSettingsTabProps) {
  const [profiles, setProfiles] = useState<ConnectorProfile[]>([]);
  const [label, setLabel] = useState('');
  const [type, setType] = useState('webhook');
  const [fields, setFields] = useState(emptyFields);
  const [error, setError] = useState('');
  const [deletingId, setDeletingId] = useState<string | null>(null);

  const formFields: ConnectorFormFields = useMemo(
    () => ({ type, ...fields }),
    [type, fields]
  );

  const load = useCallback(async () => {
    try {
      setProfiles(await api.listConnectors());
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }, [api]);

  useEffect(() => {
    void load();
  }, [load]);

  const patchField = <K extends keyof ReturnType<typeof emptyFields>>(
    key: K,
    value: ReturnType<typeof emptyFields>[K]
  ) => {
    setFields((prev) => ({ ...prev, [key]: value }));
  };

  const validationError = validateConnectorForm(formFields);
  const canSave = !validationError;

  const add = async () => {
    setError('');
    const err = validateConnectorForm(formFields);
    if (err) {
      setError(err);
      return;
    }
    try {
      await api.saveConnector(
        {
          id: '',
          type,
          label: label.trim() || type,
          secret: fields.secret,
          config: buildConnectorConfig(formFields),
        },
        true
      );
      setLabel('');
      setFields(emptyFields());
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  };

  const remove = async (id: string, label: string) => {
    const name = label.trim() || id;
    if (!window.confirm(`Remove connector “${name}”? This cannot be undone.`)) {
      return;
    }
    setError('');
    setDeletingId(id);
    // Optimistic remove so the row disappears even if a slow reload follows.
    setProfiles((prev) => prev.filter((p) => p.id !== id));
    try {
      await api.deleteConnector(id);
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      await load();
    } finally {
      setDeletingId(null);
    }
  };

  const inputClass =
    'w-full px-2 py-1 text-sm border border-slack-border rounded bg-slack-bgHover';

  return (
    <div className="space-y-4">
      <p className="text-sm text-slack-textMuted">
        Connector profiles store secrets outside runbook JSON. Reference by ID in action tasks and
        stream subscriptions. For email/SMS runbook actions, add an Email (SMTP) or SMS connector
        here, then pick it in the runbook editor.
      </p>
      {error ? <p className="text-xs text-red-400">{error}</p> : null}
      {profiles.length === 0 ? (
        <p className="text-sm text-slack-textMuted">No connectors yet. Add one below.</p>
      ) : (
        <ul className="space-y-2">
          {profiles.map((p) => {
            const detail = connectorListDetail(p.config, p.type);
            return (
              <li key={p.id} className="text-sm border border-slack-border rounded p-3 space-y-2">
                <div className="min-w-0">
                  <div>
                    <span className="font-medium">{p.label}</span>
                    <span className="text-slack-textMuted ml-2">{p.type}</span>
                    {p.secret_set ? (
                      <span className="text-xs text-green-500 ml-2">secret set</span>
                    ) : null}
                  </div>
                  {detail ? (
                    <div className="text-xs text-slack-textMuted break-all mt-1">{detail}</div>
                  ) : null}
                  <div className="text-[10px] text-slack-textMuted/70 mt-1 font-mono break-all">
                    id: {p.id}
                  </div>
                </div>
                <button
                  type="button"
                  data-testid={`connector-remove-${p.id}`}
                  className="px-3 py-1.5 text-sm rounded border border-red-500/50 text-red-400 hover:bg-red-500/10 disabled:opacity-50"
                  disabled={deletingId === p.id}
                  onClick={() => void remove(p.id, p.label)}
                >
                  {deletingId === p.id ? 'Removing…' : 'Remove'}
                </button>
              </li>
            );
          })}
        </ul>
      )}
      <div className="border border-slack-border rounded p-3 space-y-2">
        <h4 className="text-sm font-medium">Add connector</h4>
        <input
          className={inputClass}
          placeholder="Label"
          value={label}
          onChange={(e) => setLabel(e.target.value)}
        />
        <select
          className={inputClass}
          value={type}
          onChange={(e) => setType(e.target.value)}
        >
          <option value="webhook">Webhook</option>
          <option value="http_auth">HTTP auth</option>
          <option value="mqtt">MQTT</option>
          <option value="kafka">Kafka</option>
          <option value="slack">Slack</option>
          <option value="sms">SMS</option>
          <option value="email">Email (SMTP)</option>
        </select>
        {type === 'mqtt' ? (
          <>
            <input
              className={inputClass}
              placeholder="Broker URL (tcp://host:1883 or ssl://…)"
              value={fields.brokerUrl}
              onChange={(e) => patchField('brokerUrl', e.target.value)}
            />
            <input
              className={inputClass}
              placeholder="Client ID (optional)"
              value={fields.clientId}
              onChange={(e) => patchField('clientId', e.target.value)}
            />
            <input
              className={inputClass}
              placeholder="Username (optional)"
              value={fields.username}
              onChange={(e) => patchField('username', e.target.value)}
            />
          </>
        ) : null}
        {type === 'kafka' ? (
          <>
            <input
              className={inputClass}
              placeholder="Brokers (host:9092,host2:9092)"
              value={fields.brokers}
              onChange={(e) => patchField('brokers', e.target.value)}
            />
            <input
              className={inputClass}
              placeholder="Consumer group ID (optional)"
              value={fields.groupId}
              onChange={(e) => patchField('groupId', e.target.value)}
            />
            <input
              className={inputClass}
              placeholder="SASL username (optional)"
              value={fields.username}
              onChange={(e) => patchField('username', e.target.value)}
            />
          </>
        ) : null}
        {type === 'webhook' || type === 'http_auth' ? (
          <input
            className={inputClass}
            placeholder={type === 'webhook' ? 'Webhook URL (optional)' : 'Auth endpoint URL (optional)'}
            value={fields.webhookUrl}
            onChange={(e) => patchField('webhookUrl', e.target.value)}
          />
        ) : null}
        {type === 'email' ? (
          <>
            <input
              className={inputClass}
              placeholder="SMTP host (required)"
              value={fields.smtpHost}
              onChange={(e) => patchField('smtpHost', e.target.value)}
            />
            <input
              className={inputClass}
              placeholder="Port (default 587)"
              value={fields.smtpPort}
              onChange={(e) => patchField('smtpPort', e.target.value)}
            />
            <input
              className={inputClass}
              placeholder="Username"
              value={fields.username}
              onChange={(e) => patchField('username', e.target.value)}
            />
            <input
              className={inputClass}
              placeholder="From address (optional; defaults to username)"
              value={fields.smtpFrom}
              onChange={(e) => patchField('smtpFrom', e.target.value)}
            />
            <p className="text-xs text-slack-textMuted">
              Use your own mail server (Postfix, Workspace, SES SMTP, etc.). Secret is the SMTP
              password.
            </p>
          </>
        ) : null}
        {type === 'sms' ? (
          <>
            <input
              className={inputClass}
              placeholder="Gateway URL (required) — your HTTP SMS endpoint"
              value={fields.smsUrl}
              onChange={(e) => patchField('smsUrl', e.target.value)}
            />
            <input
              className={inputClass}
              placeholder="From / sender id (optional)"
              value={fields.smsFrom}
              onChange={(e) => patchField('smsFrom', e.target.value)}
            />
            <input
              className={inputClass}
              placeholder="Format: form (default) or json"
              value={fields.smsFormat}
              onChange={(e) => patchField('smsFormat', e.target.value)}
            />
            <p className="text-xs text-slack-textMuted">
              Hub POSTs to this URL. Point it at your own gateway (modem companion, self-hosted
              bridge, etc.).
            </p>
          </>
        ) : null}
        <input
          className={inputClass}
          placeholder={
            type === 'email'
              ? 'SMTP password'
              : type === 'mqtt' || type === 'kafka'
                ? 'Password / SASL secret'
                : type === 'sms'
                  ? 'Auth token (optional Bearer/Basic)'
                  : 'Secret / token'
          }
          type="password"
          value={fields.secret}
          onChange={(e) => patchField('secret', e.target.value)}
        />
        {validationError && (type === 'email' || type === 'sms') ? (
          <p className="text-xs text-slack-textMuted">{validationError}</p>
        ) : null}
        <button
          type="button"
          className="px-3 py-1.5 text-sm rounded bg-slack-accent text-white disabled:opacity-50"
          disabled={!canSave}
          onClick={() => void add()}
        >
          Save connector
        </button>
      </div>
    </div>
  );
}
