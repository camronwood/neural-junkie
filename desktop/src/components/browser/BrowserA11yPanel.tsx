import { useState } from 'react';
import { browserA11yAudit } from './browserSidecarApi';

interface BrowserA11yPanelProps {
  previewUrl: string;
  disabled?: boolean;
}

export function BrowserA11yPanel({ previewUrl, disabled }: BrowserA11yPanelProps) {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [violations, setViolations] = useState<
    Array<{ id: string; impact: string; help: string; node_count: number }>
  >([]);

  const runAudit = async () => {
    if (!previewUrl.trim()) return;
    setLoading(true);
    setError(null);
    try {
      const result = await browserA11yAudit(previewUrl);
      setViolations(result.violations || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'A11y audit failed');
      setViolations([]);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex min-h-0 flex-col gap-2 p-2 text-xs">
      <div className="flex items-center gap-2">
        <button
          type="button"
          disabled={disabled || loading || !previewUrl.trim()}
          onClick={() => void runAudit()}
          className="rounded border border-slack-border px-2 py-1 hover:bg-slack-bgHover disabled:opacity-50"
        >
          {loading ? 'Running…' : 'Run a11y audit'}
        </button>
        {violations.length > 0 && (
          <span className="text-slack-textMuted">{violations.length} violation(s)</span>
        )}
      </div>
      {error && <p className="text-red-400">{error}</p>}
      <ul className="min-h-0 flex-1 overflow-auto space-y-1">
        {violations.map((v) => (
          <li key={v.id} className="rounded border border-slack-border p-2">
            <span className="font-medium text-slack-text">[{v.impact}] {v.id}</span>
            <p className="text-slack-textMuted">{v.help}</p>
            <p className="text-[10px] text-slack-textMuted">{v.node_count} node(s)</p>
          </li>
        ))}
        {!loading && !error && violations.length === 0 && (
          <li className="text-slack-textMuted">No audit run yet.</li>
        )}
      </ul>
    </div>
  );
}
