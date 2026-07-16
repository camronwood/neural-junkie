import { useCallback, useEffect, useState } from 'react';
import { useChatStore } from '../../stores/chatStore';
import type { InferenceUsageSummary, UsageBucketTotals } from '../../types/inferenceUsage';
import {
  formatCostUsd,
  formatSessionUsageLine,
  formatTokenCount,
} from '../../utils/inferenceUsageFormat';

interface InferenceUsageSettingsTabProps {
  hubHttp: string;
  isActive: boolean;
}

function TotalsCard({ title, totals }: { title: string; totals: UsageBucketTotals }) {
  return (
    <div className="rounded-lg border border-slack-border bg-slack-bgHover p-4">
      <div className="text-sm font-medium text-slack-textMuted mb-2">{title}</div>
      <div className="text-2xl font-semibold text-slack-text">
        {formatTokenCount(totals.prompt_tokens + totals.completion_tokens)} tokens
      </div>
      <div className="mt-1 text-sm text-slack-textMuted">
        {formatTokenCount(totals.prompt_tokens)} in · {formatTokenCount(totals.completion_tokens)} out
        {totals.calls > 0 ? ` · ${totals.calls} calls` : ''}
      </div>
      <div className="mt-1 text-sm text-slack-text">
        Est. cost {formatCostUsd(totals.estimated_cost_usd)}
      </div>
    </div>
  );
}

