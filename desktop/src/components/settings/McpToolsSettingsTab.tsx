import { useCallback, useEffect, useState } from 'react';
import type { ChatAPI } from '../../api/chatAPI';
import type { UserMCPServer, UserMCPTool } from '../../api/domains/userToolsApi';

interface McpToolsSettingsTabProps {
  api: ChatAPI;
}

/** MCP Tool Wizard — create HTTP tools, test, grant to custom experts; register remote MCP URLs. */
export function McpToolsSettingsTab({ api }: McpToolsSettingsTabProps) {
  const [tools, setTools] = useState<UserMCPTool[]>([]);
  const [servers, setServers] = useState<UserMCPServer[]>([]);
  const [error, setError] = useState('');
  const [testOut, setTestOut] = useState('');
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [url, setUrl] = useState('');
  const [jsonPath, setJsonPath] = useState('');
  const [grantName, setGrantName] = useState('');
  const [serverName, setServerName] = useState('');
  const [serverUrl, setServerUrl] = useState('');
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    try {
      const [t, s] = await Promise.all([api.listUserTools(), api.listUserServers()]);
      setTools(t);
      setServers(s);
      setError('');
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }, [api]);

  useEffect(() => {
    void load();
  }, [load]);

  const createTool = async () => {
    setBusy(true);
    setError('');
    try {
      await api.createUserTool({
        name: name.trim(),
        description: description.trim(),
        url: url.trim(),
        method: 'GET',
        json_path: jsonPath.trim() || undefined,
      });
      setName('');
      setDescription('');
      setUrl('');
      setJsonPath('');
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  };

  const testTool = async (id: string) => {
    setTestOut('');
    try {
      const r = await api.testUserTool(id);
      setTestOut(r.success ? String(r.result ?? '') : r.error || 'test failed');
    } catch (e) {
      setTestOut(e instanceof Error ? e.message : String(e));
    }
  };

  const grant = async (id: string, grant: boolean) => {
    const agent = grantName.trim();
    if (!agent) {
      setError('Enter a custom expert display name to grant/revoke.');
      return;
    }
    try {
      await api.grantUserTool(id, agent, grant);
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  };

  const addServer = async () => {
    setBusy(true);
    setError('');
    try {
      await api.createUserServer({
        name: serverName.trim(),
        url: serverUrl.trim(),
        transport_type: 'http',
      });
      setServerName('');
      setServerUrl('');
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="max-w-3xl space-y-8">
      <div>
        <h3 className="text-lg font-semibold text-slack-text">MCP Tool Wizard</h3>
        <p className="mt-1 text-sm text-slack-textMuted">
          Create HTTP-fetch tools, test them, and grant access to custom experts by name. Register remote
          MCP server URLs for later dialing (registry persisted on the hub).
        </p>
      </div>

      {error ? <p className="text-sm text-red-400">{error}</p> : null}

      <section className="space-y-3">
        <h4 className="text-sm font-semibold text-slack-text">New HTTP tool</h4>
        <input
          className="w-full px-3 py-2 bg-slack-bg border border-slack-border rounded text-sm text-slack-text"
          placeholder="Name"
          value={name}
          onChange={(e) => setName(e.target.value)}
        />
        <input
          className="w-full px-3 py-2 bg-slack-bg border border-slack-border rounded text-sm text-slack-text"
          placeholder="Description"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
        />
        <input
          className="w-full px-3 py-2 bg-slack-bg border border-slack-border rounded text-sm text-slack-text"
          placeholder="https://api.example.com/…"
          value={url}
          onChange={(e) => setUrl(e.target.value)}
        />
        <input
          className="w-full px-3 py-2 bg-slack-bg border border-slack-border rounded text-sm text-slack-text"
          placeholder="JSON path (optional) e.g. data.items.0.title"
          value={jsonPath}
          onChange={(e) => setJsonPath(e.target.value)}
        />
        <button
          type="button"
          disabled={busy || !name.trim() || !url.trim()}
          className="px-3 py-1.5 rounded bg-slack-accent text-white text-sm disabled:opacity-50"
          onClick={() => void createTool()}
        >
          Create tool
        </button>
      </section>

      <section className="space-y-2">
        <h4 className="text-sm font-semibold text-slack-text">Your tools</h4>
        <div className="flex gap-2 items-center mb-2">
          <input
            className="flex-1 px-3 py-2 bg-slack-bg border border-slack-border rounded text-sm text-slack-text"
            placeholder="Custom expert name for grant/revoke"
            value={grantName}
            onChange={(e) => setGrantName(e.target.value)}
          />
        </div>
        {tools.length === 0 ? (
          <p className="text-sm text-slack-textMuted">No user tools yet.</p>
        ) : (
          tools.map((t) => (
            <div key={t.id} className="p-3 rounded border border-slack-border space-y-2">
              <div className="font-medium text-sm text-slack-text">{t.name}</div>
              <div className="text-xs text-slack-textMuted truncate">{t.url}</div>
              <div className="text-xs text-slack-textMuted">
                Granted: {(t.granted_agents || []).join(', ') || '—'}
              </div>
              <div className="flex flex-wrap gap-2">
                <button
                  type="button"
                  className="text-xs px-2 py-1 rounded border border-slack-border"
                  onClick={() => void testTool(t.id)}
                >
                  Test
                </button>
                <button
                  type="button"
                  className="text-xs px-2 py-1 rounded border border-slack-border"
                  onClick={() => void grant(t.id, true)}
                >
                  Grant
                </button>
                <button
                  type="button"
                  className="text-xs px-2 py-1 rounded border border-slack-border"
                  onClick={() => void grant(t.id, false)}
                >
                  Revoke
                </button>
                <button
                  type="button"
                  className="text-xs px-2 py-1 rounded border border-red-500/40 text-red-400"
                  onClick={() => void api.deleteUserTool(t.id).then(load)}
                >
                  Delete
                </button>
              </div>
            </div>
          ))
        )}
        {testOut ? (
          <pre className="text-xs bg-slack-bgHover p-2 rounded overflow-auto max-h-40 whitespace-pre-wrap">
            {testOut}
          </pre>
        ) : null}
      </section>

      <section className="space-y-3">
        <h4 className="text-sm font-semibold text-slack-text">Remote MCP registry</h4>
        <p className="text-xs text-slack-textMuted">
          Persisted list of MCP server endpoints. Dialing/stdio connect lands in a follow-up; registry is
          usable for documentation and upcoming connect wiring.
        </p>
        <input
          className="w-full px-3 py-2 bg-slack-bg border border-slack-border rounded text-sm text-slack-text"
          placeholder="Server name"
          value={serverName}
          onChange={(e) => setServerName(e.target.value)}
        />
        <input
          className="w-full px-3 py-2 bg-slack-bg border border-slack-border rounded text-sm text-slack-text"
          placeholder="https://mcp.example.com/sse"
          value={serverUrl}
          onChange={(e) => setServerUrl(e.target.value)}
        />
        <button
          type="button"
          disabled={busy || !serverName.trim() || !serverUrl.trim()}
          className="px-3 py-1.5 rounded border border-slack-border text-sm disabled:opacity-50"
          onClick={() => void addServer()}
        >
          Add server
        </button>
        {servers.map((s) => (
          <div key={s.id} className="flex items-center justify-between gap-2 p-2 border border-slack-border rounded">
            <div>
              <div className="text-sm text-slack-text">{s.name}</div>
              <div className="text-xs text-slack-textMuted">{s.url}</div>
            </div>
            <button
              type="button"
              className="text-xs text-red-400"
              onClick={() => void api.deleteUserServer(s.id).then(load)}
            >
              Remove
            </button>
          </div>
        ))}
      </section>
    </div>
  );
}
