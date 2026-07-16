import type { UsageTelemetryPayload } from '../types/inferenceUsage';

export function formatTokenCount(n: number): string {
  if (!Number.isFinite(n) || n <= 0) return '0';
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
  if (n >= 10_000) return `${(n / 1_000).toFixed(1)}k`;
  if (n >= 1_000) return `${(n / 1_000).toFixed(2)}k`;
  return String(Math.round(n));
}

export function formatCostUsd(n: number): string {
  if (!Number.isFinite(n) || n <= 0) return '$0.00';
  if (n < 0.01) return `$${n.toFixed(4)}`;
  return `$${n.toFixed(2)}`;
}

export function formatUsageTelemetryHeadline(payload: UsageTelemetryPayload | Record<string, unknown>): string {
  const p = payload as UsageTelemetryPayload;
  const prompt = typeof p.prompt_tokens === 'number' ? p.prompt_tokens : 0;
  const completion = typeof p.completion_tokens === 'number' ? p.completion_tokens : 0;
  const parts = [`${formatTokenCount(prompt)} in`, `${formatTokenCount(completion)} out`];
  if (typeof p.tok_per_s === 'number' && p.tok_per_s > 0) {
    parts.push(`${Math.round(p.tok_per_s)} tok/s`);
  }
  return parts.join(' · ');
}

export function formatUsageTelemetrySubline(payload: UsageTelemetryPayload | Record<string, unknown>): string {
  const p = payload as UsageTelemetryPayload;
  const parts: string[] = [];
  if (p.model) parts.push(p.model);
  if (p.provider_id) parts.push(p.provider_id);
  if (p.cost_tier) parts.push(p.cost_tier);
  const cost = typeof p.estimated_cost_usd === 'number' ? p.estimated_cost_usd : 0;
  if (cost > 0) parts.push(formatCostUsd(cost));
  else if (p.provider_id && /ollama|lmstudio|local/i.test(p.provider_id)) parts.push('local · $0');
  return parts.join(' · ');
}

export function formatSessionUsageLine(totals: {
  promptTokens: number;
  completionTokens: number;
  estimatedCostUsd: number;
}): string {
  const cost =
    totals.estimatedCostUsd > 0 ? ` · ${formatCostUsd(totals.estimatedCostUsd)}` : '';
  return `${formatTokenCount(totals.promptTokens)} in · ${formatTokenCount(totals.completionTokens)} out${cost}`;
}
