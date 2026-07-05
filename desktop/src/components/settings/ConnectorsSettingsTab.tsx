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

  const add = async () => {
    setError('');
    try {
      await api.saveConnector({ id: '', type, label: label.trim() || type, secret }, true);
      setLabel('');
      setSecret('');
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  };

  return (
    <div className="space-y-4">
      <p className="text-sm text-slack-textMuted">
        Connector profiles store secrets outside runbook JSON. Reference by ID in action tasks.
      </p>
      {error ? <p className="text-xs text-red-400">{error}</p> : null}
      <ul className="space-y-2">
        {profiles.map((p) => (
          <li key={p.id} className="text-sm border border-slack-border rounded p-2">
            <span className="font-medium">{p.label}</span>
            <span className="text-slack-textMuted ml-2">{p.type}</span>
            {p.secret_set ? <span className="text-xs text-green-500 ml-2">secret set</span> : null}
          </li>
        ))}
      </ul>
      <div className="border border-slack-border rounded p-3 space-y-2">
        <h4 className="text-sm font-medium">Add connector</h4>
        <input className="w-full px-2 py-1 text-sm border border-slack-border rounded bg-slack-bgHover" placeholder="Label" value={label} onChange={(e) => setLabel(e.target.value)} />
        <select className="w-full px-2 py-1 text-sm border border-slack-border rounded bg-slack-bgHover" value={type} onChange={(e) => setType(e.target.value)}>
          <option value="webhook">Webhook</option>
          <option value="http_auth">HTTP auth</option>
          <option value="slack">Slack</option>
          <option value="sms">SMS</option>
        </select>
        <input className="w-full px-2 py-1 text-sm border border-slack-border rounded bg-slack-bgHover" placeholder="Secret / token" type="password" value={secret} onChange={(e) => setSecret(e.target.value)} />
        <button type="button" className="px-3 py-1.5 text-sm rounded bg-slack-accent text-white" onClick={() => void add()}>
          Save connector
        </button>
      </div>
    </div>
  );
}
