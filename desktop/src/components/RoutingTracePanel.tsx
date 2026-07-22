import { useEffect, useState, type ReactNode } from 'react';
import { ChatAPI } from '../api/chatAPI';
import { getHubBaseURL } from '../config/hubUrl';
import type { TurnTraceResponse } from '../types/protocol';
import {
  formatContextSelectionSummary,
  formatGovernanceSummary,
  formatOmissionReasons,
  formatRetrievalLabel,
  formatRoutingAttempt,
  formatRoutingModelLine,
  formatTierLabel,
  formatUsageTelemetryHeadline,
  formatUsageTelemetrySubline,
  isTurnTraceResponse,
} from '../utils/routingTraceFormat';
import { usagePayloadFromRecord } from '../types/inferenceUsage';

interface RoutingTracePanelProps {
  channel: string;
  messageId: string;
  query?: string;
  enabled: boolean;
}

interface OrchestrationTrace {
  run?: { id?: string; status?: string; maxConcurrency?: number; max_concurrency?: number };
  tasks?: Array<{ id?: string; title?: string; status?: string; attemptCount?: number; attempt_count?: number }>;
  events?: Array<{ id?: number; type?: string; taskID?: string; task_id?: string; createdAt?: string; created_at?: string }>;
  inputs?: Array<{ id?: string; kind?: string; status?: string }>;
  workers?: Array<{ id?: string; status?: string; queue?: string }>;
  metrics?: Array<{
    task_id?: string;
    queue_delay_ms?: number;
    execution_ms?: number;
    retries?: number;
    cache_hit?: boolean;
    failure_reason?: string;
    inference_usage?: unknown[];
  }>;
  enforced?: boolean;
}

function TraceSection({ title, children }: { title: string; children: ReactNode }) {
  return (
    <div className="mb-2">
      <div className="font-medium text-slate-300">{title}</div>
      <div className="text-slate-400 pl-2">{children}</div>
    </div>
  );
}

function TraceField({ label, value }: { label: string; value?: string | number | boolean }) {
  if (value === undefined || value === '' || value === false) return null;
  const display = typeof value === 'boolean' ? (value ? 'yes' : 'no') : String(value);
  return (
    <div>
      <span className="text-slate-500">{label}: </span>
      <span>{display}</span>
    </div>
  );
}

