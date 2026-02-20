import { useState, useEffect } from 'react';

interface ProviderConfig {
  id: string;
  type: string;
  name: string;
  endpoint?: string;
  api_key?: string;
  model?: string;
  headers?: Record<string, string>;
  work_dir?: string;
}

interface ProviderManagerProps {
  serverAddr: string;
}

const PROVIDER_TYPES = [
  { value: 'ollama', label: 'Ollama (Local)' },
  { value: 'anthropic', label: 'Anthropic (Claude)' },
  { value: 'openai-compatible', label: 'OpenAI-Compatible (Bedrock, Azure, Groq, etc.)' },
  { value: 'cursor-cli', label: 'Cursor CLI' },
  { value: 'gemini-cli', label: 'Gemini CLI' },
];

export function ProviderManager({ serverAddr }: ProviderManagerProps) {
  const [providers, setProviders] = useState<ProviderConfig[]>([]);
  const [editing, setEditing] = useState<ProviderConfig | null>(null);
  const [isNew, setIsNew] = useState(false);
  const [testResult, setTestResult] = useState<Record<string, { success: boolean; error?: string }>>({});

  useEffect(() => { loadProviders(); }, []);

  async function loadProviders() {
    try {
      const resp = await fetch(`${serverAddr}/api/providers`);
      const data = await resp.json();
      setProviders(data);
    } catch (e) {
      console.error('Failed to load providers:', e);
    }
  }

  async function saveProvider(p: ProviderConfig) {
    const method = isNew ? 'POST' : 'PUT';
    const url = isNew ? `${serverAddr}/api/providers` : `${serverAddr}/api/providers/${p.id}`;
    await fetch(url, {
      method,
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(p),
    });
    setEditing(null);
    setIsNew(false);
    loadProviders();
  }

  async function deleteProvider(id: string) {
    if (!confirm(`Remove provider "${id}"?`)) return;
    const resp = await fetch(`${serverAddr}/api/providers/${id}`, { method: 'DELETE' });
    if (!resp.ok) {
      const err = await resp.text();
      alert(err);
      return;
    }
    loadProviders();
  }

  async function testProvider(id: string) {
    setTestResult(prev => ({ ...prev, [id]: { success: false } }));
    try {
      const resp = await fetch(`${serverAddr}/api/providers/${id}/test`, { method: 'POST' });
      const data = await resp.json();
      setTestResult(prev => ({ ...prev, [id]: data }));
    } catch (e) {
      setTestResult(prev => ({ ...prev, [id]: { success: false, error: String(e) } }));
    }
  }

  function startAdd() {
    setEditing({ id: '', type: 'openai-compatible', name: '' });
    setIsNew(true);
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold text-gray-300">AI Providers</h3>
        <button onClick={startAdd} className="px-3 py-1 text-xs bg-blue-600 text-white rounded hover:bg-blue-500">
          Add Provider
        </button>
      </div>

      {/* Provider list */}
      <div className="space-y-2">
        {providers.map(p => (
          <div key={p.id} className="flex items-center justify-between p-3 bg-gray-800 rounded-lg">
            <div>
              <div className="text-sm text-white font-medium">{p.name || p.id}</div>
              <div className="text-xs text-gray-500">{p.type} {p.model ? `· ${p.model}` : ''}</div>
            </div>
            <div className="flex items-center gap-2">
              {testResult[p.id] && (
                <span className={`text-xs ${testResult[p.id].success ? 'text-green-400' : 'text-red-400'}`}>
                  {testResult[p.id].success ? 'Connected' : testResult[p.id].error?.slice(0, 30) || 'Failed'}
                </span>
              )}
              <button onClick={() => testProvider(p.id)} className="px-2 py-1 text-xs bg-gray-700 text-gray-300 rounded hover:bg-gray-600">
                Test
              </button>
              <button onClick={() => { setEditing(p); setIsNew(false); }} className="px-2 py-1 text-xs bg-gray-700 text-gray-300 rounded hover:bg-gray-600">
                Edit
              </button>
              <button onClick={() => deleteProvider(p.id)} className="px-2 py-1 text-xs bg-red-700/50 text-red-300 rounded hover:bg-red-700">
                Remove
              </button>
            </div>
          </div>
        ))}
      </div>

      {/* Edit/Add form */}
      {editing && (
        <div className="p-4 bg-gray-800/50 border border-gray-700 rounded-lg space-y-3">
          <div className="text-sm font-medium text-white">{isNew ? 'Add Provider' : 'Edit Provider'}</div>
          <div className="grid grid-cols-2 gap-3">
            <input
              value={editing.id}
              onChange={(e) => setEditing({ ...editing, id: e.target.value })}
              placeholder="ID (e.g. my-claude)"
              disabled={!isNew}
              className="px-3 py-2 bg-gray-900 border border-gray-700 rounded text-sm text-white disabled:opacity-50"
            />
            <select
              value={editing.type}
              onChange={(e) => setEditing({ ...editing, type: e.target.value })}
              className="px-3 py-2 bg-gray-900 border border-gray-700 rounded text-sm text-white"
            >
              {PROVIDER_TYPES.map(t => (
                <option key={t.value} value={t.value}>{t.label}</option>
              ))}
            </select>
            <input
              value={editing.name}
              onChange={(e) => setEditing({ ...editing, name: e.target.value })}
              placeholder="Display name"
              className="px-3 py-2 bg-gray-900 border border-gray-700 rounded text-sm text-white"
            />
            <input
              value={editing.model || ''}
              onChange={(e) => setEditing({ ...editing, model: e.target.value })}
              placeholder="Model"
              className="px-3 py-2 bg-gray-900 border border-gray-700 rounded text-sm text-white"
            />
            {(editing.type === 'ollama' || editing.type === 'openai-compatible') && (
              <input
                value={editing.endpoint || ''}
                onChange={(e) => setEditing({ ...editing, endpoint: e.target.value })}
                placeholder="Endpoint URL"
                className="col-span-2 px-3 py-2 bg-gray-900 border border-gray-700 rounded text-sm text-white"
              />
            )}
            {(editing.type === 'anthropic' || editing.type === 'openai-compatible') && (
              <input
                value={editing.api_key || ''}
                onChange={(e) => setEditing({ ...editing, api_key: e.target.value })}
                placeholder="API Key"
                type="password"
                className="col-span-2 px-3 py-2 bg-gray-900 border border-gray-700 rounded text-sm text-white"
              />
            )}
          </div>
          <div className="flex gap-2">
            <button onClick={() => saveProvider(editing)} className="px-4 py-2 bg-blue-600 text-white text-sm rounded hover:bg-blue-500">
              {isNew ? 'Add' : 'Save'}
            </button>
            <button onClick={() => { setEditing(null); setIsNew(false); }} className="px-4 py-2 bg-gray-700 text-gray-300 text-sm rounded hover:bg-gray-600">
              Cancel
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
