import { useEffect, useRef } from 'react';
import type { ArenaStepMeta } from './arenaSidecarApi';

export type ArenaPlayLogEntry = {
  id: string;
  model?: string;
  seat?: string;
  reply?: string;
  moveLabel?: string;
  skipped?: boolean;
  thinking?: boolean;
};

export function formatArenaMoveLabel(
  challenge: string,
  parsed?: Record<string, unknown>,
  parsedAnswer?: string,
): string | undefined {
  if (parsedAnswer) return parsedAnswer;
  if (!parsed) return undefined;
  if (parsed.column != null) return `column ${String(parsed.column)}`;
  const move = parsed.move ?? parsed.uci;
  if (move) return String(move);
  return undefined;
}

export function stepMetaToLogEntry(step: ArenaStepMeta | undefined, challenge: string): ArenaPlayLogEntry | null {
  if (!step) return null;
  if (step.skipped) {
    return {
      id: crypto.randomUUID(),
      seat: step.seat,
      skipped: true,
      moveLabel: step.reason === 'human_turn' ? 'Waiting for human' : step.reason,
    };
  }
  const parsed = step.parsed_move as Record<string, unknown> | undefined;
  const moveLabel = formatArenaMoveLabel(challenge, parsed, step.parsed_answer);
  return {
    id: crypto.randomUUID(),
    model: step.model,
    seat: step.seat,
    reply: step.reply,
    moveLabel,
  };
}

interface ArenaPlayLogProps {
  entries: ArenaPlayLogEntry[];
  autoRunning?: boolean;
}

export function ArenaPlayLog({ entries, autoRunning }: ArenaPlayLogProps) {
  const scrollerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const scroller = scrollerRef.current;
    if (!scroller || entries.length === 0) return;
    // Scroll only this panel — scrollIntoView can drag parent modals and trap you at the bottom.
    scroller.scrollTop = scroller.scrollHeight;
  }, [entries.length, autoRunning]);

  return (
    <div className="arena-retro-playlog" ref={scrollerRef}>
      <h3>Play-by-play</h3>
      {autoRunning && (
        <p className="arena-retro-playlog-live" aria-live="polite">
          Live · models thinking…
        </p>
      )}
      {entries.length === 0 && !autoRunning && (
        <p className="arena-retro-playlog-empty">Model moves and reasoning appear here.</p>
      )}
      <ol className="arena-retro-playlog-list">
        {entries.map((entry, index) => (
          <li key={entry.id} className="arena-retro-playlog-item">
            <div className="arena-retro-playlog-head">
              <span className="arena-retro-playlog-num">{index + 1}</span>
              {entry.skipped ? (
                <span className="arena-retro-playlog-model muted">{entry.moveLabel ?? 'Skipped'}</span>
              ) : (
                <span className="arena-retro-playlog-model" title={entry.model}>
                  {entry.model ?? 'model'}
                  {entry.moveLabel ? ` → ${entry.moveLabel}` : ''}
                </span>
              )}
            </div>
            {entry.reply && !entry.skipped && (
              <p className="arena-retro-playlog-reply">{entry.reply}</p>
            )}
          </li>
        ))}
      </ol>
    </div>
  );
}
