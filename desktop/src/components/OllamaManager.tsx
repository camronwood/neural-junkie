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
  const [models, setModels] = useState<string[]>([]);
  const [pullModel, setPullModel] = useState('');
  const [pulling, setPulling] = useState(false);
  const [pullProgress, setPullProgress] = useState('');

  useEffect(() => { refresh(); }, []);

  async function refresh() {
    try {
      const resp = await fetch(`${serverAddr}/api/ollama/install-status`);
      const data = await resp.json();
      setStatus(data);
      if (data.running) {
        const modelsResp = await fetch(`${serverAddr}/api/ollama/models`);
        if (modelsResp.ok) {
          const modelsData = await modelsResp.json();
          setModels(modelsData.models?.map((m: { name: string }) => m.name) || []);
        }
      }
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

  async function handlePull() {
    if (!pullModel) return;
    setPulling(true);
    setPullProgress('Starting...');

    try {
      const resp = await fetch(`${serverAddr}/api/ollama/pull`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ model: pullModel }),
      });
      const reader = resp.body?.getReader();
      const decoder = new TextDecoder();
      if (reader) {
        while (true) {
          const { done, value } = await reader.read();
          if (done) break;
          const text = decoder.decode(value);
          const lines = text.split('\n').filter(l => l.startsWith('data: '));
          for (const line of lines) {
            try {
              const data = JSON.parse(line.replace('data: ', ''));
              if (data.percent) {
                setPullProgress(`${data.percent.toFixed(1)}%`);
              } else if (data.status) {
                setPullProgress(data.status);
              }
            } catch { /* ignore */ }
          }
        }
      }
    } catch (e) {
      setPullProgress(`Error: ${e}`);
    }
    setPulling(false);
    setPullModel('');
    refresh();
  }

  return (
    <div className="space-y-4">
      <h3 className="text-sm font-semibold text-gray-300">Ollama</h3>

      {status === null ? (
        <div className="text-gray-500 text-sm">Loading...</div>
      ) : (
        <div className="space-y-3">
          <div className="flex items-center gap-3">
            <div className={`w-2 h-2 rounded-full ${status.running ? 'bg-green-400' : status.installed ? 'bg-yellow-400' : 'bg-red-400'}`} />
            <span className="text-sm text-gray-300">
              {status.running ? 'Running' : status.installed ? 'Installed (stopped)' : 'Not installed'}
            </span>
            {status.version && <span className="text-xs text-gray-600">{status.version}</span>}
          </div>

          {status.installed && (
            <div className="flex gap-2">
              {status.running ? (
                <button onClick={handleStop} className="px-3 py-1 text-xs bg-red-700/50 text-red-300 rounded hover:bg-red-700">
                  Stop
                </button>
              ) : (
                <button onClick={handleStart} className="px-3 py-1 text-xs bg-green-700/50 text-green-300 rounded hover:bg-green-700">
                  Start
                </button>
              )}
              <button onClick={refresh} className="px-3 py-1 text-xs bg-gray-700 text-gray-300 rounded hover:bg-gray-600">
                Refresh
              </button>
            </div>
          )}

          {status.running && models.length > 0 && (
            <div>
              <div className="text-xs text-gray-500 mb-1">Installed models:</div>
              <div className="flex flex-wrap gap-1">
                {models.map(m => (
                  <span key={m} className="px-2 py-0.5 bg-gray-800 text-gray-300 text-xs rounded">{m}</span>
                ))}
              </div>
            </div>
          )}

          {status.running && (
            <div className="flex gap-2 items-center">
              <input
                value={pullModel}
                onChange={(e) => setPullModel(e.target.value)}
                placeholder="Model name (e.g. qwen2.5-coder:14b)"
                className="flex-1 px-3 py-1.5 bg-gray-900 border border-gray-700 rounded text-sm text-white"
              />
              <button
                onClick={handlePull}
                disabled={pulling || !pullModel}
                className="px-3 py-1.5 text-xs bg-blue-600 text-white rounded hover:bg-blue-500 disabled:opacity-50"
              >
                {pulling ? pullProgress : 'Pull'}
              </button>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
