import { useCallback, useEffect, useState } from 'react';
import type { ArenaSidecarStatus } from '../api/chatAPI';
import { ChatAPI } from '../api/chatAPI';

export interface ModelArenaToolsPanelProps {
  hubHttp: string;
  isActive: boolean;
  packEnabled?: boolean;
}

function statusLabel(st: ArenaSidecarStatus | null): string {
  if (!st) return 'Checking…';
  if (st.installing) return 'Installing chess dependencies…';
  if (st.chess_available) return 'Chess ready (python-chess installed)';
  if (st.venv_ready) return 'Venv present but python-chess missing';
  return 'Chess dependencies not installed';
}

function statusClass(st: ArenaSidecarStatus | null): string {
  if (!st) return 'text-slack-textMuted';
  if (st.chess_available) return 'text-green-400';
  if (st.installing) return 'text-amber-300';
  return 'text-amber-400';
}

export function ModelArenaToolsPanel({ hubHttp, isActive, packEnabled = true }: ModelArenaToolsPanelProps) {
  const [status, setStatus] = useState<ArenaSidecarStatus | null>(null);
  const [loading, setLoading] = useState(false);
  const [installing, setInstalling] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [ok, setOk] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const api = new ChatAPI(hubHttp);
      setStatus(await api.fetchArenaSidecarStatus());
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }, [hubHttp]);

  useEffect(() => {
    if (!isActive) return;
    void refresh();
  }, [isActive, refresh]);

  useEffect(() => {
    if (!isActive || !status?.installing) return;
    const id = window.setInterval(() => void refresh(), 3000);
    return () => window.clearInterval(id);
  }, [isActive, status?.installing, refresh]);

  const runInstall = async () => {
    const confirmed = window.confirm(
      'Install Model Arena chess dependencies?\n\n' +
        'Creates ~/.neural-junkie/arena/venv and installs python-chess from the pack requirements file. ' +
        'Connect Four and logic puzzles work without this step.',
    );
    if (!confirmed) return;
    setInstalling(true);
    setError(null);
    setOk(null);
    try {
      const api = new ChatAPI(hubHttp);
      const resp = await api.installArenaSidecarDeps('model-arena');
      setStatus(resp.sidecar);
      await api.restartArenaSidecar('model-arena');
      setOk('Chess dependencies installed. Try Chess mode in the Arena workbench.');
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      await refresh();
    } finally {
      setInstalling(false);
    }
  };

  const busy = loading || installing || status?.installing;

  return (
    <div className="rounded-lg border border-slack-border p-4">
      <h3 className="text-base font-semibold text-slack-text mb-2">Model Arena — sidecar deps</h3>
      <p className="text-sm text-slack-textMuted mb-3">
        Connect Four and logic puzzles run without extra setup. Chess needs{' '}
        <code className="font-mono text-xs">python-chess</code> in a local venv managed here.
      </p>

      <p className={`text-sm font-medium mb-3 ${statusClass(status)}`}>{statusLabel(status)}</p>
      {status && (
        <ul className="mb-3 space-y-1 text-xs font-mono text-slack-textMuted">
          <li>Python venv: {status.venv_ready ? '✓' : '—'} {status.paths.venv}</li>
          {status.python_version && <li>{status.python_version}</li>}
        </ul>
      )}

      <div className="flex flex-wrap gap-2">
        {!status?.chess_available && (
          <button
            type="button"
            disabled={busy || !packEnabled}
            onClick={() => void runInstall()}
            className="rounded bg-slack-accent px-4 py-2 text-sm text-white hover:bg-slack-accentHover disabled:opacity-50"
          >
            {installing ? 'Installing…' : 'Install chess dependencies'}
          </button>
        )}
        <button
          type="button"
          disabled={busy}
          onClick={() => void refresh()}
          className="rounded border border-slack-border px-4 py-2 text-sm text-slack-text hover:bg-slack-bgHover disabled:opacity-50"
        >
          Refresh status
        </button>
      </div>

      {!packEnabled && (
        <p className="mt-3 text-xs text-amber-300">Enable the Model Arena pack from the Store tab first.</p>
      )}
      {status?.last_error && (
        <p className="mt-2 whitespace-pre-wrap text-sm text-red-400">{status.last_error}</p>
      )}
      {error && <p className="mt-2 text-sm text-red-500">{error}</p>}
      {ok && <p className="mt-2 text-sm text-green-600">{ok}</p>}
    </div>
  );
}
