import { useEffect, useState } from 'react';
import { ChatAPI } from '../api/chatAPI';
import { getHubBaseURL } from '../config/hubUrl';

interface RoutingTracePanelProps {
  channel: string;
  messageId: string;
  query?: string;
  enabled: boolean;
}

export function RoutingTracePanel({ channel, messageId, query, enabled }: RoutingTracePanelProps) {
  const [trace, setTrace] = useState<Record<string, unknown> | null>(null);
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
        if (!cancelled) setTrace(data);
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
    <div className="mx-3 mb-2 rounded border border-slate-700 bg-slate-900/80 p-2 text-xs text-slate-200" data-testid="routing-trace-panel">
      <div className="font-semibold mb-1">Routing trace</div>
      {error && <div className="text-red-300">{error}</div>}
      {trace && (
        <pre className="whitespace-pre-wrap break-all">{JSON.stringify(trace, null, 2)}</pre>
      )}
    </div>
  );
}
