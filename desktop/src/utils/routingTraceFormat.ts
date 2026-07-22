import type {
  RoutingGovernanceMeta,
  RoutingMeta,
  RoutingTelemetryPayload,
  TurnTraceContextSelection,
  TurnTraceResponse,
} from '../types/protocol';

const RETRIEVAL_LABELS: Record<string, string> = {
  general: 'General',
  conversation_memory: 'Memory',
  codebase: 'Codebase',
  collab_artifact: 'Collab',
  prior_reference: 'Prior ref',
};

const TIER_LABELS: Record<string, string> = {
  cheap: 'Cheap',
  standard: 'Standard',
  premium: 'Premium',
};

export function formatRetrievalLabel(mode?: string): string {
  const key = mode?.trim();
  if (!key) return '';
  return RETRIEVAL_LABELS[key] ?? key;
}

export function formatTierLabel(tier?: string): string {
  const key = tier?.trim();
  if (!key) return '';
  return TIER_LABELS[key] ?? key;
}

export function formatGovernanceSummary(gov?: RoutingGovernanceMeta | null): string {
  if (!gov) return '';
  const parts: string[] = [];
  if (gov.composer_mode) parts.push(gov.composer_mode);
  if (gov.context_scope && gov.context_scope !== 'none') parts.push(gov.context_scope);
  if (gov.impl_session || gov.can_run_impl_session) parts.push('impl');
  return parts.join(' · ');
}

export function formatRoutingModelLine(payload: RoutingTelemetryPayload | TurnTraceResponse['routing']): string {
  if (!payload) return '';
  const model =
    ('chat_model' in (payload as RoutingTelemetryPayload)
      ? (payload as RoutingTelemetryPayload).chat_model
      : (payload as TurnTraceResponse['routing'])?.model) ?? '';
  const tool =
    ('tool_model' in (payload as RoutingTelemetryPayload)
      ? (payload as RoutingTelemetryPayload).tool_model
      : (payload as TurnTraceResponse['routing'])?.tool_model) ?? '';
  const chat = model.trim();
  const tools = tool.trim();
  if (chat && tools && tools !== chat) return `${chat} · tools: ${tools}`;
  return chat || tools;
}

export function formatRoutingTelemetryHeadline(payload: RoutingTelemetryPayload): string {
  const model = formatRoutingModelLine(payload);
  const tier = formatTierLabel(payload.cost_tier);
  const retrieval = formatRetrievalLabel(payload.knowledge_route);
  const parts = [model, tier, retrieval].filter(Boolean);
  return parts.join(' · ');
}

export function formatRoutingTelemetrySubline(payload: RoutingTelemetryPayload): string {
  const parts: string[] = [];
  if (payload.reason) parts.push(payload.reason);
  if (payload.source) parts.push(payload.source);
  const gov = formatGovernanceSummary(payload.governance);
  if (gov) parts.push(gov);
  return parts.join(' · ');
}

export function formatToolTelemetryHeadline(payload: Record<string, unknown>): string {
  const name = typeof payload.name === 'string' ? payload.name : 'tool';
  const kind = typeof payload.kind === 'string' ? payload.kind : '';
  const iteration = typeof payload.iteration === 'number' ? payload.iteration : undefined;
  let line = name;
  if (kind) line += ` (${kind})`;
  if (iteration && iteration > 0) line += ` #${iteration}`;
  return line;
}

export function formatToolTelemetrySubline(payload: Record<string, unknown>): string {
  const preview = typeof payload.preview === 'string' ? payload.preview.trim() : '';
  if (preview) return preview;
  const max = typeof payload.max_iterations === 'number' ? payload.max_iterations : undefined;
  if (max && max > 0) return `max iterations: ${max}`;
  return '';
}

export function formatContextSelectionSummary(selection?: TurnTraceContextSelection): string {
  if (!selection) return '';
  const sections = selection.selected_sections?.filter(Boolean) ?? [];
  const selected = selection.selected_context_ids?.length ?? 0;
  const dropped = selection.dropped_context_ids?.length ?? 0;
  const parts = sections.length > 0 ? [sections.join(', ')] : [];
  if (selected > 0) parts.push(`${selected} selected`);
  if (dropped > 0) parts.push(`${dropped} omitted`);
  if ((selection.digest_version ?? 0) > 0) parts.push(`digest v${selection.digest_version}`);
  return parts.join(' · ');
}

export function formatRoutingAttempt(attempt: {
  model?: string;
  provider_id?: string;
  tier?: string;
  reason?: string;
  failure_reason?: string;
}): string {
  const target = attempt.model || attempt.provider_id || 'provider';
  const tier = attempt.tier ? ` · ${attempt.tier}` : '';
  const outcome = attempt.failure_reason
    ? ` · failed: ${attempt.failure_reason}`
    : attempt.reason
      ? ` · ${attempt.reason}`
      : '';
  return `${target}${tier}${outcome}`;
}

export function formatOmissionReasons(selection?: TurnTraceContextSelection): string[] {
  if (!selection) return [];
  return Object.entries({
    ...(selection.omission_reasons ?? {}),
    ...(selection.budget_omission_reasons ?? {}),
  })
    .sort(([left], [right]) => left.localeCompare(right))
    .map(([id, reason]) => `${id}: ${reason}`);
}

export { formatUsageTelemetryHeadline, formatUsageTelemetrySubline } from './inferenceUsageFormat';

export function routingMetaToTraceSections(meta: RoutingMeta) {
  return {
    routing: {
      model: meta.model,
      tool_model: meta.tool_model,
      provider_id: meta.provider_id,
      domain: meta.domain,
      cost_tier: meta.cost_tier,
      reason: meta.reason,
      source: meta.source,
    },
    retrieval: {
      mode: meta.knowledge_route,
      reason: meta.knowledge_reason,
    },
    governance: {
      composer_mode: meta.composer_mode,
      context_scope: meta.context_scope,
      impl_session: meta.impl_session,
    },
  };
}

export function isTurnTraceResponse(raw: unknown): raw is TurnTraceResponse {
  return !!raw && typeof raw === 'object' && !Array.isArray(raw);
}
