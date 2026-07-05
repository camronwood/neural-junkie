import { useEffect, useState } from 'react';
import {
  putSystemSecurity,
  type SaveSettingsResult,
  type SettingsTabProps,
} from './settingsShared';

type ServerForm = {
  host: string;
  port: number;
  listen_all: boolean;
  cors_any: boolean;
  cors_origins: string;
  ws_origins: string;
};

type SessionForm = {
  restore_on_startup: boolean;
  skip_restore_once: boolean;
  force_restore_large: boolean;
};

type DebugForm = {
  enabled: boolean;
  pprof_addr: string;
};

type MCPResourcesForm = {
  enabled: boolean;
  port: number;
  exports_dir: string;
};

export function ServerNetworkSettingsTab({ hubHttp, isActive }: SettingsTabProps) {
  const [server, setServer] = useState<ServerForm>({
    host: 'localhost',
    port: 18765,
    listen_all: false,
    cors_any: false,
    cors_origins: '',
    ws_origins: '',
  });
  const [session, setSession] = useState<SessionForm>({
    restore_on_startup: false,
    skip_restore_once: false,
    force_restore_large: false,
  });
  const [debug, setDebug] = useState<DebugForm>({ enabled: false, pprof_addr: '127.0.0.1:6060' });
  const [mcpResources, setMcpResources] = useState<MCPResourcesForm>({
    enabled: false,
    port: 8086,
    exports_dir: '',
  });
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);
  const [saveResult, setSaveResult] = useState<SaveSettingsResult | null>(null);

  useEffect(() => {
    if (!isActive) return;
    let cancelled = false;
    (async () => {
      try {
        const r = await fetch(`${hubHttp}/api/system/security`);
        if (!r.ok) throw new Error(await r.text());
        const data = await r.json();
        if (cancelled) return;
        const srv = data.server ?? {};
        setServer({
          host: String(srv.host ?? 'localhost'),
          port: Number(srv.port ?? 18765),
          listen_all: !!srv.listen_all,
          cors_any: !!srv.cors_any,
          cors_origins: Array.isArray(srv.cors_origins) ? srv.cors_origins.join(', ') : '',
          ws_origins: Array.isArray(srv.ws_origins) ? srv.ws_origins.join(', ') : '',
        });
        const sess = data.session ?? {};
        setSession({
          restore_on_startup: !!sess.restore_on_startup,
          skip_restore_once: !!sess.skip_restore_once,
          force_restore_large: !!sess.force_restore_large,
        });
        const dbg = data.debug ?? {};
        setDebug({
          enabled: !!dbg.enabled,
          pprof_addr: String(dbg.pprof_addr ?? '127.0.0.1:6060'),
        });
        const mcp = data.mcp_resources ?? {};
        setMcpResources({
          enabled: !!mcp.enabled,
          port: Number(mcp.port ?? 8086),
          exports_dir: String(mcp.exports_dir ?? ''),
        });
      } catch (e) {
        if (!cancelled) setErr(e instanceof Error ? e.message : String(e));
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [isActive, hubHttp]);

  const save = async () => {
    setBusy(true);
    setErr(null);
    setSaveResult(null);
    try {
      const split = (s: string) =>
        s
          .split(',')
          .map((x) => x.trim())
          .filter(Boolean);
      const result = await putSystemSecurity(hubHttp, {
        server: {
          host: server.host.trim() || 'localhost',
          port: server.port || 18765,
          listen_all: server.listen_all,
          cors_any: server.cors_any,
          cors_origins: split(server.cors_origins),
          ws_origins: split(server.ws_origins),
        },
        session,
        debug,
        mcp_resources: {
          enabled: mcpResources.enabled,
          port: mcpResources.port || 8086,
          exports_dir: mcpResources.exports_dir.trim(),
        },
      });
      setSaveResult(result);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  };

  if (!isActive) return null;

  return (
    <div className="space-y-6 max-w-2xl">
      <div>
        <h3 className="text-lg font-semibold text-slack-text mb-1">Server & network</h3>
        <p className="text-sm text-slack-textMuted mb-4">
          Hub bind address, CORS, session restore, debug routes, and MCP resource server. Environment
          variables still override these when set.
        </p>
      </div>

      {err && <p className="text-sm text-red-400">{err}</p>}
      {saveResult?.requires_restart && (
        <p className="text-sm text-amber-200 rounded border border-amber-700/40 bg-amber-950/20 p-3">
          Saved — restart the hub for: {(saveResult.restart_reasons ?? []).join(', ') || 'network/debug'}.
        </p>
      )}

      <section className="space-y-3">
        <h4 className="text-sm font-medium text-slack-text">HTTP server</h4>
        <label className="flex gap-2 items-center text-sm">
          <span className="w-24 text-slack-textMuted">Host</span>
          <input
            className="flex-1 px-2 py-1 rounded border border-slack-border bg-slack-bg text-slack-text"
            value={server.host}
            onChange={(e) => setServer((s) => ({ ...s, host: e.target.value }))}
          />
        </label>
        <label className="flex gap-2 items-center text-sm">
          <span className="w-24 text-slack-textMuted">Port</span>
          <input
            type="number"
            className="w-32 px-2 py-1 rounded border border-slack-border bg-slack-bg text-slack-text"
            value={server.port}
            onChange={(e) => setServer((s) => ({ ...s, port: Number(e.target.value) }))}
          />
        </label>
        <label className="flex gap-2 items-center text-sm">
          <input
            type="checkbox"
            checked={server.listen_all}
            onChange={(e) => setServer((s) => ({ ...s, listen_all: e.target.checked }))}
          />
          Listen on all interfaces (0.0.0.0) — requires hub token
        </label>
        <label className="flex gap-2 items-center text-sm">
          <input
            type="checkbox"
            checked={server.cors_any}
            onChange={(e) => setServer((s) => ({ ...s, cors_any: e.target.checked }))}
          />
          Allow all CORS origins (legacy)
        </label>
        <label className="block text-sm">
          <span className="text-slack-textMuted">Extra CORS origins (comma-separated)</span>
          <input
            className="mt-1 w-full px-2 py-1 rounded border border-slack-border bg-slack-bg text-slack-text"
            value={server.cors_origins}
            onChange={(e) => setServer((s) => ({ ...s, cors_origins: e.target.value }))}
          />
        </label>
        <label className="block text-sm">
          <span className="text-slack-textMuted">Extra WebSocket origins (comma-separated)</span>
          <input
            className="mt-1 w-full px-2 py-1 rounded border border-slack-border bg-slack-bg text-slack-text"
            value={server.ws_origins}
            onChange={(e) => setServer((s) => ({ ...s, ws_origins: e.target.value }))}
          />
        </label>
      </section>

      <section className="space-y-2">
        <h4 className="text-sm font-medium text-slack-text">Session restore</h4>
        <label className="flex gap-2 items-center text-sm">
          <input
            type="checkbox"
            checked={session.restore_on_startup}
            onChange={(e) => setSession((s) => ({ ...s, restore_on_startup: e.target.checked }))}
          />
          Restore last session on startup
        </label>
        <label className="flex gap-2 items-center text-sm">
          <input
            type="checkbox"
            checked={session.skip_restore_once}
            onChange={(e) => setSession((s) => ({ ...s, skip_restore_once: e.target.checked }))}
          />
          Skip restore once (next boot only)
        </label>
        <label className="flex gap-2 items-center text-sm">
          <input
            type="checkbox"
            checked={session.force_restore_large}
            onChange={(e) => setSession((s) => ({ ...s, force_restore_large: e.target.checked }))}
          />
          Force restore large session files (&gt;64MB, may OOM)
        </label>
      </section>

      <section className="space-y-2">
        <h4 className="text-sm font-medium text-slack-text">Debug</h4>
        <label className="flex gap-2 items-center text-sm">
          <input
            type="checkbox"
            checked={debug.enabled}
            onChange={(e) => setDebug((d) => ({ ...d, enabled: e.target.checked }))}
          />
          Enable debug routes and pprof (loopback)
        </label>
        <label className="block text-sm">
          <span className="text-slack-textMuted">pprof address</span>
          <input
            className="mt-1 w-full px-2 py-1 rounded border border-slack-border bg-slack-bg text-slack-text font-mono"
            value={debug.pprof_addr}
            onChange={(e) => setDebug((d) => ({ ...d, pprof_addr: e.target.value }))}
          />
        </label>
      </section>

      <section className="space-y-2">
        <h4 className="text-sm font-medium text-slack-text">MCP resource server</h4>
        <label className="flex gap-2 items-center text-sm">
          <input
            type="checkbox"
            checked={mcpResources.enabled}
            onChange={(e) => setMcpResources((m) => ({ ...m, enabled: e.target.checked }))}
          />
          Enable MCP resource server
        </label>
        <label className="flex gap-2 items-center text-sm">
          <span className="w-24 text-slack-textMuted">Port</span>
          <input
            type="number"
            className="w-32 px-2 py-1 rounded border border-slack-border bg-slack-bg text-slack-text"
            value={mcpResources.port}
            onChange={(e) => setMcpResources((m) => ({ ...m, port: Number(e.target.value) }))}
          />
        </label>
        <label className="block text-sm">
          <span className="text-slack-textMuted">Exports directory</span>
          <input
            className="mt-1 w-full px-2 py-1 rounded border border-slack-border bg-slack-bg text-slack-text"
            placeholder="~/.neural-junkie/exports"
            value={mcpResources.exports_dir}
            onChange={(e) => setMcpResources((m) => ({ ...m, exports_dir: e.target.value }))}
          />
        </label>
      </section>

      <button
        type="button"
        disabled={busy}
        onClick={() => void save()}
        className="px-4 py-2 rounded bg-slack-accent text-white text-sm hover:opacity-90 disabled:opacity-50"
      >
        {busy ? 'Saving…' : 'Save server & network'}
      </button>
    </div>
  );
}
