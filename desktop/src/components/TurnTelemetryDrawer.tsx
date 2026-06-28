import { useMemo, useState } from 'react';
import type { TurnTelemetryEvent } from '../types/protocol';
import { useChatStore } from '../stores/chatStore';

interface TurnTelemetryDrawerProps {
  channel: string;
  enabled: boolean;
}

function formatElapsed(at: number): string {
  const sec = Math.max(0, Math.floor((Date.now() - at) / 1000));
  if (sec < 60) return `${sec}s ago`;
  return `${Math.floor(sec / 60)}m ago`;
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
        <ul className="max-h-40 overflow-y-auto px-3 py-2 space-y-1 font-mono">
          {rows.length === 0 ? (
            <li className="text-slate-500">No telemetry yet this session.</li>
          ) : (
            rows.map((ev: TurnTelemetryEvent) => (
              <li key={ev.id} className="truncate" title={ev.detail}>
                <span className="text-slate-500">{formatElapsed(ev.at)}</span>{' '}
                <span className="text-cyan-400/90">{ev.kind}</span>{' '}
                <span className="text-slate-300">{ev.agentName}</span>{' '}
                <span className="text-slate-400">— {ev.detail}</span>
              </li>
            ))
          )}
        </ul>
      )}
    </div>
  );
}
