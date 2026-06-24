import { useCallback, useEffect, useState } from 'react';
import { ChatAPI } from '../../api/chatAPI';
import { useChatStore } from '../../stores/chatStore';
import type { SettingsTabProps } from './settingsShared';

type HubSecurity = {
  hub_token_configured: boolean;
  auth_required: boolean;
  relaxed_local: boolean;
  bootstrap_configured: boolean;
  listen_all: boolean;
  loopback_only: boolean;
};

type APIKeyRecord = {
  id: string;
  name: string;
  role: string;
  prefix: string;
  created_at: string;
  revoked: boolean;
};

function isForbiddenError(message: string | null): boolean {
  if (!message) return false;
  const lower = message.toLowerCase();
  return lower.includes('forbidden') || lower.includes('admin');
}

async function readLocalBootstrapToken(): Promise<string> {
  if (typeof window === 'undefined' || !(window as { __TAURI__?: unknown }).__TAURI__) {
    throw new Error('Local bootstrap read is only available in the desktop app');
  }
  const { invoke } = await import('@tauri-apps/api/tauri');
  const token = await invoke<string>('read_hub_bootstrap_token');
  if (!token.trim()) {
    throw new Error('Bootstrap token file is empty');
  }
  return token.trim();
}

export function SecuritySettingsTab({ hubHttp, isActive }: SettingsTabProps) {
  const username = useChatStore((s) => s.username);
  const [hubSecurity, setHubSecurity] = useState<HubSecurity | null>(null);
  const [apiKeys, setApiKeys] = useState<APIKeyRecord[]>([]);
  const [newKeyName, setNewKeyName] = useState('');
  const [newKeyRole, setNewKeyRole] = useState('member');
  const [createdKey, setCreatedKey] = useState<string | null>(null);
  const [keysError, setKeysError] = useState<string | null>(null);
  const [loadingKeys, setLoadingKeys] = useState(false);
  const [adminUnlocked, setAdminUnlocked] = useState(false);
  const [bootstrapInput, setBootstrapInput] = useState('');
  const [unlockError, setUnlockError] = useState<string | null>(null);
  const [unlocking, setUnlocking] = useState(false);

  const loadApiKeys = useCallback(async () => {
    setLoadingKeys(true);
    setKeysError(null);
    try {
      const api = new ChatAPI(hubHttp);
      const rows = (await api.listAPIKeys()) as APIKeyRecord[];
      setApiKeys(rows.filter((k) => !k.revoked));
      setAdminUnlocked(true);
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e);
      setKeysError(msg);
      setApiKeys([]);
      if (isForbiddenError(msg)) {
        setAdminUnlocked(false);
      }
    } finally {
      setLoadingKeys(false);
    }
  }, [hubHttp]);

  useEffect(() => {
    if (!isActive) return;
    let cancelled = false;
    (async () => {
      try {
        const r = await fetch(`${hubHttp}/api/system/security`);
        if (!r.ok) return;
        const data = await r.json();
        if (!cancelled) setHubSecurity(data);
      } catch {
        if (!cancelled) setHubSecurity(null);
      }
    })();
    void loadApiKeys();
    return () => {
      cancelled = true;
    };
  }, [isActive, hubHttp, loadApiKeys]);

  const handleUnlockAdmin = async (bootstrapToken: string) => {
    setUnlockError(null);
    setUnlocking(true);
    try {
      const api = new ChatAPI(hubHttp);
      const who = (username || 'local').trim() || 'local';
      await api.createAdminSession(who, bootstrapToken);
      setAdminUnlocked(true);
      setBootstrapInput('');
      await loadApiKeys();
    } catch (e) {
      setUnlockError(e instanceof Error ? e.message : String(e));
    } finally {
      setUnlocking(false);
    }
  };

  const handleUseLocalBootstrap = async () => {
    setUnlockError(null);
    try {
      const token = await readLocalBootstrapToken();
      await handleUnlockAdmin(token);
    } catch (e) {
      setUnlockError(e instanceof Error ? e.message : String(e));
    }
  };

  const handleCreateKey = async () => {
    setKeysError(null);
    setCreatedKey(null);
    try {
      const api = new ChatAPI(hubHttp);
      const { api_key } = await api.createAPIKey(newKeyName.trim() || 'automation', newKeyRole);
      setCreatedKey(api_key);
      setNewKeyName('');
      await loadApiKeys();
    } catch (e) {
      setKeysError(e instanceof Error ? e.message : String(e));
    }
  };

  const handleRevokeKey = async (id: string) => {
    setKeysError(null);
    try {
      const api = new ChatAPI(hubHttp);
      await api.revokeAPIKey(id);
      await loadApiKeys();
    } catch (e) {
      setKeysError(e instanceof Error ? e.message : String(e));
    }
  };

  const needsAdminUnlock = !adminUnlocked && (isForbiddenError(keysError) || hubSecurity?.relaxed_local);

  if (!isActive) return null;

  return (
<div className="space-y-6">
    <div>
      <h3 className="text-lg font-semibold text-slack-text mb-2">Hub security</h3>
      <p className="text-sm text-slack-textMuted mb-4">
        Neural Junkie is local-first. The hub binds to loopback by default. For shared machines or LAN
        access, configure environment variables before starting the hub — see{' '}
        <a
          href="https://github.com/camronwood/neural-junkie/blob/main/docs/SECURITY.md"
          className="text-indigo-400 hover:underline"
          target="_blank"
          rel="noreferrer"
        >
          SECURITY.md
        </a>
        .
      </p>
      {hubSecurity ? (
        <ul className="text-sm space-y-2 text-slack-text">
          <li>
            <span className="text-slack-textMuted">Loopback only:</span>{' '}
            {hubSecurity.loopback_only ? 'Yes' : 'No (NEURAL_JUNKIE_LISTEN_ALL=1)'}
          </li>
          <li>
            <span className="text-slack-textMuted">Hub token configured:</span>{' '}
            {hubSecurity.hub_token_configured ? 'Yes' : 'No'}
          </li>
          <li>
            <span className="text-slack-textMuted">Strict auth (NEURAL_JUNKIE_AUTH_REQUIRED):</span>{' '}
            {hubSecurity.auth_required ? 'On' : 'Off'}
          </li>
          <li>
            <span className="text-slack-textMuted">Relaxed local (dev escape hatch):</span>{' '}
            {hubSecurity.relaxed_local ? 'On' : 'Off'}
          </li>
          <li>
            <span className="text-slack-textMuted">Bootstrap token configured:</span>{' '}
            {hubSecurity.bootstrap_configured ? 'Yes' : 'No'}
          </li>
        </ul>
      ) : (
        <p className="text-sm text-slack-textMuted">Could not read hub security status.</p>
      )}
      {hubSecurity?.listen_all && !hubSecurity.hub_token_configured && (
        <p className="mt-3 text-sm text-amber-400">
          Hub is listening on all interfaces without NEURAL_JUNKIE_HUB_TOKEN — set a hub token before exposing
          the hub on a network. The server refuses to start in this configuration unless NEURAL_JUNKIE_DEBUG=1
          or NEURAL_JUNKIE_RELAXED_LOCAL=1.
        </p>
      )}
      {hubSecurity?.auth_required && hubSecurity.relaxed_local && (
        <p className="mt-3 text-sm text-amber-400">
          Both NEURAL_JUNKIE_AUTH_REQUIRED and NEURAL_JUNKIE_RELAXED_LOCAL are set — loopback clients get a
          synthetic member session; remote clients still need a real session or API key.
        </p>
      )}
    </div>

    <div>
      <h3 className="text-lg font-semibold text-slack-text mb-2">API keys</h3>
      <p className="text-sm text-slack-textMuted mb-3">
        Service account keys for CI/scripts (<code className="font-mono">nj_…</code>). Requires an{' '}
        <strong className="text-slack-text">admin</strong> hub session (login alone grants member only).
        Pass <code className="font-mono">--api-key</code> to standalone agents or{' '}
        <code className="font-mono">Authorization: Bearer nj_…</code> on hub HTTP calls.
      </p>

      {needsAdminUnlock && hubSecurity?.bootstrap_configured && (
        <div className="mb-4 rounded-lg border border-amber-700/40 bg-amber-950/20 p-3 space-y-2">
          <p className="text-sm text-amber-200">
            Your session is <strong>member</strong> (chat works; API key admin does not). Unlock admin with
            the hub bootstrap token from{' '}
            <code className="font-mono text-xs">~/.neural-junkie/bootstrap.token</code>.
          </p>
          {unlockError && <p className="text-sm text-red-400">{unlockError}</p>}
          <div className="flex flex-wrap gap-2">
            <input
              type="password"
              placeholder="Bootstrap token"
              value={bootstrapInput}
              onChange={(e) => setBootstrapInput(e.target.value)}
              className="flex-1 min-w-[12rem] px-2 py-1 rounded border border-slack-border bg-slack-bg text-sm text-slack-text font-mono"
            />
            <button
              type="button"
              disabled={unlocking || !bootstrapInput.trim()}
              onClick={() => void handleUnlockAdmin(bootstrapInput)}
              className="px-3 py-1 rounded bg-amber-700 text-white text-sm hover:opacity-90 disabled:opacity-50"
            >
              {unlocking ? 'Unlocking…' : 'Unlock admin'}
            </button>
            <button
              type="button"
              disabled={unlocking}
              onClick={() => void handleUseLocalBootstrap()}
              className="px-3 py-1 rounded border border-slack-border text-sm text-slack-textMuted hover:text-slack-text"
            >
              Use local bootstrap file
            </button>
          </div>
        </div>
      )}

      {adminUnlocked && (
        <p className="text-sm text-emerald-400 mb-2">Admin session active — you can manage API keys.</p>
      )}

      {keysError && !needsAdminUnlock && <p className="text-sm text-red-400 mb-2">{keysError}</p>}
      {keysError && needsAdminUnlock && !hubSecurity?.bootstrap_configured && (
        <p className="text-sm text-red-400 mb-2">
          {keysError} — bootstrap token not configured on this hub.
        </p>
      )}

      <div className="flex flex-wrap gap-2 mb-3">
        <input
          type="text"
          placeholder="Key name"
          value={newKeyName}
          onChange={(e) => setNewKeyName(e.target.value)}
          disabled={!adminUnlocked}
          className="px-2 py-1 rounded border border-slack-border bg-slack-bg text-sm text-slack-text disabled:opacity-50"
        />
        <select
          value={newKeyRole}
          onChange={(e) => setNewKeyRole(e.target.value)}
          disabled={!adminUnlocked}
          className="px-2 py-1 rounded border border-slack-border bg-slack-bg text-sm text-slack-text disabled:opacity-50"
        >
          <option value="admin">admin</option>
          <option value="member">member</option>
          <option value="viewer">viewer</option>
        </select>
        <button
          type="button"
          onClick={() => void handleCreateKey()}
          disabled={!adminUnlocked}
          className="px-3 py-1 rounded bg-slack-accent text-white text-sm hover:opacity-90 disabled:opacity-50"
        >
          Create key
        </button>
        <button
          type="button"
          onClick={() => void loadApiKeys()}
          disabled={loadingKeys}
          className="px-3 py-1 rounded border border-slack-border text-sm text-slack-textMuted"
        >
          Refresh
        </button>
      </div>
      {createdKey && (
        <div className="mb-3 p-3 rounded border border-emerald-700/50 bg-emerald-950/30 text-sm text-emerald-200">
          Copy this key now — it will not be shown again:
          <code className="block mt-1 font-mono break-all">{createdKey}</code>
        </div>
      )}
      {loadingKeys ? (
        <p className="text-sm text-slack-textMuted">Loading keys…</p>
      ) : apiKeys.length === 0 ? (
        <p className="text-sm text-slack-textMuted">
          {adminUnlocked ? 'No active API keys.' : 'Unlock admin to list keys.'}
        </p>
      ) : (
        <ul className="space-y-2 text-sm">
          {apiKeys.map((k) => (
            <li key={k.id} className="flex items-center justify-between gap-2 rounded border border-slack-border p-2">
              <div>
                <span className="text-slack-text font-medium">{k.name || 'unnamed'}</span>
                <span className="text-slack-textMuted ml-2">{k.prefix}</span>
                <span className="text-slack-textMuted ml-2">({k.role})</span>
              </div>
              <button
                type="button"
                onClick={() => void handleRevokeKey(k.id)}
                className="px-2 py-0.5 rounded bg-red-900/60 text-red-200 text-xs"
              >
                Revoke
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>

    <div className="rounded-lg border border-slack-border bg-slack-bgHover/30 p-4 text-xs text-slack-textMuted space-y-2">
      <p>
        <strong className="text-slack-text">Shared machine:</strong> set{' '}
        <code className="font-mono">NEURAL_JUNKIE_AUTH_REQUIRED=1</code> and{' '}
        <code className="font-mono">NEURAL_JUNKIE_HUB_TOKEN</code> to random secrets.
      </p>
      <p>
        <strong className="text-slack-text">Roles:</strong> viewer (read-only), member (send/approve), admin (API keys + ACL).
      </p>
    </div>
  </div>

  );
}
