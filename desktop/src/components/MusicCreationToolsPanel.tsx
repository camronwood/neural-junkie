import { useCallback, useEffect, useState } from 'react';
import type { ACEStepStatus } from '../api/chatAPI';
import { ChatAPI } from '../api/chatAPI';

export interface MusicCreationToolsPanelProps {
  hubHttp: string;
  isActive: boolean;
}

function statusLabel(st: ACEStepStatus | null): string {
  if (!st) return 'Checking…';
  if st.demo_mode) return 'Demo mode (NJ_MUSIC_DEMO=1)';
  if st.installing) return 'Installing ACE-Step…';
  if (st.ready) return 'Ready';
  return 'Not installed';
}

function statusClass(st: ACEStepStatus | null): string {
  if (!st) return 'text-slack-textMuted';
  if (st.demo_mode || st.ready) return 'text-green-400';
  if (st.installing) return 'text-amber-300';
  return 'text-amber-400';
}

export function MusicCreationToolsPanel({ hubHttp, isActive }: MusicCreationToolsPanelProps) {
  const [status, setStatus] = useState<ACEStepStatus | null>(null);
  const [loading, setLoading] = useState(false);
  const [installing, setInstalling] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [ok, setOk] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const api = new ChatAPI(hubHttp);
      setStatus(await api.fetchACEStepStatus());
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
    const id = window.setInterval(() => void refresh(), 5000);
    return () => window.clearInterval(id);
  }, [isActive, status?.installing, refresh]);

  const runInstall = async () => {
    const confirmed = window.confirm(
      'Download and install ACE-Step 1.5?\n\n' +
        'This clones the ACE-Step repo, creates a Python 3.12 venv, and downloads model weights (~several GB). ' +
        'It can take 10–30 minutes depending on your connection.\n\n' +
        'Requires Python 3.11 or 3.12 (pyenv or Homebrew).',
    );
    if (!confirmed) return;
    setInstalling(true);
    setError(null);
    setOk(null);
    try {
      const api = new ChatAPI(hubHttp);
      const resp = await api.installACEStep();
      setStatus(resp.acestep);
      setOk('ACE-Step installed. Try /generate-music in chat.');
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
      <h3 className="text-base font-semibold text-slack-text mb-2">Music creation — ACE-Step</h3>
      <p className="text-sm text-slack-textMuted mb-3">
        Full song generation uses a local ACE-Step 1.5 sidecar. Install once after enabling the pack.
        Ollama models for lyrics (<code className="font-mono text-xs">qwen2.5:7b</code>,{' '}
        <code className="font-mono text-xs">qwen3.5:9b</code>) pull automatically when Ollama is running.
      </p>
      <p className={`text-sm font-medium mb-3 ${statusClass(status)}`}>{statusLabel(status)}</p>
      {status && !status.demo_mode && (
        <ul className="mb-3 space-y-1 text-xs font-mono text-slack-textMuted">
          <li>Python venv: {status.venv_ready ? '✓' : '—'} {status.paths.venv}</li>
          <li>ACE-Step project: {status.project_ready ? '✓' : '—'} {status.paths.project}</li>
          <li>Model weights: {status.checkpoint_ready ? '✓' : '—'} {status.paths.checkpoint}</li>
          {status.python_version && <li>{status.python_version}</li>}
        </ul>
      )}
      <div className="flex flex-wrap gap-2">
        {!status?.ready && !status?.demo_mode && (
          <button
            type="button"
            disabled={busy}
            onClick={() => void runInstall()}
            className="rounded bg-slack-accent px-4 py-2 text-sm text-white hover:bg-slack-accentHover disabled:opacity-50"
          >
            {busy ? 'Installing…' : 'Install ACE-Step'}
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
      {status?.last_error && (
        <p className="mt-2 whitespace-pre-wrap text-sm text-red-400">{status.last_error}</p>
      )}
      {error && <p className="mt-2 text-sm text-red-500">{error}</p>}
      {ok && <p className="mt-2 text-sm text-green-600">{ok}</p>}
      {!status?.ready && !status?.demo_mode && (
        <p className="mt-3 text-xs text-slack-textMuted">
          UI smoke test without weights: set <code className="font-mono">NJ_MUSIC_DEMO=1</code> and restart the hub.
        </p>
      )}
    </div>
  );
}
