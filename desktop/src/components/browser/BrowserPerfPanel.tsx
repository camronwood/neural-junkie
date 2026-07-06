import { useState } from 'react';
import { browserMetrics } from './browserSidecarApi';

interface BrowserPerfPanelProps {
  previewUrl: string;
  disabled?: boolean;
}

export function BrowserPerfPanel({ previewUrl, disabled }: BrowserPerfPanelProps) {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [metrics, setMetrics] = useState<Record<string, number> | null>(null);

  const measure = async () => {
    if (!previewUrl.trim()) return;
    setLoading(true);
    setError(null);
    try {
      const result = await browserMetrics(previewUrl);
      setMetrics(result.metrics || null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Metrics failed');
      setMetrics(null);
    } finally {
      setLoading(false);
    }
  };

  const rows = metrics
    ? [
        ['FCP (ms)', metrics.fcp_ms],
        ['Load (ms)', metrics.load_ms],
        ['DOM ready (ms)', metrics.dom_content_loaded_ms],
        ['DOM nodes', metrics.dom_nodes],
        ['Resources', metrics.resource_count],
        ['Transfer size', metrics.transfer_size],
      ]
    : [];

  return (
    <div className="flex min-h-0 flex-col gap-2 p-2 text-xs">
      <button
        type="button"
        disabled={disabled || loading || !previewUrl.trim()}
        onClick={() => void measure()}
        className="self-start rounded border border-slack-border px-2 py-1 hover:bg-slack-bgHover disabled:opacity-50"
      >
        {loading ? 'Measuring…' : 'Measure performance'}
      </button>
      {error && <p className="text-red-400">{error}</p>}
      {rows.length > 0 ? (
        <dl className="grid grid-cols-2 gap-1">
          {rows.map(([label, value]) => (
            <div key={label} className="contents">
              <dt className="text-slack-textMuted">{label}</dt>
              <dd className="font-mono text-slack-text">{Math.round(Number(value))}</dd>
            </div>
          ))}
        </dl>
      ) : (
        !loading && !error && <p className="text-slack-textMuted">No metrics yet.</p>
      )}
    </div>
  );
}
