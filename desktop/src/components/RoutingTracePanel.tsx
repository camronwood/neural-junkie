import { useEffect, useState, type ReactNode } from 'react';
import { ChatAPI } from '../api/chatAPI';
import { getHubBaseURL } from '../config/hubUrl';
import type { TurnTraceResponse } from '../types/protocol';
import {
  formatGovernanceSummary,
  formatRetrievalLabel,
  formatRoutingModelLine,
  formatTierLabel,
  isTurnTraceResponse,
} from '../utils/routingTraceFormat';

interface RoutingTracePanelProps {
  channel: string;
  messageId: string;
  query?: string;
  enabled: boolean;
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

  return (
    <div className="space-y-1">
      {routing && (
        <TraceSection title="Model routing">
          <TraceField label="Model" value={formatRoutingModelLine(routing) || routing.model} />
          <TraceField label="Provider" value={routing.provider_id} />
          <TraceField label="Domain" value={routing.domain} />
          <TraceField label="Tier" value={formatTierLabel(routing.cost_tier) || routing.cost_tier} />
          <TraceField label="Reason" value={routing.reason} />
          <TraceField label="Classifier" value={routing.source} />
        </TraceSection>
      )}
      {retrieval && (retrieval.mode || retrieval.reason) && (
        <TraceSection title="Retrieval">
          <TraceField
            label="Mode"
            value={formatRetrievalLabel(retrieval.mode) || retrieval.mode}
          />
          <TraceField label="Reason" value={retrieval.reason} />
        </TraceSection>
      )}
      {governance && formatGovernanceSummary(governance) && (
        <TraceSection title="Governance">
          <TraceField label="Composer" value={governance.composer_mode} />
          <TraceField label="Context scope" value={governance.context_scope} />
          <TraceField label="Impl session" value={governance.impl_session} />
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
