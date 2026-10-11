import type { ArenaBenchmarkJob, ArenaBenchmarkRank } from './arenaSidecarApi';

export function isBenchmarkJobActive(status: string | undefined): boolean {
  const s = (status || '').toLowerCase();
  return s === 'queued' || s === 'running';
}

export function formatPassRate(rate: number | undefined): string {
  if (typeof rate !== 'number' || Number.isNaN(rate)) return '—';
  return `${Math.round(rate * 100)}%`;
}

export function formatLogicAccuracy(correct?: number, total?: number): string {
  if (typeof total !== 'number' || total <= 0) return '—';
  return `${correct ?? 0}/${total}`;
}

export function benchmarkProgressLabel(job: ArenaBenchmarkJob | null): string {
  if (!job) return '';
  const status = (job.status || '').toLowerCase();
  if (status === 'queued') return 'Queued…';
  if (status === 'cancelled') return 'Cancelled';
  if (status === 'failed') return job.error ? `Failed · ${job.error}` : 'Failed';
  if (status === 'completed') return 'Completed';
  const model = job.current_model || '…';
  const challenge = job.current_challenge || '…';
  const mi = (job.model_index ?? 0) + 1;
  const mc = job.models?.length ?? 0;
  return `Model ${mi}/${mc} · ${model} · ${challenge}`;
}

/** Prefer server ranking; fall back to sorting results by pass_rate. */
export function rankingFromJob(job: ArenaBenchmarkJob | null): ArenaBenchmarkRank[] {
  if (!job) return [];
  if (Array.isArray(job.ranking) && job.ranking.length > 0) {
    return job.ranking;
  }
  const results = job.results ?? [];
  const scored = results
    .filter((r) => {
      const s = (r.status || '').toLowerCase();
      return s === 'completed' || s === 'failed' || s === 'skipped';
    })
    .slice()
    .sort((a, b) => {
      const pr = (b.pass_rate ?? 0) - (a.pass_rate ?? 0);
      if (pr !== 0) return pr;
      const lc = (b.logic_correct ?? 0) - (a.logic_correct ?? 0);
      if (lc !== 0) return lc;
      return a.model.localeCompare(b.model);
    });
  return scored.map((r, i) => ({
    rank: i + 1,
    model: r.model,
    pass_rate: r.pass_rate ?? 0,
    logic_correct: r.logic_correct,
    logic_total: r.logic_total,
    connect4_completed: r.connect4_completed,
    capability_passed: r.capability_passed,
    error: r.error,
  }));
}

export function toggleModelSelection(selected: string[], model: string): string[] {
  const tag = model.trim();
  if (!tag) return selected;
  if (selected.includes(tag)) {
    return selected.filter((m) => m !== tag);
  }
  return [...selected, tag].sort((a, b) => a.localeCompare(b));
}