function BucketTable({
  title,
  rows,
}: {
  title: string;
  rows: Array<[string, UsageBucketTotals]>;
}) {
  if (rows.length === 0) {
    return (
      <div className="rounded-lg border border-slack-border bg-slack-bgHover p-4">
        <div className="font-medium text-slack-text mb-2">{title}</div>
        <div className="text-sm text-slack-textMuted">No data yet.</div>
      </div>
    );
  }
  return (
    <div className="rounded-lg border border-slack-border bg-slack-bgHover p-4 overflow-x-auto">
      <div className="font-medium text-slack-text mb-3">{title}</div>
      <table className="w-full text-sm">
        <thead>
          <tr className="text-left text-slack-textMuted border-b border-slack-border">
            <th className="py-2 pr-3">Name</th>
            <th className="py-2 pr-3">In</th>
            <th className="py-2 pr-3">Out</th>
            <th className="py-2 pr-3">Calls</th>
            <th className="py-2">Est. cost</th>
          </tr>
        </thead>
        <tbody>
          {rows.map(([name, bucket]) => (
            <tr key={name} className="border-b border-slack-border/50 text-slack-text">
              <td className="py-2 pr-3 font-mono text-xs">{name}</td>
              <td className="py-2 pr-3">{formatTokenCount(bucket.prompt_tokens)}</td>
              <td className="py-2 pr-3">{formatTokenCount(bucket.completion_tokens)}</td>
              <td className="py-2 pr-3">{bucket.calls}</td>
              <td className="py-2">{formatCostUsd(bucket.estimated_cost_usd)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

export function InferenceUsageSettingsTab({ hubHttp, isActive }: InferenceUsageSettingsTabProps) {
  const sessionUsage = useChatStore((s) => s.sessionUsage);
  const clearSessionUsage = useChatStore((s) => s.clearSessionUsage);
  const [summary, setSummary] = useState<InferenceUsageSummary | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [resetBusy, setResetBusy] = useState(false);

  const loadSummary = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${hubHttp}/api/inference/usage`);
      if (!res.ok) throw new Error(await res.text());
      const data = (await res.json()) as InferenceUsageSummary;
      setSummary(data);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }, [hubHttp]);

  useEffect(() => {
    if (isActive) void loadSummary();
  }, [isActive, loadSummary]);

  const resetHubStats = async () => {
    if (!window.confirm('Reset all persisted inference usage stats on the hub?')) return;
    setResetBusy(true);
    try {
      const res = await fetch(`${hubHttp}/api/inference/usage`, { method: 'DELETE' });
      if (!res.ok) throw new Error(await res.text());
      clearSessionUsage();
      await loadSummary();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setResetBusy(false);
    }
  };

  if (!isActive) return null;

  const hubTotals = summary?.totals ?? {
    prompt_tokens: 0,
    completion_tokens: 0,
    calls: 0,
    estimated_cost_usd: 0,
  };

  const providerRows = Object.entries(summary?.by_provider ?? {}).sort(
    (a, b) =>
      b[1].prompt_tokens +
      b[1].completion_tokens -
      (a[1].prompt_tokens + a[1].completion_tokens),
  );
  const modelRows = Object.entries(summary?.by_model ?? {}).sort(
    (a, b) =>
      b[1].prompt_tokens +
      b[1].completion_tokens -
      (a[1].prompt_tokens + a[1].completion_tokens),
  );

  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-lg font-semibold text-slack-text mb-1">Inference usage</h3>
        <p className="text-sm text-slack-textMuted">
          Token counts and estimated cloud spend per turn. Local providers (Ollama, LM Studio) show
          tokens with $0 cost. Estimates use public list pricing when the model is recognized.
        </p>
      </div>

      <div className="grid gap-4 md:grid-cols-2">
        <TotalsCard
          title="This desktop session"
          totals={{
            prompt_tokens: sessionUsage.promptTokens,
            completion_tokens: sessionUsage.completionTokens,
            calls: sessionUsage.calls,
            estimated_cost_usd: sessionUsage.estimatedCostUsd,
          }}
        />
        <TotalsCard title="Hub lifetime (persisted)" totals={hubTotals} />
      </div>

      <div className="text-sm text-slack-textMuted">
        Session rollup: {formatSessionUsageLine(sessionUsage)}
        {summary?.updated_at && (
          <span className="ml-3">
            Hub updated {new Date(summary.updated_at).toLocaleString()}
          </span>
        )}
      </div>

      <div className="flex flex-wrap gap-2">
        <button
          type="button"
          onClick={() => void loadSummary()}
          disabled={loading}
          className="rounded bg-slack-accent px-3 py-2 text-sm font-medium text-white hover:opacity-90 disabled:opacity-50"
        >
          {loading ? 'Refreshing…' : 'Refresh'}
        </button>
        <button
          type="button"
          onClick={() => clearSessionUsage()}
          className="rounded border border-slack-border px-3 py-2 text-sm text-slack-text hover:bg-slack-bgHover"
        >
          Clear session totals
        </button>
        <button
          type="button"
          onClick={() => void resetHubStats()}
          disabled={resetBusy}
          className="rounded border border-red-500/40 px-3 py-2 text-sm text-red-300 hover:bg-red-500/10 disabled:opacity-50"
        >
          {resetBusy ? 'Resetting…' : 'Reset hub stats'}
        </button>
      </div>

      {error && <div className="text-sm text-red-300">{error}</div>}

      <div className="grid gap-4 xl:grid-cols-2">
        <BucketTable title="By provider" rows={providerRows} />
        <BucketTable title="By model" rows={modelRows} />
      </div>

      <div className="rounded-lg border border-slack-border bg-slack-bgHover p-4">
        <div className="font-medium text-slack-text mb-3">Recent turns</div>
        {!summary?.recent?.length ? (
          <div className="text-sm text-slack-textMuted">No turns recorded yet.</div>
        ) : (
          <ul className="space-y-2 text-sm font-mono max-h-72 overflow-y-auto">
            {[...summary.recent].reverse().slice(0, 30).map((turn, i) => (
              <li key={`${turn.at}-${i}`} className="text-slack-text border-b border-slack-border/40 pb-2">
                <span className="text-slack-textMuted">
                  {new Date(turn.at).toLocaleTimeString()}
                </span>{' '}
                <span className="text-slack-text">{turn.agent_name}</span>{' '}
                <span className="text-slack-textMuted">#{turn.channel}</span>
                <div className="text-slate-300 pl-4">
                  {formatTokenCount(turn.prompt_tokens)} in · {formatTokenCount(turn.completion_tokens)} out
                  {turn.model ? ` · ${turn.model}` : ''}
                  {turn.estimated_cost_usd && turn.estimated_cost_usd > 0
                    ? ` · ${formatCostUsd(turn.estimated_cost_usd)}`
                    : ''}
                </div>
              </li>
            ))}
          </ul>
        )}
      </div>

      <p className="text-xs text-slack-textMuted">
        Enable <strong>Turn telemetry drawer</strong> under Models &amp; performance to see per-turn
        usage live above the composer. Hub stats persist to{' '}
        <code className="font-mono">~/.neural-junkie/inference-stats.json</code>.
      </p>
    </div>
  );
}
