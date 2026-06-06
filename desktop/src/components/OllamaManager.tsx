import { useState, useEffect, useCallback } from 'react';
import {
  fetchOllamaRuntimeStatus,
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

  const refresh = useCallback(async () => {
    try {
      const data = await fetchOllamaRuntimeStatus(serverAddr);
      setStatus(data);
    } catch {
      setStatus({ installed: false, running: false });
    }
  }, [serverAddr]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  async function handleStart() {
    await startOllamaRuntime(serverAddr);
    setTimeout(() => void refresh(), 1200);
  }

  async function handleStop() {
    await stopOllamaRuntime(serverAddr);
    setTimeout(() => void refresh(), 400);
  }

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

          {status.installed && (
            <div className="flex gap-2">
              {status.running ? (
                <button
                  type="button"
                  onClick={() => void handleStop()}
                  className="px-3 py-1 text-xs bg-red-700/50 text-red-300 rounded hover:bg-red-700"
                >
                  Stop
                </button>
              ) : (
                <button
                  type="button"
                  onClick={() => void handleStart()}
                  className="px-3 py-1 text-xs bg-green-700/50 text-green-300 rounded hover:bg-green-700"
                >
                  Start
                </button>
              )}
              <button
                type="button"
                onClick={() => void refresh()}
                className="px-3 py-1 text-xs bg-gray-700 text-gray-300 rounded hover:bg-gray-600"
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
