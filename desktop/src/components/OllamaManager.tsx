import { useState, useEffect } from 'react';

interface OllamaManagerProps {
  serverAddr: string;
}

interface OllamaStatus {
  installed: boolean;
  running: boolean;
  version?: string;
  path?: string;
}

export function OllamaManager({ serverAddr }: OllamaManagerProps) {
  const [status, setStatus] = useState<OllamaStatus | null>(null);

  useEffect(() => {
    refresh();
  }, []);

  async function refresh() {
    try {
      const resp = await fetch(`${serverAddr}/api/ollama/install-status`);
      const data = await resp.json();
      setStatus(data);
    } catch {
      setStatus({ installed: false, running: false });
    }
  }

  async function handleStart() {
    await fetch(`${serverAddr}/api/ollama/start`, { method: 'POST' });
    setTimeout(refresh, 2000);
  }

  async function handleStop() {
    await fetch(`${serverAddr}/api/ollama/stop`, { method: 'POST' });
    setTimeout(refresh, 1000);
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
              {status.running ? 'Running' : status.installed ? 'Installed (stopped)' : 'Not installed'}
            </span>
            {status.version && <span className="text-xs text-gray-600">{status.version}</span>}
          </div>

          {status.installed && (
            <div className="flex gap-2">
              {status.running ? (
                <button
                  type="button"
                  onClick={handleStop}
                  className="px-3 py-1 text-xs bg-red-700/50 text-red-300 rounded hover:bg-red-700"
                >
                  Stop
                </button>
              ) : (
                <button
                  type="button"
                  onClick={handleStart}
                  className="px-3 py-1 text-xs bg-green-700/50 text-green-300 rounded hover:bg-green-700"
                >
                  Start
                </button>
              )}
              <button
                type="button"
                onClick={refresh}
                className="px-3 py-1 text-xs bg-gray-700 text-gray-300 rounded hover:bg-gray-600"
              >
                Refresh
              </button>
            </div>
          )}

          <p className="text-xs text-gray-500">
            Open this anytime from the <strong className="text-gray-400">chat toolbar</strong> (amber model icon),{' '}
            <strong className="text-gray-400">⇧⌘M</strong> / <strong className="text-gray-400">Ctrl+Shift+M</strong>, or the command palette:{' '}
            <span className="font-mono text-gray-400">/nj-open-model-library</span>.
          </p>
        </div>
      )}
    </div>
  );
}
