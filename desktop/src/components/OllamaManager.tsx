import { useState, useEffect, useCallback } from 'react';
import {
  fetchOllamaRuntimeStatus,
  installOllamaRuntime,
  startOllamaRuntime,
  stopOllamaRuntime,
  type OllamaRuntimeStatus,
} from '../utils/ollamaRuntime';

interface OllamaManagerProps {
  serverAddr: string;
  /** When false, hides the “open from toolbar” hint (e.g. inside the model library modal). */
  showLibraryHint?: boolean;
}

export function OllamaManager({ serverAddr, showLibraryHint = true }: OllamaManagerProps) {
  const [status, setStatus] = useState<OllamaRuntimeStatus | null>(null);
  const [busy, setBusy] = useState(false);
  const [installProgress, setInstallProgress] = useState('');
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    try {
      const data = await fetchOllamaRuntimeStatus(serverAddr);
      setStatus(data);
      setError(null);
    } catch {
      setStatus({ installed: false, running: false });
    }
  }, [serverAddr]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  async function handleStart() {
    setBusy(true);
    setError(null);
    try {
      await startOllamaRuntime(serverAddr);
      setTimeout(() => void refresh(), 1200);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to start Ollama');
    } finally {
      setBusy(false);
    }
  }

  async function handleStop() {
    setBusy(true);
    setError(null);
    try {
      await stopOllamaRuntime(serverAddr);
      setTimeout(() => void refresh(), 400);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to stop Ollama');
    } finally {
      setBusy(false);
    }
  }

  async function handleInstall() {
    setBusy(true);
    setError(null);
    setInstallProgress('Preparing Ollama install…');
    try {
      await installOllamaRuntime(serverAddr, (msg) => setInstallProgress(msg));
      setInstallProgress('');
      await refresh();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Ollama install failed');
    } finally {
      setBusy(false);
    }
  }

  const notInstalled = status !== null && !status.installed;
  const canAutoInstall = status?.autoInstallSupported !== false;

  return (
    <div className="space-y-4">
      <h3 className="text-sm font-semibold text-gray-300">Ollama</h3>

      {status === null ? (
        <div className="text-gray-500 text-sm">Loading...</div>
      ) : (
        <div className="space-y-3">
          <div className="flex items-center gap-3">
            <div
              className={`w-2 h-2 rounded-full ${status.running ? 'bg-green-400' : status.installed ? 'bg-yellow-400' : 'bg-red-400'}`}
            />
            <span className="text-sm text-gray-300">
              {status.running
                ? status.bundled
                  ? 'Running (bundled)'
                  : 'Running'
                : status.installed
                  ? status.bundled
                    ? 'Bundled (stopped)'
                    : 'Installed (stopped)'
                  : 'Not installed'}
            </span>
            {status.version && <span className="text-xs text-gray-600">{status.version}</span>}
          </div>

          {error && <p className="text-xs text-red-400">{error}</p>}
          {installProgress && <p className="text-xs text-gray-400">{installProgress}</p>}

          {notInstalled ? (
            <div className="flex flex-wrap gap-2 items-center">
              {canAutoInstall ? (
                <button
                  type="button"
                  onClick={() => void handleInstall()}
                  disabled={busy}
                  className="px-3 py-1 text-xs bg-blue-600 text-white rounded hover:bg-blue-500 disabled:opacity-50"
                  data-testid="ollama-manager-install"
                >
                  {busy ? 'Installing…' : 'Install Ollama'}
                </button>
              ) : (
                <a
                  href="https://ollama.com"
                  target="_blank"
                  rel="noreferrer"
                  className="px-3 py-1 text-xs bg-blue-600 text-white rounded hover:bg-blue-500"
                >
                  Get Ollama
                </a>
              )}
              <button
                type="button"
                onClick={() => void refresh()}
                disabled={busy}
                className="px-3 py-1 text-xs bg-gray-700 text-gray-300 rounded hover:bg-gray-600 disabled:opacity-50"
              >
                Refresh
              </button>
              {canAutoInstall && (
                <p className="text-xs text-gray-500 w-full">
                  One-click install (internet required; Linux may ask for your password).
                </p>
              )}
            </div>
          ) : (
            <div className="flex gap-2">
              {status.running ? (
                <button
                  type="button"
                  onClick={() => void handleStop()}
                  disabled={busy}
                  className="px-3 py-1 text-xs bg-red-700/50 text-red-300 rounded hover:bg-red-700 disabled:opacity-50"
                >
                  Stop
                </button>
              ) : (
                <button
                  type="button"
                  onClick={() => void handleStart()}
                  disabled={busy}
                  className="px-3 py-1 text-xs bg-green-700/50 text-green-300 rounded hover:bg-green-700 disabled:opacity-50"
                >
                  Start
                </button>
              )}
              <button
                type="button"
                onClick={() => void refresh()}
                disabled={busy}
                className="px-3 py-1 text-xs bg-gray-700 text-gray-300 rounded hover:bg-gray-600 disabled:opacity-50"
              >
                Refresh
              </button>
            </div>
          )}

          {showLibraryHint && (
            <p className="text-xs text-gray-500">
              Use the <strong className="text-gray-400">OLL</strong> chip in the chat toolbar for runtime status, or open
              the <strong className="text-gray-400">model library</strong> (amber icon),{' '}
              <strong className="text-gray-400">⇧⌘M</strong> / <strong className="text-gray-400">Ctrl+Shift+M</strong>, or
              the command palette: <span className="font-mono text-gray-400">/nj-open-model-library</span>.
            </p>
          )}
        </div>
      )}
    </div>
  );
}
