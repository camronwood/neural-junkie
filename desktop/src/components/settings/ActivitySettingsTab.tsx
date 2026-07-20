import { useActivityLogStore, type ActivityKind } from '../../stores/activityLogStore';
import type { SettingsTabProps } from './settingsShared';

const KIND_LABEL: Record<ActivityKind, string> = {
  channel: 'Channel',
  file: 'File',
  terminal: 'Terminal',
  command: 'Command',
  settings: 'Settings',
  agent: 'Agent',
  other: 'Other',
};

function formatTime(ts: number): string {
  try {
    return new Date(ts).toLocaleString();
  } catch {
    return String(ts);
  }
}

export function ActivitySettingsTab({ isActive }: SettingsTabProps) {
  const events = useActivityLogStore((s) => s.events);
  const clear = useActivityLogStore((s) => s.clear);

  if (!isActive) return null;

  return (
    <div className="space-y-4">
      <div className="flex items-start justify-between gap-3">
        <div>
          <h3 className="text-lg font-semibold text-slack-text mb-1">Activity log</h3>
          <p className="text-sm text-slack-textMuted">
            Local history of channels, files, terminal sessions, and related actions on this device.
            Kept for the last {500} events.
          </p>
        </div>
        <button
          type="button"
          onClick={() => {
            if (window.confirm('Clear the activity log on this device?')) clear();
          }}
          className="shrink-0 px-3 py-1.5 text-xs rounded border border-slack-border text-slack-textMuted hover:text-slack-text hover:bg-slack-bgHover"
        >
          Clear
        </button>
      </div>

      {events.length === 0 ? (
        <p className="text-sm text-slack-textMuted py-8 text-center">No activity recorded yet.</p>
      ) : (
        <ul className="divide-y divide-slack-border border border-slack-border rounded max-h-[60vh] overflow-y-auto">
          {events.map((ev) => (
            <li key={ev.id} className="px-3 py-2.5 hover:bg-slack-bgHover/50">
              <div className="flex items-baseline justify-between gap-2">
                <span className="text-sm text-slack-text font-medium">{ev.title}</span>
                <span className="text-[11px] text-slack-textMuted shrink-0">{formatTime(ev.ts)}</span>
              </div>
              <div className="mt-0.5 flex flex-wrap gap-x-3 gap-y-0.5 text-[11px] text-slack-textMuted">
                <span className="uppercase tracking-wide opacity-70">{KIND_LABEL[ev.kind]}</span>
                {ev.channel && <span>#{ev.channel}</span>}
                {ev.path && <span className="font-mono truncate max-w-[28rem]">{ev.path}</span>}
                {ev.detail && !ev.path && <span className="truncate max-w-[28rem]">{ev.detail}</span>}
              </div>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