function StructuredTrace({ trace }: { trace: TurnTraceResponse }) {
  const routing = trace.routing;
  const retrieval = trace.retrieval;
  const governance = trace.governance;
  const compress = trace.compress;
  const selection = trace.context_selection;
  const orchestration = (trace as TurnTraceResponse & { orchestration?: OrchestrationTrace })
    .orchestration;

  return (
    <div className="space-y-1">
      {orchestration?.run && (
        <TraceSection title="Orchestration">
          <TraceField label="Run" value={orchestration.run.id} />
          <TraceField label="State" value={orchestration.run.status} />
          <TraceField
            label="Concurrency"
            value={orchestration.run.maxConcurrency ?? orchestration.run.max_concurrency}
          />
          <TraceField label="Durable claims enforced" value={orchestration.enforced} />
          <TraceField label="Tasks" value={orchestration.tasks?.length} />
          <TraceField label="Pending input" value={orchestration.inputs?.length} />
          <TraceField
            label="Workers"
            value={orchestration.workers
              ?.map((worker) => `${worker.id ?? 'worker'}:${worker.status ?? 'unknown'}`)
              .join(', ')}
          />
          {orchestration.tasks && orchestration.tasks.length > 0 && (
            <ul className="list-disc pl-4">
              {orchestration.tasks.map((task) => (
                <li key={task.id ?? task.title}>
                  {task.title ?? task.id ?? 'task'} · {task.status ?? 'unknown'} · attempt{' '}
                  {task.attemptCount ?? task.attempt_count ?? 0}
                </li>
              ))}
            </ul>
          )}
          {orchestration.metrics && orchestration.metrics.length > 0 && (
            <ul className="pl-4 text-slate-500">
              {orchestration.metrics.map((metric) => (
                <li key={metric.task_id}>
                  {metric.task_id?.slice(0, 8) ?? 'task'} · queue {metric.queue_delay_ms ?? 0}ms ·
                  run {metric.execution_ms ?? 0}ms · retries {metric.retries ?? 0}
                  {metric.cache_hit ? ' · cache hit' : ''}
                  {metric.inference_usage?.length
                    ? ` · ${metric.inference_usage.length} usage record(s)`
                    : ''}
                  {metric.failure_reason ? ` · ${metric.failure_reason}` : ''}
                </li>
              ))}
            </ul>
          )}
          {orchestration.events && orchestration.events.length > 0 && (
            <details className="mt-1">
              <summary className="cursor-pointer text-slate-500">Recent state events</summary>
              <ul className="pl-4">
                {orchestration.events.slice(-8).map((event) => (
                  <li key={event.id} className="font-mono text-[11px]">
                    {event.type ?? 'event'}
                    {event.taskID || event.task_id
                      ? ` · task ${String(event.taskID ?? event.task_id).slice(0, 8)}`
                      : ''}
                  </li>
                ))}
              </ul>
            </details>
          )}
        </TraceSection>
      )}
      {routing && (
        <TraceSection title="Model routing">
          <TraceField label="Model" value={formatRoutingModelLine(routing) || routing.model} />
          <TraceField label="Provider" value={routing.provider_id} />
          <TraceField label="Domain" value={routing.domain} />
          <TraceField label="Tier" value={formatTierLabel(routing.cost_tier) || routing.cost_tier} />
          <TraceField label="Reason" value={routing.reason} />
          <TraceField label="Classifier" value={routing.source} />
          {routing.attempts && routing.attempts.length > 0 && (
            <ul className="list-disc pl-4">
              {routing.attempts.map((attempt, i) => (
                <li key={`${attempt.provider_id ?? ''}-${attempt.model ?? ''}-${i}`}>
                  {formatRoutingAttempt(attempt)}
                </li>
              ))}
            </ul>
          )}
          <TraceField label="Recovery signals" value={routing.failure_evidence?.join(', ')} />
        </TraceSection>
      )}
      {retrieval && (retrieval.mode || retrieval.reason) && (
        <TraceSection title="Retrieval">
          <TraceField
            label="Mode"
            value={formatRetrievalLabel(retrieval.mode) || retrieval.mode}
          />
          <TraceField label="Reason" value={retrieval.reason} />
          <TraceField label="Memory results" value={retrieval.memory_count} />
          <TraceField label="Codebase results" value={retrieval.codebase_count} />
        </TraceSection>
      )}
      {selection && (
        <TraceSection title="Context selection">
          <TraceField label="Selection" value={formatContextSelectionSummary(selection)} />
          <TraceField
            label="Recovery"
            value={
              selection.recovery?.active
                ? `${selection.recovery.correction_count ?? 0} correction(s), ${selection.recovery.superseded_count ?? 0} superseded`
                : undefined
            }
          />
          <TraceField
            label="Compression"
            value={
              selection.compression?.applied
                ? `${selection.compression.original_bytes ?? 0}→${selection.compression.final_bytes ?? 0} bytes`
                : selection.compression?.summary_checkpoint
                  ? 'summary checkpoint'
                  : undefined
            }
          />
          {formatOmissionReasons(selection).map((reason) => (
            <div key={reason}>
              <span className="text-slate-500">Omitted: </span>
              <span>{reason}</span>
            </div>
          ))}
          {selection.provenance && selection.provenance.length > 0 && (
            <ul className="list-disc pl-4">
              {selection.provenance.map((item, i) => (
                <li key={`${item.id ?? item.section ?? 'context'}-${i}`}>
                  {item.id || item.section || 'context'} · {item.source || 'unknown'}
                  {item.score != null ? ` · relevance ${item.score}` : ''}
                  {item.freshness ? ` · ${item.freshness}` : ''}
                </li>
              ))}
            </ul>
          )}
        </TraceSection>
      )}
      {governance && formatGovernanceSummary(governance) && (
        <TraceSection title="Governance">
          <TraceField label="Composer" value={governance.composer_mode} />
          <TraceField label="Context scope" value={governance.context_scope} />
          <TraceField label="Impl session" value={governance.impl_session} />
        </TraceSection>
      )}
      {trace.inference_usage && typeof trace.inference_usage === 'object' && (
        <TraceSection title="Inference usage">
          {(() => {
            const parsed = usagePayloadFromRecord(trace.inference_usage as Record<string, unknown>);
            if (!parsed) return <div className="text-slate-500">No usage recorded</div>;
            return (
              <>
                <div>{formatUsageTelemetryHeadline(parsed)}</div>
                <div className="text-slate-500">{formatUsageTelemetrySubline(parsed)}</div>
              </>
            );
          })()}
        </TraceSection>
      )}
      {Array.isArray(trace.tool_steps) && trace.tool_steps.length > 0 && (
        <TraceSection title="Tools">
          <div>{trace.tool_steps.length} step(s)</div>
          <ul className="list-disc pl-4">
            {(trace.tool_steps as Array<Record<string, unknown>>).map((step, i) => (
              <li key={i}>
                {String(step.name ?? 'tool')} · {String(step.kind ?? '')}
                {typeof step.iteration === 'number' ? ` (#${step.iteration})` : ''}
              </li>
            ))}
          </ul>
        </TraceSection>
      )}
      {Array.isArray(trace.spans) && trace.spans.length > 0 && (
        <TraceSection title="Spans">
          <ul className="space-y-0.5">
            {trace.spans.map((span) => {
              const dur =
                span.start_ms != null && span.end_ms != null
                  ? `${span.end_ms - span.start_ms}ms`
                  : '';
              return (
                <li key={span.id ?? span.name} className="font-mono text-[11px]">
                  <span className={span.status === 'error' ? 'text-red-300' : 'text-slate-300'}>
                    {span.name}
                  </span>
                  {dur ? <span className="text-slate-500"> · {dur}</span> : null}
                  {span.parent_id ? (
                    <span className="text-slate-600"> · parent {span.parent_id.slice(0, 8)}</span>
                  ) : null}
                </li>
              );
            })}
          </ul>
        </TraceSection>
      )}
      {compress && (compress.strategy || compress.bytes_in || compress.bytes_out) && (
        <TraceSection title="Context">
          <TraceField label="Strategy" value={compress.strategy} />
          <TraceField label="Bytes in" value={compress.bytes_in} />
          <TraceField label="Bytes out" value={compress.bytes_out} />
        </TraceSection>
      )}
      {trace.reasoning_text && (
        <TraceSection title="Reasoning">
          <div className="whitespace-pre-wrap break-words">{trace.reasoning_text}</div>
        </TraceSection>
      )}
    </div>
  );
}

