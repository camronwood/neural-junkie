export type UsageBucketTotals = {
  prompt_tokens: number;
  completion_tokens: number;
  calls: number;
  estimated_cost_usd: number;
};

export type InferenceUsageTurn = {
  at: string;
  channel: string;
  agent_id: string;
  agent_name: string;
  provider_id?: string;
  model?: string;
  cost_tier?: string;
  prompt_tokens: number;
  completion_tokens: number;
  calls?: number;
  estimated_cost_usd?: number;
  ttft_ms?: number;
  tok_per_s?: number;
};

export type InferenceUsageSummary = {
  started_at: string;
  updated_at: string;
  totals: UsageBucketTotals;
  by_provider: Record<string, UsageBucketTotals>;
  by_model: Record<string, UsageBucketTotals>;
  recent: InferenceUsageTurn[];
};

export type SessionUsageTotals = {
  promptTokens: number;
  completionTokens: number;
  calls: number;
  estimatedCostUsd: number;
};

export const EMPTY_SESSION_USAGE: SessionUsageTotals = {
  promptTokens: 0,
  completionTokens: 0,
  calls: 0,
  estimatedCostUsd: 0,
};

export type UsageTelemetryPayload = {
  prompt_tokens?: number;
  completion_tokens?: number;
  calls?: number;
  estimated_cost_usd?: number;
  provider_id?: string;
  model?: string;
  cost_tier?: string;
  ttft_ms?: number;
  tok_per_s?: number;
};

export function usagePayloadFromRecord(raw: Record<string, unknown>): UsageTelemetryPayload | null {
  const prompt = num(raw.prompt_tokens);
  const completion = num(raw.completion_tokens);
  const calls = num(raw.calls);
  if (prompt === 0 && completion === 0 && calls === 0) return null;
  return {
    prompt_tokens: prompt,
    completion_tokens: completion,
    calls: calls || (prompt > 0 || completion > 0 ? 1 : 0),
    estimated_cost_usd: num(raw.estimated_cost_usd),
    provider_id: str(raw.provider_id),
    model: str(raw.model),
    cost_tier: str(raw.cost_tier),
    ttft_ms: num(raw.ttft_ms),
    tok_per_s: num(raw.tok_per_s),
  };
}

function num(v: unknown): number {
  if (typeof v === 'number' && Number.isFinite(v)) return v;
  if (typeof v === 'string' && v.trim() !== '') {
    const n = Number(v);
    return Number.isFinite(n) ? n : 0;
  }
  return 0;
}

function str(v: unknown): string | undefined {
  return typeof v === 'string' && v.trim() ? v.trim() : undefined;
}

export function addSessionUsage(
  prev: SessionUsageTotals,
  payload: UsageTelemetryPayload,
): SessionUsageTotals {
  const prompt = payload.prompt_tokens ?? 0;
  const completion = payload.completion_tokens ?? 0;
  const calls = payload.calls ?? (prompt > 0 || completion > 0 ? 1 : 0);
  return {
    promptTokens: prev.promptTokens + prompt,
    completionTokens: prev.completionTokens + completion,
    calls: prev.calls + calls,
    estimatedCostUsd: prev.estimatedCostUsd + (payload.estimated_cost_usd ?? 0),
  };
}
