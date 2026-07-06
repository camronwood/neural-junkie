import { useState } from 'react';
import { browserVisualDiff, browserAcceptBaseline, type BrowserViewport } from './browserSidecarApi';

interface BrowserVisualDiffPanelProps {
  previewUrl: string;
  workspaceId: string;
  htmlPath: string;
  viewport: BrowserViewport | null;
  disabled?: boolean;
}

function baselineRelPath(htmlPath: string, viewport: BrowserViewport | null) {
  const safe = htmlPath.replace(/[^a-zA-Z0-9._-]+/g, '_');
  const suffix = viewport ? `${viewport.width}x${viewport.height}` : 'full';
  return `.nj/browser-baselines/${safe}/${suffix}.png`;
}

export function BrowserVisualDiffPanel({
  previewUrl,
  workspaceId,
  htmlPath,
  viewport,
  disabled,
}: BrowserVisualDiffPanelProps) {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [matchPct, setMatchPct] = useState<number | null>(null);
  const [diffSrc, setDiffSrc] = useState<string | null>(null);

  const baselinePath = baselineRelPath(htmlPath, viewport);

  const compare = async () => {
    if (!previewUrl.trim()) return;
    setLoading(true);
    setError(null);
    try {
      const result = await browserVisualDiff(previewUrl, baselinePath, workspaceId, viewport);
      setMatchPct(result.match_pct);
      if (result.diff_png_b64) {
        setDiffSrc(`data:image/png;base64,${result.diff_png_b64}`);
      } else {
        setDiffSrc(null);
      }
      if (!result.baseline_exists) {
        setError('No baseline yet — click Accept baseline after the page looks correct.');
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Visual diff failed');
      setMatchPct(null);
      setDiffSrc(null);
    } finally {
      setLoading(false);
    }
  };

  const acceptBaseline = async () => {
    if (!previewUrl.trim()) return;
    setLoading(true);
    setError(null);
    try {
      await browserAcceptBaseline(previewUrl, baselinePath, workspaceId, viewport);
      setMatchPct(100);
      setDiffSrc(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save baseline');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex min-h-0 flex-col gap-2 p-2 text-xs">
      <div className="flex flex-wrap gap-2">
        <button
          type="button"
          disabled={disabled || loading || !previewUrl.trim()}
          onClick={() => void compare()}
          className="rounded border border-slack-border px-2 py-1 hover:bg-slack-bgHover disabled:opacity-50"
        >
          {loading ? 'Working…' : 'Compare baseline'}
        </button>
        <button
          type="button"
          disabled={disabled || loading || !previewUrl.trim()}
          onClick={() => void acceptBaseline()}
          className="rounded border border-slack-border px-2 py-1 hover:bg-slack-bgHover disabled:opacity-50"
        >
          Accept baseline
        </button>
      </div>
      <p className="font-mono text-[10px] text-slack-textMuted truncate">{baselinePath}</p>
      {matchPct !== null && <p className="text-slack-text">Match: {matchPct.toFixed(1)}%</p>}
      {error && <p className="text-amber-400">{error}</p>}
      {diffSrc && (
        <img src={diffSrc} alt="Visual diff" className="max-h-40 border border-slack-border object-contain" />
      )}
    </div>
  );
}