export function RoutingTracePanel({ channel, messageId, query, enabled }: RoutingTracePanelProps) {
  const [trace, setTrace] = useState<TurnTraceResponse | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!enabled || !messageId) {
      setTrace(null);
      return;
    }
    let cancelled = false;
    const api = new ChatAPI(getHubBaseURL());
    void api
      .fetchTurnTrace(channel, messageId, query)
      .then((data) => {
        if (!cancelled && isTurnTraceResponse(data)) setTrace(data);
      })
      .catch((e: unknown) => {
        if (!cancelled) setError(e instanceof Error ? e.message : String(e));
      });
    return () => {
      cancelled = true;
    };
  }, [channel, messageId, query, enabled]);

  if (!enabled) return null;

  return (
    <div
      className="mx-3 mb-2 rounded border border-slate-700 bg-slate-900/80 p-2 text-xs text-slate-200"
      data-testid="routing-trace-panel"
    >
      <div className="font-semibold mb-1">Routing trace</div>
      <p className="text-slate-400 mb-1">
        Post-hoc snapshot for the highlighted message. Enable Settings → Turn telemetry drawer for
        live events.
      </p>
      {error && <div className="text-red-300">{error}</div>}
      {trace && (
        <>
          <StructuredTrace trace={trace} />
          <details className="mt-2">
            <summary className="cursor-pointer text-slate-500 hover:text-slate-300">Raw JSON</summary>
            <pre className="whitespace-pre-wrap break-all mt-1">{JSON.stringify(trace, null, 2)}</pre>
          </details>
        </>
      )}
    </div>
  );
}
