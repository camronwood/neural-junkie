import { useMemo, useState } from 'react';
import { parseRoutingTelemetryPayload, type TurnTelemetryEvent } from '../types/protocol';
import { useChatStore } from '../stores/chatStore';
import {
  formatRoutingTelemetryHeadline,
  formatRoutingTelemetrySubline,
  formatToolTelemetryHeadline,
  formatToolTelemetrySubline,
} from '../utils/routingTraceFormat';

interface TurnTelemetryDrawerProps {
  channel: string;
  enabled: boolean;
}

function formatElapsed(at: number): string {
  const sec = Math.max(0, Math.floor((Date.now() - at) / 1000));
  if (sec < 60) return `${sec}s ago`;
  return `${Math.floor(sec / 60)}m ago`;
}

function TelemetryRow({ ev }: { ev: TurnTelemetryEvent }) {
  if (ev.kind === 'routing' && ev.payload) {
    const payload = parseRoutingTelemetryPayload(ev.payload);
    if (payload) {
      const headline = formatRoutingTelemetryHeadline(payload);
      const subline = formatRoutingTelemetrySubline(payload);
      return (
        <li
          key={ev.id}
          className="leading-snug"
          data-testid="turn-telemetry-routing-row"
          title={ev.detail}
        >
          <span className="text-slate-500">{formatElapsed(ev.at)}</span>{' '}
          <span className="text-cyan-400/90">routing</span>{' '}
          <span className="text-slate-300">{ev.agentName}</span>
          <div className="text-slate-200 pl-4">{headline || ev.detail}</div>
          {subline && <div className="text-slate-500 pl-4 truncate">{subline}</div>}
        </li>
      );
    }
  }

  if (ev.kind === 'tool' && ev.payload) {
    const headline = formatToolTelemetryHeadline(ev.payload);
    const subline = formatToolTelemetrySubline(ev.payload);
    return (
      <li key={ev.id} className="leading-snug" data-testid="turn-telemetry-tool-row" title={ev.detail}>
        <span className="text-slate-500">{formatElapsed(ev.at)}</span>{' '}
        <span className="text-cyan-400/90">tool</span>{' '}
        <span className="text-slate-300">{ev.agentName}</span>
        <div className="text-slate-200 pl-4">{headline}</div>
        {subline && <div className="text-slate-500 pl-4 truncate">{subline}</div>}
      </li>
    );
  }

  return (
    <li key={ev.id} className="truncate" title={ev.detail}>
      <span className="text-slate-500">{formatElapsed(ev.at)}</span>{' '}
      <span className="text-cyan-400/90">{ev.kind}</span>{' '}
      <span className="text-slate-300">{ev.agentName}</span>{' '}
      <span className="text-slate-400">— {ev.detail}</span>
    </li>
  );
}

export function TurnTelemetryDrawer({ channel, enabled }: TurnTelemetryDrawerProps) {
  const events = useChatStore((s) => s.turnTelemetryByChannel.get(channel) ?? []);
  const clearTurnTelemetry = useChatStore((s) => s.clearTurnTelemetry);
  const [open, setOpen] = useState(true);

  const rows = useMemo(() => [...events].reverse().slice(0, 40), [events]);

  if (!enabled) return null;

  return (
    <div
      className="mx-3 mb-1 rounded border border-slate-700/80 bg-slate-900/70 text-xs text-slate-200"
      data-testid="turn-telemetry-drawer"
    >
      <div className="flex items-center justify-between px-3 py-2 border-b border-slate-700/60">
        <button
          type="button"
          className="font-semibold text-left hover:text-white"
          onClick={() => setOpen((v) => !v)}
        >
          Turn telemetry ({events.length})
        </button>
        <div className="flex items-center gap-2">
          <button
            type="button"
            className="text-slate-400 hover:text-slate-200"
            onClick={() => clearTurnTelemetry(channel)}
          >
            Clear
          </button>
          <button
            type="button"
            className="text-slate-400 hover:text-slate-200"
            onClick={() => setOpen((v) => !v)}
          >
            {open ? 'Hide' : 'Show'}
          </button>
        </div>
      </div>
      {open && (
        <ul className="max-h-52 overflow-y-auto px-3 py-2 space-y-2 font-mono">
          {rows.length === 0 ? (
            <li className="text-slate-500">No telemetry yet this session.</li>
          ) : (
            rows.map((ev) => <TelemetryRow key={ev.id} ev={ev} />)
          )}
        </ul>
      )}
    </div>
  );
}
