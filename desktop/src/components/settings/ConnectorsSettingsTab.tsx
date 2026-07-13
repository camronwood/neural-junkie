import { useCallback, useEffect, useState } from 'react';
import type { ChatAPI } from '../../api/chatAPI';
import type { ConnectorProfile } from '../../types/protocol';

interface ConnectorsSettingsTabProps {
  api: ChatAPI;
}

export function ConnectorsSettingsTab({ api }: ConnectorsSettingsTabProps) {
  const [profiles, setProfiles] = useState<ConnectorProfile[]>([]);
  const [label, setLabel] = useState('');
  const [type, setType] = useState('webhook');
  const [secret, setSecret] = useState('');
  const [brokerUrl, setBrokerUrl] = useState('');
  const [clientId, setClientId] = useState('');
  const [username, setUsername] = useState('');
  const [brokers, setBrokers] = useState('');
  const [groupId, setGroupId] = useState('');
  const [webhookUrl, setWebhookUrl] = useState('');
  const [error, setError] = useState('');

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

  const buildConfig = (): Record<string, string> => {
    if (type === 'mqtt') {
      const cfg: Record<string, string> = {};
      if (brokerUrl.trim()) cfg.broker_url = brokerUrl.trim();
      if (clientId.trim()) cfg.client_id = clientId.trim();
      if (username.trim()) cfg.username = username.trim();
      return cfg;
    }
    if (type === 'kafka') {
      const cfg: Record<string, string> = {};
      if (brokers.trim()) cfg.brokers = brokers.trim();
      if (groupId.trim()) cfg.group_id = groupId.trim();
      if (username.trim()) cfg.username = username.trim();
      return cfg;
    }
    if (type === 'webhook' && webhookUrl.trim()) {
      return { url: webhookUrl.trim() };
    }
    return {};
  };

  const add = async () => {
    setError('');
    try {
      await api.saveConnector(
        {
          id: '',
          type,
          label: label.trim() || type,
          secret,
          config: buildConfig(),
        },
        true
      );
      setLabel('');
      setSecret('');
      setBrokerUrl('');
      setClientId('');
      setUsername('');
      setBrokers('');
      setGroupId('');
      setWebhookUrl('');
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  };

  return (
    <div className="space-y-4">
      <p className="text-sm text-slack-textMuted">
        Connector profiles store secrets outside runbook JSON. Reference by ID in action tasks and stream subscriptions.
      </p>
      {error ? <p className="text-xs text-red-400">{error}</p> : null}
      <ul className="space-y-2">
        {profiles.map((p) => (
          <li key={p.id} className="text-sm border border-slack-border rounded p-2">
            <span className="font-medium">{p.label}</span>
            <span className="text-slack-textMuted ml-2">{p.type}</span>
            {p.secret_set ? <span className="text-xs text-green-500 ml-2">secret set</span> : null}
            {p.config?.broker_url ? (
              <span className="text-xs text-slack-textMuted ml-2">{p.config.broker_url}</span>
            ) : null}
            {p.config?.brokers ? (
              <span className="text-xs text-slack-textMuted ml-2">{p.config.brokers}</span>
            ) : null}
          </li>
        ))}
      </ul>
      <div className="border border-slack-border rounded p-3 space-y-2">
        <h4 className="text-sm font-medium">Add connector</h4>
        <input
          className="w-full px-2 py-1 text-sm border border-slack-border rounded bg-slack-bgHover"
          placeholder="Label"
          value={label}
          onChange={(e) => setLabel(e.target.value)}
        />
        <select
          className="w-full px-2 py-1 text-sm border border-slack-border rounded bg-slack-bgHover"
          value={type}
          onChange={(e) => setType(e.target.value)}
        >
          <option value="webhook">Webhook</option>
          <option value="http_auth">HTTP auth</option>
          <option value="mqtt">MQTT</option>
          <option value="kafka">Kafka</option>
          <option value="slack">Slack</option>
          <option value="sms">SMS</option>
        </select>
        {type === 'mqtt' ? (
          <>
            <input
              className="w-full px-2 py-1 text-sm border border-slack-border rounded bg-slack-bgHover"
              placeholder="Broker URL (tcp://host:1883 or ssl://…)"
              value={brokerUrl}
              onChange={(e) => setBrokerUrl(e.target.value)}
            />
            <input
              className="w-full px-2 py-1 text-sm border border-slack-border rounded bg-slack-bgHover"
              placeholder="Client ID (optional)"
              value={clientId}
              onChange={(e) => setClientId(e.target.value)}
            />
            <input
              className="w-full px-2 py-1 text-sm border border-slack-border rounded bg-slack-bgHover"
              placeholder="Username (optional)"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
            />
          </>
        ) : null}
        {type === 'kafka' ? (
          <>
            <input
              className="w-full px-2 py-1 text-sm border border-slack-border rounded bg-slack-bgHover"
              placeholder="Brokers (host:9092,host2:9092)"
              value={brokers}
              onChange={(e) => setBrokers(e.target.value)}
            />
            <input
              className="w-full px-2 py-1 text-sm border border-slack-border rounded bg-slack-bgHover"
              placeholder="Consumer group ID (optional)"
              value={groupId}
              onChange={(e) => setGroupId(e.target.value)}
            />
            <input
              className="w-full px-2 py-1 text-sm border border-slack-border rounded bg-slack-bgHover"
              placeholder="SASL username (optional)"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
            />
          </>
        ) : null}
        {type === 'webhook' ? (
          <input
            className="w-full px-2 py-1 text-sm border border-slack-border rounded bg-slack-bgHover"
            placeholder="Webhook URL (optional)"
            value={webhookUrl}
            onChange={(e) => setWebhookUrl(e.target.value)}
          />
        ) : null}
        <input
          className="w-full px-2 py-1 text-sm border border-slack-border rounded bg-slack-bgHover"
          placeholder={
            type === 'mqtt' || type === 'kafka'
              ? 'Password / SASL secret'
              : 'Secret / token'
          }
          type="password"
          value={secret}
          onChange={(e) => setSecret(e.target.value)}
        />
        <button
          type="button"
          className="px-3 py-1.5 text-sm rounded bg-slack-accent text-white"
          onClick={() => void add()}
        >
          Save connector
        </button>
      </div>
    </div>
  );
}
